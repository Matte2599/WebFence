//go:build m3cdplab && linux

// This optional Linux lab uses only synthetic CDP responses. It is not a
// desktop browser or a scanner and accepts no target URL.
package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Matte2599/WebFence/internal/browser"
)

const (
	syntheticOrigin = "http://site.test"
	maxCDPFrame     = 64 << 10
	maxRequests     = 10
)

type helperConfig struct {
	Chrome string `json:"chrome"`
}

type trialResult struct {
	Loaded          bool `json:"loaded"`
	ScriptSeen      bool `json:"script_seen"`
	APISeen         bool `json:"api_seen"`
	Document        int  `json:"document"`
	Script          int  `json:"script"`
	API             int  `json:"api"`
	Redirect        int  `json:"redirect"`
	OutsideImage    int  `json:"outside_image"`
	OutsideRedirect int  `json:"outside_redirect"`
}

func main() {
	var err error
	if len(os.Args) == 2 && os.Args[1] == "--browser-helper" {
		err = runChild()
	} else if len(os.Args) == 1 {
		err = runParent()
	} else {
		err = errors.New("the CDP lab accepts no target arguments")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "M3 CDP lab failed:", err)
		os.Exit(1)
	}
	if len(os.Args) == 1 {
		fmt.Println("PASS M3 headless browser fixture: real HTTP origin, document, script, fetch, redirect and outside requests under Linux network filter")
	}
}

func runParent() error {
	chrome := os.Getenv("WF_CDP_CHROME")
	if !filepath.IsAbs(chrome) || len(chrome) > 4096 {
		return errors.New("absolute synthetic-lab Chromium path required")
	}
	info, err := os.Stat(chrome)
	if err != nil || info.IsDir() {
		return errors.New("Chromium executable is unavailable")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(helperConfig{Chrome: chrome})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	resultJSON, err := browser.RunHelper(ctx, executable, payload, browser.HelperLimits{
		MaxRuntime: 20 * time.Second, MaxOutputBytes: 4096,
	})
	if err != nil {
		return err
	}
	var result trialResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return errors.New("invalid CDP helper result")
	}
	if !result.Loaded || !result.ScriptSeen || !result.APISeen || result.Document != 1 ||
		result.Script != 1 || result.API != 1 || result.Redirect != 1 ||
		result.OutsideImage != 1 || result.OutsideRedirect != 1 {
		return fmt.Errorf("unexpected synthetic CDP result: %+v", result)
	}
	return nil
}

func runChild() error {
	if err := browser.ApplyHelperResourceLimits(); err != nil {
		return err
	}
	var config helperConfig
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return errors.New("invalid CDP helper configuration")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF || !filepath.IsAbs(config.Chrome) {
		return errors.New("invalid CDP helper configuration")
	}
	if err := browser.ApplyHelperNetworkIsolation(); err != nil {
		return err
	}
	for _, probe := range []struct{ network, address string }{
		{"tcp4", "127.0.0.1:9"}, {"tcp6", "[::1]:9"},
	} {
		conn, err := net.DialTimeout(probe.network, probe.address, 100*time.Millisecond)
		if conn != nil {
			_ = conn.Close()
		}
		if !errors.Is(err, syscall.EPERM) {
			return errors.New("direct browser networking is not blocked")
		}
	}
	result, err := runCDP(config.Chrome)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

type cdpMessage struct {
	ID        int             `json:"id"`
	Method    string          `json:"method"`
	SessionID string          `json:"sessionId"`
	Params    json.RawMessage `json:"params"`
	Result    json.RawMessage `json:"result"`
	Error     json.RawMessage `json:"error"`
}

type cdpPipe struct {
	writer   *os.File
	reader   *bufio.Reader
	nextID   int
	replies  map[int]cdpMessage
	result   trialResult
	requests int
}

func runCDP(chrome string) (trialResult, error) {
	var empty trialResult
	profile, err := os.MkdirTemp("", "wf-cdp-profile-")
	if err != nil {
		return empty, err
	}
	defer os.RemoveAll(profile)
	chromeRead, parentWrite, err := os.Pipe()
	if err != nil {
		return empty, err
	}
	defer parentWrite.Close()
	parentRead, chromeWrite, err := os.Pipe()
	if err != nil {
		_ = chromeRead.Close()
		return empty, err
	}
	defer parentRead.Close()
	cmd := exec.Command(chrome,
		"--headless=new", "--no-sandbox", "--disable-dev-shm-usage",
		"--disable-background-networking", "--disable-component-update",
		"--disable-sync", "--disable-extensions", "--no-first-run",
		"--no-default-browser-check", "--remote-debugging-pipe",
		"--user-data-dir="+profile)
	cmd.ExtraFiles = []*os.File{chromeRead, chromeWrite}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = chromeRead.Close()
		_ = chromeWrite.Close()
		return empty, errors.New("headless Chromium did not start")
	}
	_ = chromeRead.Close()
	_ = chromeWrite.Close()
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	if err := checkChromeConfinement(cmd.Process.Pid); err != nil {
		return empty, err
	}
	pipe := &cdpPipe{writer: parentWrite, reader: bufio.NewReaderSize(parentRead, maxCDPFrame),
		replies: make(map[int]cdpMessage)}
	targetReply, err := pipe.command("Target.createTarget", map[string]any{"url": "about:blank"}, "")
	if err != nil {
		return empty, err
	}
	var target struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(targetReply, &target); err != nil || target.TargetID == "" {
		return empty, errors.New("invalid CDP target")
	}
	attachReply, err := pipe.command("Target.attachToTarget", map[string]any{
		"targetId": target.TargetID, "flatten": true,
	}, "")
	if err != nil {
		return empty, err
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(attachReply, &attached); err != nil || attached.SessionID == "" {
		return empty, errors.New("invalid CDP session")
	}
	session := attached.SessionID
	for _, item := range []struct {
		method string
		params any
	}{
		{"Fetch.enable", map[string]any{"patterns": []map[string]string{{"urlPattern": "*", "requestStage": "Request"}}}},
		{"Page.enable", nil},
		{"Runtime.enable", nil},
	} {
		if _, err := pipe.command(item.method, item.params, session); err != nil {
			return empty, err
		}
	}
	if _, err := pipe.command("Page.navigate", map[string]any{"url": syntheticOrigin + "/app/"}, session); err != nil {
		return empty, err
	}
	for !(pipe.result.Loaded && pipe.result.ScriptSeen && pipe.result.APISeen &&
		pipe.result.OutsideImage > 0 && pipe.result.OutsideRedirect > 0) {
		if err := pipe.receive(); err != nil {
			return empty, err
		}
	}
	return pipe.result, nil
}

func checkChromeConfinement(pid int) error {
	status, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return errors.New("cannot inspect Chromium confinement")
	}
	if !strings.Contains(string(status), "NoNewPrivs:\t1\n") ||
		!strings.Contains(string(status), "Seccomp:\t2\n") {
		return errors.New("Chromium did not inherit no-new-privileges and seccomp")
	}
	return nil
}

func (p *cdpPipe) send(method string, params any, session string) (int, error) {
	p.nextID++
	message := map[string]any{"id": p.nextID, "method": method, "params": params}
	if session != "" {
		message["sessionId"] = session
	}
	encoded, err := json.Marshal(message)
	if err != nil || len(encoded) >= maxCDPFrame {
		return 0, errors.New("invalid CDP command")
	}
	encoded = append(encoded, 0)
	for len(encoded) > 0 {
		n, err := p.writer.Write(encoded)
		if err != nil || n == 0 {
			return 0, errors.New("CDP pipe write failed")
		}
		encoded = encoded[n:]
	}
	return p.nextID, nil
}

func (p *cdpPipe) command(method string, params any, session string) (json.RawMessage, error) {
	id, err := p.send(method, params, session)
	if err != nil {
		return nil, err
	}
	for {
		if response, ok := p.replies[id]; ok {
			delete(p.replies, id)
			if len(response.Error) != 0 {
				return nil, errors.New("CDP command rejected")
			}
			return response.Result, nil
		}
		if err := p.receive(); err != nil {
			return nil, err
		}
	}
}

func (p *cdpPipe) receive() error {
	frame, err := p.reader.ReadSlice(0)
	if err != nil || len(frame) == 0 || len(frame) > maxCDPFrame {
		return errors.New("CDP frame is missing or too large")
	}
	var message cdpMessage
	if err := json.Unmarshal(frame[:len(frame)-1], &message); err != nil {
		return errors.New("invalid CDP frame")
	}
	if message.ID != 0 {
		if len(p.replies) >= maxRequests*2 {
			return errors.New("too many CDP replies")
		}
		p.replies[message.ID] = message
		return nil
	}
	switch message.Method {
	case "Fetch.requestPaused":
		return p.fulfill(message)
	case "Page.loadEventFired":
		p.result.Loaded = true
	case "Runtime.consoleAPICalled":
		var event struct {
			Args []struct {
				Value string `json:"value"`
			} `json:"args"`
		}
		if err := json.Unmarshal(message.Params, &event); err != nil {
			return errors.New("invalid CDP console event")
		}
		for _, arg := range event.Args {
			p.result.ScriptSeen = p.result.ScriptSeen || arg.Value == "wf-script"
			p.result.APISeen = p.result.APISeen || arg.Value == "wf-api-synthetic"
		}
	}
	return nil
}

func (p *cdpPipe) fulfill(message cdpMessage) error {
	var event struct {
		RequestID string `json:"requestId"`
		Request   struct {
			URL    string `json:"url"`
			Method string `json:"method"`
		} `json:"request"`
	}
	if err := json.Unmarshal(message.Params, &event); err != nil || event.RequestID == "" {
		return errors.New("invalid intercepted CDP request")
	}
	if p.requests >= maxRequests {
		return errors.New("CDP request budget exhausted")
	}
	p.requests++
	var body, contentType, location string
	if event.Request.Method == "GET" {
		switch event.Request.URL {
		case syntheticOrigin + "/app/":
			p.result.Document++
			body, contentType = `<html><body><script src="/main.js"></script><img src="http://outside.test/x"></body></html>`, "text/html"
		case syntheticOrigin + "/main.js":
			p.result.Script++
			body, contentType = `console.log('wf-script'); fetch('/api').then(r => r.text()).then(x => console.log('wf-api-' + x)); fetch('/redirect')`, "application/javascript"
		case syntheticOrigin + "/api":
			p.result.API++
			body, contentType = "synthetic", "text/plain"
		case syntheticOrigin + "/redirect":
			p.result.Redirect++
			location = "http://outside.test/secret"
		case "http://outside.test/x":
			p.result.OutsideImage++
		case "http://outside.test/secret":
			p.result.OutsideRedirect++
		}
	}
	if body == "" && location == "" {
		_, err := p.send("Fetch.failRequest", map[string]any{
			"requestId": event.RequestID, "errorReason": "BlockedByClient",
		}, message.SessionID)
		return err
	}
	headers := []map[string]string{{"name": "Content-Type", "value": contentType}}
	status := 200
	if location != "" {
		status = 302
		headers = []map[string]string{{"name": "Location", "value": location}}
	}
	_, err := p.send("Fetch.fulfillRequest", map[string]any{
		"requestId": event.RequestID, "responseCode": status,
		"responseHeaders": headers, "body": base64.StdEncoding.EncodeToString([]byte(body)),
	}, message.SessionID)
	return err
}

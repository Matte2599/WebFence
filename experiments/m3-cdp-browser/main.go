//go:build m3cdplab && linux

// This optional Linux lab mediates synthetic CDP requests through the
// managed gate and broker. It is not a desktop browser or scanner.
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Matte2599/WebFence/internal/browser"
)

const (
	maxCDPFrame = 64 << 10
	maxRequests = 10
	maxCDPBody  = 16 << 10
)

type helperConfig struct {
	Chrome        string `json:"chrome"`
	Origin        string `json:"origin"`
	CanaryAddress string `json:"canary_address"`
	DeniedFile    string `json:"denied_file"`
	BrokerFDs     int    `json:"broker_fds"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type trialResult struct {
	Loaded             bool `json:"loaded"`
	ScriptSeen         bool `json:"script_seen"`
	APISeen            bool `json:"api_seen"`
	RedirectBlocked    bool `json:"redirect_blocked"`
	RevokedBlocked     bool `json:"revoked_blocked"`
	AfterRevokedDenied bool `json:"after_revoked_denied"`
	AllowedFileLoaded  bool `json:"allowed_file_loaded"`
	DeniedFileBlocked  bool `json:"denied_file_blocked"`
	SecureContext      bool `json:"secure_context"`
	Document           int  `json:"document"`
	Script             int  `json:"script"`
	API                int  `json:"api"`
	Redirect           int  `json:"redirect"`
	Revoked            int  `json:"revoked"`
	AfterRevoked       int  `json:"after_revoked"`
	OutsideImage       int  `json:"outside_image"`
	OutsideRedirect    int  `json:"outside_redirect"`
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
		fmt.Println("PASS M3 CDP broker fixtures: HTTP(S) gate/broker, outside resource/redirect, revocation, direct TCP, Chromium file and renderer sandbox canaries checked")
	}
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
	if err := decoder.Decode(&extra); err != io.EOF || !filepath.IsAbs(config.Chrome) ||
		config.Origin == "" || config.CanaryAddress == "" || !filepath.IsAbs(config.DeniedFile) ||
		config.BrokerFDs != brokerConnections ||
		config.Username == "" || config.Password == "" {
		return errors.New("invalid CDP helper configuration")
	}
	inherited := make(chan net.Conn, config.BrokerFDs)
	defer func() {
		close(inherited)
		for conn := range inherited {
			_ = conn.Close()
		}
	}()
	for i := range config.BrokerFDs {
		file := os.NewFile(uintptr(3+i), "broker-connection")
		if file == nil {
			return errors.New("missing inherited broker connection")
		}
		conn, err := net.FileConn(file)
		_ = file.Close()
		if err != nil {
			return errors.New("invalid inherited broker connection")
		}
		if _, ok := conn.(*net.UnixConn); !ok {
			_ = conn.Close()
			return errors.New("broker connection is not Unix IPC")
		}
		inherited <- conn
	}
	if err := browser.ApplyHelperNetworkIsolation(); err != nil {
		return err
	}
	canaryHost, canaryPort, err := net.SplitHostPort(config.CanaryAddress)
	port, portErr := strconv.Atoi(canaryPort)
	if err != nil || portErr != nil || canaryHost != "127.0.0.1" || port < 1 || port > 65535 {
		return errors.New("invalid synthetic canary address")
	}
	for _, probe := range []struct{ network, address string }{
		{"tcp4", config.CanaryAddress}, {"tcp6", "[::1]:9"},
	} {
		conn, err := net.DialTimeout(probe.network, probe.address, 100*time.Millisecond)
		if conn != nil {
			_ = conn.Close()
		}
		if !errors.Is(err, syscall.EPERM) {
			return errors.New("direct browser networking is not blocked")
		}
	}
	if body, err := os.ReadFile(config.DeniedFile); err != nil || string(body) != "synthetic-private-file" {
		return errors.New("synthetic private file was unavailable before confinement")
	}
	client := brokerClient(config, inherited)
	result, err := runCDP(config.Chrome, config.Origin, config.DeniedFile, client)
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
	writer     *os.File
	reader     *bufio.Reader
	nextID     int
	replies    map[int]cdpMessage
	result     trialResult
	requests   int
	loadEvents int
	origin     string
	client     *http.Client
}

func runCDP(chrome, origin, deniedFile string, client *http.Client) (trialResult, error) {
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
	// Landlock restricts the calling thread and the Chromium process forked
	// from it. Never release this thread back to Go's scheduler: the helper
	// exits immediately after this single trial.
	runtime.LockOSThread()
	if err := applyCDPFileBoundary(os.Getenv("HOME"), deniedFile); err != nil {
		_ = chromeRead.Close()
		_ = chromeWrite.Close()
		return empty, err
	}
	cmd := exec.Command(chrome,
		"--headless=new", "--disable-dev-shm-usage",
		"--disable-background-networking", "--disable-component-update",
		"--disable-sync", "--disable-extensions", "--no-first-run",
		"--no-default-browser-check", "--remote-debugging-pipe",
		"--user-data-dir="+profile)
	cmd.ExtraFiles = []*os.File{chromeRead, chromeWrite}
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr // Temporary CI diagnostic; remove before merge.
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
		replies: make(map[int]cdpMessage), origin: origin, client: client}
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
	if _, err := pipe.command("Page.navigate", map[string]any{"url": origin + "/app/"}, session); err != nil {
		return empty, err
	}
	for !(pipe.result.Loaded && pipe.result.ScriptSeen && pipe.result.APISeen &&
		pipe.result.RedirectBlocked && pipe.result.RevokedBlocked && pipe.result.AfterRevokedDenied &&
		pipe.result.OutsideImage > 0) {
		if err := pipe.receive(); err != nil {
			return empty, err
		}
	}
	if err := checkRendererSandbox(cmd.Process.Pid); err != nil {
		return empty, err
	}
	if err := pipe.checkBrowserFileBoundary(session, profile, deniedFile); err != nil {
		return empty, err
	}
	return pipe.result, nil
}

// The HTTP(S) fixture is complete before disabling Fetch interception. Only
// two locally created plaintext files are navigated through this CDP session.
func (p *cdpPipe) checkBrowserFileBoundary(session, profile, deniedFile string) error {
	allowedFile := filepath.Join(profile, "allowed-canary.txt")
	const allowedContent = "synthetic-browser-private-allowed"
	if err := os.WriteFile(allowedFile, []byte(allowedContent), 0600); err != nil {
		return errors.New("cannot prepare allowed browser file canary")
	}
	if _, err := p.command("Fetch.disable", nil, session); err != nil {
		return err
	}
	allowedURL := (&url.URL{Scheme: "file", Path: allowedFile}).String()
	priorLoads := p.loadEvents
	response, err := p.command("Page.navigate", map[string]any{"url": allowedURL}, session)
	if err != nil {
		return err
	}
	var navigation struct {
		ErrorText string `json:"errorText"`
	}
	if err := json.Unmarshal(response, &navigation); err != nil || navigation.ErrorText != "" {
		return errors.New("Chromium could not navigate to allowed file canary")
	}
	for p.loadEvents == priorLoads {
		if err := p.receive(); err != nil {
			return err
		}
	}
	response, err = p.command("Runtime.evaluate", map[string]any{
		"expression": "document.body && document.body.innerText", "returnByValue": true,
	}, session)
	if err != nil {
		return err
	}
	var evaluated struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &evaluated); err != nil || evaluated.Result.Value != allowedContent {
		return errors.New("Chromium could not read allowed file canary")
	}
	p.result.AllowedFileLoaded = true
	deniedURL := (&url.URL{Scheme: "file", Path: deniedFile}).String()
	response, err = p.command("Page.navigate", map[string]any{"url": deniedURL}, session)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(response, &navigation); err != nil || navigation.ErrorText != "net::ERR_ACCESS_DENIED" {
		return fmt.Errorf("Chromium denied-file navigation was not blocked by Landlock: %q", navigation.ErrorText)
	}
	p.result.DeniedFileBlocked = true
	return nil
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

// A Chromium renderer must enter a user namespace distinct from its browser
// process. This guards against a successful fixture run with Chromium's
// sandbox silently disabled by a future launch or container change.
func checkRendererSandbox(browserPID int) error {
	browserNamespace, err := os.Readlink(fmt.Sprintf("/proc/%d/ns/user", browserPID))
	if err != nil {
		return errors.New("cannot inspect Chromium user namespace")
	}
	browserStatus, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", browserPID))
	if err != nil {
		return errors.New("cannot inspect Chromium seccomp filters")
	}
	browserFilters := seccompFilterCount(browserStatus)
	if browserFilters < 1 {
		return errors.New("Chromium browser seccomp filter was not observed")
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return errors.New("cannot inspect Chromium renderers")
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == browserPID || !chromeDescendant(pid, browserPID) {
			continue
		}
		command, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil || !strings.Contains(string(command), "--type=renderer") {
			continue
		}
		status, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
		if err != nil || !strings.Contains(string(status), "NoNewPrivs:\t1\n") ||
			!strings.Contains(string(status), "Seccomp:\t2\n") ||
			seccompFilterCount(status) <= browserFilters {
			continue
		}
		rendererNamespace, err := os.Readlink(fmt.Sprintf("/proc/%d/ns/user", pid))
		if err == nil && rendererNamespace != browserNamespace {
			return nil
		}
	}
	return errors.New("Chromium renderer user-namespace sandbox was not observed")
}

func seccompFilterCount(status []byte) int {
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "Seccomp_filters:") {
			count, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Seccomp_filters:")))
			return count
		}
	}
	return 0
}

func chromeDescendant(pid, ancestor int) bool {
	for range 8 {
		status, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
		if err != nil {
			return false
		}
		var parent int
		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "PPid:") {
				parent, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "PPid:")))
				break
			}
		}
		if parent == ancestor {
			return true
		}
		if parent <= 1 || parent == pid {
			return false
		}
		pid = parent
	}
	return false
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
		p.loadEvents++
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
			p.result.SecureContext = p.result.SecureContext || arg.Value == "wf-secure-true"
			p.result.APISeen = p.result.APISeen || arg.Value == "wf-api-synthetic"
			p.result.RedirectBlocked = p.result.RedirectBlocked || arg.Value == "wf-redirect-502"
			p.result.RevokedBlocked = p.result.RevokedBlocked || arg.Value == "wf-revoked-502"
			p.result.AfterRevokedDenied = p.result.AfterRevokedDenied || arg.Value == "wf-after-revoke-403"
		}
	}
	return nil
}

func (p *cdpPipe) fulfill(message cdpMessage) error {
	var event struct {
		RequestID    string `json:"requestId"`
		ResourceType string `json:"resourceType"`
		Request      struct {
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
	request, err := http.NewRequest(event.Request.Method, event.Request.URL, nil)
	if err != nil || p.client == nil {
		return p.failRequest(event.RequestID, message.SessionID)
	}
	destination := "image"
	switch event.ResourceType {
	case "Document":
		destination = "document"
	case "XHR", "Fetch":
		destination = "empty"
	case "Script":
		destination = "script"
	}
	request.Header.Set("Sec-Fetch-Dest", destination)
	response, err := p.client.Do(request)
	if err != nil {
		return p.failRequest(event.RequestID, message.SessionID)
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxCDPBody+1))
	_ = response.Body.Close()
	if readErr != nil || len(body) > maxCDPBody {
		return p.failRequest(event.RequestID, message.SessionID)
	}
	if event.Request.Method == http.MethodGet {
		outside := "http://outside.test"
		if strings.HasPrefix(p.origin, "https://") {
			outside = "https://outside.test"
		}
		switch event.Request.URL {
		case p.origin + "/app/":
			if response.StatusCode == http.StatusOK {
				p.result.Document++
			}
		case p.origin + "/app/main.js":
			if response.StatusCode == http.StatusOK {
				p.result.Script++
			}
		case p.origin + "/app/api":
			if response.StatusCode == http.StatusOK {
				p.result.API++
			}
		case p.origin + "/app/redirect":
			if response.StatusCode == http.StatusBadGateway {
				p.result.Redirect++
			}
		case p.origin + "/app/slow":
			if response.StatusCode == http.StatusBadGateway {
				p.result.Revoked++
			}
		case p.origin + "/app/after-revoke":
			if response.StatusCode == http.StatusForbidden {
				p.result.AfterRevoked++
			}
		case outside + "/x":
			if response.StatusCode == http.StatusForbidden {
				p.result.OutsideImage++
			}
		case outside + "/secret":
			p.result.OutsideRedirect++
		}
	}
	headers := []map[string]string{{"name": "Content-Type", "value": response.Header.Get("Content-Type")}}
	_, err = p.send("Fetch.fulfillRequest", map[string]any{
		"requestId": event.RequestID, "responseCode": response.StatusCode,
		"responseHeaders": headers, "body": base64.StdEncoding.EncodeToString(body),
	}, message.SessionID)
	return err
}

func (p *cdpPipe) failRequest(id, session string) error {
	_, err := p.send("Fetch.failRequest", map[string]any{
		"requestId": id, "errorReason": "BlockedByClient",
	}, session)
	return err
}

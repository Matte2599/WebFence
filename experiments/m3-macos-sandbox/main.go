//go:build darwin

// This optional lab checks whether a network-denied macOS App Sandbox can
// initialize a headless Chromium CDP session. It uses no external target.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const maxCDPFrame = 64 << 10
const probeBundleID = "org.webfence.m3-sandbox-probe"

type boundedLog struct {
	mu   sync.Mutex
	data []byte
}

func (log *boundedLog) Write(p []byte) (int, error) {
	log.mu.Lock()
	defer log.mu.Unlock()
	if len(log.data) < 8192 {
		log.data = append(log.data, p[:min(len(p), 8192-len(log.data))]...)
	}
	return len(p), nil
}

func (log *boundedLog) category() string {
	log.mu.Lock()
	defer log.mu.Unlock()
	if strings.Contains(string(log.data), "bootstrap_check_in") {
		return "mach_bootstrap_denied"
	}
	if strings.Contains(string(log.data), "Permission denied") ||
		strings.Contains(string(log.data), "Operation not permitted") {
		return "chrome_permission_denied"
	}
	return "cdp_invalid_response"
}

type trialResult struct {
	TCPDenied     bool   `json:"tcp_denied"`
	ChromeStarted bool   `json:"chrome_started"`
	Failure       string `json:"failure,omitempty"`
}

func main() {
	var err error
	switch {
	case len(os.Args) == 1:
		err = runParent()
	case len(os.Args) == 4 && os.Args[1] == "--sandboxed-child":
		err = runChild(os.Args[2], os.Args[3])
	default:
		err = errors.New("invalid lab invocation")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "M3 macOS sandbox lab failed:", err)
		os.Exit(1)
	}
}

func runParent() error {
	child := os.Getenv("WF_M3_SANDBOXED_PROBE")
	chrome := os.Getenv("WF_M3_CHROME")
	if !filepath.IsAbs(child) || !filepath.IsAbs(chrome) || child == chrome {
		return errors.New("absolute probe and Chromium paths required")
	}
	for _, path := range []string{child, chrome} {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("probe or Chromium executable is unavailable")
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return errors.New("cannot open synthetic loopback canary")
	}
	defer listener.Close()
	hits := make(chan struct{}, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
			hits <- struct{}{}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, child, "--sandboxed-child", listener.Addr().String(), chrome)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if ctx.Err() != nil {
		return errors.New("sandboxed probe timed out")
	}
	if err != nil {
		return fmt.Errorf("sandboxed probe exited before reporting: %v", err)
	}
	if len(output) > 4096 {
		return errors.New("sandboxed probe result is too large")
	}
	var result trialResult
	if err := json.Unmarshal(output, &result); err != nil {
		return errors.New("invalid sandboxed probe result")
	}
	select {
	case <-hits:
		return errors.New("sandboxed probe reached the loopback canary")
	default:
	}
	if !result.TCPDenied {
		return errors.New("App Sandbox did not deny direct TCP")
	}
	if !result.ChromeStarted {
		return fmt.Errorf("App Sandbox denied TCP, but Chromium CDP did not start (%s)", result.Failure)
	}
	fmt.Println("PASS M3 macOS App Sandbox: direct TCP denied and Chromium CDP initialized")
	return nil
}

func runChild(address, chrome string) error {
	result := trialResult{}
	host, _, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" || !filepath.IsAbs(chrome) {
		return errors.New("invalid synthetic lab configuration")
	}
	conn, err := net.DialTimeout("tcp4", address, time.Second)
	if conn != nil {
		_ = conn.Close()
	}
	if !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.EACCES) {
		return json.NewEncoder(os.Stdout).Encode(result)
	}
	result.TCPDenied = true
	result.ChromeStarted, result.Failure = checkChromeCDP(chrome)
	return json.NewEncoder(os.Stdout).Encode(result)
}

func checkChromeCDP(chrome string) (bool, string) {
	home := os.Getenv("HOME")
	containerTemp := filepath.Join(home, "Library", "Containers",
		probeBundleID, "Data", "tmp")
	if strings.Contains(home, "/Library/Containers/") {
		containerTemp = filepath.Join(home, "tmp")
	}
	profile, err := os.MkdirTemp(containerTemp, "wf-m3-macos-chrome-")
	if err != nil {
		if _, statErr := os.Stat(containerTemp); statErr != nil {
			return false, "container_temp_unavailable"
		}
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			return false, "profile_denied"
		}
		return false, "profile_unavailable"
	}
	defer os.RemoveAll(profile)
	chromeRead, parentWrite, err := os.Pipe()
	if err != nil {
		return false, "pipe_unavailable"
	}
	defer parentWrite.Close()
	parentRead, chromeWrite, err := os.Pipe()
	if err != nil {
		_ = chromeRead.Close()
		return false, "pipe_unavailable"
	}
	defer parentRead.Close()
	cmd := exec.Command(chrome,
		"--headless=new", "--disable-background-networking", "--disable-component-update",
		"--disable-sync", "--disable-extensions", "--no-first-run", "--no-default-browser-check",
		"--remote-debugging-pipe", "--user-data-dir="+profile)
	cmd.ExtraFiles = []*os.File{chromeRead, chromeWrite}
	cmd.Stdout = io.Discard
	chromeLog := &boundedLog{}
	cmd.Stderr = chromeLog
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		_ = chromeRead.Close()
		_ = chromeWrite.Close()
		return false, "chrome_start_failed"
	}
	_ = chromeRead.Close()
	_ = chromeWrite.Close()
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	waitConsumed := false
	defer func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Process.Kill()
		if !waitConsumed {
			<-waited
		}
	}()
	if _, err := parentWrite.Write([]byte("{\"id\":1,\"method\":\"Browser.getVersion\"}\x00")); err != nil {
		return false, "cdp_write_failed"
	}
	replies := make(chan string, 1)
	go func() {
		reader := bufio.NewReaderSize(parentRead, maxCDPFrame)
		for range 8 {
			frame, err := reader.ReadSlice(0)
			if errors.Is(err, io.EOF) {
				replies <- "cdp_pipe_closed"
				return
			}
			if err != nil || len(frame) == 0 || len(frame) > maxCDPFrame {
				replies <- "cdp_invalid_frame"
				return
			}
			var message struct {
				ID     int             `json:"id"`
				Result json.RawMessage `json:"result"`
			}
			if json.Unmarshal(frame[:len(frame)-1], &message) == nil && message.ID == 1 {
				if len(message.Result) == 0 {
					replies <- "cdp_error_response"
				} else {
					replies <- ""
				}
				return
			}
		}
		replies <- "cdp_reply_missing"
	}()
	select {
	case failure := <-replies:
		if failure == "" {
			return true, ""
		}
		if category := chromeLog.category(); category != "cdp_invalid_response" {
			return false, category
		}
		if failure == "cdp_pipe_closed" {
			select {
			case exitErr := <-waited:
				waitConsumed = true
				if exitErr != nil {
					return false, fmt.Sprintf("chrome_exit_%v", exitErr)
				}
				return false, "chrome_exited_without_cdp"
			case <-time.After(time.Second):
			}
		}
		return false, failure
	case <-time.After(8 * time.Second):
		if category := chromeLog.category(); category != "cdp_invalid_response" {
			return false, category
		}
		return false, "cdp_timeout"
	}
}

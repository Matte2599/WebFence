//go:build m3cdplab && linux

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestBrokerIPCForwardsAbsoluteHTTPSWithoutConnect(t *testing.T) {
	child, parent := net.Pipe()
	defer parent.Close()
	inherited := make(chan net.Conn, 1)
	inherited <- child
	seen := make(chan *http.Request, 1)
	errors := make(chan error, 1)
	go func() {
		request, err := http.ReadRequest(bufio.NewReader(parent))
		if err != nil {
			errors <- err
			return
		}
		seen <- request
		_, err = io.WriteString(parent, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nok")
		errors <- err
	}()
	client := brokerClient(helperConfig{Username: "webfence", Password: "fixture"}, inherited)
	request, err := http.NewRequest(http.MethodGet, "https://site.test:443/app/", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || string(body) != "ok" {
		t.Fatalf("broker IPC response: %d %q %v", response.StatusCode, body, err)
	}
	forwarded := <-seen
	if err := <-errors; err != nil || forwarded.Method != http.MethodGet ||
		forwarded.URL.String() != "https://site.test:443/app/" ||
		forwarded.Header.Get("Proxy-Authorization") != "Basic d2ViZmVuY2U6Zml4dHVyZQ==" {
		t.Fatalf("absolute HTTPS IPC request: method=%s URL=%s err=%v", forwarded.Method, forwarded.URL, err)
	}
}

func TestCDPRejectsOversizedFrame(t *testing.T) {
	pipe := &cdpPipe{reader: bufio.NewReaderSize(
		bytes.NewReader(append(bytes.Repeat([]byte{'x'}, maxCDPFrame), 0)), maxCDPFrame)}
	if err := pipe.receive(); err == nil {
		t.Fatal("oversized untrusted CDP frame accepted")
	}
}

func TestCDPUnknownRequestsConsumeBudget(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	defer write.Close()
	pipe := &cdpPipe{writer: write, client: &http.Client{Transport: deniedTransport{}}}
	for i := 0; i < maxRequests; i++ {
		params, err := json.Marshal(map[string]any{
			"requestId": "request", "request": map[string]string{
				"url": "http://unapproved.test/" + strings.Repeat("x", i), "method": "GET",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := pipe.fulfill(cdpMessage{SessionID: "test", Params: params}); err != nil {
			t.Fatalf("blocked request %d: %v", i, err)
		}
	}
	params := json.RawMessage(`{"requestId":"excess","request":{"url":"http://unapproved.test/","method":"GET"}}`)
	if err := pipe.fulfill(cdpMessage{SessionID: "test", Params: params}); err == nil {
		t.Fatal("request after budget exhaustion accepted")
	}
	if pipe.requests != maxRequests {
		t.Fatalf("requests used = %d, want %d", pipe.requests, maxRequests)
	}
}

type deniedTransport struct{}

func (deniedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusForbidden, Header: make(http.Header),
		Body: io.NopCloser(strings.NewReader("denied"))}, nil
}

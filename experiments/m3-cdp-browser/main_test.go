//go:build m3cdplab && linux

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

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
	pipe := &cdpPipe{writer: write}
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

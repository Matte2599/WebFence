//go:build windows

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCDPBoundary(t *testing.T) {
	valid := `{"id":1,"result":{"product":"fixture/1"}}` + "\x00" +
		`{"id":2,"result":{"targetId":"target"}}` + "\x00" +
		`{"id":3,"result":{"sessionId":"session"}}` + "\x00" +
		`{"id":4,"result":{"result":{"type":"string","value":"local fixture"}}}` + "\x00"
	tests := []struct {
		name, data string
		valid      bool
	}{
		{"synthetic DOM", valid, true},
		{"event before response", `{"method":"Target.targetCreated"}` + "\x00" + valid, true},
		{"closed pipe", "", false},
		{"missing target", strings.Replace(valid, `"targetId":"target"`, `"targetId":""`, 1), false},
		{"missing session", strings.Replace(valid, `"sessionId":"session"`, `"sessionId":""`, 1), false},
		{"oversized frame", strings.Repeat("x", 64<<10) + "\x00", false},
		{"protocol error", `{"id":1,"error":{"code":-1}}` + "\x00", false},
		{"DOM mismatch", strings.Replace(valid, "local fixture", "wrong", 1), false},
		{"script exception", strings.Replace(valid, `"type":"string","value":"local fixture"`, `"type":"undefined"`, 1), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			err := checkBrowserCDP(strings.NewReader(test.data), &output)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, err)
			}
			if test.valid && !strings.Contains(output.String(), `"sessionId":"session"`) {
				t.Fatal("evaluation omitted attached session")
			}
		})
	}
}

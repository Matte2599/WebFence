package cdppipe

import (
	"bytes"
	"strings"
	"testing"
)

func TestNestedEventKeepsPendingReply(t *testing.T) {
	frames := `{"method":"Fetch.requestPaused","sessionId":"page","params":{}}` + "\x00" +
		`{"id":1,"sessionId":"page","result":{"ready":true}}` + "\x00" +
		`{"id":2,"sessionId":"page","result":{}}` + "\x00"
	var output bytes.Buffer
	c := New(strings.NewReader(frames), &output)
	c.Handler = func(event Message) error {
		return c.Call("Fetch.fulfillRequest", struct{}{}, event.Session, &struct{}{})
	}
	var result struct {
		Ready bool `json:"ready"`
	}
	if err := c.Call("Page.navigate", struct{}{}, "page", &result); err != nil || !result.Ready {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestCDPRejectsMalformedAndUnexpectedReplies(t *testing.T) {
	for _, frame := range []string{
		`{"id":2,"result":{}}` + "\x00", `{"id":1,"sessionId":"other","result":{}}` + "\x00",
		`{"id":1,"result":null}` + "\x00", `{"id":1,"error":{"code":-1}}` + "\x00",
		strings.Repeat("x", MaxFrame) + "\x00", strings.Repeat("{}\x00", 128), "",
	} {
		var output bytes.Buffer
		if New(strings.NewReader(frame), &output).Call("fixture", struct{}{}, "", &struct{}{}) == nil {
			t.Fatal("invalid frame accepted")
		}
	}
}

package cdppipe

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Matte2599/WebFence/experiments/internal/headlessfixture"
)

// Mediated runs the synthetic HTTP(S) page through the managed broker. It
// never continues a paused request to Chromium's network stack.
func (c *Client) Mediated(session string) error {
	for _, scheme := range []string{"http", "https"} {
		if err := c.mediatedScheme(session, scheme); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) mediatedScheme(session, scheme string) error {
	f, err := headlessfixture.New(scheme)
	if err != nil {
		return err
	}
	defer f.Close()
	requests := 0
	c.Handler = func(event Message) error {
		if event.Method != "Fetch.requestPaused" {
			return nil
		}
		requests++
		if requests > 16 || event.Session != session {
			return errors.New("CDP intercepted request session/budget")
		}
		var paused struct {
			ID      string `json:"requestId"`
			Type    string `json:"resourceType"`
			Request struct {
				URL         string `json:"url"`
				Method      string `json:"method"`
				HasPostData bool   `json:"hasPostData"`
				PostData    string `json:"postData"`
			} `json:"request"`
		}
		if json.Unmarshal(event.Params, &paused) != nil || paused.ID == "" || len(paused.ID) > 1024 {
			return errors.New("CDP malformed intercepted request")
		}
		if paused.Request.HasPostData || paused.Request.PostData != "" {
			return c.Call("Fetch.failRequest", map[string]any{"requestId": paused.ID, "errorReason": "BlockedByClient"}, session, &struct{}{})
		}
		reply := f.Forward(context.Background(), paused.Request.Method, paused.Request.URL, paused.Type)
		headers := []map[string]string{}
		for name, values := range reply.Header {
			for _, value := range values {
				headers = append(headers, map[string]string{"name": name, "value": value})
			}
		}
		return c.Call("Fetch.fulfillRequest", map[string]any{"requestId": paused.ID, "responseCode": reply.Status,
			"responseHeaders": headers, "body": base64.StdEncoding.EncodeToString(reply.Body)}, session, &struct{}{})
	}
	defer func() { c.Handler = nil }()
	if err := c.Call("Fetch.enable", map[string]any{"patterns": []map[string]string{{"urlPattern": "*", "requestStage": "Request"}}}, session, &struct{}{}); err != nil {
		return err
	}
	var navigation struct {
		Error string `json:"errorText"`
	}
	if err := c.Call("Page.navigate", map[string]string{"url": f.Origin + "/app/"}, session, &navigation); err != nil {
		return err
	}
	if navigation.Error != "" {
		return errors.New("mediated navigation failed")
	}
	secure := "false"
	if scheme == "https" {
		secure = "true"
	}
	want := f.Origin + "|" + secure + "||synthetic|502|502|403"
	for range 100 {
		var result struct {
			Exception json.RawMessage `json:"exceptionDetails"`
			Result    struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"result"`
		}
		if err := c.Call("Runtime.evaluate", map[string]any{"expression": "window.wfResult||''", "returnByValue": true}, session, &result); err != nil {
			return err
		}
		if len(result.Exception) != 0 || result.Result.Type != "string" {
			return errors.New("mediated JavaScript exception/type")
		}
		if result.Result.Value == want {
			if err := c.Call("Fetch.disable", struct{}{}, session, &struct{}{}); err != nil {
				return err
			}
			return f.Verify()
		}
		if strings.Contains(result.Result.Value, "failed") {
			return errors.New("mediated page script failed")
		}
		time.Sleep(20 * time.Millisecond)
	}
	return errors.New("mediated page result deadline")
}

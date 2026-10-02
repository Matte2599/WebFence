// Package cdppipe is a bounded CDP pipe client for synthetic experiments.
package cdppipe

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
)

const MaxFrame = 64 << 10

type Message struct {
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Session string          `json:"sessionId"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

type Client struct {
	reader  *bufio.Reader
	writer  io.Writer
	next    int
	waiting map[int]string
	replies map[int]Message
	events  int
	Handler func(Message) error
}

func New(input io.Reader, output io.Writer) *Client {
	return &Client{reader: bufio.NewReaderSize(input, MaxFrame), writer: output, waiting: make(map[int]string), replies: make(map[int]Message)}
}

func (c *Client) Call(method string, params any, session string, result any) error {
	if len(c.waiting) >= 8 {
		return errors.New("CDP nesting limit")
	}
	c.next++
	id := c.next
	c.waiting[id] = session
	defer delete(c.waiting, id)
	request := map[string]any{"id": id, "method": method, "params": params}
	if session != "" {
		request["sessionId"] = session
	}
	data, err := json.Marshal(request)
	if err != nil || len(data) >= MaxFrame {
		return errors.New("CDP request frame limit")
	}
	data = append(data, 0)
	if n, err := c.writer.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return io.ErrShortWrite
	}
	for range 128 {
		if reply, ok := c.replies[id]; ok {
			delete(c.replies, id)
			if len(reply.Error) != 0 || len(reply.Result) == 0 || string(reply.Result) == "null" {
				return errors.New("CDP command rejected")
			}
			return json.Unmarshal(reply.Result, result)
		}
		frame, err := c.reader.ReadSlice(0)
		if err != nil {
			return err
		}
		var reply Message
		if json.Unmarshal(frame[:len(frame)-1], &reply) != nil {
			return errors.New("CDP malformed frame")
		}
		if reply.ID != 0 {
			want, ok := c.waiting[reply.ID]
			if !ok || reply.Session != want {
				return errors.New("CDP unexpected response session/id")
			}
			if _, exists := c.replies[reply.ID]; exists {
				return errors.New("CDP duplicate response")
			}
			c.replies[reply.ID] = reply
		} else {
			c.events++
			if c.events > 1024 {
				return errors.New("CDP event budget exceeded")
			}
			if c.Handler != nil {
				if err := c.Handler(reply); err != nil {
					return err
				}
			}
		}
	}
	return errors.New("CDP message budget exceeded")
}

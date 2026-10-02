// The parent-side synthetic broker is controlled only through inherited pipes.
// It accepts a fixture scheme, never an external target URL or credentials.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Matte2599/WebFence/experiments/internal/headlessfixture"
)

func run(input io.Reader, output io.Writer, scheme string) error {
	f, err := headlessfixture.New(scheme)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(output)
	if err := encoder.Encode(map[string]string{"origin": f.Origin}); err != nil {
		return err
	}
	reader := bufio.NewReaderSize(input, 16<<10)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for range 20 {
		frame, err := reader.ReadSlice('\n')
		if err != nil {
			return errors.New("invalid broker pipe frame")
		}
		var request struct {
			Method string `json:"method"`
			URL    string `json:"url"`
			Type   string `json:"type"`
			Finish bool   `json:"finish"`
		}
		decoder := json.NewDecoder(bytes.NewReader(frame))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&request) != nil {
			return errors.New("invalid broker pipe request")
		}
		var trailing any
		if decoder.Decode(&trailing) != io.EOF {
			return errors.New("trailing broker pipe data")
		}
		if request.Finish {
			if request.Method != "" || request.URL != "" || request.Type != "" {
				return errors.New("invalid finish frame")
			}
			if err := f.Verify(); err != nil {
				return err
			}
			return encoder.Encode(map[string]bool{"verified": true})
		}
		if err := encoder.Encode(f.Forward(ctx, request.Method, request.URL, request.Type)); err != nil {
			return err
		}
	}
	return errors.New("broker pipe message budget exceeded")
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "fixture scheme required")
		os.Exit(1)
	}
	// Bound a controller that stops sending frames, including while stdin blocks.
	time.AfterFunc(25*time.Second, func() { os.Exit(1) })
	if err := run(os.Stdin, os.Stdout, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "headless broker fixture:", err)
		os.Exit(1)
	}
}

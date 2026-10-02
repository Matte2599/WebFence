package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPipeRejectsBodiesUnknownFieldsAndAmbiguousFrames(t *testing.T) {
	for _, input := range []string{
		"", strings.Repeat("x", 16<<10) + "\n",
		`{"method":"GET","url":"http://outside.test/","body":"synthetic"}` + "\n",
		`{"method":"GET","url":"http://outside.test/","authorization":"synthetic"}` + "\n",
		`{} {}` + "\n", `{"finish":true,"url":"http://outside.test/"}` + "\n",
		`{"finish":true}` + "\n",
	} {
		var output bytes.Buffer
		if run(strings.NewReader(input), &output, "https") == nil {
			t.Fatal("malformed/incomplete fixture accepted")
		}
		if strings.Contains(output.String(), `"verified":true`) {
			t.Fatal("failure reported as verified")
		}
	}
}

// webfence-verify validates a WebFence report without networking or Qt.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Matte2599/WebFence/internal/reporting"
)

func main() {
	bundle := flag.String("bundle", "", "absolute path to a WebFence .wfr bundle")
	trustPath := flag.String("trust", "", "absolute path to an independently trusted public-key registry")
	flag.Parse()
	if !filepath.IsAbs(*bundle) || !filepath.IsAbs(*trustPath) || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: webfence-verify -bundle ABSOLUTE.wfr -trust ABSOLUTE-trust.json")
		os.Exit(2)
	}
	trust, err := reporting.OpenTrustStore(*trustPath)
	if err == nil {
		var result reporting.Verification
		result, err = reporting.Verify(*bundle, trust)
		if err == nil {
			encoded, _ := json.Marshal(result)
			fmt.Println(string(encoded))
			return
		}
	}
	code := "verification_failed"
	switch {
	case errors.Is(err, reporting.ErrUnsigned):
		code = "unsigned"
	case errors.Is(err, reporting.ErrUntrustedKey):
		code = "untrusted_key"
	case errors.Is(err, reporting.ErrRevokedKey):
		code = "revoked_key"
	case errors.Is(err, reporting.ErrInvalidSignature):
		code = "invalid_signature"
	case errors.Is(err, reporting.ErrInvalidBundle):
		code = "invalid_bundle"
	}
	fmt.Fprintln(os.Stderr, code)
	os.Exit(1)
}

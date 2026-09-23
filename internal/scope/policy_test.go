package scope_test

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/Matte2599/WebFence/internal/scope"
)

func TestExactOrigins(t *testing.T) {
	p, err := scope.New([]string{"https://EXAMPLE.invalid/", "http://127.0.0.1:8080", "https://[2001:db8::1]"})
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"https://example.invalid/a", "HTTPS://EXAMPLE.INVALID:443/a", "http://127.0.0.1:8080/", "https://[2001:0db8:0:0:0:0:0:1]:443/a"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := p.Check(raw); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, raw := range []string{"http://example.invalid/a", "https://example.invalid:444/a", "https://sub.example.invalid/a", "https://example.invalid.evil.invalid", "https://other.invalid", "http://127.0.0.2:8080", "https://[2001:db8::2]"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := p.Check(raw); !errors.Is(err, scope.ErrOutOfScope) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestInvalidURLs(t *testing.T) {
	p, _ := scope.New([]string{"https://example.invalid"})
	for _, raw := range []string{
		"", "/relative", "//example.invalid", "https:example.invalid", "file:///tmp/a", "ftp://example.invalid",
		"https://user:SECRET@example.invalid", "https://example.invalid@evil.invalid", "https://example.invalid/#fragment", "https://example.invalid/#",
		" https://example.invalid", "https://example.invalid/\n", "https://example.invalid/\\evil", "https://example.invalid/\u00a0", "https://example.invalid/\xff",
		"https://example.invalid:", "https://example.invalid:0", "https://example.invalid:65536", "https://example.invalid:0443", "https://example.invalid:+443",
		"https://example.invalid.", "https://*.example.invalid", "https://exämple.invalid", "https://bad_name.invalid", "https://-bad.invalid", "https://bad-.invalid", "https://bad..invalid",
		"https://%65xample.invalid", "https://example.invalid/%zz", "https://example.invalid:443:443",
		"http://127.1", "http://2130706433", "http://0177.0.0.1", "http://0x7f000001", "http://127.0.0.0x1",
		"http://[127.0.0.1]", "http://::1", "http://[::ffff:127.0.0.1]", "http://[fe80::1%25en0]",
		"https://" + strings.Repeat("a", 64) + ".invalid", "https://example.invalid/" + strings.Repeat("x", scope.MaxURLBytes),
	} {
		t.Run(raw, func(t *testing.T) {
			u, err := p.Check(raw)
			if u != nil || !errors.Is(err, scope.ErrInvalidURL) {
				t.Fatalf("URL=%v error=%v", u, err)
			}
		})
	}
}

func TestConfigurationDoesNotSilentlyBroadenScope(t *testing.T) {
	for _, origins := range [][]string{nil, {}, {"https://example.invalid/private"}, {"https://example.invalid/?token=SECRET"}, {"https://example.invalid/?"}, {"https://example.invalid/%2f"}, {"https://example.invalid", "invalid"}} {
		p, err := scope.New(origins)
		if !errors.Is(err, scope.ErrInvalidScope) {
			t.Fatalf("got %v", err)
		}
		if _, err := p.Check("https://example.invalid"); !errors.Is(err, scope.ErrOutOfScope) {
			t.Fatalf("partial policy escaped: %v", err)
		}
	}
}

func TestPreservesRequestTargetAndOwnsData(t *testing.T) {
	origins := []string{"https://example.invalid"}
	p, _ := scope.New(origins)
	origins[0] = "https://other.invalid"
	raw := "https://EXAMPLE.invalid/a/../b%2Fc?b=2&a=1&a=3&encoded=%2f+%20"
	u, err := p.Check(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.RequestURI() != "/a/../b%2Fc?b=2&a=1&a=3&encoded=%2f+%20" {
		t.Fatal(u.RequestURI())
	}
	u.Host = "other.invalid"
	if _, err := p.Check(raw); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Check("https://other.invalid"); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatal(err)
	}
}

func TestZeroValueAndConcurrentChecks(t *testing.T) {
	var zero scope.Policy
	if _, err := zero.Check("https://example.invalid"); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatal(err)
	}
	p, _ := scope.New([]string{"https://example.invalid"})
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 100 {
				if _, err := p.Check("https://example.invalid/path"); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
}

func FuzzCheck(f *testing.F) {
	for _, raw := range []string{"https://example.invalid/a%2fb?x=1&x=2", "HTTPS://EXAMPLE.INVALID:443/", "http://127.1", "https://example.invalid@other.invalid", "https://example.invalid/\\foo"} {
		f.Add(raw)
	}
	p, _ := scope.New([]string{"https://example.invalid"})
	f.Fuzz(func(t *testing.T, raw string) {
		u, err := p.Check(raw)
		if err != nil {
			if u != nil {
				t.Fatal("URL returned with error")
			}
			return
		}
		if u.Scheme != "https" || u.Host != "example.invalid:443" || u.User != nil || u.Opaque != "" {
			t.Fatal("escaped origin")
		}
		again, err := p.Check(u.String())
		if err != nil || again.RequestURI() != u.RequestURI() {
			t.Fatal("unstable canonical URL")
		}
	})
}

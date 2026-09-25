package scope

import (
	"errors"
	"net/url"
	"testing"
)

func TestRequestPolicyAllowExcludeAndSegmentBoundary(t *testing.T) {
	p, err := NewRequestPolicy([]string{"GET", "HEAD"}, []string{"/docs", "/status"}, []string{"/docs/logout", "/docs/admin"})
	if err != nil {
		t.Fatal(err)
	}
	check := func(method, raw string) error {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return p.Check(method, u)
	}
	for _, raw := range []string{"http://fixture.test/docs", "http://fixture.test/docs/a?x=1", "http://fixture.test/status"} {
		if err := check("GET", raw); err != nil {
			t.Fatalf("allowed path %s: %v", raw, err)
		}
	}
	if err := check("HEAD", "http://fixture.test/docs/a"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"http://fixture.test/document", "http://fixture.test/docs/logout", "http://fixture.test/docs/logout/now", "http://fixture.test/docs/admin"} {
		if err := check("GET", raw); !errors.Is(err, ErrPathDenied) {
			t.Fatalf("path %s: %v", raw, err)
		}
	}
	if err := check("POST", "http://fixture.test/docs"); !errors.Is(err, ErrMethodDenied) {
		t.Fatal(err)
	}
}

func TestRequestPolicyRejectsAmbiguousRoutesAndInvalidConfiguration(t *testing.T) {
	for _, prefix := range []string{"", "docs", "/docs/", "/docs//a", "/docs/../admin", "/docs/%2fadmin", "/docs;admin", "/docs?x=1", "/docs\\admin"} {
		if _, err := NewRequestPolicy([]string{"GET"}, []string{prefix}, nil); !errors.Is(err, ErrRequestPolicy) {
			t.Fatalf("accepted prefix %q: %v", prefix, err)
		}
	}
	for _, methods := range [][]string{nil, {"POST"}, {"GET", "GET"}, {"get"}} {
		if _, err := NewRequestPolicy(methods, []string{"/"}, nil); !errors.Is(err, ErrRequestPolicy) {
			t.Fatalf("accepted methods %q: %v", methods, err)
		}
	}
	p, err := NewRequestPolicy([]string{"GET"}, []string{"/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"http://fixture.test/docs%2fadmin", "http://fixture.test/docs/../admin", "http://fixture.test/docs//admin", "http://fixture.test/docs;%2fadmin", "http://fixture.test/%252e%252e/admin"} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Check("GET", u); !errors.Is(err, ErrAmbiguousPath) {
			t.Fatalf("ambiguous path %q: %v", raw, err)
		}
	}
	if err := (RequestPolicy{}).Check("GET", &url.URL{Scheme: "http", Host: "fixture.test", Path: "/"}); !errors.Is(err, ErrRequestPolicy) {
		t.Fatal(err)
	}
}

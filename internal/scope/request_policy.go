package scope

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrRequestPolicy = errors.New("scope_invalid_request_policy")
	ErrMethodDenied  = errors.New("scope_method_not_allowed")
	ErrPathDenied    = errors.New("scope_path_not_allowed")
	ErrAmbiguousPath = errors.New("scope_ambiguous_path")
)

// RequestPolicy is an immutable, explicit method/path allowlist. It supplements
// a run's origin authorization; its zero value denies all requests. Paths are
// compared on segment boundaries and never rewrite the URL sent to the target.
type RequestPolicy struct {
	methods  map[string]struct{}
	allowed  []string
	excluded []string
}

// NewRequestPolicy accepts only read-only HTTP methods for M1. An allowed
// prefix is mandatory; "/" must be chosen explicitly to cover every path.
// Exclusions take precedence over allows.
func NewRequestPolicy(methods, allowed, excluded []string) (RequestPolicy, error) {
	if len(methods) == 0 || len(methods) > 2 || len(allowed) == 0 || len(allowed) > 64 || len(excluded) > 64 {
		return RequestPolicy{}, ErrRequestPolicy
	}
	p := RequestPolicy{methods: make(map[string]struct{}, len(methods)),
		allowed: make([]string, 0, len(allowed)), excluded: make([]string, 0, len(excluded))}
	for _, method := range methods {
		if method != "GET" && method != "HEAD" {
			return RequestPolicy{}, ErrRequestPolicy
		}
		if _, duplicate := p.methods[method]; duplicate {
			return RequestPolicy{}, ErrRequestPolicy
		}
		p.methods[method] = struct{}{}
	}
	for _, raw := range allowed {
		if !validPolicyPrefix(raw) {
			return RequestPolicy{}, ErrRequestPolicy
		}
		p.allowed = append(p.allowed, raw)
	}
	for _, raw := range excluded {
		if !validPolicyPrefix(raw) {
			return RequestPolicy{}, ErrRequestPolicy
		}
		p.excluded = append(p.excluded, raw)
	}
	return p, nil
}

func (p RequestPolicy) Valid() bool { return len(p.methods) != 0 && len(p.allowed) != 0 }

func (p RequestPolicy) Check(method string, u *url.URL) error {
	if !p.Valid() || u == nil || !u.IsAbs() || u.Host == "" {
		return ErrRequestPolicy
	}
	if _, ok := p.methods[method]; !ok {
		return ErrMethodDenied
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	// Encoded delimiters, double encoding and dot segments can be interpreted
	// differently by proxies/frameworks. Leave them uncovered in this alpha.
	if !validRequestPath(path) {
		return ErrAmbiguousPath
	}
	for _, prefix := range p.excluded {
		if pathMatches(path, prefix) {
			return ErrPathDenied
		}
	}
	for _, prefix := range p.allowed {
		if pathMatches(path, prefix) {
			return nil
		}
	}
	return ErrPathDenied
}

func validPolicyPrefix(path string) bool {
	return len(path) <= 1024 && validRequestPath(path) && (path == "/" || !strings.HasSuffix(path, "/"))
}

func validRequestPath(path string) bool {
	if path == "" || path[0] != '/' || strings.Contains(path, "//") || strings.ContainsAny(path, "%\\;?#") {
		return false
	}
	for _, c := range path {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	for _, part := range strings.Split(path, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}

func pathMatches(path, prefix string) bool {
	return prefix == "/" || path == prefix || strings.HasPrefix(path, prefix+"/")
}

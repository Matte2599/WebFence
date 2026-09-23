// Package scope validates exact HTTP origins without performing network I/O.
// It is only the URL layer: a successful check is not proof of authorization
// and does not replace DNS, destination-IP, method, path or budget policies.
package scope

import (
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxURLBytes = 16 * 1024

// Errors are stable codes; they never include a URL, query or credential.
var (
	ErrInvalidURL   = errors.New("scope_invalid_url")
	ErrInvalidScope = errors.New("scope_invalid_configuration")
	ErrOutOfScope   = errors.New("scope_origin_not_allowed")
)

// Policy is immutable after New and safe for concurrent checks. Its zero value
// denies every origin. No wildcard, DNS alias or implicit subdomain is accepted.
type Policy struct {
	origins map[string]struct{}
}

// New accepts origins only: scheme, host, optional port and optional literal /.
// A path, query or fragment is rejected rather than silently broadening scope.
func New(origins []string) (Policy, error) {
	if len(origins) == 0 {
		return Policy{}, ErrInvalidScope
	}
	p := Policy{origins: make(map[string]struct{}, len(origins))}
	for _, raw := range origins {
		u, err := parse(raw)
		if err != nil || (u.EscapedPath() != "" && u.EscapedPath() != "/") || u.RawQuery != "" || u.ForceQuery {
			return Policy{}, ErrInvalidScope
		}
		p.origins[u.Scheme+"://"+u.Host] = struct{}{}
	}
	return p, nil
}

// Check returns a fresh URL with canonical scheme, host and explicit port.
// Escaped paths and raw queries are preserved, never cleaned or reordered.
// Relative links must be resolved against a trusted base before calling Check;
// the resulting absolute destination must be checked again for every redirect.
func (p Policy) Check(raw string) (*url.URL, error) {
	u, err := parse(raw)
	if err != nil {
		return nil, err
	}
	if _, ok := p.origins[u.Scheme+"://"+u.Host]; !ok {
		return nil, ErrOutOfScope
	}
	return u, nil
}

func parse(raw string) (*url.URL, error) {
	if len(raw) == 0 || len(raw) > MaxURLBytes || !utf8.ValidString(raw) {
		return nil, ErrInvalidURL
	}
	for _, r := range raw {
		if unicode.IsSpace(r) || unicode.IsControl(r) || r == '\\' || r == '#' {
			return nil, ErrInvalidURL
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Opaque != "" || u.Host == "" {
		return nil, ErrInvalidURL
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrInvalidURL
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || strings.Contains(host, "%") {
		return nil, ErrInvalidURL
	}
	port := u.Port()
	// Enforce one unambiguous authority grammar, including bracketed IPv6.
	authorityHost := host
	if strings.HasPrefix(u.Host, "[") {
		ip, err := netip.ParseAddr(host)
		if err != nil || !ip.Is6() || ip.Is4In6() || ip.Zone() != "" {
			return nil, ErrInvalidURL
		}
		authorityHost = "[" + host + "]"
		host = ip.String()
	} else if ip, err := netip.ParseAddr(host); err == nil {
		if !ip.Is4() {
			return nil, ErrInvalidURL
		}
		host = ip.String()
	} else if !validDNSName(host) {
		return nil, ErrInvalidURL
	}
	if port != "" {
		authorityHost += ":" + port
	}
	if strings.ToLower(u.Host) != authorityHost {
		return nil, ErrInvalidURL
	}
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	} else {
		n, err := strconv.ParseUint(port, 10, 16)
		if err != nil || n == 0 || strconv.FormatUint(n, 10) != port {
			return nil, ErrInvalidURL
		}
	}
	u.Host = net.JoinHostPort(host, port)
	return u, nil
}

func validDNSName(host string) bool {
	if len(host) > 253 || strings.HasSuffix(host, ".") {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	// Reject hosts that another URL parser could interpret as legacy IPv4
	// (integer, shortened, octal or hex), rather than as the same DNS name.
	last := labels[len(labels)-1]
	if strings.Trim(last, "0123456789") == "" {
		return false
	}
	if strings.HasPrefix(last, "0x") && strings.Trim(last[2:], "0123456789abcdef") == "" {
		return false
	}
	return true
}

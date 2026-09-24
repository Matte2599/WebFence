package scanner

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"golang.org/x/net/html"
)

const (
	MaxDiscoveryBodyBytes  = 8 << 20
	MaxDiscoveryTokenBytes = 64 << 10
	MaxDiscoveryTokens     = 100000
	MaxDiscoveryReferences = 512
)

var ErrDiscoveryInput = errors.New("scanner_invalid_discovery_input")

// FormSurface records only an action and a method class. Fields and values
// are never read; observing a form does not authorize its submission.
type FormSurface struct {
	Action string
	Method string // GET, POST or OTHER
}

// Surface is ephemeral and contains potentially sensitive URLs. Do not log,
// persist or serialize it as a report. Only its redacted summary is exported
// from RunHeaderLab; no discovered URL is fetched by that run.
type Surface struct {
	Links      []string
	Forms      []FormSurface
	OutOfScope int
	Invalid    int
	Incomplete bool
}

// ObserveHTML extracts links and form actions from a bounded UTF-8 response.
// Every destination is checked against the managed run's exact origins. The
// first base href affects relative resolution, including an external base;
// it cannot enlarge the authorized scope.
func ObserveHTML(permit project.RunScope, finalURL string, body []byte) (Surface, error) {
	base, err := permit.CheckOrigin(finalURL)
	if err != nil {
		return Surface{}, err
	}
	if len(body) > MaxDiscoveryBodyBytes {
		return Surface{}, ErrDiscoveryInput
	}
	if !utf8.Valid(body) {
		return Surface{Incomplete: true}, nil
	}
	documentURL := base
	surface := Surface{}
	z := html.NewTokenizer(bytes.NewReader(body))
	z.SetMaxBuf(MaxDiscoveryTokenBytes)
	seen := make(map[string]struct{})
	baseSeen := false
	baseUsable := true
	type reference struct{ kind, raw, method string }
	references := make([]reference, 0, 16)
	finished := false
	for tokens := 0; tokens < MaxDiscoveryTokens; tokens++ {
		if tokens%1024 == 0 {
			if err := permit.Validate(); err != nil {
				return Surface{}, err
			}
		}
		typeOfToken := z.Next()
		if typeOfToken == html.ErrorToken {
			finished = true
			if !errors.Is(z.Err(), io.EOF) {
				surface.Incomplete = true
			}
			break
		}
		if typeOfToken != html.StartTagToken && typeOfToken != html.SelfClosingTagToken {
			continue
		}
		token := z.Token()
		if token.Data != "base" && token.Data != "a" && token.Data != "form" {
			continue
		}
		attrs := make(map[string]string, len(token.Attr))
		for _, attr := range token.Attr {
			if _, exists := attrs[attr.Key]; !exists {
				attrs[attr.Key] = attr.Val
			}
		}
		if token.Data == "base" {
			if raw, exists := attrs["href"]; exists && !baseSeen {
				baseSeen = true
				resolved, ok := resolveReference(base, raw)
				if !ok || (resolved.Scheme != "http" && resolved.Scheme != "https") {
					baseUsable = false
					surface.Incomplete = true
				} else {
					base = resolved
				}
			}
			continue
		}
		var raw string
		if token.Data == "a" {
			var exists bool
			raw, exists = attrs["href"]
			if !exists || raw == "" || strings.HasPrefix(raw, "#") {
				continue
			}
		} else {
			raw = attrs["action"] // An omitted action addresses this page.
		}
		if len(references) == MaxDiscoveryReferences {
			surface.Incomplete = true
			break
		}
		references = append(references, reference{kind: token.Data, raw: raw, method: attrs["method"]})
	}
	if !finished {
		surface.Incomplete = true
	}
	for _, item := range references {
		resolutionBase := base
		if item.kind == "form" && item.raw == "" {
			resolutionBase = documentURL
		}
		resolved, ok := resolveReference(resolutionBase, item.raw)
		if !ok {
			surface.Invalid++
			continue
		}
		// A bad first base cannot be used for relative destinations. Absolute
		// URLs remain independent of it.
		if !baseUsable && !(item.kind == "form" && item.raw == "") {
			ref, _ := url.Parse(item.raw)
			if !ref.IsAbs() {
				surface.Invalid++
				continue
			}
		}
		checked, checkErr := permit.CheckOrigin(resolved.String())
		if checkErr != nil {
			if errors.Is(checkErr, scope.ErrOutOfScope) {
				surface.OutOfScope++
			} else if errors.Is(checkErr, scope.ErrInvalidURL) {
				surface.Invalid++
			} else {
				return Surface{}, checkErr // revocation/expiry must stop the run
			}
			continue
		}
		if item.kind == "a" {
			key := checked.String()
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				surface.Links = append(surface.Links, key)
			}
		} else {
			method := strings.ToUpper(strings.TrimSpace(item.method))
			if method != "POST" && method != "GET" {
				if method == "" {
					method = "GET"
				} else {
					method = "OTHER"
				}
			}
			surface.Forms = append(surface.Forms, FormSurface{Action: checked.String(), Method: method})
		}
	}
	if err := permit.Validate(); err != nil {
		return Surface{}, err
	}
	return surface, nil
}

func resolveReference(base *url.URL, raw string) (*url.URL, bool) {
	if len(raw) > scope.MaxURLBytes || !utf8.ValidString(raw) {
		return nil, false
	}
	for _, r := range raw {
		if unicode.IsSpace(r) || unicode.IsControl(r) || r == '\\' {
			return nil, false
		}
	}
	ref, err := url.Parse(raw)
	if err != nil || ref.User != nil || ref.Opaque != "" {
		return nil, false
	}
	resolved := base.ResolveReference(ref)
	resolved.Fragment = ""
	resolved.RawFragment = ""
	return resolved, true
}

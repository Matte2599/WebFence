// Package apiimport turns an untrusted OpenAPI document into a bounded,
// offline inventory. It never resolves references or performs network I/O.
package apiimport

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
)

const (
	MaxDocumentBytes = 1 << 20
	MaxJSONDepth     = 32
	MaxJSONTokens    = 20000
	MaxPaths         = 512
	MaxOperations    = 1024
)

var (
	ErrDocument = errors.New("apiimport_invalid_document")
	ErrLimit    = errors.New("apiimport_limit_exceeded")
	versionRE   = regexp.MustCompile(`^3\.(0|1)\.[0-9]+$`)
)

type Disposition string

const (
	Candidate      Disposition = "candidate"
	Template       Disposition = "template"
	Authentication Disposition = "authentication_required"
	Method         Disposition = "method_not_allowed"
	Policy         Disposition = "policy_denied"
	InvalidPath    Disposition = "invalid_path"
	Reference      Disposition = "reference_unresolved"
)

// Route is an ephemeral, non-persisted inventory entry. CandidateURL is set
// only for static read-only routes admitted by the run and request policy.
type Route struct {
	Path         string
	Method       string
	Disposition  Disposition
	CandidateURL string
}

type Inventory struct {
	Version string
	Routes  []Route
}

// Import accepts the origin selected by the operator, not a document server.
// The caller must explicitly opt in before scheduling any candidate URL.
func Import(document []byte, origin string, permit project.RunScope, policy scope.RequestPolicy) (Inventory, error) {
	if len(document) == 0 || !utf8.Valid(document) {
		return Inventory{}, ErrDocument
	}
	if len(document) > MaxDocumentBytes {
		return Inventory{}, ErrLimit
	}
	if err := permit.Validate(); err != nil {
		return Inventory{}, err
	}
	base, err := permit.CheckOrigin(origin)
	if err != nil {
		return Inventory{}, err
	}
	if !pureOrigin(base) || !policy.Valid() {
		return Inventory{}, ErrDocument
	}
	dec := json.NewDecoder(bytes.NewReader(document))
	dec.UseNumber()
	budget := MaxJSONTokens
	value, err := decode(dec, 0, &budget)
	if err != nil {
		return Inventory{}, err
	}
	if _, err = dec.Token(); err != io.EOF {
		return Inventory{}, ErrDocument
	}
	root, ok := value.(map[string]any)
	if !ok {
		return Inventory{}, ErrDocument
	}
	version, ok := root["openapi"].(string)
	if !ok || !versionRE.MatchString(version) {
		return Inventory{}, ErrDocument
	}
	paths, ok := root["paths"].(map[string]any)
	if !ok {
		return Inventory{}, ErrDocument
	}
	if len(paths) > MaxPaths {
		return Inventory{}, ErrLimit
	}
	pathNames := make([]string, 0, len(paths))
	for path := range paths {
		pathNames = append(pathNames, path)
	}
	sort.Strings(pathNames)
	inv := Inventory{Version: version, Routes: make([]Route, 0)}
	for _, path := range pathNames {
		item, ok := paths[path].(map[string]any)
		if !ok {
			return Inventory{}, ErrDocument
		}
		methods := make([]string, 0, len(item))
		for method := range item {
			if isOperation(method) {
				methods = append(methods, method)
			}
		}
		sort.Strings(methods)
		for _, method := range methods {
			if len(inv.Routes) >= MaxOperations {
				return Inventory{}, ErrLimit
			}
			op, ok := item[method].(map[string]any)
			if !ok {
				return Inventory{}, ErrDocument
			}
			route := Route{Path: path, Method: strings.ToUpper(method)}
			switch {
			case item["$ref"] != nil || op["$ref"] != nil:
				route.Disposition = Reference
			case !safePath(path):
				route.Disposition = InvalidPath
			case strings.ContainsAny(path, "{}"):
				route.Disposition = Template
			case route.Method != "GET" && route.Method != "HEAD":
				route.Disposition = Method
			case requiresSecurity(root, op):
				route.Disposition = Authentication
			default:
				candidate := base.Scheme + "://" + base.Host + path
				u, checkErr := permit.CheckOrigin(candidate)
				if checkErr != nil {
					return Inventory{}, checkErr
				}
				if policy.Check(route.Method, u) != nil {
					route.Disposition = Policy
				} else {
					route.Disposition = Candidate
					route.CandidateURL = u.String()
				}
			}
			inv.Routes = append(inv.Routes, route)
		}
	}
	if err := permit.Validate(); err != nil {
		return Inventory{}, err
	}
	return inv, nil
}

func pureOrigin(u *url.URL) bool {
	return u != nil && (u.EscapedPath() == "" || u.EscapedPath() == "/") &&
		u.RawPath == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == ""
}

func safePath(path string) bool {
	if len(path) == 0 || len(path) > 1024 || path[0] != '/' || strings.Contains(path, "//") || strings.ContainsAny(path, `%\\;?#`) {
		return false
	}
	for _, c := range path {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func requiresSecurity(root, op map[string]any) bool {
	security, present := op["security"]
	if !present {
		security, present = root["security"]
	}
	if !present {
		return false
	}
	list, ok := security.([]any)
	return !ok || len(list) != 0
}

func isOperation(s string) bool {
	switch s {
	case "get", "head", "post", "put", "patch", "delete", "options", "trace":
		return true
	}
	return false
}

// decode enforces depth, token and duplicate-key limits before interpreting
// any OpenAPI fields. The byte cap prevents large scalar allocations.
func decode(dec *json.Decoder, depth int, budget *int) (any, error) {
	if depth > MaxJSONDepth || *budget <= 0 {
		return nil, ErrLimit
	}
	*budget--
	token, err := dec.Token()
	if err != nil {
		return nil, ErrDocument
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delim {
	case '{':
		object := make(map[string]any)
		for dec.More() {
			if *budget <= 0 {
				return nil, ErrLimit
			}
			*budget--
			key, err := dec.Token()
			if err != nil {
				return nil, ErrDocument
			}
			name, ok := key.(string)
			if !ok {
				return nil, ErrDocument
			}
			if _, exists := object[name]; exists {
				return nil, ErrDocument
			}
			value, err := decode(dec, depth+1, budget)
			if err != nil {
				return nil, err
			}
			object[name] = value
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim('}') {
			return nil, ErrDocument
		}
		return object, nil
	case '[':
		list := make([]any, 0)
		for dec.More() {
			value, err := decode(dec, depth+1, budget)
			if err != nil {
				return nil, err
			}
			list = append(list, value)
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim(']') {
			return nil, ErrDocument
		}
		return list, nil
	default:
		return nil, ErrDocument
	}
}

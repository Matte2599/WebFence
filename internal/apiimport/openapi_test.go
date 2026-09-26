package apiimport

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
)

func importFixture(t *testing.T) (project.RunScope, scope.RequestPolicy) {
	t.Helper()
	p, err := project.New(project.Draft{
		ID: "m3-api", Name: "Synthetic API", TargetOwner: "Fixture",
		AuthorizationReference: "synthetic", AuthorizationConfirmed: true,
		AuthorizationExpiresAt: time.Now().Add(time.Hour),
		Origins:                []string{"http://api.test:8080"},
	})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := p.BeginRun()
	if err != nil {
		t.Fatal(err)
	}
	permit, err = permit.BindLifecycle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := scope.NewRequestPolicy([]string{"GET", "HEAD"}, []string{"/v1"}, []string{"/v1/private"})
	if err != nil {
		t.Fatal(err)
	}
	return permit, policy
}

func TestImportOfflineInventoryAndPolicy(t *testing.T) {
	permit, policy := importFixture(t)
	doc := []byte(`{
  "openapi":"3.1.1",
  "servers":[{"url":"https://outside.example/hidden"}],
  "security":[{"apiKey":[]}],
  "paths":{
    "/v1/public":{"get":{"security":[]},"head":{"security":[]}},
    "/v1/private/data":{"get":{"security":[]}},
    "/v1/users/{id}":{"get":{"security":[]}},
    "/v1/write":{"post":{"security":[]}},
    "/v1/secure":{"get":{}},
    "/v1/ref":{"$ref":"https://outside.example/path","get":{"security":[]}},
    "/v1/%2e%2e/escape":{"get":{"security":[]}}
  }
}`)
	got, err := Import(doc, "http://api.test:8080", permit, policy)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "3.1.1" || len(got.Routes) != 8 {
		t.Fatalf("unexpected inventory: version=%s routes=%d", got.Version, len(got.Routes))
	}
	want := map[string]Disposition{
		"GET /v1/public": Candidate, "HEAD /v1/public": Candidate,
		"GET /v1/private/data": Policy, "GET /v1/users/{id}": Template,
		"POST /v1/write": Method, "GET /v1/secure": Authentication,
		"GET /v1/ref": Reference, "GET /v1/%2e%2e/escape": InvalidPath,
	}
	for _, route := range got.Routes {
		key := route.Method + " " + route.Path
		if route.Disposition != want[key] {
			t.Errorf("%s: %s, want %s", key, route.Disposition, want[key])
		}
		if route.Disposition == Candidate {
			if route.CandidateURL != "http://api.test:8080"+route.Path {
				t.Errorf("candidate %s: %s", key, route.CandidateURL)
			}
		} else if route.CandidateURL != "" {
			t.Errorf("non-candidate URL leaked for %s", key)
		}
	}
}

func TestImportRejectsAmbiguousAndUnboundedDocuments(t *testing.T) {
	permit, policy := importFixture(t)
	tests := []struct {
		name string
		doc  []byte
		want error
	}{
		{"duplicate key", []byte(`{"openapi":"3.0.4","paths":{"/v1/a":{"get":{},"get":{}}}}`), ErrDocument},
		{"trailing JSON", []byte(`{"openapi":"3.0.4","paths":{}} {}`), ErrDocument},
		{"old version", []byte(`{"openapi":"2.0","paths":{}}`), ErrDocument},
		{"bad operation", []byte(`{"openapi":"3.0.4","paths":{"/v1/a":{"get":null}}}`), ErrDocument},
		{"too large", bytes.Repeat([]byte{'x'}, MaxDocumentBytes+1), ErrLimit},
		{"too deep", []byte(`{"openapi":"3.1.0","paths":{},"x":` + strings.Repeat("[", MaxJSONDepth+1) + `0` + strings.Repeat("]", MaxJSONDepth+1) + `}`), ErrLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Import(tt.doc, "http://api.test:8080", permit, policy)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestImportNeverUsesDocumentServerAsOrigin(t *testing.T) {
	permit, policy := importFixture(t)
	doc := []byte(`{"openapi":"3.0.4","servers":[{"url":"http://api.test:8080/override"}],"paths":{"/v1/a":{"get":{}}}}`)
	if _, err := Import(doc, "https://outside.example", permit, policy); !errors.Is(err, scope.ErrOutOfScope) {
		t.Fatalf("outside origin: %v", err)
	}
	if _, err := Import(doc, "http://api.test:8080/override", permit, policy); !errors.Is(err, ErrDocument) {
		t.Fatalf("origin with path: %v", err)
	}
	if _, err := Import(doc, "http://api.test:8080/%2f", permit, policy); !errors.Is(err, ErrDocument) {
		t.Fatalf("encoded origin path: %v", err)
	}
	got, err := Import(doc, "http://api.test:8080", permit, policy)
	if err != nil || len(got.Routes) != 1 || got.Routes[0].CandidateURL != "http://api.test:8080/v1/a" {
		t.Fatalf("server override: %+v %v", got, err)
	}
}

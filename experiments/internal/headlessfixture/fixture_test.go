package headlessfixture

import (
	"context"
	"testing"
)

func TestManagedHTTPAndHTTPSFixture(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		t.Run(scheme, func(t *testing.T) {
			f, err := New(scheme)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			for _, request := range []struct {
				path, kind string
				status     int
			}{
				{"/app/", "Document", 200}, {"/app/main.js", "Script", 200}, {"/app/api", "Fetch", 200},
				{"/app/redirect", "Fetch", 502}, {"/app/slow", "Fetch", 502}, {"/app/after-revoke", "Fetch", 403},
			} {
				if request.path == "/app/main.js" {
					if f.Forward(context.Background(), "GET", f.Outside+"/outside", "Image").Status != 403 {
						t.Fatal("outside resource allowed")
					}
				}
				reply := f.Forward(context.Background(), "GET", f.Origin+request.path, request.kind)
				if reply.Status != request.status {
					t.Fatalf("%s status=%d", request.path, reply.Status)
				}
				if reply.Header.Get("Set-Cookie") != "" {
					t.Fatal("fixture cookie leaked")
				}
			}
			if err := f.Verify(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

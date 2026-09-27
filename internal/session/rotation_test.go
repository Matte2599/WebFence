package session

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCookieRotationPreservesTwoIdentities(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = r.ParseForm()
			id := r.Form.Get("username")
			if r.Method != http.MethodPost || r.Form.Get("password") != id+"-pass" ||
				(id != "alice" && id != "bob") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: id + "-login", Path: "/private", HttpOnly: true})
		case "/private/verify", "/private/data":
			cookie, err := r.Cookie("sid")
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			id, stage, ok := strings.Cut(cookie.Value, "-")
			if !ok || (id != "alice" && id != "bob") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			switch r.URL.Path {
			case "/private/verify":
				if stage == "login" {
					http.SetCookie(w, &http.Cookie{Name: "sid", Value: id + "-ready", Path: "/private", HttpOnly: true})
				} else if stage != "ready" && stage != "data" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				_, _ = io.WriteString(w, id)
			case "/private/data":
				if stage == "ready" {
					http.SetCookie(w, &http.Cookie{Name: "sid", Value: id + "-data", Path: "/private", HttpOnly: true})
				} else if stage != "data" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				_, _ = io.WriteString(w, "private-"+id)
			}
		default:
			http.NotFound(w, r)
		}
	})
	manager, broker, base := fixtureManager(t, handler)
	for _, id := range []string{"alice", "bob"} {
		s, err := manager.Login(context.Background(), account(id, id+"-ref"))
		if err != nil {
			t.Fatalf("%s login rotation: %v", id, err)
		}
		result, err := s.Fetch(context.Background(), base+"/private/data")
		if err != nil || string(result.Body) != "private-"+id || s.Identity() != id {
			t.Fatalf("%s resource rotation: status=%d err=%v", id, result.StatusCode, err)
		}
		if err := s.Verify(context.Background()); err != nil {
			t.Fatalf("%s final verification: %v", id, err)
		}
		s.Close()
	}
	if broker.RequestsUsed() != 14 {
		t.Fatalf("rotation probes escaped shared budget: %d", broker.RequestsUsed())
	}
}

func TestCookieRotationFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  []string
	}{
		{"domain", []string{"sid=alice-next; Domain=127.0.0.1; Path=/private"}},
		{"wider-path", []string{"sid=alice-next; Path=/"}},
		{"duplicate", []string{"sid=alice-next; Path=/private", "sid=alice-other; Path=/private"}},
		{"deleted", []string{"sid=alice-next; Path=/private; Max-Age=0"}},
		{"identity-changed", []string{"sid=bob-next; Path=/private"}},
		{"repeated-rotation", []string{"sid=alice-next; Path=/private"}},
		{"redirect-confirmation", []string{"sid=alice-next; Path=/private"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var outside atomic.Int32
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/auth/login":
					w.Header().Set("Set-Cookie", "sid=alice-start; Path=/private")
				case "/private/verify":
					cookie, err := r.Cookie("sid")
					if err != nil {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					if tc.name == "repeated-rotation" && cookie.Value == "alice-next" {
						w.Header().Set("Set-Cookie", "sid=alice-again; Path=/private")
					}
					if tc.name == "redirect-confirmation" && cookie.Value == "alice-next" {
						w.Header().Set("Location", "/outside")
						w.WriteHeader(http.StatusFound)
						return
					}
					if cookie.Value == "bob-next" {
						_, _ = io.WriteString(w, "bob")
					} else {
						_, _ = io.WriteString(w, "alice")
					}
				case "/private/data":
					for _, value := range tc.set {
						w.Header().Add("Set-Cookie", value)
					}
					_, _ = io.WriteString(w, "private-alice")
				case "/outside":
					outside.Add(1)
				default:
					http.NotFound(w, r)
				}
			})
			manager, _, base := fixtureManager(t, handler)
			s, err := manager.Login(context.Background(), account("alice", "alice-ref"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			result, err := s.Fetch(context.Background(), base+"/private/data")
			if !errors.Is(err, ErrInconclusive) || result.StatusCode != 0 || len(result.Body) != 0 || outside.Load() != 0 {
				t.Fatalf("unsafe rotation accepted: status=%d err=%v", result.StatusCode, err)
			}
			if err := s.Verify(context.Background()); !errors.Is(err, ErrExpired) {
				t.Fatalf("invalidated session remained usable: %v", err)
			}
		})
	}
}

func TestConcurrentFetchUsesRotatedCookie(t *testing.T) {
	var oldHits, newHits atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			w.Header().Set("Set-Cookie", "sid=alice-old; Path=/private")
		case "/private/verify":
			cookie, err := r.Cookie("sid")
			if err != nil || cookie.Value != "alice-old" && cookie.Value != "alice-new" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, "alice")
		case "/private/data":
			cookie, err := r.Cookie("sid")
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			switch cookie.Value {
			case "alice-old":
				if oldHits.Add(1) != 1 {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Set-Cookie", "sid=alice-new; Path=/private")
			case "alice-new":
				newHits.Add(1)
			default:
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, "private-alice")
		default:
			http.NotFound(w, r)
		}
	})
	manager, _, base := fixtureManager(t, handler)
	s, err := manager.Login(context.Background(), account("alice", "alice-ref"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var wg sync.WaitGroup
	failures := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := s.Fetch(context.Background(), base+"/private/data")
			if err != nil {
				failures <- err
			} else if string(result.Body) != "private-alice" {
				failures <- ErrInconclusive
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatalf("concurrent fetch lost verified identity: %v", err)
	}
	if oldHits.Load() != 1 || newHits.Load() != 11 {
		t.Fatalf("stale cookie reused: old=%d new=%d", oldHits.Load(), newHits.Load())
	}
}

func TestRotationConfirmationConsumesBudget(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			w.Header().Set("Set-Cookie", "sid=alice-old; Path=/private")
		case "/private/verify":
			if _, err := r.Cookie("sid"); err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, "alice")
		case "/private/data":
			w.Header().Set("Set-Cookie", "sid=alice-new; Path=/private")
			_, _ = io.WriteString(w, "private-alice")
		default:
			http.NotFound(w, r)
		}
	})
	manager, broker, base := fixtureManager(t, handler)
	s, err := manager.Login(context.Background(), account("alice", "alice-ref"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for range 26 {
		if _, err := broker.Fetch(context.Background(), base+"/private/verify"); err != nil {
			t.Fatal(err)
		}
	}
	result, err := s.Fetch(context.Background(), base+"/private/data")
	if !errors.Is(err, ErrInconclusive) || result.StatusCode != 0 || broker.RequestsUsed() != 30 {
		t.Fatalf("rotation bypassed budget: status=%d err=%v used=%d", result.StatusCode, err, broker.RequestsUsed())
	}
}

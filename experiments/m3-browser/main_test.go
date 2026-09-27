//go:build m3browserlab

package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateHelperConfig(t *testing.T) {
	good := helperConfig{Origin: "http://site.test:1234", ProxyEndpoint: "http://127.0.0.1:5678",
		Username: "webfence", Password: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}
	if err := validateHelperConfig(good); err != nil {
		t.Fatalf("valid loopback fixture rejected: %v", err)
	}
	for name, change := range map[string]func(*helperConfig){
		"external origin": func(c *helperConfig) { c.Origin = "https://example.com" },
		"origin path":     func(c *helperConfig) { c.Origin = "http://site.test:1234/other" },
		"external proxy":  func(c *helperConfig) { c.ProxyEndpoint = "http://outside.test:5678" },
		"proxy userinfo":  func(c *helperConfig) { c.ProxyEndpoint = "http://u:p@127.0.0.1:5678" },
		"wrong username":  func(c *helperConfig) { c.Username = "guest" },
	} {
		t.Run(name, func(t *testing.T) {
			config := good
			change(&config)
			if err := validateHelperConfig(config); err == nil {
				t.Fatal("invalid helper configuration accepted")
			}
		})
	}
}

func TestValidateSchemeConfig(t *testing.T) {
	// The helper must refuse alternate origins and socket destinations before
	// starting Qt, even if the parent were to send a malformed payload.
	directory, err := os.MkdirTemp("", "wf-config-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	socketPath := filepath.Join(directory, "broker.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	good := helperConfig{Mode: "scheme", Origin: "http://site.test:1234", SocketPath: socketPath,
		Username: "webfence", Password: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}
	if err := validateSchemeConfig(good); err != nil {
		t.Fatalf("valid local scheme configuration rejected: %v", err)
	}
	for name, change := range map[string]func(*helperConfig){
		"external origin": func(c *helperConfig) { c.Origin = "https://outside.test" },
		"TCP proxy":       func(c *helperConfig) { c.ProxyEndpoint = "http://127.0.0.1:80" },
		"relative socket": func(c *helperConfig) { c.SocketPath = "broker.sock" },
		"other socket":    func(c *helperConfig) { c.SocketPath = "/tmp/other.sock" },
		"wrong mode":      func(c *helperConfig) { c.Mode = "other" },
		"wrong secret":    func(c *helperConfig) { c.Password = "short" },
	} {
		t.Run(name, func(t *testing.T) {
			config := good
			change(&config)
			if err := validateSchemeConfig(config); err == nil {
				t.Fatal("invalid scheme configuration accepted")
			}
		})
	}
	if err := os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := validateSchemeConfig(good); err == nil {
		t.Fatal("scheme configuration accepted a nonprivate socket directory")
	}
}

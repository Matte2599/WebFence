//go:build m3browserlab

package main

import "testing"

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

//go:build m3browserlab && linux

package main

import "github.com/Matte2599/WebFence/internal/browser"

func applyLinuxSchemeNetworkIsolation() error { return browser.ApplyHelperNetworkIsolation() }

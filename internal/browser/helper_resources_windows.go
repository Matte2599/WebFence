//go:build windows

package browser

// Windows limits are installed by RunHelper's Job Object before the helper
// receives its configuration. This keeps the same child startup contract.
func ApplyHelperResourceLimits() error { return nil }

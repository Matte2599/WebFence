//go:build linux && !amd64 && !arm64

package browser

// This laboratory filter has been reviewed only for the Linux amd64 and
// arm64 syscall ABIs. Unsupported builds fail closed.
func ApplyHelperNetworkIsolation() error { return ErrHelperFailed }

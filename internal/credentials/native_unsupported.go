//go:build (!darwin && !linux && !windows) || (darwin && !cgo)

package credentials

func nativeBackend(string) backend { return nil }

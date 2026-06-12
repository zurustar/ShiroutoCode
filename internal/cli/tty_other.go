//go:build !darwin && !linux

package cli

// enableCookedUTF8 is a no-op on platforms without termios support.
func enableCookedUTF8(fd int) func() { return func() {} }

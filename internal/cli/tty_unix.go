//go:build darwin || linux

package cli

import "golang.org/x/sys/unix"

// enableCookedUTF8 puts the terminal into canonical mode with echo and the
// IUTF8 input flag for the duration of REPL prompt reading. With IUTF8 set the
// line discipline erases a whole multibyte (e.g. Japanese) character on one
// backspace instead of a single byte, and the terminal emulator — which knows
// the true glyph widths — handles the on-screen editing of full-width
// characters correctly.
//
// It returns a restore function (a no-op if the fd is not a terminal or the
// ioctl failed, so callers can always defer it safely).
func enableCookedUTF8(fd int) func() {
	prev, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return func() {}
	}
	t := *prev
	t.Lflag |= unix.ICANON | unix.ECHO | unix.ECHOE | unix.ECHOK | unix.ISIG
	t.Iflag |= unix.IUTF8 | unix.ICRNL
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, &t); err != nil {
		return func() {}
	}
	return func() { _ = unix.IoctlSetTermios(fd, ioctlWriteTermios, prev) }
}

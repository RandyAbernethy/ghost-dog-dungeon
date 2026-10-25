package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

func consoleColumns() int {
	var size struct{ Rows, Columns, X, Y uint16 }
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, os.Stdout.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&size)), 0, 0, 0)
	if errno != 0 || size.Columns == 0 {
		return 60
	}
	return int(size.Columns)
}

func enableRawMode(f *os.File) (func(), bool) {
	fd := int(f.Fd())
	state, err := getTermios(fd)
	if err != nil {
		return nil, false
	}
	raw := *state
	raw.Lflag &^= syscall.ICANON | syscall.ECHO | syscall.ISIG
	raw.Iflag &^= syscall.ICRNL | syscall.INLCR
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if err := setTermios(fd, &raw); err != nil {
		return nil, false
	}
	return func() { _ = setTermios(fd, state) }, true
}

func getTermios(fd int) (*syscall.Termios, error) {
	termios := &syscall.Termios{}
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return termios, nil
}

func setTermios(fd int, termios *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

// Arrow keys send multiple bytes beginning with Esc. A short wait distinguishes
// those sequences from a standalone Esc without blocking until another key.
func (in *rawInput) escapeByteReady() (bool, error) {
	if in.Buffered() > 0 {
		return true, nil
	}
	deadline := time.Now().Add(50 * time.Millisecond)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return false, nil
		}
		fd := int(in.file.Fd())
		var readFDs syscall.FdSet
		if fd < 0 || fd >= len(readFDs.Bits)*64 {
			return false, fmt.Errorf("terminal file descriptor %d cannot be selected", fd)
		}
		readFDs.Bits[fd/64] |= 1 << uint(fd%64)
		timeout := syscall.NsecToTimeval(remaining.Nanoseconds())
		n, err := syscall.Select(fd+1, &readFDs, nil, nil, &timeout)
		if err == syscall.EINTR {
			continue
		}
		return n > 0, err
	}
}

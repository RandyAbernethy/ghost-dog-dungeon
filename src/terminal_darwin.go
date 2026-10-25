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
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCGETA), uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return termios, nil
}

func setTermios(fd int, termios *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCSETA), uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

// Darwin's select reports readiness through a 32-bit descriptor bitset.
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
		if fd < 0 || fd >= len(readFDs.Bits)*32 {
			return false, fmt.Errorf("terminal file descriptor %d cannot be selected", fd)
		}
		mask := int32(1 << uint(fd%32))
		readFDs.Bits[fd/32] |= mask
		timeout := syscall.NsecToTimeval(remaining.Nanoseconds())
		err := syscall.Select(fd+1, &readFDs, nil, nil, &timeout)
		if err == syscall.EINTR {
			continue
		}
		return readFDs.Bits[fd/32]&mask != 0, err
	}
}

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"
)

func (g *game) loop(stdin io.Reader, stdout io.Writer) error {
	scanner := bufio.NewScanner(stdin)
	var rawIn *rawInput
	if f, ok := stdin.(*os.File); ok {
		if restore, ok := enableRawMode(f); ok {
			rawIn = &rawInput{Reader: bufio.NewReader(f), file: f}
			defer restore()
		}
	}
	g.renderChallengeRecords(stdout)
	g.render(stdout)
	for !g.won && !g.quit && g.player.HP > 0 {
		var cmd string
		if rawIn != nil {
			fmt.Fprintln(stdout, "  Keys act immediately: wasd/arrows, qezx, ., </>, p f b g u t o n r, v fountain, y interact, i inventory, E equipment, B book, j challenges, c codex, m inspect, k search, Esc map, S save, L load, Ctrl-C quit; type :equip NUMBER, :drop NUMBER, or :choose 1/2/3 + Enter")
			value, eof, err := readRawCommand(rawIn, stdout)
			if err != nil {
				return err
			}
			if eof {
				fmt.Fprintln(stdout, "\nThe dungeon waits in silence as you slip away.")
				return nil
			}
			cmd = normalizeCommand(strings.TrimSpace(strings.ToLower(value)))
		} else {
			fmt.Fprintln(stdout, "  Commands: wasd/arrows, q e z x, ., </>, p f b g u t o n r, v fountain, y interact, choose 1/2/3, i inventory, equipment, equip NUMBER, drop NUMBER, book, challenges, c codex, m inspect, k search, map, save, load, quit")
			fmt.Fprint(stdout, "\nCommand> ")
			if !scanner.Scan() {
				fmt.Fprintln(stdout, "\nThe dungeon waits in silence as you slip away.")
				return nil
			}
			cmd = normalizeCommand(strings.TrimSpace(strings.ToLower(scanner.Text())))
		}
		if cmd == "" {
			cmd = "."
		}
		switch cmd {
		case "map":
			g.render(stdout)
			continue
		case "i", "inventory":
			g.renderInventory(stdout)
			continue
		case "c", "codex":
			g.renderCodex(stdout)
			continue
		case "inspect", "look", "m":
			g.renderInspect(stdout)
			continue
		case "book", "read", "read book":
			g.renderBook(stdout)
			continue
		case "equipment", "equip":
			g.renderEquipment(stdout)
			continue
		case "j", "challenges", "records":
			g.renderChallenges(stdout)
			continue
		}
		g.processCommand(cmd)
		if !g.quit && !g.won && g.player.HP > 0 {
			g.render(stdout)
		}
	}

	switch {
	case g.won:
		g.render(stdout)
		if g.dog.Freed && g.dog.Alive {
			fmt.Fprintln(stdout, "\nYou escaped the dungeon with your ghost dog. Victory!")
		} else {
			fmt.Fprintln(stdout, "\nYou escaped the dungeon. Victory!")
		}
	case g.player.HP <= 0:
		g.render(stdout)
		fmt.Fprintln(stdout, "\nYou fall in the dark. The Dread Lich keeps the dungeon.")
	case g.quit:
		fmt.Fprintln(stdout, "You retreat before the dungeon can claim you.")
	}
	if g.won || g.player.HP <= 0 {
		if err := g.recordChallenges(); err != nil {
			fmt.Fprintf(stdout, "Could not save challenge records: %v\n", err)
		}
		g.renderChallenges(stdout)
	}
	return nil
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

type rawInput struct {
	*bufio.Reader
	file *os.File
}

func readRawCommand(in *rawInput, out io.Writer) (string, bool, error) {
	buf := ""
	display := ""
	textMode := false
	redrawRawPrompt(out, display)
	for {
		var one [1]byte
		n, err := in.Read(one[:])
		if err == io.EOF {
			return "", true, nil
		}
		if err != nil {
			return "", false, err
		}
		if n == 0 {
			return "", true, nil
		}
		switch one[0] {
		case '\r', '\n':
			fmt.Fprintln(out)
			return buf, false, nil
		case 3, 4:
			fmt.Fprintln(out)
			return "quit", false, nil
		case 127, 8:
			if len(buf) > 0 {
				_, size := utf8.DecodeLastRuneInString(buf)
				buf = buf[:len(buf)-size]
				display = buf
				redrawRawPrompt(out, display)
			}
		case 27:
			cmd, _, err := readEscapeCommand(in)
			if err != nil {
				return "", false, err
			}
			if cmd != "" {
				fmt.Fprintln(out)
				return cmd, false, nil
			}
		case ':':
			if !textMode && len(buf) == 0 {
				textMode = true
				display = ":"
				redrawRawPrompt(out, display)
				continue
			}
		default:
			if !textMode {
				if cmd, shown, ok := rawKeyCommand(one[0]); ok {
					fmt.Fprintf(out, "\r\x1b[2KCommand> %s\n", shown)
					return cmd, false, nil
				}
			}
			buf += string(one[0])
			display = buf
			redrawRawPrompt(out, display)
		}
	}
}

func rawKeyCommand(key byte) (string, string, bool) {
	switch key {
	case 'w', 'a', 's', 'd', 'q', 'e', 'z', 'x', '.', 'p', 'f', 'b', 'g', 'u', 't', 'o', 'n', 'r', '<', '>', 'v', 'y':
		return string(key), string(key), true
	}
	switch key {
	case 'i':
		return "inventory", "i", true
	case 'c':
		return "codex", "c", true
	case 'm':
		return "inspect", "m", true
	case 'k':
		return "search", "k", true
	case 'B':
		return "book", "B", true
	case 'E':
		return "equipment", "E", true
	case 'j':
		return "challenges", "j", true
	case 'S':
		return "save", "S", true
	case 'L':
		return "load", "L", true
	case '?':
		return "h", "?", true
	default:
		return "", "", false
	}
}

func redrawRawPrompt(out io.Writer, display string) {
	fmt.Fprintf(out, "\r\x1b[2KCommand> %s", display)
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

func readEscapeCommand(in *rawInput) (string, string, error) {
	seq := []byte{27}
	for len(seq) < 6 {
		ready, err := in.escapeByteReady()
		if err != nil {
			return "", "", err
		}
		if !ready {
			break
		}
		next, err := in.Peek(1)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", err
		}
		// Preserve a following ordinary key for the next command.
		if len(seq) == 1 && next[0] != '[' && next[0] != 'O' {
			break
		}
		key, err := in.ReadByte()
		if err != nil {
			return "", "", err
		}
		seq = append(seq, key)
		if len(seq) > 2 && ((key >= 'A' && key <= 'Z') || (key >= 'a' && key <= 'z') || key == '~') {
			break
		}
	}
	cmd := normalizeCommand(strings.ToLower(string(seq)))
	switch cmd {
	case "map":
		return cmd, "Esc", nil
	case "w":
		return cmd, "↑", nil
	case "s":
		return cmd, "↓", nil
	case "a":
		return cmd, "←", nil
	case "d":
		return cmd, "→", nil
	case "q":
		return cmd, "↖", nil
	case "e":
		return cmd, "↗", nil
	case "z":
		return cmd, "↙", nil
	case "x":
		return cmd, "↘", nil
	default:
		return "", "", nil
	}
}

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

func (g *game) loop(stdin io.Reader, stdout io.Writer) error {
	scanner := bufio.NewScanner(stdin)
	var rawIn *rawInput
	interactive := false
	var pack *inventoryMenu
	if f, ok := stdin.(*os.File); ok {
		if info, err := f.Stat(); err == nil {
			interactive = info.Mode()&os.ModeCharDevice != 0
		}
		if restore, ok := enableRawMode(f); ok {
			rawIn = &rawInput{Reader: bufio.NewReader(f), file: f}
			defer restore()
		}
	}
	fmt.Fprintln(stdout, "Ghost Dog Dungeon version "+gameVersion)
	fmt.Fprintf(stdout, "Dungeon seed: %d\n", g.seed)
	defer func() {
		fmt.Fprintf(stdout, "\nDungeon seed: %d\n", g.seed)
	}()
	renderOpeningArt(stdout)
	g.renderChallengeRecords(stdout)
	if interactive {
		fmt.Fprint(stdout, "\nPress enter to begin the descent! > ")
		begin, err := waitForDescent(scanner, rawIn)
		if err != nil {
			return err
		}
		if !begin {
			return nil
		}
		fmt.Fprint(stdout, "\x1b[2J\x1b[H")
	}
	g.render(stdout)
	for !g.won && !g.quit && g.player.HP > 0 {
		var cmd string
		if rawIn != nil {
			if pack == nil {
				uiText(stdout, "Move: arrows/wasd (qezx diagonal) | i: Pack | y: Interact")
				uiText(stdout, "m: Inspect | ?: Help | Esc: Map | Ctrl-C: Quit")
			}
			readCommand := readRawCommand
			if pack != nil {
				readCommand = readRawMenuCommand
			}
			value, eof, err := readCommand(rawIn, stdout)
			if err != nil {
				return err
			}
			if eof {
				fmt.Fprintln(stdout, "\nThe dungeon waits in silence as you slip away.")
				return nil
			}
			cmd = normalizeCommand(strings.TrimSpace(strings.ToLower(value)))
		} else {
			if pack == nil {
				uiText(stdout, "Move: wasd/qezx | i: Pack | y: Interact | m: Inspect")
				uiText(stdout, "h: Help | map: Return | quit: Quit (command + Enter)")
			}
			fmt.Fprint(stdout, "\nCommand> ")
			if !scanner.Scan() {
				fmt.Fprintln(stdout, "\nThe dungeon waits in silence as you slip away.")
				return nil
			}
			cmd = normalizeCommand(strings.TrimSpace(strings.ToLower(scanner.Text())))
		}
		if pack != nil {
			handled, close := g.packCommand(pack, cmd)
			if close {
				pack = nil
			}
			if handled {
				if pack != nil {
					g.renderPack(stdout, pack, rawIn != nil)
				} else if g.player.HP > 0 {
					g.render(stdout)
				}
				continue
			}
		}
		if len(cmd) == 1 && cmd[0] >= '1' && cmd[0] <= '3' {
			cmd = "choose " + cmd
		}
		if cmd == "" {
			cmd = "."
		}
		switch cmd {
		case "h", "help", "?":
			g.renderHelp(stdout, rawIn != nil)
			continue
		case "map":
			g.render(stdout)
			continue
		case "i", "inventory":
			pack = &inventoryMenu{}
			g.renderPack(stdout, pack, rawIn != nil)
			continue
		case "inventory details", "details":
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
			pack = &inventoryMenu{}
			g.renderPack(stdout, pack, rawIn != nil)
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
		if g.dog.active() {
			fmt.Fprintln(stdout, "\nYou escaped the dungeon with your ghost dog. Victory!")
		} else {
			fmt.Fprintln(stdout, "\nYou escaped the dungeon. Victory!")
		}
		g.renderEnding(stdout)
	case g.player.HP <= 0:
		g.render(stdout)
		fmt.Fprintln(stdout, "\nYou fall in the dark. The Dread Lich keeps the dungeon.")
	case g.quit:
		fmt.Fprintln(stdout, "You retreat before the dungeon can claim you.")
	}
	if g.won || g.player.HP <= 0 {
		g.renderChronicle(stdout)
		achievements, err := g.recordChallenges()
		if err != nil {
			fmt.Fprintf(stdout, "Could not save challenge records: %v\n", err)
		}
		g.renderChallenges(stdout)
		renderRecordAchievements(stdout, achievements)
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
	case '1', '2', '3':
		return "choose " + string(key), string(key), true
	default:
		return "", "", false
	}
}

func redrawRawPrompt(out io.Writer, display string) {
	fmt.Fprintf(out, "\r\x1b[2KCommand> %s", display)
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

// The welcome prompt shares the gameplay reader so buffered commands are retained.
func waitForDescent(scanner *bufio.Scanner, rawIn *rawInput) (bool, error) {
	if rawIn != nil {
		for {
			key, err := rawIn.ReadByte()
			if err == io.EOF {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			switch key {
			case '\r', '\n':
				return true, nil
			case 3, 4:
				return false, nil
			}
		}
	}
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			return true, nil
		}
	}
	return false, scanner.Err()
}

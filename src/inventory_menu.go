package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Menus are transient input state. Only their confirmed actions reach the game.
type inventoryMenu struct {
	selected int
	detail   bool
}

type packEntry struct {
	number      int // Stable number shared with equip/drop commands.
	label       string
	description string
	comparison  string
	command     string
	equipped    bool
}

func (g *game) packEntries() []packEntry {
	var entries []packEntry
	for i, w := range g.player.Weapons {
		bonuses := weaponBonusText(w)
		if bonuses != "" {
			bonuses = ", " + bonuses
		}
		entries = append(entries, packEntry{i + 1, fmt.Sprintf("%s (%d-%d, reach %d%s)", coloredWeaponName(w), w.Min, w.Max, w.effectiveReach(), bonuses), weaponDescription(w), compareWeapon(w, g.player.Weapon), "equip", w == g.player.Weapon})
	}
	for i, a := range g.player.Armors {
		entries = append(entries, packEntry{len(g.player.Weapons) + i + 1, fmt.Sprintf("%s (%s): %d guard, %d spell ward, %+d strike damage", coloredArmorName(a), slotLabel(a.Slot), a.Defense, a.SpellWard, a.StrikeBonus), armorDescription(a), compareArmor(a, g.player.armorForSlot(a.Slot)), "equip", a == g.player.armorForSlot(a.Slot)})
	}
	commands := []string{"p", "f", "g", "b", "u", "t", "o", "n", "r"}
	for i, line := range g.inventoryLines() {
		if *line.quantity > 0 {
			entries = append(entries, packEntry{len(g.player.Weapons) + len(g.player.Armors) + i + 1, fmt.Sprintf("%s x%d (map %c) [use %s]", line.name, *line.quantity, itemGlyphs[line.kind], commands[i]), line.description, "", commands[i], false})
		}
	}
	return entries
}

func (g *game) renderPack(out io.Writer, menu *inventoryMenu, raw bool) {
	entries := g.packEntries()
	fmt.Fprintln(out, "\n=== Inventory & Equipment ===")
	uiText(out, fmt.Sprintf("Head: %s | Body: %s | Leg: %s", g.player.HeadArmor.Name, g.player.BodyArmor.Name, g.player.LegArmor.Name))
	uiText(out, fmt.Sprintf("Guard %d | Block %d | Spell ward %d | Strike bonus %+d", g.player.armorDefense(), g.player.armorBlock(), g.player.spellWard(), g.player.strikeBonus()))
	if len(entries) == 0 {
		fmt.Fprintln(out, "Your pack is empty. Esc or map returns to the map.")
		return
	}
	menu.selected = max(0, min(menu.selected, len(entries)-1))
	for i, entry := range entries {
		mark, active := " ", ""
		if i == menu.selected {
			mark = ">"
		}
		if entry.equipped {
			active = " [equipped]"
		}
		uiText(out, fmt.Sprintf("%s %d. %s%s", mark, entry.number, entry.label, active))
	}
	if menu.detail {
		entry := entries[menu.selected]
		uiText(out, entry.description)
		if entry.comparison != "" {
			uiText(out, entry.comparison)
		}
		if entry.command == "equip" {
			uiText(out, "Enter/e: Equip | d: Drop | i: List | Esc/map: Close")
		} else {
			uiText(out, "Enter/u: Use | d: Drop | i: List | Esc/map: Close")
		}
		uiText(out, "Confirming an action spends a turn only if it succeeds.")
	} else if raw {
		uiText(out, "Up/down: Select | e: Equip | d: Drop | Enter: Details")
		uiText(out, "Number + Enter: Details | Esc: Map | ?: Help")
	} else {
		uiText(out, "Enter an item number for details; e equips, d drops; map closes the pack.")
	}
	if raw {
		uiText(out, "B: Book of Ghost Dog | j: Personal records")
	} else {
		uiText(out, "Book of Ghost Dog: book | Personal records: j")
	}
}

// handled=false lets ordinary commands leave the menu without a second prompt.
func (g *game) packCommand(menu *inventoryMenu, cmd string) (handled, close bool) {
	entries := g.packEntries()
	if cmd == "map" || cmd == "cancel" {
		return true, true
	}
	if len(entries) == 0 {
		return false, true
	}
	menu.selected = max(0, min(menu.selected, len(entries)-1))
	if n, err := strconv.Atoi(cmd); err == nil {
		for i, entry := range entries {
			if entry.number == n {
				menu.selected, menu.detail = i, true
				return true, false
			}
		}
		return true, false
	}
	if cmd == "back" || cmd == "i" || cmd == "inventory" {
		menu.detail = false
		return true, false
	}
	if cmd == "w" || cmd == "up" || cmd == "s" || cmd == "down" {
		step := 1
		if cmd == "w" || cmd == "up" {
			step = -1
		}
		menu.selected = (menu.selected + step + len(entries)) % len(entries)
		menu.detail = false
		return true, false
	}
	if cmd == "" && !menu.detail {
		menu.detail = true
		return true, false
	}
	if menu.detail || cmd == "e" || cmd == "equip" || cmd == "d" || cmd == "drop" {
		entry := entries[menu.selected]
		var action string
		switch {
		case cmd == "d" || cmd == "drop":
			action = fmt.Sprintf("drop %d", entry.number)
		case cmd == "" || entry.command == "equip" && (cmd == "e" || cmd == "equip") || entry.command != "equip" && (cmd == "u" || cmd == "use"):
			action = entry.command
			if action == "equip" {
				action = fmt.Sprintf("equip %d", entry.number)
			}
		}
		if action != "" {
			g.processCommand(action)
			return true, true
		}
		if cmd == "e" || cmd == "equip" || cmd == "u" || cmd == "use" {
			return true, false
		}
	}
	return false, true
}

// Menu input uses the same escape decoder, but letters and Enter belong to the
// menu rather than map shortcuts. Numbers may contain several digits.
func readRawMenuCommand(in *rawInput, out io.Writer) (string, bool, error) {
	var digits strings.Builder
	redrawRawPrompt(out, "")
	for {
		key, err := in.ReadByte()
		if err == io.EOF {
			return "", true, nil
		}
		if err != nil {
			return "", false, err
		}
		switch {
		case key == 3 || key == 4:
			return "quit", false, nil
		case key == 27:
			cmd, _, err := readEscapeCommand(in)
			if cmd != "" || err != nil {
				return cmd, false, err
			}
		case key == '\r' || key == '\n':
			return digits.String(), false, nil
		case key == 127 || key == 8:
			value := digits.String()
			if len(value) > 0 {
				digits.Reset()
				digits.WriteString(value[:len(value)-1])
			}
			redrawRawPrompt(out, digits.String())
		case key >= '0' && key <= '9':
			if digits.Len() < 9 {
				digits.WriteByte(key)
			}
			redrawRawPrompt(out, digits.String())
		default:
			if cmd, _, ok := rawKeyCommand(key); ok {
				return cmd, false, nil
			}
			return strings.ToLower(string(key)), false, nil
		}
	}
}

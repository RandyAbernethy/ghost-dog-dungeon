package main

import (
	"fmt"
	"io"
)

func (g *game) renderHelp(w io.Writer, raw bool) {
	fmt.Fprintln(w, "\n=== Help ===")
	uiText(w, "Move/attack: wasd or arrows; qezx diagonals. Wait: . | Stairs: < or >")
	uiText(w, "Inventory: i. Select an item, read its comparison, then confirm Use, Equip or Drop. Reading and cancelling are free.")
	if raw {
		uiText(w, "Type :details and Enter for the complete pack report, including companion rules and unowned consumables.")
	} else {
		uiText(w, "The details command shows the complete pack report, including unowned consumables and companion rules.")
	}
	if raw {
		uiText(w, "Keys act immediately. Pack: up/down and Enter; numbers + Enter also select. Esc returns to the map.")
		uiText(w, "Book: B | Equipment: E | Save: S | Load: L | Quit: Ctrl-C")
		uiText(w, "Advanced commands: type :equip NUMBER or :drop NUMBER, then Enter.")
	} else {
		uiText(w, "Type a command and press Enter. Pack: item number, then Enter; e equips, u uses, d drops. map cancels.")
		uiText(w, "Book: book | Equipment: equipment | Save: save | Load: load | Quit: quit")
		uiText(w, "Advanced commands: equip NUMBER or drop NUMBER, then Enter.")
	}
	uiText(w, "Interact: y | Fountain: v | Search: k | Inspect: m | Codex: c | Records: j")
	uiText(w, "Ghost gifts: y to read, then 1/2/3 to choose. Press Enter after the number in command-and-Enter mode.")
	uiText(w, "Consumable shortcuts: p potion; f fire; g blink; b ward; u sun; t frost; o starfire; n phoenix ash; r recall.")
	uiText(w, "Follow drafts and disturbed mortar to secret doors. Approach the indicated wall and search (k).")
	uiText(w, "Map symbols: # wall | . floor | @ you | D Ghost Dog | d chained dog | > down | < up/out | F/f fountain/spent")
	uiText(w, "Features: G ghost of a friendly villager | + lever | & keepsake | ^ shrine | S lost spirit | O ritual sigil | X Epic strike")
	uiText(w, "Crypt: { entrance | } exit | | sealed entrance")
	uiText(w, "Floor items show only their physical form until pickup: ? scroll, = orb, % charm, ! potion, * stone, ) weapon, [ armor. Phoenix Ash shares the potion appearance.")
	seenItems := map[rune]bool{}
	for kind := itemWeapon; kind <= itemSideStoryScroll; kind++ {
		if seenItems[itemGlyphs[kind]] {
			continue
		}
		seenItems[itemGlyphs[kind]] = true
		fmt.Fprintf(w, "%c %s\n", itemGlyphs[kind], itemMapLabel(item{Kind: kind}))
	}
	uiText(w, "Discovered monsters: c opens the codex. Discovered crypt traps: m identifies their symbols and effects.")
}

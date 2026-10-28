package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func (g *game) render(w io.Writer) {
	g.noteVisibleMonsters()
	lvl := g.current()
	if g.inCrypt {
		fmt.Fprintln(w)
		uiText(w, "=== Ghost Dog Dungeon: "+cryptName+" ===")
	} else {
		fmt.Fprintf(w, "\n=== Ghost Dog Dungeon: Level %d/%d ===\n", g.currentLevel+1, levelCount)
	}
	if lvl.Layout != "" {
		uiText(w, fmt.Sprintf("%s — %s", lvl.Theme, lvl.Layout))
	}
	uiText(w, fmt.Sprintf("HP %d/%d | Armor %d (block %d) | Turns %d", g.player.HP, g.player.MaxHP, g.player.armorDefense(), g.player.armorBlock(), g.stats.Turns))
	weaponLine := fmt.Sprintf("Weapon %s (%d-%d) | Reach %d", g.player.Weapon.Name, g.player.Weapon.Min, g.player.Weapon.Max, g.player.Weapon.effectiveReach())
	if bonuses := weaponBonusText(g.player.Weapon); bonuses != "" {
		weaponLine += " | " + bonuses
	}
	uiText(w, weaponLine)
	var bonuses []string
	if ward := g.player.spellWard(); ward != 0 {
		bonuses = append(bonuses, fmt.Sprintf("Spell ward %d", ward))
	}
	if strike := g.player.strikeBonus(); strike != 0 {
		bonuses = append(bonuses, fmt.Sprintf("Strike bonus %+d", strike))
	}
	if len(bonuses) > 0 {
		uiText(w, strings.Join(bonuses, " | "))
	}
	if g.dog.Freed {
		status := "gone"
		if g.dog.Alive {
			status = fmt.Sprintf("%d/%d HP", g.dog.HP, g.dog.MaxHP)
			if g.dog.Chained {
				status += fmt.Sprintf("; chained on floor %d — move onto d to free him", g.dog.ChainFloor+1)
			}
		}
		uiText(w, fmt.Sprintf("Ghost dog: adopted (%s)", status))
	} else {
		fmt.Fprintln(w, "Ghost dog: not yet freed")
	}
	if g.player.ShieldTurns > 0 || g.player.HexedTurns > 0 || g.player.BoundTurns > 0 || g.player.PoisonTurns > 0 {
		effects := fmt.Sprintf("Effects: ward %d  hex %d", g.player.ShieldTurns, g.player.HexedTurns)
		if g.player.BoundTurns > 0 {
			effects += fmt.Sprintf("  %s %d", g.player.Binding.label(), g.player.BoundTurns)
		}
		if g.player.PoisonTurns > 0 {
			effects += fmt.Sprintf("  poison %d", g.player.PoisonTurns)
		}
		uiText(w, effects)
	}
	for y := 0; y < mapHeight; y++ {
		var line strings.Builder
		for x := 0; x < mapWidth; x++ {
			p := pos{X: x, Y: y}
			if !g.canSee(p) {
				line.WriteRune(' ')
				continue
			}
			ch := lvl.Tiles[y][x]
			m := g.monsterAt(lvl, p)
			f := g.fountainAt(lvl, p)
			it := g.itemAt(lvl, p)
			event := lvl.eventAt(p)
			switch {
			case g.player.Pos == p:
				ch = glyphPlayer
			case m != nil:
				ch = m.glyph()
			case g.dogPresent() && g.dog.Pos == p:
				ch = glyphDog
				if g.dog.Chained {
					ch = glyphChainedDog
				}
			case g.cryptDoorGlyph(p) != 0:
				ch = g.cryptDoorGlyph(p)
			case g.trapAt(p) != nil && g.trapAt(p).Discovered && !g.trapAt(p).Spent:
				ch = g.trapAt(p).glyph()
			case ch != '#' && g.epicDangerAt(p):
				line.WriteString("\x1b[1;31m" + string(glyphEpicWarning) + ansiReset)
				continue
			case lvl.DogChain != nil && *lvl.DogChain == p && !g.dog.Freed:
				ch = glyphChainedDog
			case lvl.HasStair && lvl.Stairs == p:
				ch = glyphStairsDown
			case lvl.HasUpStair && lvl.UpStairs == p:
				ch = glyphStairsUp
			case lvl.HasEscapeStair && lvl.EscapeStairs == p && (lvl.primarySecret() == nil || lvl.primarySecret().Revealed):
				ch = glyphStairsUp
			case ch != '#' && f != nil && !f.Used:
				ch = glyphFountain
			case ch != '#' && f != nil:
				ch = glyphSpentFountain
			case ch != '#' && it != nil:
				ch = it.glyph()
			case ch != '#' && event != nil && !event.Used:
				ch = event.glyph()
			case ch != '#' && lvl.Objective.marker(p) != 0:
				ch = lvl.Objective.marker(p)
			default:
				// Keep the terrain glyph.
			}
			if g.epicDangerAt(p) {
				line.WriteString("\x1b[1;31m")
				line.WriteString(string(ch))
				line.WriteString(ansiReset)
			} else {
				line.WriteRune(ch)
			}
		}
		fmt.Fprintln(w, line.String())
	}
	g.renderLegend(w)
	g.renderEpicWarnings(w)
	g.renderObjective(w)
	g.renderCryptFeatures(w)
	if note := g.contextHint(); note != "" {
		uiText(w, "Here: "+note)
	}
	if g.stats.Turns == 0 {
		fmt.Fprintln(w, "Dungeon events:")
	} else {
		fmt.Fprintf(w, "Turn %d events (%d monsters killed):\n", g.stats.Turns, g.stats.MonstersKilled)
	}
	for _, msg := range g.messages {
		// Snapshot paths must remain intact for copying, including spaces.
		if strings.HasPrefix(msg, "Game saved to ") || strings.HasPrefix(msg, "Game loaded from ") {
			fmt.Fprintln(w, "- "+msg)
			continue
		}
		if strings.HasPrefix(msg, "You pack ") {
			if pickup, comparison, ok := strings.Cut(msg, ". Compared to "); ok {
				wrappedText(w, pickup+".", 80, "- ", "  ")
				wrappedText(w, "Compared to "+comparison, 80, "  ", "  ")
				continue
			}
		}
		wrappedText(w, msg, 80, "- ", "  ")
	}
}

func (g *game) renderLegend(w io.Writer) {
	uiText(w, fmt.Sprintf("Legend: %c you | %c wall | %c floor", glyphPlayer, glyphWall, glyphFloor))
	// List only entities that can actually be seen, in stable map order.
	seen := map[string]bool{}
	var entries []string
	add := func(symbol rune, name string) {
		entry := fmt.Sprintf("%c %s", symbol, name)
		if !seen[entry] {
			seen[entry] = true
			entries = append(entries, entry)
		}
	}
	lvl := g.current()
	for y := 0; y < mapHeight; y++ {
		for x := 0; x < mapWidth; x++ {
			p := pos{x, y}
			if !g.canSee(p) || lvl.Tiles[y][x] == '#' || distance(g.player.Pos, p) > 4 {
				continue
			}
			if lvl.DogChain != nil && *lvl.DogChain == p && !g.dog.Freed {
				add(glyphChainedDog, "chained Ghost Dog")
			}
			if m := g.monsterAt(lvl, p); m != nil {
				add(m.glyph(), m.Name)
			}
			if it := g.itemAt(lvl, p); it != nil {
				add(it.glyph(), itemMapLabel(*it))
			}
			if f := g.fountainAt(lvl, p); f != nil {
				if f.Used {
					add(glyphSpentFountain, "spent fountain")
				} else {
					add(glyphFountain, "fountain")
				}
			}
			if e := lvl.eventAt(p); e != nil && !e.Used {
				add(e.glyph(), e.displayName())
			}
			if symbol := g.cryptDoorGlyph(p); symbol != 0 {
				add(symbol, "crypt door")
			}
		}
	}
	if g.dogPresent() && g.canSee(g.dog.Pos) && distance(g.player.Pos, g.dog.Pos) <= 4 {
		if g.dog.Chained {
			add(glyphChainedDog, "chained Ghost Dog")
		} else {
			add(glyphDog, "Ghost Dog")
		}
	}
	if lvl.HasStair && g.canSee(lvl.Stairs) && distance(g.player.Pos, lvl.Stairs) <= 4 {
		add(glyphStairsDown, "down")
	}
	if lvl.HasUpStair && g.canSee(lvl.UpStairs) && distance(g.player.Pos, lvl.UpStairs) <= 4 || lvl.HasEscapeStair && lvl.primarySecret() != nil && lvl.primarySecret().Revealed && distance(g.player.Pos, lvl.EscapeStairs) <= 4 {
		add(glyphStairsUp, "up/out")
	}
	if len(entries) > 0 {
		uiText(w, "Nearby: "+strings.Join(entries, " | "))
	}
}

func itemMapLabel(it item) string {
	switch it.Kind {
	case itemWeapon:
		return "weapon"
	case itemArmor:
		return "armor"
	case itemFireScroll, itemGhostRecallScroll, itemStoryScroll, itemSideStoryScroll:
		return "scroll"
	case itemSunOrb, itemStarfireOrb:
		return "orb"
	case itemPotion, itemPhoenixAsh:
		return "potion"
	case itemWardingCharm, itemFrostCharm:
		return "charm"
	case itemBlinkStone:
		return "stone"
	default:
		return "item"
	}
}

func (g *game) renderInventory(w io.Writer) {
	fmt.Fprintln(w, "\n=== Inventory & Equipment ===")
	fmt.Fprintf(w, "Weapon: %s (%d-%d)\n  %s\n", coloredWeaponName(g.player.Weapon), g.player.Weapon.Min, g.player.Weapon.Max, weaponDescription(g.player.Weapon))
	fmt.Fprintf(w, "Head: %s (+%d guard, %s)\n  %s\n", coloredArmorName(g.player.HeadArmor), g.player.HeadArmor.Defense, rarityLabel(g.player.HeadArmor.Rarity), armorDescription(g.player.HeadArmor))
	fmt.Fprintf(w, "Body: %s (+%d guard, %s)\n  %s\n", coloredArmorName(g.player.BodyArmor), g.player.BodyArmor.Defense, rarityLabel(g.player.BodyArmor.Rarity), armorDescription(g.player.BodyArmor))
	fmt.Fprintf(w, "Leg: %s (+%d guard, %s)\n  %s\n", coloredArmorName(g.player.LegArmor), g.player.LegArmor.Defense, rarityLabel(g.player.LegArmor.Rarity), armorDescription(g.player.LegArmor))
	fmt.Fprintf(w, "Total armor: %d guard, blocking %d damage from each hit before wards.\n", g.player.armorDefense(), g.player.armorBlock())
	if g.player.hasGodArmor() {
		fmt.Fprintln(w, "God armor can absorb up to 90% of a physical blow; even divine protection can be worn down.")
	}
	fmt.Fprintf(w, "Spell ward: blocks %d extra spell damage. Strike bonus: %+d weapon damage.\n", g.player.spellWard(), g.player.strikeBonus())
	g.renderEquipment(w)
	fmt.Fprintf(w, "\nBook of Ghost Dog: %d/10 chapters, %d side stories. Press B or type book to read at any time.\n", len(g.player.StoryChapters), len(g.player.SideStories))
	fmt.Fprintf(w, "Optional challenges: %d turns, %d kills, %d Ghost Dog falls. Press j for records.\n", g.stats.Turns, g.stats.MonstersKilled, g.stats.GhostDogFalls)
	fmt.Fprintln(w, "Consumables:")
	for i, line := range g.inventoryLines() {
		fmt.Fprintf(w, "%d. %s x%d (map %c)\n  %s\n", len(g.player.Weapons)+len(g.player.Armors)+i+1, line.name, *line.quantity, itemGlyphs[line.kind], line.description)
	}
	companion := "The ghost dog is still chained somewhere below."
	if g.dog.active() {
		companion = fmt.Sprintf("Ghost dog companion: fighting at %d/%d HP and will attack nearby monsters.", g.dog.HP, g.dog.MaxHP)
	} else if g.dog.Chained {
		companion = fmt.Sprintf("Ghost dog companion: bound by a silver chain on floor %d at %d/%d HP. Move onto his d tile to free him.", g.dog.ChainFloor+1, g.dog.HP, g.dog.MaxHP)
	} else if g.dog.Freed {
		companion = "Ghost dog companion: dispersed into silver mist; defeating the Dread Lich will restore it at full health."
	}
	fmt.Fprintln(w, "Companion:")
	fmt.Fprintln(w, "- "+companion)
	fmt.Fprintln(w, "Tip: press c for the codex or m to inspect nearby threats and loot.")
}

func (g *game) renderCodex(w io.Writer) {
	g.noteVisibleMonsters()
	fmt.Fprintln(w, "\n=== Enemy Codex ===")
	known := g.knownMonsterKinds()
	if len(known) == 0 {
		fmt.Fprintln(w, "You have not survived long enough to study anything yet.")
		return
	}
	for _, kind := range known {
		m := newMonster(kind)
		fmt.Fprintf(w, "- %c %s  HP %d  Damage %d-%d  Base difficulty %d\n", m.glyph(), m.Name, m.MaxHP, m.MinDamage, m.MaxDamage, monsterDifficulty(m))
		fmt.Fprintln(w, "  "+monsterSummary(kind))
		fmt.Fprintln(w, "  Trick: "+monsterPower(kind))
	}
	fmt.Fprintln(w, "Tip: press i to review your equipment and consumables.")
}

func (g *game) renderInspect(w io.Writer) {
	g.noteVisibleMonsters()
	fmt.Fprintln(w, "\n=== Inspect ===")
	if g.inCrypt {
		fmt.Fprintf(w, "You stand at %s in %s.\n", pointLabel(g.player.Pos), cryptName)
	} else {
		fmt.Fprintf(w, "You stand at %s on level %d.\n", pointLabel(g.player.Pos), g.currentLevel+1)
	}
	fmt.Fprintln(w, inspectTileDescription(g.current(), g.player.Pos))
	g.renderEpicWarnings(w)
	g.renderObjective(w)
	g.renderCryptFeatures(w)
	if g.current().HasStair {
		fmt.Fprintf(w, "Stairs down: %s.\n", relativeTo(g.player.Pos, g.current().Stairs))
	}
	if g.current().HasUpStair {
		fmt.Fprintf(w, "Stairs up: %s.\n", relativeTo(g.player.Pos, g.current().UpStairs))
	}
	if g.current().HasEscapeStair && (g.current().primarySecret() == nil || g.current().primarySecret().Revealed) {
		fmt.Fprintf(w, "Hidden way out: %s.\n", relativeTo(g.player.Pos, g.current().EscapeStairs))
	}
	for _, f := range g.current().Fountains {
		if g.current().Tiles[f.Pos.Y][f.Pos.X] != '#' && distance(g.player.Pos, f.Pos) <= 6 {
			status := "unused"
			if f.Used {
				status = "spent"
			}
			fmt.Fprintf(w, "Fountain (%s): %s.\n", status, relativeTo(g.player.Pos, f.Pos))
		}
	}
	fmt.Fprintln(w, "Nearby threats:")
	threats := g.visibleThreats(6)
	if len(threats) == 0 {
		fmt.Fprintln(w, "- None close enough to matter right now.")
	} else {
		for _, m := range threats {
			fmt.Fprintf(w, "- %s: %s (%c) HP %d/%d damage %d-%d difficulty %d. %s\n", relativeTo(g.player.Pos, m.Pos), m.Name, m.glyph(), m.HP, m.MaxHP, m.MinDamage, m.MaxDamage, monsterDifficulty(*m), monsterPower(m.Kind))
		}
	}
	fmt.Fprintln(w, "Nearby loot:")
	loot := g.visibleItems(6)
	if len(loot) == 0 {
		fmt.Fprintln(w, "- No notable loot nearby.")
	} else {
		for _, it := range loot {
			fmt.Fprintf(w, "- %s: %s (map %c)\n", relativeTo(g.player.Pos, it.Pos), floorItemText(it), it.glyph())
		}
	}
	if g.dogPresent() && g.canSee(g.dog.Pos) {
		fmt.Fprintf(w, "Ghost dog: %s.\n", relativeTo(g.player.Pos, g.dog.Pos))
		if g.dog.Chained {
			fmt.Fprintln(w, "Ghost Dog is chained (d) and cannot act. Move onto his tile to break the silver chain.")
		}
	}
	fmt.Fprintln(w, "Tip: use movement or . to wait; i opens your inventory.")
}

func inspectTileDescription(lvl *level, p pos) string {
	switch {
	case lvl.HasEscapeStair && lvl.EscapeStairs == p && (lvl.primarySecret() == nil || lvl.primarySecret().Revealed):
		return "You are standing on the hidden stairs up and out of the dungeon."
	case lvl.HasUpStair && lvl.UpStairs == p:
		return "You are standing on stairs leading back up."
	case lvl.HasStair && lvl.Stairs == p:
		return "You are standing on the stairs down."
	case lvl.DogChain != nil && *lvl.DogChain == p:
		return "A silver chain lies here where the ghost dog was bound."
	}
	for _, f := range lvl.Fountains {
		if f.Pos == p {
			if f.Used {
				return "An empty stone fountain; its healing water is spent."
			}
			return "A clear fountain. Stand here or beside it and press v to drink."
		}
	}
	return "The floor is cold stone, scarred by old battles and dragging claws."
}

func floorItemText(it item) string {
	return "unidentified " + itemMapLabel(it) + ". Pick it up to identify it."
}

// Full details are appropriate for owned items and explicit friendly-ghost offers.
func inspectItemText(it item) string {
	switch it.Kind {
	case itemWeapon:
		return fmt.Sprintf("weapon %s (%d-%d, reach %d). %s", coloredWeaponName(it.Weapon), it.Weapon.Min, it.Weapon.Max, it.Weapon.effectiveReach(), weaponDescription(it.Weapon))
	case itemArmor:
		return fmt.Sprintf("%s armor for the %s slot (+%d guard, %s). %s", coloredArmorName(it.Armor), slotLabel(it.Armor.Slot), it.Armor.Defense, rarityLabel(it.Armor.Rarity), armorDescription(it.Armor))
	case itemPotion:
		return "healing potion. Restores 12-18 health."
	case itemFireScroll:
		return "fire scroll. Blasts the nearest enemy within 6 tiles."
	case itemBlinkStone:
		return "blink stone. Teleports you and an unchained Ghost Dog far from monsters, outside secret rooms."
	case itemWardingCharm:
		return "warding charm. Reduces damage to you and your Ghost Dog for 3 enemy turns."
	case itemSunOrb:
		return "sun orb. Burns every undead enemy on the floor."
	case itemFrostCharm:
		return "frost charm. Freezes and wounds the nearest enemy within 6 tiles."
	case itemStarfireOrb:
		return "starfire orb. Deals 18-24 damage to the nearest enemy within 6 tiles."
	case itemPhoenixAsh:
		return "phoenix ash. Restores 22-30 health when used below full health."
	case itemGhostRecallScroll:
		return "Ghost Dog recall scroll. Use r to summon your fallen companion at full health."
	case itemStoryScroll:
		if it.StoryChapter > 0 && it.StoryChapter <= len(ghostDogStory) {
			return fmt.Sprintf("Ghost Dog story scroll %d/10: %s", it.StoryChapter, ghostDogStory[it.StoryChapter-1])
		}
		return "Ghost Dog story scroll."
	case itemSideStoryScroll:
		if it.SideStoryID >= 1 && it.SideStoryID <= len(ghostDogSideStories) {
			return "Side-story scroll: " + ghostDogSideStories[it.SideStoryID-1].Title + ". Adds a separate tale to your Book of Ghost Dog."
		}
		return "Ghost Dog side-story scroll."
	default:
		return it.Name
	}
}

func (g *game) visibleThreats(limit int) []*monster {
	var out []*monster
	for _, m := range g.current().Monsters {
		if m.HP > 0 && distance(g.player.Pos, m.Pos) <= limit && g.canSee(m.Pos) {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := distance(g.player.Pos, out[i].Pos), distance(g.player.Pos, out[j].Pos)
		if di == dj {
			return out[i].Name < out[j].Name
		}
		return di < dj
	})
	return out
}

func (g *game) visibleItems(limit int) []item {
	var out []item
	for _, it := range g.current().Items {
		if distance(g.player.Pos, it.Pos) <= limit && g.current().Tiles[it.Pos.Y][it.Pos.X] != '#' && g.canSee(it.Pos) {
			out = append(out, it)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := distance(g.player.Pos, out[i].Pos), distance(g.player.Pos, out[j].Pos)
		if di == dj {
			return out[i].Name < out[j].Name
		}
		return di < dj
	})
	return out
}

func pointLabel(p pos) string { return fmt.Sprintf("(%d,%d)", p.X, p.Y) }

func relativeTo(origin, target pos) string {
	dx, dy := target.X-origin.X, target.Y-origin.Y
	parts := []string{}
	if dy < 0 {
		parts = append(parts, fmt.Sprintf("%d north", -dy))
	} else if dy > 0 {
		parts = append(parts, fmt.Sprintf("%d south", dy))
	}
	if dx < 0 {
		parts = append(parts, fmt.Sprintf("%d west", -dx))
	} else if dx > 0 {
		parts = append(parts, fmt.Sprintf("%d east", dx))
	}
	if len(parts) == 0 {
		return "here"
	}
	return strings.Join(parts, ", ")
}

type inventoryLine struct {
	name        string
	quantity    *int
	description string
	kind        itemKind
}

func (g *game) inventoryLines() []inventoryLine {
	return []inventoryLine{
		{"Healing Potion", &g.player.Potions, "Use p to restore 12-18 health instantly.", itemPotion},
		{"Fire Scroll", &g.player.FireScrolls, "Use f to blast the nearest monster within 6 tiles and scorch adjacent foes.", itemFireScroll},
		{"Blink Stone", &g.player.BlinkStones, "Use g to teleport you and an unchained Ghost Dog far from monsters, outside secret rooms.", itemBlinkStone},
		{"Warding Charm", &g.player.WardingCharms, "Use b to reduce incoming damage to you and your Ghost Dog for the next 3 enemy turns.", itemWardingCharm},
		{"Sun Orb", &g.player.SunOrbs, "Use u to burn every undead monster on the current floor.", itemSunOrb},
		{"Frost Charm", &g.player.FrostCharms, "Use t to freeze and damage the nearest enemy for two turns.", itemFrostCharm},
		{"Starfire Orb", &g.player.StarfireOrbs, "Use o to deal 18-24 damage to the nearest enemy within 6 tiles.", itemStarfireOrb},
		{"Phoenix Ash", &g.player.PhoenixAshes, "Use n to restore 22-30 health below full health.", itemPhoenixAsh},
		{"Ghost Dog Recall Scroll", &g.player.GhostRecallScrolls, "Use r to summon a fallen Ghost Dog at full hit points.", itemGhostRecallScroll},
	}
}

func (g *game) contextHint() string {
	if hint := g.cryptHint(); hint != "" || g.inCrypt {
		return hint
	}
	if g.dogPresent() && g.dog.Chained && distance(g.player.Pos, g.dog.Pos) <= 1 {
		return "Ghost Dog is chained — move onto his d tile to free him."
	}
	lvl := g.current()
	if lvl.HasEscapeStair && lvl.primarySecret() != nil && lvl.primarySecret().Revealed && g.player.Pos == lvl.EscapeStairs {
		return "stairs up and out — press < to escape."
	}
	if lvl.HasUpStair && g.player.Pos == lvl.UpStairs {
		return "stairs up — press < to climb."
	}
	if lvl.HasStair && g.player.Pos == lvl.Stairs {
		return "stairs down — press > to descend."
	}
	return "i: inventory | m: inspect | h: help"
}

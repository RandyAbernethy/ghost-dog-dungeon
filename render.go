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
	fmt.Fprintf(w, "\n=== Ghost Dog Dungeon: Level %d/%d ===\n", g.currentLevel+1, levelCount)
	if lvl.Layout != "" {
		fmt.Fprintf(w, "%s — %s\n", lvl.Theme, lvl.Layout)
	}
	fmt.Fprintf(w, "HP %d/%d  Weapon %s (%d-%d)  Armor %d (block %d)  Potions %d  Fire %d  Blink %d  Ward %d  Sun %d  Frost %d  Starfire %d  Ash %d  Recall %d  Story %d/10\n", g.player.HP, g.player.MaxHP, g.player.Weapon.Name, g.player.Weapon.Min, g.player.Weapon.Max, g.player.armorDefense(), g.player.armorBlock(), g.player.Potions, g.player.FireScrolls, g.player.BlinkStones, g.player.WardingCharms, g.player.SunOrbs, g.player.FrostCharms, g.player.StarfireOrbs, g.player.PhoenixAshes, g.player.GhostRecallScrolls, len(g.player.StoryChapters))
	fmt.Fprintf(w, "Reach %d  Spell ward %d  Strike bonus %+d  Turns %d  Kills %d\n", g.player.Weapon.effectiveReach(), g.player.spellWard(), g.player.strikeBonus(), g.stats.Turns, g.stats.MonstersKilled)
	if g.dog.Freed {
		status := "gone"
		if g.dog.Alive {
			status = fmt.Sprintf("%d/%d HP", g.dog.HP, g.dog.MaxHP)
		}
		fmt.Fprintf(w, "Ghost dog: adopted (%s)\n", status)
	} else {
		fmt.Fprintln(w, "Ghost dog: not yet freed")
	}
	if g.player.ShieldTurns > 0 || g.player.HexedTurns > 0 || g.player.WebbedTurns > 0 {
		fmt.Fprintf(w, "Effects: ward %d  hex %d  web %d\n", g.player.ShieldTurns, g.player.HexedTurns, g.player.WebbedTurns)
	}
	for y := 0; y < mapHeight; y++ {
		var line strings.Builder
		for x := 0; x < mapWidth; x++ {
			p := pos{X: x, Y: y}
			ch := lvl.Tiles[y][x]
			m := g.monsterAt(lvl, p)
			f := g.fountainAt(lvl, p)
			it := g.itemAt(lvl, p)
			event := lvl.eventAt(p)
			switch {
			case g.player.Pos == p:
				ch = '@'
			case m != nil:
				ch = m.Glyph
			case g.dog.Freed && g.dog.Alive && g.dog.Pos == p:
				ch = 'D'
			case lvl.DogChain != nil && *lvl.DogChain == p && !g.dog.Freed:
				ch = 'd'
			case lvl.HasStair && lvl.Stairs == p:
				ch = '>'
			case lvl.HasUpStair && lvl.UpStairs == p:
				ch = '<'
			case lvl.HasEscapeStair && lvl.EscapeStairs == p && (lvl.Secret == nil || lvl.Secret.Revealed):
				ch = '<'
			case ch != '#' && f != nil && !f.Used:
				ch = 'F'
			case ch != '#' && f != nil:
				ch = 'f'
			case ch != '#' && it != nil:
				ch = it.Glyph
			case ch != '#' && event != nil && !event.Used:
				ch = event.glyph()
			default:
				// Keep the terrain glyph.
			}
			line.WriteRune(ch)
		}
		fmt.Fprintln(w, line.String())
	}
	fmt.Fprintln(w, "Legend: @ you  D ghost dog  d chained ghost dog  > down  < up/out  F/f fountain/spent  ) weapon  [ armor  ! potion  ? scroll  * magic  G friendly ghost  + lever  & keepsake  ^ shrine")
	if note := g.contextHint(); note != "" {
		fmt.Fprintln(w, "Here: "+note)
	}
	fmt.Fprintln(w, "Recent events:")
	for _, msg := range g.messages {
		fmt.Fprintln(w, "- "+msg)
	}
}

func (g *game) renderInventory(w io.Writer) {
	fmt.Fprintln(w, "\n=== Inventory & Equipment ===")
	fmt.Fprintf(w, "Weapon: %s (%d-%d)\n  %s\n", g.player.Weapon.Name, g.player.Weapon.Min, g.player.Weapon.Max, weaponDescription(g.player.Weapon))
	fmt.Fprintf(w, "Head: %s (+%d guard, %s)\n  %s\n", coloredArmorName(g.player.HeadArmor), g.player.HeadArmor.Defense, rarityLabel(g.player.HeadArmor.Rarity), armorDescription(g.player.HeadArmor))
	fmt.Fprintf(w, "Body: %s (+%d guard, %s)\n  %s\n", coloredArmorName(g.player.BodyArmor), g.player.BodyArmor.Defense, rarityLabel(g.player.BodyArmor.Rarity), armorDescription(g.player.BodyArmor))
	fmt.Fprintf(w, "Leg: %s (+%d guard, %s)\n  %s\n", coloredArmorName(g.player.LegArmor), g.player.LegArmor.Defense, rarityLabel(g.player.LegArmor.Rarity), armorDescription(g.player.LegArmor))
	fmt.Fprintf(w, "Total armor: %d guard, blocking %d damage from each hit before wards.\n", g.player.armorDefense(), g.player.armorBlock())
	fmt.Fprintf(w, "Spell ward: blocks %d extra spell damage. Strike bonus: %+d weapon damage.\n", g.player.spellWard(), g.player.strikeBonus())
	g.renderEquipment(w)
	fmt.Fprintf(w, "\nBook of Ghost Dog: %d/10 chapters, %d side stories. Press B or type book to read at any time.\n", len(g.player.StoryChapters), len(g.player.SideStories))
	fmt.Fprintf(w, "Optional challenges: %d turns, %d kills, %d Ghost Dog falls. Press j for records.\n", g.stats.Turns, g.stats.MonstersKilled, g.stats.GhostDogFalls)
	fmt.Fprintln(w, "Consumables:")
	for i, line := range g.inventoryLines() {
		fmt.Fprintf(w, "%d. %s x%d\n  %s\n", len(g.player.Weapons)+len(g.player.Armors)+i+1, line.name, *line.quantity, line.description)
	}
	companion := "The ghost dog is still chained somewhere below."
	if g.dog.Freed && g.dog.Alive {
		companion = fmt.Sprintf("Ghost dog companion: fighting at %d/%d HP and will attack nearby monsters.", g.dog.HP, g.dog.MaxHP)
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
		fmt.Fprintf(w, "- %c %s  HP %d  Damage %d-%d\n", m.Glyph, m.Name, m.MaxHP, m.MinDamage, m.MaxDamage)
		fmt.Fprintln(w, "  "+monsterSummary(kind))
		fmt.Fprintln(w, "  Trick: "+monsterPower(kind))
	}
	fmt.Fprintln(w, "Tip: press i to review your equipment and consumables.")
}

func (g *game) renderInspect(w io.Writer) {
	g.noteVisibleMonsters()
	fmt.Fprintln(w, "\n=== Inspect ===")
	fmt.Fprintf(w, "You stand at %s on level %d.\n", pointLabel(g.player.Pos), g.currentLevel+1)
	fmt.Fprintln(w, inspectTileDescription(g.current(), g.player.Pos))
	if g.current().HasStair {
		fmt.Fprintf(w, "Stairs down: %s.\n", relativeTo(g.player.Pos, g.current().Stairs))
	}
	if g.current().HasUpStair {
		fmt.Fprintf(w, "Stairs up: %s.\n", relativeTo(g.player.Pos, g.current().UpStairs))
	}
	if g.current().HasEscapeStair && (g.current().Secret == nil || g.current().Secret.Revealed) {
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
			fmt.Fprintf(w, "- %s: %s (%c) HP %d/%d damage %d-%d. %s\n", relativeTo(g.player.Pos, m.Pos), m.Name, m.Glyph, m.HP, m.MaxHP, m.MinDamage, m.MaxDamage, monsterPower(m.Kind))
		}
	}
	fmt.Fprintln(w, "Nearby loot:")
	loot := g.visibleItems(6)
	if len(loot) == 0 {
		fmt.Fprintln(w, "- No notable loot nearby.")
	} else {
		for _, it := range loot {
			fmt.Fprintf(w, "- %s: %s\n", relativeTo(g.player.Pos, it.Pos), inspectItemText(it))
		}
	}
	if g.dog.Freed && g.dog.Alive {
		fmt.Fprintf(w, "Ghost dog: %s.\n", relativeTo(g.player.Pos, g.dog.Pos))
	}
	fmt.Fprintln(w, "Tip: use movement or . to wait, or press k beside suspicious walls to search.")
}

func inspectTileDescription(lvl *level, p pos) string {
	switch {
	case lvl.HasEscapeStair && lvl.EscapeStairs == p && (lvl.Secret == nil || lvl.Secret.Revealed):
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

func inspectItemText(it item) string {
	switch it.Kind {
	case itemWeapon:
		return fmt.Sprintf("weapon %s (%d-%d, reach %d). %s", it.Weapon.Name, it.Weapon.Min, it.Weapon.Max, it.Weapon.effectiveReach(), weaponDescription(it.Weapon))
	case itemArmor:
		return fmt.Sprintf("%s armor for the %s slot (+%d guard, %s). %s %s", coloredArmorName(it.Armor), slotLabel(it.Armor.Slot), it.Armor.Defense, rarityLabel(it.Armor.Rarity), armorTraits(it.Armor), armorDescription(it.Armor))
	case itemPotion:
		return "healing potion. Restores 12-18 health."
	case itemFireScroll:
		return "fire scroll. Blasts the nearest enemy within 6 tiles."
	case itemBlinkStone:
		return "blink stone. Teleports you and Ghost Dog far from monsters, outside secret rooms."
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
		if m.HP > 0 && distance(g.player.Pos, m.Pos) <= limit {
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
		if distance(g.player.Pos, it.Pos) <= limit && g.current().Tiles[it.Pos.Y][it.Pos.X] != '#' {
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
	glyph       rune
}

func (g *game) inventoryLines() []inventoryLine {
	return []inventoryLine{
		{"Healing Potion", &g.player.Potions, "Use p to restore 12-18 health instantly.", itemPotion, '!'},
		{"Fire Scroll", &g.player.FireScrolls, "Use f to blast the nearest monster within 6 tiles and scorch adjacent foes.", itemFireScroll, '?'},
		{"Blink Stone", &g.player.BlinkStones, "Use g to teleport you and Ghost Dog far from monsters, outside secret rooms.", itemBlinkStone, '*'},
		{"Warding Charm", &g.player.WardingCharms, "Use b to reduce incoming damage to you and your Ghost Dog for the next 3 enemy turns.", itemWardingCharm, '*'},
		{"Sun Orb", &g.player.SunOrbs, "Use u to burn every undead monster on the current floor.", itemSunOrb, '*'},
		{"Frost Charm", &g.player.FrostCharms, "Use t to freeze and damage the nearest enemy for two turns.", itemFrostCharm, '*'},
		{"Starfire Orb", &g.player.StarfireOrbs, "Use o to deal 18-24 damage to the nearest enemy within 6 tiles.", itemStarfireOrb, '*'},
		{"Phoenix Ash", &g.player.PhoenixAshes, "Use n to restore 22-30 health below full health.", itemPhoenixAsh, '*'},
		{"Ghost Dog Recall Scroll", &g.player.GhostRecallScrolls, "Use r to summon a fallen Ghost Dog at full hit points.", itemGhostRecallScroll, '?'},
	}
}

func (g *game) contextHint() string {
	lvl := g.current()
	if hint := g.eventHint(); hint != "" {
		return hint
	}
	for _, f := range lvl.Fountains {
		if !f.Used && lvl.Tiles[f.Pos.Y][f.Pos.X] != '#' && distance(g.player.Pos, f.Pos) <= 1 {
			return "healing fountain nearby — press v to drink."
		}
	}
	if lvl.HasEscapeStair && lvl.Secret != nil && lvl.Secret.Revealed && g.player.Pos == lvl.EscapeStairs {
		return "stairs up and out — press < to escape."
	}
	if lvl.HasUpStair && g.player.Pos == lvl.UpStairs {
		return "stairs up — press < to climb."
	}
	if lvl.HasStair && g.player.Pos == lvl.Stairs {
		return "stairs down — press > to descend."
	}
	if hidden := lvl.hiddenRoomNear(g.player.Pos); hidden != nil {
		if hidden == lvl.Secret && lvl.HasEscapeStair && g.bossAlive(lvl) {
			return "a necromantic seal chills this wall — the Lich still holds it shut."
		}
		return "this wall feels strange — try search."
	}
	return "press i for equipment, c for the codex, m to inspect, or k to search suspicious walls."
}

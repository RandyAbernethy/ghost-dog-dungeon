package main

import (
	"fmt"
	"io"
)

func epicStrikeName(kind monsterKind) string {
	switch kind {
	case monsterIronColossus:
		return "Hammerfall"
	case monsterGraveTitan:
		return "Grave Cross"
	case monsterAshenWyrm:
		return "Cinder Lane"
	case monsterHollowReaper:
		return "Soul Scythe"
	case monsterFrostMonarch:
		return "Ice Bloom"
	case monsterStormLeviathan:
		return "Thunder Marks"
	case monsterDuskTyrant:
		return "Eclipse"
	case monsterObsidianGolem:
		return "Shard Fan"
	}
	return ""
}

func containsPosition(positions []pos, p pos) bool {
	for _, candidate := range positions {
		if candidate == p {
			return true
		}
	}
	return false
}

func (g *game) epicDangerAt(p pos) bool {
	for _, m := range g.current().Monsters {
		if m.HP > 0 && epicMonster(m.Kind) && containsPosition(m.StrikeTiles, p) {
			return true
		}
	}
	return false
}

func (g *game) prepareEpicStrike(m *monster) {
	aim := g.player.Pos
	if g.dogPresent() && distance(m.Pos, g.dog.Pos) < distance(m.Pos, aim) {
		aim = g.dog.Pos
	}
	if distance(m.Pos, aim) > 2 || !g.clearEpicLine(m.Pos, aim) {
		return
	}
	var tiles []pos
	add := func(p pos) {
		if g.inBounds(p) && p != m.Pos && g.current().Tiles[p.Y][p.X] != '#' &&
			g.clearEpicLine(m.Pos, p) && !containsPosition(tiles, p) {
			tiles = append(tiles, p)
		}
	}
	add(aim)
	switch m.Kind {
	case monsterIronColossus, monsterAshenWyrm:
		radius := 1
		if m.Kind == monsterAshenWyrm {
			radius = 2
		}
		// The hammer sweeps across the approach; the Wyrm burns along it.
		horizontal := abs(aim.X-m.Pos.X) >= abs(aim.Y-m.Pos.Y)
		if m.Kind == monsterIronColossus {
			horizontal = !horizontal
		}
		for offset := -radius; offset <= radius; offset++ {
			p := aim
			if horizontal {
				p.X += offset
			} else {
				p.Y += offset
			}
			add(p)
		}
	case monsterGraveTitan, monsterFrostMonarch:
		for _, d := range []pos{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			add(pos{aim.X + d.X, aim.Y + d.Y})
		}
	case monsterHollowReaper:
		add(pos{aim.X - 1, aim.Y - 1})
		add(pos{aim.X + 1, aim.Y + 1})
	case monsterStormLeviathan:
		if distance(m.Pos, g.player.Pos) <= 2 {
			add(g.player.Pos)
		}
		if g.dogPresent() && distance(m.Pos, g.dog.Pos) <= 2 {
			add(g.dog.Pos)
		}
	case monsterDuskTyrant:
		for _, d := range []pos{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
			add(pos{aim.X + d.X, aim.Y + d.Y})
		}
	case monsterObsidianGolem:
		for dx := -1; dx <= 1; dx++ {
			add(pos{aim.X + dx, aim.Y + sign(aim.Y-m.Pos.Y)})
		}
	}
	// Do not promise a dodge when a wall or another actor blocks every exit.
	canEscape := func(p pos) bool {
		if !containsPosition(tiles, p) {
			return true
		}
		for _, next := range g.openNeighbors(p, true) {
			if !containsPosition(tiles, next) &&
				(!g.dogPresent() || next != g.dog.Pos) {
				return true
			}
		}
		return false
	}
	if !canEscape(g.player.Pos) || (g.dogPresent() && !canEscape(g.dog.Pos)) {
		return
	}
	m.StrikeTiles = tiles
	g.addMessage(fmt.Sprintf("%s prepares %s! Red X tiles will be struck on its next active turn. Move clear; the marked tiles will not follow you.", m.Name, epicStrikeName(m.Kind)))
}

// Walls stop both warning and damage; hidden chambers remain concealed.
func (g *game) clearEpicLine(from, to pos) bool {
	dx, dy := abs(to.X-from.X), -abs(to.Y-from.Y)
	sx, sy := sign(to.X-from.X), sign(to.Y-from.Y)
	err := dx + dy
	for {
		if !g.inBounds(from) || g.current().Tiles[from.Y][from.X] == '#' {
			return false
		}
		if from == to {
			return true
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			from.X += sx
		}
		if e2 <= dx {
			err += dx
			from.Y += sy
		}
	}
}

func (g *game) resolveEpicStrike(m *monster) bool {
	if !epicMonster(m.Kind) || len(m.StrikeTiles) == 0 {
		return false
	}
	tiles := m.StrikeTiles
	m.StrikeTiles = nil
	m.RetaliationTurns = 0
	name := epicStrikeName(m.Kind)
	g.addMessage(m.Name + " unleashes " + name + " on the marked tiles!")
	hit := false
	if containsPosition(tiles, g.player.Pos) {
		hit = true
		before := g.player.HP
		if m.Kind == monsterIronColossus && g.player.ShieldTurns > 0 {
			g.player.ShieldTurns = 0
			g.addMessage("Hammerfall crushes your warding charm.")
		}
		damage := g.randRange(m.MinDamage, m.MaxDamage)
		source := m.Name + "'s " + name
		if m.Kind == monsterIronColossus || m.Kind == monsterObsidianGolem {
			g.hurtPlayer(damage, source)
		} else {
			g.hurtPlayerSpell(damage, source)
		}
		if g.player.HP < before {
			switch m.Kind {
			case monsterFrostMonarch:
				g.bindPlayer(bindingIce)
			case monsterDuskTyrant:
				g.hexPlayer()
			case monsterGraveTitan:
				m.HP = min(m.MaxHP, m.HP+8)
			}
		}
		g.godArmorReprisal(m, before)
	}
	if g.dogPresent() && containsPosition(tiles, g.dog.Pos) {
		hit = true
		g.hurtDog(g.randRange(m.MinDamage, m.MaxDamage), m.Name+"'s "+name)
	}
	if !hit {
		g.addMessage(name + " strikes empty stone. You escaped the warning.")
	}
	return true
}

func (g *game) cancelEpicWarning(m *monster) {
	m.RetaliationTurns = 0
	if len(m.StrikeTiles) > 0 {
		m.StrikeTiles = nil
		g.addMessage(m.Name + " abandons its prepared attack.")
	}
}

func (g *game) dodgeEpicWarning() bool {
	if !g.dog.active() || !g.epicDangerAt(g.dog.Pos) {
		return false
	}
	var best *pos
	for _, p := range g.openNeighbors(g.dog.Pos, true) {
		if !g.epicDangerAt(p) && (best == nil || distance(p, g.player.Pos) < distance(*best, g.player.Pos)) {
			candidate := p
			best = &candidate
		}
	}
	if best != nil {
		g.dog.Pos = *best
		g.addMessage("Ash springs clear of the marked tiles instead of attacking.")
	} else if step, ok := g.epicEscapeStep(g.dog.Pos, g.epicDangerAt); ok {
		g.dog.Pos = step
		g.addMessage("Ash starts a two-step retreat from the wide warning instead of attacking.")
	}
	// If surrounded, stay put rather than spending the action on an attack.
	return true
}

func (g *game) renderEpicWarnings(w io.Writer) {
	for _, m := range g.current().Monsters {
		if m.HP > 0 && epicMonster(m.Kind) && m.Awakened {
			uiText(w, fmt.Sprintf("%s: rage %d/%d | normal range 2; half signature damage", m.Name, m.Rage, epicRageLimit))
		}
		if m.HP <= 0 || len(m.StrikeTiles) == 0 {
			continue
		}
		if m.RetaliationTurns > 0 {
			uiText(w, fmt.Sprintf("DANGER: RETALIATION in %d action(s). Move now or use protection/interrupt/blink.", m.RetaliationTurns))
		}
		uiText(w, fmt.Sprintf("WARNING: %s — %s; red X tiles strike on its next active turn.", m.Name, epicStrikeName(m.Kind)))
		if containsPosition(m.StrikeTiles, g.player.Pos) {
			uiText(w, "YOU are on a marked tile. Move clear.")
		}
		if m.FrozenTurns > 0 || m.StunnedTurns > 0 {
			uiText(w, "Attack delayed while disabled.")
		}
	}
}

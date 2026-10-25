package main

import "fmt"

const epicRageLimit = 4

func (g *game) epicNormalTurn(m *monster) {
	target := g.player.Pos
	dog := g.dogPresent() && distance(m.Pos, g.dog.Pos) < distance(m.Pos, target)
	if dog {
		target = g.dog.Pos
	}
	if distance(m.Pos, target) > 2 || !g.clearEpicLine(m.Pos, target) {
		g.moveMonster(m)
		return
	}
	damage := g.randRange(max(1, m.MinDamage/2), max(1, m.MaxDamage/2))
	if distance(m.Pos, target) == 1 {
		if dog {
			g.attackDogDamage(m, max(1, damage-1))
		} else {
			g.attackPlayerDamage(m, damage)
		}
		return
	}
	source := m.Name + " uses a normal ranged strike"
	if dog {
		g.hurtDog(damage, source)
		return
	}
	before := g.player.HP
	if m.Kind == monsterIronColossus || m.Kind == monsterObsidianGolem {
		g.hurtPlayer(damage, source)
	} else {
		g.hurtPlayerSpell(damage, source)
	}
	g.godArmorReprisal(m, before)
}

// Find a legal first step leading out of every pending pattern in at most two
// actions. Occupied squares count as blocked; path order is deterministic.
func (g *game) epicEscapeStep(from pos, dangerous func(pos) bool) (pos, bool) {
	player := from == g.player.Pos
	if player && g.player.BoundTurns > 0 {
		return pos{}, false
	}
	for _, next := range g.openNeighbors(from, true) {
		if !dangerous(next) && g.legalEpicEscapeStep(from, next, player) {
			return next, true
		}
	}
	for _, next := range g.openNeighbors(from, true) {
		if !g.legalEpicEscapeStep(from, next, player) {
			continue
		}
		for _, end := range g.openNeighbors(next, true) {
			if end != from && !dangerous(end) && g.legalEpicEscapeStep(next, end, player) {
				return next, true
			}
		}
	}
	return pos{}, false
}

func (g *game) legalEpicEscapeStep(from, to pos, player bool) bool {
	if g.dogPresent() && to == g.dog.Pos {
		return false
	}
	if !player {
		return true
	}
	// A direction with a reach target attacks instead of moving.
	dx, dy := to.X-from.X, to.Y-from.Y
	for step := 2; step <= g.player.Weapon.effectiveReach(); step++ {
		target := pos{from.X + dx*step, from.Y + dy*step}
		if !g.inBounds(target) || g.current().Tiles[target.Y][target.X] == '#' {
			break
		}
		if g.monsterAt(g.current(), target) != nil {
			return false
		}
	}
	return true
}

func (g *game) prepareEpicRetaliation(m *monster) bool {
	var tiles []pos
	addArea := func(center pos) {
		if distance(m.Pos, center) > 4 || !g.clearEpicLine(m.Pos, center) {
			return
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				p := pos{center.X + dx, center.Y + dy}
				if g.inBounds(p) && p != m.Pos && g.current().Tiles[p.Y][p.X] != '#' && g.clearEpicLine(m.Pos, p) && !containsPosition(tiles, p) {
					tiles = append(tiles, p)
				}
			}
		}
	}
	addArea(g.player.Pos)
	if g.dogPresent() {
		addArea(g.dog.Pos)
	}
	if len(tiles) == 0 {
		return false
	}
	dangerous := func(p pos) bool { return containsPosition(tiles, p) || g.epicDangerAt(p) }
	if containsPosition(tiles, g.player.Pos) {
		if _, ok := g.epicEscapeStep(g.player.Pos, dangerous); !ok {
			return false
		}
	}
	if g.dogPresent() && containsPosition(tiles, g.dog.Pos) {
		if g.dog.Chained {
			return false
		}
		if _, ok := g.epicEscapeStep(g.dog.Pos, dangerous); !ok {
			return false
		}
	}
	m.StrikeTiles, m.RetaliationTurns, m.Rage = tiles, 2, 0
	g.addMessage(fmt.Sprintf("%s rage erupts! RETALIATION: %s covers a wide area in TWO actions. Move twice now, interrupt, protect, or blink; marked tiles will not follow you.", m.Name, epicStrikeName(m.Kind)))
	return true
}

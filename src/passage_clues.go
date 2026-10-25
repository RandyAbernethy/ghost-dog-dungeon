package main

import "strings"

// Follow open terrain to a door's approach, rather than leaking clues through
// intervening walls or unopened parent chambers. No random or persisted state.
func (g *game) clueNear(from, door pos) bool {
	if distance(from, door) > 3 {
		return false
	}
	queue := []pos{from}
	steps := map[pos]int{from: 0}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if distance(p, door) <= 1 {
			return true
		}
		if steps[p] >= 2 {
			continue
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				next := pos{p.X + dx, p.Y + dy}
				if !g.inBounds(next) || g.current().Tiles[next.Y][next.X] == '#' {
					continue
				}
				if _, exists := steps[next]; !exists {
					steps[next] = steps[p] + 1
					queue = append(queue, next)
				}
			}
		}
	}
	return false
}

func (g *game) passageClues(from pos) []string {
	var clues []string
	add := func(door pos, detail string) {
		if g.clueNear(from, door) {
			clues = append(clues, detail+" "+relativeTo(from, door)+".")
		}
	}
	if g.inCrypt {
		if !g.crypt.ExitRevealed {
			add(g.crypt.Exit, "Cold air leaks through the stone")
		}
		return clues
	}
	lvl := g.current()
	for _, root := range lvl.Secrets {
		if hidden := root.nextHiddenRoom(); hidden != nil {
			detail := "A cool draft stirs dust beside disturbed mortar"
			if hidden == lvl.primarySecret() && lvl.HasEscapeStair && g.bossAlive(lvl) {
				detail = "A necromantic seal chills the wall; the Lich holds it shut"
			} else if g.dogPresent() {
				detail = "Ash sniffs a draft beside disturbed mortar"
			}
			add(hidden.Door, detail)
		}
	}
	if c := g.crypt; c != nil && g.currentLevel == c.ParentFloor && !c.EntranceRevealed && !c.Floor.Visited {
		add(c.Entrance, "Cold dust gathers beneath a burial seal")
	}
	return clues
}

func (g *game) passageClue(from pos) string { return strings.Join(g.passageClues(from), " ") }

// Narrative rolls use a copy of the saved RNG, salted with the saved turn so
// quiet movement can vary hints without changing combat or campaign replays.
// Only successful movement calls this; reads and saves reuse the current log.
func (g *game) maybePassageHint() {
	if g.won || g.player.HP <= 0 {
		return
	}
	clues := g.passageClues(g.player.Pos)
	if len(clues) == 0 {
		return
	}
	rng := *g.rng
	rng.State ^= uint64(g.stats.Turns) * 0x9e3779b97f4a7c15
	if rng.Intn(100) < 20 {
		g.addMessage(clues[rng.Intn(len(clues))])
	}
}

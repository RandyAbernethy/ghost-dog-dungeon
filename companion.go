package main

import (
	"fmt"
	"math"
)

func (g *game) restoreGhostDogAfterLichDeath() {
	if !g.dog.Freed || g.dog.Alive {
		return
	}
	g.dog.Alive = true
	g.dog.HP = g.dog.MaxHP
	g.placeDogNearPlayer()
	g.addMessage("The Dread Lich's death frees the Ghost Dog. It reappears at full hit points.")
}

func (g *game) dogTurn() {
	if !g.dog.Alive {
		return
	}
	if focus := g.activeDogFocus(); focus != nil {
		if distance(g.dog.Pos, focus.Pos) == 1 {
			g.dogAttack(focus)
			return
		}
		g.moveDogToward(focus.Pos)
		if distance(g.dog.Pos, focus.Pos) == 1 {
			g.dogAttack(focus)
		}
		return
	}
	if target := g.adjacentMonster(g.dog.Pos); target != nil {
		g.dogAttack(target)
		return
	}
	g.keepDogWithPlayer()
}

func (g *game) dogAttack(target *monster) {
	dmg := g.randRange(6, 9)
	g.addMessage(fmt.Sprintf("Ghost dog tears into %s for %d damage.", target.Name, dmg))
	g.damageMonster(target, dmg, "mauls")
}

func (g *game) activeDogFocus() *monster {
	if g.dogFocus != nil && g.dogFocus.HP > 0 && (distance(g.player.Pos, g.dogFocus.Pos) <= 2 || distance(g.dog.Pos, g.dogFocus.Pos) == 1) {
		return g.dogFocus
	}
	if target := g.adjacentMonster(g.player.Pos); target != nil {
		g.dogFocus = target
		return target
	}
	g.dogFocus = nil
	return nil
}

func (g *game) keepDogWithPlayer() {
	if g.dog.Pos == g.player.Pos || g.monsterAt(g.current(), g.dog.Pos) != nil {
		if g.placeDogNearPlayer() {
			return
		}
	}
	if distance(g.dog.Pos, g.player.Pos) <= 1 {
		return
	}
	best := g.bestDogNeighborNearPlayer()
	if best != nil {
		g.dog.Pos = *best
		return
	}
	g.moveActorToward(&g.dog.Pos, g.player.Pos, true)
}

func (g *game) moveDogToward(target pos) {
	if distance(g.dog.Pos, g.player.Pos) > 2 {
		if best := g.bestDogNeighborNearPlayer(); best != nil {
			g.dog.Pos = *best
			if distance(g.dog.Pos, target) <= 1 {
				return
			}
		}
	}
	if distance(target, g.player.Pos) == 1 && g.stepDogTowardPlayersFight(target) {
		return
	}
	g.moveActorToward(&g.dog.Pos, target, true)
	if distance(g.dog.Pos, g.player.Pos) > 2 {
		g.keepDogWithPlayer()
	}
}

// Find one step toward the fight without crossing the player, walls, or other
// monsters. Keep the detour within the companion's usual two-tile radius.
func (g *game) stepDogTowardPlayersFight(target pos) bool {
	type route struct {
		position  pos
		firstStep pos
	}
	start := g.dog.Pos
	queue := []route{{position: start, firstStep: start}}
	seen := map[pos]bool{start: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if distance(current.position, target) == 1 {
			g.dog.Pos = current.firstStep
			return true
		}
		for _, next := range g.openNeighbors(current.position, true) {
			if seen[next] || distance(next, g.player.Pos) > 2 {
				continue
			}
			seen[next] = true
			firstStep := current.firstStep
			if current.position == start {
				firstStep = next
			}
			queue = append(queue, route{position: next, firstStep: firstStep})
		}
	}
	return false
}

func (g *game) bestDogNeighborNearPlayer() *pos {
	spots := g.openNeighbors(g.player.Pos, true)
	if len(spots) == 0 {
		return nil
	}
	best := spots[0]
	bestScore := 999999
	for _, p := range spots {
		if !g.canDogOccupyAfterPlayerMove(p, g.player.Pos) {
			continue
		}
		score := distance(p, g.dog.Pos)*10 + distance(p, g.player.Pos) + g.doorwayPenalty(p)
		if score < bestScore {
			best = p
			bestScore = score
		}
	}
	if bestScore == 999999 {
		return nil
	}
	return &best
}

func (g *game) placeDogNearPlayer() bool {
	if best := g.bestDogNeighborNearPlayer(); best != nil {
		g.dog.Pos = *best
		return true
	}
	lvl := g.current()
	best := pos{}
	bestDistance := math.MaxInt
	found := false
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{X: x, Y: y}
			if lvl.Tiles[y][x] == '#' || p == g.player.Pos || g.monsterAt(lvl, p) != nil {
				continue
			}
			if d := distance(p, g.player.Pos); d < bestDistance {
				best, bestDistance, found = p, d, true
			}
		}
	}
	if found {
		g.dog.Pos = best
	}
	return found
}

func (g *game) doorwayPenalty(p pos) int {
	orth := 0
	for _, delta := range []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		n := pos{X: p.X + delta.X, Y: p.Y + delta.Y}
		if g.inBounds(n) && g.current().Tiles[n.Y][n.X] != '#' {
			orth++
		}
	}
	if orth <= 2 {
		return 8
	}
	return 0
}

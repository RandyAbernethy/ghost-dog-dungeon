package main

import (
	"math"
)

func newSimpleRNG(seed int64) *simpleRNG {
	state := uint64(seed)
	if state == 0 {
		state = 1
	}
	return &simpleRNG{State: state}
}

func (r *simpleRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	x := r.State
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	if x == 0 {
		x = 0x9e3779b97f4a7c15
	}
	r.State = x
	return int(x % uint64(n))
}

func (g *game) randomOpenTile(lvl *level, occupied map[pos]bool) pos {
	floors := make([]pos, 0, mapWidth*mapHeight)
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{X: x, Y: y}
			if occupied[p] || lvl.Tiles[y][x] == '#' {
				continue
			}
			floors = append(floors, p)
		}
	}
	if len(floors) == 0 {
		panic("could not place tile")
	}
	return floors[g.rng.Intn(len(floors))]
}

func (g *game) randomOpenTileFarFrom(lvl *level, occupied map[pos]bool, origin pos, minDistance int) pos {
	floors := make([]pos, 0, mapWidth*mapHeight)
	far := []pos{}
	best := pos{}
	bestDist := -1
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{X: x, Y: y}
			if occupied[p] || lvl.Tiles[y][x] == '#' {
				continue
			}
			floors = append(floors, p)
			d := distance(origin, p)
			if d >= minDistance {
				far = append(far, p)
			}
			if d > bestDist {
				bestDist = d
				best = p
			}
		}
	}
	if len(far) > 0 {
		return far[g.rng.Intn(len(far))]
	}
	if len(floors) == 0 {
		panic("could not place far tile")
	}
	return best
}

func (g *game) itemAt(lvl *level, p pos) *item {
	for i := range lvl.Items {
		if lvl.Items[i].Pos == p {
			return &lvl.Items[i]
		}
	}
	return nil
}

func (g *game) fountainAt(lvl *level, p pos) *fountain {
	for i := range lvl.Fountains {
		if lvl.Fountains[i].Pos == p {
			return &lvl.Fountains[i]
		}
	}
	return nil
}

func (g *game) monsterAt(lvl *level, p pos) *monster {
	for _, m := range lvl.Monsters {
		if m.HP > 0 && m.Pos == p {
			return m
		}
	}
	return nil
}

func (g *game) bossAlive(lvl *level) bool {
	for _, m := range lvl.Monsters {
		if m.Boss && m.HP > 0 {
			return true
		}
	}
	return false
}

func (g *game) adjacentMonster(p pos) *monster {
	for _, m := range g.current().Monsters {
		if m.HP > 0 && distance(m.Pos, p) == 1 {
			return m
		}
	}
	return nil
}

func (g *game) nearestMonster(limit int) *monster {
	return g.nearestMonsterFromLimit(g.player.Pos, limit)
}

func (g *game) nearestMonsterFrom(origin pos) *monster {
	return g.nearestMonsterFromLimit(origin, math.MaxInt)
}

func (g *game) nearestMonsterFromLimit(origin pos, limit int) *monster {
	var best *monster
	bestDist := math.MaxInt
	for _, m := range g.current().Monsters {
		if m.HP <= 0 {
			continue
		}
		d := distance(origin, m.Pos)
		if d < bestDist && d <= limit {
			best, bestDist = m, d
		}
	}
	return best
}

func (g *game) openNeighbors(center pos, ignoreDog bool) []pos {
	options := make([]pos, 0, 8)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			p := pos{X: center.X + dx, Y: center.Y + dy}
			if g.isWalkable(p, ignoreDog) {
				options = append(options, p)
			}
		}
	}
	return options
}

func (g *game) isWalkable(p pos, ignoreDog bool) bool {
	if !g.inBounds(p) || g.current().Tiles[p.Y][p.X] == '#' || g.player.Pos == p {
		return false
	}
	if !ignoreDog && !g.dog.Freed && g.current().DogChain != nil && *g.current().DogChain == p {
		return false
	}
	if !ignoreDog && g.dog.Freed && g.dog.Alive && g.dog.Pos == p {
		return false
	}
	return g.monsterAt(g.current(), p) == nil
}

func (g *game) inBounds(p pos) bool { return p.X >= 0 && p.X < mapWidth && p.Y >= 0 && p.Y < mapHeight }

func distance(a, b pos) int {
	dx := abs(a.X - b.X)
	dy := abs(a.Y - b.Y)
	if dx > dy {
		return dx
	}
	return dy
}

func sign(v int) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	default:
		return 0
	}
}

func (g *game) randRange(minVal, maxVal int) int {
	if maxVal <= minVal {
		return minVal
	}
	return minVal + g.rng.Intn(maxVal-minVal+1)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

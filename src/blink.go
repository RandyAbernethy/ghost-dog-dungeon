package main

import "math"

type blinkLanding struct {
	player pos
	dog    pos
}

func (g *game) safestBlinkLanding() (blinkLanding, bool) {
	lvl := g.current()
	secret := map[pos]bool{}
	for _, room := range lvl.secretRooms() {
		secret[room.Door] = true
		for _, p := range room.Tiles {
			secret[p] = true
		}
		for _, p := range room.Walls {
			secret[p] = true
		}
	}
	clearance := map[pos]int{}
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{X: x, Y: y}
			if lvl.Tiles[y][x] == '#' || secret[p] || g.monsterAt(lvl, p) != nil || (!g.dog.Freed && lvl.DogChain != nil && p == *lvl.DogChain) {
				continue
			}
			nearest := math.MaxInt
			for _, m := range lvl.Monsters {
				if m.HP > 0 {
					nearest = min(nearest, distance(p, m.Pos))
				}
			}
			clearance[p] = nearest
		}
	}
	withDog := g.dog.Freed && g.dog.Alive
	bestSafety, bestPlayerSafety := -1, -1
	var best []blinkLanding
	consider := func(playerPos, dogPos pos, safety int) {
		playerSafety := clearance[playerPos]
		if safety > bestSafety || (safety == bestSafety && playerSafety > bestPlayerSafety) {
			bestSafety, bestPlayerSafety = safety, playerSafety
			best = best[:0]
		}
		if safety == bestSafety && playerSafety == bestPlayerSafety {
			best = append(best, blinkLanding{player: playerPos, dog: dogPos})
		}
	}
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{X: x, Y: y}
			playerSafety, ok := clearance[p]
			if !ok || p == g.player.Pos {
				continue
			}
			if !withDog {
				consider(p, g.dog.Pos, playerSafety)
				continue
			}
			// Score the pair by whichever companion is closer to danger.
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					dogPos := pos{X: x + dx, Y: y + dy}
					if dogSafety, ok := clearance[dogPos]; ok {
						consider(p, dogPos, min(playerSafety, dogSafety))
					}
				}
			}
		}
	}
	if len(best) == 0 {
		return blinkLanding{}, false
	}
	return best[g.rng.Intn(len(best))], true
}

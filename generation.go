package main

func floorBudget(depth int) int { return 40 + depth*30 }

func cloneTiles(tiles [][]rune) [][]rune {
	clone := make([][]rune, len(tiles))
	for i := range tiles {
		clone[i] = append([]rune(nil), tiles[i]...)
	}
	return clone
}

func countFloorTiles(tiles [][]rune) int {
	count := 0
	for y := range tiles {
		for x := range tiles[y] {
			if tiles[y][x] != '#' {
				count++
			}
		}
	}
	return count
}

func carveRoom(tiles [][]rune, rm room) {
	for y := rm.Y; y < rm.Y+rm.H; y++ {
		for x := rm.X; x < rm.X+rm.W; x++ {
			tiles[y][x] = '.'
		}
	}
}

func connectRooms(tiles [][]rune, a, b pos, rng *simpleRNG) {
	if rng.Intn(2) == 0 {
		carveHorizontal(tiles, a.X, b.X, a.Y)
		carveVertical(tiles, a.Y, b.Y, b.X)
	} else {
		carveVertical(tiles, a.Y, b.Y, a.X)
		carveHorizontal(tiles, a.X, b.X, b.Y)
	}
}

func carveHorizontal(tiles [][]rune, x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		tiles[y][x] = '.'
	}
}

func carveVertical(tiles [][]rune, y1, y2, x int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		tiles[y][x] = '.'
	}
}

func center(r room) pos { return pos{X: r.X + r.W/2, Y: r.Y + r.H/2} }

func roomsOverlap(a, b room, margin int) bool {
	return a.X-margin < b.X+b.W && a.X+a.W+margin > b.X && a.Y-margin < b.Y+b.H && a.Y+a.H+margin > b.Y
}

func (g *game) populateLevel(lvl *level) {
	occupied := map[pos]bool{lvl.Start: true}
	if lvl.Index > 0 {
		lvl.HasUpStair = true
		lvl.UpStairs = lvl.Start
	}
	if lvl.Index < levelCount-1 {
		lvl.HasStair = true
		lvl.Stairs = g.randomOpenTile(lvl, occupied)
		occupied[lvl.Stairs] = true
	}
	if lvl.Index == 1 {
		dogPos := g.randomOpenTile(lvl, occupied)
		lvl.DogChain = &dogPos
		occupied[dogPos] = true
	}
	var bossPos pos
	bossPlaced := false
	if lvl.Index == levelCount-1 {
		bossPos, bossPlaced = g.bossGuardPosition(lvl, occupied)
		if bossPlaced {
			occupied[bossPos] = true
		}
	}
	for _, it := range g.themedItems(lvl) {
		it.Pos = g.randomOpenTile(lvl, occupied)
		occupied[it.Pos] = true
		lvl.Items = append(lvl.Items, it)
	}
	for _, spec := range g.themedMonsters(lvl) {
		if spec.Boss {
			spec.StoryChapter = levelCount
			if bossPlaced {
				spec.Pos = bossPos
			} else {
				spec.Pos = g.randomOpenTileFarFrom(lvl, occupied, lvl.Start, 12)
			}
		} else {
			spec.Pos = g.randomOpenTile(lvl, occupied)
		}
		occupied[spec.Pos] = true
		m := spec
		lvl.Monsters = append(lvl.Monsters, &m)
	}
}

func (g *game) placeFountains() {
	order := make([]int, len(g.levels))
	for i := range order {
		order[i] = i
	}
	for i := len(order) - 1; i > 0; i-- {
		j := g.rng.Intn(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	count := 3 + g.rng.Intn(3)
	for _, levelIndex := range order[:count] {
		lvl := g.levels[levelIndex]
		spots := make([]pos, 0, floorBudget(levelIndex))
		for y := 1; y < mapHeight-1; y++ {
			for x := 1; x < mapWidth-1; x++ {
				p := pos{X: x, Y: y}
				if lvl.Tiles[y][x] == '#' || p == lvl.Start || p == g.player.Pos ||
					(lvl.HasStair && p == lvl.Stairs) || (lvl.HasUpStair && p == lvl.UpStairs) ||
					(lvl.DogChain != nil && p == *lvl.DogChain) || g.monsterAt(lvl, p) != nil || g.itemAt(lvl, p) != nil || g.fountainAt(lvl, p) != nil || lvl.eventAt(p) != nil {
					continue
				}
				spots = append(spots, p)
			}
		}
		if len(spots) > 0 {
			lvl.Fountains = append(lvl.Fountains, fountain{Pos: spots[g.rng.Intn(len(spots))]})
		}
	}
}

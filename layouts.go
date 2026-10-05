package main

type layoutKind string

const (
	layoutCaves    layoutKind = "winding caves"
	layoutCrypt    layoutKind = "narrow crypt passages"
	layoutChambers layoutKind = "connected chambers"
	layoutHall     layoutKind = "pillared halls"
)

var layoutKinds = []layoutKind{layoutCaves, layoutCrypt, layoutChambers, layoutHall}

func makeStyledFloorPlan(rng *simpleRNG, depth int, style layoutKind) ([][]rune, pos) {
	tiles := blankFloor()
	budget := floorBudget(depth)
	w, h := 4+rng.Intn(3), 3+rng.Intn(2)
	if style == layoutCrypt {
		w, h = 3, 3
	}
	if style == layoutHall {
		w, h = 9+rng.Intn(4), 3+depth/4
	}
	for w*h > budget-4 {
		h--
	}
	startRoom := room{X: 1 + rng.Intn(mapWidth-w-2), Y: 1 + rng.Intn(mapHeight-h-2), W: w, H: h}
	carveRoom(tiles, startRoom)
	start := center(startRoom)
	rooms := []room{startRoom}
	if style == layoutChambers || style == layoutHall {
		for attempts := 0; attempts < 100 && len(rooms) < 2+depth/2; attempts++ {
			candidate := room{X: 1 + rng.Intn(mapWidth-7), Y: 1 + rng.Intn(mapHeight-6), W: 3 + rng.Intn(4), H: 3 + rng.Intn(3)}
			if candidate.X+candidate.W >= mapWidth-1 || candidate.Y+candidate.H >= mapHeight-1 {
				continue
			}
			overlaps := false
			for _, existing := range rooms {
				if roomsOverlap(existing, candidate, 1) {
					overlaps = true
					break
				}
			}
			if overlaps {
				continue
			}
			trial := cloneTiles(tiles)
			carveRoom(trial, candidate)
			connectRooms(trial, center(rooms[rng.Intn(len(rooms))]), center(candidate), rng)
			if countFloorTiles(trial) > budget {
				continue
			}
			tiles = trial
			rooms = append(rooms, candidate)
		}
		// Extra connections make some floors offer alternate approaches.
		for i := 0; i < len(rooms)/2; i++ {
			trial := cloneTiles(tiles)
			connectRooms(trial, center(rooms[rng.Intn(len(rooms))]), center(rooms[rng.Intn(len(rooms))]), rng)
			if countFloorTiles(trial) <= budget {
				tiles = trial
			}
		}
	}
	growStyledFloor(tiles, budget, rng, style, nil)
	if style == layoutHall {
		protected := map[pos]bool{}
		for attempts, pillars := 0, 0; attempts < 80 && pillars < 1+depth/3; attempts++ {
			p := pos{2 + rng.Intn(mapWidth-4), 2 + rng.Intn(mapHeight-4)}
			if p == start || tiles[p.Y][p.X] != '.' || openCardinalNeighbors(tiles, p) != 4 {
				continue
			}
			tiles[p.Y][p.X] = '#'
			if len(tileDistances(tiles, start)) != budget-1 {
				tiles[p.Y][p.X] = '.'
				continue
			}
			pillars++
			protected[p] = true
			growStyledFloor(tiles, budget, rng, style, protected)
		}
	}
	return tiles, start
}

func blankFloor() [][]rune {
	tiles := make([][]rune, mapHeight)
	for y := range tiles {
		tiles[y] = make([]rune, mapWidth)
		for x := range tiles[y] {
			tiles[y][x] = '#'
		}
	}
	return tiles
}

var cardinalDirections = []pos{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

func openCardinalNeighbors(tiles [][]rune, p pos) int {
	count := 0
	for _, d := range cardinalDirections {
		if tiles[p.Y+d.Y][p.X+d.X] == '.' {
			count++
		}
	}
	return count
}

func growStyledFloor(tiles [][]rune, budget int, rng *simpleRNG, style layoutKind, protected map[pos]bool) {
	for count := countFloorTiles(tiles); count < budget; count++ {
		type candidate struct {
			p      pos
			weight int
		}
		candidates := []candidate{}
		total := 0
		for y := 1; y < mapHeight-1; y++ {
			for x := 1; x < mapWidth-1; x++ {
				p := pos{x, y}
				if tiles[y][x] != '#' || protected[p] {
					continue
				}
				neighbors := openCardinalNeighbors(tiles, p)
				if neighbors == 0 {
					continue
				}
				weight := neighbors * neighbors
				if style == layoutCrypt {
					weight = 1
					if neighbors == 1 {
						weight = 12
					}
				}
				if style == layoutCaves {
					weight *= neighbors * neighbors
				}
				candidates = append(candidates, candidate{p, weight})
				total += weight
			}
		}
		pick := rng.Intn(total)
		for _, c := range candidates {
			pick -= c.weight
			if pick < 0 {
				tiles[c.p.Y][c.p.X] = '.'
				break
			}
		}
	}
}

// A compact, connected fallback leaves ample wall space for planned chambers.
func compactFloorPlan(depth int, mirror bool) ([][]rune, pos) {
	tiles := blankFloor()
	width := 20
	if floorBudget(depth) <= 80 {
		width = 8
	}
	for i := 0; i < floorBudget(depth); i++ {
		x, y := 2+i%width, 1+i/width
		if mirror {
			x = mapWidth - 1 - x
		}
		tiles[y][x] = '.'
	}
	start := pos{2 + width/2, 2}
	if mirror {
		start.X = mapWidth - 1 - start.X
	}
	return tiles, start
}

// Cardinal connectivity is stricter than the game's eight-direction movement.
func tileDistances(tiles [][]rune, start pos) map[pos]int {
	distances := map[pos]int{}
	if start.X < 0 || start.X >= mapWidth || start.Y < 0 || start.Y >= mapHeight || tiles[start.Y][start.X] == '#' {
		return distances
	}
	distances[start] = 0
	queue := []pos{start}
	for head := 0; head < len(queue); head++ {
		p := queue[head]
		for _, d := range cardinalDirections {
			next := pos{p.X + d.X, p.Y + d.Y}
			if next.X < 1 || next.X >= mapWidth-1 || next.Y < 1 || next.Y >= mapHeight-1 || tiles[next.Y][next.X] == '#' {
				continue
			}
			if _, seen := distances[next]; seen {
				continue
			}
			distances[next] = distances[p] + 1
			queue = append(queue, next)
		}
	}
	return distances
}

package main

import "fmt"

const cryptName = "This Lich's crypt"
const cryptDepth = 7 // Equipment and total monster difficulty match floor eight.

type lichCrypt struct {
	ParentFloor      int         `json:"parent_floor"`
	Entrance         pos         `json:"entrance"`
	ReturnDoor       pos         `json:"return_door"`
	EntranceRevealed bool        `json:"entrance_revealed,omitempty"`
	ReturnRevealed   bool        `json:"return_revealed,omitempty"`
	Exit             pos         `json:"exit"`
	ExitRevealed     bool        `json:"exit_revealed,omitempty"`
	Completed        bool        `json:"completed,omitempty"`
	Floor            *level      `json:"floor"`
	Traps            []cryptTrap `json:"traps"`
}

func (g *game) addLichCrypt() error {
	type entrancePair struct {
		floor   int
		in, out pos
	}
	var candidates []entrancePair
	for depth := 3; depth <= 6; depth++ {
		lvl := g.levels[depth]
		reserved := lvl.secretFootprint(true)
		for _, hidden := range lvl.secretRooms() {
			// One search must not compete with an ordinary secret-room door.
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					reserved[pos{hidden.Door.X + dx, hidden.Door.Y + dy}] = true
				}
			}
		}
		for _, event := range lvl.Events {
			if event.Shortcut != nil {
				reserved[*event.Shortcut] = true
			}
		}
		var doors []pos
		for y := 1; y < mapHeight-1; y++ {
			for x := 1; x < mapWidth-1; x++ {
				p := pos{x, y}
				if lvl.Tiles[y][x] != '#' || reserved[p] {
					continue
				}
				open := 0
				for _, d := range cardinalDirections {
					n := pos{x + d.X, y + d.Y}
					if lvl.Tiles[n.Y][n.X] == '.' && !reserved[n] && g.featureTileFree(lvl, n) &&
						(lvl.Objective == nil || !pointInRoom(n, lvl.Objective.Area)) {
						open++
					}
				}
				if open == 1 {
					doors = append(doors, p)
				}
			}
		}
		for _, entrance := range doors {
			for _, exit := range doors {
				if distance(entrance, exit) >= 6 {
					candidates = append(candidates, entrancePair{depth, entrance, exit})
				}
			}
		}
	}
	if len(candidates) == 0 {
		return fmt.Errorf("seed %d: no safe crypt doors on floors 4–7", g.seed)
	}
	choice := candidates[g.rng.Intn(len(candidates))]
	lvl, exit, traps, err := g.makeCryptFloor()
	if err != nil {
		return err
	}
	g.crypt = &lichCrypt{ParentFloor: choice.floor, Entrance: choice.in, ReturnDoor: choice.out, Floor: lvl, Exit: exit, Traps: traps}
	return nil
}

// Two full-height mazes meet at one tile. Neither rooms nor diagonal movement
// can bypass the Under Lich's crossing through the central dividing wall.
func (g *game) makeCryptFloor() (*level, pos, []cryptTrap, error) {
	const middle = mapWidth/2 - 1
	tiles := blankFloor()
	for _, bounds := range [][2]int{{1, middle - 1}, {middle + 1, mapWidth - 2}} {
		g.carveCryptMaze(tiles, bounds[0], bounds[1])
		for n := 0; n < 7; n++ {
			w, h := 2+g.rng.Intn(2), 2+g.rng.Intn(2)
			x := bounds[0] + g.rng.Intn(bounds[1]-bounds[0]-w+2)
			y := 1 + g.rng.Intn(mapHeight-h-1)
			carveRoom(tiles, room{x, y, w, h})
		}
	}
	gate := pos{middle, mapHeight / 2}
	for _, p := range []pos{{middle - 1, gate.Y}, gate, {middle + 1, gate.Y}} {
		tiles[p.Y][p.X] = '.'
	}
	// Fill the even-width outer strip with small alcoves along the right edge.
	for y := 1; y < mapHeight-1; y++ {
		if tiles[y][mapWidth-3] == '.' {
			tiles[y][mapWidth-2] = '.'
		}
	}
	for x := 1; x < mapWidth-1; x++ {
		if x != middle && tiles[mapHeight-3][x] == '.' {
			tiles[mapHeight-2][x] = '.'
		}
	}
	distances := tileDistances(tiles, gate)
	start, far := pos{1, 1}, pos{mapWidth - 2, 1}
	for y := 1; y < mapHeight-1; y++ {
		if p := (pos{1, y}); distances[p] > distances[start] {
			start = p
		}
		if p := (pos{mapWidth - 2, y}); distances[p] > distances[far] {
			far = p
		}
	}
	lvl := &level{Index: cryptDepth, Tiles: tiles, Start: start, Layout: layoutKind("crypt maze"), Theme: floorTheme("sealed burial chambers")}
	roster, err := g.cryptRoster()
	if err != nil {
		return nil, pos{}, nil, err
	}
	occupied := map[pos]bool{start: true, far: true, gate: true}
	for i, m := range roster {
		m.Pos = gate
		if i > 0 {
			spots := cryptOpenSpots(lvl, occupied, func(p pos) bool { return distance(p, start) >= 5 && distance(p, gate) >= 2 })
			if len(spots) == 0 {
				return nil, pos{}, nil, fmt.Errorf("crypt has no monster position")
			}
			m.Pos = spots[g.rng.Intn(len(spots))]
		}
		occupied[m.Pos] = true
		copy := m
		lvl.Monsters = append(lvl.Monsters, &copy)
	}
	// Use ordinary floor-eight equipment and supplies, without duplicating a story chapter.
	for _, it := range g.itemsForLevel(cryptDepth) {
		if it.Kind == itemStoryScroll {
			continue
		}
		spots := cryptOpenSpots(lvl, occupied, nil)
		it.Pos = spots[g.rng.Intn(len(spots))]
		occupied[it.Pos] = true
		lvl.Items = append(lvl.Items, it)
	}
	// Offer equipment on the entrance side before the mandatory guardian.
	for i := range lvl.Items {
		if lvl.Items[i].Kind != itemWeapon && lvl.Items[i].Kind != itemArmor {
			continue
		}
		delete(occupied, lvl.Items[i].Pos)
		spots := cryptOpenSpots(lvl, occupied, func(p pos) bool { return p.X < middle && distance(p, start) >= 2 })
		lvl.Items[i].Pos = spots[g.rng.Intn(len(spots))]
		occupied[lvl.Items[i].Pos] = true
	}
	lvl.DifficultyBudget = rosterDifficulty(roster)
	lvl.InitialDifficulty = lvl.DifficultyBudget
	traps := g.placeCryptTraps(lvl, occupied, gate)
	return lvl, pos{mapWidth - 1, far.Y}, traps, nil
}

func (g *game) carveCryptMaze(tiles [][]rune, left, right int) {
	start := pos{left, 1}
	tiles[start.Y][start.X] = '.'
	stack := []pos{start}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		var options []pos
		for _, d := range cardinalDirections {
			n := pos{p.X + 2*d.X, p.Y + 2*d.Y}
			if n.X >= left && n.X <= right && n.Y < mapHeight-1 && n.Y >= 1 && tiles[n.Y][n.X] == '#' {
				options = append(options, n)
			}
		}
		if len(options) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		n := options[g.rng.Intn(len(options))]
		tiles[(p.Y+n.Y)/2][(p.X+n.X)/2] = '.'
		tiles[n.Y][n.X] = '.'
		stack = append(stack, n)
	}
}

func cryptOpenSpots(lvl *level, occupied map[pos]bool, allowed func(pos) bool) []pos {
	var spots []pos
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{x, y}
			if lvl.Tiles[y][x] != '#' && !occupied[p] && (allowed == nil || allowed(p)) {
				spots = append(spots, p)
			}
		}
	}
	return spots
}

func (g *game) cryptRoster() ([]monster, error) {
	guardian := newMonster(monsterUnderLich)
	var pool []monster
	seen := map[monsterKind]bool{}
	for _, m := range monsterPoolForLevel(cryptDepth) {
		if m.Undead && !m.Boss && !seen[m.Kind] {
			seen[m.Kind] = true
			pool = append(pool, scaleMonsterForDepth(m, cryptDepth))
		}
	}
	for i := len(pool) - 1; i > 0; i-- {
		j := g.rng.Intn(i + 1)
		pool[i], pool[j] = pool[j], pool[i]
	}
	low, high := floorDifficultyBounds(cryptDepth)
	choices := make([][]monster, high+1)
	choices[monsterDifficulty(guardian)] = []monster{guardian}
	for _, m := range pool {
		for copy := 0; copy < 2; copy++ {
			score := monsterDifficulty(m)
			for total := high; total >= score; total-- {
				previous := choices[total-score]
				if choices[total] != nil || previous == nil || len(previous) >= 9 {
					continue
				}
				choices[total] = append(append([]monster(nil), previous...), m)
			}
		}
	}
	base := floorDifficultyBaselines[cryptDepth]
	for delta := 0; delta <= high-low; delta++ {
		for _, total := range []int{base - delta, base + delta} {
			if total >= low && total <= high && choices[total] != nil {
				return choices[total], nil
			}
		}
	}
	return nil, fmt.Errorf("could not construct an undead crypt roster in floor-eight difficulty bounds")
}

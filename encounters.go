package main

type encounterKind string

const (
	encounterScatter  encounterKind = "scattered patrols"
	encounterPacks    encounterKind = "two small packs"
	encounterTreasure encounterKind = "guarded treasure"
	encounterEscort   encounterKind = "guard and spellcaster"
)

var encounterKinds = []encounterKind{encounterScatter, encounterPacks, encounterTreasure, encounterEscort}

func (g *game) arrangeEncounters(lvl *level) {
	lvl.Encounter = encounterKinds[g.rng.Intn(len(encounterKinds))]
	ranged, melee := false, false
	for _, m := range lvl.Monsters {
		if m.Boss {
			continue
		}
		if rangedMonster(m.Kind) {
			ranged = true
		} else {
			melee = true
		}
	}
	if lvl.Encounter == encounterEscort && (!ranged || !melee) {
		lvl.Encounter = encounterScatter
	}
	if lvl.Special != nil && lvl.Special.Kind == roomArmory {
		lvl.Encounter = encounterTreasure
	}
	escort := map[*monster]bool{}
	if lvl.Encounter == encounterEscort {
		guards := 0
		caster := false
		for _, m := range lvl.Monsters {
			if m.Boss {
				continue
			}
			if rangedMonster(m.Kind) {
				if !caster {
					escort[m] = true
					caster = true
				}
			} else if guards < 2 {
				escort[m] = true
				guards++
			}
		}
	}
	occupied := map[pos]bool{lvl.Start: true}
	for _, it := range lvl.Items {
		occupied[it.Pos] = true
	}
	for _, f := range lvl.Fountains {
		occupied[f.Pos] = true
	}
	for _, e := range lvl.Events {
		occupied[e.Pos] = true
	}
	if lvl.HasStair {
		occupied[lvl.Stairs] = true
	}
	if lvl.DogChain != nil {
		occupied[*lvl.DogChain] = true
	}
	for _, m := range lvl.Monsters {
		if m.Boss {
			occupied[m.Pos] = true
		}
	}
	anchor := g.encounterPosition(lvl, occupied, nil)
	second := g.encounterPosition(lvl, occupied, nil)
	if lvl.Encounter == encounterPacks {
		for tries := 0; tries < 20 && distance(anchor, second) < 5; tries++ {
			second = g.encounterPosition(lvl, occupied, nil)
		}
	}
	if lvl.Encounter == encounterTreasure {
		if lvl.Special != nil {
			anchor = lvl.Special.Pos
		} else {
			for _, it := range lvl.Items {
				if lvl.Tiles[it.Pos.Y][it.Pos.X] == '.' && distance(it.Pos, lvl.Start) >= 4 {
					anchor = it.Pos
					break
				}
			}
		}
	}
	for i, m := range lvl.Monsters {
		if m.Boss {
			continue
		}
		var near *pos
		groupLimit := 3
		if lvl.Encounter == encounterPacks {
			groupLimit = 6
		}
		grouped := i < groupLimit
		if lvl.Encounter == encounterEscort {
			grouped = escort[m]
		}
		if lvl.Encounter != encounterScatter && grouped {
			target := anchor
			near = &target
			if lvl.Encounter == encounterPacks && i%2 == 1 {
				target = second
			}
			if lvl.Encounter == encounterEscort && rangedMonster(m.Kind) {
				target = pos{anchor.X + sign(anchor.X-lvl.Start.X), anchor.Y + sign(anchor.Y-lvl.Start.Y)}
			}
		}
		m.Pos = g.encounterPosition(lvl, occupied, near)
		occupied[m.Pos] = true
	}
}

func (g *game) encounterPosition(lvl *level, occupied map[pos]bool, near *pos) pos {
	spots := []pos{}
	best := mapWidth + mapHeight
	for clearance := 3; clearance >= 0; clearance-- {
		for y := 1; y < mapHeight-1; y++ {
			for x := 1; x < mapWidth-1; x++ {
				p := pos{x, y}
				if lvl.Tiles[y][x] == '#' || occupied[p] || distance(p, lvl.Start) < clearance {
					continue
				}
				if near != nil {
					d := distance(p, *near)
					if d < best {
						best = d
						spots = spots[:0]
					}
					if d != best {
						continue
					}
				}
				spots = append(spots, p)
			}
		}
		if len(spots) > 0 {
			return spots[g.rng.Intn(len(spots))]
		}
	}
	return lvl.Start // Only reachable for malformed or completely occupied maps.
}

func rangedMonster(kind monsterKind) bool {
	switch kind {
	case monsterCultist, monsterLanternWisp, monsterGloomArcher, monsterHexPriest, monsterBellRevenant, monsterCinderDrake, monsterVoidSeer, monsterSoulLeech, monsterStormHerald:
		return true
	}
	return false
}

package main

type specialRoomKind string

const (
	roomArmory specialRoomKind = "guarded armory"
	roomShrine specialRoomKind = "healing shrine"
	roomKennel specialRoomKind = "abandoned kennel"
	roomVault  specialRoomKind = "treasure vault"
)

type specialRoom struct {
	Kind    specialRoomKind `json:"kind"`
	Area    room            `json:"area"`
	Pos     pos             `json:"pos"`
	Entered bool            `json:"entered,omitempty"`
}

func (g *game) featureTileFree(lvl *level, p pos) bool {
	if !g.inBounds(p) || lvl.Tiles[p.Y][p.X] == '#' || p == lvl.Start || (lvl.HasStair && p == lvl.Stairs) || (lvl.HasUpStair && p == lvl.UpStairs) || (lvl.DogChain != nil && p == *lvl.DogChain) {
		return false
	}
	return g.itemAt(lvl, p) == nil && g.monsterAt(lvl, p) == nil && g.fountainAt(lvl, p) == nil && lvl.eventAt(p) == nil
}

func (g *game) addSpecialRoom(lvl *level) {
	if g.rng.Intn(100) >= 55 {
		return
	}
	candidates := []room{}
	for y := 1; y < mapHeight-4; y++ {
		for x := 1; x < mapWidth-4; x++ {
			r := room{X: x, Y: y, W: 3, H: 3}
			if distance(center(r), lvl.Start) < 4 || !g.featureTileFree(lvl, center(r)) {
				continue
			}
			open := true
			for yy := y; yy < y+3; yy++ {
				for xx := x; xx < x+3; xx++ {
					if lvl.Tiles[yy][xx] == '#' {
						open = false
					}
				}
			}
			if open {
				candidates = append(candidates, r)
			}
		}
	}
	if len(candidates) == 0 {
		return
	}
	kinds := []specialRoomKind{roomArmory, roomShrine, roomKennel, roomVault}
	r := candidates[g.rng.Intn(len(candidates))]
	lvl.Special = &specialRoom{Kind: kinds[g.rng.Intn(len(kinds))], Area: r, Pos: center(r)}
	if lvl.Special.Kind == roomShrine {
		lvl.Events = append(lvl.Events, floorEvent{Kind: eventShrine, Pos: lvl.Special.Pos})
		return
	}
	// Rooms concentrate existing loot; they do not add stronger equipment.
	for i, it := range lvl.Items {
		if lvl.Tiles[it.Pos.Y][it.Pos.X] == '#' {
			continue
		}
		matches := false
		switch lvl.Special.Kind {
		case roomArmory:
			matches = it.Kind == itemWeapon || it.Kind == itemArmor
		case roomKennel:
			matches = it.Kind == itemPotion
		case roomVault:
			matches = it.Kind == itemBlinkStone || it.Kind == itemFireScroll || it.Kind == itemWardingCharm || it.Kind == itemSunOrb || it.Kind == itemStarfireOrb
		}
		if matches {
			lvl.Items[i].Pos = lvl.Special.Pos
			return
		}
	}
	if lvl.Special.Kind == roomVault {
		for i, it := range lvl.Items {
			if lvl.Tiles[it.Pos.Y][it.Pos.X] == '.' && (it.Kind == itemWeapon || it.Kind == itemArmor) {
				lvl.Items[i].Pos = lvl.Special.Pos
				return
			}
		}
	}
}

func (g *game) describeSpecialRoom() {
	r := g.current().Special
	if r == nil || r.Entered {
		return
	}
	p := g.player.Pos
	if p.X < r.Area.X || p.X >= r.Area.X+r.Area.W || p.Y < r.Area.Y || p.Y >= r.Area.Y+r.Area.H {
		return
	}
	r.Entered = true
	switch r.Kind {
	case roomArmory:
		g.addMessage("You enter a guarded armory. Old weapon racks line the stone walls.")
	case roomShrine:
		g.addMessage("A healing shrine glows in this chamber. Stand nearby and press y to receive its blessing.")
	case roomKennel:
		g.addMessage("You find an abandoned kennel. A worn dog blanket rests beside a forgotten supply cache.")
	case roomVault:
		g.addMessage("You enter a treasure vault. Its broken chest still holds something useful.")
	}
}

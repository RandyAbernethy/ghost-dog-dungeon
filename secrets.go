package main

func secretBounds(door, dir pos, w, h int) (int, int, int, int, bool) {
	var x0, y0, x1, y1 int
	switch {
	case dir.X == 1:
		x0, x1 = door.X+1, door.X+w
		y0, y1 = door.Y-h/2, door.Y-h/2+h-1
	case dir.X == -1:
		x0, x1 = door.X-w, door.X-1
		y0, y1 = door.Y-h/2, door.Y-h/2+h-1
	case dir.Y == 1:
		y0, y1 = door.Y+1, door.Y+h
		x0, x1 = door.X-w/2, door.X-w/2+w-1
	case dir.Y == -1:
		y0, y1 = door.Y-h, door.Y-1
		x0, x1 = door.X-w/2, door.X-w/2+w-1
	default:
		return 0, 0, 0, 0, false
	}
	if x0 < 1 || y0 < 1 || x1 >= mapWidth-1 || y1 >= mapHeight-1 {
		return 0, 0, 0, 0, false
	}
	return x0, y0, x1, y1, true
}

func (g *game) addSecretTreasure(lvl *level) {
	for _, hidden := range lvl.secretRooms() {
		g.addRoomTreasure(lvl, hidden)
	}
	g.addSecretSideStory(lvl)
	g.addSecretFountain(lvl)
}

func (g *game) addRoomTreasure(lvl *level, hidden *secretRoom) {
	tiles := hidden.floorTiles()
	if len(tiles) == 0 {
		return
	}
	count := 1
	if g.rng.Intn(100) < 20 {
		count++
	}
	used := map[pos]bool{}
	pool := g.secretTreasurePool(lvl.Index)
	// Secret equipment is unique within a run. Consumables can still repeat,
	// but a searched room should not hand back a weapon or armor already found.
	pool = g.filterFoundSecretEquipment(pool)
	for i := 0; i < count && len(pool) > 0; i++ {
		idx := g.rng.Intn(len(pool))
		it := pool[idx]
		pool = append(pool[:idx], pool[idx+1:]...)
		for tries := 0; tries < 60; tries++ {
			p := tiles[g.rng.Intn(len(tiles))]
			if used[p] || g.itemAt(lvl, p) != nil || g.fountainAt(lvl, p) != nil || (lvl.HasEscapeStair && p == lvl.EscapeStairs) {
				continue
			}
			it.Pos = p
			used[p] = true
			lvl.Items = append(lvl.Items, it)
			break
		}
	}
}

func (g *game) filterFoundSecretEquipment(pool []item) []item {
	found := map[string]bool{
		g.player.Weapon.Name:    true,
		g.player.HeadArmor.Name: true,
		g.player.BodyArmor.Name: true,
		g.player.LegArmor.Name:  true,
	}
	for _, carried := range g.player.Weapons {
		found[carried.Name] = true
	}
	for _, carried := range g.player.Armors {
		found[carried.Name] = true
	}
	for _, lvl := range g.levels {
		if lvl == nil {
			continue
		}
		for _, existing := range lvl.Items {
			if existing.Kind == itemWeapon || existing.Kind == itemArmor {
				found[existing.Name] = true
			}
		}
	}
	filtered := make([]item, 0, len(pool))
	for _, candidate := range pool {
		if (candidate.Kind == itemWeapon || candidate.Kind == itemArmor) && found[candidate.Name] {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func (g *game) secretTreasurePool(depth int) []item {
	pool := []item{
		{Kind: itemPotion, Name: "healing potion", Glyph: '!'},
		{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'},
		{Kind: itemBlinkStone, Name: "blink stone", Glyph: '*'},
		{Kind: itemFrostCharm, Name: "frost charm", Glyph: '*'},
	}
	if depth >= 4 {
		pool = append(pool,
			item{Kind: itemSunOrb, Name: "sun orb", Glyph: '*'},
			item{Kind: itemStarfireOrb, Name: "starfire orb", Glyph: '*'},
			item{Kind: itemPhoenixAsh, Name: "phoenix ash", Glyph: '*'},
			makeWeaponLoot(weapon{Name: "Emberbrand", Min: 11, Max: 16, Magic: true}),
			makeWeaponLoot(weapon{Name: "Tempest Spear", Min: 9, Max: 13, Magic: true, Reach: 2}),
			makeWeaponLoot(weapon{Name: "Voidglass Dagger", Min: 13, Max: 18, Magic: true}),
			makeWeaponLoot(weapon{Name: "Lichbane Greatsword", Min: 14, Max: 19, Magic: true}),
			makeWeaponLoot(weapon{Name: "Gravetide Maul", Min: 16, Max: 22, Magic: true}),
			makeArmorItem(armor{Name: "Aegis of Echoes", Slot: slotBody, Defense: 6, Rarity: rarityLegendary}),
			makeArmorItem(armor{Name: "Crown of the Hollow Star", Slot: slotHead, Defense: 6, Rarity: rarityLegendary}),
			makeArmorItem(armor{Name: "Wraithstep Greaves", Slot: slotLeg, Defense: 6, Rarity: rarityLegendary}),
		)
	}
	if depth >= 7 {
		pool = append(pool,
			makeWeaponLoot(weapon{Name: "Star-Eater Blade", Min: 18, Max: 24, Magic: true}),
			makeArmorItem(armor{Name: "Voidheart Plate", Slot: slotBody, Defense: 7, Rarity: rarityLegendary}),
		)
	}
	return pool
}

func (g *game) addEscapeSanctum(lvl *level) {
	if g.placeSecretRoom(lvl, 3+g.rng.Intn(3), 3+g.rng.Intn(2)) || g.placeSecretRoom(lvl, 3, 3) {
		lvl.HasEscapeStair = true
		lvl.EscapeStairs = lvl.Secret.Tiles[g.rng.Intn(len(lvl.Secret.Tiles))]
	}
}

func (g *game) bossGuardPosition(lvl *level, occupied map[pos]bool) (pos, bool) {
	if lvl.Secret == nil {
		return pos{}, false
	}
	best := pos{}
	bestDist := -1
	for _, d := range []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		p := pos{X: lvl.Secret.Door.X + d.X, Y: lvl.Secret.Door.Y + d.Y}
		if !g.inBounds(p) || lvl.Tiles[p.Y][p.X] != '.' || occupied[p] {
			continue
		}
		if dist := distance(p, lvl.Start); dist > bestDist {
			best = p
			bestDist = dist
		}
	}
	if bestDist < 0 {
		return pos{}, false
	}
	return best, true
}

func (g *game) addSecretFountain(lvl *level) {
	for _, hidden := range lvl.secretRooms() {
		g.addRoomFountain(lvl, hidden)
	}
}

func (g *game) addRoomFountain(lvl *level, hidden *secretRoom) {
	tiles := hidden.floorTiles()
	for _, f := range lvl.Fountains {
		for _, p := range tiles {
			if f.Pos == p {
				return
			}
		}
	}
	for _, p := range tiles {
		if g.itemAt(lvl, p) == nil && g.monsterAt(lvl, p) == nil && (!lvl.HasEscapeStair || p != lvl.EscapeStairs) {
			lvl.Fountains = append(lvl.Fountains, fountain{Pos: p})
			return
		}
	}
}

// Check every possible entrance so a required room cannot be lost to unlucky
// placement attempts. Its tiles stay walls until the player discovers it.
func (g *game) placeSecretRoom(lvl *level, w, h int) bool {
	var candidates []*secretRoom
	reserved := lvl.secretFootprint(true)
	dirs := []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			if lvl.Tiles[y][x] != '.' {
				continue
			}
			for _, dir := range dirs {
				door := pos{X: x + dir.X, Y: y + dir.Y}
				if lvl.Tiles[door.Y][door.X] != '#' || reserved[door] || (lvl.Index == levelCount-1 && lvl.Secret == nil && distance(door, lvl.Start) < 11) {
					continue
				}
				x0, y0, x1, y1, ok := secretBounds(door, dir, w, h)
				if !ok {
					continue
				}
				tiles := make([]pos, 0, w*h)
				for yy := y0; yy <= y1 && ok; yy++ {
					for xx := x0; xx <= x1; xx++ {
						if lvl.Tiles[yy][xx] != '#' || reserved[pos{xx, yy}] {
							ok = false
							break
						}
						tiles = append(tiles, pos{X: xx, Y: yy})
					}
				}
				if ok {
					candidates = append(candidates, &secretRoom{Door: door, Tiles: tiles})
				}
			}
		}
	}
	if len(candidates) == 0 {
		return false
	}
	hidden := candidates[g.rng.Intn(len(candidates))]
	if lvl.Secret == nil {
		lvl.Secret = hidden
	} else {
		lvl.ExtraSecrets = append(lvl.ExtraSecrets, hidden)
	}
	return true
}

func (g *game) addNestedSecretRoom(lvl *level) {
	size := 6 + g.rng.Intn(2)
	if !g.placeSecretRoom(lvl, size, size) {
		return
	}
	outer := lvl.Secret
	corner := outer.Tiles[0]
	center := pos{X: corner.X + size/2, Y: corner.Y + size/2}
	last := 2
	if size == 6 {
		last = 1
	}
	doors := []pos{
		{X: center.X - 2, Y: center.Y}, {X: center.X + last, Y: center.Y},
		{X: center.X, Y: center.Y - 2}, {X: center.X, Y: center.Y + last},
	}
	inner := &secretRoom{Door: doors[g.rng.Intn(len(doors))], SideStoryID: levelCount + 1}
	for y := center.Y - 2; y <= center.Y+last; y++ {
		for x := center.X - 2; x <= center.X+last; x++ {
			p := pos{X: x, Y: y}
			if x == center.X-2 || x == center.X+last || y == center.Y-2 || y == center.Y+last {
				inner.Walls = append(inner.Walls, p)
			} else {
				inner.Tiles = append(inner.Tiles, p)
			}
		}
	}
	outer.Inner = inner
}

// Secret remains the primary root for compatibility with existing save files.
func (lvl *level) secretRooms() []*secretRoom {
	rooms := lvl.Secret.rooms()
	for _, root := range lvl.ExtraSecrets {
		rooms = append(rooms, root.rooms()...)
	}
	return rooms
}

func (lvl *level) secretFootprint(margin bool) map[pos]bool {
	blocked := map[pos]bool{}
	for _, hidden := range lvl.secretRooms() {
		points := append([]pos{hidden.Door}, hidden.Tiles...)
		points = append(points, hidden.Walls...)
		for _, p := range points {
			blocked[p] = true
			if margin {
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						blocked[pos{p.X + dx, p.Y + dy}] = true
					}
				}
			}
		}
	}
	return blocked
}

func (lvl *level) hiddenRoomNear(p pos) *secretRoom {
	roots := append([]*secretRoom{lvl.Secret}, lvl.ExtraSecrets...)
	for _, root := range roots {
		if hidden := root.nextHiddenRoom(); hidden != nil && distance(p, hidden.Door) <= 1 {
			return hidden
		}
	}
	return nil
}

// The outer rectangle contains the inner chamber, but opening the outer door
// must leave the inner chamber and its enclosing wall intact.
func (hidden *secretRoom) floorTiles() []pos {
	if hidden.Inner == nil {
		return hidden.Tiles
	}
	blocked := map[pos]bool{hidden.Inner.Door: true}
	for _, p := range hidden.Inner.Tiles {
		blocked[p] = true
	}
	for _, p := range hidden.Inner.Walls {
		blocked[p] = true
	}
	tiles := make([]pos, 0, len(hidden.Tiles))
	for _, p := range hidden.Tiles {
		if !blocked[p] {
			tiles = append(tiles, p)
		}
	}
	return tiles
}

func (hidden *secretRoom) rooms() []*secretRoom {
	if hidden == nil {
		return nil
	}
	return append([]*secretRoom{hidden}, hidden.Inner.rooms()...)
}

func (hidden *secretRoom) nextHiddenRoom() *secretRoom {
	for hidden != nil && hidden.Revealed {
		hidden = hidden.Inner
	}
	return hidden
}

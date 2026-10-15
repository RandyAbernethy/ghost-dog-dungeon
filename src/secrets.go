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
	g.addLinkedRelic(lvl)
}

func (g *game) addRoomTreasure(lvl *level, hidden *secretRoom) {
	tiles := hidden.Tiles
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
		lvl.EscapeStairs = lvl.primarySecret().Tiles[g.rng.Intn(len(lvl.primarySecret().Tiles))]
	}
}

func (g *game) bossGuardPosition(lvl *level, occupied map[pos]bool) (pos, bool) {
	if lvl.primarySecret() == nil {
		return pos{}, false
	}
	best := pos{}
	bestDist := -1
	for _, d := range []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		p := pos{X: lvl.primarySecret().Door.X + d.X, Y: lvl.primarySecret().Door.Y + d.Y}
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
	tiles := hidden.Tiles
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
				if lvl.Tiles[door.Y][door.X] != '#' || reserved[door] || (lvl.Index == levelCount-1 && lvl.primarySecret() == nil && distance(door, lvl.Start) < 11) {
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
	lvl.Secrets = append(lvl.Secrets, hidden)
	return true
}

func (g *game) addLinkedSecretRooms(lvl *level) {
	if !g.placeSecretRoom(lvl, 3+g.rng.Intn(2), 3+g.rng.Intn(2)) {
		return
	}
	parent := lvl.Secrets[len(lvl.Secrets)-1]
	width, height := 2+g.rng.Intn(2), 2+g.rng.Intn(2)
	reserved := lvl.secretFootprint(false)
	parentTiles := map[pos]bool{}
	for _, p := range parent.Tiles {
		parentTiles[p] = true
	}
	var candidates []*secretRoom
	for _, p := range parent.Tiles {
		for _, dir := range []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			door := pos{X: p.X + dir.X, Y: p.Y + dir.Y}
			x0, y0, x1, y1, ok := secretBounds(door, dir, width, height)
			if !ok {
				continue
			}
			next := &secretRoom{Door: door, SideStoryID: levelCount + 1}
			for y := y0; y <= y1; y++ {
				for x := x0; x <= x1; x++ {
					next.Tiles = append(next.Tiles, pos{X: x, Y: y})
				}
			}
			if g.linkedRoomFits(lvl, next, parentTiles, reserved) {
				candidates = append(candidates, next)
			}
		}
	}
	if len(candidates) == 0 {
		// Discard the incomplete pair so the bounded floor-generation retry can try another layout.
		lvl.Secrets = lvl.Secrets[:len(lvl.Secrets)-1]
		return
	}
	parent.Next = candidates[g.rng.Intn(len(candidates))]
}

func (g *game) linkedRoomFits(lvl *level, next *secretRoom, parentTiles, reserved map[pos]bool) bool {
	for _, p := range append([]pos{next.Door}, next.Tiles...) {
		if reserved[p] {
			return false
		}
		// A wall buffer prevents access from ordinary corridors or unrelated secrets.
		// Only the connecting door may touch the first chamber.
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				q := pos{X: p.X + dx, Y: p.Y + dy}
				if !g.inBounds(q) || lvl.Tiles[q.Y][q.X] != '#' || (reserved[q] && !parentTiles[q]) || (p != next.Door && parentTiles[q]) {
					return false
				}
			}
		}
	}
	return true
}

func (lvl *level) primarySecret() *secretRoom {
	if len(lvl.Secrets) == 0 {
		return nil
	}
	return lvl.Secrets[0]
}

func (lvl *level) secretRooms() []*secretRoom {
	var rooms []*secretRoom
	for _, root := range lvl.Secrets {
		rooms = append(rooms, root.rooms()...)
	}
	return rooms
}

func (lvl *level) secretFootprint(margin bool) map[pos]bool {
	blocked := map[pos]bool{}
	for _, hidden := range lvl.secretRooms() {
		points := append([]pos{hidden.Door}, hidden.Tiles...)
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
	roots := lvl.Secrets
	for _, root := range roots {
		if hidden := root.nextHiddenRoom(); hidden != nil && distance(p, hidden.Door) <= 1 {
			return hidden
		}
	}
	return nil
}

func (hidden *secretRoom) rooms() []*secretRoom {
	if hidden == nil {
		return nil
	}
	return append([]*secretRoom{hidden}, hidden.Next.rooms()...)
}

func (hidden *secretRoom) nextHiddenRoom() *secretRoom {
	for hidden != nil && hidden.Revealed {
		hidden = hidden.Next
	}
	return hidden
}

package main

// Check every possible entrance so a required room cannot be lost to unlucky
// placement attempts. Its tiles stay walls until the player discovers it.
func (g *game) placeSecretRoom(lvl *level, w, h int) bool {
	var candidates []*secretRoom
	dirs := []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			if lvl.Tiles[y][x] != '.' {
				continue
			}
			for _, dir := range dirs {
				door := pos{X: x + dir.X, Y: y + dir.Y}
				if lvl.Tiles[door.Y][door.X] != '#' {
					continue
				}
				x0, y0, x1, y1, ok := secretBounds(door, dir, w, h)
				if !ok {
					continue
				}
				tiles := make([]pos, 0, w*h)
				for yy := y0; yy <= y1 && ok; yy++ {
					for xx := x0; xx <= x1; xx++ {
						if lvl.Tiles[yy][xx] != '#' {
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
	lvl.Secret = candidates[g.rng.Intn(len(candidates))]
	return true
}

func (g *game) addNestedSecretRoom(lvl *level) {
	if !g.placeSecretRoom(lvl, 7, 7) {
		return
	}
	outer := lvl.Secret
	corner := outer.Tiles[0]
	center := pos{X: corner.X + 3, Y: corner.Y + 3}
	doors := []pos{
		{X: center.X - 2, Y: center.Y}, {X: center.X + 2, Y: center.Y},
		{X: center.X, Y: center.Y - 2}, {X: center.X, Y: center.Y + 2},
	}
	inner := &secretRoom{Door: doors[g.rng.Intn(len(doors))], SideStoryID: levelCount + 1}
	for y := center.Y - 2; y <= center.Y+2; y++ {
		for x := center.X - 2; x <= center.X+2; x++ {
			p := pos{X: x, Y: y}
			if x == center.X-2 || x == center.X+2 || y == center.Y-2 || y == center.Y+2 {
				inner.Walls = append(inner.Walls, p)
			} else {
				inner.Tiles = append(inner.Tiles, p)
			}
		}
	}
	outer.Inner = inner
	g.addSecretTreasure(lvl)
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

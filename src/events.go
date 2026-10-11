package main

type eventKind string

const (
	eventGhost    eventKind = "friendly ghost"
	eventKeepsake eventKind = "Ash's keepsake"
	eventLever    eventKind = "shortcut lever"
	eventShrine   eventKind = "healing shrine"
)

type floorEvent struct {
	Kind     eventKind `json:"kind"`
	Pos      pos       `json:"pos"`
	Used     bool      `json:"used,omitempty"`
	Shortcut *pos      `json:"shortcut,omitempty"`
	Gifts    []item    `json:"gifts,omitempty"`
}

func (e *floorEvent) glyph() rune {
	switch e.Kind {
	case eventGhost:
		return 'G'
	case eventKeepsake:
		return '&'
	case eventLever:
		return '+'
	case eventShrine:
		return '^'
	}
	return '.'
}

func (lvl *level) eventAt(p pos) *floorEvent {
	for i := range lvl.Events {
		if lvl.Events[i].Pos == p {
			return &lvl.Events[i]
		}
	}
	return nil
}

func (g *game) nearbyEvent() *floorEvent {
	for i := range g.current().Events {
		e := &g.current().Events[i]
		if !e.Used && distance(e.Pos, g.player.Pos) <= 1 && g.current().Tiles[e.Pos.Y][e.Pos.X] != '#' {
			return e
		}
	}
	return nil
}

func (g *game) addFloorEvent(lvl *level) {
	if g.rng.Intn(100) >= 25 {
		return
	}
	kinds := []eventKind{eventGhost, eventKeepsake, eventLever}
	kind := kinds[g.rng.Intn(len(kinds))]
	if kind == eventLever && g.addShortcutLever(lvl) {
		return
	}
	if kind == eventLever {
		kind = eventGhost
	}
	spots := []pos{}
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{x, y}
			if g.featureTileFree(lvl, p) && distance(p, lvl.Start) >= 3 {
				spots = append(spots, p)
			}
		}
	}
	if len(spots) > 0 {
		lvl.Events = append(lvl.Events, floorEvent{Kind: kind, Pos: spots[g.rng.Intn(len(spots))]})
	}
}

func (g *game) addShortcutLever(lvl *level) bool {
	reserved := lvl.secretFootprint(true)
	candidates := []pos{}
	for y := 2; y < mapHeight-2; y++ {
		for x := 2; x < mapWidth-2; x++ {
			p := pos{x, y}
			if lvl.Tiles[y][x] == '#' && !reserved[p] {
				candidates = append(candidates, p)
			}
		}
	}
	attempts := min(60, len(candidates))
	for tries := 0; tries < attempts; tries++ {
		i := g.rng.Intn(len(candidates))
		cut := candidates[i]
		candidates[i] = candidates[len(candidates)-1]
		candidates = candidates[:len(candidates)-1]
		for _, d := range []pos{{1, 0}, {0, 1}} {
			a, b := pos{cut.X - d.X, cut.Y - d.Y}, pos{cut.X + d.X, cut.Y + d.Y}
			if lvl.Tiles[a.Y][a.X] != '.' || lvl.Tiles[b.Y][b.X] != '.' || tileDistances(lvl.Tiles, a)[b] < 6 {
				continue
			}
			for _, lever := range []pos{a, b} {
				if g.featureTileFree(lvl, lever) {
					shortcut := cut
					lvl.Events = append(lvl.Events, floorEvent{Kind: eventLever, Pos: lever, Shortcut: &shortcut})
					return true
				}
			}
		}
	}
	return false
}

func (g *game) interactEvent() bool {
	e := g.nearbyEvent()
	if e == nil {
		g.addMessage("There is nothing nearby to interact with.")
		return false
	}
	switch e.Kind {
	case eventGhost:
		g.describeGhostGifts(e)
		return false
	case eventShrine:
		if g.player.HP == g.player.MaxHP && (!g.dog.Freed || !g.dog.Alive || g.dog.HP == g.dog.MaxHP) {
			g.addMessage("The shrine waits until its blessing can heal you or Ghost Dog.")
			return false
		}
		g.player.HP = min(g.player.MaxHP, g.player.HP+6)
		if g.dog.Freed && g.dog.Alive {
			g.dog.HP = min(g.dog.MaxHP, g.dog.HP+10)
		}
		g.addMessage("The shrine's gentle light restores your strength and comforts your living companion.")
	case eventKeepsake:
		if !g.dog.Freed || !g.dog.Alive || distance(g.dog.Pos, e.Pos) > 2 {
			g.addMessage("A faded collar lies here. Ghost Dog may remember it when he is with you.")
			return false
		}
		g.dog.HP = min(g.dog.MaxHP, g.dog.HP+5)
		g.player.WardingCharms++
		g.addMessage("Ash recognizes the old collar and brightens. Beneath it you find one warding charm.")
	case eventLever:
		if e.Shortcut == nil {
			return false
		}
		g.current().Tiles[e.Shortcut.Y][e.Shortcut.X] = '.'
		g.addMessage("You pull the lever. A stone panel slides aside, opening a shortcut.")
	default:
		return false
	}
	e.Used = true
	return true
}

func (g *game) describeNearbyObjects(previousLevel int, previousPos pos) {
	lvl := g.current()
	arriving := func(p pos) bool {
		return lvl.Tiles[p.Y][p.X] != '#' && distance(g.player.Pos, p) <= 1 &&
			(previousLevel != g.currentLevel || distance(previousPos, p) > 1)
	}
	// Crossing an object's proximity boundary is enough to track visits, even after loading.
	for _, e := range lvl.Events {
		if e.Used || !arriving(e.Pos) {
			continue
		}
		switch e.Kind {
		case eventGhost:
			g.addMessage("A friendly ghost offers a choice of three gifts. Press y (interact) to hear its offer.")
		case eventShrine:
			g.addMessage("A healing shrine glows nearby. Its one-use blessing heals you and a living Ghost Dog; press y (interact) to receive it.")
		case eventKeepsake:
			g.addMessage("Ash's keepsake, a faded collar, lies nearby. Bring a living Ghost Dog close to find a warding charm and restore his health; press y (interact).")
		case eventLever:
			g.addMessage("A shortcut lever stands nearby. Press y (interact) to open a passage through a stone wall.")
		}
	}
	for _, f := range lvl.Fountains {
		if !f.Used && arriving(f.Pos) {
			g.addMessage("A healing fountain flows nearby. Press v to drink, healing you and a living Ghost Dog. It refills when you leave this floor.")
		}
	}
}

package main

import "fmt"

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
		g.addMessage("The friendly ghost offers one gift: 1. a healing potion, or 2. a blink stone. Type :choose 1 or :choose 2 and Enter (choose 1 or choose 2 in line-input mode).")
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

func (g *game) chooseGhostGift(choice byte) bool {
	e := g.nearbyEvent()
	if e == nil || e.Kind != eventGhost || (choice != 1 && choice != 2) {
		g.addMessage("A nearby friendly ghost must offer that gift.")
		return false
	}
	if choice == 1 {
		g.player.Potions++
		g.addMessage("The friendly ghost gives you one healing potion, then fades.")
	} else {
		g.player.BlinkStones++
		g.addMessage("The friendly ghost gives you one blink stone, then fades.")
	}
	e.Used = true
	return true
}

func (g *game) eventHint() string {
	if e := g.nearbyEvent(); e != nil {
		return fmt.Sprintf("%s nearby; press y to interact", e.Kind)
	}
	return ""
}

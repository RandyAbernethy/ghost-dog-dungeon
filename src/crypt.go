package main

import (
	"fmt"
	"io"
)

func (g *game) cryptDoorGlyph(p pos) rune {
	c := g.crypt
	if c == nil {
		return 0
	}
	if g.inCrypt {
		if c.ExitRevealed && p == c.Exit {
			return glyphCryptExit
		}
		return 0
	}
	if g.currentLevel != c.ParentFloor {
		return 0
	}
	if c.EntranceRevealed && p == c.Entrance {
		if c.Floor.Visited {
			return glyphSealedCrypt
		}
		return glyphCryptEntrance
	}
	if c.ReturnRevealed && p == c.ReturnDoor {
		return glyphCryptExit
	}
	return 0
}

func (g *game) useCryptDoor(next pos) (bool, bool) {
	c := g.crypt
	if c == nil {
		return false, false
	}
	if g.inCrypt {
		if next != c.Exit || !c.ExitRevealed {
			return false, false
		}
		g.inCrypt = false
		c.Completed, c.ReturnRevealed = true, true
		parent := g.levels[c.ParentFloor]
		parent.Tiles[c.ReturnDoor.Y][c.ReturnDoor.X] = '.'
		g.player.Pos = c.ReturnDoor
		g.dogFocus = nil
		if g.dog.active() {
			g.placeDogNearPlayer()
		}
		g.addMessage(fmt.Sprintf("You pass through the far secret door and return to floor %d. You survived %s.", c.ParentFloor+1, cryptName))
		return true, true
	}
	if g.currentLevel != c.ParentFloor || next != c.Entrance || !c.EntranceRevealed {
		return false, false
	}
	if c.Floor.Visited {
		g.addMessage("The crypt's one-way entrance is permanently sealed.")
		return true, false
	}
	parent := g.levels[c.ParentFloor]
	parent.Tiles[c.Entrance.Y][c.Entrance.X] = '#'
	for i := range parent.Fountains {
		parent.Fountains[i].Used = false
	}
	g.inCrypt, c.Floor.Visited = true, true
	g.player.Pos = c.Floor.Start
	g.player.ShieldTurns = 0
	g.dogFocus = nil
	if g.dog.active() {
		g.placeDogNearPlayer()
	}
	g.addMessage("You enter " + cryptName + ". The door seals behind you. Only the far secret exit leads back.")
	g.addMessage("Darkness limits sight to three tiles; existing wards fade.")
	return true, true
}

func (g *game) searchCrypt() bool {
	c := g.crypt
	if c == nil {
		return false
	}
	if !g.inCrypt {
		if g.currentLevel != c.ParentFloor || c.EntranceRevealed || distance(g.player.Pos, c.Entrance) > 1 {
			return false
		}
		c.EntranceRevealed = true
		g.current().Tiles[c.Entrance.Y][c.Entrance.X] = '.'
		g.addMessage("You uncover a secret door to " + cryptName + ". Step through { to enter; the entrance seals behind you. You must cross the maze to return. Undead of floor-eight strength and the Under Lich guard the route.")
		return true
	}
	found := false
	if !c.ExitRevealed && distance(g.player.Pos, c.Exit) <= 1 {
		c.ExitRevealed = true
		c.Floor.Tiles[c.Exit.Y][c.Exit.X] = '.'
		g.addMessage("Your search opens the far secret door. Step through } to return to the dungeon.")
		found = true
	}
	for i := range c.Traps {
		t := &c.Traps[i]
		if !t.Discovered && !t.Spent && distance(g.player.Pos, t.Pos) <= 1 && g.rng.Intn(2) == 0 {
			t.Discovered = true
			g.addMessage("You discover a " + string(t.Kind) + " trap " + relativeTo(g.player.Pos, t.Pos) + ". Press y from an adjacent tile to disarm it.")
			found = true
		}
	}
	if !found {
		g.addMessage("You search the crypt's stone, but uncover nothing new.")
	}
	return true
}

func (g *game) cryptHint() string {
	c := g.crypt
	if c == nil {
		return ""
	}
	if g.inCrypt {
		if c.ExitRevealed && distance(g.player.Pos, c.Exit) <= 1 {
			return "the far secret door } leads back to the dungeon. Step through to leave the crypt."
		}
		for _, t := range c.Traps {
			if t.Discovered && !t.Spent && distance(g.player.Pos, t.Pos) <= 1 {
				return "a " + string(t.Kind) + " trap is nearby — press y beside it to disarm; stepping on it still triggers it."
			}
		}
		return "The entrance behind you is sealed."
	}
	if g.currentLevel == c.ParentFloor && !c.Floor.Visited && distance(g.player.Pos, c.Entrance) <= 1 {
		if c.EntranceRevealed {
			return "the crypt door { is a one-way entrance. The entrance seals behind you; only the far secret exit leads back."
		}
	}
	return ""
}

// Crypt sight includes the first wall on a ray, but reveals nothing behind it.
func (g *game) canSee(p pos) bool {
	if !g.inCrypt {
		return true
	}
	from := g.player.Pos
	if !g.inBounds(p) || distance(from, p) > 3 {
		return false
	}
	dx, dy := abs(p.X-from.X), -abs(p.Y-from.Y)
	sx, sy := sign(p.X-from.X), sign(p.Y-from.Y)
	err := dx + dy
	for {
		if from == p {
			return true
		}
		if g.current().Tiles[from.Y][from.X] == '#' {
			return false
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			from.X += sx
		}
		if e2 <= dx {
			err += dx
			from.Y += sy
		}
	}
}

func (g *game) renderCryptFeatures(w io.Writer) {
	if !g.inCrypt {
		return
	}
	for _, t := range g.crypt.Traps {
		if t.Discovered && !t.Spent && g.canSee(t.Pos) {
			uiText(w, fmt.Sprintf("Trap: %s (%c), %s. Press y beside it to disarm.", t.Kind, t.glyph(), relativeTo(g.player.Pos, t.Pos)))
		}
	}
}

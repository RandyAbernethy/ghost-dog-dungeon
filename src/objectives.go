package main

import (
	"fmt"
	"io"
)

type objectiveKind string

const (
	objectiveSpirit objectiveKind = "The Vigil of the Lost Spirit"
	objectiveSigils objectiveKind = "The Broken Ritual"
)

type ritualSigil struct {
	Pos    pos  `json:"pos"`
	Broken bool `json:"broken,omitempty"`
}

type objectiveRoom struct {
	Kind        objectiveKind `json:"kind"`
	Area        room          `json:"area"`
	Pos         pos           `json:"pos"`
	Sigils      []ritualSigil `json:"sigils,omitempty"`
	Started     bool          `json:"started,omitempty"`
	StartedTurn int           `json:"started_turn,omitempty"`
	Progress    int           `json:"progress,omitempty"`
	Integrity   int           `json:"integrity,omitempty"`
	Completed   bool          `json:"completed,omitempty"`
	Failed      bool          `json:"failed,omitempty"`
	Reward      item          `json:"reward"`
}

func (o *objectiveRoom) marker(p pos) rune {
	if o == nil || o.Completed || o.Failed {
		return 0
	}
	if o.Kind == objectiveSpirit && o.Pos == p {
		return 'S'
	}
	for _, s := range o.Sigils {
		if s.Pos == p && !s.Broken {
			return 'O'
		}
	}
	return 0
}

func (o *objectiveRoom) instructions() string {
	if o.Kind == objectiveSpirit {
		return "A lost spirit waits here. Press y beside S to begin a three-turn vigil. Keep yourself or living Ash within three tiles; adjacent enemies erode its three points of resolve. Leaving the floor abandons it."
	}
	return "Three ritual sigils glow here. Stand on each O and press y to break it, spending one turn per sigil. Break all three to earn a reward."
}

// Add at most one optional objective, using existing open terrain and a reward
// chosen now. Loading and interacting never roll rooms or rewards.
func (g *game) addObjectiveRoom() {
	if g.rng.Intn(100) >= 35 {
		return
	}
	type candidate struct {
		floor int
		area  room
	}
	var candidates []candidate
	for depth := 2; depth < 8 && depth < len(g.levels); depth++ {
		lvl := g.levels[depth]
		if lvl.Objective != nil {
			return
		}
		secret := lvl.secretFootprint(true)
		for y := 2; y < mapHeight-4; y++ {
			for x := 2; x < mapWidth-4; x++ {
				r := room{X: x, Y: y, W: 3, H: 3}
				if distance(center(r), lvl.Start) < 5 {
					continue
				}
				clear := true
				for yy := y; yy < y+3; yy++ {
					for xx := x; xx < x+3; xx++ {
						p := pos{xx, yy}
						if secret[p] || !g.featureTileFree(lvl, p) ||
							(lvl.Special != nil && pointInRoom(p, lvl.Special.Area)) {
							clear = false
						}
					}
				}
				if clear {
					candidates = append(candidates, candidate{depth, r})
				}
			}
		}
	}
	if len(candidates) == 0 {
		return
	}
	c := candidates[g.rng.Intn(len(candidates))]
	o := &objectiveRoom{Kind: objectiveSpirit, Area: c.area, Pos: center(c.area), Integrity: 3}
	if g.rng.Intn(2) == 1 {
		o.Kind = objectiveSigils
		o.Sigils = []ritualSigil{{Pos: pos{c.area.X, c.area.Y}}, {Pos: pos{c.area.X + 2, c.area.Y}}, {Pos: pos{c.area.X + 1, c.area.Y + 2}}}
	}
	pool := []item{
		{Kind: itemPotion, Name: "healing potion", Glyph: '!'},
		{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'},
		{Kind: itemFrostCharm, Name: "frost charm", Glyph: '*'},
	}
	o.Reward = pool[g.rng.Intn(len(pool))]
	g.levels[c.floor].Objective = o
}

func pointInRoom(p pos, r room) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

func (g *game) interactObjective() bool {
	o := g.current().Objective
	if o == nil || o.Completed || o.Failed {
		return false
	}
	if o.Kind == objectiveSpirit && distance(g.player.Pos, o.Pos) <= 1 {
		if o.Started {
			g.addMessage(fmt.Sprintf("The vigil is underway: %d/3 turns, resolve %d/3. Stay close and protect the spirit.", o.Progress, o.Integrity))
			return false
		}
		o.Started, o.StartedTurn = true, g.stats.Turns+1
		g.addMessage("You promise to protect the lost spirit. The three-turn vigil begins after this turn.")
		return true
	}
	if o.Kind == objectiveSigils {
		for i := range o.Sigils {
			s := &o.Sigils[i]
			if s.Pos == g.player.Pos && !s.Broken {
				s.Broken, o.Started = true, true
				o.Progress++
				g.addMessage(fmt.Sprintf("You break a ritual sigil (%d/3). Its light gutters out.", o.Progress))
				if o.Progress == len(o.Sigils) {
					g.completeObjective(o)
				}
				return true
			}
		}
	}
	if distance(g.player.Pos, o.Pos) <= 3 {
		g.addMessage(o.instructions())
	}
	return false
}

func (g *game) advanceObjective() {
	if g.won || g.player.HP <= 0 {
		return
	}
	for depth, lvl := range g.levels {
		o := lvl.Objective
		if o == nil || o.Kind != objectiveSpirit || !o.Started || o.Completed || o.Failed || g.stats.Turns <= o.StartedTurn {
			continue
		}
		near := distance(g.player.Pos, o.Pos) <= 3 ||
			(g.dog.Freed && g.dog.Alive && distance(g.dog.Pos, o.Pos) <= 3)
		if g.inCrypt || depth != g.currentLevel || !near {
			o.Failed = true
			g.addMessage("The abandoned spirit fades. Its vigil has ended without a reward.")
			continue
		}
		for _, m := range lvl.Monsters {
			if m.HP > 0 && distance(m.Pos, o.Pos) <= 1 {
				o.Integrity--
				g.addMessage(fmt.Sprintf("Nearby enemies trouble the spirit. Resolve: %d/3.", o.Integrity))
				break
			}
		}
		if o.Integrity <= 0 {
			o.Failed = true
			g.addMessage("The spirit's light goes out. The vigil is lost.")
			continue
		}
		o.Progress++
		g.addMessage(fmt.Sprintf("You keep the spirit company (%d/3 turns).", o.Progress))
		if o.Progress == 3 {
			g.completeObjective(o)
		}
	}
}

func (g *game) completeObjective(o *objectiveRoom) {
	if o.Completed || o.Failed {
		return
	}
	o.Completed = true
	g.adventure.ObjectivesDone = append(g.adventure.ObjectivesDone, string(o.Kind))
	g.addMessage("Objective complete: " + string(o.Kind) + ". You receive " + o.Reward.Name + ".")
	g.collectItem(o.Reward)
}

func (g *game) renderObjective(w io.Writer) {
	o := g.current().Objective
	if o == nil || distance(g.player.Pos, o.Pos) > 6 {
		return
	}
	status := "optional"
	if o.Completed {
		status = "completed"
	} else if o.Failed {
		status = "failed"
	}
	fmt.Fprintf(w, "Objective: %s (%s), %d/3.\n", o.Kind, status, o.Progress)
	if !o.Completed && !o.Failed {
		fmt.Fprintln(w, o.instructions())
		if o.Kind == objectiveSpirit && o.Started {
			fmt.Fprintf(w, "Spirit resolve: %d/3.\n", o.Integrity)
		}
	}
}

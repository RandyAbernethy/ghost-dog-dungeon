package main

import "fmt"

type trapKind string

const (
	trapBlock     trapKind = "falling stone block"
	trapPit       trapKind = "collapsing pit"
	trapDarts     trapKind = "dart volley"
	trapFlames    trapKind = "flame jet"
	trapFrost     trapKind = "frost rune"
	trapBell      trapKind = "grave bell"
	trapGas       trapKind = "choking gas"
	trapRepulsion trapKind = "repulsion rune"
)

var cryptTrapKinds = []trapKind{trapBlock, trapPit, trapDarts, trapFlames, trapFrost, trapBell, trapGas, trapRepulsion}

type cryptTrap struct {
	Kind       trapKind `json:"kind"`
	Pos        pos      `json:"pos"`
	Damage     int      `json:"damage"`
	Discovered bool     `json:"discovered,omitempty"`
	Spent      bool     `json:"spent,omitempty"`
}

func (g *game) placeCryptTraps(lvl *level, occupied map[pos]bool, gate pos) []cryptTrap {
	kinds := append([]trapKind(nil), cryptTrapKinds...)
	for i := len(kinds) - 1; i > 0; i-- {
		j := g.rng.Intn(i + 1)
		kinds[i], kinds[j] = kinds[j], kinds[i]
	}
	count := 1 + g.rng.Intn(4)
	traps := make([]cryptTrap, 0, count)
	for _, kind := range kinds[:count] {
		spots := cryptOpenSpots(lvl, occupied, func(p pos) bool { return distance(p, lvl.Start) >= 3 && distance(p, gate) >= 2 })
		p := spots[g.rng.Intn(len(spots))]
		occupied[p] = true
		damage := g.randRange(6, 10)
		if kind == trapBlock || kind == trapFlames {
			damage += 4
		}
		traps = append(traps, cryptTrap{Kind: kind, Pos: p, Damage: damage})
	}
	return traps
}

func (g *game) trapAt(p pos) *cryptTrap {
	if !g.inCrypt {
		return nil
	}
	for i := range g.crypt.Traps {
		if g.crypt.Traps[i].Pos == p {
			return &g.crypt.Traps[i]
		}
	}
	return nil
}

func (g *game) disarmCryptTrap() bool {
	if !g.inCrypt {
		return false
	}
	for i := range g.crypt.Traps {
		t := &g.crypt.Traps[i]
		if t.Discovered && !t.Spent && distance(g.player.Pos, t.Pos) <= 1 {
			t.Spent = true
			g.addMessage("You safely disarm the " + string(t.Kind) + " trap.")
			return true
		}
	}
	return false
}

func (g *game) triggerCryptTrap(previous pos) {
	t := g.trapAt(g.player.Pos)
	if t == nil || t.Spent {
		return
	}
	t.Discovered, t.Spent = true, true
	switch t.Kind {
	case trapBlock:
		g.hurtPlayer(t.Damage, "A stone block crashes from the ceiling")
	case trapPit:
		g.hurtPlayer(t.Damage, "A pit opens beneath your feet")
		if g.player.HP > 0 {
			g.player.BoundTurns, g.player.Binding = 1, bindingPit
			g.addMessage("You struggle for a handhold on the pit's crumbling edge.")
		}
	case trapDarts:
		g.hurtPlayer(t.Damage, "Hidden slits fire a volley of darts")
	case trapFlames:
		g.hurtPlayerSpell(t.Damage, "A flame jet erupts from the crypt wall")
	case trapFrost:
		g.hurtPlayerSpell(t.Damage, "A frost rune bursts beneath you")
		if g.player.HP > 0 {
			g.bindPlayer(bindingIce)
		}
	case trapBell:
		g.hurtPlayerSpell(t.Damage, "A grave bell tolls inside your skull")
		if g.player.HP > 0 {
			g.hexPlayer()
		}
	case trapGas:
		g.player.PoisonTurns = max(g.player.PoisonTurns, 3)
		g.addMessage("Choking gas surrounds you. Your lungs begin to burn.")
	case trapRepulsion:
		g.hurtPlayerSpell(t.Damage, "A repulsion rune hurls you backward")
		if g.player.HP > 0 && g.isWalkable(previous, false) {
			g.player.Pos = previous
			g.addMessage("You stumble back onto the tile you came from.")
		}
	}
}

func (g *game) advancePoison() {
	if g.won || g.player.HP <= 0 || g.player.PoisonTurns <= 0 {
		return
	}
	g.player.PoisonTurns--
	// Inhaled poison is not a physical blow or a spell; armor cannot absorb it.
	g.player.HP = max(0, g.player.HP-2)
	g.addMessage(fmt.Sprintf("Crypt poison burns your lungs for 2 damage. (%d/%d)", g.player.HP, g.player.MaxHP))
}

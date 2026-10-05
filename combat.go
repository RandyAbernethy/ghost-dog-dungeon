package main

import (
	"fmt"
)

func (g *game) attackMonster(m *monster) {
	damage := g.randRange(g.player.Weapon.Min, g.player.Weapon.Max) + g.player.strikeBonus()
	if g.player.Weapon.Magic && m.Undead {
		damage += 2
		g.addMessage(fmt.Sprintf("%s flares against the undead.", g.player.Weapon.Name))
	}
	if g.player.HexedTurns > 0 {
		damage -= 3
		if damage < 1 {
			damage = 1
		}
		g.player.HexedTurns = 0
		g.addMessage("The hex weakens your strike.")
	}
	g.dogFocus = m
	g.addMessage(fmt.Sprintf("You strike %s with %s for %d damage.", m.Name, g.player.Weapon.Name, damage))
	g.damageMonster(m, damage, "hits")
	if m.HP > 0 && distance(g.player.Pos, m.Pos) <= 1 {
		switch m.Kind {
		case monsterMirrorShade:
			g.hurtPlayer(g.randRange(2, 4), "Mirror Shade reflects your blow")
		case monsterSlime:
			g.hurtPlayer(1, "Acid splashes onto you")
		}
	}
}

func (g *game) damageMonster(m *monster, damage int, verb string) {
	if m.HP <= 0 {
		return
	}
	if blocked := min(max(0, damage), monsterGuard(m.Kind)); blocked > 0 {
		g.addMessage(fmt.Sprintf("%s's armor blocks %d damage.", m.Name, blocked))
	}
	damage = max(0, damage-monsterGuard(m.Kind))
	g.knownMonsters[m.Kind] = true
	if g.turnDamage == nil {
		g.turnDamage = make(map[*monster]int)
	}
	dealt := min(max(damage, 0), m.HP)
	m.HP -= damage
	g.turnDamage[m] += dealt
	if m.HP > 0 {
		reaction, stuns := monsterDamageReaction(m.Name, verb, dealt, m.MaxHP)
		if stuns {
			m.StunnedTurns = max(m.StunnedTurns, 1)
		}
		g.addMessage(fmt.Sprintf("%s %s. (%d/%d)", reaction, hpState(verb), m.HP, m.MaxHP))
		return
	}
	m.HP = 0
	g.stats.MonstersKilled++
	g.addMessage(fmt.Sprintf("%s falls.", m.Name))
	lvl := g.current()
	if g.dogFocus == m {
		g.dogFocus = nil
	}
	if m.Boss {
		g.restoreGhostDogAfterLichDeath()
		g.unlockEscapeRoute()
		if m.StoryChapter > 0 {
			lvl.Items = append(lvl.Items, item{Pos: m.Pos, Kind: itemStoryScroll, Name: fmt.Sprintf("Ghost Dog story scroll %d", m.StoryChapter), Glyph: '?', StoryChapter: m.StoryChapter})
			g.addMessage("The Dread Lich drops the tenth Ghost Dog story scroll.")
		}
	}
	if g.dog.Freed && !g.dog.Alive && g.rng.Intn(100) < 10 {
		lvl.Items = append(lvl.Items, item{Pos: m.Pos, Kind: itemGhostRecallScroll, Name: "Ghost Dog recall scroll", Glyph: '?'})
		g.addMessage("A fallen foe leaves behind a Ghost Dog recall scroll.")
	}
}

func (g *game) unlockEscapeRoute() {
	lvl := g.current()
	if lvl.HasEscapeStair && lvl.Secret != nil && !lvl.Secret.Revealed {
		g.revealSecretRoom(lvl)
		g.addMessage("The Dread Lich collapses and hidden stone grinds open around a secret stairway out.")
		return
	}
	g.addMessage("The Dread Lich collapses and the way out is finally yours.")
}

func monsterDamageReaction(name, verb string, damage, maxHP int) (reaction string, stuns bool) {
	switch verb {
	case "burns":
		return name + " burns", false
	case "scorches":
		return name + " scorches", false
	case "sears":
		return name + " sears", false
	case "freezes":
		return name + " freezes", false
	default:
		percent := max(0, damage) * 100 / max(1, maxHP)
		switch {
		case percent < 20:
			return name + " barely flinches", false
		case percent < 40:
			return name + " recoils", false
		case percent < 60:
			return name + " staggers", false
		case percent < 80:
			return name + " reels in agony", false
		default:
			return name + " nearly collapses", true
		}
	}
}

func hpState(verb string) string {
	switch verb {
	case "burns", "scorches", "sears":
		return "in the blaze"
	case "freezes":
		return "under the frost"
	default:
		return "from the hit"
	}
}

func (g *game) advanceEnemies() {
	if g.won || g.player.HP <= 0 {
		return
	}
	if g.dog.Freed && g.dog.Alive {
		g.dogTurn()
	}
	if g.won {
		return
	}
	for _, m := range g.current().Monsters {
		if m.HP <= 0 {
			continue
		}
		g.monsterTurn(m)
		if g.player.HP <= 0 || g.won {
			return
		}
	}
	if g.player.ShieldTurns > 0 {
		g.player.ShieldTurns--
	}
}

func (g *game) monsterTurn(m *monster) {
	g.knownMonsters[m.Kind] = true
	if m.StunnedTurns > 0 {
		m.StunnedTurns--
		g.addMessage(fmt.Sprintf("%s is stunned and cannot act.", m.Name))
		return
	}
	if m.FrozenTurns > 0 {
		m.FrozenTurns--
		g.addMessage(fmt.Sprintf("%s shudders inside the ice.", m.Name))
		return
	}
	if m.FleeTurns > 0 {
		g.fleeMonster(m)
		m.FleeTurns--
		return
	}
	if g.turnDamage[m] > m.MaxHP/2 && g.rng.Intn(100) < 50 {
		m.FleeTurns = 1 + g.rng.Intn(4)
		g.addMessage(fmt.Sprintf("%s panics under the combined assault and flees for %d turn(s).", m.Name, m.FleeTurns))
		g.fleeMonster(m)
		m.FleeTurns--
		return
	}
	if m.Kind == monsterIronrootBrute {
		m.ActionTurns++
		if m.ActionTurns%2 == 0 {
			g.addMessage("The Ironroot Brute pauses to pull its roots from the stone.")
			return
		}
	}
	if g.dog.Freed && g.dog.Alive && distance(m.Pos, g.dog.Pos) == 1 && distance(m.Pos, g.player.Pos) > 1 {
		g.attackDog(m)
		return
	}
	if distance(m.Pos, g.player.Pos) == 1 {
		g.attackPlayer(m)
		return
	}
	if g.newMonsterPower(m) {
		return
	}
	switch m.Kind {
	case monsterBlinker:
		if distance(m.Pos, g.player.Pos) <= 4 && g.rng.Intn(100) < 35 {
			spots := g.openNeighbors(g.player.Pos, false)
			if len(spots) > 0 {
				m.Pos = spots[g.rng.Intn(len(spots))]
				g.addMessage("A Blink Stalker folds through space and appears beside you.")
				if distance(m.Pos, g.player.Pos) == 1 {
					g.attackPlayer(m)
				}
				return
			}
		}
	case monsterSpider:
		if distance(m.Pos, g.player.Pos) <= 4 && g.rng.Intn(100) < 20 {
			g.player.WebbedTurns = 1
			g.addMessage("A Web Spider spits silk across your legs.")
			return
		}
	case monsterHexPriest:
		if distance(m.Pos, g.player.Pos) <= 5 && g.rng.Intn(100) < 25 {
			g.player.HexedTurns = 1
			g.hurtPlayerSpell(2, "The Hex Priest whispers a crooked curse")
			return
		}
	case monsterBoneHound:
		if distance(m.Pos, g.player.Pos) <= 5 && g.rng.Intn(100) < 30 {
			g.moveMonster(m)
			if distance(m.Pos, g.player.Pos) == 1 {
				g.attackPlayer(m)
			} else {
				g.moveMonster(m)
			}
			return
		}
	case monsterCinderDrake:
		if distance(m.Pos, g.player.Pos) <= 4 && g.rng.Intn(100) < 30 {
			g.hurtPlayerSpell(g.randRange(3, 6), "The Cinder Drake breathes a cone of embers")
			return
		}
	case monsterVoidSeer:
		if distance(m.Pos, g.player.Pos) <= 5 && g.rng.Intn(100) < 28 {
			g.player.HexedTurns = 1
			g.hurtPlayerSpell(g.randRange(2, 5), "The Void Seer tears at your thoughts")
			return
		}
	case monsterSoulLeech:
		if distance(m.Pos, g.player.Pos) <= 4 && g.rng.Intn(100) < 30 {
			g.hurtPlayerSpell(g.randRange(3, 6), "The Soul Leech siphons you from a distance")
			m.HP = min(m.MaxHP, m.HP+2)
			g.addMessage("The Soul Leech drinks in the stolen vitality.")
			return
		}
	case monsterStormHerald:
		if distance(m.Pos, g.player.Pos) <= 4 && g.rng.Intn(100) < 30 {
			g.hurtPlayerSpell(g.randRange(3, 6), "The Storm Herald calls lightning down on you")
			if g.dog.Freed && g.dog.Alive && distance(m.Pos, g.dog.Pos) <= 4 {
				g.hurtDog(g.randRange(3, 6), "Lightning from the Storm Herald")
			}
			return
		}
	case monsterBoss:
		if distance(m.Pos, g.player.Pos) <= 5 && g.rng.Intn(100) < 30 {
			g.player.HexedTurns = 1
			g.hurtPlayerSpell(g.randRange(5, 8), "The Dread Lich hurls a bolt of grave-fire")
			return
		}
	}
	g.moveMonster(m)
}

func (g *game) fleeMonster(m *monster) {
	currentThreatDistance := distance(m.Pos, g.player.Pos)
	if g.dog.Freed && g.dog.Alive {
		currentThreatDistance = min(currentThreatDistance, distance(m.Pos, g.dog.Pos))
	}
	best := pos{}
	bestThreatDistance := currentThreatDistance
	found := false
	for _, p := range g.openNeighbors(m.Pos, false) {
		threatDistance := distance(p, g.player.Pos)
		if g.dog.Freed && g.dog.Alive {
			threatDistance = min(threatDistance, distance(p, g.dog.Pos))
		}
		if !found || threatDistance > bestThreatDistance {
			best, bestThreatDistance, found = p, threatDistance, true
		}
	}
	if found {
		m.Pos = best
		threats := "you"
		if g.dog.Freed && g.dog.Alive {
			threats += " and the Ghost Dog"
		}
		g.addMessage(fmt.Sprintf("%s scrambles away from %s.", m.Name, threats))
	}
}

func (g *game) moveMonster(m *monster) {
	target := g.player.Pos
	if g.dog.Freed && g.dog.Alive && distance(m.Pos, g.dog.Pos) < distance(m.Pos, target) && distance(m.Pos, g.dog.Pos) <= 5 {
		target = g.dog.Pos
	}
	g.moveActorToward(&m.Pos, target, false)
}

func (g *game) moveActorToward(from *pos, target pos, isDog bool) {
	candidates := []pos{{X: from.X + sign(target.X-from.X), Y: from.Y}, {X: from.X, Y: from.Y + sign(target.Y-from.Y)}}
	if target.X != from.X && target.Y != from.Y {
		candidates = append(candidates, pos{X: from.X + sign(target.X-from.X), Y: from.Y + sign(target.Y-from.Y)})
	}
	candidates = append(candidates, g.openNeighbors(*from, !isDog)...)
	seen := map[pos]bool{}
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		seen[p] = true
		if g.isWalkable(p, isDog) {
			*from = p
			return
		}
	}
}

func (g *game) attackPlayer(m *monster) {
	engaged := g.adjacentMonster(g.dog.Pos)
	if !g.dog.Freed || !g.dog.Alive || engaged == nil {
		g.dogFocus = m
	} else if g.dogFocus == nil || g.dogFocus.HP <= 0 || distance(g.dog.Pos, g.dogFocus.Pos) != 1 {
		g.dogFocus = engaged
	}
	damage := g.randRange(m.MinDamage, m.MaxDamage)
	if m.Boss {
		damage--
	} else if m.Kind != monsterOrc && m.Kind != monsterMirrorShade {
		damage--
	}
	if damage < 1 {
		damage = 1
	}
	source := m.Name + " hits you"
	switch m.Kind {
	case monsterWraith:
		m.HP = min(m.MaxHP, m.HP+2)
		source = "The Wraith drains your warmth"
	case monsterSoulLeech:
		m.HP = min(m.MaxHP, m.HP+3)
		source = "The Soul Leech drains your vitality"
	case monsterSpider:
		if g.rng.Intn(100) < 25 {
			g.player.WebbedTurns = 1
			source = "The Web Spider bites and tangles you"
		}
	case monsterHexPriest:
		if g.rng.Intn(100) < 35 {
			g.player.HexedTurns = 1
			source = "The Hex Priest strikes and brands you with a curse"
		}
	case monsterFrostHound:
		if g.rng.Intn(100) < 30 {
			g.player.WebbedTurns = 1
			source = "The Frostbound Hound bites and freezes your feet"
		}
	case monsterCrownlessKnight:
		if g.player.ShieldTurns > 0 {
			g.player.ShieldTurns = 0
			g.addMessage("The Crownless Knight shatters your warding charm.")
		}
	case monsterBoss:
		if g.rng.Intn(100) < 30 {
			g.player.HexedTurns = 1
			source = "The Dread Lich carves a rune of ruin into you"
		}
	}
	g.hurtPlayer(damage, source)
	if m.Kind == monsterSoulLeech {
		g.addMessage("The Soul Leech knits itself together with your stolen strength.")
	}
}

func (g *game) hurtPlayer(damage int, source string) {
	damage -= g.player.armorBlock()
	if g.player.ShieldTurns > 0 {
		damage -= 3
	}
	if damage < 0 {
		damage = 0
	}
	g.player.HP = max(0, g.player.HP-damage)
	g.addMessage(fmt.Sprintf("%s for %d damage. (%d/%d)", source, damage, g.player.HP, g.player.MaxHP))
}

func (g *game) attackDog(m *monster) {
	damage := g.randRange(max(1, m.MinDamage-1), max(1, m.MaxDamage-1))
	g.hurtDog(damage, m.Name+" claws")
}

func (g *game) hurtDog(damage int, source string) {
	if !g.dog.Alive {
		return
	}
	if g.player.ShieldTurns > 0 && g.dog.Freed && g.dog.Alive {
		damage = max(0, damage-3)
	}
	g.dog.HP -= damage
	g.addMessage(fmt.Sprintf("%s hits the ghost dog for %d damage. (%d/%d)", source, damage, max(g.dog.HP, 0), g.dog.MaxHP))
	if g.dog.HP <= 0 {
		g.dog.HP = 0
		g.dog.Alive = false
		g.stats.GhostDogFalls++
		g.dogFocus = nil
		g.addMessage("The Ghost Dog can only be freed now by a magic spell or by killing the Dread Lich")
	}
}

func (g *game) hurtPlayerSpell(damage int, source string) {
	g.hurtPlayer(max(0, damage-g.player.spellWard()), source)
}

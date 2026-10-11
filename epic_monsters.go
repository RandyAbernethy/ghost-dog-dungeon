package main

var epicMonsterKinds = [...]monsterKind{monsterIronColossus, monsterGraveTitan, monsterAshenWyrm, monsterHollowReaper}

func epicMonster(kind monsterKind) bool {
	for _, epic := range epicMonsterKinds {
		if kind == epic {
			return true
		}
	}
	return false
}

func epicMonsterSpec(kind monsterKind) (monster, bool) {
	var m monster
	switch kind {
	case monsterIronColossus:
		m = monster{Glyph: 'C', MaxHP: 220, MinDamage: 32, MaxDamage: 46}
	case monsterGraveTitan:
		m = monster{Glyph: 'G', MaxHP: 200, MinDamage: 30, MaxDamage: 42, Undead: true}
	case monsterAshenWyrm:
		m = monster{Glyph: 'A', MaxHP: 180, MinDamage: 28, MaxDamage: 40}
	case monsterHollowReaper:
		m = monster{Glyph: 'R', MaxHP: 190, MinDamage: 29, MaxDamage: 41, Undead: true}
	default:
		return monster{}, false
	}
	m.Kind, m.Name, m.HP = kind, string(kind), m.MaxHP
	return m, true
}

func epicMonsterDescription(kind monsterKind) string {
	trick := ""
	switch kind {
	case monsterIronColossus:
		trick = "Blocks 9 damage per hit and shatters warding charms before striking."
	case monsterGraveTitan:
		trick = "Undead; blocks 6 damage per hit and heals 8 health when its melee attack deals damage."
	case monsterAshenWyrm:
		trick = "Blocks 5 damage per hit; its melee attacks flare into fire that harms adjacent foes."
	case monsterHollowReaper:
		trick = "Undead; blocks 4 damage per hit and reflects 12 damage when struck at close range. Use reach."
	default:
		return ""
	}
	return "Epic: wakes within four tiles, acts once every three turns, and can be outrun. Very dangerous without God armor or an Artifact. " + trick
}

func (g *game) awakenEpic(m *monster) {
	if m.Awakened {
		return
	}
	m.Awakened = true
	g.addMessage(m.Name + " awakens! It acts once every three turns but hits with terrible force. Retreat or face it with God armor or an Artifact.")
}

func (g *game) epicRest(m *monster) bool {
	if !epicMonster(m.Kind) {
		return false
	}
	if !m.Awakened {
		if distance(m.Pos, g.player.Pos) > 4 {
			return true
		}
		g.awakenEpic(m)
	}
	m.ActionTurns++
	return (m.ActionTurns-1)%3 != 0
}

// Optional Epics are separate from ordinary budgets, never compulsory stair guards.
// Removing their spawn tile must leave every ordinary floor tile reachable.
func (g *game) addEpicEncounter(lvl *level) {
	if !relicDepth(lvl.Index) {
		return
	}
	for _, floor := range g.levels {
		if floor == nil {
			continue
		}
		for _, m := range floor.Monsters {
			if epicMonster(m.Kind) {
				return
			}
		}
	}
	if g.rng.Intn(100) >= 12 {
		return
	}
	var spots []pos
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{x, y}
			if !g.featureTileFree(lvl, p) || distance(p, lvl.Start) < 8 || (lvl.HasStair && distance(p, lvl.Stairs) < 6) {
				continue
			}
			bypass := cloneTiles(lvl.Tiles)
			bypass[y][x] = '#'
			if len(tileDistances(bypass, lvl.Start)) == countFloorTiles(lvl.Tiles)-1 {
				spots = append(spots, p)
			}
		}
	}
	if len(spots) == 0 {
		return
	}
	m := scaleMonsterForDepth(newMonster(epicMonsterKinds[g.rng.Intn(len(epicMonsterKinds))]), lvl.Index)
	m.Pos = spots[g.rng.Intn(len(spots))]
	lvl.Monsters = append(lvl.Monsters, &m)
}

func (g *game) epicMeleePower(m *monster, dealt int) {
	if !epicMonster(m.Kind) || dealt <= 0 {
		return
	}
	switch m.Kind {
	case monsterGraveTitan:
		m.HP = min(m.MaxHP, m.HP+8)
		g.addMessage("The Epic Grave Titan drinks in vitality and heals.")
	case monsterAshenWyrm:
		g.addMessage("The Epic Ashen Wyrm's strike erupts in scorching fire.")
		if g.player.HP > 0 && distance(m.Pos, g.player.Pos) == 1 {
			g.hurtPlayerSpell(12, "Scorching wyrm fire")
		}
		if g.dog.Freed && g.dog.Alive && distance(m.Pos, g.dog.Pos) == 1 {
			g.hurtDog(12, "Scorching wyrm fire")
		}
	}
}

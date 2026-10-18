package main

var epicMonsterKinds = [...]monsterKind{
	monsterIronColossus, monsterGraveTitan, monsterAshenWyrm, monsterHollowReaper,
	monsterFrostMonarch, monsterStormLeviathan, monsterDuskTyrant, monsterObsidianGolem,
}

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
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 220, MinDamage: 32, MaxDamage: 46}
	case monsterGraveTitan:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 200, MinDamage: 30, MaxDamage: 42, Undead: true}
	case monsterAshenWyrm:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 180, MinDamage: 28, MaxDamage: 40}
	case monsterHollowReaper:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 190, MinDamage: 29, MaxDamage: 41, Undead: true}
	case monsterFrostMonarch:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 185, MinDamage: 28, MaxDamage: 40, Undead: true}
	case monsterStormLeviathan:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 185, MinDamage: 28, MaxDamage: 40}
	case monsterDuskTyrant:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 205, MinDamage: 30, MaxDamage: 42, Undead: true}
	case monsterObsidianGolem:
		m = monster{Glyph: monsterGlyphs[kind], MaxHP: 230, MinDamage: 31, MaxDamage: 43}
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
	case monsterFrostMonarch:
		trick = "Undead; blocks 5 damage per hit. Damaging melee strikes freeze your feet when you are adjacent, costing one attempted move."
	case monsterStormLeviathan:
		trick = "Blocks 5 damage per hit. Damaging melee strikes arc 9 lightning damage into you and living Ghost Dog within two tiles. Spell wards protect you."
	case monsterDuskTyrant:
		trick = "Undead; blocks 6 damage per hit. Damaging melee strikes hex you when you are adjacent, weakening your next strike."
	case monsterObsidianGolem:
		trick = "Blocks 12 damage per hit with its obsidian shell. Heavy weapons or Worldsplitter can overcome it."
	default:
		return ""
	}
	return "Epic: wakes within four tiles, acts once every three turns, and can be outrun. Very dangerous without God armor or an Artifact. " + trick + " Prepares " + epicStrikeName(kind) + " during its last rest turn: leave the red X tiles before its next active turn. Ash dodges instead of attacking. Defeating it leaves one elite treasure."
}

func epicTreasurePool() []item {
	return append(relicPool(),
		item{Kind: itemStarfireOrb, Name: "starfire orb", Glyph: itemGlyphs[itemStarfireOrb]},
		item{Kind: itemSunOrb, Name: "sun orb", Glyph: itemGlyphs[itemSunOrb]},
		item{Kind: itemPhoenixAsh, Name: "phoenix ash", Glyph: itemGlyphs[itemPhoenixAsh]},
		item{Kind: itemGhostRecallScroll, Name: "Ghost Dog recall scroll", Glyph: itemGlyphs[itemGhostRecallScroll]},
	)
}

func (g *game) dropEpicTreasure(m *monster) {
	// Magic rewards remain available even if every relic has already been found.
	pool := g.filterFoundSecretEquipment(epicTreasurePool())
	treasure := pool[g.rng.Intn(len(pool))]
	treasure.Pos = m.Pos
	g.current().Items = append(g.current().Items, treasure)
	name := treasure.Name
	switch treasure.Kind {
	case itemWeapon:
		name = coloredWeaponName(treasure.Weapon)
	case itemArmor:
		name = coloredArmorName(treasure.Armor)
	}
	g.addMessage(m.Name + " drops an elite treasure: " + name + ".")
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
	if m.ActionTurns%3 == 0 && len(m.StrikeTiles) == 0 {
		g.prepareEpicStrike(m)
	}
	return (m.ActionTurns-1)%3 != 0
}

func (lvl *level) epicEncounter() *monster {
	for _, m := range lvl.Monsters {
		if epicMonster(m.Kind) {
			return m
		}
	}
	return nil
}

// Epics are separate from ordinary budgets, never compulsory stair guards.
// Removing their spawn tile must leave every ordinary floor tile reachable.
func (g *game) addEpicEncounter(lvl *level) bool {
	if !relicDepth(lvl.Index) {
		return false
	}
	if lvl.epicEncounter() != nil {
		return true
	}
	// A generated relic guarantees an encounter on its own floor, even if an
	// earlier floor already received the dungeon's random Epic encounter.
	if !lvl.hasRelic() {
		for _, floor := range g.levels {
			if floor != nil && floor.epicEncounter() != nil {
				return false
			}
		}
		if g.rng.Intn(100) >= 12 {
			return false
		}
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
		return false
	}
	m := scaleMonsterForDepth(newMonster(epicMonsterKinds[g.rng.Intn(len(epicMonsterKinds))]), lvl.Index)
	m.Pos = spots[g.rng.Intn(len(spots))]
	lvl.Monsters = append(lvl.Monsters, &m)
	return true
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
	case monsterFrostMonarch:
		if g.player.HP > 0 && distance(m.Pos, g.player.Pos) == 1 {
			g.addMessage("The Epic Frost Monarch's strike spreads ice around your feet.")
			g.bindPlayer(bindingIce)
		}
	case monsterStormLeviathan:
		g.addMessage("The Epic Storm Leviathan's strike erupts in forked lightning.")
		if g.player.HP > 0 && distance(m.Pos, g.player.Pos) <= 2 {
			g.hurtPlayerSpell(9, "Leviathan lightning")
		}
		if g.dog.Freed && g.dog.Alive && distance(m.Pos, g.dog.Pos) <= 2 {
			g.hurtDog(9, "Leviathan lightning")
		}
	case monsterDuskTyrant:
		if g.player.HP > 0 && distance(m.Pos, g.player.Pos) == 1 {
			g.addMessage("The Epic Dusk Tyrant's strike brands you with a hex.")
			g.hexPlayer()
		}
	}
}

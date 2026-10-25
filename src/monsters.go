package main

var allMonsterKinds = []monsterKind{
	monsterRat,
	monsterSkeleton,
	monsterGoblin,
	monsterOrc,
	monsterCultist,
	monsterSlime,
	monsterSpider,
	monsterWraith,
	monsterBlinker,
	monsterHexPriest,
	monsterMirrorShade,
	monsterBoneHound,
	monsterGargoyle,
	monsterGraveKnight,
	monsterCinderDrake,
	monsterVoidSeer,
	monsterSoulLeech,
	monsterStormHerald,
	monsterBoss,
	monsterUnderLich,
	monsterAshBeetle, monsterChainImp, monsterLanternWisp, monsterChapelSentinel,
	monsterGloomArcher, monsterBellRevenant, monsterIronrootBrute, monsterFrostHound,
	monsterRiftWeaver, monsterCrownlessKnight,
	monsterTunnelBat, monsterCaveCrawler, monsterRustGuard, monsterBarrowArcher,
	monsterGraveAcolyte, monsterEmberHound, monsterObsidianGuard, monsterHollowWarden,
	monsterIronColossus, monsterGraveTitan, monsterAshenWyrm, monsterHollowReaper,
	monsterFrostMonarch, monsterStormLeviathan, monsterDuskTyrant, monsterObsidianGolem,
}

func monsterPoolForLevel(depth int) []monster {
	levels := [][]monster{{
		newMonster(monsterRat), newMonster(monsterRat), newMonster(monsterGoblin), newMonster(monsterGoblin), newMonster(monsterSkeleton), newMonster(monsterSlime), newMonster(monsterCultist),
	}, {
		newMonster(monsterGoblin), newMonster(monsterOrc), newMonster(monsterOrc), newMonster(monsterCultist), newMonster(monsterSpider), newMonster(monsterSkeleton), newMonster(monsterSlime), newMonster(monsterBoneHound),
	}, {
		newMonster(monsterSkeleton), newMonster(monsterSlime), newMonster(monsterCultist), newMonster(monsterWraith), newMonster(monsterBlinker), newMonster(monsterHexPriest), newMonster(monsterSpider), newMonster(monsterBoneHound),
	}, {
		newMonster(monsterMirrorShade), newMonster(monsterBlinker), newMonster(monsterHexPriest), newMonster(monsterWraith), newMonster(monsterOrc), newMonster(monsterSpider), newMonster(monsterCultist), newMonster(monsterGargoyle), newMonster(monsterBoneHound),
	}, {
		newMonster(monsterMirrorShade), newMonster(monsterBlinker), newMonster(monsterHexPriest), newMonster(monsterWraith), newMonster(monsterSoulLeech), newMonster(monsterGargoyle), newMonster(monsterGargoyle), newMonster(monsterBoneHound), newMonster(monsterSkeleton),
	}, {
		newMonster(monsterMirrorShade), newMonster(monsterBlinker), newMonster(monsterHexPriest), newMonster(monsterWraith), newMonster(monsterSoulLeech), newMonster(monsterOrc), newMonster(monsterGargoyle), newMonster(monsterBoneHound), newMonster(monsterGraveKnight), newMonster(monsterCinderDrake),
	}, {
		newMonster(monsterMirrorShade), newMonster(monsterBlinker), newMonster(monsterHexPriest), newMonster(monsterWraith), newMonster(monsterStormHerald), newMonster(monsterGargoyle), newMonster(monsterBoneHound), newMonster(monsterGraveKnight), newMonster(monsterCinderDrake), newMonster(monsterVoidSeer),
	}, {
		newMonster(monsterMirrorShade), newMonster(monsterCinderDrake), newMonster(monsterVoidSeer), newMonster(monsterGraveKnight), newMonster(monsterSoulLeech), newMonster(monsterWraith), newMonster(monsterStormHerald), newMonster(monsterGargoyle), newMonster(monsterBoneHound),
	}, {
		newMonster(monsterMirrorShade), newMonster(monsterCinderDrake), newMonster(monsterCinderDrake), newMonster(monsterVoidSeer), newMonster(monsterVoidSeer), newMonster(monsterGraveKnight), newMonster(monsterStormHerald), newMonster(monsterBlinker), newMonster(monsterSoulLeech), newMonster(monsterBoneHound),
	}, {
		newMonster(monsterMirrorShade), newMonster(monsterCinderDrake), newMonster(monsterVoidSeer), newMonster(monsterGraveKnight), newMonster(monsterHexPriest), newMonster(monsterStormHerald), newMonster(monsterGargoyle), newMonster(monsterBoneHound), newMonster(monsterSoulLeech), newMonster(monsterBoss),
	}}
	pool := append(levels[depth], newMonster(newMonsterKindsByFloor[depth]))
	for _, extra := range additionalMonsters {
		if depth >= extra.FirstFloor && depth <= extra.FirstFloor+3 {
			pool = append(pool, newMonster(extra.Kind))
		}
	}
	return pool
}

func scaleMonsterForDepth(m monster, depth int) monster {
	m.MaxHP += depth * 2
	m.HP = m.MaxHP
	damageBonus := depth / 3
	m.MinDamage += damageBonus
	m.MaxDamage += damageBonus
	return m
}

func newMonster(kind monsterKind) monster {
	if spec, ok := newMonsterSpec(kind); ok {
		return spec
	}
	switch kind {
	case monsterRat:
		return monster{Kind: kind, Name: "Dungeon Rat", HP: 8, MaxHP: 8, MinDamage: 2, MaxDamage: 4}
	case monsterSkeleton:
		return monster{Kind: kind, Name: "Skeleton", HP: 11, MaxHP: 11, MinDamage: 3, MaxDamage: 5, Undead: true}
	case monsterGoblin:
		return monster{Kind: kind, Name: "Goblin Knifer", HP: 10, MaxHP: 10, MinDamage: 3, MaxDamage: 6}
	case monsterOrc:
		return monster{Kind: kind, Name: "Orc Brute", HP: 16, MaxHP: 16, MinDamage: 4, MaxDamage: 7}
	case monsterCultist:
		return monster{Kind: kind, Name: "Cultist", HP: 12, MaxHP: 12, MinDamage: 3, MaxDamage: 6}
	case monsterSlime:
		return monster{Kind: kind, Name: "Caustic Slime", HP: 14, MaxHP: 14, MinDamage: 3, MaxDamage: 5}
	case monsterSpider:
		return monster{Kind: kind, Name: "Web Spider", HP: 13, MaxHP: 13, MinDamage: 3, MaxDamage: 5}
	case monsterWraith:
		return monster{Kind: kind, Name: "Wraith", HP: 15, MaxHP: 15, MinDamage: 4, MaxDamage: 7, Undead: true}
	case monsterBlinker:
		return monster{Kind: kind, Name: "Blink Stalker", HP: 16, MaxHP: 16, MinDamage: 4, MaxDamage: 7}
	case monsterHexPriest:
		return monster{Kind: kind, Name: "Hex Priest", HP: 18, MaxHP: 18, MinDamage: 4, MaxDamage: 7}
	case monsterMirrorShade:
		return monster{Kind: kind, Name: "Mirror Shade", HP: 20, MaxHP: 20, MinDamage: 5, MaxDamage: 8, Undead: true}
	case monsterBoneHound:
		return monster{Kind: kind, Name: "Bone Hound", HP: 18, MaxHP: 18, MinDamage: 4, MaxDamage: 7, Undead: true}
	case monsterGargoyle:
		return monster{Kind: kind, Name: "Stone Gargoyle", HP: 24, MaxHP: 24, MinDamage: 5, MaxDamage: 8}
	case monsterGraveKnight:
		return monster{Kind: kind, Name: "Grave Knight", HP: 26, MaxHP: 26, MinDamage: 6, MaxDamage: 9, Undead: true}
	case monsterCinderDrake:
		return monster{Kind: kind, Name: "Cinder Drake", HP: 27, MaxHP: 27, MinDamage: 5, MaxDamage: 9}
	case monsterVoidSeer:
		return monster{Kind: kind, Name: "Void Seer", HP: 23, MaxHP: 23, MinDamage: 5, MaxDamage: 8, Undead: true}
	case monsterSoulLeech:
		return monster{Kind: kind, Name: "Soul Leech", HP: 25, MaxHP: 25, MinDamage: 5, MaxDamage: 8, Undead: true}
	case monsterStormHerald:
		return monster{Kind: kind, Name: "Storm Herald", HP: 28, MaxHP: 28, MinDamage: 6, MaxDamage: 9}
	case monsterBoss:
		return monster{Kind: kind, Name: "Dread Lich", HP: 60, MaxHP: 60, MinDamage: 7, MaxDamage: 11, Undead: true, Boss: true}
	default:
		panic("unknown monster kind")
	}
}

func monsterSummary(kind monsterKind) string {
	if description := newMonsterDescription(kind); description != "" {
		return description
	}
	switch kind {
	case monsterRat:
		return "Fast vermin that harass you early and force bad positioning."
	case monsterSkeleton:
		return "Basic undead guards that hit harder than they look."
	case monsterGoblin:
		return "Quick knife-fighters that pressure you in packs."
	case monsterOrc:
		return "Heavy bruisers with enough health to survive several hits."
	case monsterCultist:
		return "Fanatics who fill rooms and soften you up for tougher monsters."
	case monsterSlime:
		return "Acidic blobs that punish melee attacks with splash damage."
	case monsterSpider:
		return "Skittering hunters that can pin you in place with webbing."
	case monsterWraith:
		return "Undead drainers that heal when they land a strike."
	case monsterBlinker:
		return "Ambush predators that teleport to punish fragile positions."
	case monsterHexPriest:
		return "Curse casters that weaken your next attack and chip away at you."
	case monsterMirrorShade:
		return "Reflective undead that punish reckless melee bursts."
	case monsterBoneHound:
		return "Undead pack-hunters that sprint into bad positions faster than most foes."
	case monsterGargoyle:
		return "Stone predators that hold ground and brawl like living statues."
	case monsterGraveKnight:
		return "Armored undead champions that anchor the deepest dungeon packs."
	case monsterCinderDrake:
		return "Ash-scaled drakes that turn open rooms into dangerous fire lanes."
	case monsterVoidSeer:
		return "Undead prophets that unravel your focus from a distance."
	case monsterSoulLeech:
		return "Vampiric dead that siphon health from range and knit themselves back together."
	case monsterStormHerald:
		return "Lightning callers that can strike you and Ghost Dog in the same burst."
	case monsterBoss:
		return "The ruler of the dungeon, guarding the sealed hidden way out from the deepest floor."
	default:
		return "A hostile creature of the dungeon."
	}
}

func monsterPower(kind monsterKind) string {
	if description := newMonsterDescription(kind); description != "" {
		return description
	}
	switch kind {
	case monsterRat:
		return "No special power, but they swarm and waste your turns."
	case monsterSkeleton:
		return "Undead: vulnerable to sun orbs."
	case monsterGoblin:
		return "No magic, just efficient stabbing pressure."
	case monsterOrc:
		return "High durability and damage for a normal foe."
	case monsterCultist:
		return "No single trick, but they support deadlier packs by crowding space."
	case monsterSlime:
		return "Acid splash deals damage back when you hit it in melee."
	case monsterSpider:
		return "Can web you from range or on hit, costing you a movement turn."
	case monsterWraith:
		return "Steals warmth and heals when it lands a hit."
	case monsterBlinker:
		return "Can teleport beside you and attack immediately."
	case monsterHexPriest:
		return "Can curse you, weakening your next strike and dealing chip damage."
	case monsterMirrorShade:
		return "Reflects part of your melee damage back at you."
	case monsterBoneHound:
		return "Sometimes surges forward for an extra burst of movement or an immediate bite."
	case monsterGargoyle:
		return "No trick beyond raw stone durability and savage melee hits."
	case monsterGraveKnight:
		return "Undead: highly durable and vulnerable to sun orbs and magical weapons."
	case monsterCinderDrake:
		return "Can breathe embers from four tiles away before closing to melee."
	case monsterVoidSeer:
		return "Can hex you from five tiles away, weakening your next strike."
	case monsterSoulLeech:
		return "Drains health from up to four tiles away and heals itself with the stolen vitality."
	case monsterStormHerald:
		return "Sometimes calls a lightning burst that can hit you and Ghost Dog together."
	case monsterBoss:
		return "Launches grave-fire bolts and guards the hidden way out. Each active turn has a 10% chance to bind a living, freed Ghost Dog within five visible tiles in silver chains instead of attacking. Move onto his d tile to free him."
	default:
		return "Unknown."
	}
}

func (g *game) noteVisibleMonsters() {
	for _, m := range g.current().Monsters {
		if m.HP > 0 && g.canSee(m.Pos) {
			g.knownMonsters[m.Kind] = true
		}
	}
}

func (g *game) knownMonsterKinds() []monsterKind {
	var kinds []monsterKind
	for _, kind := range allMonsterKinds {
		if g.knownMonsters[kind] {
			kinds = append(kinds, kind)
		}
	}
	return kinds
}

var newMonsterKindsByFloor = [...]monsterKind{
	monsterAshBeetle, monsterChainImp, monsterLanternWisp, monsterChapelSentinel,
	monsterGloomArcher, monsterBellRevenant, monsterIronrootBrute, monsterFrostHound,
	monsterRiftWeaver, monsterCrownlessKnight,
}

func newMonsterSpec(kind monsterKind) (monster, bool) {
	var m monster
	switch kind {
	case monsterUnderLich:
		m = monster{MaxHP: 160, MinDamage: 11, MaxDamage: 17, Undead: true}
	case monsterAshBeetle:
		m = monster{MaxHP: 9, MinDamage: 1, MaxDamage: 3}
	case monsterChainImp:
		m = monster{MaxHP: 12, MinDamage: 2, MaxDamage: 4}
	case monsterLanternWisp:
		m = monster{MaxHP: 10, MinDamage: 2, MaxDamage: 4, Undead: true}
	case monsterChapelSentinel:
		m = monster{MaxHP: 18, MinDamage: 3, MaxDamage: 6}
	case monsterGloomArcher:
		m = monster{MaxHP: 17, MinDamage: 3, MaxDamage: 5}
	case monsterBellRevenant:
		m = monster{MaxHP: 20, MinDamage: 3, MaxDamage: 6, Undead: true}
	case monsterIronrootBrute:
		m = monster{MaxHP: 26, MinDamage: 4, MaxDamage: 7}
	case monsterFrostHound:
		m = monster{MaxHP: 23, MinDamage: 4, MaxDamage: 7, Undead: true}
	case monsterRiftWeaver:
		m = monster{MaxHP: 24, MinDamage: 4, MaxDamage: 7}
	case monsterCrownlessKnight:
		m = monster{MaxHP: 32, MinDamage: 5, MaxDamage: 8, Undead: true}
	default:
		if m, ok := additionalMonsterSpec(kind); ok {
			return m, true
		}
		return epicMonsterSpec(kind)
	}
	m.Kind, m.Name, m.HP = kind, string(kind), m.MaxHP
	return m, true
}

// These abilities run after frozen/fleeing, adjacent attacks, and dog targeting.
func (g *game) newMonsterPower(m *monster) bool {
	d := distance(m.Pos, g.player.Pos)
	switch m.Kind {
	case monsterChainImp:
		if d <= 3 && g.rng.Intn(100) < 25 {
			g.bindPlayer(bindingChains)
			g.addMessage("A Chain Imp hooks your ankles with a silver chain.")
			return true
		}
	case monsterLanternWisp:
		if d <= 3 && g.rng.Intn(100) < 30 {
			g.hurtPlayerSpell(g.randRange(2, 4), "The Lantern Wisp casts a pale spark")
			return true
		}
	case monsterGloomArcher, monsterBarrowArcher:
		if d <= 5 && g.rng.Intn(100) < 40 {
			g.hurtPlayer(g.randRange(m.MinDamage, m.MaxDamage), "The "+m.Name+" fires a barbed arrow")
			return true
		}
	case monsterBellRevenant, monsterGraveAcolyte:
		if d <= 4 && g.rng.Intn(100) < 30 {
			g.hexPlayer()
			g.hurtPlayerSpell(2, "The "+m.Name+" voices a hollow curse")
			return true
		}
	case monsterRiftWeaver:
		if d <= 4 && g.rng.Intn(100) < 30 {
			spots := g.openNeighbors(g.player.Pos, false)
			if len(spots) > 0 {
				m.Pos = spots[g.rng.Intn(len(spots))]
				g.bindPlayer(bindingRift)
				g.addMessage("A Rift Weaver steps through a tear in the air and binds your feet.")
				return true
			}
		}
	}
	return false
}

func monsterGuard(kind monsterKind) int {
	switch kind {
	case monsterUnderLich:
		return 6
	case monsterAshBeetle, monsterRustGuard:
		return 1
	case monsterChapelSentinel, monsterObsidianGuard:
		return 2
	case monsterIronrootBrute:
		return 3
	case monsterIronColossus:
		return 9
	case monsterGraveTitan:
		return 6
	case monsterAshenWyrm:
		return 5
	case monsterHollowReaper:
		return 4
	case monsterFrostMonarch, monsterStormLeviathan:
		return 5
	case monsterDuskTyrant:
		return 6
	case monsterObsidianGolem:
		return 12
	default:
		return 0
	}
}

func newMonsterDescription(kind monsterKind) string {
	switch kind {
	case monsterUnderLich:
		return "Undead crypt guardian: 160 health, 6 guard. Holds the maze's only crossing and cannot be driven from it. Acts every other turn; claws adjacent foes or casts grave-fire and a hex within three visible tiles. Killing it does not break the Dread Lich's curse."
	case monsterAshBeetle:
		return "Its ash-coated shell blocks 1 damage per hit."
	case monsterChainImp:
		return "Can chain your feet from three tiles away, costing a movement turn."
	case monsterLanternWisp:
		return "Undead; casts sparks from three tiles away. Spell wards reduce their damage."
	case monsterChapelSentinel:
		return "Its chapel shield blocks 2 damage per hit; heavy weapons overcome it."
	case monsterGloomArcher:
		return "Fires physical arrows from five tiles away; guard protects against them."
	case monsterBellRevenant:
		return "Undead; its distant bell deals spell damage and weakens your next strike."
	case monsterIronrootBrute:
		return "Blocks 3 damage per hit but rests every other turn; use reach and timing."
	case monsterFrostHound:
		return "Undead; its freezing bite can cost you a movement turn."
	case monsterRiftWeaver:
		return "Can teleport beside you and bind your feet; save a blink stone."
	case monsterCrownlessKnight:
		return "Undead; its melee strike shatters an active warding charm."
	default:
		if description := additionalMonsterDescription(kind); description != "" {
			return description
		}
		return epicMonsterDescription(kind)
	}
}

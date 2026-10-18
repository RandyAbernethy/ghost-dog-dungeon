package main

// Calibrated from 1,000 rosters per floor in the previous version, including themes.
var floorDifficultyBaselines = [...]int{26, 72, 123, 212, 294, 382, 494, 611, 718, 903}
var floorHealthBaselines = [...]int{10, 31, 55, 91, 134, 182, 239, 293, 350, 445}
var floorDamageBaselines = [...]int{7, 18, 27, 50, 66, 84, 114, 134, 153, 193}

func monsterDifficulty(m monster) int {
	bonus := 0
	switch m.Kind {
	case monsterUnderLich:
		bonus = 16 - (m.MinDamage + m.MaxDamage) // It attacks every other turn.
	case monsterSlime, monsterWraith, monsterHollowWarden, monsterSpider, monsterCaveCrawler:
		bonus = 6
	case monsterBlinker, monsterGloomArcher, monsterBarrowArcher, monsterCrownlessKnight:
		bonus = 7
	case monsterHexPriest, monsterCinderDrake, monsterBellRevenant, monsterGraveAcolyte:
		bonus = 8
	case monsterMirrorShade:
		bonus = 10
	case monsterBoneHound, monsterEmberHound, monsterChainImp, monsterLanternWisp, monsterFrostHound:
		bonus = 5
	case monsterVoidSeer, monsterRiftWeaver:
		bonus = 9
	case monsterSoulLeech, monsterStormHerald:
		bonus = 12
	case monsterBoss:
		bonus = 14
	case monsterIronrootBrute:
		// It attacks and moves only every other turn; its shell still matters.
		bonus = -(m.MinDamage + m.MaxDamage)
	case monsterIronColossus:
		bonus = 12
	case monsterGraveTitan:
		bonus = 20
	case monsterAshenWyrm:
		bonus = 15
	case monsterHollowReaper:
		bonus = 18
	case monsterFrostMonarch, monsterDuskTyrant:
		bonus = 20
	case monsterStormLeviathan:
		bonus = 18
	case monsterObsidianGolem:
		bonus = 12
	}
	score := m.MaxHP + 2*(m.MinDamage+m.MaxDamage) + 3*monsterGuard(m.Kind) + bonus
	if epicMonster(m.Kind) {
		score -= 4 * (m.MinDamage + m.MaxDamage) / 3
	}
	return max(1, score)
}

func rosterDifficulty(roster []monster) int {
	total := 0
	for _, m := range roster {
		total += monsterDifficulty(m)
	}
	return total
}

func floorDifficultyBounds(depth int) (int, int) {
	base := floorDifficultyBaselines[depth]
	radius := max(2, base*7/100)
	low, high := base-radius, base+radius
	if depth > 0 {
		low = max(low, (floorDifficultyBaselines[depth-1]+base)/2+1)
	}
	if depth < levelCount-1 {
		high = min(high, (base+floorDifficultyBaselines[depth+1])/2)
	}
	return low, high
}

func controllingMonster(kind monsterKind) bool {
	switch kind {
	case monsterSpider, monsterCaveCrawler, monsterHexPriest, monsterVoidSeer, monsterBoss,
		monsterChainImp, monsterBellRevenant, monsterGraveAcolyte, monsterFrostHound, monsterRiftWeaver:
		return true
	}
	return false
}

func fairFloorRoster(roster []monster, depth int) bool {
	health, damage, ranged, controls := 0, 0, 0, 0
	for _, m := range roster {
		health += m.MaxHP
		damage += m.MinDamage + m.MaxDamage
		if rangedMonster(m.Kind) {
			ranged++
		}
		if controllingMonster(m.Kind) {
			controls++
		}
	}
	hpRadius := max(2, floorHealthBaselines[depth]*10/100)
	damageRadius := max(3, floorDamageBaselines[depth]*12/100)
	return abs(health-floorHealthBaselines[depth]) <= hpRadius &&
		abs(damage-floorDamageBaselines[depth]) <= damageRadius &&
		ranged <= max(1, depth/2+1) && controls <= max(1, (depth+3)/3)
}

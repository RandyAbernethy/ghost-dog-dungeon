package main

// Reserve feature symbols before assigning monster, item, or trap symbols.
// Every symbol is one printable ASCII cell, independent of terminal colors.
const (
	glyphWall          = '#'
	glyphFloor         = '.'
	glyphPlayer        = '@'
	glyphDog           = 'D'
	glyphChainedDog    = 'd'
	glyphStairsDown    = '>'
	glyphStairsUp      = '<'
	glyphFountain      = 'F'
	glyphSpentFountain = 'f'
	glyphFriendlyGhost = 'G'
	glyphLever         = '+'
	glyphKeepsake      = '&'
	glyphShrine        = '^'
	glyphLostSpirit    = 'S'
	glyphRitualSigil   = 'O'
	glyphEpicWarning   = 'X'
	glyphCryptEntrance = '{'
	glyphCryptExit     = '}'
	glyphSealedCrypt   = '|'
)

var monsterGlyphs = map[monsterKind]rune{
	monsterRat: 'r', monsterSkeleton: 's', monsterGoblin: 'g', monsterOrc: 'o',
	monsterCultist: 'c', monsterSlime: 'm', monsterSpider: 'w', monsterWraith: 'W',
	monsterBlinker: 'b', monsterHexPriest: 'h', monsterMirrorShade: 'M', monsterBoneHound: 'H',
	monsterGargoyle: 'Y', monsterGraveKnight: 'K', monsterCinderDrake: 'A', monsterVoidSeer: 'V',
	monsterSoulLeech: 'L', monsterStormHerald: 'T', monsterBoss: 'B', monsterUnderLich: 'U',
	monsterAshBeetle: 'j', monsterChainImp: 'i', monsterLanternWisp: 'l', monsterChapelSentinel: 'C',
	monsterGloomArcher: 'a', monsterBellRevenant: 'R', monsterIronrootBrute: 'I', monsterFrostHound: 'y',
	monsterRiftWeaver: 'z', monsterCrownlessKnight: 'k', monsterTunnelBat: 't', monsterCaveCrawler: 'v',
	monsterRustGuard: 'n', monsterBarrowArcher: 'J', monsterGraveAcolyte: 'P', monsterEmberHound: 'E',
	monsterObsidianGuard: 'Q', monsterHollowWarden: 'u',
	monsterIronColossus: '1', monsterGraveTitan: '2', monsterAshenWyrm: '3', monsterHollowReaper: '4',
	monsterFrostMonarch: '5', monsterStormLeviathan: '6', monsterDuskTyrant: '7', monsterObsidianGolem: '8',
}

var itemGlyphs = [...]rune{
	itemWeapon:            ')',
	itemArmor:             '[',
	itemPotion:            '!',
	itemFireScroll:        '?',
	itemBlinkStone:        '*',
	itemWardingCharm:      ']',
	itemSunOrb:            '$',
	itemFrostCharm:        '%',
	itemStarfireOrb:       '=',
	itemPhoenixAsh:        '(',
	itemGhostRecallScroll: ';',
	itemStoryScroll:       ':',
	itemSideStoryScroll:   ',',
}

var trapGlyphs = map[trapKind]rune{
	trapBlock:     '_',
	trapPit:       '0',
	trapDarts:     '"',
	trapFlames:    '/',
	trapFrost:     '\\',
	trapBell:      '`',
	trapGas:       '~',
	trapRepulsion: '-',
}

// Glyph fields remain serialized for save compatibility. Display by kind so old
// saves use the current symbols without rewriting their saved gameplay state.
func (m monster) glyph() rune {
	if ch := monsterGlyphs[m.Kind]; ch != 0 {
		return ch
	}
	return m.Glyph
}

func (it item) glyph() rune {
	if it.Kind >= 0 && int(it.Kind) < len(itemGlyphs) {
		return itemGlyphs[it.Kind]
	}
	return it.Glyph
}

func (t cryptTrap) glyph() rune {
	return trapGlyphs[t.Kind]
}

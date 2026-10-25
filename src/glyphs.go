package main

// Reserve feature symbols before assigning monster, item, or trap symbols.
// Every symbol is one printable ASCII cell, independent of terminal colors.
// Item subtypes share symbols within their physical categories.
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
	itemWardingCharm:      '%',
	itemSunOrb:            '=',
	itemFrostCharm:        '%',
	itemStarfireOrb:       '=',
	itemPhoenixAsh:        '!',
	itemGhostRecallScroll: '?',
	itemStoryScroll:       '?',
	itemSideStoryScroll:   '?',
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

func (m monster) glyph() rune {
	return monsterGlyphs[m.Kind]
}

func (it item) glyph() rune {
	return itemGlyphs[it.Kind]
}

func (t cryptTrap) glyph() rune {
	return trapGlyphs[t.Kind]
}

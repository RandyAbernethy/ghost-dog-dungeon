package main

const (
	levelCount        = 10
	mapWidth          = 38
	mapHeight         = 18
	defaultSavePrefix = "ghostdog-dungeon"
	ansiReset         = "\x1b[0m"
)

type pos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type weapon struct {
	Name  string `json:"name"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Magic bool   `json:"magic,omitempty"`
	Reach int    `json:"reach,omitempty"`
}

type armorSlot string

const (
	slotHead armorSlot = "head"
	slotBody armorSlot = "body"
	slotLeg  armorSlot = "leg"
)

type rarity string

const (
	rarityCommon    rarity = "common"
	rarityUncommon  rarity = "uncommon"
	rarityRare      rarity = "rare"
	rarityEpic      rarity = "epic"
	rarityLegendary rarity = "legendary"
)

type armor struct {
	Name        string    `json:"name"`
	Slot        armorSlot `json:"slot"`
	Defense     int       `json:"defense"`
	Rarity      rarity    `json:"rarity"`
	SpellWard   int       `json:"spell_ward,omitempty"`
	StrikeBonus int       `json:"strike_bonus,omitempty"`
}

type itemKind int

const (
	itemWeapon itemKind = iota
	itemArmor
	itemPotion
	itemFireScroll
	itemBlinkStone
	itemWardingCharm
	itemSunOrb
	itemFrostCharm
	itemStarfireOrb
	itemPhoenixAsh
	itemGhostRecallScroll
	itemStoryScroll
	itemSideStoryScroll
)

type item struct {
	Pos          pos      `json:"pos"`
	Kind         itemKind `json:"kind"`
	Name         string   `json:"name"`
	Glyph        rune     `json:"glyph"`
	Weapon       weapon   `json:"weapon"`
	Armor        armor    `json:"armor"`
	StoryChapter int      `json:"story_chapter,omitempty"`
	SideStoryID  int      `json:"side_story_id,omitempty"`
}

type monsterKind string

const (
	monsterRat             monsterKind = "Rat"
	monsterSkeleton        monsterKind = "Skeleton"
	monsterGoblin          monsterKind = "Goblin"
	monsterOrc             monsterKind = "Orc"
	monsterCultist         monsterKind = "Cultist"
	monsterSlime           monsterKind = "Caustic Slime"
	monsterSpider          monsterKind = "Web Spider"
	monsterWraith          monsterKind = "Wraith"
	monsterBlinker         monsterKind = "Blink Stalker"
	monsterHexPriest       monsterKind = "Hex Priest"
	monsterMirrorShade     monsterKind = "Mirror Shade"
	monsterBoneHound       monsterKind = "Bone Hound"
	monsterGargoyle        monsterKind = "Stone Gargoyle"
	monsterGraveKnight     monsterKind = "Grave Knight"
	monsterCinderDrake     monsterKind = "Cinder Drake"
	monsterVoidSeer        monsterKind = "Void Seer"
	monsterSoulLeech       monsterKind = "Soul Leech"
	monsterStormHerald     monsterKind = "Storm Herald"
	monsterBoss            monsterKind = "Dread Lich"
	monsterAshBeetle       monsterKind = "Ash Beetle"
	monsterChainImp        monsterKind = "Chain Imp"
	monsterLanternWisp     monsterKind = "Lantern Wisp"
	monsterChapelSentinel  monsterKind = "Chapel Sentinel"
	monsterGloomArcher     monsterKind = "Gloom Archer"
	monsterBellRevenant    monsterKind = "Bell Revenant"
	monsterIronrootBrute   monsterKind = "Ironroot Brute"
	monsterFrostHound      monsterKind = "Frostbound Hound"
	monsterRiftWeaver      monsterKind = "Rift Weaver"
	monsterCrownlessKnight monsterKind = "Crownless Knight"
)

type monster struct {
	Kind         monsterKind `json:"kind"`
	Name         string      `json:"name"`
	Glyph        rune        `json:"glyph"`
	Pos          pos         `json:"pos"`
	HP           int         `json:"hp"`
	MaxHP        int         `json:"max_hp"`
	MinDamage    int         `json:"min_damage"`
	MaxDamage    int         `json:"max_damage"`
	Undead       bool        `json:"undead"`
	Boss         bool        `json:"boss"`
	FrozenTurns  int         `json:"frozen_turns,omitempty"`
	StunnedTurns int         `json:"stunned_turns,omitempty"`
	FleeTurns    int         `json:"flee_turns,omitempty"`
	StoryChapter int         `json:"story_chapter,omitempty"`
	ActionTurns  int         `json:"action_turns,omitempty"`
}

type player struct {
	Pos                pos      `json:"pos"`
	HP                 int      `json:"hp"`
	MaxHP              int      `json:"max_hp"`
	Weapon             weapon   `json:"weapon"`
	HeadArmor          armor    `json:"head_armor"`
	BodyArmor          armor    `json:"body_armor"`
	LegArmor           armor    `json:"leg_armor"`
	Potions            int      `json:"potions"`
	FireScrolls        int      `json:"fire_scrolls"`
	BlinkStones        int      `json:"blink_stones"`
	WardingCharms      int      `json:"warding_charms"`
	SunOrbs            int      `json:"sun_orbs"`
	FrostCharms        int      `json:"frost_charms"`
	StarfireOrbs       int      `json:"starfire_orbs"`
	PhoenixAshes       int      `json:"phoenix_ashes"`
	GhostRecallScrolls int      `json:"ghost_recall_scrolls,omitempty"`
	StoryChapters      []int    `json:"story_chapters,omitempty"`
	SideStories        []int    `json:"side_stories,omitempty"`
	Weapons            []weapon `json:"weapons,omitempty"`
	Armors             []armor  `json:"armors,omitempty"`
	ShieldTurns        int      `json:"shield_turns"`
	HexedTurns         int      `json:"hexed_turns"`
	WebbedTurns        int      `json:"webbed_turns"`
}

type ghostDog struct {
	Pos   pos  `json:"pos"`
	HP    int  `json:"hp"`
	MaxHP int  `json:"max_hp"`
	Freed bool `json:"freed"`
	Alive bool `json:"alive"`
}

type level struct {
	Index          int           `json:"index"`
	Tiles          [][]rune      `json:"-"`
	Monsters       []*monster    `json:"monsters"`
	Items          []item        `json:"items"`
	Stairs         pos           `json:"stairs"`
	HasStair       bool          `json:"has_stair"`
	UpStairs       pos           `json:"up_stairs"`
	HasUpStair     bool          `json:"has_up_stair"`
	EscapeStairs   pos           `json:"escape_stairs"`
	HasEscapeStair bool          `json:"has_escape_stair"`
	DogChain       *pos          `json:"dog_chain,omitempty"`
	Start          pos           `json:"start"`
	Secret         *secretRoom   `json:"secret,omitempty"`
	ExtraSecrets   []*secretRoom `json:"extra_secrets,omitempty"`
	Layout         layoutKind    `json:"layout,omitempty"`
	Theme          floorTheme    `json:"theme,omitempty"`
	Encounter      encounterKind `json:"encounter,omitempty"`
	Special        *specialRoom  `json:"special,omitempty"`
	Events         []floorEvent  `json:"events,omitempty"`
	Visited        bool          `json:"visited"`
	Fountains      []fountain    `json:"fountains,omitempty"`
}

type fountain struct {
	Pos  pos  `json:"pos"`
	Used bool `json:"used"`
}

type secretRoom struct {
	Door        pos         `json:"door"`
	Tiles       []pos       `json:"tiles"`
	Revealed    bool        `json:"revealed"`
	Walls       []pos       `json:"walls,omitempty"`
	Inner       *secretRoom `json:"inner,omitempty"`
	SideStoryID int         `json:"side_story_id,omitempty"`
}

type room struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type simpleRNG struct {
	State uint64 `json:"state"`
}

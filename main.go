package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

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
}

type armorSlot string

const (
	slotHead armorSlot = "head"
	slotBody armorSlot = "body"
	slotFeet armorSlot = "feet"
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
	Name    string    `json:"name"`
	Slot    armorSlot `json:"slot"`
	Defense int       `json:"defense"`
	Rarity  rarity    `json:"rarity"`
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
)

type item struct {
	Pos          pos      `json:"pos"`
	Kind         itemKind `json:"kind"`
	Name         string   `json:"name"`
	Glyph        rune     `json:"glyph"`
	Weapon       weapon   `json:"weapon"`
	Armor        armor    `json:"armor"`
	StoryChapter int      `json:"story_chapter,omitempty"`
}

type monsterKind string

const (
	monsterRat         monsterKind = "Rat"
	monsterSkeleton    monsterKind = "Skeleton"
	monsterGoblin      monsterKind = "Goblin"
	monsterOrc         monsterKind = "Orc"
	monsterCultist     monsterKind = "Cultist"
	monsterSlime       monsterKind = "Caustic Slime"
	monsterSpider      monsterKind = "Web Spider"
	monsterWraith      monsterKind = "Wraith"
	monsterBlinker     monsterKind = "Blink Stalker"
	monsterHexPriest   monsterKind = "Hex Priest"
	monsterMirrorShade monsterKind = "Mirror Shade"
	monsterBoneHound   monsterKind = "Bone Hound"
	monsterGargoyle    monsterKind = "Stone Gargoyle"
	monsterGraveKnight monsterKind = "Grave Knight"
	monsterCinderDrake monsterKind = "Cinder Drake"
	monsterVoidSeer    monsterKind = "Void Seer"
	monsterSoulLeech   monsterKind = "Soul Leech"
	monsterStormHerald monsterKind = "Storm Herald"
	monsterBoss        monsterKind = "Dread Lich"
)

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
}

var ghostDogStory = [...]string{
	"A small dog named Ash lived in the chapel yard, where the bell was his favorite song.",
	"Ash chose Mara, the bellkeeper's daughter, and followed her everywhere through the village.",
	"One winter, a stranger in a crown of bone came asking for the chapel's buried names.",
	"When the villagers refused, the stranger opened a stair beneath the chapel and called the dead upward.",
	"Ash barked until Mara woke. She led the children through a back passage while he drew the dead away.",
	"The stranger was the Dread Lich. He struck Ash with a curse meant to silence him forever.",
	"Mara chained Ash beside the stair, hoping the silver links would keep the Lich's curse from spreading.",
	"The curse killed Ash, but his loyalty held his spirit fast. He guarded the stair and waited for a kind voice.",
	"The Lich fed on Ash's bound spirit to keep his own heart beating. Only the Lich's death can end that bond.",
	"The final page is in the Lich's keeping: when he falls, Ash is free to choose his own way home.",
}

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
	FleeTurns    int         `json:"flee_turns,omitempty"`
	StoryChapter int         `json:"story_chapter,omitempty"`
}

type player struct {
	Pos                pos    `json:"pos"`
	HP                 int    `json:"hp"`
	MaxHP              int    `json:"max_hp"`
	Weapon             weapon `json:"weapon"`
	HeadArmor          armor  `json:"head_armor"`
	BodyArmor          armor  `json:"body_armor"`
	FeetArmor          armor  `json:"feet_armor"`
	Potions            int    `json:"potions"`
	FireScrolls        int    `json:"fire_scrolls"`
	BlinkStones        int    `json:"blink_stones"`
	WardingCharms      int    `json:"warding_charms"`
	SunOrbs            int    `json:"sun_orbs"`
	FrostCharms        int    `json:"frost_charms"`
	StarfireOrbs       int    `json:"starfire_orbs"`
	PhoenixAshes       int    `json:"phoenix_ashes"`
	GhostRecallScrolls int    `json:"ghost_recall_scrolls,omitempty"`
	StoryChapters      []int  `json:"story_chapters,omitempty"`
	ShieldTurns        int    `json:"shield_turns"`
	HexedTurns         int    `json:"hexed_turns"`
	WebbedTurns        int    `json:"webbed_turns"`
}

type ghostDog struct {
	Pos   pos  `json:"pos"`
	HP    int  `json:"hp"`
	MaxHP int  `json:"max_hp"`
	Freed bool `json:"freed"`
	Alive bool `json:"alive"`
}

type level struct {
	Index          int         `json:"index"`
	Tiles          [][]rune    `json:"-"`
	Monsters       []*monster  `json:"monsters"`
	Items          []item      `json:"items"`
	Stairs         pos         `json:"stairs"`
	HasStair       bool        `json:"has_stair"`
	UpStairs       pos         `json:"up_stairs"`
	HasUpStair     bool        `json:"has_up_stair"`
	EscapeStairs   pos         `json:"escape_stairs"`
	HasEscapeStair bool        `json:"has_escape_stair"`
	DogChain       *pos        `json:"dog_chain,omitempty"`
	Start          pos         `json:"start"`
	Secret         *secretRoom `json:"secret,omitempty"`
	Visited        bool        `json:"visited"`
	Fountains      []fountain  `json:"fountains,omitempty"`
}

type fountain struct {
	Pos  pos  `json:"pos"`
	Used bool `json:"used"`
}

type secretRoom struct {
	Door     pos   `json:"door"`
	Tiles    []pos `json:"tiles"`
	Revealed bool  `json:"revealed"`
}

type room struct {
	X int
	Y int
	W int
	H int
}

type simpleRNG struct {
	State uint64 `json:"state"`
}

type saveState struct {
	RNGState      uint64        `json:"rng_state"`
	Seed          int64         `json:"seed"`
	SaveFile      string        `json:"save_file"`
	Levels        []*level      `json:"levels"`
	CurrentLevel  int           `json:"current_level"`
	Player        player        `json:"player"`
	Dog           ghostDog      `json:"dog"`
	Messages      []string      `json:"messages"`
	KnownMonsters []monsterKind `json:"known_monsters"`
	Won           bool          `json:"won"`
	Quit          bool          `json:"quit"`
	LevelTiles    []string      `json:"level_tiles"`
}

type game struct {
	rng              *simpleRNG
	seed             int64
	saveFile         string
	timestampedSaves bool
	levels           []*level
	currentLevel     int
	player           player
	dog              ghostDog
	messages         []string
	knownMonsters    map[monsterKind]bool
	dogFocus         *monster
	turnDamage       map[*monster]int
	won              bool
	quit             bool
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("ghostdog-dungeon", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Int64("seed", time.Now().UnixNano(), "random seed for dungeon generation")
	saveFile := flags.String("save-file", "", "fixed path used by save/load commands; otherwise saves use a timestamped name")
	loadFile := flags.String("load-file", "", "load this save file at startup")
	loadSave := flags.Bool("load", false, "load the path supplied with --save-file (legacy form)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: ghostdog-dungeon [--seed N] [--save-file path] [--load-file path]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Enter the dungeon, free the ghost dog, reach the tenth floor, and escape by finding the hidden way out beyond the Dread Lich.")
		fmt.Fprintln(stderr, "Commands: w/a/s/d or arrows move, q/e/z/x diagonals, . wait, </> stairs, p potion, f fire, b ward, g blink, u sun, t frost, o starfire, n ash, r recall, v fountain, i inventory, c codex, m inspect, k search, S save, L load, Ctrl-C quit")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return errors.New("this game does not take positional arguments")
	}

	if *loadSave && *loadFile != "" {
		return errors.New("use either --load-file or --load, not both")
	}
	if *loadSave && *saveFile == "" {
		return errors.New("--load needs a path; use --load-file path or --load --save-file path")
	}
	var g *game
	var err error
	if *loadFile != "" || *loadSave {
		path := *loadFile
		if path == "" {
			path = *saveFile
		}
		g, err = loadGame(path)
		if err != nil {
			return err
		}
		g.addMessage("Loaded saved game.")
	} else {
		g = newGameWithSaveFile(*seed, *saveFile)
	}
	return g.loop(stdin, stdout)
}

func newGame(seed int64) *game {
	return newGameWithSaveFile(seed, "")
}

func newGameWithSaveFile(seed int64, saveFile string) *game {
	rng := newSimpleRNG(seed)
	g := &game{
		rng:              rng,
		seed:             seed,
		saveFile:         saveFile,
		timestampedSaves: saveFile == "",
		knownMonsters:    map[monsterKind]bool{},
		player: player{
			HP:            46,
			MaxHP:         46,
			Weapon:        weapon{Name: "Rusty Knife", Min: 3, Max: 6},
			HeadArmor:     armor{Name: "Padded Hood", Slot: slotHead, Defense: 1, Rarity: rarityCommon},
			BodyArmor:     armor{Name: "Worn Coat", Slot: slotBody, Defense: 1, Rarity: rarityCommon},
			FeetArmor:     armor{Name: "Frayed Boots", Slot: slotFeet, Defense: 0, Rarity: rarityCommon},
			Potions:       3,
			FireScrolls:   1,
			BlinkStones:   0,
			WardingCharms: 0,
			SunOrbs:       0,
		},
		dog: ghostDog{HP: 34, MaxHP: 34, Alive: true},
	}

	g.levels = make([]*level, levelCount)
	for i := 0; i < levelCount; i++ {
		tiles, start := makeFloorPlan(rng, i)
		lvl := &level{Index: i, Tiles: tiles, Start: start}
		g.levels[i] = lvl
		g.populateLevel(lvl)
	}
	g.player.Pos = g.levels[0].Start
	g.placeFountains()

	g.levels[0].Visited = true
	g.messages = []string{
		"You step into the dungeon. The door slams shut behind you.",
		"Find the ghost dog, delve through ten ever-larger floors, and uncover the hidden way out guarded by the Dread Lich.",
	}
	return g
}

func newSimpleRNG(seed int64) *simpleRNG {
	state := uint64(seed)
	if state == 0 {
		state = 1
	}
	return &simpleRNG{State: state}
}

func (r *simpleRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	x := r.State
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	if x == 0 {
		x = 0x9e3779b97f4a7c15
	}
	r.State = x
	return int(x % uint64(n))
}

func makeFloorPlan(rng *simpleRNG, depth int) ([][]rune, pos) {
	targetFloors := floorBudget(depth)
	tiles := make([][]rune, mapHeight)
	for y := 0; y < mapHeight; y++ {
		tiles[y] = make([]rune, mapWidth)
		for x := 0; x < mapWidth; x++ {
			tiles[y][x] = '#'
		}
	}

	rooms := []room{}
	startRoom := room{X: 2 + rng.Intn(3), Y: 2 + rng.Intn(3), W: 6 + depth/4, H: 4 + depth/5}
	if depth == 0 {
		startRoom = room{X: 3, Y: 5, W: 8, H: 5}
	}
	carveRoom(tiles, startRoom)
	rooms = append(rooms, startRoom)

	targetRooms := depth + 1
	attempts := 0
	for len(rooms) < targetRooms && attempts < 220 {
		attempts++
		candidate := room{X: 1 + rng.Intn(mapWidth-8), Y: 1 + rng.Intn(mapHeight-6), W: 4 + rng.Intn(3), H: 3 + rng.Intn(2)}
		if candidate.X+candidate.W >= mapWidth-1 || candidate.Y+candidate.H >= mapHeight-1 {
			continue
		}
		blocked := false
		for _, existing := range rooms {
			if roomsOverlap(candidate, existing, 1) {
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}
		trial := cloneTiles(tiles)
		carveRoom(trial, candidate)
		prev := rooms[rng.Intn(len(rooms))]
		connectRooms(trial, center(prev), center(candidate), rng)
		if countFloorTiles(trial) > targetFloors {
			continue
		}
		tiles = trial
		rooms = append(rooms, candidate)
	}

	for i := 0; i < len(rooms)/3; i++ {
		a := rooms[rng.Intn(len(rooms))]
		b := rooms[rng.Intn(len(rooms))]
		if a == b {
			continue
		}
		trial := cloneTiles(tiles)
		connectRooms(trial, center(a), center(b), rng)
		if countFloorTiles(trial) <= targetFloors {
			tiles = trial
		}
	}

	growFloorTiles(tiles, targetFloors, rng)

	start := center(startRoom)
	tiles[start.Y][start.X] = '.'
	return tiles, start
}

func floorBudget(depth int) int { return 40 + depth*30 }

func cloneTiles(tiles [][]rune) [][]rune {
	clone := make([][]rune, len(tiles))
	for i := range tiles {
		clone[i] = append([]rune(nil), tiles[i]...)
	}
	return clone
}

func countFloorTiles(tiles [][]rune) int {
	count := 0
	for y := range tiles {
		for x := range tiles[y] {
			if tiles[y][x] != '#' {
				count++
			}
		}
	}
	return count
}

func growFloorTiles(tiles [][]rune, target int, rng *simpleRNG) {
	for countFloorTiles(tiles) < target {
		candidates := []pos{}
		for y := 1; y < mapHeight-1; y++ {
			for x := 1; x < mapWidth-1; x++ {
				if tiles[y][x] != '#' {
					continue
				}
				for _, delta := range []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
					next := pos{X: x + delta.X, Y: y + delta.Y}
					if tiles[next.Y][next.X] != '#' {
						candidates = append(candidates, pos{X: x, Y: y})
						break
					}
				}
			}
		}
		if len(candidates) == 0 {
			panic("could not grow dungeon floor")
		}
		chosen := candidates[rng.Intn(len(candidates))]
		tiles[chosen.Y][chosen.X] = '.'
	}
}

func carveRoom(tiles [][]rune, rm room) {
	for y := rm.Y; y < rm.Y+rm.H; y++ {
		for x := rm.X; x < rm.X+rm.W; x++ {
			tiles[y][x] = '.'
		}
	}
}

func connectRooms(tiles [][]rune, a, b pos, rng *simpleRNG) {
	if rng.Intn(2) == 0 {
		carveHorizontal(tiles, a.X, b.X, a.Y)
		carveVertical(tiles, a.Y, b.Y, b.X)
	} else {
		carveVertical(tiles, a.Y, b.Y, a.X)
		carveHorizontal(tiles, a.X, b.X, b.Y)
	}
}

func carveHorizontal(tiles [][]rune, x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		tiles[y][x] = '.'
	}
}

func carveVertical(tiles [][]rune, y1, y2, x int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		tiles[y][x] = '.'
	}
}

func center(r room) pos { return pos{X: r.X + r.W/2, Y: r.Y + r.H/2} }

func roomsOverlap(a, b room, margin int) bool {
	return a.X-margin < b.X+b.W && a.X+a.W+margin > b.X && a.Y-margin < b.Y+b.H && a.Y+a.H+margin > b.Y
}

func (g *game) addSecretRoom(lvl *level) {
	if g.rng.Intn(100) >= 40 {
		return
	}
	dirs := []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
	for tries := 0; tries < 240; tries++ {
		base := pos{X: 1 + g.rng.Intn(mapWidth-2), Y: 1 + g.rng.Intn(mapHeight-2)}
		if lvl.Tiles[base.Y][base.X] != '.' {
			continue
		}
		dir := dirs[g.rng.Intn(len(dirs))]
		door := pos{X: base.X + dir.X, Y: base.Y + dir.Y}
		if !g.inBounds(door) || lvl.Tiles[door.Y][door.X] != '#' {
			continue
		}
		w := 3 + g.rng.Intn(3)
		h := 3 + g.rng.Intn(2)
		x0, y0, x1, y1, ok := secretBounds(door, dir, w, h)
		if !ok {
			continue
		}
		tiles := make([]pos, 0, w*h)
		valid := true
		for y := y0; y <= y1 && valid; y++ {
			for x := x0; x <= x1; x++ {
				if lvl.Tiles[y][x] != '#' {
					valid = false
					break
				}
				tiles = append(tiles, pos{X: x, Y: y})
			}
		}
		if !valid {
			continue
		}
		lvl.Secret = &secretRoom{Door: door, Tiles: tiles}
		g.addSecretTreasure(lvl)
		return
	}
}

func secretBounds(door, dir pos, w, h int) (int, int, int, int, bool) {
	var x0, y0, x1, y1 int
	switch {
	case dir.X == 1:
		x0, x1 = door.X+1, door.X+w
		y0, y1 = door.Y-h/2, door.Y-h/2+h-1
	case dir.X == -1:
		x0, x1 = door.X-w, door.X-1
		y0, y1 = door.Y-h/2, door.Y-h/2+h-1
	case dir.Y == 1:
		y0, y1 = door.Y+1, door.Y+h
		x0, x1 = door.X-w/2, door.X-w/2+w-1
	case dir.Y == -1:
		y0, y1 = door.Y-h, door.Y-1
		x0, x1 = door.X-w/2, door.X-w/2+w-1
	default:
		return 0, 0, 0, 0, false
	}
	if x0 < 1 || y0 < 1 || x1 >= mapWidth-1 || y1 >= mapHeight-1 {
		return 0, 0, 0, 0, false
	}
	return x0, y0, x1, y1, true
}

func (g *game) addSecretTreasure(lvl *level) {
	if lvl.Secret == nil || len(lvl.Secret.Tiles) == 0 {
		return
	}
	count := 1
	if g.rng.Intn(100) < 20 {
		count++
	}
	used := map[pos]bool{}
	pool := g.secretTreasurePool(lvl.Index)
	// Secret equipment is unique within a run. Consumables can still repeat,
	// but a searched room should not hand back a weapon or armor already found.
	pool = g.filterFoundSecretEquipment(pool)
	for i := 0; i < count && len(pool) > 0; i++ {
		idx := g.rng.Intn(len(pool))
		it := pool[idx]
		pool = append(pool[:idx], pool[idx+1:]...)
		for tries := 0; tries < 60; tries++ {
			p := lvl.Secret.Tiles[g.rng.Intn(len(lvl.Secret.Tiles))]
			if used[p] {
				continue
			}
			it.Pos = p
			used[p] = true
			lvl.Items = append(lvl.Items, it)
			break
		}
	}
}

func (g *game) filterFoundSecretEquipment(pool []item) []item {
	found := map[string]bool{
		g.player.Weapon.Name:    true,
		g.player.HeadArmor.Name: true,
		g.player.BodyArmor.Name: true,
		g.player.FeetArmor.Name: true,
	}
	for _, lvl := range g.levels {
		if lvl == nil {
			continue
		}
		for _, existing := range lvl.Items {
			if existing.Kind == itemWeapon || existing.Kind == itemArmor {
				found[existing.Name] = true
			}
		}
	}
	filtered := make([]item, 0, len(pool))
	for _, candidate := range pool {
		if (candidate.Kind == itemWeapon || candidate.Kind == itemArmor) && found[candidate.Name] {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func (g *game) secretTreasurePool(depth int) []item {
	pool := []item{
		{Kind: itemPotion, Name: "healing potion", Glyph: '!'},
		{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'},
		{Kind: itemBlinkStone, Name: "blink stone", Glyph: '*'},
		{Kind: itemFrostCharm, Name: "frost charm", Glyph: '*'},
	}
	if depth >= 4 {
		pool = append(pool,
			item{Kind: itemSunOrb, Name: "sun orb", Glyph: '*'},
			item{Kind: itemStarfireOrb, Name: "starfire orb", Glyph: '*'},
			item{Kind: itemPhoenixAsh, Name: "phoenix ash", Glyph: '*'},
			makeWeaponItem(weapon{Name: "Emberbrand", Min: 11, Max: 16, Magic: true}),
			makeWeaponItem(weapon{Name: "Tempest Spear", Min: 12, Max: 17, Magic: true}),
			makeWeaponItem(weapon{Name: "Voidglass Dagger", Min: 13, Max: 18, Magic: true}),
			makeWeaponItem(weapon{Name: "Lichbane Greatsword", Min: 14, Max: 19, Magic: true}),
			makeWeaponItem(weapon{Name: "Gravetide Maul", Min: 16, Max: 22, Magic: true}),
			makeArmorItem(armor{Name: "Aegis of Echoes", Slot: slotBody, Defense: 6, Rarity: rarityLegendary}),
			makeArmorItem(armor{Name: "Crown of the Hollow Star", Slot: slotHead, Defense: 6, Rarity: rarityLegendary}),
			makeArmorItem(armor{Name: "Wraithstep Greaves", Slot: slotFeet, Defense: 6, Rarity: rarityLegendary}),
		)
	}
	if depth >= 7 {
		pool = append(pool,
			makeWeaponItem(weapon{Name: "Star-Eater Blade", Min: 18, Max: 24, Magic: true}),
			makeArmorItem(armor{Name: "Voidheart Plate", Slot: slotBody, Defense: 7, Rarity: rarityLegendary}),
		)
	}
	return pool
}

func (g *game) addEscapeSanctum(lvl *level) {
	dirs := []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
	for minDistance := 16; minDistance >= 8; minDistance-- {
		for tries := 0; tries < 240; tries++ {
			base := g.randomOpenTileFarFrom(lvl, map[pos]bool{}, lvl.Start, minDistance)
			dir := dirs[g.rng.Intn(len(dirs))]
			door := pos{X: base.X + dir.X, Y: base.Y + dir.Y}
			if !g.inBounds(door) || lvl.Tiles[door.Y][door.X] != '#' {
				continue
			}
			w := 4 + g.rng.Intn(2)
			h := 3 + g.rng.Intn(2)
			x0, y0, x1, y1, ok := secretBounds(door, dir, w, h)
			if !ok {
				continue
			}
			tiles := make([]pos, 0, w*h)
			valid := true
			for y := y0; y <= y1 && valid; y++ {
				for x := x0; x <= x1; x++ {
					if lvl.Tiles[y][x] != '#' {
						valid = false
						break
					}
					tiles = append(tiles, pos{X: x, Y: y})
				}
			}
			if !valid {
				continue
			}
			lvl.Secret = &secretRoom{Door: door, Tiles: tiles}
			lvl.HasEscapeStair = true
			lvl.EscapeStairs = tiles[g.rng.Intn(len(tiles))]
			g.addSecretTreasure(lvl)
			return
		}
	}
	g.addSecretRoom(lvl)
	if lvl.Secret != nil && len(lvl.Secret.Tiles) > 0 {
		lvl.HasEscapeStair = true
		lvl.EscapeStairs = lvl.Secret.Tiles[g.rng.Intn(len(lvl.Secret.Tiles))]
	}
}

func (g *game) bossGuardPosition(lvl *level, occupied map[pos]bool) (pos, bool) {
	if lvl.Secret == nil {
		return pos{}, false
	}
	best := pos{}
	bestDist := -1
	for _, d := range []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		p := pos{X: lvl.Secret.Door.X + d.X, Y: lvl.Secret.Door.Y + d.Y}
		if !g.inBounds(p) || lvl.Tiles[p.Y][p.X] != '.' || occupied[p] {
			continue
		}
		if dist := distance(p, lvl.Start); dist > bestDist {
			best = p
			bestDist = dist
		}
	}
	if bestDist < 0 {
		return pos{}, false
	}
	return best, true
}

func makeWeaponItem(w weapon) item {
	return item{Kind: itemWeapon, Name: w.Name, Glyph: ')', Weapon: w}
}

func makeArmorItem(a armor) item {
	return item{Kind: itemArmor, Name: a.Name, Glyph: '[', Armor: a}
}

func (g *game) populateLevel(lvl *level) {
	if lvl.Index == levelCount-1 {
		g.addEscapeSanctum(lvl)
	} else {
		g.addSecretRoom(lvl)
	}
	occupied := map[pos]bool{lvl.Start: true}
	if lvl.Index > 0 {
		lvl.HasUpStair = true
		lvl.UpStairs = lvl.Start
	}
	if lvl.Index < levelCount-1 {
		lvl.HasStair = true
		lvl.Stairs = g.randomOpenTile(lvl, occupied)
		occupied[lvl.Stairs] = true
	}
	if lvl.Index == 1 {
		dogPos := g.randomOpenTile(lvl, occupied)
		lvl.DogChain = &dogPos
		occupied[dogPos] = true
	}
	for _, it := range g.itemsForLevel(lvl.Index) {
		it.Pos = g.randomOpenTile(lvl, occupied)
		occupied[it.Pos] = true
		lvl.Items = append(lvl.Items, it)
	}
	for _, spec := range g.monstersForLevel(lvl.Index) {
		if spec.Boss {
			spec.StoryChapter = levelCount
			if p, ok := g.bossGuardPosition(lvl, occupied); ok {
				spec.Pos = p
			} else {
				spec.Pos = g.randomOpenTileFarFrom(lvl, occupied, lvl.Start, 12)
			}
		} else {
			spec.Pos = g.randomOpenTile(lvl, occupied)
		}
		occupied[spec.Pos] = true
		m := spec
		lvl.Monsters = append(lvl.Monsters, &m)
	}
}

func (g *game) placeFountains() {
	order := make([]int, len(g.levels))
	for i := range order {
		order[i] = i
	}
	for i := len(order) - 1; i > 0; i-- {
		j := g.rng.Intn(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	count := 1 + g.rng.Intn(3)
	for _, levelIndex := range order[:count] {
		lvl := g.levels[levelIndex]
		spots := make([]pos, 0, floorBudget(levelIndex))
		for y := 1; y < mapHeight-1; y++ {
			for x := 1; x < mapWidth-1; x++ {
				p := pos{X: x, Y: y}
				if lvl.Tiles[y][x] == '#' || p == lvl.Start || p == g.player.Pos ||
					(lvl.HasStair && p == lvl.Stairs) || (lvl.HasUpStair && p == lvl.UpStairs) ||
					(lvl.DogChain != nil && p == *lvl.DogChain) || g.monsterAt(lvl, p) != nil || g.itemAt(lvl, p) != nil {
					continue
				}
				spots = append(spots, p)
			}
		}
		if len(spots) > 0 {
			lvl.Fountains = append(lvl.Fountains, fountain{Pos: spots[g.rng.Intn(len(spots))]})
		}
	}
}

func (g *game) itemsForLevel(depth int) []item {
	items := []item{}
	if depth < levelCount-1 {
		items = append(items, item{Kind: itemStoryScroll, Name: fmt.Sprintf("Ghost Dog story scroll %d", depth+1), Glyph: '?', StoryChapter: depth + 1})
	}
	if depth%2 == 0 {
		items = append(items, item{Kind: itemPotion, Name: "healing potion", Glyph: '!'})
	}
	weaponPools := [][]weapon{{
		{Name: "Short Sword", Min: 4, Max: 7},
		{Name: "Chapel Mace", Min: 4, Max: 8},
	}, {
		{Name: "Iron Spear", Min: 5, Max: 8},
		{Name: "Hooked Glaive", Min: 5, Max: 9},
	}, {
		{Name: "Battle Axe", Min: 6, Max: 10},
		{Name: "Dawnstar Flail", Min: 6, Max: 10},
	}, {
		{Name: "Moonblade", Min: 7, Max: 11},
		{Name: "Rune Saber", Min: 7, Max: 12},
	}, {
		{Name: "Dragontooth Pike", Min: 8, Max: 13},
		{Name: "Starforged Hammer", Min: 9, Max: 14},
	}, {
		{Name: "Gravecleaver", Min: 10, Max: 15},
		{Name: "Hollowfang Spear", Min: 10, Max: 16},
	}, {
		{Name: "Frostbite Axe", Min: 11, Max: 16},
		{Name: "Gloomsteel Saber", Min: 12, Max: 16},
	}, {
		{Name: "Graveglass Halberd", Min: 12, Max: 17},
		{Name: "Stormcaller Blade", Min: 13, Max: 17},
	}, {
		{Name: "Wyrmheart Maul", Min: 13, Max: 18},
		{Name: "Gloaming Pike", Min: 14, Max: 18},
	}, {
		{Name: "Dawnforged Greatsword", Min: 15, Max: 19},
		{Name: "Kingsbane Axe", Min: 16, Max: 19},
	}}
	armorPools := [][]armor{{
		{Name: "Leather Jerkin", Slot: slotBody, Defense: 2, Rarity: rarityUncommon},
		{Name: "Scout Cap", Slot: slotHead, Defense: 2, Rarity: rarityUncommon},
	}, {
		{Name: "Iron Helm", Slot: slotHead, Defense: 2, Rarity: rarityUncommon},
		{Name: "Wolfhide Boots", Slot: slotFeet, Defense: 1, Rarity: rarityUncommon},
	}, {
		{Name: "Reinforced Greaves", Slot: slotFeet, Defense: 2, Rarity: rarityRare},
		{Name: "Scale Coat", Slot: slotBody, Defense: 3, Rarity: rarityRare},
	}, {
		{Name: "Knight Mail", Slot: slotBody, Defense: 3, Rarity: rarityEpic},
		{Name: "Nightstride Boots", Slot: slotFeet, Defense: 3, Rarity: rarityEpic},
	}, {
		{Name: "Warden Crown", Slot: slotHead, Defense: 3, Rarity: rarityLegendary},
		{Name: "Sunplate Cuirass", Slot: slotBody, Defense: 4, Rarity: rarityLegendary},
	}, {
		{Name: "Dreadmark Mantle", Slot: slotBody, Defense: 4, Rarity: rarityEpic},
		{Name: "Gloamrunner Boots", Slot: slotFeet, Defense: 3, Rarity: rarityEpic},
	}, {
		{Name: "Aetherweave Hood", Slot: slotHead, Defense: 4, Rarity: rarityEpic},
		{Name: "Ashguard Coat", Slot: slotBody, Defense: 4, Rarity: rarityEpic},
	}, {
		{Name: "Stormscale Greaves", Slot: slotFeet, Defense: 4, Rarity: rarityEpic},
		{Name: "Graveward Helm", Slot: slotHead, Defense: 4, Rarity: rarityEpic},
	}, {
		{Name: "Ruinplate Vest", Slot: slotBody, Defense: 5, Rarity: rarityEpic},
		{Name: "Mourner's Helm", Slot: slotHead, Defense: 5, Rarity: rarityEpic},
	}, {
		{Name: "Starforged Cuirass", Slot: slotBody, Defense: 5, Rarity: rarityLegendary},
		{Name: "Voidwalker Boots", Slot: slotFeet, Defense: 5, Rarity: rarityLegendary},
	}}
	items = append(items, makeWeaponItem(weaponPools[depth][g.rng.Intn(len(weaponPools[depth]))]))
	items = append(items, makeArmorItem(armorPools[depth][g.rng.Intn(len(armorPools[depth]))]))
	switch depth {
	case 0:
		items = append(items, item{Kind: itemFireScroll, Name: "fire scroll", Glyph: '?'})
	case 1:
		items = append(items, item{Kind: itemBlinkStone, Name: "blink stone", Glyph: '*'})
	case 2:
		items = append(items, item{Kind: itemFrostCharm, Name: "frost charm", Glyph: '*'})
	case 3:
		items = append(items, item{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'})
	case 4:
		items = append(items, item{Kind: itemSunOrb, Name: "sun orb", Glyph: '*'})
	case 5:
		items = append(items, item{Kind: itemFrostCharm, Name: "frost charm", Glyph: '*'})
	case 6:
		items = append(items, item{Kind: itemWardingCharm, Name: "warding charm", Glyph: '*'})
	case 7:
		items = append(items, item{Kind: itemFireScroll, Name: "fire scroll", Glyph: '?'})
	case 8:
		items = append(items, item{Kind: itemBlinkStone, Name: "blink stone", Glyph: '*'})
	case 9:
		items = append(items, item{Kind: itemSunOrb, Name: "sun orb", Glyph: '*'})
	}
	return items
}

func monsterCountForDepth(depth int) int {
	// The floor generator fills exactly this many walkable tiles, so quota is
	// one monster per roughly 30 tiles (level one remains a single encounter).
	return max(1, floorBudget(depth)/30)
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
	return levels[depth]
}

func scaleMonsterForDepth(m monster, depth int) monster {
	m.MaxHP += depth * 2
	m.HP = m.MaxHP
	damageBonus := depth / 3
	m.MinDamage += damageBonus
	m.MaxDamage += damageBonus
	return m
}

func (g *game) monstersForLevel(depth int) []monster {
	roster := monsterPoolForLevel(depth)
	count := min(monsterCountForDepth(depth), len(roster))
	var spawn []monster
	if depth == levelCount-1 {
		spawn = append([]monster(nil), roster[:count-1]...)
		spawn = append(spawn, roster[len(roster)-1])
	} else {
		spawn = append([]monster(nil), roster[:count]...)
	}
	for i := range spawn {
		spawn[i] = scaleMonsterForDepth(spawn[i], depth)
	}
	return spawn
}

func newMonster(kind monsterKind) monster {
	switch kind {
	case monsterRat:
		return monster{Kind: kind, Name: "Dungeon Rat", Glyph: 'r', HP: 8, MaxHP: 8, MinDamage: 2, MaxDamage: 4}
	case monsterSkeleton:
		return monster{Kind: kind, Name: "Skeleton", Glyph: 's', HP: 11, MaxHP: 11, MinDamage: 3, MaxDamage: 5, Undead: true}
	case monsterGoblin:
		return monster{Kind: kind, Name: "Goblin Knifer", Glyph: 'g', HP: 10, MaxHP: 10, MinDamage: 3, MaxDamage: 6}
	case monsterOrc:
		return monster{Kind: kind, Name: "Orc Brute", Glyph: 'o', HP: 16, MaxHP: 16, MinDamage: 4, MaxDamage: 7}
	case monsterCultist:
		return monster{Kind: kind, Name: "Cultist", Glyph: 'c', HP: 12, MaxHP: 12, MinDamage: 3, MaxDamage: 6}
	case monsterSlime:
		return monster{Kind: kind, Name: "Caustic Slime", Glyph: 'm', HP: 14, MaxHP: 14, MinDamage: 3, MaxDamage: 5}
	case monsterSpider:
		return monster{Kind: kind, Name: "Web Spider", Glyph: 'w', HP: 13, MaxHP: 13, MinDamage: 3, MaxDamage: 5}
	case monsterWraith:
		return monster{Kind: kind, Name: "Wraith", Glyph: 'W', HP: 15, MaxHP: 15, MinDamage: 4, MaxDamage: 7, Undead: true}
	case monsterBlinker:
		return monster{Kind: kind, Name: "Blink Stalker", Glyph: 'b', HP: 16, MaxHP: 16, MinDamage: 4, MaxDamage: 7}
	case monsterHexPriest:
		return monster{Kind: kind, Name: "Hex Priest", Glyph: 'h', HP: 18, MaxHP: 18, MinDamage: 4, MaxDamage: 7}
	case monsterMirrorShade:
		return monster{Kind: kind, Name: "Mirror Shade", Glyph: 'M', HP: 20, MaxHP: 20, MinDamage: 5, MaxDamage: 8, Undead: true}
	case monsterBoneHound:
		return monster{Kind: kind, Name: "Bone Hound", Glyph: 'H', HP: 18, MaxHP: 18, MinDamage: 4, MaxDamage: 7, Undead: true}
	case monsterGargoyle:
		return monster{Kind: kind, Name: "Stone Gargoyle", Glyph: 'Y', HP: 24, MaxHP: 24, MinDamage: 5, MaxDamage: 8}
	case monsterGraveKnight:
		return monster{Kind: kind, Name: "Grave Knight", Glyph: 'K', HP: 26, MaxHP: 26, MinDamage: 6, MaxDamage: 9, Undead: true}
	case monsterCinderDrake:
		return monster{Kind: kind, Name: "Cinder Drake", Glyph: 'A', HP: 27, MaxHP: 27, MinDamage: 5, MaxDamage: 9}
	case monsterVoidSeer:
		return monster{Kind: kind, Name: "Void Seer", Glyph: 'V', HP: 23, MaxHP: 23, MinDamage: 5, MaxDamage: 8, Undead: true}
	case monsterSoulLeech:
		return monster{Kind: kind, Name: "Soul Leech", Glyph: 'L', HP: 25, MaxHP: 25, MinDamage: 5, MaxDamage: 8, Undead: true}
	case monsterStormHerald:
		return monster{Kind: kind, Name: "Storm Herald", Glyph: 'T', HP: 28, MaxHP: 28, MinDamage: 6, MaxDamage: 9}
	case monsterBoss:
		return monster{Kind: kind, Name: "Dread Lich", Glyph: 'B', HP: 48, MaxHP: 48, MinDamage: 6, MaxDamage: 10, Undead: true, Boss: true}
	default:
		panic("unknown monster kind")
	}
}

func (g *game) loop(stdin io.Reader, stdout io.Writer) error {
	scanner := bufio.NewScanner(stdin)
	var rawIn *os.File
	if f, ok := stdin.(*os.File); ok {
		if restore, ok := enableRawMode(f); ok {
			rawIn = f
			defer restore()
		}
	}
	g.render(stdout)
	for !g.won && !g.quit && g.player.HP > 0 {
		var cmd string
		if rawIn != nil {
			fmt.Fprintln(stdout, "  Keys act immediately: wasd/arrows, qezx, ., </>, p f b g u t o n r, v fountain, i inventory, c codex, m inspect, k search, S save, L load, Ctrl-C quit; type :command + Enter for words")
			value, eof, err := readRawCommand(rawIn, stdout)
			if err != nil {
				return err
			}
			if eof {
				fmt.Fprintln(stdout, "\nThe dungeon waits in silence as you slip away.")
				return nil
			}
			cmd = normalizeCommand(strings.TrimSpace(strings.ToLower(value)))
		} else {
			fmt.Fprintln(stdout, "  Commands: wasd/arrows, q e z x, ., </>, p f b g u t o n r, v fountain, i inventory, c codex, m inspect, k search, save, load, quit")
			fmt.Fprint(stdout, "\nCommand> ")
			if !scanner.Scan() {
				fmt.Fprintln(stdout, "\nThe dungeon waits in silence as you slip away.")
				return nil
			}
			cmd = normalizeCommand(strings.TrimSpace(strings.ToLower(scanner.Text())))
		}
		if cmd == "" {
			cmd = "."
		}
		switch cmd {
		case "i", "inventory":
			g.renderInventory(stdout)
			continue
		case "c", "codex":
			g.renderCodex(stdout)
			continue
		case "inspect", "look", "m":
			g.renderInspect(stdout)
			continue
		}
		g.processCommand(cmd)
		if !g.quit && !g.won && g.player.HP > 0 {
			g.render(stdout)
		}
	}

	switch {
	case g.won:
		g.render(stdout)
		fmt.Fprintln(stdout, "\nYou escaped the dungeon with your ghost dog. Victory!")
	case g.player.HP <= 0:
		g.render(stdout)
		fmt.Fprintln(stdout, "\nYou fall in the dark. The Dread Lich keeps the dungeon.")
	case g.quit:
		fmt.Fprintln(stdout, "You retreat before the dungeon can claim you.")
	}
	return nil
}

func enableRawMode(f *os.File) (func(), bool) {
	fd := int(f.Fd())
	state, err := getTermios(fd)
	if err != nil {
		return nil, false
	}
	raw := *state
	raw.Lflag &^= syscall.ICANON | syscall.ECHO | syscall.ISIG
	raw.Iflag &^= syscall.ICRNL | syscall.INLCR
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if err := setTermios(fd, &raw); err != nil {
		return nil, false
	}
	return func() { _ = setTermios(fd, state) }, true
}

func getTermios(fd int) (*syscall.Termios, error) {
	termios := &syscall.Termios{}
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return termios, nil
}

func setTermios(fd int, termios *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func readRawCommand(in *os.File, out io.Writer) (string, bool, error) {
	buf := ""
	display := ""
	textMode := false
	redrawRawPrompt(out, display)
	for {
		var one [1]byte
		n, err := in.Read(one[:])
		if err != nil {
			return "", false, err
		}
		if n == 0 {
			return "", true, nil
		}
		switch one[0] {
		case '\r', '\n':
			fmt.Fprintln(out)
			return buf, false, nil
		case 3, 4:
			fmt.Fprintln(out)
			return "quit", false, nil
		case 127, 8:
			if len(buf) > 0 {
				_, size := utf8.DecodeLastRuneInString(buf)
				buf = buf[:len(buf)-size]
				display = buf
				redrawRawPrompt(out, display)
			}
		case 27:
			cmd, _ := readEscapeCommand(in)
			if cmd != "" {
				fmt.Fprintln(out)
				return cmd, false, nil
			}
		case ':':
			if !textMode && len(buf) == 0 {
				textMode = true
				display = ":"
				redrawRawPrompt(out, display)
				continue
			}
		default:
			if !textMode {
				if cmd, shown, ok := rawKeyCommand(one[0]); ok {
					fmt.Fprintf(out, "\r\x1b[2KCommand> %s\n", shown)
					return cmd, false, nil
				}
			}
			buf += string(one[0])
			display = buf
			redrawRawPrompt(out, display)
		}
	}
}

func rawKeyCommand(key byte) (string, string, bool) {
	switch key {
	case 'w', 'a', 's', 'd', 'q', 'e', 'z', 'x', '.', 'p', 'f', 'b', 'g', 'u', 't', 'o', 'n', 'r', '<', '>', 'v':
		return string(key), string(key), true
	}
	switch key {
	case 'i':
		return "inventory", "i", true
	case 'c':
		return "codex", "c", true
	case 'm':
		return "inspect", "m", true
	case 'k':
		return "search", "k", true
	case 'S':
		return "save", "S", true
	case 'L':
		return "load", "L", true
	case '?':
		return "h", "?", true
	default:
		return "", "", false
	}
}

func redrawRawPrompt(out io.Writer, display string) {
	fmt.Fprintf(out, "\r\x1b[2KCommand> %s", display)
}

func readEscapeCommand(in *os.File) (string, string) {
	seq := []byte{27}
	var one [1]byte
	for len(seq) < 6 {
		n, err := in.Read(one[:])
		if err != nil || n == 0 {
			break
		}
		seq = append(seq, one[0])
		if (one[0] >= 'A' && one[0] <= 'Z') || (one[0] >= 'a' && one[0] <= 'z') || one[0] == '~' {
			break
		}
	}
	cmd := normalizeCommand(strings.ToLower(string(seq)))
	switch cmd {
	case "w":
		return cmd, "↑"
	case "s":
		return cmd, "↓"
	case "a":
		return cmd, "←"
	case "d":
		return cmd, "→"
	case "q":
		return cmd, "↖"
	case "e":
		return cmd, "↗"
	case "z":
		return cmd, "↙"
	case "x":
		return cmd, "↘"
	default:
		return "", ""
	}
}

func (g *game) processCommand(cmd string) {
	g.turnDamage = make(map[*monster]int)
	if g.player.WebbedTurns > 0 && isMoveCommand(cmd) {
		g.player.WebbedTurns--
		g.addMessage("Sticky webs hold you in place for a turn.")
		g.advanceEnemies()
		return
	}

	acted := false
	switch cmd {
	case "w", "a", "s", "d", "q", "e", "z", "x":
		acted = g.tryMove(cmd)
	case ".":
		acted = true
		g.addMessage("You wait and let the dungeon move around you.")
	case ">":
		acted = g.tryDescend()
	case "<":
		acted = g.tryAscend()
	case "p":
		acted = g.usePotion()
	case "f":
		acted = g.useFireScroll()
	case "b":
		acted = g.useWardingCharm()
	case "g":
		acted = g.useBlinkStone()
	case "u":
		acted = g.useSunOrb()
	case "t":
		acted = g.useFrostCharm()
	case "o":
		acted = g.useStarfireOrb()
	case "n":
		acted = g.usePhoenixAsh()
	case "r":
		acted = g.useGhostRecallScroll()
	case "v":
		acted = g.drinkFromFountain()
	case "k":
		acted = g.search()
	case "search":
		acted = g.search()
	case "save":
		if err := g.save(); err != nil {
			g.addMessage("Save failed: " + err.Error())
		} else {
			g.addMessage("Game saved to " + g.saveFile)
		}
	case "load":
		if g.saveFile == "" {
			g.addMessage("No save file is selected; restart with --load-file path to load a run.")
			break
		}
		loaded, err := loadGame(g.saveFile)
		if err != nil {
			g.addMessage("Load failed: " + err.Error())
		} else {
			*g = *loaded
			g.dogFocus = nil
			g.addMessage("Game loaded from " + g.saveFile)
		}
	case "h":
		g.addMessage("Commands act on one key: wasd/arrows move, q/e/z/x diagonals, . wait, </> stairs, p/f/b/g/u/t/o/n items, r recall dog, v drink fountain, i inventory, c codex, m inspect, k search, S save, L load; Ctrl-C quits. Prefix a word with : and press Enter.")
	case "quit", "exit":
		g.quit = true
	default:
		g.addMessage("Unknown command. Press h for help.")
	}

	if acted && !g.won && g.player.HP > 0 {
		g.advanceEnemies()
	}
}

func normalizeCommand(cmd string) string {
	switch cmd {
	case "[a", "up":
		return "w"
	case "[b", "down":
		return "s"
	case "[d", "left":
		return "a"
	case "[c", "right":
		return "d"
	case "[h", "[1~", "oh", "home":
		return "q"
	case "[5~", "pgup", "pg up", "pageup":
		return "e"
	case "[f", "[4~", "[8~", "of", "end":
		return "z"
	case "[6~", "pgdn", "pg dn", "pagedown":
		return "x"
	default:
		return cmd
	}
}

func isMoveCommand(cmd string) bool {
	switch cmd {
	case "w", "a", "s", "d", "q", "e", "z", "x":
		return true
	default:
		return false
	}
}

func moveDelta(cmd string) (int, int, bool) {
	switch normalizeCommand(cmd) {
	case "w":
		return 0, -1, true
	case "s":
		return 0, 1, true
	case "a":
		return -1, 0, true
	case "d":
		return 1, 0, true
	case "q":
		return -1, -1, true
	case "e":
		return 1, -1, true
	case "z":
		return -1, 1, true
	case "x":
		return 1, 1, true
	default:
		return 0, 0, false
	}
}

func (g *game) tryMove(cmd string) bool {
	dx, dy, ok := moveDelta(cmd)
	if !ok {
		return false
	}
	next := pos{X: g.player.Pos.X + dx, Y: g.player.Pos.Y + dy}
	lvl := g.current()
	if !g.inBounds(next) || lvl.Tiles[next.Y][next.X] == '#' {
		g.addMessage("Stone blocks your path.")
		return false
	}
	if m := g.monsterAt(lvl, next); m != nil {
		g.attackMonster(m)
		return true
	}
	if g.dog.Freed && g.dog.Alive && g.dog.Pos == next {
		if !g.stepDogAside(next, g.player.Pos) {
			g.addMessage("Ghost dog has no space to drift aside.")
			return false
		}
		g.player.Pos = next
		g.addMessage("Ghost dog drifts aside and keeps pace with you.")
		g.collectItems()
		return true
	}
	g.player.Pos = next
	if lvl.DogChain != nil && *lvl.DogChain == next && !g.dog.Freed {
		g.dog.Freed = true
		g.placeDogNearPlayer()
		g.addMessage("You free the ghost dog from its silver chain. It chooses you immediately.")
	}
	g.collectItems()
	return true
}

func (g *game) stepDogAside(target, fallback pos) bool {
	candidates := []pos{fallback}
	for _, p := range g.openNeighbors(target, true) {
		candidates = append(candidates, p)
	}
	seen := map[pos]bool{}
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		seen[p] = true
		if g.canDogOccupyAfterPlayerMove(p, target) {
			g.dog.Pos = p
			return true
		}
	}
	return false
}

func (g *game) canDogOccupyAfterPlayerMove(p, futurePlayer pos) bool {
	if !g.inBounds(p) || g.current().Tiles[p.Y][p.X] == '#' || p == futurePlayer {
		return false
	}
	return g.monsterAt(g.current(), p) == nil
}

func (g *game) search() bool {
	lvl := g.current()
	if lvl.Secret == nil || lvl.Secret.Revealed {
		g.addMessage("You search the walls, but find no hidden seams.")
		return true
	}
	if distance(g.player.Pos, lvl.Secret.Door) > 1 {
		g.addMessage("You search nearby stone, but uncover nothing unusual.")
		return true
	}
	if lvl.HasEscapeStair && g.bossAlive(lvl) {
		g.addMessage("Ancient death-magic locks the hidden seam tight. The Lich still binds it.")
		return true
	}
	g.revealSecretRoom(lvl)
	g.addMessage("Your search finds a hollow seam. A secret door swings open.")
	return true
}

func (g *game) revealSecretRoom(lvl *level) {
	if lvl.Secret == nil || lvl.Secret.Revealed {
		return
	}
	lvl.Secret.Revealed = true
	lvl.Tiles[lvl.Secret.Door.Y][lvl.Secret.Door.X] = '.'
	for _, p := range lvl.Secret.Tiles {
		lvl.Tiles[p.Y][p.X] = '.'
	}
}

func (g *game) tryDescend() bool {
	lvl := g.current()
	if !lvl.HasStair || g.player.Pos != lvl.Stairs {
		g.addMessage("There are no stairs beneath your feet.")
		return false
	}
	g.currentLevel++
	nextLevel := g.current()
	g.player.Pos = nextLevel.UpStairs
	firstVisit := !nextLevel.Visited
	nextLevel.Visited = true
	g.afterFloorTransition(fmt.Sprintf("You descend to level %d.", g.currentLevel+1), firstVisit)
	g.collectItems()
	return true
}

func (g *game) tryAscend() bool {
	lvl := g.current()
	if lvl.HasEscapeStair && lvl.Secret != nil && lvl.Secret.Revealed && g.player.Pos == lvl.EscapeStairs {
		g.won = true
		g.addMessage("You climb the hidden stairs and leave the dungeon behind.")
		return true
	}
	if !lvl.HasUpStair || g.player.Pos != lvl.UpStairs {
		g.addMessage("There are no stairs leading up here.")
		return false
	}
	g.currentLevel--
	prevLevel := g.current()
	prevLevel.Visited = true
	g.player.Pos = prevLevel.Stairs
	g.afterFloorTransition(fmt.Sprintf("You climb back to level %d.", g.currentLevel+1), false)
	g.regenerateStairGuardians(prevLevel)
	g.collectItems()
	return true
}

func (g *game) afterFloorTransition(message string, healDog bool) {
	if g.dog.Freed && g.dog.Alive {
		if healDog {
			heal := g.randRange(2, 3)
			g.dog.HP = min(g.dog.MaxHP, g.dog.HP+heal)
			g.addMessage(fmt.Sprintf("Ghost dog recovers %d health on this new depth. (%d/%d)", heal, g.dog.HP, g.dog.MaxHP))
		}
		g.placeDogNearPlayer()
	}
	g.addMessage(message)
}

func (g *game) regenerateStairGuardians(lvl *level) {
	if !lvl.HasStair {
		return
	}
	want := 2 + g.rng.Intn(2)
	spots := make([]pos, 0, want)
	for radius := 1; radius <= 2 && len(spots) < want; radius++ {
		for y := lvl.Stairs.Y - radius; y <= lvl.Stairs.Y+radius && len(spots) < want; y++ {
			for x := lvl.Stairs.X - radius; x <= lvl.Stairs.X+radius && len(spots) < want; x++ {
				p := pos{X: x, Y: y}
				if distance(p, lvl.Stairs) != radius || !g.inBounds(p) || lvl.Tiles[y][x] == '#' || p == g.player.Pos ||
					(!g.dog.Freed && lvl.DogChain != nil && p == *lvl.DogChain) {
					continue
				}
				if g.monsterAt(lvl, p) != nil || g.itemAt(lvl, p) != nil || (g.dog.Freed && g.dog.Alive && g.dog.Pos == p) {
					continue
				}
				spots = append(spots, p)
			}
		}
	}
	pool := monsterPoolForLevel(lvl.Index)
	if len(spots) == 0 || len(pool) == 0 {
		return
	}
	count := min(want, len(spots))
	for i := 0; i < count; i++ {
		spotIndex := g.rng.Intn(len(spots))
		spawnPos := spots[spotIndex]
		spots = append(spots[:spotIndex], spots[spotIndex+1:]...)
		spec := pool[g.rng.Intn(len(pool))]
		for spec.Boss && len(pool) > 1 {
			spec = pool[g.rng.Intn(len(pool))]
		}
		spawn := scaleMonsterForDepth(spec, lvl.Index)
		spawn.Pos = spawnPos
		lvl.Monsters = append(lvl.Monsters, &spawn)
	}
	g.addMessage("Stairwell guardians stir behind you as you return.")
}

func (g *game) usePotion() bool {
	if g.player.Potions == 0 {
		g.addMessage("You have no healing potions.")
		return false
	}
	g.player.Potions--
	heal := g.randRange(12, 18)
	g.player.HP += heal
	if g.player.HP > g.player.MaxHP {
		g.player.HP = g.player.MaxHP
	}
	g.addMessage(fmt.Sprintf("You drink a potion and recover %d health.", heal))
	return true
}

func (g *game) useFireScroll() bool {
	if g.player.FireScrolls == 0 {
		g.addMessage("You have no fire scrolls.")
		return false
	}
	m := g.nearestMonster(6)
	if m == nil {
		g.addMessage("No monster is close enough for the fire scroll.")
		return false
	}
	g.player.FireScrolls--
	damage := g.randRange(10, 16)
	g.addMessage(fmt.Sprintf("The fire scroll explodes around %s for %d damage.", m.Name, damage))
	g.damageMonster(m, damage, "burns")
	for _, other := range g.current().Monsters {
		if other != m && other.HP > 0 && distance(other.Pos, m.Pos) == 1 {
			splash := g.randRange(3, 6)
			g.damageMonster(other, splash, "scorches")
		}
	}
	return true
}

func (g *game) useWardingCharm() bool {
	if g.player.WardingCharms == 0 {
		g.addMessage("You have no warding charms.")
		return false
	}
	g.player.WardingCharms--
	g.player.ShieldTurns = 3
	g.addMessage("A pale ward circles you for the next few enemy turns.")
	return true
}

func (g *game) useBlinkStone() bool {
	if g.player.BlinkStones == 0 {
		g.addMessage("You have no blink stones.")
		return false
	}
	targets := g.openNeighbors(g.player.Pos, false)
	if len(targets) == 0 {
		g.addMessage("The blink stone fizzles in the cramped corridor.")
		return false
	}
	g.player.BlinkStones--
	g.player.Pos = targets[g.rng.Intn(len(targets))]
	g.addMessage("Space folds and drops you a few steps away.")
	g.collectItems()
	return true
}

func (g *game) useSunOrb() bool {
	if g.player.SunOrbs == 0 {
		g.addMessage("You have no sun orbs.")
		return false
	}
	g.player.SunOrbs--
	hit := 0
	for _, m := range g.current().Monsters {
		if m.HP > 0 && m.Undead {
			dmg := g.randRange(9, 14)
			g.damageMonster(m, dmg, "sears")
			hit++
		}
	}
	if hit == 0 {
		g.addMessage("The sun orb flashes, but no undead are here.")
	} else {
		g.addMessage(fmt.Sprintf("Sunfire crashes through %d undead foe(s).", hit))
	}
	return true
}

func (g *game) useFrostCharm() bool {
	if g.player.FrostCharms == 0 {
		g.addMessage("You have no frost charms.")
		return false
	}
	target := g.nearestMonster(6)
	if target == nil {
		g.addMessage("The frost charm finds no nearby target.")
		return false
	}
	g.player.FrostCharms--
	damage := g.randRange(7, 11)
	target.FrozenTurns = 2
	g.addMessage(fmt.Sprintf("The frost charm locks %s in rime for %d damage.", target.Name, damage))
	g.damageMonster(target, damage, "freezes")
	return true
}

func (g *game) useStarfireOrb() bool {
	if g.player.StarfireOrbs == 0 {
		g.addMessage("You have no starfire orbs.")
		return false
	}
	target := g.nearestMonster(6)
	if target == nil {
		g.addMessage("The starfire orb finds no nearby target.")
		return false
	}
	g.player.StarfireOrbs--
	damage := g.randRange(18, 24)
	g.addMessage(fmt.Sprintf("Starfire crashes into %s for %d damage.", target.Name, damage))
	g.damageMonster(target, damage, "sears")
	return true
}

func (g *game) usePhoenixAsh() bool {
	if g.player.PhoenixAshes == 0 {
		g.addMessage("You have no phoenix ash.")
		return false
	}
	if g.player.HP >= g.player.MaxHP {
		g.addMessage("You are already at full health; save the phoenix ash.")
		return false
	}
	g.player.PhoenixAshes--
	heal := g.randRange(22, 30)
	g.player.HP = min(g.player.MaxHP, g.player.HP+heal)
	g.addMessage(fmt.Sprintf("Phoenix ash restores your health. (%d/%d)", g.player.HP, g.player.MaxHP))
	return true
}

func (g *game) useGhostRecallScroll() bool {
	if g.player.GhostRecallScrolls == 0 {
		g.addMessage("You have no Ghost Dog recall scrolls.")
		return false
	}
	if !g.dog.Freed {
		g.addMessage("The scroll cannot call a Ghost Dog you have not freed.")
		return false
	}
	if g.dog.Alive {
		g.addMessage("The Ghost Dog is already at your side.")
		return false
	}
	g.dog.Alive = true
	if !g.placeDogNearPlayer() {
		g.dog.Alive = false
		g.addMessage("The scroll finds no safe place for the Ghost Dog to return.")
		return false
	}
	g.player.GhostRecallScrolls--
	g.dog.HP = g.dog.MaxHP
	g.addMessage("The recall scroll calls the Ghost Dog back to your side at full hit points.")
	return true
}

func (g *game) drinkFromFountain() bool {
	var nearby *fountain
	for i := range g.current().Fountains {
		f := &g.current().Fountains[i]
		if !f.Used && distance(g.player.Pos, f.Pos) <= 1 {
			nearby = f
			break
		}
	}
	if nearby == nil {
		g.addMessage("There is no unused fountain close enough to drink from.")
		return false
	}
	if g.player.HP >= g.player.MaxHP && (!g.dog.Freed || !g.dog.Alive || g.dog.HP >= g.dog.MaxHP) {
		g.addMessage("You and Ghost Dog are already at full health; save the fountain.")
		return false
	}
	nearby.Used = true
	playerHeal := g.randRange(4, 8)
	oldHP := g.player.HP
	g.player.HP = min(g.player.MaxHP, g.player.HP+playerHeal)
	actualPlayerHeal := g.player.HP - oldHP
	if g.dog.Freed && g.dog.Alive {
		dogHeal := g.randRange(4, 8)
		oldDogHP := g.dog.HP
		g.dog.HP = min(g.dog.MaxHP, g.dog.HP+dogHeal)
		g.addMessage(fmt.Sprintf("The fountain restores %d health to Ghost Dog. (%d/%d)", g.dog.HP-oldDogHP, g.dog.HP, g.dog.MaxHP))
	}
	g.addMessage(fmt.Sprintf("You drink from the fountain and recover %d health. (%d/%d)", actualPlayerHeal, g.player.HP, g.player.MaxHP))
	return true
}

func (g *game) attackMonster(m *monster) {
	damage := g.randRange(g.player.Weapon.Min, g.player.Weapon.Max)
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
	if m.HP > 0 {
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
	g.knownMonsters[m.Kind] = true
	if g.turnDamage == nil {
		g.turnDamage = make(map[*monster]int)
	}
	dealt := min(max(damage, 0), m.HP)
	m.HP -= damage
	g.turnDamage[m] += dealt
	if m.HP > 0 {
		g.addMessage(fmt.Sprintf("%s %s. (%d/%d)", capitalizeMonsterVerb(m.Name, verb), hpState(verb), m.HP, m.MaxHP))
		return
	}
	m.HP = 0
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

func (g *game) restoreGhostDogAfterLichDeath() {
	if !g.dog.Freed || g.dog.Alive {
		return
	}
	g.dog.Alive = true
	g.dog.HP = g.dog.MaxHP
	g.placeDogNearPlayer()
	g.addMessage("The Dread Lich's death frees the Ghost Dog. It reappears at full hit points.")
}

func capitalizeMonsterVerb(name, verb string) string {
	switch verb {
	case "burns":
		return name + " burns"
	case "scorches":
		return name + " scorches"
	case "sears":
		return name + " sears"
	case "freezes":
		return name + " freezes"
	default:
		return name + " reels"
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

func (g *game) dogTurn() {
	if !g.dog.Alive {
		return
	}
	if focus := g.activeDogFocus(); focus != nil {
		if distance(g.dog.Pos, focus.Pos) == 1 {
			g.dogAttack(focus)
			return
		}
		g.moveDogToward(focus.Pos)
		if distance(g.dog.Pos, focus.Pos) == 1 {
			g.dogAttack(focus)
		}
		return
	}
	if target := g.adjacentMonster(g.dog.Pos); target != nil {
		g.dogAttack(target)
		return
	}
	g.keepDogWithPlayer()
}

func (g *game) dogAttack(target *monster) {
	dmg := g.randRange(6, 9)
	g.addMessage(fmt.Sprintf("Ghost dog tears into %s for %d damage.", target.Name, dmg))
	g.damageMonster(target, dmg, "mauls")
}

func (g *game) activeDogFocus() *monster {
	if g.dogFocus != nil && g.dogFocus.HP > 0 && distance(g.player.Pos, g.dogFocus.Pos) <= 2 {
		return g.dogFocus
	}
	if target := g.adjacentMonster(g.player.Pos); target != nil {
		g.dogFocus = target
		return target
	}
	g.dogFocus = nil
	return nil
}

func (g *game) keepDogWithPlayer() {
	if g.dog.Pos == g.player.Pos || g.monsterAt(g.current(), g.dog.Pos) != nil {
		if g.placeDogNearPlayer() {
			return
		}
	}
	if distance(g.dog.Pos, g.player.Pos) <= 1 {
		return
	}
	best := g.bestDogNeighborNearPlayer()
	if best != nil {
		g.dog.Pos = *best
		return
	}
	g.moveActorToward(&g.dog.Pos, g.player.Pos, true)
}

func (g *game) moveDogToward(target pos) {
	if distance(g.dog.Pos, g.player.Pos) > 2 {
		if best := g.bestDogNeighborNearPlayer(); best != nil {
			g.dog.Pos = *best
			if distance(g.dog.Pos, target) <= 1 {
				return
			}
		}
	}
	g.moveActorToward(&g.dog.Pos, target, true)
	if distance(g.dog.Pos, g.player.Pos) > 2 {
		g.keepDogWithPlayer()
	}
}

func (g *game) bestDogNeighborNearPlayer() *pos {
	spots := g.openNeighbors(g.player.Pos, true)
	if len(spots) == 0 {
		return nil
	}
	best := spots[0]
	bestScore := 999999
	for _, p := range spots {
		if !g.canDogOccupyAfterPlayerMove(p, g.player.Pos) {
			continue
		}
		score := distance(p, g.dog.Pos)*10 + distance(p, g.player.Pos) + g.doorwayPenalty(p)
		if score < bestScore {
			best = p
			bestScore = score
		}
	}
	if bestScore == 999999 {
		return nil
	}
	return &best
}

func (g *game) placeDogNearPlayer() bool {
	if best := g.bestDogNeighborNearPlayer(); best != nil {
		g.dog.Pos = *best
		return true
	}
	lvl := g.current()
	best := pos{}
	bestDistance := math.MaxInt
	found := false
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{X: x, Y: y}
			if lvl.Tiles[y][x] == '#' || p == g.player.Pos || g.monsterAt(lvl, p) != nil {
				continue
			}
			if d := distance(p, g.player.Pos); d < bestDistance {
				best, bestDistance, found = p, d, true
			}
		}
	}
	if found {
		g.dog.Pos = best
	}
	return found
}

func (g *game) doorwayPenalty(p pos) int {
	orth := 0
	for _, delta := range []pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		n := pos{X: p.X + delta.X, Y: p.Y + delta.Y}
		if g.inBounds(n) && g.current().Tiles[n.Y][n.X] != '#' {
			orth++
		}
	}
	if orth <= 2 {
		return 8
	}
	return 0
}

func (g *game) monsterTurn(m *monster) {
	g.knownMonsters[m.Kind] = true
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
	if g.dog.Freed && g.dog.Alive && distance(m.Pos, g.dog.Pos) == 1 && distance(m.Pos, g.player.Pos) > 1 {
		g.attackDog(m)
		return
	}
	if distance(m.Pos, g.player.Pos) == 1 {
		g.attackPlayer(m)
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
			g.hurtPlayer(2, "The Hex Priest whispers a crooked curse")
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
			g.hurtPlayer(g.randRange(3, 6), "The Cinder Drake breathes a cone of embers")
			return
		}
	case monsterVoidSeer:
		if distance(m.Pos, g.player.Pos) <= 5 && g.rng.Intn(100) < 28 {
			g.player.HexedTurns = 1
			g.hurtPlayer(g.randRange(2, 5), "The Void Seer tears at your thoughts")
			return
		}
	case monsterSoulLeech:
		if distance(m.Pos, g.player.Pos) <= 4 && g.rng.Intn(100) < 30 {
			g.hurtPlayer(g.randRange(3, 6), "The Soul Leech siphons you from a distance")
			m.HP = min(m.MaxHP, m.HP+2)
			g.addMessage("The Soul Leech drinks in the stolen vitality.")
			return
		}
	case monsterStormHerald:
		if distance(m.Pos, g.player.Pos) <= 4 && g.rng.Intn(100) < 30 {
			g.hurtPlayer(g.randRange(3, 6), "The Storm Herald calls lightning down on you")
			if g.dog.Freed && g.dog.Alive && distance(m.Pos, g.dog.Pos) <= 4 {
				g.hurtDog(g.randRange(3, 6), "Lightning from the Storm Herald")
			}
			return
		}
	case monsterBoss:
		if distance(m.Pos, g.player.Pos) <= 5 && g.rng.Intn(100) < 30 {
			g.player.HexedTurns = 1
			g.hurtPlayer(g.randRange(5, 8), "The Dread Lich hurls a bolt of grave-fire")
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
		g.addMessage(fmt.Sprintf("%s scrambles away from you and the Ghost Dog.", m.Name))
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
	g.dogFocus = m
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
	g.dog.HP -= damage
	g.addMessage(fmt.Sprintf("%s hits the ghost dog for %d damage. (%d/%d)", source, damage, max(g.dog.HP, 0), g.dog.MaxHP))
	if g.dog.HP <= 0 {
		g.dog.HP = 0
		g.dog.Alive = false
		g.dogFocus = nil
		g.addMessage("The Ghost Dog can only be freed now by killing the Dread Lich")
	}
}

func (g *game) collectItems() {
	lvl := g.current()
	kept := lvl.Items[:0]
	for _, it := range lvl.Items {
		if it.Pos != g.player.Pos {
			kept = append(kept, it)
			continue
		}
		switch it.Kind {
		case itemWeapon:
			cmp := compareWeapon(it.Weapon, g.player.Weapon)
			if weaponScore(it.Weapon) > weaponScore(g.player.Weapon) {
				g.player.Weapon = it.Weapon
				g.addMessage(fmt.Sprintf("You equip the %s (%d-%d). %s %s", it.Weapon.Name, it.Weapon.Min, it.Weapon.Max, cmp, weaponDescription(it.Weapon)))
			} else {
				g.addMessage(fmt.Sprintf("You find the %s (%d-%d), but keep %s. %s %s", it.Weapon.Name, it.Weapon.Min, it.Weapon.Max, g.player.Weapon.Name, cmp, weaponDescription(it.Weapon)))
			}
		case itemArmor:
			equipped := g.player.armorForSlot(it.Armor.Slot)
			cmp := compareArmor(it.Armor, equipped)
			candidateName := coloredArmorName(it.Armor)
			currentName := coloredArmorName(equipped)
			if it.Armor.Defense > equipped.Defense {
				g.player.setArmor(it.Armor)
				g.addMessage(fmt.Sprintf("You equip %s on your %s slot. %s %s", candidateName, slotLabel(it.Armor.Slot), cmp, armorDescription(it.Armor)))
			} else {
				g.addMessage(fmt.Sprintf("You find %s for your %s slot, but keep %s. %s %s", candidateName, slotLabel(it.Armor.Slot), currentName, cmp, armorDescription(it.Armor)))
			}
		case itemPotion:
			g.player.Potions++
			g.addMessage("You stash a healing potion. Restores 12-18 health when used with p.")
		case itemFireScroll:
			g.player.FireScrolls++
			g.addMessage("You pocket a fire scroll. It detonates near the closest foe when used with f.")
		case itemBlinkStone:
			g.player.BlinkStones++
			g.addMessage("You take a blink stone. Use g to teleport to a nearby open tile.")
		case itemWardingCharm:
			g.player.WardingCharms++
			g.addMessage("You take a warding charm. Use b to soften incoming blows for 3 enemy turns.")
		case itemSunOrb:
			g.player.SunOrbs++
			g.addMessage("You cradle a sun orb. Use u to burn every undead monster on this floor.")
		case itemFrostCharm:
			g.player.FrostCharms++
			g.addMessage("You take a frost charm. Use t to freeze and wound the nearest foe.")
		case itemStarfireOrb:
			g.player.StarfireOrbs++
			g.addMessage("You take a starfire orb. Use o to blast the nearest foe for heavy damage.")
		case itemPhoenixAsh:
			g.player.PhoenixAshes++
			g.addMessage("You gather phoenix ash. Use n to restore a large amount of health.")
		case itemGhostRecallScroll:
			g.player.GhostRecallScrolls++
			g.addMessage("You find a Ghost Dog recall scroll. Use r to call your fallen companion back at full health.")
		case itemStoryScroll:
			if it.StoryChapter >= 1 && it.StoryChapter <= len(ghostDogStory) {
				seen := false
				for _, chapter := range g.player.StoryChapters {
					if chapter == it.StoryChapter {
						seen = true
						break
					}
				}
				if !seen {
					g.player.StoryChapters = append(g.player.StoryChapters, it.StoryChapter)
				}
				g.addMessage(fmt.Sprintf("Ghost Dog story scroll %d/10: %s", it.StoryChapter, ghostDogStory[it.StoryChapter-1]))
			}
		}
	}
	lvl.Items = kept
}

func (g *game) save() error {
	if g.timestampedSaves || g.saveFile == "" {
		g.saveFile = nextTimestampedSaveFile(time.Now())
	}
	state := saveState{RNGState: g.rng.State, Seed: g.seed, SaveFile: g.saveFile, Levels: g.levels, CurrentLevel: g.currentLevel, Player: g.player, Dog: g.dog, Messages: append([]string(nil), g.messages...), KnownMonsters: append([]monsterKind(nil), g.knownMonsterKinds()...), Won: g.won, Quit: g.quit, LevelTiles: make([]string, len(g.levels))}
	for i, lvl := range g.levels {
		state.LevelTiles[i] = encodeTiles(lvl.Tiles)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(g.saveFile, data, 0o644)
}

func nextTimestampedSaveFile(now time.Time) string {
	base := defaultSavePrefix + "-" + now.Format("20060102-150405")
	path := base + ".json"
	for suffix := 1; ; suffix++ {
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			return path
		}
		if err != nil {
			return path
		}
		path = fmt.Sprintf("%s-%02d.json", base, suffix)
	}
}

func loadGame(path string) (*game, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state saveState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if len(state.Levels) != levelCount || len(state.LevelTiles) != levelCount {
		return nil, errors.New("save file is missing dungeon levels")
	}
	if state.CurrentLevel < 0 || state.CurrentLevel >= levelCount {
		return nil, fmt.Errorf("save file has invalid current level %d", state.CurrentLevel+1)
	}
	for i, lvl := range state.Levels {
		tiles, err := decodeTiles(state.LevelTiles[i])
		if err != nil {
			return nil, err
		}
		lvl.Tiles = tiles
	}
	known := map[monsterKind]bool{}
	for _, kind := range state.KnownMonsters {
		known[kind] = true
	}
	g := &game{rng: &simpleRNG{State: state.RNGState}, seed: state.Seed, saveFile: path, levels: state.Levels, currentLevel: state.CurrentLevel, player: state.Player, dog: state.Dog, messages: append([]string(nil), state.Messages...), knownMonsters: known, won: state.Won}
	if g.player.HeadArmor.Name == "" {
		g.player.HeadArmor = armor{Name: "Padded Hood", Slot: slotHead, Defense: 1, Rarity: rarityCommon}
	}
	if g.player.BodyArmor.Name == "" {
		g.player.BodyArmor = armor{Name: "Worn Coat", Slot: slotBody, Defense: 1, Rarity: rarityCommon}
	}
	if g.player.FeetArmor.Name == "" {
		g.player.FeetArmor = armor{Name: "Frayed Boots", Slot: slotFeet, Defense: 0, Rarity: rarityCommon}
	}
	if g.player.HeadArmor.Rarity == "" {
		g.player.HeadArmor.Rarity = rarityCommon
	}
	if g.player.BodyArmor.Rarity == "" {
		g.player.BodyArmor.Rarity = rarityCommon
	}
	if g.player.FeetArmor.Rarity == "" {
		g.player.FeetArmor.Rarity = rarityCommon
	}
	if g.knownMonsters == nil {
		g.knownMonsters = map[monsterKind]bool{}
	}
	if g.currentLevel >= 0 && g.currentLevel < len(g.levels) && g.levels[g.currentLevel] != nil {
		g.levels[g.currentLevel].Visited = true
	}
	if countFountains(g.levels) == 0 {
		g.placeFountains()
	}
	g.repairActorPositions()
	return g, nil
}

func countFountains(levels []*level) int {
	count := 0
	for _, lvl := range levels {
		if lvl != nil {
			count += len(lvl.Fountains)
		}
	}
	return count
}

func (g *game) repairActorPositions() {
	for levelIndex, lvl := range g.levels {
		if lvl == nil {
			continue
		}
		for _, m := range lvl.Monsters {
			if m.HP <= 0 {
				continue
			}
			if m.Boss && m.StoryChapter == 0 {
				m.StoryChapter = levelCount
			}
			blocked := lvl.DogChain != nil && m.Pos == *lvl.DogChain && !g.dog.Freed
			if levelIndex == g.currentLevel {
				blocked = blocked || m.Pos == g.player.Pos || (g.dog.Freed && g.dog.Alive && m.Pos == g.dog.Pos)
			}
			if !blocked {
				continue
			}
			best := pos{}
			bestDistance := math.MaxInt
			found := false
			for y := 1; y < mapHeight-1; y++ {
				for x := 1; x < mapWidth-1; x++ {
					p := pos{X: x, Y: y}
					if lvl.Tiles[y][x] == '#' || p == m.Pos || (!g.dog.Freed && lvl.DogChain != nil && p == *lvl.DogChain) {
						continue
					}
					if levelIndex == g.currentLevel && (p == g.player.Pos || (g.dog.Freed && g.dog.Alive && p == g.dog.Pos)) {
						continue
					}
					occupied := false
					for _, other := range lvl.Monsters {
						if other != m && other.HP > 0 && other.Pos == p {
							occupied = true
							break
						}
					}
					if occupied {
						continue
					}
					if d := distance(p, m.Pos); d < bestDistance {
						best, bestDistance, found = p, d, true
					}
				}
			}
			if found {
				m.Pos = best
			}
		}
	}
	if g.dog.Freed && g.dog.Alive {
		lvl := g.current()
		if g.dog.Pos == g.player.Pos || g.monsterAt(lvl, g.dog.Pos) != nil || !g.inBounds(g.dog.Pos) || lvl.Tiles[g.dog.Pos.Y][g.dog.Pos.X] == '#' {
			g.placeDogNearPlayer()
		}
	}
}

func encodeTiles(tiles [][]rune) string {
	lines := make([]string, len(tiles))
	for i := range tiles {
		lines[i] = string(tiles[i])
	}
	return strings.Join(lines, "\n")
}

func decodeTiles(raw string) ([][]rune, error) {
	lines := strings.Split(raw, "\n")
	if len(lines) != mapHeight {
		return nil, fmt.Errorf("invalid map height %d", len(lines))
	}
	tiles := make([][]rune, len(lines))
	for i, line := range lines {
		runes := []rune(line)
		if len(runes) != mapWidth {
			return nil, fmt.Errorf("invalid map width %d", len(runes))
		}
		tiles[i] = runes
	}
	return tiles, nil
}

func (g *game) render(w io.Writer) {
	g.noteVisibleMonsters()
	lvl := g.current()
	fmt.Fprintf(w, "\n=== Ghost Dog Dungeon: Level %d/%d ===\n", g.currentLevel+1, levelCount)
	fmt.Fprintf(w, "HP %d/%d  Weapon %s (%d-%d)  Armor %d (block %d)  Potions %d  Fire %d  Blink %d  Ward %d  Sun %d  Frost %d  Starfire %d  Ash %d  Recall %d  Story %d/10\n", g.player.HP, g.player.MaxHP, g.player.Weapon.Name, g.player.Weapon.Min, g.player.Weapon.Max, g.player.armorDefense(), g.player.armorBlock(), g.player.Potions, g.player.FireScrolls, g.player.BlinkStones, g.player.WardingCharms, g.player.SunOrbs, g.player.FrostCharms, g.player.StarfireOrbs, g.player.PhoenixAshes, g.player.GhostRecallScrolls, len(g.player.StoryChapters))
	if g.dog.Freed {
		status := "gone"
		if g.dog.Alive {
			status = fmt.Sprintf("%d/%d HP", g.dog.HP, g.dog.MaxHP)
		}
		fmt.Fprintf(w, "Ghost dog: adopted (%s)\n", status)
	} else {
		fmt.Fprintln(w, "Ghost dog: not yet freed")
	}
	if g.player.ShieldTurns > 0 || g.player.HexedTurns > 0 || g.player.WebbedTurns > 0 {
		fmt.Fprintf(w, "Effects: ward %d  hex %d  web %d\n", g.player.ShieldTurns, g.player.HexedTurns, g.player.WebbedTurns)
	}
	for y := 0; y < mapHeight; y++ {
		var line strings.Builder
		for x := 0; x < mapWidth; x++ {
			p := pos{X: x, Y: y}
			ch := lvl.Tiles[y][x]
			m := g.monsterAt(lvl, p)
			f := g.fountainAt(lvl, p)
			it := g.itemAt(lvl, p)
			switch {
			case g.player.Pos == p:
				ch = '@'
			case m != nil:
				ch = m.Glyph
			case g.dog.Freed && g.dog.Alive && g.dog.Pos == p:
				ch = 'D'
			case lvl.DogChain != nil && *lvl.DogChain == p && !g.dog.Freed:
				ch = 'd'
			case lvl.HasStair && lvl.Stairs == p:
				ch = '>'
			case lvl.HasUpStair && lvl.UpStairs == p:
				ch = '<'
			case lvl.HasEscapeStair && lvl.EscapeStairs == p && (lvl.Secret == nil || lvl.Secret.Revealed):
				ch = '<'
			case f != nil && !f.Used:
				ch = 'F'
			case f != nil:
				ch = 'f'
			case ch != '#' && it != nil:
				ch = it.Glyph
			default:
				// Keep the terrain glyph.
			}
			line.WriteRune(ch)
		}
		fmt.Fprintln(w, line.String())
	}
	fmt.Fprintln(w, "Legend: @ you  D ghost dog  d chained ghost dog  > down  < up/out  F/f fountain/spent  ) weapon  [ armor  ! potion  ? scroll  * magic")
	if note := g.contextHint(); note != "" {
		fmt.Fprintln(w, "Here: "+note)
	}
	fmt.Fprintln(w, "Recent events:")
	for _, msg := range g.messages {
		fmt.Fprintln(w, "- "+msg)
	}
}

func (g *game) renderInventory(w io.Writer) {
	fmt.Fprintln(w, "\n=== Inventory & Equipment ===")
	fmt.Fprintf(w, "Weapon: %s (%d-%d)\n  %s\n", g.player.Weapon.Name, g.player.Weapon.Min, g.player.Weapon.Max, weaponDescription(g.player.Weapon))
	fmt.Fprintf(w, "Head: %s (+%d guard, %s)\n  %s\n", coloredArmorName(g.player.HeadArmor), g.player.HeadArmor.Defense, rarityLabel(g.player.HeadArmor.Rarity), armorDescription(g.player.HeadArmor))
	fmt.Fprintf(w, "Body: %s (+%d guard, %s)\n  %s\n", coloredArmorName(g.player.BodyArmor), g.player.BodyArmor.Defense, rarityLabel(g.player.BodyArmor.Rarity), armorDescription(g.player.BodyArmor))
	fmt.Fprintf(w, "Feet: %s (+%d guard, %s)\n  %s\n", coloredArmorName(g.player.FeetArmor), g.player.FeetArmor.Defense, rarityLabel(g.player.FeetArmor.Rarity), armorDescription(g.player.FeetArmor))
	fmt.Fprintf(w, "Total armor: %d guard, blocking %d damage from each hit before wards.\n", g.player.armorDefense(), g.player.armorBlock())
	fmt.Fprintln(w, "Consumables:")
	for _, line := range g.inventoryLines() {
		fmt.Fprintf(w, "- %s x%d\n  %s\n", line.name, line.count, line.description)
	}
	companion := "The ghost dog is still chained somewhere below."
	if g.dog.Freed && g.dog.Alive {
		companion = fmt.Sprintf("Ghost dog companion: fighting at %d/%d HP and will attack nearby monsters.", g.dog.HP, g.dog.MaxHP)
	} else if g.dog.Freed {
		companion = "Ghost dog companion: dispersed into silver mist; defeating the Dread Lich will restore it at full health."
	}
	fmt.Fprintln(w, "Companion:")
	fmt.Fprintln(w, "- "+companion)
	fmt.Fprintln(w, "Tip: press c for the codex or m to inspect nearby threats and loot.")
}

func (g *game) renderCodex(w io.Writer) {
	g.noteVisibleMonsters()
	fmt.Fprintln(w, "\n=== Enemy Codex ===")
	known := g.knownMonsterKinds()
	if len(known) == 0 {
		fmt.Fprintln(w, "You have not survived long enough to study anything yet.")
		return
	}
	for _, kind := range known {
		m := newMonster(kind)
		fmt.Fprintf(w, "- %c %s  HP %d  Damage %d-%d\n", m.Glyph, m.Name, m.MaxHP, m.MinDamage, m.MaxDamage)
		fmt.Fprintln(w, "  "+monsterSummary(kind))
		fmt.Fprintln(w, "  Trick: "+monsterPower(kind))
	}
	fmt.Fprintln(w, "Tip: press i to review your equipment and consumables.")
}

func (g *game) renderInspect(w io.Writer) {
	g.noteVisibleMonsters()
	fmt.Fprintln(w, "\n=== Inspect ===")
	fmt.Fprintf(w, "You stand at %s on level %d.\n", pointLabel(g.player.Pos), g.currentLevel+1)
	fmt.Fprintln(w, inspectTileDescription(g.current(), g.player.Pos))
	if g.current().HasStair {
		fmt.Fprintf(w, "Stairs down: %s.\n", relativeTo(g.player.Pos, g.current().Stairs))
	}
	if g.current().HasUpStair {
		fmt.Fprintf(w, "Stairs up: %s.\n", relativeTo(g.player.Pos, g.current().UpStairs))
	}
	if g.current().HasEscapeStair && (g.current().Secret == nil || g.current().Secret.Revealed) {
		fmt.Fprintf(w, "Hidden way out: %s.\n", relativeTo(g.player.Pos, g.current().EscapeStairs))
	}
	for _, f := range g.current().Fountains {
		if distance(g.player.Pos, f.Pos) <= 6 {
			status := "unused"
			if f.Used {
				status = "spent"
			}
			fmt.Fprintf(w, "Fountain (%s): %s.\n", status, relativeTo(g.player.Pos, f.Pos))
		}
	}
	fmt.Fprintln(w, "Nearby threats:")
	threats := g.visibleThreats(6)
	if len(threats) == 0 {
		fmt.Fprintln(w, "- None close enough to matter right now.")
	} else {
		for _, m := range threats {
			fmt.Fprintf(w, "- %s: %s (%c) HP %d/%d damage %d-%d. %s\n", relativeTo(g.player.Pos, m.Pos), m.Name, m.Glyph, m.HP, m.MaxHP, m.MinDamage, m.MaxDamage, monsterPower(m.Kind))
		}
	}
	fmt.Fprintln(w, "Nearby loot:")
	loot := g.visibleItems(6)
	if len(loot) == 0 {
		fmt.Fprintln(w, "- No notable loot nearby.")
	} else {
		for _, it := range loot {
			fmt.Fprintf(w, "- %s: %s\n", relativeTo(g.player.Pos, it.Pos), inspectItemText(it))
		}
	}
	if g.dog.Freed && g.dog.Alive {
		fmt.Fprintf(w, "Ghost dog: %s.\n", relativeTo(g.player.Pos, g.dog.Pos))
	}
	fmt.Fprintln(w, "Tip: use movement or . to wait, or press k beside suspicious walls to search.")
}

func inspectTileDescription(lvl *level, p pos) string {
	switch {
	case lvl.HasEscapeStair && lvl.EscapeStairs == p && (lvl.Secret == nil || lvl.Secret.Revealed):
		return "You are standing on the hidden stairs up and out of the dungeon."
	case lvl.HasUpStair && lvl.UpStairs == p:
		return "You are standing on stairs leading back up."
	case lvl.HasStair && lvl.Stairs == p:
		return "You are standing on the stairs down."
	case lvl.DogChain != nil && *lvl.DogChain == p:
		return "A silver chain lies here where the ghost dog was bound."
	}
	for _, f := range lvl.Fountains {
		if f.Pos == p {
			if f.Used {
				return "An empty stone fountain; its healing water is spent."
			}
			return "A clear fountain. Stand here or beside it and press v to drink."
		}
	}
	return "The floor is cold stone, scarred by old battles and dragging claws."
}

func inspectItemText(it item) string {
	switch it.Kind {
	case itemWeapon:
		return fmt.Sprintf("weapon %s (%d-%d). %s", it.Weapon.Name, it.Weapon.Min, it.Weapon.Max, weaponDescription(it.Weapon))
	case itemArmor:
		return fmt.Sprintf("%s armor for the %s slot (+%d guard, %s). %s", coloredArmorName(it.Armor), slotLabel(it.Armor.Slot), it.Armor.Defense, rarityLabel(it.Armor.Rarity), armorDescription(it.Armor))
	case itemPotion:
		return "healing potion. Restores 12-18 health."
	case itemFireScroll:
		return "fire scroll. Blasts the nearest enemy within 6 tiles."
	case itemBlinkStone:
		return "blink stone. Teleports you to a nearby open tile."
	case itemWardingCharm:
		return "warding charm. Reduces damage for 3 enemy turns."
	case itemSunOrb:
		return "sun orb. Burns every undead enemy on the floor."
	case itemFrostCharm:
		return "frost charm. Freezes and wounds the nearest enemy within 6 tiles."
	case itemStarfireOrb:
		return "starfire orb. Deals 18-24 damage to the nearest enemy within 6 tiles."
	case itemPhoenixAsh:
		return "phoenix ash. Restores 22-30 health when used below full health."
	case itemGhostRecallScroll:
		return "Ghost Dog recall scroll. Use r to summon your fallen companion at full health."
	case itemStoryScroll:
		if it.StoryChapter > 0 && it.StoryChapter <= len(ghostDogStory) {
			return fmt.Sprintf("Ghost Dog story scroll %d/10: %s", it.StoryChapter, ghostDogStory[it.StoryChapter-1])
		}
		return "Ghost Dog story scroll."
	default:
		return it.Name
	}
}

func (g *game) visibleThreats(limit int) []*monster {
	var out []*monster
	for _, m := range g.current().Monsters {
		if m.HP > 0 && distance(g.player.Pos, m.Pos) <= limit {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := distance(g.player.Pos, out[i].Pos), distance(g.player.Pos, out[j].Pos)
		if di == dj {
			return out[i].Name < out[j].Name
		}
		return di < dj
	})
	return out
}

func (g *game) visibleItems(limit int) []item {
	var out []item
	for _, it := range g.current().Items {
		if distance(g.player.Pos, it.Pos) <= limit && g.current().Tiles[it.Pos.Y][it.Pos.X] != '#' {
			out = append(out, it)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := distance(g.player.Pos, out[i].Pos), distance(g.player.Pos, out[j].Pos)
		if di == dj {
			return out[i].Name < out[j].Name
		}
		return di < dj
	})
	return out
}

func pointLabel(p pos) string { return fmt.Sprintf("(%d,%d)", p.X, p.Y) }

func relativeTo(origin, target pos) string {
	dx, dy := target.X-origin.X, target.Y-origin.Y
	parts := []string{}
	if dy < 0 {
		parts = append(parts, fmt.Sprintf("%d north", -dy))
	} else if dy > 0 {
		parts = append(parts, fmt.Sprintf("%d south", dy))
	}
	if dx < 0 {
		parts = append(parts, fmt.Sprintf("%d west", -dx))
	} else if dx > 0 {
		parts = append(parts, fmt.Sprintf("%d east", dx))
	}
	if len(parts) == 0 {
		return "here"
	}
	return strings.Join(parts, ", ")
}

func (g *game) inventoryLines() []struct {
	name        string
	count       int
	description string
} {
	return []struct {
		name        string
		count       int
		description string
	}{{"Healing Potion", g.player.Potions, "Use p to restore 12-18 health instantly."}, {"Fire Scroll", g.player.FireScrolls, "Use f to blast the nearest monster within 6 tiles and scorch adjacent foes."}, {"Blink Stone", g.player.BlinkStones, "Use g to teleport to a nearby open tile when surrounded."}, {"Warding Charm", g.player.WardingCharms, "Use b to reduce incoming damage for the next 3 enemy turns."}, {"Sun Orb", g.player.SunOrbs, "Use u to burn every undead monster on the current floor."}, {"Frost Charm", g.player.FrostCharms, "Use t to freeze and damage the nearest enemy for two turns."}, {"Starfire Orb", g.player.StarfireOrbs, "Use o to deal 18-24 damage to the nearest enemy within 6 tiles."}, {"Phoenix Ash", g.player.PhoenixAshes, "Use n to restore 22-30 health below full health."}, {"Ghost Dog Recall Scroll", g.player.GhostRecallScrolls, "Use r to summon a fallen Ghost Dog at full hit points."}, {"Ghost Dog Story Scrolls", len(g.player.StoryChapters), "Collected story chapters are preserved with your save."}}
}

func (g *game) contextHint() string {
	lvl := g.current()
	for _, f := range lvl.Fountains {
		if !f.Used && distance(g.player.Pos, f.Pos) <= 1 {
			return "healing fountain nearby — press v to drink."
		}
	}
	if lvl.HasEscapeStair && lvl.Secret != nil && lvl.Secret.Revealed && g.player.Pos == lvl.EscapeStairs {
		return "stairs up and out — press < to escape."
	}
	if lvl.HasUpStair && g.player.Pos == lvl.UpStairs {
		return "stairs up — press < to climb."
	}
	if lvl.HasStair && g.player.Pos == lvl.Stairs {
		return "stairs down — press > to descend."
	}
	if lvl.Secret != nil && !lvl.Secret.Revealed && distance(g.player.Pos, lvl.Secret.Door) <= 1 {
		if lvl.HasEscapeStair && g.bossAlive(lvl) {
			return "a necromantic seal chills this wall — the Lich still holds it shut."
		}
		return "this wall feels strange — try search."
	}
	return "press i for equipment, c for the codex, m to inspect, or k to search suspicious walls."
}

func weaponDescription(w weapon) string {
	switch w.Name {
	case "Rusty Knife":
		return "A chipped backup blade, quick but weak."
	case "Short Sword":
		return "A balanced edge that makes early fights much safer."
	case "Iron Spear":
		return "A sturdy reach weapon that hits a little harder every swing."
	case "Chapel Mace":
		return "Blessed iron that caves in skulls better than your starting knife."
	case "Hooked Glaive":
		return "A polearm with a wicked bite, good for steady deep-floor damage."
	case "Battle Axe":
		return "Heavy steel that can end weaker monsters in a few brutal chops."
	case "Moonblade":
		return "A silvered sword that feels made for cursed halls and undead flesh."
	case "Dawnstar Flail":
		return "A brutal chain-weapon that thrives in tight rooms and ugly melees."
	case "Rune Saber":
		return "Its etched edge holds a steady killing line against elite monsters."
	case "Gravecleaver":
		return "A heavy cleaver made for the armored dead below."
	case "Hollowfang Spear":
		return "A barbed spear that keeps deep-floor horrors at a careful distance."
	case "Dragontooth Pike":
		return "A savage relic with enough force to carry you into the middle depths."
	case "Starforged Hammer":
		return "A dense hammer whose star-forged head breaks through mid-dungeon foes."
	case "Emberbrand":
		return "A fire-rune blade whose magic burns brighter against undead foes."
	case "Tempest Spear":
		return "A storm-charged spear that strikes with magical force against undead."
	case "Voidglass Dagger":
		return "A black-glass blade that cuts through undead essence as well as flesh."
	case "Lichbane Greatsword":
		return "An ancient enchanted greatsword made to bring down undead tyrants."
	case "Frostbite Axe":
		return "A deep-floor axe forged to bite through thick hides and armor."
	case "Gloomsteel Saber":
		return "Dark steel with a keen edge, tempered beneath the oldest halls."
	case "Graveglass Halberd":
		return "A long, heavy polearm built to keep deep-floor horrors at bay."
	case "Stormcaller Blade":
		return "A balanced blade that carries a sharp crack through every strike."
	case "Wyrmheart Maul":
		return "A dense war-maul made for creatures that shrug off ordinary blows."
	case "Gloaming Pike":
		return "A dark-tipped pike with enough force to pierce the lich's guard."
	case "Dawnforged Greatsword":
		return "A deep-floor greatsword with a bright, punishing edge."
	case "Kingsbane Axe":
		return "A heavy axe whose broad head can split even a monster's guard."
	case "Gravetide Maul":
		return "A secret relic that releases a burst of force against undead foes."
	case "Star-Eater Blade":
		return "A mythic secret blade that tears through undead and mortal alike."
	default:
		return fmt.Sprintf("Reliable steel dealing %d-%d damage.", w.Min, w.Max)
	}
}

func armorDescription(a armor) string {
	switch a.Name {
	case "Padded Hood":
		return "Thin padding for your skull; barely enough to stop a lucky scrape."
	case "Worn Coat":
		return "An old coat that still turns knives and claws a little."
	case "Frayed Boots":
		return "Almost no protection, but better than bare feet on dungeon stone."
	case "Leather Jerkin":
		return "Tough hide stitched for dungeon delvers who expect real hits."
	case "Scout Cap":
		return "Light headgear with just enough structure to turn away a nasty cut."
	case "Iron Helm":
		return "A dented helm that makes curses and clubs feel less final."
	case "Wolfhide Boots":
		return "Fur-lined boots that keep your footing when the halls turn murderous."
	case "Reinforced Greaves":
		return "Plated shins and boots that steady you against lunging beasts."
	case "Scale Coat":
		return "Linked scales spread hard blows across your chest and shoulders."
	case "Knight Mail":
		return "Layered steel rings that turn many brutal blows into survivable ones."
	case "Warden Crown":
		return "An ancient circlet that hardens your brow and your will together."
	case "Nightstride Boots":
		return "Quiet, heavy boots built for long marches through murderous halls."
	case "Sunplate Cuirass":
		return "Radiant plate that makes the last floor feel much less final."
	case "Dreadmark Mantle":
		return "A reinforced mantle that keeps deep-floor blows from landing cleanly."
	case "Gloamrunner Boots":
		return "Quiet boots stitched for long runs through hostile halls."
	case "Aetherweave Hood":
		return "A tightly woven hood that guards against steel and strange magic."
	case "Ashguard Coat":
		return "A soot-black coat that turns aside the worst blows from below."
	case "Stormscale Greaves":
		return "Layered greaves that brace your legs against heavy strikes."
	case "Graveward Helm":
		return "A sealed helm that holds firm against both blades and curses."
	case "Ruinplate Vest":
		return "A late-floor cuirass built from plates salvaged from a fallen citadel."
	case "Mourner's Helm":
		return "A deep-floor helm that guards your head through the worst encounters."
	case "Starforged Cuirass":
		return "Rare star-metal plate with exceptional protection for the deepest floor."
	case "Voidwalker Boots":
		return "Rare boots that steady every step through the dungeon's oldest reaches."
	case "Aegis of Echoes":
		return "Secret-forged armor that turns aside attacks with a lingering ward."
	case "Crown of the Hollow Star":
		return "A secret circlet that shelters its wearer from brutal blows."
	case "Wraithstep Greaves":
		return "Secret greaves that protect without slowing a careful retreat."
	case "Voidheart Plate":
		return "The dungeon's rarest armor, hidden where the oldest powers sleep."
	default:
		return fmt.Sprintf("Protective gear for your %s slot worth %d guard.", slotLabel(a.Slot), a.Defense)
	}
}

func compareWeapon(candidate, current weapon) string {
	return fmt.Sprintf("Compare to %s: %+d min, %+d max damage.", current.Name, candidate.Min-current.Min, candidate.Max-current.Max)
}

func compareArmor(candidate, current armor) string {
	return fmt.Sprintf("Compare to %s: %+d guard in the %s slot.", current.Name, candidate.Defense-current.Defense, slotLabel(candidate.Slot))
}

func rarityLabel(r rarity) string {
	switch r {
	case rarityCommon:
		return "common"
	case rarityUncommon:
		return "uncommon"
	case rarityRare:
		return "rare"
	case rarityEpic:
		return "epic"
	case rarityLegendary:
		return "legendary"
	default:
		return string(r)
	}
}

func rarityColor(r rarity) string {
	switch r {
	case rarityCommon:
		return "\x1b[37m"
	case rarityUncommon:
		return "\x1b[32m"
	case rarityRare:
		return "\x1b[34m"
	case rarityEpic:
		return "\x1b[35m"
	case rarityLegendary:
		return "\x1b[33m"
	default:
		return ""
	}
}

func coloredArmorName(a armor) string {
	prefix := strings.Title(rarityLabel(a.Rarity)) + " " + a.Name
	if color := rarityColor(a.Rarity); color != "" {
		return color + prefix + ansiReset
	}
	return prefix
}

func weaponScore(w weapon) int {
	score := w.Min + w.Max
	if w.Magic {
		score += 3
	}
	return score
}

func slotLabel(slot armorSlot) string {
	switch slot {
	case slotHead:
		return "head"
	case slotBody:
		return "body"
	case slotFeet:
		return "feet"
	default:
		return string(slot)
	}
}

func monsterSummary(kind monsterKind) string {
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
		return "Launches grave-fire bolts, carries the tenth Ghost Dog story scroll, and guards the hidden way out."
	default:
		return "Unknown."
	}
}

func (g *game) noteVisibleMonsters() {
	for _, m := range g.current().Monsters {
		if m.HP > 0 {
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

func (p *player) armorForSlot(slot armorSlot) armor {
	switch slot {
	case slotHead:
		return p.HeadArmor
	case slotBody:
		return p.BodyArmor
	case slotFeet:
		return p.FeetArmor
	default:
		return armor{}
	}
}

func (p *player) setArmor(a armor) {
	switch a.Slot {
	case slotHead:
		p.HeadArmor = a
	case slotBody:
		p.BodyArmor = a
	case slotFeet:
		p.FeetArmor = a
	}
}

func (p *player) armorDefense() int {
	return p.HeadArmor.Defense + p.BodyArmor.Defense + p.FeetArmor.Defense
}
func (p *player) armorBlock() int { return p.armorDefense() / 2 }

func (g *game) addMessage(msg string) {
	g.messages = append(g.messages, msg)
	if len(g.messages) > 8 {
		g.messages = g.messages[len(g.messages)-8:]
	}
}

func (g *game) current() *level { return g.levels[g.currentLevel] }

func (g *game) randomOpenTile(lvl *level, occupied map[pos]bool) pos {
	floors := make([]pos, 0, mapWidth*mapHeight)
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{X: x, Y: y}
			if occupied[p] || lvl.Tiles[y][x] == '#' {
				continue
			}
			floors = append(floors, p)
		}
	}
	if len(floors) == 0 {
		panic("could not place tile")
	}
	return floors[g.rng.Intn(len(floors))]
}

func (g *game) randomOpenTileFarFrom(lvl *level, occupied map[pos]bool, origin pos, minDistance int) pos {
	floors := make([]pos, 0, mapWidth*mapHeight)
	far := []pos{}
	best := pos{}
	bestDist := -1
	for y := 1; y < mapHeight-1; y++ {
		for x := 1; x < mapWidth-1; x++ {
			p := pos{X: x, Y: y}
			if occupied[p] || lvl.Tiles[y][x] == '#' {
				continue
			}
			floors = append(floors, p)
			d := distance(origin, p)
			if d >= minDistance {
				far = append(far, p)
			}
			if d > bestDist {
				bestDist = d
				best = p
			}
		}
	}
	if len(far) > 0 {
		return far[g.rng.Intn(len(far))]
	}
	if len(floors) == 0 {
		panic("could not place far tile")
	}
	return best
}

func (g *game) itemAt(lvl *level, p pos) *item {
	for i := range lvl.Items {
		if lvl.Items[i].Pos == p {
			return &lvl.Items[i]
		}
	}
	return nil
}

func (g *game) fountainAt(lvl *level, p pos) *fountain {
	for i := range lvl.Fountains {
		if lvl.Fountains[i].Pos == p {
			return &lvl.Fountains[i]
		}
	}
	return nil
}

func (g *game) monsterAt(lvl *level, p pos) *monster {
	for _, m := range lvl.Monsters {
		if m.HP > 0 && m.Pos == p {
			return m
		}
	}
	return nil
}

func (g *game) bossAlive(lvl *level) bool {
	for _, m := range lvl.Monsters {
		if m.Boss && m.HP > 0 {
			return true
		}
	}
	return false
}

func (g *game) adjacentMonster(p pos) *monster {
	for _, m := range g.current().Monsters {
		if m.HP > 0 && distance(m.Pos, p) == 1 {
			return m
		}
	}
	return nil
}

func (g *game) nearestMonster(limit int) *monster {
	return g.nearestMonsterFromLimit(g.player.Pos, limit)
}
func (g *game) nearestMonsterFrom(origin pos) *monster {
	return g.nearestMonsterFromLimit(origin, math.MaxInt)
}
func (g *game) nearestMonsterFromLimit(origin pos, limit int) *monster {
	var best *monster
	bestDist := math.MaxInt
	for _, m := range g.current().Monsters {
		if m.HP <= 0 {
			continue
		}
		d := distance(origin, m.Pos)
		if d < bestDist && d <= limit {
			best, bestDist = m, d
		}
	}
	return best
}

func (g *game) openNeighbors(center pos, ignoreDog bool) []pos {
	options := make([]pos, 0, 8)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			p := pos{X: center.X + dx, Y: center.Y + dy}
			if g.isWalkable(p, ignoreDog) {
				options = append(options, p)
			}
		}
	}
	return options
}

func (g *game) isWalkable(p pos, ignoreDog bool) bool {
	if !g.inBounds(p) || g.current().Tiles[p.Y][p.X] == '#' || g.player.Pos == p {
		return false
	}
	if !ignoreDog && !g.dog.Freed && g.current().DogChain != nil && *g.current().DogChain == p {
		return false
	}
	if !ignoreDog && g.dog.Freed && g.dog.Alive && g.dog.Pos == p {
		return false
	}
	return g.monsterAt(g.current(), p) == nil
}

func (g *game) inBounds(p pos) bool { return p.X >= 0 && p.X < mapWidth && p.Y >= 0 && p.Y < mapHeight }

func distance(a, b pos) int {
	dx := abs(a.X - b.X)
	dy := abs(a.Y - b.Y)
	if dx > dy {
		return dx
	}
	return dy
}

func sign(v int) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	default:
		return 0
	}
}

func (g *game) randRange(minVal, maxVal int) int {
	if maxVal <= minVal {
		return minVal
	}
	return minVal + g.rng.Intn(maxVal-minVal+1)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

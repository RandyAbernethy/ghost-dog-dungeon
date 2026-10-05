package main

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
	stats            runStats
	recordsFile      string
	generationError  error
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
		stats:            runStats{HistoryKnown: true},
		player: player{
			HP:            46,
			MaxHP:         46,
			Weapon:        weapon{Name: "Rusty Knife", Min: 3, Max: 7},
			HeadArmor:     armor{Name: "Padded Hood", Slot: slotHead, Defense: 1, Rarity: rarityCommon},
			BodyArmor:     armor{Name: "Worn Coat", Slot: slotBody, Defense: 1, Rarity: rarityCommon},
			LegArmor:      armor{Name: "Frayed Boots", Slot: slotLeg, Defense: 0, Rarity: rarityCommon},
			Potions:       3,
			FireScrolls:   1,
			BlinkStones:   0,
			WardingCharms: 0,
			SunOrbs:       0,
		},
		dog: ghostDog{HP: 34, MaxHP: 34, Alive: true},
	}
	g.player.ensureCarriedEquipment()

	if err := g.buildDungeon(); err != nil {
		g.generationError = err
		return g
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

func (g *game) addMessage(msg string) {
	g.messages = append(g.messages, msg)
	if len(g.messages) > 8 {
		g.messages = g.messages[len(g.messages)-8:]
	}
}

func (g *game) current() *level { return g.levels[g.currentLevel] }

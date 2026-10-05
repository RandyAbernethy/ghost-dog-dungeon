package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

func (a *armor) UnmarshalJSON(data []byte) error {
	type armorData armor
	var saved armorData
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}
	// Normalize the old slot and generated armor names throughout saved games.
	if saved.Slot == "feet" {
		saved.Slot = slotLeg
	}
	if saved.Slot == slotLeg {
		saved.Name = strings.ReplaceAll(saved.Name, " feet armor", " leg armor")
	}
	*a = armor(saved)
	return nil
}

func (p *player) UnmarshalJSON(data []byte) error {
	type playerData player
	var saved struct {
		playerData
		LegacyLegArmor armor `json:"feet_armor"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}
	*p = player(saved.playerData)
	if p.LegArmor.Name == "" {
		p.LegArmor = saved.LegacyLegArmor
	}
	return nil
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
	Stats         *runStats     `json:"stats,omitempty"`
	RecordsFile   string        `json:"records_file,omitempty"`
}

func (g *game) save() error {
	if g.timestampedSaves || g.saveFile == "" {
		g.saveFile = nextTimestampedSaveFile(time.Now())
	}
	state := saveState{RNGState: g.rng.State, Seed: g.seed, SaveFile: g.saveFile, Levels: g.levels, CurrentLevel: g.currentLevel, Player: g.player, Dog: g.dog, Messages: append([]string(nil), g.messages...), KnownMonsters: append([]monsterKind(nil), g.knownMonsterKinds()...), Won: g.won, Quit: g.quit, LevelTiles: make([]string, len(g.levels))}
	state.Stats, state.RecordsFile = &g.stats, g.recordsFile
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
	if state.Stats != nil {
		g.stats = *state.Stats
	}
	g.recordsFile = state.RecordsFile
	if g.player.HeadArmor.Name == "" {
		g.player.HeadArmor = armor{Name: "Padded Hood", Slot: slotHead, Defense: 1, Rarity: rarityCommon}
	}
	if g.player.BodyArmor.Name == "" {
		g.player.BodyArmor = armor{Name: "Worn Coat", Slot: slotBody, Defense: 1, Rarity: rarityCommon}
	}
	if g.player.LegArmor.Name == "" {
		g.player.LegArmor = armor{Name: "Frayed Boots", Slot: slotLeg, Defense: 0, Rarity: rarityCommon}
	}
	if g.player.HeadArmor.Rarity == "" {
		g.player.HeadArmor.Rarity = rarityCommon
	}
	if g.player.BodyArmor.Rarity == "" {
		g.player.BodyArmor.Rarity = rarityCommon
	}
	if g.player.LegArmor.Rarity == "" {
		g.player.LegArmor.Rarity = rarityCommon
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
	g.player.ensureCarriedEquipment()
	for _, lvl := range g.levels {
		for i := range lvl.Items {
			if lvl.Items[i].Kind == itemArmor && lvl.Items[i].Armor.Slot == slotLeg {
				lvl.Items[i].Name = lvl.Items[i].Armor.Name
			}
		}
		g.addSecretSideStory(lvl)
		g.addSecretFountain(lvl)
		g.prepareGhostGifts(lvl)
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

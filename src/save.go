package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const saveFormatVersion = 1

type saveState struct {
	FormatVersion int           `json:"format_version"`
	RNGState      uint64        `json:"rng_state"`
	Seed          int64         `json:"seed"`
	Levels        []*level      `json:"levels"`
	CurrentLevel  int           `json:"current_level"`
	Player        player        `json:"player"`
	Dog           ghostDog      `json:"dog"`
	Messages      []string      `json:"messages"`
	KnownMonsters []monsterKind `json:"known_monsters"`
	Won           bool          `json:"won"`
	LevelTiles    []string      `json:"level_tiles"`
	Stats         runStats      `json:"stats"`
}

func prepareSaveDirectory(path string) (string, error) {
	if path == "" {
		return "", errors.New("--save-file-path must name a directory")
	}
	dir, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve save directory %q: %w", path, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create save directory %q for --save-file-path: %w", dir, err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("inspect save directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("--save-file-path %q is not a directory", dir)
	}
	return dir, nil
}

func (g *game) save() error {
	state := saveState{
		FormatVersion: saveFormatVersion,
		RNGState:      g.rng.State, Seed: g.seed, Levels: g.levels,
		CurrentLevel: g.currentLevel, Player: g.player, Dog: g.dog,
		Messages:      append([]string(nil), g.messages...),
		KnownMonsters: g.knownMonsterKinds(), Won: g.won, Stats: g.stats,
		LevelTiles: make([]string, len(g.levels)),
	}
	for i, lvl := range g.levels {
		state.LevelTiles[i] = encodeTiles(lvl.Tiles)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode saved game: %w", err)
	}
	f, err := createTimestampedSaveFile(g.saveDir, time.Now())
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		removeErr := os.Remove(f.Name())
		return fmt.Errorf("write save file %q: %w", f.Name(), errors.Join(err, removeErr))
	}
	g.saveFile = f.Name()
	return nil
}

func createTimestampedSaveFile(dir string, now time.Time) (*os.File, error) {
	base := defaultSavePrefix + "-" + now.Format("20060102-150405")
	for suffix := 0; ; suffix++ {
		name := base + ".json"
		if suffix > 0 {
			name = fmt.Sprintf("%s-%02d.json", base, suffix)
		}
		path := filepath.Join(dir, name)
		// Exclusive creation keeps repeated saves and concurrent runs from overwriting each other.
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("create timestamped save in %q: %w", dir, err)
		}
		return f, nil
	}
}

func loadGame(path string) (*game, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read save file %q: %w", path, err)
	}
	var state saveState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode save file %q: %w", path, err)
	}
	if state.FormatVersion != saveFormatVersion {
		return nil, fmt.Errorf("unsupported save format %d in %q; this release requires format %d, so start a new game", state.FormatVersion, path, saveFormatVersion)
	}
	if len(state.Levels) != levelCount || len(state.LevelTiles) != levelCount {
		return nil, errors.New("save file is missing dungeon levels")
	}
	if state.CurrentLevel < 0 || state.CurrentLevel >= levelCount {
		return nil, fmt.Errorf("save file has invalid current level %d", state.CurrentLevel+1)
	}
	if state.RNGState == 0 {
		return nil, errors.New("save file has an invalid random state")
	}
	for i, lvl := range state.Levels {
		if lvl == nil {
			return nil, fmt.Errorf("save file is missing floor %d", i+1)
		}
		tiles, err := decodeTiles(state.LevelTiles[i])
		if err != nil {
			return nil, fmt.Errorf("decode floor %d: %w", i+1, err)
		}
		lvl.Tiles = tiles
	}
	known := map[monsterKind]bool{}
	for _, kind := range state.KnownMonsters {
		known[kind] = true
	}
	return &game{
		rng: &simpleRNG{State: state.RNGState}, seed: state.Seed,
		saveFile: path, saveDir: filepath.Dir(path), levels: state.Levels,
		currentLevel: state.CurrentLevel, player: state.Player, dog: state.Dog,
		messages: append([]string(nil), state.Messages...), knownMonsters: known,
		won: state.Won, stats: state.Stats,
	}, nil
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

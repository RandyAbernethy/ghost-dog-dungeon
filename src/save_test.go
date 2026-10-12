package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func lastSavePath(t *testing.T, output string) string {
	t.Helper()
	var path string
	for _, line := range strings.Split(stripANSI(output), "\n") {
		if after, ok := strings.CutPrefix(line, "- Game saved to "); ok {
			path = strings.TrimSpace(after)
		}
	}
	if path == "" {
		t.Fatal("game output did not report a saved file")
	}
	return path
}

func TestEverySaveCreatesSnapshotAndFailureKeepsLatestSave(t *testing.T) {
	g := newGameWithSaveDir(42, t.TempDir())
	g.recordsFile = filepath.Join(t.TempDir(), "records.json")
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	first := g.saveFile
	firstData, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	g.processCommand(".")
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	second := g.saveFile
	if first == second || filepath.Dir(second) != g.saveDir || !strings.HasPrefix(filepath.Base(second), defaultSavePrefix+"-") {
		t.Fatal("every save should create a fresh timestamped file in the chosen directory")
	}
	unchanged, err := os.ReadFile(first)
	if err != nil || !bytes.Equal(firstData, unchanged) {
		t.Fatal("a later save must preserve the earlier snapshot")
	}
	loaded, err := loadGame(second)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.seed != g.seed || loaded.rng.State != g.rng.State || loaded.stats != g.stats || !reflect.DeepEqual(loaded.player, g.player) || !reflect.DeepEqual(loaded.levels, g.levels) {
		t.Fatal("loading must restore the exact snapshot without generating or repairing state")
	}
	var saved map[string]json.RawMessage
	if err := json.Unmarshal(firstData, &saved); err != nil {
		t.Fatal(err)
	}
	for _, setting := range []string{"save_file", "records_file", "quit"} {
		if _, ok := saved[setting]; ok {
			t.Fatalf("runtime setting %q should not be persisted in the saved dungeon", setting)
		}
	}
	beforeRNG, beforeStats := g.rng.State, g.stats
	g.saveDir = filepath.Join(first, "cannot-be-a-directory")
	if err := g.save(); err == nil || !strings.Contains(err.Error(), g.saveDir) {
		t.Fatal("a failed save should identify the unusable directory")
	}
	if g.saveFile != second || g.rng.State != beforeRNG || g.stats != beforeStats {
		t.Fatal("a failed save must preserve the last usable snapshot and game state")
	}
}

func TestTimestampCollisionCreatesDistinctFilesWithoutOverwrite(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 12, 9, 8, 7, 0, time.UTC)
	paths := make([]string, 3)
	for i := range paths {
		f, err := createTimestampedSaveFile(dir, now)
		if err != nil {
			t.Fatal(err)
		}
		paths[i] = f.Name()
		if _, err := f.WriteString("snapshot"); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	for i, path := range paths {
		for _, previous := range paths[:i] {
			if path == previous {
				t.Fatal("saves in the same second must have distinct filenames")
			}
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "snapshot" {
			t.Fatal("a collision must not truncate an existing snapshot")
		}
	}
}

func TestLoadRestoresSnapshotAndPreservesRuntimeDirectories(t *testing.T) {
	g := openArena()
	g.saveDir = t.TempDir()
	g.recordsFile = filepath.Join(t.TempDir(), "records.json")
	g.processCommand("save")
	first := g.saveFile
	beforePlayer, beforeStats, beforeRNG := g.player, g.stats, g.rng.State
	g.processCommand("d")
	g.saveDir = t.TempDir()
	saveDir, recordsFile := g.saveDir, g.recordsFile
	g.processCommand("load")
	if !reflect.DeepEqual(g.player, beforePlayer) || g.stats != beforeStats || g.rng.State != beforeRNG || g.saveFile != first {
		t.Fatal("the load command must restore its latest snapshot without spending a turn")
	}
	if g.saveDir != saveDir || g.recordsFile != recordsFile {
		t.Fatal("loading must preserve the current save directory and records path")
	}
	g.processCommand("save")
	if filepath.Dir(g.saveFile) != saveDir || g.saveFile == first || g.stats != beforeStats || g.rng.State != beforeRNG {
		t.Fatal("saving after loading must create a fresh snapshot in the selected directory without spending a turn")
	}
}

func TestLoadRejectsUnsupportedSaveFormats(t *testing.T) {
	g := newGameWithSaveDir(42, t.TempDir())
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []int{0, saveFormatVersion + 1} {
		var state saveState
		if err := json.Unmarshal(data, &state); err != nil {
			t.Fatal(err)
		}
		state.FormatVersion = version
		invalid, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "unsupported.json")
		if err := os.WriteFile(path, invalid, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadGame(path); err == nil || !strings.Contains(err.Error(), "unsupported save format") || !strings.Contains(err.Error(), "start a new game") {
			t.Fatal("unsupported saves must fail with a helpful message instead of being migrated")
		}
	}
}

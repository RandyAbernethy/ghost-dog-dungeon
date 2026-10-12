package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestExecutableBuildSaveAndLoadFromAnotherDirectory(t *testing.T) {
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "ghostdog")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", binary, "./src")
	build.Dir = filepath.Dir(packageDir)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build game from repository root: %v\n%s", err, output)
	}
	run := func(workdir, input string, args ...string) (string, error) {
		t.Helper()
		cmd := exec.Command(binary, args...)
		cmd.Dir, cmd.Env = workdir, append(os.Environ(), "HOME="+workdir)
		cmd.Stdin = strings.NewReader(input)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}
	readSave := func(path string) map[string]json.RawMessage {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var saved map[string]json.RawMessage
		if err := json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		return saved
	}
	const banner = "Ghost Dog Dungeon version 1.0.1\n"
	assertRunIdentity := func(output string, seed int64) {
		t.Helper()
		seedLine := fmt.Sprintf("Dungeon seed: %d\n", seed)
		if !strings.HasPrefix(output, banner+seedLine) || strings.Count(output, banner) != 1 || !strings.HasSuffix(output, seedLine) || strings.Count(output, seedLine) != 2 {
			t.Fatal("the game should show its version once, with the actual dungeon seed immediately below it and again on exit")
		}
	}
	var first map[string]json.RawMessage
	for i := 0; i < 2; i++ {
		workdir := t.TempDir()
		saveDir := filepath.Join(workdir, "saved games", "nested")
		output, err := run(workdir, "book\n.\nsave\nquit\n", "--seed", "42", "--save-file-path", saveDir)
		if err != nil {
			t.Fatalf("new game: %v\n%s", err, output)
		}
		assertRunIdentity(output, 42)
		path := executableSavePath(t, output)
		if filepath.Dir(path) != saveDir {
			t.Fatalf("save %q is outside the configured directory %q", path, saveDir)
		}
		for _, want := range []string{"=== Ghost Dog Dungeon:", "=== Book of Ghost Dog ===", "Turn 1 events (0 monsters killed):", "You wait and let the dungeon move around you.", "Game saved to " + path} {
			if !strings.Contains(output, want) {
				t.Fatalf("new game did not show %q", want)
			}
		}
		saved := readSave(path)
		var stats struct {
			Turns int `json:"turns"`
		}
		if err := json.Unmarshal(saved["stats"], &stats); err != nil || stats.Turns != 1 {
			t.Fatal("reading the book and saving should not spend turns")
		}
		if first == nil {
			first = saved
		} else {
			for _, field := range []string{"rng_state", "player", "dog", "levels", "level_tiles", "stats"} {
				if !bytes.Equal(first[field], saved[field]) {
					t.Fatalf("same seed and commands did not reproduce %s", field)
				}
			}
		}
		continuedDir := filepath.Join(workdir, "continued games")
		output, err = run(workdir, "map\nbook\ni\n\x1b\nsave\nquit\n", "--load-file", path, "--save-file-path", continuedDir, "--seed", "99")
		if err != nil || !strings.Contains(output, "Loaded saved game.") || !strings.Contains(output, "Turn 1 events (0 monsters killed):") {
			t.Fatalf("load and read screens: %v\n%s", err, output)
		}
		assertRunIdentity(output, 42)
		loadedPath := executableSavePath(t, output)
		if loadedPath == path || filepath.Dir(loadedPath) != continuedDir {
			t.Fatal("loading and saving should create a fresh snapshot in the selected directory")
		}
		if !reflect.DeepEqual(saved, readSave(path)) {
			t.Fatal("saving a loaded game must preserve the original snapshot")
		}
		loaded := readSave(loadedPath)
		for _, field := range []string{"rng_state", "player", "dog", "levels", "level_tiles", "stats"} {
			var before, after any
			if err := json.Unmarshal(saved[field], &before); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(loaded[field], &after); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("reading screens after loading changed %s", field)
			}
		}
		if _, err := os.Stat(filepath.Join(workdir, ".ghost-dog-data.json")); !os.IsNotExist(err) {
			t.Fatal("quitting an unfinished run must not submit records")
		}
	}
	output, err := run(dir, "", "--help")
	if err != nil || !strings.Contains(output, "Usage: ghostdog-dungeon") || !strings.Contains(output, "-save-file-path string") || !strings.Contains(output, "random seed for dungeon generation (default is current time in nanoseconds)\n") || strings.Contains(output, "  -load\n") || strings.Contains(output, "  -save-file string") {
		t.Fatalf("help: %v\n%s", err, output)
	}
	output, err = run(dir, "", "--version", "--load-file", filepath.Join(dir, "missing.json"), "--save-file-path", filepath.Join(dir, "must-not-be-created"))
	if err != nil || output != banner {
		t.Fatalf("version should print and exit without starting or loading a run: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(dir, "must-not-be-created")); !os.IsNotExist(err) {
		t.Fatal("printing the version must not create a save directory")
	}
	output, err = run(dir, "", "--load-file", filepath.Join(dir, "missing.json"))
	if err == nil || !strings.Contains(output, "error:") {
		t.Fatalf("missing save should report a startup error: %v\n%s", err, output)
	}
	for _, oldFlag := range []string{"--load", "--save-file"} {
		output, err := run(dir, "", oldFlag)
		if err == nil || !strings.Contains(output, "flag provided but not defined") {
			t.Fatalf("removed flag %s should be rejected: %v\n%s", oldFlag, err, output)
		}
	}
	blocker := filepath.Join(dir, "regular-file")
	if err := os.WriteFile(blocker, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, badDir := range []string{"", blocker, filepath.Join(blocker, "child")} {
		output, err := run(dir, "", "--save-file-path", badDir)
		if err == nil || !strings.Contains(output, "--save-file-path") || strings.Contains(output, "=== Ghost Dog Dungeon:") {
			t.Fatalf("invalid save directory should fail before gameplay with a helpful error: %v\n%s", err, output)
		}
	}
	// Omitting the directory writes timestamped snapshots in the working directory.
	workdir := t.TempDir()
	output, err = run(workdir, "save\nsave\nload\nquit\n", "--seed", "7")
	if err != nil || strings.Contains(output, "Save failed:") || strings.Contains(output, "Load failed:") {
		t.Fatalf("default-directory saves: %v\n%s", err, output)
	}
	assertRunIdentity(output, 7)
	files, err := filepath.Glob(filepath.Join(workdir, "ghostdog-dungeon-*.json"))
	if err != nil || len(files) != 2 {
		t.Fatalf("repeated saves should produce two snapshots, got %v, err=%v", files, err)
	}
	var previousSeed int64
	for i := 0; i < 2; i++ {
		workdir := t.TempDir()
		output, err := run(workdir, "save\nquit\n")
		if err != nil {
			t.Fatalf("default seed run: %v\n%s", err, output)
		}
		saved := readSave(executableSavePath(t, output))
		var seed int64
		if err := json.Unmarshal(saved["seed"], &seed); err != nil {
			t.Fatal(err)
		}
		if seed == 0 || (i > 0 && seed == previousSeed) {
			t.Fatal("omitting --seed must choose a fresh timestamp seed for each new run")
		}
		previousSeed = seed
		assertRunIdentity(output, seed)
	}
	var zeroSeedDungeon map[string]json.RawMessage
	for i := 0; i < 2; i++ {
		workdir := t.TempDir()
		output, err := run(workdir, "save\nquit\n", "--seed", "0")
		if err != nil {
			t.Fatalf("explicit zero seed run: %v\n%s", err, output)
		}
		saved := readSave(executableSavePath(t, output))
		if string(saved["seed"]) != "0" {
			t.Fatal("an explicit --seed 0 must not be replaced by the timestamp default")
		}
		assertRunIdentity(output, 0)
		if zeroSeedDungeon != nil {
			for _, field := range []string{"rng_state", "levels", "level_tiles"} {
				if !bytes.Equal(zeroSeedDungeon[field], saved[field]) {
					t.Fatalf("explicit zero seed did not reproduce %s", field)
				}
			}
		}
		zeroSeedDungeon = saved
	}
	output, err = run(dir, "", "--seed", "-123")
	if err != nil || !strings.Contains(output, "The dungeon waits in silence as you slip away.") {
		t.Fatalf("end of input: %v\n%s", err, output)
	}
	assertRunIdentity(output, -123)
}

func executableSavePath(t *testing.T, output string) string {
	t.Helper()
	var path string
	for _, line := range strings.Split(output, "\n") {
		if after, ok := strings.CutPrefix(line, "- Game saved to "); ok {
			path = strings.TrimSpace(after)
		}
	}
	if path == "" {
		t.Fatal("game output did not report a saved file")
	}
	return path
}

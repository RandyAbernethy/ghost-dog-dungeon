package tests

import (
	"bytes"
	"encoding/json"
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
	var first map[string]json.RawMessage
	for i := 0; i < 2; i++ {
		workdir := t.TempDir()
		path := filepath.Join(workdir, "run.json")
		output, err := run(workdir, "book\n.\nsave\nquit\n", "--seed", "42", "--save-file", path)
		if err != nil {
			t.Fatalf("new game: %v\n%s", err, output)
		}
		for _, want := range []string{"=== Ghost Dog Dungeon:", "=== Book of Ghost Dog ===", "Turn 1 events:", "You wait and let the dungeon move around you.", "Game saved to " + path} {
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
		output, err = run(workdir, "map\nbook\ni\n\x1b\nsave\nquit\n", "--load-file", path)
		if err != nil || !strings.Contains(output, "Loaded saved game.") || !strings.Contains(output, "Turn 1 events:") {
			t.Fatalf("load and read screens: %v\n%s", err, output)
		}
		loaded := readSave(path)
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
	if err != nil || !strings.Contains(output, "Usage: ghostdog-dungeon") {
		t.Fatalf("help: %v\n%s", err, output)
	}
	output, err = run(dir, "", "--load-file", filepath.Join(dir, "missing.json"))
	if err == nil || !strings.Contains(output, "error:") {
		t.Fatalf("missing save should report a startup error: %v\n%s", err, output)
	}
}

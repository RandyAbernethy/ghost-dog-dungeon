package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	testRoot       string
	testPackageDir string
	testBinaryOnce sync.Once
	testBinary     string
	testBuildErr   error
)

// Build only the state needed by arena tests. Ten explicit floors also allow
// these fixtures to exercise the current save format without invoking generation.
func openArena() *game {
	g := &game{
		seed: 24, rng: newSimpleRNG(24), knownMonsters: map[monsterKind]bool{},
		levels: make([]*level, levelCount),
		player: player{
			Pos: pos{5, 5}, HP: 46, MaxHP: 46,
			Weapon:    weapon{Name: "Rusty Knife", Min: 3, Max: 7},
			HeadArmor: armor{Name: "Padded Hood", Slot: slotHead, Defense: 1, Rarity: rarityCommon},
			BodyArmor: armor{Name: "Worn Coat", Slot: slotBody, Defense: 1, Rarity: rarityCommon},
			LegArmor:  armor{Name: "Frayed Boots", Slot: slotLeg, Rarity: rarityCommon},
			Potions:   3, FireScrolls: 1,
		},
		dog: ghostDog{HP: 34, MaxHP: 34, Alive: true},
	}
	for depth := range g.levels {
		lvl := &level{
			Index: depth, Tiles: blankFloor(), Start: pos{5, 5},
			Stairs: pos{30, 10}, HasStair: depth < levelCount-1,
			UpStairs: pos{5, 5}, HasUpStair: depth > 0, Visited: depth == 0,
		}
		for y := 1; y < mapHeight-1; y++ {
			for x := 1; x < mapWidth-1; x++ {
				lvl.Tiles[y][x] = '.'
			}
		}
		g.levels[depth] = lvl
	}
	g.player.ensureCarriedEquipment()
	return g
}

func TestMain(m *testing.M) {
	var err error
	testRoot, err = os.MkdirTemp("", "ghostdog-tests-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	testPackageDir, err = os.Getwd()
	if err == nil {
		err = os.Mkdir(filepath.Join(testRoot, "home"), 0700)
	}
	if err == nil {
		for _, key := range []string{"HOME", "USERPROFILE"} {
			if err = os.Setenv(key, filepath.Join(testRoot, "home")); err != nil {
				break
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if cleanupErr := os.RemoveAll(testRoot); cleanupErr != nil {
			fmt.Fprintln(os.Stderr, cleanupErr)
		}
		os.Exit(1)
	}
	code := m.Run()
	if err := os.RemoveAll(testRoot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}

// Windows record lookup uses USERPROFILE; Unix uses HOME. Replace inherited
// entries rather than appending duplicate keys with platform-dependent handling.
func testEnvironment(home string) []string {
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(key, "HOME") && !strings.EqualFold(key, "USERPROFILE") {
			env = append(env, entry)
		}
	}
	return append(env, "HOME="+home, "USERPROFILE="+home)
}

func testCommand(t *testing.T, timeout time.Duration, name string, args ...string) *exec.Cmd {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	return cmd
}

// Keep the executable until the package finishes, so scenarios share one build.
func gameBinary(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("compiled-executable scenarios run in the full suite")
	}
	testBinaryOnce.Do(func() {
		name := "ghostdog"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		testBinary = filepath.Join(testRoot, name)
		build := testCommand(t, 2*time.Minute, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", testBinary, ".")
		build.Dir = testPackageDir
		if output, err := build.CombinedOutput(); err != nil {
			testBuildErr = fmt.Errorf("build game: %w\n%s", err, output)
		}
	})
	if testBuildErr != nil {
		t.Fatal(testBuildErr)
	}
	return testBinary
}

func gameCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	cmd := testCommand(t, 30*time.Second, gameBinary(t), args...)
	cmd.Dir = t.TempDir()
	cmd.Env = testEnvironment(cmd.Dir)
	return cmd
}

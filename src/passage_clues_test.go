package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestCluesRespectWallsAndParentRooms(t *testing.T) {
	g := openArena()
	lvl := g.current()
	outer := &secretRoom{Door: pos{8, 5}, Tiles: []pos{{9, 5}}}
	inner := &secretRoom{Door: pos{10, 5}, Tiles: []pos{{11, 5}}}
	outer.Next = inner
	lvl.Secrets = []*secretRoom{outer}
	lvl.Tiles[5][8], lvl.Tiles[5][9], lvl.Tiles[5][10], lvl.Tiles[5][11] = '#', '#', '#', '#'
	if clue := g.passageClue(g.player.Pos); !strings.Contains(clue, "3 east") {
		t.Fatalf("three-tile clue must give direction: %s", clue)
	}
	if len(g.passageClues(pos{7, 5})) != 1 {
		t.Fatal("unopened linked chambers must not leak their clues")
	}
	revealRoom(lvl, outer)
	if clue := g.passageClue(pos{9, 5}); !strings.Contains(clue, "1 east") {
		t.Fatal("opening parent should expose the child's clue")
	}
	for y := 1; y < mapHeight-1; y++ {
		lvl.Tiles[y][7] = '#'
	}
	if g.clueNear(pos{6, 5}, inner.Door) {
		t.Fatal("clues must not travel through a separating wall")
	}
}

func TestClueSeedSweepAndPersistence(t *testing.T) {
	seeds := 80
	if testing.Short() {
		seeds = 8
	}
	for seed := int64(1); seed <= int64(seeds); seed++ {
		g := newGame(seed)
		if g.generationError != nil {
			t.Fatal(g.generationError)
		}
		beforeRNG := g.rng.State
		for index, lvl := range g.levels {
			g.currentLevel = index
			for _, root := range lvl.Secrets {
				for hidden := root; hidden != nil; hidden = hidden.Next {
					approach, ok := floorNextToDoor(g, hidden.Door)
					if !ok || !reachable(lvl.Tiles, lvl.Start, approach) || !g.clueNear(approach, hidden.Door) {
						t.Fatalf("seed %d floor %d: clue must be reachable before opening", seed, index+1)
					}
					if clue := g.passageClue(approach); clue == "" || strings.Contains(clue, "search") {
						t.Fatal("reachable door needs a candidate clue without search instructions")
					}
					revealRoom(lvl, hidden)
				}
			}
		}
		if beforeRNG != g.rng.State {
			t.Fatal("clues and revealing cannot draw RNG")
		}
	}
	g := newGame(42)
	g.saveDir = t.TempDir()
	before := g.passageClues(g.player.Pos)
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGame(g.saveFile)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	loaded.render(&out)
	loaded.renderInspect(&out)
	if !reflect.DeepEqual(before, loaded.passageClues(loaded.player.Pos)) || loaded.rng.State != g.rng.State {
		t.Fatal("clues must remain identical across reading and loading")
	}
}

func subtleHintArena() *game {
	g := openArena()
	g.current().Secrets = []*secretRoom{{Door: pos{8, 5}, Tiles: []pos{{9, 5}}}}
	g.current().Tiles[5][8], g.current().Tiles[5][9] = '#', '#'
	g.messages = nil
	return g
}

func TestPassageHintsUseTwentyPercentBoundary(t *testing.T) {
	g := subtleHintArena()
	found := map[int]bool{}
	for seed := int64(1); len(found) < 2 && seed < 10000; seed++ {
		rng := newSimpleRNG(seed)
		roll := rng.Intn(100)
		if roll != 19 && roll != 20 {
			continue
		}
		g.rng = newSimpleRNG(seed)
		g.messages = nil
		state := g.rng.State
		g.maybePassageHint()
		if (len(g.messages) > 0) != (roll == 19) || g.rng.State != state {
			t.Fatal("rolls below 20 must hint without advancing combat RNG; 20 must not")
		}
		if strings.Contains(strings.Join(g.messages, " "), "search") {
			t.Fatal("hints must never tell the player to search")
		}
		found[roll] = true
	}
	if len(found) != 2 {
		t.Fatal("chance boundary seeds not exercised")
	}
	hints := 0
	g.rng = newSimpleRNG(42)
	for turn := 1; turn <= 10000; turn++ {
		g.stats.Turns = turn
		g.messages = nil
		g.maybePassageHint()
		if len(g.messages) > 0 {
			hints++
		}
	}
	if hints < 1800 || hints > 2200 {
		t.Fatalf("expected about 20%% hints, got %d/10000", hints)
	}
}

func TestPassageHintReadsDoNotRevealOrRerollSecrets(t *testing.T) {
	g := subtleHintArena()
	var initial bytes.Buffer
	g.render(&initial)
	g.renderInspect(&initial)
	if strings.Contains(initial.String(), "mortar") || strings.Contains(g.contextHint(), "search") {
		t.Fatal("standing near a secret must not guarantee a hint or a search prompt")
	}
	// Choose a saved seed that offers a hint on the first eligible move.
	for seed := int64(1); ; seed++ {
		rng := newSimpleRNG(seed)
		rng.State ^= 0x9e3779b97f4a7c15
		if rng.Intn(100) < 20 {
			g.rng = newSimpleRNG(seed)
			break
		}
	}
	state := g.rng.State
	g.processCommand("d")
	if len(g.messages) != 2 || !strings.Contains(g.messages[1], "mortar") || strings.Contains(g.messages[1], "search") {
		t.Fatal("a successful move may offer one subtle hint")
	}
	g.saveDir = t.TempDir()
	player, stats := g.player, g.stats
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("m\nmap\nsave\nload\nm\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	if g.rng.State != state || !reflect.DeepEqual(g.player, player) || g.stats != stats {
		t.Fatal("reads and loading cannot retry or advance hint rolls")
	}
	count := 0
	for _, msg := range g.messages {
		if strings.Contains(msg, "mortar") {
			count++
		}
	}
	if count != 1 {
		t.Fatal("saved current-turn hint must be retained exactly once")
	}
}

func TestPassageHintsThroughExecutableSaveLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("compiled subtle-hint replay")
	}
	g := subtleHintArena()
	g.saveDir = t.TempDir()
	for seed := int64(1); ; seed++ {
		rng := newSimpleRNG(seed)
		rng.State ^= 0x9e3779b97f4a7c15
		if rng.Intn(100) < 20 {
			g.rng = newSimpleRNG(seed)
			break
		}
	}
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader("d\nm\nsave\nload\nmap\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hint replay: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "mortar") || strings.Contains(string(output), "Approach the wall and search") {
		t.Fatal("compiled movement must show a subtle hint without a search prompt")
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	g.processCommand("d")
	if loaded.rng.State != g.rng.State || loaded.stats != g.stats || !reflect.DeepEqual(loaded.player, g.player) {
		t.Fatal("saving and loading cannot change hint or movement outcomes")
	}
}

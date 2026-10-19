package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestObjectiveRoomsThroughExecutableAndSaveLoad(t *testing.T) {
	for _, kind := range []objectiveKind{objectiveSpirit, objectiveSigils} {
		t.Run(string(kind), func(t *testing.T) {
			g := objectiveArena(kind)
			g.saveDir = t.TempDir()
			if err := g.save(); err != nil {
				t.Fatal(err)
			}
			actions := ".\n.\n.\n"
			if kind == objectiveSigils {
				actions = "d\ny\ns\ny\n"
			}
			cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
			cmd.Stdin = strings.NewReader("y\nsave\nload\nm\n" + actions + "save\nload\ny\nsave\nquit\n")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("executable: %v\n%s", err, output)
			}
			loaded, err := loadGame(lastSavePath(t, string(output)))
			if err != nil {
				t.Fatal(err)
			}
			if !loaded.current().Objective.Completed || loaded.player.WardingCharms != 1 || !reflect.DeepEqual(loaded.adventure.ObjectivesDone, []string{string(kind)}) || loaded.rng.State != g.rng.State || !strings.Contains(string(output), "Objective complete: "+string(kind)) {
				t.Fatal("the executable must preserve progress and award the saved reward once")
			}
		})
	}
}

func TestEpicTelegraphThroughExecutableAndSaveLoad(t *testing.T) {
	g, _ := warnedEpicArena(monsterStormLeviathan)
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader(".\nsave\nload\nm\na\nsave\nquit\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executable: %v\n%s", err, output)
	}
	loaded, err := loadGame(lastSavePath(t, string(output)))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.stats.Turns != 2 || loaded.player.HP != 200 || len(loaded.current().Monsters[0].StrikeTiles) != 0 || !strings.Contains(string(output), "YOU are on a marked tile") || !strings.Contains(string(output), "Thunder Marks strikes empty stone") {
		t.Fatal("the executable must display the saved warning and let the player dodge it")
	}
}

func TestStoryEndingAndChronicleThroughExecutable(t *testing.T) {
	g := openArena()
	g.currentLevel = 9
	g.dog.Freed, g.dog.Pos = true, pos{6, 5}
	g.current().HasEscapeStair, g.current().EscapeStairs = true, g.player.Pos
	g.current().Secrets = []*secretRoom{{Revealed: true, Tiles: []pos{g.player.Pos}}}
	for i := range ghostDogSideStories {
		g.player.SideStories = append(g.player.SideStories, i+1)
	}
	g.stats.GhostDogFalls = 1
	g.adventure = adventureState{RescueFloor: 2, EpicsDefeated: []string{"Epic Frost Monarch"}, ObjectivesDone: []string{string(objectiveSpirit)}}
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	var reference string
	for _, resume := range []bool{false, true} {
		name, commands := "uninterrupted", "book\n<\n"
		if resume {
			name, commands = "save_and_load", "book\nsave\nload\n<\n"
		}
		t.Run(name, func(t *testing.T) {
			cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
			cmd.Stdin = strings.NewReader(commands)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("executable: %v\n%s", err, output)
			}
			text := string(output)
			for _, want := range []string{"You climb the hidden stairs and leave the dungeon behind.", "Victory!", "=== Your Adventure ===", "silver chain on floor 2", "You defeated Epic Frost Monarch", "You completed " + string(objectiveSpirit), "=== Optional Challenges ===", "=== Personal Records ===", "NEW RECORD!"} {
				if !strings.Contains(text, want) {
					t.Fatalf("the finished run did not show %q", want)
				}
			}
			_, afterVictory, ok := strings.Cut(text, "You escaped the dungeon with your ghost dog. Victory!\n")
			if !ok {
				t.Fatal("the companion victory message must remain unchanged")
			}
			ending, _, ok := strings.Cut(afterVictory, "=== Your Adventure ===")
			lines := strings.Split(strings.TrimSpace(ending), "\n")
			if !ok || len(lines) != 2 || lines[0] != "Beside the door, you see a clean bowl filled with fresh rain water." {
				t.Fatalf("the bowl must be followed by just one reminiscence before the chronicle: %q", ending)
			}
			if resume && ending != reference {
				t.Fatal("save/load must retain the same ending selection")
			}
			reference = ending
			if !strings.HasSuffix(text, fmt.Sprintf("Dungeon seed: %d\n", g.seed)) {
				t.Fatal("the dungeon seed must still finish the output")
			}
		})
	}
}

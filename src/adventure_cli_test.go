package main

import (
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
	g.player.SideStories = []int{9, 10}
	g.adventure = adventureState{RescueFloor: 2, EpicsDefeated: []string{"Epic Frost Monarch"}, ObjectivesDone: []string{string(objectiveSpirit)}}
	g.saveDir = t.TempDir()
	if err := g.save(); err != nil {
		t.Fatal(err)
	}
	cmd := gameCommand(t, "--load-file", g.saveFile, "--save-file-path", g.saveDir)
	cmd.Stdin = strings.NewReader("book\nsave\nload\n<\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executable: %v\n%s", err, output)
	}
	for _, want := range []string{"Victory!", "Mara's dream", "home is the hand", "=== Your Adventure ===", "silver chain on floor 2", "You defeated Epic Frost Monarch", "You completed " + string(objectiveSpirit), "NEW RECORD!"} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("the finished run did not show %q", want)
		}
	}
}

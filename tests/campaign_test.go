package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type campaignSnapshot struct {
	Seed         int64 `json:"seed"`
	CurrentLevel int   `json:"current_level"`
	Player       struct {
		HP            int                   `json:"hp"`
		Weapon        struct{ Name string } `json:"weapon"`
		StoryChapters []int                 `json:"story_chapters"`
		SideStories   []int                 `json:"side_stories"`
	} `json:"player"`
	Dog struct {
		Freed bool `json:"freed"`
		Alive bool `json:"alive"`
	} `json:"dog"`
	Stats struct {
		Turns              int `json:"turns"`
		MonstersKilled     int `json:"monsters_killed"`
		DifficultyDefeated int `json:"difficulty_defeated"`
		GhostDogFalls      int `json:"ghost_dog_falls"`
	} `json:"stats"`
	Levels []struct {
		Visited  bool `json:"visited"`
		Monsters []struct {
			HP   int  `json:"hp"`
			Boss bool `json:"boss"`
		} `json:"monsters"`
		Secrets []struct{ Revealed bool } `json:"secrets"`
	} `json:"levels"`
}

type campaignResult struct {
	Seed       int64 `json:"seed"`
	Turns      int   `json:"turns"`
	Kills      int   `json:"kills"`
	Difficulty int   `json:"difficulty"`
	Escaped    bool  `json:"escaped"`
}

func TestSeededCampaignFromNewGameToEscape(t *testing.T) {
	gameBinary(t)
	route, err := os.ReadFile(filepath.Join(testPackageDir, "testdata", "seed-6.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var reference map[string]json.RawMessage
	for _, resume := range []bool{false, true} {
		name := "uninterrupted"
		if resume {
			name = "save_and_load"
		}
		t.Run(name, func(t *testing.T) {
			var commands []string
			for _, line := range strings.Split(string(route), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") || line == "load" && !resume {
					continue
				}
				if len(line) > 1 && strings.Trim(line, "wasdqezx") == "" {
					for _, key := range line {
						commands = append(commands, string(key))
					}
				} else {
					commands = append(commands, line)
				}
			}
			dir := t.TempDir()
			cmd := gameCommand(t, "--seed", "6", "--save-file-path", filepath.Join(dir, "saves"))
			cmd.Dir, cmd.Env = dir, testEnvironment(dir)
			cmd.Stdin = strings.NewReader(strings.Join(commands, "\n") + "\n")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("campaign process: %v\n%s", err, output[max(0, len(output)-4000):])
			}
			if !strings.Contains(string(output), "Victory!") || !strings.Contains(string(output), "You free the ghost dog") ||
				!strings.Contains(string(output), "Dread Lich falls.") {
				t.Fatal("the fresh game must rescue Ghost Dog, defeat the Lich, and escape through normal commands")
			}
			paths := executableSavePaths(t, string(output))
			floors := []int{0, 1, 2, 3, 4, 4, 5, 6, 7, 8, 9, 9}
			if len(paths) != len(floors) {
				t.Fatalf("got %d checkpoints, want %d", len(paths), len(floors))
			}
			var final campaignSnapshot
			var checkpoint, finalState map[string]json.RawMessage
			for i, path := range paths {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var snapshot campaignSnapshot
				if err := json.Unmarshal(data, &snapshot); err != nil {
					t.Fatal(err)
				}
				final = snapshot
				if final.Seed != 6 || final.CurrentLevel != floors[i] || final.Player.HP <= 0 || len(final.Levels) != 10 {
					t.Fatalf("checkpoint %d: seed=%d floor=%d HP=%d levels=%d", i, final.Seed, final.CurrentLevel+1, final.Player.HP, len(final.Levels))
				}
				if i == 0 && (final.Stats.Turns != 0 || final.Dog.Freed) || i >= 2 && !final.Dog.Freed {
					t.Fatalf("checkpoint %d: unexpected starting state or Ghost Dog was not rescued on floor two", i)
				}
				var state map[string]json.RawMessage
				if err := json.Unmarshal(data, &state); err != nil {
					t.Fatal(err)
				}
				// Load messages name the current snapshot; all persisted gameplay
				// fields must remain identical, including the future RNG state.
				delete(state, "messages")
				if i == 4 {
					checkpoint = state
				}
				if i == 5 && !reflect.DeepEqual(checkpoint, state) {
					t.Fatal("reading and restoring the floor-five checkpoint changed gameplay state")
				}
				finalState = state
			}
			dead, bosses := 0, 0
			for depth, lvl := range final.Levels {
				if !lvl.Visited {
					t.Fatalf("floor %d was never visited", depth+1)
				}
				for _, m := range lvl.Monsters {
					if m.HP <= 0 {
						dead++
					}
					if m.Boss {
						bosses++
						if m.HP > 0 {
							t.Fatal("the final checkpoint still has a living Lich")
						}
					}
				}
			}
			// This completion route leaves chapter seven behind while collecting
			// the Lich's final chapter and side stories through ordinary pickups.
			chapters := []int{1, 2, 3, 4, 5, 6, 8, 9, 10}
			if bosses != 1 || dead != final.Stats.MonstersKilled || dead == 0 || final.Stats.DifficultyDefeated <= 0 ||
				!final.Dog.Alive || final.Player.Weapon.Name == "Rusty Knife" || !reflect.DeepEqual(final.Player.StoryChapters, chapters) || len(final.Player.SideStories) == 0 ||
				len(final.Levels[9].Secrets) == 0 || !final.Levels[9].Secrets[0].Revealed {
				t.Fatalf("campaign did not preserve combat, equipment, story, companion, or escape progress: kills=%d/%d chapters=%v sides=%v dog=%+v weapon=%s bosses=%d", dead, final.Stats.MonstersKilled, final.Player.StoryChapters, final.Player.SideStories, final.Dog, final.Player.Weapon.Name, bosses)
			}
			data, err := os.ReadFile(filepath.Join(dir, ".ghost-dog-data.json"))
			if err != nil {
				t.Fatal("victory did not write records to the isolated home: ", err)
			}
			var records map[string]campaignResult
			if err := json.Unmarshal(data, &records); err != nil {
				t.Fatal(err)
			}
			keys := []string{"fewest_turns", "most_monsters_killed", "most_difficulty_defeated"}
			if final.Stats.GhostDogFalls == 0 {
				keys = append(keys, "ghost_dog_never_fell")
			} else if _, exists := records["ghost_dog_never_fell"]; exists {
				t.Fatal("a campaign with Ghost Dog falling must not earn a safe escape record")
			}
			for _, key := range keys {
				record := records[key]
				if record.Seed != 6 || !record.Escaped || record.Turns != final.Stats.Turns+1 ||
					record.Kills != dead || record.Difficulty != final.Stats.DifficultyDefeated {
					t.Fatalf("%s did not record the completed campaign and final stair turn: %+v", key, record)
				}
			}
			if reference == nil {
				reference = finalState
			} else if !reflect.DeepEqual(reference, finalState) {
				t.Fatal("save/load continuation diverged from uninterrupted play")
			}
			t.Logf("seed 6: %d floors, %d turns, %d kills, %d chapters", len(final.Levels), final.Stats.Turns+1, dead, len(final.Player.StoryChapters))
		})
	}
}

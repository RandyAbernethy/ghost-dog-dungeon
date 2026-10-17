package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultRecordsPathUsesUserHomeAcrossSaveDirectories(t *testing.T) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(userHome, ".ghost-dog-data.json")
	for _, saveDir := range []string{".", "saves", t.TempDir()} {
		g := newGameWithSaveDir(1, saveDir)
		got, err := g.challengeRecordsPath()
		if err != nil || got != want {
			t.Fatalf("save directory %q uses records %q, err=%v; want %q", saveDir, got, err, want)
		}
		if g.recordsFile != "" {
			t.Fatal("default home storage should not be tied to a saved dungeon")
		}
	}
	g := newGame(1)
	g.recordsFile = filepath.Join(t.TempDir(), "custom-records.json")
	got, err := g.challengeRecordsPath()
	if err != nil || got != g.recordsFile {
		t.Fatal("an explicit records path should still be supported")
	}
}

func TestFinishedRunRecordCelebrations(t *testing.T) {
	all := []string{"Fewest turns: 25 turns", "Ghost Dog safe escape: 25 turns", "Most monsters killed: 7", "Most difficulty defeated: 250 points"}
	for _, tc := range []struct {
		name       string
		firstRun   bool
		escaped    bool
		stats      runStats
		newRecords []string
		ties       []string
	}{
		{"first escape", true, true, runStats{Turns: 25, MonstersKilled: 7, DifficultyDefeated: 250}, all, nil},
		{"improved escape", false, true, runStats{Turns: 24, MonstersKilled: 8, DifficultyDefeated: 251},
			[]string{"Fewest turns: 24 turns", "Ghost Dog safe escape: 24 turns", "Most monsters killed: 8", "Most difficulty defeated: 251 points"}, nil},
		{"tied escape", false, true, runStats{Turns: 25, MonstersKilled: 7, DifficultyDefeated: 250}, nil, all},
		{"worse escape", false, true, runStats{Turns: 26, MonstersKilled: 6, DifficultyDefeated: 249}, nil, nil},
		{"mixed results", false, true, runStats{Turns: 24, MonstersKilled: 7, DifficultyDefeated: 249},
			[]string{"Fewest turns: 24 turns", "Ghost Dog safe escape: 24 turns"}, []string{"Most monsters killed: 7"}},
		{"death improves scores", false, false, runStats{Turns: 1, MonstersKilled: 8, DifficultyDefeated: 251},
			[]string{"Most monsters killed: 8", "Most difficulty defeated: 251 points"}, nil},
		{"death ties scores", false, false, runStats{Turns: 1, MonstersKilled: 7, DifficultyDefeated: 250},
			nil, []string{"Most monsters killed: 7", "Most difficulty defeated: 250 points"}},
		{"first death sets zero scores", true, false, runStats{},
			[]string{"Most monsters killed: 0", "Most difficulty defeated: 0 points"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.recordsFile = filepath.Join(t.TempDir(), "records.json")
			var previous challengeRecords
			var before []byte
			var beforeInfo os.FileInfo
			if !tc.firstRun {
				best := &challengeResult{Seed: 99, Turns: 25, Kills: 7, Difficulty: 250, Escaped: true}
				previous = challengeRecords{FewestTurns: best, SafeEscape: best, MostKills: best, MostDifficulty: best}
				var err error
				before, err = json.Marshal(previous)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(g.recordsFile, before, 0600); err != nil {
					t.Fatal(err)
				}
				beforeInfo, err = os.Stat(g.recordsFile)
				if err != nil {
					t.Fatal(err)
				}
			}
			g.won, g.dog.Freed = tc.escaped, true
			g.stats = tc.stats
			if !tc.escaped {
				g.player.HP = 0
			}
			beforeRNG := g.rng.State
			var out bytes.Buffer
			if err := g.loop(strings.NewReader(""), &out); err != nil {
				t.Fatal(err)
			}
			output := out.String()
			if strings.Count(output, "NEW RECORD!") != len(tc.newRecords) || strings.Count(output, "RECORD TIED!") != len(tc.ties) {
				t.Fatalf("wrong record announcements:\n%s", output)
			}
			for _, group := range []struct {
				label  string
				scores []string
			}{{"NEW RECORD! ", tc.newRecords}, {"RECORD TIED! ", tc.ties}} {
				for _, score := range group.scores {
					if !strings.Contains(output, "*** "+group.label+score+" ***") {
						t.Fatalf("missing prominent announcement for %s%s:\n%s", group.label, score, output)
					}
				}
			}
			if len(tc.newRecords)+len(tc.ties) > 0 {
				if strings.Count(output, "A RUN FOR THE RECORD BOOKS!") != 1 ||
					strings.Index(output, "A RUN FOR THE RECORD BOOKS!") < strings.LastIndex(output, "=== Personal Records ===") ||
					!strings.Contains(output, "\x1b[1;33m************************************************************") {
					t.Fatal("records should receive one bold gold celebration after the final statistics")
				}
			} else if strings.Contains(output, "A RUN FOR THE RECORD BOOKS!") {
				t.Fatal("a run without a new or tied record must not receive a celebration")
			}
			if g.stats != tc.stats || g.rng.State != beforeRNG {
				t.Fatal("reporting records changed the run statistics or RNG")
			}
			stored, err := readChallengeRecords(g.recordsFile)
			if err != nil {
				t.Fatal(err)
			}
			if tc.firstRun || len(tc.newRecords) == 4 {
				want := &challengeResult{Seed: g.seed, Turns: tc.stats.Turns, Kills: tc.stats.MonstersKilled, Difficulty: tc.stats.DifficultyDefeated, Escaped: tc.escaped}
				expected := challengeRecords{MostKills: want, MostDifficulty: want}
				if tc.escaped {
					expected.FewestTurns, expected.SafeEscape = want, want
				}
				if !reflect.DeepEqual(stored, expected) {
					t.Fatalf("announced records were not saved: got %+v, want %+v", stored, expected)
				}
			}
			if !tc.firstRun && len(tc.newRecords) == 0 {
				after, err := os.ReadFile(g.recordsFile)
				if err != nil {
					t.Fatal(err)
				}
				afterInfo, err := os.Stat(g.recordsFile)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(before, after) || !os.SameFile(beforeInfo, afterInfo) {
					t.Fatal("ties and worse results must preserve the original record file and holders")
				}
			}
			if !tc.escaped && !tc.firstRun && (!reflect.DeepEqual(stored.FewestTurns, previous.FewestTurns) || !reflect.DeepEqual(stored.SafeEscape, previous.SafeEscape)) {
				t.Fatal("death must not replace either escape record")
			}
			if tc.name == "mixed results" && (!reflect.DeepEqual(stored.MostKills, previous.MostKills) || !reflect.DeepEqual(stored.MostDifficulty, previous.MostDifficulty) || stored.FewestTurns.Turns != 24 || stored.SafeEscape.Turns != 24) {
				t.Fatal("mixed results must replace only strictly improved records")
			}
		})
	}
}

func TestSafeEscapeRecordCelebrationRequiresLivingUnfallenCompanion(t *testing.T) {
	for _, tc := range []struct {
		name  string
		freed bool
		alive bool
		falls int
	}{
		{"never freed", false, true, 0},
		{"dead", true, false, 0},
		{"revived", true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := openArena()
			g.recordsFile = filepath.Join(t.TempDir(), "records.json")
			g.won, g.dog.Freed, g.dog.Alive = true, tc.freed, tc.alive
			g.stats.GhostDogFalls = tc.falls
			var out bytes.Buffer
			if err := g.loop(strings.NewReader(""), &out); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "NEW RECORD! Ghost Dog safe escape:") || strings.Contains(out.String(), "RECORD TIED! Ghost Dog safe escape:") {
				t.Fatal("an ineligible companion must not earn a safe escape announcement")
			}
			records, err := readChallengeRecords(g.recordsFile)
			if err != nil || records.SafeEscape != nil {
				t.Fatalf("an ineligible companion must not earn a safe escape record: %+v, %v", records, err)
			}
		})
	}
}

func TestRecordScreensSavingAndQuitDoNotCelebrate(t *testing.T) {
	g := openArena()
	g.saveDir = t.TempDir()
	g.recordsFile = filepath.Join(g.saveDir, "records.json")
	g.stats = runStats{Turns: 20, MonstersKilled: 100, DifficultyDefeated: 999}
	var out bytes.Buffer
	if err := g.loop(strings.NewReader("challenges\nsave\nload\nchallenges\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "RECORD!") || strings.Contains(out.String(), "RECORD TIED!") || strings.Contains(out.String(), "A RUN FOR THE RECORD BOOKS!") {
		t.Fatal("reading, saving, loading, and quitting must not celebrate unfinished results")
	}
	if _, err := os.Stat(g.recordsFile); !os.IsNotExist(err) {
		t.Fatalf("unfinished run wrote records: %v", err)
	}
	if g.stats.Turns != 20 || g.saveFile == "" {
		t.Fatal("reading records and saving must stay free actions")
	}
}

func TestRecordStorageFailureDoesNotCelebrate(t *testing.T) {
	for _, failure := range []string{"read", "write"} {
		t.Run(failure, func(t *testing.T) {
			g := openArena()
			g.won = true
			g.recordsFile = filepath.Join(t.TempDir(), "missing", "records.json")
			if failure == "read" {
				g.recordsFile = filepath.Join(t.TempDir(), "records.json")
				if err := os.WriteFile(g.recordsFile, []byte("invalid JSON"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			if err := g.loop(strings.NewReader(""), &out); err != nil {
				t.Fatal(err)
			}
			output := out.String()
			if !strings.Contains(output, "Could not save challenge records:") || strings.Contains(output, "NEW RECORD!") || strings.Contains(output, "RECORD TIED!") || strings.Contains(output, "A RUN FOR THE RECORD BOOKS!") {
				t.Fatalf("storage failure should report its error without celebrating unsaved records:\n%s", output)
			}
		})
	}
}

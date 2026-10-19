package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type runStats struct {
	Turns              int `json:"turns"`
	MonstersKilled     int `json:"monsters_killed"`
	GhostDogFalls      int `json:"ghost_dog_falls"`
	DifficultyDefeated int `json:"difficulty_defeated,omitempty"`
}

type challengeResult struct {
	Seed       int64 `json:"seed"`
	Turns      int   `json:"turns"`
	Kills      int   `json:"kills"`
	Escaped    bool  `json:"escaped"`
	Difficulty int   `json:"difficulty,omitempty"`
}

type challengeRecords struct {
	FewestTurns    *challengeResult `json:"fewest_turns,omitempty"`
	SafeEscape     *challengeResult `json:"ghost_dog_never_fell,omitempty"`
	MostKills      *challengeResult `json:"most_monsters_killed,omitempty"`
	MostDifficulty *challengeResult `json:"most_difficulty_defeated,omitempty"`
}

type recordAchievement struct {
	Description string
	Tied        bool
}

func defaultRecordsFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory for challenge records: %w", err)
	}
	return filepath.Join(home, ".ghost-dog-data.json"), nil
}

func (g *game) challengeRecordsPath() (string, error) {
	if g.recordsFile != "" {
		return g.recordsFile, nil
	}
	return defaultRecordsFile()
}

func readChallengeRecords(path string) (challengeRecords, error) {
	var records challengeRecords
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return records, nil
	}
	if err != nil {
		return records, err
	}
	err = json.Unmarshal(data, &records)
	return records, err
}

func (g *game) recordChallenges() ([]recordAchievement, error) {
	if !g.won && g.player.HP > 0 {
		return nil, nil
	}
	path, err := g.challengeRecordsPath()
	if err != nil {
		return nil, err
	}
	records, err := readChallengeRecords(path)
	if err != nil {
		return nil, err
	}
	result := challengeResult{Seed: g.seed, Turns: g.stats.Turns, Kills: g.stats.MonstersKilled, Escaped: g.won, Difficulty: g.stats.DifficultyDefeated}
	var achievements []recordAchievement
	changed := false
	// Compare against the previous records before replacing them, so a new best
	// remains distinguishable from a tie when the final statistics are displayed.
	if g.won && (records.FewestTurns == nil || result.Turns <= records.FewestTurns.Turns) {
		tied := records.FewestTurns != nil && result.Turns == records.FewestTurns.Turns
		achievements = append(achievements, recordAchievement{fmt.Sprintf("Fewest turns: %d turns", result.Turns), tied})
		if !tied {
			records.FewestTurns = &result
			changed = true
		}
	}
	if g.won && g.dog.active() && g.stats.GhostDogFalls == 0 && (records.SafeEscape == nil || result.Turns <= records.SafeEscape.Turns) {
		tied := records.SafeEscape != nil && result.Turns == records.SafeEscape.Turns
		achievements = append(achievements, recordAchievement{fmt.Sprintf("Ghost Dog safe escape: %d turns", result.Turns), tied})
		if !tied {
			records.SafeEscape = &result
			changed = true
		}
	}
	if records.MostKills == nil || result.Kills >= records.MostKills.Kills {
		tied := records.MostKills != nil && result.Kills == records.MostKills.Kills
		achievements = append(achievements, recordAchievement{fmt.Sprintf("Most monsters killed: %d", result.Kills), tied})
		if !tied {
			records.MostKills = &result
			changed = true
		}
	}
	if records.MostDifficulty == nil || result.Difficulty >= records.MostDifficulty.Difficulty {
		tied := records.MostDifficulty != nil && result.Difficulty == records.MostDifficulty.Difficulty
		achievements = append(achievements, recordAchievement{fmt.Sprintf("Most difficulty defeated: %d points", result.Difficulty), tied})
		if !tied {
			records.MostDifficulty = &result
			changed = true
		}
	}
	if !changed {
		return achievements, nil
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".ghost-dog-data-*.tmp")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return nil, writeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return nil, err
	}
	return achievements, nil
}

func renderRecordAchievements(w io.Writer, achievements []recordAchievement) {
	if len(achievements) == 0 {
		return
	}
	fmt.Fprintln(w, "\n\x1b[1;33m************************************************************")
	fmt.Fprintln(w, "              A RUN FOR THE RECORD BOOKS!")
	for _, achievement := range achievements {
		label := "NEW RECORD!"
		if achievement.Tied {
			label = "RECORD TIED!"
		}
		fmt.Fprintf(w, "*** %s %s ***\n", label, achievement.Description)
	}
	fmt.Fprintln(w, "************************************************************"+ansiReset)
}

func (g *game) renderChallenges(w io.Writer) {
	fmt.Fprintln(w, "\n=== Optional Challenges ===")
	fmt.Fprintf(w, "This run: %d turns, %d monsters killed, Ghost Dog fell %d time(s).\n", g.stats.Turns, g.stats.MonstersKilled, g.stats.GhostDogFalls)
	fmt.Fprintf(w, "Difficulty defeated: %d points. Tougher enemies earn more points.\n", g.stats.DifficultyDefeated)
	fmt.Fprintln(w, "Fewest turns: escape as quickly as you can. Most kills: counts kills by you and Ghost Dog in a run ending in escape or death.")
	if g.stats.GhostDogFalls > 0 {
		fmt.Fprintln(w, "Escape without Ghost Dog falling: failed this run, even if he is revived.")
	} else if g.won && g.dog.active() {
		fmt.Fprintln(w, "Escape without Ghost Dog falling: achieved this run.")
	} else if g.won || g.player.HP <= 0 {
		fmt.Fprintln(w, "Escape without Ghost Dog falling: this run ended without a safe escape together.")
	} else {
		fmt.Fprintln(w, "Escape without Ghost Dog falling: still possible. Free him and bring him out alive.")
	}
	g.renderChallengeRecords(w)
}

func (g *game) renderChallengeRecords(w io.Writer) {
	fmt.Fprintln(w, "\n=== Personal Records ===")
	path, err := g.challengeRecordsPath()
	if err != nil {
		fmt.Fprintf(w, "Could not read challenge records: %v\n", err)
		return
	}
	records, err := readChallengeRecords(path)
	if err != nil {
		fmt.Fprintf(w, "Could not read challenge records: %v\n", err)
		return
	}
	if records.FewestTurns == nil {
		fmt.Fprintln(w, "Fewest turns record: no escape recorded yet.")
	} else {
		fmt.Fprintf(w, "Fewest turns record: %d turns (seed %d).\n", records.FewestTurns.Turns, records.FewestTurns.Seed)
	}
	if records.SafeEscape == nil {
		fmt.Fprintln(w, "Ghost Dog safe escape record: not achieved yet.")
	} else {
		fmt.Fprintf(w, "Ghost Dog safe escape record: achieved in %d turns (seed %d).\n", records.SafeEscape.Turns, records.SafeEscape.Seed)
	}
	if records.MostKills == nil {
		fmt.Fprintln(w, "Most monsters killed record: no finished run recorded yet.")
	} else {
		fmt.Fprintf(w, "Most monsters killed record: %d (seed %d).\n", records.MostKills.Kills, records.MostKills.Seed)
	}
	if records.MostDifficulty == nil {
		fmt.Fprintln(w, "Most difficulty defeated record: no finished run recorded yet.")
	} else {
		fmt.Fprintf(w, "Most difficulty defeated record: %d points (seed %d).\n", records.MostDifficulty.Difficulty, records.MostDifficulty.Seed)
	}
}

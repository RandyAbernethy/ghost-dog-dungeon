package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type runStats struct {
	Turns          int  `json:"turns"`
	MonstersKilled int  `json:"monsters_killed"`
	GhostDogFalls  int  `json:"ghost_dog_falls"`
	HistoryKnown   bool `json:"history_known"`
}

type challengeResult struct {
	Seed    int64 `json:"seed"`
	Turns   int   `json:"turns"`
	Kills   int   `json:"kills"`
	Escaped bool  `json:"escaped"`
}

type challengeRecords struct {
	FewestTurns *challengeResult `json:"fewest_turns,omitempty"`
	SafeEscape  *challengeResult `json:"ghost_dog_never_fell,omitempty"`
	MostKills   *challengeResult `json:"most_monsters_killed,omitempty"`
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

func (g *game) recordChallenges() error {
	if !g.stats.HistoryKnown || (!g.won && g.player.HP > 0) {
		return nil
	}
	path, err := g.challengeRecordsPath()
	if err != nil {
		return err
	}
	records, err := readChallengeRecords(path)
	if err != nil {
		return err
	}
	result := challengeResult{Seed: g.seed, Turns: g.stats.Turns, Kills: g.stats.MonstersKilled, Escaped: g.won}
	changed := false
	if records.MostKills == nil || result.Kills > records.MostKills.Kills {
		records.MostKills = &result
		changed = true
	}
	if g.won && (records.FewestTurns == nil || result.Turns < records.FewestTurns.Turns) {
		records.FewestTurns = &result
		changed = true
	}
	if g.won && g.dog.Freed && g.dog.Alive && g.stats.GhostDogFalls == 0 && (records.SafeEscape == nil || result.Turns < records.SafeEscape.Turns) {
		records.SafeEscape = &result
		changed = true
	}
	if !changed {
		return nil
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".ghost-dog-data-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

func (g *game) renderChallenges(w io.Writer) {
	fmt.Fprintln(w, "\n=== Optional Challenges ===")
	fmt.Fprintf(w, "This run: %d turns, %d monsters killed, Ghost Dog fell %d time(s).\n", g.stats.Turns, g.stats.MonstersKilled, g.stats.GhostDogFalls)
	fmt.Fprintln(w, "Fewest turns: escape as quickly as you can. Most kills: counts kills by you and Ghost Dog in a run ending in escape or death.")
	if !g.stats.HistoryKnown {
		fmt.Fprintln(w, "This older save has no complete challenge history. Start a new game to set records.")
	} else if g.stats.GhostDogFalls > 0 {
		fmt.Fprintln(w, "Escape without Ghost Dog falling: failed this run, even if he is revived.")
	} else if g.won && g.dog.Freed && g.dog.Alive {
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
}

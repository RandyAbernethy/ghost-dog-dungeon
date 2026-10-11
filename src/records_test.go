package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultRecordsPathUsesUserHomeAcrossSaveFiles(t *testing.T) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(userHome, ".ghost-dog-data.json")
	for _, saveFile := range []string{"", "run.json", filepath.Join(t.TempDir(), "run.json")} {
		g := newGameWithSaveFile(1, saveFile)
		got, err := g.challengeRecordsPath()
		if err != nil || got != want {
			t.Fatalf("save %q uses records %q, err=%v; want %q", saveFile, got, err, want)
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

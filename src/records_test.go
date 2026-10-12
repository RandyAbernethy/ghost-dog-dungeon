package main

import (
	"os"
	"path/filepath"
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

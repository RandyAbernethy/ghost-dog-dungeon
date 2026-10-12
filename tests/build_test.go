package tests

import (
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMakeAllBuildsEachSupportedPlatform(t *testing.T) {
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(packageDir)
	// Track build inputs so Go's test cache also notices Makefile and game changes.
	if _, err := os.ReadFile(filepath.Join(repo, "Makefile")); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(repo, "src"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			if _, err := os.ReadFile(filepath.Join(repo, "src", entry.Name())); err != nil {
				t.Fatal(err)
			}
		}
	}
	dir := filepath.Join(t.TempDir(), "release binaries")
	cmd := exec.Command("make", "all", "BIN_DIR="+dir, "GO="+filepath.Join(runtime.GOROOT(), "bin", "go"))
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make all: %v\n%s", err, output)
	}
	for _, platform := range []struct{ os, arch string }{
		{"linux", "amd64"}, {"linux", "arm64"},
		{"windows", "amd64"}, {"windows", "arm64"},
		{"darwin", "amd64"}, {"darwin", "arm64"},
	} {
		name := "ghostdog-" + platform.os + "-" + platform.arch
		if platform.os == "windows" {
			name += ".exe"
		}
		path := filepath.Join(dir, name)
		info, err := buildinfo.ReadFile(path)
		if err != nil {
			t.Fatalf("read compiled binary %s: %v", name, err)
		}
		settings := map[string]string{}
		for _, setting := range info.Settings {
			settings[setting.Key] = setting.Value
		}
		if settings["GOOS"] != platform.os || settings["GOARCH"] != platform.arch || settings["CGO_ENABLED"] != "0" {
			t.Fatalf("%s has incorrect build settings: %v", name, settings)
		}
		if platform.os == runtime.GOOS && platform.arch == runtime.GOARCH {
			run := exec.Command(path, "--version")
			run.Dir, run.Env = t.TempDir(), append(os.Environ(), "HOME="+t.TempDir())
			if output, err := run.CombinedOutput(); err != nil || !strings.HasPrefix(string(output), "Ghost Dog Dungeon version ") {
				t.Fatalf("run native release binary: %v\n%s", err, output)
			}
		}
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 6 {
		t.Fatalf("expected exactly six release binaries, got %d: %v", len(files), err)
	}
}

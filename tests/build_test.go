package tests

import (
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMakeAllBuildsEachSupportedPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("the release matrix runs in the full suite")
	}
	makePath, err := exec.LookPath("make")
	if err != nil {
		if runtime.GOOS == "linux" {
			t.Fatal("full Linux verification requires make: ", err)
		}
		t.Skip("make is unavailable; the full Linux CI job verifies release builds")
	}
	trackGameBuildInputs(t)
	repo := filepath.Dir(testPackageDir)
	dir := filepath.Join(t.TempDir(), "release binaries")
	args := []string{"all", "BIN_DIR=" + filepath.ToSlash(dir), "GO=" + filepath.ToSlash(filepath.Join(runtime.GOROOT(), "bin", "go"))}
	if runtime.GOOS == "windows" {
		sh, err := exec.LookPath("sh")
		if err != nil {
			t.Skip("release recipes need a POSIX shell; Linux CI verifies release builds")
		}
		args = append(args, "SHELL="+filepath.ToSlash(sh))
	}
	cmd := testCommand(t, 2*time.Minute, makePath, args...)
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
			run := testCommand(t, 30*time.Second, path, "--version")
			run.Dir = t.TempDir()
			run.Env = testEnvironment(run.Dir)
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

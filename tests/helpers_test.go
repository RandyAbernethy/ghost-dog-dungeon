package tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	testRoot       string
	testPackageDir string
	testBinaryOnce sync.Once
	testBinary     string
	testBuildErr   error
)

func TestMain(m *testing.M) {
	var err error
	testRoot, err = os.MkdirTemp("", "ghostdog-tests-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	testPackageDir, err = os.Getwd()
	if err == nil {
		err = os.Mkdir(filepath.Join(testRoot, "home"), 0700)
	}
	if err == nil {
		for _, key := range []string{"HOME", "USERPROFILE"} {
			if err = os.Setenv(key, filepath.Join(testRoot, "home")); err != nil {
				break
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if cleanupErr := os.RemoveAll(testRoot); cleanupErr != nil {
			fmt.Fprintln(os.Stderr, cleanupErr)
		}
		os.Exit(1)
	}
	code := m.Run()
	if err := os.RemoveAll(testRoot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}

// Windows record lookup uses USERPROFILE; Unix uses HOME. Replace inherited
// entries rather than appending duplicate keys with platform-dependent handling.
func testEnvironment(home string) []string {
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(key, "HOME") && !strings.EqualFold(key, "USERPROFILE") {
			env = append(env, entry)
		}
	}
	return append(env, "HOME="+home, "USERPROFILE="+home)
}

func testCommand(t *testing.T, timeout time.Duration, name string, args ...string) *exec.Cmd {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	return cmd
}

// Keep the executable until the package finishes, so scenarios share one build.
func gameBinary(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("compiled-executable scenarios run in the full suite")
	}
	testBinaryOnce.Do(func() {
		trackGameBuildInputs(t)
		name := "ghostdog"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		testBinary = filepath.Join(testRoot, name)
		build := testCommand(t, 2*time.Minute, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", testBinary, "./src")
		build.Dir = filepath.Dir(testPackageDir)
		if output, err := build.CombinedOutput(); err != nil {
			testBuildErr = fmt.Errorf("build game: %w\n%s", err, output)
		}
	})
	if testBuildErr != nil {
		t.Fatal(testBuildErr)
	}
	return testBinary
}

func gameCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	cmd := testCommand(t, 30*time.Second, gameBinary(t), args...)
	cmd.Dir = t.TempDir()
	cmd.Env = testEnvironment(cmd.Dir)
	return cmd
}

// Go's test cache cannot observe source reads made by the nested go build.
// Track them here even when a host skips the optional release-matrix test.
func trackGameBuildInputs(t *testing.T) {
	t.Helper()
	repo := filepath.Dir(testPackageDir)
	for _, name := range []string{"go.mod", "Makefile", "src/assets/puppy.txt"} {
		if _, err := os.ReadFile(filepath.Join(repo, name)); err != nil {
			t.Fatal(err)
		}
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
}

# Working on Ghost Dog Dungeon

This is a Go terminal roguelike with ten main dungeon floors and an optional hidden crypt. Keep it readable, fair,
and reproducible from a seed.
Keep the save format backward compatible.


## Workflow

- Inspect `git status --short` before editing; preserve unrelated changes. The current request takes precedence
  over earlier design assumptions.
- Trace behavior through input, action, enemy turn, display, and persistence; fix the underlying behavior.
- Prefer focused changes. For risky structural rewrites, validate a temporary candidate before replacing working
  source. Separate structural moves from behavior changes where practical.
- Work in this repository unless isolation is needed. Do not commit, push, publish, or overwrite existing player
  executables, saves, or records unless requested.
- When changing behavior or build interfaces, update the affected README sections, gameplay contracts, and regression
  tests together. Update this file when working conventions or shared safeguards change.
- Share concise progress updates during longer work. Report results, verification, and remaining limitations honestly.


## Source organization

Keep `go.mod`, the README, and the Makefile at the root; supporting documentation lives in `docs/`.
Game source and tests needing private state share one Go package in `src/`. Use `tests/` for black-box executable
tests and ignored `bin/` for game binaries. Keep Monte Carlo code and data in `montecarlo/`, with its binaries and
temporary files under ignored `montecarlo/bin/` and `montecarlo/tmp/`. Keep related behavior together; avoid large
miscellaneous files and add abstractions only for a real boundary.

Start with `src/main.go` for startup, `src/commands.go` for actions, `src/render.go` for presentation, and
`src/save.go` for persistence. Keep OS APIs in `src/terminal_{linux,darwin,other}.go` and Linux PTY tests in
`src/terminal_linux_test.go`.


## Go quality

- Use `gofmt`, idiomatic names, and small cohesive functions. Comments should explain constraints or decisions.
- Prefer the standard library; add dependencies only for a clear need.
- Return useful errors at I/O and startup boundaries. Do not silently discard failures or panic for expected errors.


## Shared gameplay contracts

- Use the game's saved RNG for all gameplay randomness. Avoid wall-clock or global randomness after seeding. Do not
  choose outcomes by ranging over a map.
- Keep rendering and inspection free of gameplay mutations, except existing codex visibility bookkeeping.
- Keep map symbols unique across monster species, item types, trap kinds, and other features. Reserve `F`/`f` for
  fountains and use the shared catalog in `src/glyphs.go`; older saves must display current symbols without state repair.
- Successful gameplay actions spend one turn. Reading event choices, reading the book, escaping a command screen,
  saving, and failed actions do not.
- Keep the complete current turn's log in action order, beginning with the player's action. Replace it only when
  another action spends a turn; never truncate busy turns.
- Save/load must preserve the full dungeon, RNG, rewards, and challenge progress exactly, without repairing,
  rerolling, or renewing one-time rewards. Use one list of secret-room roots.
- Use `g.current()` for the active area. The crypt is saved separately from the ten main floors; while inside it,
  `currentLevel` retains the entrance floor for the return journey. Never use that index to select active terrain.
- Every save creates a fresh timestamped snapshot under `--save-file-path` (default: current directory). Create missing
  directories at startup and report useful errors. Loading preserves current save-directory and record-storage
  settings; never overwrite the source snapshot or previous saves.
- Ghost Dog acts and accompanies the player only when freed, alive, and unchained. Use `g.dog.active()` for actions and
  following, and `g.dogPresent()` for targeting or passive healing on the current floor. A chained dog remains on his
  saved anchor floor until the player walks onto his tile. Preserve combat expectations in
  `src/dog_combat_test.go` and the companion tests in `src/main_test.go`; revival requires explicit mechanics.
- Count kills and defeated difficulty once per dead monster, including companion, consumable, and guardian kills.
  Award death rewards once through the common death handler for every kill source. Only finished runs submit records.

Before changing generation, balance, encounters, equipment, events, or movement effects, read the relevant sections of
[Gameplay contracts](docs/gameplay-contracts.md). That document holds the specialized rules and regression-test references.


## Verification

See [Build](README.md#build) for tool prerequisites and local build commands.

For Go changes, run these checks from the repository root, choosing an unused build path under `bin/` or a temporary
directory:

```sh
gofmt -w <changed Go files>
go build -o <temporary binary path> ./src
go test ./...
go vet ./...
git diff --check
```

- For documentation-only changes, check links and `git diff --check`; builds and gameplay tests are unnecessary.
- Use `go test -short ./...` for development feedback; run the full suite before completing Go changes.
- Test observable behavior and invariants. Use explicit arena fixtures for combat tests. For generation or balance
  changes, preserve and extend the seed sweeps referenced in the gameplay contracts.
- Exercise new commands through the command loop and save/load, retaining compiled-executable regression coverage.
  Package tests may build `.` from `src/`; black-box tests must build `./src` from the repository root.
- `go test ./...` already covers `make all` through `tests/build_test.go`, using a temporary `BIN_DIR` for Linux,
  Windows, and macOS on AMD64 and ARM64. Full Linux runs require it; other hosts skip the matrix if build tools are
  unavailable. Do not repeat it after a successful full suite without a new concern.
- For terminal changes, also verify raw keys on native Linux. Cross-compilation does not verify foreign console
  behavior; Windows uses the command-and-Enter fallback. Report unavailable runtime checks.
- Use `t.TempDir()` for test saves and records. Reuse the executable helpers for shared builds, process timeouts,
  and isolation of both `HOME` and Windows `USERPROFILE`.
- Once relevant checks pass, avoid repeating or expanding them without a new change, failure, or unresolved concern.
- For security remediation, use at least the Go version in `go.mod` and scan the exact rebuilt binaries as well as
  source. Follow [Security scans](docs/security.md); do not suppress findings to obtain a clean result.

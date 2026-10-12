# Working on Ghost Dog Dungeon

This is a Go terminal roguelike with ten dungeon floors. Keep the game readable, fair, reproducible from a seed.
This is the first release: keep the code simple and support only the current save format. Reject unsupported
formats with a helpful error, and notify the user when a save-format change requires a new game.


## Workflow

- Read the user's current request and inspect `git status --short` before editing. Preserve unrelated changes. The
  user's latest instructions take precedence over earlier design assumptions.
- Trace behavior from input through the game action, enemy turn, display, and persistence. Fix the underlying behavior
  rather than only its message.
- Prefer focused changes. For broad refactors, prepare and test a temporary candidate before replacing working source.
  Move behavior first, then change it; preserve declarations and comments.
- Keep work in the current repository unless isolation is needed. Do not commit, push, publish, or overwrite the tracked
  game executable unless requested.
- Share concise progress updates during longer work. Report the result, relevant verification, and any remaining limitation honestly.


## Source organization

Keep `go.mod`, documentation, and the Makefile at the root. All game source and tests that need private game state
live in `src/`, in the same Go package. Use `tests/` for black-box executable tests that do not import game internals.
Keep local build output in ignored `bin/`; do not move or overwrite a user's existing executable or save files.
The file map below is relative to `src/`.

- `main.go`: entry point, flags, startup errors.
- `model.go`, `game.go`, `world.go`: data types, game initialization, common spatial helpers and seeded randomness.
- `dungeon.go`, `layouts.go`, `generation.go`: dungeon plans, geometry, placement, and fountains.
- `secrets.go`: independent and nested secret rooms, discovery geometry, rewards, and escape sanctum.
- `themes.go`, `difficulty.go`, `rosters.go`, `encounters.go`, `features.go`, `events.go`: floor identity, difficulty budgets,
  balanced encounters, optional rooms, and interactive events.
- `combat.go`, `monsters.go`, `companion.go`, `consumables.go`, `blink.go`: combat, monster definitions, Ghost Dog, and usable items.
- `monster_variety.go`, `equipment_variety.go`: additional depth-specific monster and equipment choices.
- `relics.go`, `relic_powers.go`, `epic_monsters.go`: rare nested-room relics, equipped powers, and optional Epic encounters.
- `equipment.go`, `loot.go`, `stories.go`, `records.go`: equipment, collection, discovered stories, and personal records.
- `commands.go`, `terminal.go`, `render.go`, `save.go`: actions, terminal input, presentation, and persistence.
- `terminal_linux.go`, `terminal_darwin.go`, `terminal_other.go`: platform-specific terminal calls. Keep operating
  system APIs out of shared input handling, and isolate Linux PTY tests in `terminal_linux_test.go`.

Keep related behavior together in this package. Add a package or abstraction only when it simplifies a real boundary. Avoid rebuilding a large miscellaneous source file.


## Go quality

- Use `gofmt`, idiomatic names, small cohesive functions, and explicit data types. Comments should explain a constraint
  or decision rather than restate the code.
- Prefer the standard library. Add dependencies only for a clear need.
- Return errors with useful context at I/O and startup boundaries. Do not silently discard errors or use panics for expected failures.
- Use the game's saved RNG for all gameplay randomness. Avoid wall-clock or global randomness after seeding. Do not
  choose outcomes by ranging over a map.
- Bound generation retries and provide a connected fallback. Reserve features before placing competing actors or items.
- Keep rendering and inspection free of gameplay mutations, except existing codex visibility bookkeeping. Reading
  screens and invalid commands must not spend turns.
- Save only current game state; do not add compatibility migrations for development saves. State must survive save/load
  exactly, without repairing the dungeon, rerolling gifts, or renewing one-time rewards. Use one list of secret-room roots.
- Every save creates a fresh timestamped snapshot in the directory selected by `--save-file-path` (default: current
  working directory). Create missing directories at startup and report useful errors. Loading must preserve the current
  save-directory and record-storage settings; never overwrite the source or previous snapshots.


## Gameplay contracts

- Generate 5–8 secret chambers per new game, counting nested chambers and the final escape sanctum. Ordinary floors may
  have zero, one, two, or rarely three chambers. Nesting is occasional and has no fixed floor.
- Keep ordinary terrain connected, stairs and rewards reachable, unrelated secret rooms separate, and undiscovered
  chambers concealed. Inner chambers require separate discovery after their parent is opened.
- Every secret chamber has its own fountain and side-story scroll. The Book of Ghost Dog shows only collected stories.
- Ordinary monster counts vary within calibrated floor difficulty budgets. Keep ordinary budgets and encounter totals strictly
  increasing, preserve the previous health and damage averages, and cap ranged attackers, disabling powers, and
  duplicate species. Ordinary equipment variety must preserve average damage, reach, and armor trait averages. Use saved RNG for all
  selection; new types and items may be absent from a run. Keep entry tiles clear and avoid crowding newly arrived
  players. Reserve one distant guard position for the Dread Lich.
- God armor and Artifact weapons are explicit rare exceptions to ordinary loot balance: only generate room treasure
  in inner secret chambers on floors 7–9, with a 25% chance per eligible chamber and at most one room relic per dungeon. Replace
  ordinary treasure so fountains and side-story scrolls still fit. Never roll relics on discovery, revisiting, or loading.
- Defeated Epic monsters guarantee one additional elite treasure at their final position, independently of nesting:
  God armor, an Artifact weapon, a Starfire Orb, a Sun Orb, Phoenix Ash, or Ghost Dog Recall. Use the saved RNG, avoid
  duplicate equipment, and award once through the common death handler for every kill source. Save/load must preserve
  the dropped or collected reward without rerolling or recreating it.
- Optional Epic monsters only occur on floors 7–9, with at most one per dungeon, outside ordinary difficulty budgets
  and guardian pools. Keep their initial position bypassable and distant from stairs; they act once every three turns.
  Gear powers require equipped pieces and must not revive Ghost Dog. God protection is powerful but not invulnerability.
- Award kill count and defeated difficulty once per dead monster, including companion, consumable, and stairwell
  guardian kills. Only finished runs submit records. Loading must preserve the full challenge progress and dungeon.
- Ghost Dog accompanies the player only when freed and alive. Preserve his existing combat behavior; revival belongs to
  explicit revival mechanics.
- Blink Stones exclude all secret chambers, including discovered and additional rooms, and move active companions together.
- Fountains refill on leaving their floor. Event gifts and shrine blessings are one-use and stay used after revisiting or loading.
- Successful gameplay actions spend one turn. Reading event choices, reading the book, escaping a command screen,
  saving, and failed actions do not.
- Keep the complete current turn's event log in action order, beginning with the player's action. Replace it when
  another action spends a turn; reading screens, saving, and failed actions preserve it. Never truncate busy turns.


## Verification

Run the applicable checks from the repository root:

```sh
gofmt -w <changed Go files>
go build -o <temporary binary path> ./src
go test ./...
go vet ./...
git diff --check
```

- Test observable behavior and important invariants, not merely the implementation's structure. Use explicit arena
  fixtures for combat tests instead of relying on a particular generated floor.
- For generation changes, check many seeds for connectivity, feature overlap, hidden-room access, room-count
  distribution, actual geometric variety, difficulty bounds, and seed reproduction.
- Exercise new commands through the command loop and save/load. Keep regression coverage for the real compiled
  executable and raw terminal keys.
- Build the game with `go build -o bin/ghostdog ./src` from the repository root. Package-level tests in `src/` may
  build `.` from their package directory. Black-box tests in `tests/` must build `./src` from the repository root.
- For terminal or build changes, verify `make all` for Linux, Windows, and macOS on AMD64 and ARM64. Use a temporary
  `BIN_DIR` during tests; keep the native Linux raw-key regression. Cross-compilation alone does not verify foreign
  console behavior. Windows currently uses the command-and-Enter fallback.
- Use `t.TempDir()` for saves and records. Give executable tests a temporary home directory; never overwrite the
  player's real records or saves.
- Once relevant checks pass, avoid repeating or expanding them without a new change, failure, or unresolved concern.

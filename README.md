# Ghostdog Dungeon

A dungeon crawler featuring a ghost dog companion. Enter the dungeon, free the ghost dog, reach the tenth floor, and
escape thorough the hidden passage... but beware the Dread Lich and his crypt protector the Under Lich!

For hidden content, equipment catalogs, encounter tactics, and ending details, see the
[gameplay guide (spoilers)](docs/gameplay-guide.md).


## TLDR

To play a downloaded copy, execute the binary for your platform in a shell:

```sh
./ghostdog
```

Each new or loaded game displays its version, dungeon seed, an ASCII puppy portrait, personal records, and the map.
The portrait appears once at startup without an extra keypress. Its reusable plain-text source is
[puppy.txt](src/assets/puppy.txt), embedded in the executable so no separate asset files are needed to play.

To build and run from source with Go 1.26.9 or newer, at the repository root run:

```sh
go build -o bin/ghostdog ./src
./bin/ghostdog
```

Windows uses command-and-Enter input. Unix shell versions respond immediately when the key command is pressed. You can
run the game on Windows in WSL for a Unix-like experience. See [Build](#build) for build tool requirements.

Use `./bin/ghostdog --help` for command line help. Available flags are:

| Flag                         | Purpose                                                                             |
| ---------------------------- | ----------------------------------------------------------------------------------- |
| `--seed <N>`                 | Set a dungeon seed.                                                                 |
| `--save-file-path  <dir>`    | Set the save file directory; created if missing. Defaults to the current directory. |
| `--load-file  <file>`        | Load a save file at startup.                                                        |
| `--records-file  <file>`     | Choose a personal-records file. Defaults to `~/.ghost-dog-data.json`.               |
| `--version`                  | Print the game version and exit.                                                    |

When `--seed` is not set, each new game uses the current time in nanoseconds as its seed, producing a different dungeon
each run. Explicitly setting a seed (e.g. `--seed 42`) allows you to create that seed's dungeon repetitively.


## Build

This code base requires Go 1.26.9 or newer. Makefile recipes require `make` and a POSIX-compatible shell, such as Bash
or Git Bash (WSL is also a good option on Windows). Local builds land in `./bin/`.

The repository layout is:

```text
.github/            GitHub action CI configuration
bin/                Local build output (git ignored)
docs/               Spoiler guide, developer gameplay contracts, and security scans
montecarlo/         Optional timed experiments with a state-aware automated player
src/                Game source and tests that exercise private game state
tests/              Black-box tests of the compiled executable
.gitignore          Lists files and folders that should not be checked in
.markdownlint.json  Defines markdown rules
AGENTS.md           Agent documentation
go.mod              Go module definition
LICENSE             Apache 2 license file
Makefile            Build automation
README.md           This file
```

This README covers building, running, and general play. For repository workflow and verification, read
[AGENTS.md](AGENTS.md). Intended gameplay invariants and their source and test references live in the
[gameplay contracts (spoilers)](docs/gameplay-contracts.md).

Go's package tests live beside the source they exercise. Tests in `tests/` build the game and run it from temporary
directories with isolated saves and home directories. For a fast development check from the repository root:

```sh
go test -short ./...
```

Short mode skips compiled-executable scenarios, the release build matrix, and the large generation seed sweeps.
Run the complete checks before committing changes:

```sh
go test -count=1 -timeout 5m ./...
go vet ./...
git diff --check
```

The full suite runs a complete campaign through the compiled executable. It compares uninterrupted play with
save/load continuation and checks saved checkpoints and final records. The route and coverage are documented in
the [campaign replay contracts (spoilers)](docs/gameplay-contracts.md#campaign-replay).

Optional timed experiments with a state-aware automated player are documented in [montecarlo/](montecarlo/README.md).
They run separately from the regression suite and keep results in `montecarlo/data/`, with binaries and temporary
files under `montecarlo/bin/` and `montecarlo/tmp/`.

The `make all` command builds all six release targets in `./bin/` when run under `sh` or `bash`. Windows GNU Make can
use `make all SHELL=sh` with Git Bash's `sh` on `PATH`.

Github Actions CI runs the full suite and vet on Ubuntu 24.04 with Go 1.26.9 and Ubuntu 26.04 with Go 1.27.2, plus
native Windows and macOS with Go 1.27.2. The raw-key PTY regression runs on Linux; Windows tests exercise
command-and-Enter input. The minimum Go version includes standard-library security fixes. See [security
scans](docs/security.md).

The Makefile provides:

- `make build`
- `make run`
- `make test`
- `make test-fast`
- `make vet`
- `make fmt`
- `make check`

Build releases for Linux, Windows, and macOS with:

```sh
make all
```

This creates six binaries in `./bin/`:

- `ghostdog-linux-amd64`
- `ghostdog-linux-arm64`
- `ghostdog-windows-amd64.exe`
- `ghostdog-windows-arm64.exe`
- `ghostdog-darwin-amd64` (Intel Macs)
- `ghostdog-darwin-arm64` (Apple Silicon)

Build one platform with:

- `make linux`
- `make windows`
- `make macos`

Override the output directory with `make all BIN_DIR=path`. Cross-builds use only Go, with CGO disabled, and need no
Windows or macOS SDK. `make build` builds the current machine's version as `bin/ghostdog`.


## Playing

On supported Unix terminals, keys act immediately. In command-and-Enter mode, enter a command and press Enter. The map
legend, nearby prompts, inventory, and inspection screen explain what you have discovered. Each monster species, item
type, and map feature has a distinct symbol. **F** means a healing fountain; **f** means a spent fountain. Inventory and
inspection show item symbols, and **c** shows discovered enemy symbols.

| Action                             | Immediate key                     | Command-and-Enter input                |
| ---------------------------------- | --------------------------------- | -------------------------------------- |
| Move or attack in a direction      | **wasd** / arrows; **qezx** diags | `w`, `a`, `s`, `d`, `q`, `e`, `z`, `x` |
| Wait                               | **.**                             | `.`                                    |
| Use stairs                         | **<** / **>**                     | `<` / `>`                              |
| Use a potion                       | **p**                             | `p`                                    |
| Use items listed in your inventory | **f b g u t o n r**               | The corresponding letter               |
| Drink from a nearby fountain       | **v**                             | `v`                                    |
| Interact with a nearby feature     | **y**                             | `interact` or `y`                      |
| Search nearby                      | **k**                             | `search` or `k`                        |
| Open inventory                     | **i**                             | `inventory` or `i`                     |
| Open equipment                     | **E**                             | `equipment`                            |
| Read the Book of Ghost Dog         | **B**                             | `book`                                 |
| View personal records              | **j**                             | `challenges` or `j`                    |
| Open the codex                     | **c**                             | `codex` or `c`                         |
| Inspect the area                   | **m**                             | `inspect` or `m`                       |
| Return to the map                  | **Esc**                           | `map`                                  |
| Save / load                        | **S** / **L**                     | `save` / `load`                        |
| Show help                          | **?**                             | `h`                                    |
| Quit                               | **Ctrl-C**                        | `quit`                                 |

Successful gameplay actions spend one turn. Reading screens, viewing choices, saving, cancelling a command screen, and
failed actions do not. The event log shows the complete current turn, beginning with your action.

Attempting to move when constrained costs one turn. Most non-movement/attack actions can be completed (e.g. drink a
potion or engage warding) while movement is constrained.

Ghost Dog appears as **D** when he can act, or **d** when silver chains hold him. Move onto **d** to free him.
A chained companion stays in place, including when you use stairs or blink. He must be free to escape with you.


## Equipment

Equipment is kept in your pack. Press **i** for inventory or **E** for the equipment list. Equip a numbered item with
`:equip NUMBER` followed by Enter. Changing equipment spends one turn. Use `:drop NUMBER` to put an unequipped item on
the floor. You can pick dropped items up again by leaving their tile and walking back onto it.

When an interaction offers numbered choices, use `:choose 1`, `:choose 2`, or `:choose 3` followed by Enter. Press
**Esc** to cancel an unfinished `:` command or return to the map without spending a turn.

Armor guards against incoming damage. Reach weapons can strike from two tiles away: press the direction toward
an enemy in a straight line. If the adjacent tile is empty, you attack without moving; you can also strike over
Ghost Dog. Check the equipment screen for each item's damage, reach, and protection.


## Book of Ghost Dog and Records

Your inventory includes the **Book of Ghost Dog** from the start. Press **B** or type `book` in command-and-Enter mode
to read collected chapters and side stories without spending a turn. Uncollected stories remain undisclosed. A finished
run provides a **Your Adventure** chronicle summarizing the run, followed by challenges, personal records, and the
dungeon seed.

Press **j** or type `challenges` to see the current run and personal records:

- **Fewest turns:** the fastest successful escape. Successful actions and turns lost to entanglement count; reading,
  inspecting, saving, and failed actions do not.
- **Escape without Ghost Dog falling:** free Ghost Dog and escape with him alive and unchained, without him ever
  reaching zero health.
- **Most monsters killed:** the most kills in a run that ends in escape or death. Kills by Ghost Dog and consumables
  count; each monster counts once.
- **Most difficulty defeated:** the greatest sum of slain monsters' difficulty points in a finished run. Tougher
  monsters contribute more than weaker ones. Each defeated monster contributes to both kill records.

Records are saved automatically in `~/.ghost-dog-data.json` when you die or escape, without using the save command.
Personal records appear at game startup and game end, and are shared across runs and save files. Saves retain equipment,
book contents, challenge progress, and the generated dungeon without rerolling them.
At game end, a bold gold celebration panel calls out every **NEW RECORD!** and **RECORD TIED!**, with its category and
score. Your first qualifying result sets a record; a tie keeps the original record holder. Fastest escape records
require victory, and the Ghost Dog safe escape record also requires that he never fell and escaped alive with you.


## Saves

Only save format 3 is supported. Earlier saves require a new game; they are not converted. Personal records are
retained. Loading restores the saved dungeon and RNG exactly.

Each save creates a new `ghostdog-dungeon-YYYYMMDD-HHMMSS.json` snapshot, adding a numeric suffix if that name already
exists. `--save-file-path` selects the directory and creates it if missing; the default is the current working
directory. When you save a Loaded game a fresh snapshot is created, leaving the original file untouched. **S** saves
without spending a turn; **L** restores the most recently saved or loaded snapshot in the current session. The save
directory and records-file path are runtime settings and are not stored in the dungeon save.

```sh
./bin/ghostdog --seed 42 --save-file-path ./saves
./bin/ghostdog --load-file ./saves/ghostdog-dungeon-20261012-090807.json --save-file-path ./saves
```

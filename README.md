# Ghostdog Dungeon

A dungeon crawler featuring a ghost dog companion. Enter the dungeon, free the ghost dog, reach the tenth floor, and
escape thorough the hidden passage... but beware the Dread Lich and his protector the Under Lich!

For hidden content, equipment catalogs, encounter tactics, and ending details, see the
[gameplay guide (spoilers)](docs/gameplay-guide.md).


## TLDR

To play a downloaded copy, execute the binary for your platform in a shell:

```sh
./ghostdog
```

Each new game displays its version, dungeon seed, personal records, and the map.

To build and run from source with Go 1.26.9 or newer, use the repository root:

```sh
go build -o bin/ghostdog ./src
./bin/ghostdog
```

Windows uses command-and-Enter input. The Unix shell examples below can be run in WSL; see [Build](#build) for tool
requirements.

Use `./bin/ghostdog --help` for command line help. Available flags are:

| Flag | Purpose |
| --- | --- |
| `--seed N` | Choose a dungeon seed. |
| `--save-file-path directory` | Set the snapshot directory; created if missing. Defaults to the current directory. |
| `--load-file file` | Load a save at startup. |
| `--records-file file` | Choose a personal-records file. Defaults to `~/.ghost-dog-data.json`. |
| `--version` | Print the game version and exit. |

When `--seed` is not set, each new game uses the current time in nanoseconds as its seed, producing a different dungeon
each run. Explicitly setting a seed (e.g. `--seed 42`) allows you to create that seed's dungeon repetitively.


## Build

This code base requires Go 1.26.9 or newer. Makefile recipes require `make` and a POSIX-compatible shell, such as Bash
or Git Bash (or WSL is a good option on Windows). Local builds land in `./bin/`.

The repository layout is:

```text
.github/            GitHub action CI configuration
bin/                Local build output (git ignored)
docs/               Spoiler guide, developer gameplay contracts, and security scans
montecarlo/         Optional timed experiments with a state-aware automated player
src/                Game source and tests that exercise private game state
tests/              Black-box tests of the compiled executable
.gitignore          File that should not be checked in
.markdownlint.json  Defines Markdown rules
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
Run the complete checks before finishing Go changes:

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

Full Linux verification requires `make` and builds all six release targets in a temporary directory. On Windows and
macOS, that check runs when `make` and the required shell are available; native executable tests still run without them.
A separate `make all` creates release binaries to keep. Windows GNU Make can use `make all SHELL=sh` with Git Bash's
`sh` on `PATH`.

CI runs the full suite and vet on Ubuntu 24.04 with Go 1.26.9 and Ubuntu 26.04 with Go 1.27.2, plus native Windows and
macOS with Go 1.27.2. The raw-key PTY regression runs on Linux; Windows tests exercise command-and-Enter input.
The minimum Go version includes standard-library security fixes. Rebuild older binaries with a patched toolchain;
updating the source does not replace downloaded or previously built executables. See [security scans](docs/security.md).

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

On supported Unix terminals, keys act immediately. In command-and-Enter mode, enter a command and press Enter.
The map legend, nearby prompts, inventory, and inspection screen explain what you have discovered.
Each monster species, item type, and map feature has a distinct symbol. **F** always means a healing fountain;
**f** means a spent fountain. Inventory and inspection show item symbols, and **c** shows discovered enemy symbols.

| Action | Immediate key | Command-and-Enter input |
| --- | --- | --- |
| Move or attack in a direction | **wasd** or arrows; **qezx** diagonals | `w`, `a`, `s`, `d`, `q`, `e`, `z`, `x` |
| Wait | **.** | `.` |
| Use stairs | **<** / **>** | `<` / `>` |
| Use a potion | **p** | `p` |
| Use magic items listed in your inventory | **f b g u t o n r** | The corresponding letter |
| Drink from a nearby fountain | **v** | `v` |
| Interact with a nearby feature | **y** | `interact` or `y` |
| Search nearby | **k** | `search` or `k` |
| Open inventory | **i** | `inventory` or `i` |
| Open equipment | **E** | `equipment` |
| Read the Book of Ghost Dog | **B** | `book` |
| View personal records | **j** | `challenges` or `j` |
| Open the codex | **c** | `codex` or `c` |
| Inspect the area | **m** | `inspect` or `m` |
| Return to the map | **Esc** | `map` |
| Save / load | **S** / **L** | `save` / `load` |
| Show help | **?** | `h` |
| Quit | **Ctrl-C** | `quit` |

Successful gameplay actions spend one turn. Reading screens, viewing choices, saving, cancelling a command screen,
and failed actions do not. The event log shows the complete current turn, beginning with your action.

Movement holds cost one attempted movement turn. The status and log name their cause; you can still take other actions.

Ghost Dog appears as **D** when he can act, or **d** when silver chains hold him. Move onto **d** to free him.
A chained companion stays in place, including when you use stairs or blink. He must be free to escape with you.


## Equipment

Equipment is kept in your pack. Press **i** for inventory or **E** for the equipment list. Equip a numbered item with
`:equip NUMBER` followed by Enter. Changing equipment spends one turn. Use `:drop NUMBER` to put an unequipped item on
the floor. Inventory numbers cover gear and consumables. Dropping a consumable releases one from its stack; successful
drops spend one turn. You can pick dropped items up again by leaving their tile and walking back onto it.

When an interaction offers numbered choices, use `:choose 1`, `:choose 2`, or `:choose 3` followed by Enter. Omit the
colon for `equip`, `drop`, and `choose` in command-and-Enter mode. Press **Esc** to cancel an unfinished `:` command
or return to the map without spending a turn.

Armor guards against incoming damage. Reach weapons can strike from two tiles away: press the direction toward
an enemy in a straight line. If the adjacent tile is empty, you attack without moving; you can also strike over
Ghost Dog. Check the equipment screen for each item's damage, reach, and protection.


## Book of Ghost Dog and Records

Your inventory includes the **Book of Ghost Dog** from the start. Press **B** or type `book` in command-and-Enter mode
to read collected chapters and side stories without spending a turn. Uncollected stories remain undisclosed.
A successful escape has a brief epilogue. A finished run includes a **Your Adventure** chronicle based on what happened
during that run, followed by challenges, personal records, and the dungeon seed.

Press **j** or type `challenges` to see the current run and personal records:

- **Fewest turns:** the fastest successful escape, including the final stair action. Successful actions and turns lost
  to entanglement count; reading, inspecting, saving, and failed actions do not.
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

Save format 2 remains supported, including older saves with fewer optional fields. Loading preserves the saved
dungeon and RNG without adding new content or inventing past events. Unsupported formats, including format 1,
require a new game. Personal records are retained.

Each save creates a new `ghostdog-dungeon-YYYYMMDD-HHMMSS.json` snapshot, adding a numeric suffix if that name already
exists. `--save-file-path` selects the directory and creates it if missing; the default is the current working
directory. Loaded games also create fresh snapshots there, leaving the original file untouched. **S** saves without
spending a turn; **L** restores the most recently saved or loaded snapshot in the current session. The save directory
and records-file path are runtime settings and are not stored in the dungeon save.

```sh
./bin/ghostdog --seed 42 --save-file-path ./saves
./bin/ghostdog --load-file ./saves/ghostdog-dungeon-20261012-090807.json --save-file-path ./saves
```

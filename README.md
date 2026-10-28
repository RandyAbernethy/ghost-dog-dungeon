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
On an interactive terminal, the welcome screen ends with **Press enter to begin the descent! >**.
Press Enter to clear it and display gameplay; this costs no turn. Redirected command scripts skip the pause.
The portrait appears once at startup. Its reusable plain-text source is
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

The chapel introduction and gameplay turn log wrap between words at 80 visible columns, including the bullet and
continuation indentation. Color codes do not count toward the width. Snapshot-path announcements stay on one line
so paths can be copied exactly.

On supported Unix terminals, keys act immediately. In command-and-Enter mode, enter a command and press Enter.
The hidden crypt’s entrance bears a scratched warning about the Under Lich’s maze, leaving its travel, fountain
and magic restrictions for players to discover. Suppressed magic reports
that the crypt consumed its power without effect.

Start with movement, **i** for the inventory menu, **y** to interact, **m** to inspect, and help (**?** on Unix,
**h** in command-and-Enter mode). Help shows the complete control and feature reference for your input mode.
Nearby event prompts describe what you see and how to interact, leaving gifts, lever effects, and the collar's
history to be discovered. Objective prompts give the interaction control; shrine messages describe only who receives healing.
Vigil messages describe the spirit's condition; objective completion text comes before the reward's pickup description.
Action messages describe Ash's dispersal, wards, blinking and traps as they happen, leaving their rules to discovery
and reference screens. Equipment messages omit repeated turn-cost reminders.
Barrow Archers need a clear line of floor tiles to you before they can attack you or Ghost Dog; walls block their attacks.
The map shows survival information and a legend of nearby visible entities. Spell ward and strike bonus appear only
when nonzero; inventory retains their complete statistics, including zeros. Consumable counts and detailed statistics live
in inventory and records. Each monster species, physical item category, and map feature has a distinct symbol:
`?` scroll (including side stories), `=` orb, `%` charm, `!` potion, `*` stone, `)` weapon, `[` armor.
Warding and Frost Charms share **%**. Phoenix Ash is a stronger healing consumable (22–30 HP) and shares the potion
appearance **!** with healing potions (12–18 HP). All floor legends, inspection text and drop announcements show only
physical categories; pickup identifies items and reveals story contents. Consumable pickup messages give the use key;
inventory and gift offers provide item properties. **F** means a
healing fountain; **f** means a spent fountain. **c** opens the discovered-enemy codex.
Status, menus, hints and warnings wrap to the Unix terminal width, capped at 80 columns. Windows and redirected output
use a compact 60-column fallback; set `COLUMNS` to override it. The map itself requires 38 columns.

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
A failed command replaces the activity display with its own feedback, such as "You have no fire scrolls.", so
earlier attacks are not replayed. Reading screens retains the latest activity, and saves preserve it.

Attempting to move when constrained costs one turn. Most non-movement/attack actions can be completed (e.g. drink a
potion or engage warding) while movement is constrained.

Ghost Dog appears as **D** when he can act, or **d** when silver chains hold him.
The nearby legend includes **d chained Ghost Dog** before his rescue when he is visible within four tiles. Move onto **d** to free him.
A chained companion stays in place, including when you use stairs or blink. He must be free to escape with you.


Nearby fountain prompts show only the fountain and the drink key, without explaining its effects or refill rules.

## Equipment

Equipment is kept in your pack. Press **i** (or **E** on Unix) to open the inventory menu. In immediate-key mode,
use up/down to highlight an item, then **e** to equip or **d** to drop it directly. Enter opens details;
entering an item number followed by Enter also opens details. In command-and-Enter mode,
enter its number. Read the equipment comparison, then press Enter or **e** to equip; for a consumable, Enter or **u**
uses it. **d** drops a selected unequipped item or one consumable. **i** returns to the list; Esc or `map` closes it.
Pickup messages show the equipment name and armor slot, followed by stat differences from your equipped item
on a separate indented line.
Full attributes and special powers are available in inventory.
The main screen shows the equipped weapon's nonzero bonuses, and each weapon in the inventory list shows its own.
These include `+2 vs undead` for magical weapons and compact Artifact power summaries; weapons without bonuses show none.
Each armor piece in the inventory list shows guard, spell ward, and strike damage, including zero bonuses.
Each carried consumable shows its use key beside its count and map symbol (for example, `[use p]` for a Healing Potion).
Reading, selecting, cancelling, and failed actions are free. A successful equip, use, or drop spends one turn.
Unowned consumables are omitted. The `details` command (`:details` in immediate-key mode) shows the complete pack
report, companion rules, book progress, and all consumable descriptions.
Advanced `:equip NUMBER` and `:drop NUMBER` commands remain available. You can pick dropped items up again by leaving
their tile and walking back onto it.

When a ghost of a friendly villager offers gifts, press **1**, **2**, or **3** (followed by Enter in command-and-Enter mode).
The advanced `choose NUMBER` command also works. Press **Esc** to cancel an unfinished `:` command or return to
the map without spending a turn.

After a successful move or arrival within three tiles along an open approach to a hidden door, there is a **20% chance**
of one subtle hint about a draft, disturbed mortar, or burial seal in the turn log. Hints do not tell you to search.
Reading, inspecting and redrawing do not generate hints or retry the chance. The search command is **k**.
Clues to linked chambers appear only after opening their parent chamber. The Lich's death opens the escape sanctum.

Epics alternate normal attacks with warned signatures. Normal attacks reach two tiles and use half the signature's
damage range before ordinary defenses. Every four damaging hits, including Ash and consumables, build full rage.
When there is room to escape, full rage prepares a wide retaliation with **two actions** of warning: start moving
immediately and move again, interrupt it, use protection, or blink. Waiting to dodge until the last action may be fatal.
Marked tiles never follow you. Ash begins escaping wide warnings early. Epics remain optional and bypassable.
Epic turn logs announce awakening, anger and incoming attacks without explaining map symbols or attack routines.

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

Only save format 4 is supported. Earlier saves require a new game; they are not converted. Personal records are
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

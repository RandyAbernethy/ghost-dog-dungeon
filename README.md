# Ghostdog Dungeon

A dungeon crawler featuring a ghost dog companion. Enter the dungeon, free the ghost dog, reach the tenth floor, and
escape thorough the hidden passage... but beware the Dread Lich!


## TLDR

To play a downloaded copy, execute the binary for your platform in a shell:

```bash
$ ./ghostdog

Ghost Dog Dungeon version 1.0.3
Dungeon seed: 1792328803698777268

=== Personal Records ===
Fewest turns record: 1527 turns (seed 1791221079717482356).
Ghost Dog safe escape record: not achieved yet.
Most monsters killed record: 74 (seed 1792015100455577880).
Most difficulty defeated record: 4912 points (seed 1792271059679411539).

=== Ghost Dog Dungeon: Level 1/10 ===
overgrown catacombs — pillared halls
HP 46/46  Weapon Rusty Knife (3-7)  Armor 2 (block 1)  Potions 3  Fire 1  Blink 0  Ward 0  Sun 0  Frost 0  Starfire 0  Ash 0  Recall 0  Story 0/10
Reach 1  Spell ward 0  Strike bonus +0  Turns 0  Kills 0  Difficulty defeated 0
Ghost dog: not yet freed
######################################
#######################[##############
#############.!)?.........############
#############......@#..*s#############
#############.&...>......#############
###############.##.#.#################
######################################
######################################
######################################
######################################
######################################
######################################
######################################
######################################
######################################
######################################
######################################
######################################
Legend: @ you  D ghost dog  d chained ghost dog  > down  < up/out  F/f fountain/spent  ) weapon  [ armor  ! potion  ? scroll  * magic  G friendly ghost  + lever  & keepsake  ^ shrine
Here: press i for equipment, c for the codex, m to inspect, or k to search suspicious walls.
Dungeon events:
- You step into the dungeon. The door slams shut behind you.
- Find the ghost dog, delve through ten ever-larger floors, and uncover the hidden way out guarded by the Dread Lich.
  Keys act immediately: wasd/arrows, qezx, ., </>, p f b g u t o n r, v fountain, y interact, i inventory, E equipment, B book, j challenges, c codex, m inspect, k search, Esc map, S save, L load, Ctrl-C quit; type :equip NUMBER, :drop NUMBER, or :choose 1/2/3 + Enter
Command>
```

To build and run from source with Go 1.26.9 or newer, use the repository root:

```sh
go build -o bin/ghostdog ./src
./bin/ghostdog
```

Windows uses command-and-Enter input. The Unix shell examples below can be run in WSL; see [Build](#build) for tool
requirements.

Use `--help` for command line help (shown here for a source build):

```bash
$ ./bin/ghostdog --help

Usage: ghostdog-dungeon [--seed N] [--save-file-path directory] [--load-file file] [--records-file file] [--version]

Enter the dungeon, free the ghost dog, reach the tenth floor, and escape by finding the hidden way out beyond the Dread Lich.
Commands: wasd/arrows move, qezx diagonals, . wait, </> stairs, p f b g u t o n r items, v fountain, y interact, i inventory, E equipment, B book, j challenges, c codex, m inspect, k search, :equip NUMBER, :drop NUMBER, :choose 1/2/3, S save, L load, Ctrl-C quit
  -load-file string
        load this save file at startup
  -records-file string
        challenge records path; defaults to ~/.ghost-dog-data.json
  -save-file-path string
        directory for timestamped save files; created if missing (default ".")
  -seed int
        random seed for dungeon generation (default is current time in nanoseconds)
  -version
        print the game version and exit
```

When `--seed` is not set, each new game uses the current time in nanoseconds as its seed, producing a different dungeon
each run. Explicitly setting a seed (e.g. `--seed 42`) allows you to create that seed's dungeon repetitively.


## Build

This code base requires Go 1.26.9 or newer. Makefile recipes require `make` and a POSIX-compatible shell, such as Bash or
Git Bash (or WSL is a good option on Windows). Local builds land in `./bin/`.

The repository layout is:

```text
.github/            GitHub action CI configuration
bin/                Local build output (git ignored)
docs/               Developer gameplay contracts and regression-test references
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

This README covers building, running, and player-facing mechanics. For repository workflow and verification, read
[AGENTS.md](AGENTS.md). Intended gameplay invariants and their source and test references live in [Gameplay
contracts](docs/gameplay-contracts.md).

Go's package tests live beside the source they exercise. Tests in `tests/` build the game and run it from temporary
directories with isolated saves and home directories. For a fast development check from the repository root:

```sh
go test -short ./...
```

Short mode skips compiled-executable scenarios, the release build matrix, and the large generation, secret-room, relic,
and roster seed sweeps. Run the complete checks before finishing Go changes:

```sh
go test -count=1 -timeout 5m ./...
go vet ./...
git diff --check
```

The full suite runs a fresh seed-6 campaign through all ten floors, rescue, combat, equipment, story collection, and
escape. It compares uninterrupted play with save/load continuation and checks saved checkpoints and final records.
The command route is in [tests/testdata/seed-6.txt](tests/testdata/seed-6.txt).

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


## Dungeon Generation

> N.B. Reading this section will help you understand how dungeon generation works and, thus, may spoil some of the fun.

Each run varies the dungeon, treasure, fountain locations, and monster groups. Floors can be winding caves, narrow crypt
passages, connected chambers, or pillared halls, with a chapel, barracks, catacomb, or kennel theme. Themes favor
different enemies and supplies within the floor's usual difficulty range. Monster groups can be scattered patrols, small
packs, treasure guards, or a guard with a spellcaster.

Monster counts and groups vary within each floor's difficulty budget, with danger increasing as you descend. Limits
on ranged attackers, disabling powers, and repeated species keep encounters varied within those budgets.

Used fountains refill when you leave their floor, so you can return for more healing if you are willing to face the
stairwell guardians, 2-4 monsters which spawn on a floor when you descend below it. Drinking from a fountain restores
4-8 health to you and 8-12 health to Ghost Dog. Stairwell guardians use smaller budgets appropriate to their floor.

Searching suspicious walls allows the player to discover secret passages. Each game has 5-8 secret chambers, including
the escape sanctum and possible linked chambers. Every secret chamber contains a fountain and its own side-story
scroll. After discovering a chamber, search its walls for hidden doors to further chambers.

Floor monster pools include new monster types with each level, while the Dread Lich is always on floor ten. Monsters
such as Tunnel Bats, Cave Crawlers, Rust Guards, Barrow Archers, Grave Acolytes, Ember Hounds, Obsidian Guards, and
Hollow Wardens can appear, though none is guaranteed in a run.

Optional rooms include guarded armories, healing shrines, abandoned kennels, and treasure vaults. Look for friendly
ghosts (**G**), shortcut levers (**+**), Ash's keepsakes (**&**), and shrines (**^**). Stand on or next to one and press
**y** (or type `: interact`) to interact. A friendly ghost offers three different, randomly chosen gifts: armor, a
weapon, a potion, a Blink Stone, a Fire Scroll, a Warding Charm, a Sun Orb, a Frost Charm, a Starfire Orb, Phoenix Ash,
or a Ghost Dog Recall Scroll. Equipment matches the floor's usual loot range. Accept one gift with `:choose 1`, `:choose
2`, or `:choose 3` followed by Enter (omit the colon in line-input mode). Each ghost keeps its choices when you return
or load a save. Reading the offer costs no turn; accepting a gift or activating another event spends one turn. Event
rewards can be used once, and stay used after revisiting or loading. A keepsake needs a living Ghost Dog nearby. Shrines
heal you and a living Ghost Dog once. Approaching within one tile of an unused event or fountain shows its description
and interaction key. The prompt repeats only after you move farther away and return, including when you arrive by stairs
or a Blink Stone.

Use **g** to spend a Blink Stone and teleport yourself and Ghost Dog away from monsters on the current floor.
The stone chooses the safest available spot outside secret rooms.

Equipment is kept in your pack, press **i** for inventory or **E** for the equipment list. Equip a numbered item with
`: equip NUMBER` followed by Enter. Changing equipment spends one turn. Use `: drop NUMBER` to put an unequipped item on
the floor. Inventory numbers cover gear and consumables. Dropping a consumable releases one from its stack; successful
drops spend one turn. You can pick dropped items up again by leaving their tile and walking back onto it. Press **Esc**
to return to the map from a command screen or cancel an unfinished `:` command without spending a turn. In line-input
mode, you can also type `map` and Enter. The event log shows the complete current turn, beginning with your action.


## Weapons and Armor

Weapon and Armor types:

- Reach weapons trade some damage for the ability to strike from two tiles away
  - Press the direction key in direct line with the enemy position
  - If the adjacent tile is empty you will not move
  - If Ghost Dog is between you and the enemy you will attack over Ghost Dog
- Armor guards against incoming damage.
- Runespun armor trades guard for extra protection against damaging spells.
- Hunter's armor trades guard for extra weapon damage.

Each floor selects from its weapon and armor pool, so equipment need not appear every run.

Later chambers in a secret-room chain on a deep floor can contain a supreme relic. There is at most one relic generated
as room treasure per dungeon. Defeating an Epic monster can award an additional relic, even without linked rooms.
These pieces are never ordinary loot or friendly-ghost gifts.

| God armor              | Protection and power                                                                              |
| ---------------------- | ------------------------------------------------------------------------------------------------- |
| Aegis of the Last Dawn | Body: 24 guard, 6 spell ward, +2 strike. Reduces living Ghost Dog's damage by 6 within two tiles. |
| Oracle Crown           | Head: 18 guard, 12 spell ward, +4 strike. Clears and prevents hexes and entanglement.             |
| Starthorn Greaves      | Leg: 18 guard, 6 spell ward, +3 strike. Surviving a damaging melee hit returns 12 damage.         |

God armor gives immense protection, but can absorb at most 90% of a physical blow. Small wounds and powerful enemies can
still kill its wearer. Carrying a piece does not activate its power; equip it first.

| Artifact weapon    | Damage and power                                                                          |
| ------------------ | ----------------------------------------------------------------------------------------- |
| Dawnbreaker        | 30–42; steals up to 8 health per successful strike, based on damage actually dealt.       |
| Winter's Last Word | 28–38, reach 2; each hit has a 25% chance to freeze a surviving target for its next turn. |
| Worldsplitter      | 38–52; ignores monster armor.                                                             |
| Storm Shepherd     | 30–40, reach 2; arcs 12 damage into one other monster within two tiles of its target.     |

All Artifacts are magical and receive the usual bonus against undead. Powers, rarity, and generated rewards survive
save/load; returning to a room cannot reroll its treasure.


## Epic Encounters

Deep floors can have a rare **Epic** encounter:

- **Iron Colossus:** crushes warding charms and blocks 9 damage per hit.
- **Grave Titan:** undead; heals itself after damaging melee attacks.
- **Ashen Wyrm:** damaging melee strikes erupt in fire around adjacent foes.
- **Hollow Reaper:** undead; reflects close-range hits. Reach avoids the reflection.
- **Frost Monarch:** undead; damaging melee strikes freeze your feet when you are adjacent, costing one attempted move.
- **Storm Leviathan:** damaging melee strikes arc lightning into you and living Ghost Dog within two tiles.
- **Dusk Tyrant:** undead; damaging melee strikes hex you when you are adjacent, weakening your next strike.
- **Obsidian Golem:** its enormous stone shell blocks 12 damage per hit. Heavy weapons or Worldsplitter help.

A floor that generates God armor or an Artifact weapon as room treasure always gets one Epic. If a random Epic was
already generated on another floor, the relic floor adds a second encounter; there is at most one Epic per floor and
two per dungeon. Ordinary floors without relics keep the rare random encounter chance. Epic treasure drops already
come from an encounter on that floor and do not summon replacements. Discovery, revisiting, dropping carried gear,
and loading do not create new encounters.

Epics spawn away from entrances and stairs, with a route around them. They wake within four tiles, warn you when they
wake, and act once every three turns, letting you retreat. God armor or an Artifact helps, but careful use of reach,
consumables, and positioning still matters. Epics never return as stairwell guardians, and killing one is not needed
to defeat the Dread Lich or escape. The Oracle Crown prevents the Frost Monarch's binding and Dusk Tyrant's hex;
spell wards reduce the Storm Leviathan's lightning damage to you.

Each Epic also has a signature attack: **Hammerfall**, **Grave Cross**, **Cinder Lane**, **Soul Scythe**,
**Ice Bloom**, **Thunder Marks**, **Eclipse**, or **Shard Fan**, respectively. Within two tiles, it can prepare
the attack on its last rest turn. **Red X** tiles show exactly where it will strike on its next active turn,
replacing its usual attack or movement. You have one action to move clear; the tiles never follow you.
Your `@` and Ash's `D` turn red when threatened, and the warning names the attack below the map and in **m** inspection.
Ash spends his action dodging rather than attacking, and avoids walking or blinking onto marked tiles.
Freezing or stunning the Epic delays its warning; fleeing or death cancels it. Walls block warnings, and the
Epic does not prepare a pattern if every neighboring escape tile is blocked. Reading screens and saving are free.

Defeating any Epic guarantees one elite treasure on its final tile: rare equipment or a powerful consumable. Kills by
Ghost Dog, magic items, and gear powers earn the same reward. Walk onto the tile to collect it; equipment still needs
to be equipped. Rewards persist through save/load. Exact generation and reward rules are in the
[gameplay contracts](docs/gameplay-contracts.md#relics-and-epic-encounters).


## Optional Objective Rooms

A new dungeon has a 35% chance to receive one optional objective room on floors 3–8, if suitable space exists.
They use existing open terrain and monsters, and are never required to descend or escape. Press **m** to inspect
their rules and progress. Two objectives are possible:

- **The Vigil of the Lost Spirit:** press **y** beside `S` to begin. Over the next three successful turns, keep
  yourself or freed, living Ash within three tiles. Adjacent enemies erode the spirit's three points of resolve;
  protect it before they extinguish its light. Moving away from both protectors or leaving the floor abandons it.
- **The Broken Ritual:** stand on each of three `O` sigils and press **y**. Each distinct sigil costs one turn
  to break; partial progress persists while you explore elsewhere.

Completing either grants one healing potion, warding charm, or frost charm, chosen when the dungeon is generated.
Progress, failure, completion, and the fixed reward survive saving and revisiting. Rewards can be earned only once.
Reading instructions or trying to break a spent sigil takes no turn.


## This Lich's crypt

Every new dungeon has a secret crypt entrance on one of floors **4–7**. Search beside the burial seal with **k**
to reveal `{`. Its warning explains the danger before you enter. Stepping through takes one turn and seals the
entrance permanently. This is an extra hidden floor; the main dungeon still has ten floors.

The crypt fills the map with small burial rooms and winding passages. You arrive at one end. The **Under Lich**
holds the maze's only central crossing: a tough undead guardian with 160 HP and 6 guard, attacking every other turn
with claws or nearby grave-fire and hexes. He cannot be frightened away from his post. His defeat counts toward
your kills and difficulty, but does not break the Dread Lich's curse or revive Ash.

- All hostile monsters are undead. Their combined difficulty, **including the Under Lich**, matches floor eight.
- There are **no fountains**, ordinary stairs, or secret rooms inside. Floor-eight weapons, armor, and magic can be
  found; equipment is placed on the entrance side of the guardian.
- You can see only **three tiles** around you, with walls blocking sight. Map, inspection, codex discovery, and
  targeted consumables honor this limit. Sun Orbs retain their floor-wide effect against undead.
- **Blink Stones and Warding Charms are consumed without effect**, spending one turn. Entry also dispels an active
  ward. Armor and its passive traits retain their ordinary behavior.
- Find the far wall where cold air leaks through, then search with **k** to reveal `}`. Stepping through returns
  you to a different secret door on the entrance floor. There is no return through the sealed entrance.

The maze has **1–4 traps**, chosen without repeating a type. Each search within one tile rolls a **50% chance**
to discover each nearby hidden trap. A discovered trap appears as `^`; stand beside it and press **y** to disarm
it safely, spending one turn. Stepping onto it before disarming still triggers it. Ash does not trigger traps.

| Trap | Effect |
| --- | --- |
| Falling stone block | Heavy physical damage from the ceiling. |
| Collapsing pit | Physical damage; your next attempted move climbs out instead. |
| Dart volley | Physical damage from hidden wall slits. |
| Flame jet | A burst of fire damage; spell wards apply. |
| Frost rune | Spell damage and an ice hold; Oracle Crown prevents the hold. |
| Grave bell | Spell damage and a hex weakening your next strike. |
| Choking gas | Two poison damage on each of three action turns, including the trigger turn; armor cannot block it. |
| Repulsion rune | Spell damage and a push back onto your previous tile if it is clear. |

The Oracle Crown cannot remove a physical pit hold. Already inhaled poison continues after leaving the crypt;
reading, saving, and failed actions do not advance it. Discovered, triggered, and disarmed traps, the guardian,
loot, and both doors persist exactly through save/load. Completing the crypt is optional and does not finish the run.
Existing format-2 saves still load unchanged; start a new game to receive the crypt.


## Book of Ghost Dog and Records

Your inventory includes the **Book of Ghost Dog** from the start. Press **B** or type `book` in line-input mode to read
collected chapters and side stories without spending a turn. The ten numbered chapters tell Ash's main story.
Every collected side story contributes a memory to the escape ending. The dream and unfinished song also respond
to whether Ash escapes with you; unread stories remain undisclosed.

Living Ash has occasional personality moments at his rescue, fountains, keepsake, and return from the mist.
During quiet exploration he may react to the floor or your injuries, at most once per floor and at least fifteen
turns after his last moment. These moments add flavor without changing combat stats or spending extra turns or RNG.
When you escape or die, **Your Adventure** recounts the floors and secrets reached, collected stories, Ash's rescue
and falls, defeated Epics, completed objectives, and final weapon. It uses recorded events and never invents missing
history from an older save. Quitting an unfinished run does not produce a chronicle.

Press **j** or type `challenges` to see the current run and personal records:

- **Fewest turns:** the fastest successful escape, including the final stair action. Successful actions and turns lost
  to entanglement count; reading, inspecting, saving, and failed actions do not.
- **Escape without Ghost Dog falling:** free Ash and escape with him alive without him ever reaching zero health.
- **Most monsters killed:** the most kills in a run that ends in escape or death. Kills by Ghost Dog and consumables
  count; each monster counts once.
- **Most difficulty defeated:** the greatest sum of slain monsters' difficulty points in a finished run. Tougher
  monsters contribute more than weaker ones. Returning stairwell guardians count toward both kill records.

Movement effects name their cause: spider silk webs you, Frostbound Hounds freeze your feet, Chain Imps pin you with
silver chains, and Rift Weavers bind you with rift magic. Each costs one attempted movement turn; other actions and
reading screens keep their existing behavior. The Oracle Crown prevents all four effects and clears them when equipped.

Save format 2 stores each movement effect and the connected chamber layout. Existing format-2 saves remain supported;
new objective, warning, adventure, and crypt fields are optional. Loading preserves the saved dungeon and RNG without
adding areas, objective rooms, or past events. Unsupported formats, including format 1, require a new game.
Personal records are retained.

Records are saved automatically in `~/.ghost-dog-data.json` when you die or escape, without using the save command.
Personal records appear at game startup and game end, and are shared across runs and save files. Saves retain equipment,
book contents, challenge progress, and generated difficulty budgets and monsters without rerolling them.
At game end, a bold gold celebration panel calls out every **NEW RECORD!** and **RECORD TIED!**, with its category and
score. Your first qualifying result sets a record; a tie keeps the original record holder. Fastest escape records
require victory, and the Ghost Dog safe escape record also requires that he never fell and escaped alive with you.

Each save creates a new `ghostdog-dungeon-YYYYMMDD-HHMMSS.json` snapshot, adding a numeric suffix if that name already
exists. `--save-file-path` selects the directory and creates it if missing; the default is the current working
directory. Loaded games also create fresh snapshots there, leaving the original file untouched. **S** saves without
spending a turn; **L** restores the most recently saved or loaded snapshot in the current session. The save directory
and records-file path are runtime settings and are not stored in the dungeon save.

```sh
./bin/ghostdog --seed 42 --save-file-path ./saves
./bin/ghostdog --load-file ./saves/ghostdog-dungeon-20261012-090807.json --save-file-path ./saves
```

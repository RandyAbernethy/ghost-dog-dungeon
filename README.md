# ghost-dog-dungeon
A nethack like way to waste time in 10 levels

Build and play:

```sh
go build -o bin/ghostdog ./src
./bin/ghostdog
```

The repository keeps module metadata, documentation, and the Makefile at the root:

```text
src/      Game source and tests that exercise private game state
tests/    Black-box tests of the compiled executable
bin/      Local build output (ignored by Git)
go.mod    Go module definition
```

Go's package tests live beside the source they exercise. Tests in `tests/` build the game and run it from temporary
directories with isolated saves and home directories. Run all checks from the repository root:

```sh
go test ./...
go vet ./...
git diff --check
```

The optional Makefile provides `make build`, `make run`, `make test`, `make vet`, `make fmt`, and `make check`.
Generated saves and personal records are ignored. For an explicit save directory, create `saves/` and pass a path
with `--save-file saves/run.json`.

Each run varies the dungeon, treasure, fountain locations, and monster groups. Floors can be winding caves, narrow
crypt passages, connected chambers, or pillared halls, with a chapel, barracks, catacomb, or kennel theme. Themes favor
different enemies and supplies within the floor's usual difficulty range. Monster groups can be scattered patrols,
small packs, treasure guards, or a guard with a spellcaster.

Monster counts vary with a difficulty budget, rather than a fixed quota per floor. Each monster's points account for
health, damage, armor, and special powers. Floor budgets vary by about 7% around values calibrated from the previous
encounters (26 points on floor one through 903 on floor ten), and both budgets and actual encounter totals increase
with depth. Groups stay close to the previous health and damage totals, with limits on ranged attackers, disabling
powers, and repeated species. Occasional weaker monsters can replace a stronger opponent. Stairwell guardians use
smaller budgets appropriate to their floor.

Search suspicious walls to discover secret passages. Each game has 5-8 secret chambers, including the escape sanctum
and any nested chambers. Floors can have zero, one, two, or rarely three chambers. Some chambers are independent;
occasionally one hides inside another, with no fixed floor for nesting. Every chamber contains a fountain and its own
side-story scroll. Floor pools include new monster types, while the Dread Lich is always on floor ten. Tunnel Bats,
Cave Crawlers, Rust Guards, Barrow Archers, Grave Acolytes, Ember Hounds, Obsidian Guards, and Hollow Wardens can appear;
none is guaranteed in a run. Used fountains refill
when you leave their floor, so you can return for more healing if you are willing to face the stairwell guardians. Drinking from
a fountain restores 4-8 health to you and 8-12 health to Ghost Dog.

Optional rooms include guarded armories, healing shrines, abandoned kennels, and treasure vaults. Look for friendly
ghosts (**G**), shortcut levers (**+**), Ash's keepsakes (**&**), and shrines (**^**). Stand on or next to one and press
**y** (or type `interact`) to interact. A friendly ghost offers three different, randomly chosen gifts: armor, a weapon,
a potion, a Blink Stone, a Fire Scroll, a Warding Charm, a Sun Orb, a Frost Charm, a Starfire Orb, Phoenix Ash, or a Ghost
Dog Recall Scroll. Equipment matches the floor's usual loot range. Accept one gift with `:choose 1`, `:choose 2`, or
`:choose 3` followed by Enter (omit the colon in line-input mode). Each ghost keeps its choices when you return or load
a save. Older saves receive three choices for any unused ghosts. Reading the offer costs no turn;
accepting a gift or activating another event spends one turn. Event rewards can be used once, and stay used after
revisiting or loading. A keepsake needs a living Ghost Dog nearby. Shrines heal you and a living Ghost Dog once.
Approaching within one tile of an unused event or fountain shows its description and interaction key. The prompt
repeats only after you move farther away and return, including when you arrive by stairs or a Blink Stone.

Use **g** to spend a Blink Stone and teleport yourself and Ghost Dog away from monsters on the current floor.
The stone chooses the safest available spot outside secret rooms. Secret rooms remain excluded after discovery.

Equipment is kept in your pack, press **i** for inventory or **E** for the equipment list. Equip a numbered item with
`:equip NUMBER` followed by Enter in immediate-key mode, or `equip NUMBER` in line-input mode (while viewing inventory).
Changing equipment spends one turn. Use `:drop NUMBER` (or `drop NUMBER` in line-input mode) to put an unequipped item
on the floor. Inventory numbers cover gear and consumables. Dropping a consumable releases one from its stack;
successful drops spend one turn. You can pick dropped items up again by leaving their tile and walking back onto it.
Press **Esc** to return to the map from a command screen or cancel an unfinished `:` command without spending a turn. In
line-input mode, you can also type `map` and Enter.

The event log shows the complete current turn, beginning with your action and then Ghost Dog's and the monsters'
responses. A successful action replaces the previous turn's log, and the heading identifies the turn number. Quiet
movement is recorded too. Reading screens, saving, and failed actions keep the current turn available; the log has no
eight-message limit, so crowded fights retain every attack and spell. The current log survives save/load.

- Reach weapons trade damage for striking from two tiles away. Press the direction toward an enemy along a clear line to
  attack without moving (you can attack over Ghost Dog).
- Armor guards against incoming damage.
- Runespun armor trades guard for extra protection against damaging spells.
- Hunter's armor trades guard for extra weapon damage.

There are 30 additional weapons and 40 additional armor pieces across the floor pools. New weapons vary between
steadier and wider damage ranges while preserving average damage and reach. New armor trades guard for spell wards
or strike bonuses, keeping the pool's average stats unchanged. Each floor selects from its pool, so equipment need
not appear every run.

Nested secret rooms on floors **7, 8, and 9** have a **25% chance** to replace one treasure with a supreme relic.
There is at most one relic per dungeon, and dungeons without an eligible inner chamber cannot contain one. These
pieces are never ordinary loot or friendly-ghost gifts. Discovering the outer room still leaves its inner room hidden.

| God armor | Protection and power |
| --- | --- |
| Aegis of the Last Dawn | Body: 24 guard, 6 spell ward, +2 strike. Reduces living Ghost Dog's damage by 6 within two tiles. |
| Oracle Crown | Head: 18 guard, 12 spell ward, +4 strike. Clears and prevents hexes and entanglement. |
| Starthorn Greaves | Leg: 18 guard, 6 spell ward, +3 strike. Surviving a damaging melee hit returns 12 damage. |

God armor gives immense protection, but can absorb at most 90% of a physical blow. Small wounds and powerful enemies
can still kill its wearer. Carrying a piece does not activate its power; equip it first.

| Artifact weapon | Damage and power |
| --- | --- |
| Dawnbreaker | 30–42; steals up to 8 health per successful strike, based on damage actually dealt. |
| Winter's Last Word | 28–38, reach 2; each hit has a 25% chance to freeze a surviving target for its next turn. |
| Worldsplitter | 38–52; ignores monster armor. |
| Storm Shepherd | 30–40, reach 2; arcs 12 damage into one other monster within two tiles of its target. |

All Artifacts are magical and receive the usual bonus against undead. Powers, rarity, and generated rewards survive
save/load; returning to a room cannot reroll its treasure.

Floors 7–9 can also have one rare **Epic** encounter: an Iron Colossus, Grave Titan, Ashen Wyrm, or Hollow Reaper.
Each eligible floor has a 12% chance, with at most one Epic in a dungeon. Epics are optional additions to the ordinary
difficulty budgets. They spawn away from entrances and stairs, with a route around them. They wake within four tiles,
warn you when they wake, and act once every three turns, letting you retreat. They have enormous health, armor, and
damage: the Colossus crushes warding charms, the Titan heals from damaging melee attacks, the Wyrm's strikes erupt in
adjacent fire, and the Reaper reflects close-range hits. God armor or an Artifact helps, but careful use of reach,
consumables, and positioning still matters. Epics never return as stairwell guardians, and killing one is not needed
to defeat the Dread Lich or escape.

Your inventory includes the **Book of Ghost Dog** from the start. Press **B** or type `book` in line-input mode to read
collected chapters and side stories without spending a turn. The ten numbered chapters tell Ash's main story.

Press **j** or type `challenges` to see the current run and personal records:

- **Fewest turns:** the fastest successful escape, including the final stair action. Successful actions and turns lost
  to entanglement count; reading, inspecting, saving, and failed actions do not.
- **Escape without Ghost Dog falling:** free Ash and escape with him alive without him ever reaching zero health.
- **Most monsters killed:** the most kills in a run that ends in escape or death. Kills by Ghost Dog and consumables
  count; each monster counts once.
- **Most difficulty defeated:** the greatest sum of slain monsters' difficulty points in a finished run. Tougher
  monsters contribute more than weaker ones. Returning stairwell guardians count toward both kill records.

Records are saved automatically in `~/.ghost-dog-data.json` when you die or escape, without using the save command.
Personal records appear at game startup and game end, and are shared across runs and save files. Saves retain equipment,
book contents, challenge progress, and generated difficulty budgets and monsters without rerolling them. Existing
saves retain their dungeon and existing records. Runs begun before difficulty scoring was added cannot set that new
record because their earlier kills have no recorded score; start a new run to compete for it.

```sh
./bin/ghostdog --seed 42 --save-file run.json
./bin/ghostdog --load-file run.json
```

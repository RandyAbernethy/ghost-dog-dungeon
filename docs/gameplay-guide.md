# Gameplay guide (spoilers)

> **Spoiler warning:** This guide reveals hidden areas, enemies, equipment powers, rewards, and ending behavior.
> For setup, controls, and general play, start with the [README](../README.md).

Enter the dungeon, free the ghost dog, reach the tenth floor, and escape through the hidden passage beyond the
Dread Lich.

- [Map symbols](#map-symbols)
- [Dungeon generation](#dungeon-generation)
- [Dread Lich and silver chains](#dread-lich-and-silver-chains)
- [Weapons and armor](#weapons-and-armor)
- [Epic encounters](#epic-encounters)
- [Optional objective rooms](#optional-objective-rooms)
- [This Lich's crypt](#this-lichs-crypt)
- [Book of Ghost Dog and endings](#book-of-ghost-dog-and-endings)
- [Movement effects](#movement-effects)
- [Example game output](#example-game-output)

Exact generation rules, persistence requirements, and regression-test references are in the
[gameplay contracts](gameplay-contracts.md).


## Map Symbols

Symbols are case-sensitive. Each monster species, physical item category, and feature has its own symbol. All scrolls
(including side stories) share **?**, orbs **=**, charms **%**, and potions **!**. Phoenix Ash is a stronger healing
consumable and uses the potion appearance. Equipment variants share their weapon or armor category.
Floor legends, inspection and monster-drop messages reveal only physical form, concealing names, powers, statistics
and story contents until pickup. Inventory and explicit gift offers identify them. Dropped items use generic floor
descriptions again.
**F** and **f** are reserved for fresh and spent fountains. Epics use **1–8**.
The map legend lists objects; inventory and inspection show item symbols, and the codex lists discovered enemies.
Symbols are derived from each monster or item kind and are not stored separately in saves.
Undiscovered rooms remain concealed as walls, and unseen crypt tiles remain blank.


### Terrain and features

| Symbol | Meaning |
| --- | --- |
| `#` | Wall (also conceals undiscovered passages) |
| `.` | Open floor |
| `@` | Player |
| `D` | Freed, living, unchained Ghost Dog |
| `d` | Chained Ghost Dog |
| `>` | Stairs down |
| `<` | Stairs up or out |
| `F` | Healing fountain |
| `f` | Spent fountain |
| `G` | Ghost of a friendly villager |
| `+` | Shortcut lever |
| `&` | Ash's keepsake |
| `^` | Healing shrine |
| `S` | Lost spirit |
| `O` | Ritual sigil |
| `X` | Epic attack warning (red) |
| `{` | Secret crypt entrance |
| `}` | Secret crypt exit or return door |
| `\|` | Permanently sealed crypt entrance |


### Items

| Symbol | Meaning |
| --- | --- |
| `)` | Weapon |
| `[` | Armor |
| `!` | Potion (healing potion or Phoenix Ash; identified on pickup) |
| `?` | Scroll (Fire, Ghost Dog recall, story chapter, or side story; identified on pickup) |
| `*` | Stone (identified on pickup) |
| `%` | Charm (Warding or Frost; identified on pickup) |
| `=` | Orb (Sun or Starfire; identified on pickup) |


### Monsters

| Symbol | Meaning |
| --- | --- |
| `r` | Dungeon Rat |
| `s` | Skeleton |
| `g` | Goblin Knifer |
| `o` | Orc Brute |
| `c` | Cultist |
| `m` | Caustic Slime |
| `w` | Web Spider |
| `W` | Wraith |
| `b` | Blink Stalker |
| `h` | Hex Priest |
| `M` | Mirror Shade |
| `H` | Bone Hound |
| `Y` | Stone Gargoyle |
| `K` | Grave Knight |
| `A` | Cinder Drake |
| `V` | Void Seer |
| `L` | Soul Leech |
| `T` | Storm Herald |
| `B` | Dread Lich |
| `U` | Under Lich |
| `j` | Ash Beetle |
| `i` | Chain Imp |
| `l` | Lantern Wisp |
| `C` | Chapel Sentinel |
| `a` | Gloom Archer |
| `R` | Bell Revenant |
| `I` | Ironroot Brute |
| `y` | Frostbound Hound |
| `z` | Rift Weaver |
| `k` | Crownless Knight |
| `t` | Tunnel Bat |
| `v` | Cave Crawler |
| `n` | Rust Guard |
| `J` | Barrow Archer |
| `P` | Grave Acolyte |
| `E` | Ember Hound |
| `Q` | Obsidian Guard |
| `u` | Hollow Warden |
| `1` | Epic Iron Colossus |
| `2` | Epic Grave Titan |
| `3` | Epic Ashen Wyrm |
| `4` | Epic Hollow Reaper |
| `5` | Epic Frost Monarch |
| `6` | Epic Storm Leviathan |
| `7` | Epic Dusk Tyrant |
| `8` | Epic Obsidian Golem |

Discovered trap symbols and effects are listed in [This Lich's crypt](#this-lichs-crypt).


## Dungeon Generation

Each run varies the dungeon, treasure, fountain locations, and monster groups. Floors can be winding caves, narrow crypt
passages, connected chambers, or pillared halls, with a chapel, barracks, catacomb, or kennel theme. Themes favor
different enemies and supplies within the floor's usual difficulty range. Monster groups can be scattered patrols, small
packs, treasure guards, or a guard with a spellcaster.

Monster counts and groups vary within each floor's difficulty budget, with danger increasing as you descend. Limits
on ranged attackers, disabling powers, and repeated species keep encounters varied within those budgets.

Used fountains refill when you leave their floor, so you can return for more healing if you are willing to face the
stairwell guardians, 2-4 monsters which spawn on a floor when you descend below it. Drinking from a fountain restores
4-8 health to you and 8-12 health to Ghost Dog. Stairwell guardians use smaller budgets appropriate to their floor.

After moving or arriving within three tiles along an open approach to a hidden passage, there is a **20% chance** of
one subtle directional hint about a draft, disturbed mortar, Ash sniffing, or a burial seal in that turn's log.
Hints never prompt you to search; inspection and map redraws do not generate or retry them. Walls block clues, and
clues to a linked chamber become eligible after its parent opens. Searching with **k** remains available without a hint.
Each game has 5-8 secret chambers, including
the escape sanctum and possible linked chambers. Every secret chamber contains a fountain and its own side-story
scroll. After discovering a chamber, search its walls for hidden doors to further chambers.

Floor monster pools include new monster types with each level, while the Dread Lich is always on floor ten. Monsters
such as Tunnel Bats, Cave Crawlers, Rust Guards, Barrow Archers, Grave Acolytes, Ember Hounds, Obsidian Guards, and
Hollow Wardens can appear, though none is guaranteed in a run.

Optional rooms include guarded armories, healing shrines, abandoned kennels, and treasure vaults. Look for friendly
ghosts (**G**), shortcut levers (**+**), Ash's keepsakes (**&**), and shrines (**^**). Stand on or next to one and press
**y** (or type `: interact`) to interact. A ghost of a friendly villager offers three different, randomly chosen gifts: armor, a
weapon, a potion, a Blink Stone, a Fire Scroll, a Warding Charm, a Sun Orb, a Frost Charm, a Starfire Orb, Phoenix Ash,
or a Ghost Dog Recall Scroll. Equipment matches the floor's usual loot range. Accept one gift with `:choose 1`, `:choose
2`, or `:choose 3` followed by Enter, or simply press **1**, **2**, or **3** (Enter in line-input mode). Each ghost keeps its choices when you return
or load a save. Reading the offer costs no turn; accepting a gift or activating another event spends one turn. Event
rewards can be used once, and stay used after revisiting or loading. A keepsake restores up to **8 HP** to a nearby
freed, living Ghost Dog and grants one warding charm. Shrines heal you and a living Ghost Dog once.
Approaching within one tile of an unused event or fountain shows its description and interaction key. The prompt repeats
only after you move farther away and return, including when you arrive by stairs or a Blink Stone.

Use **g** to spend a Blink Stone and teleport yourself and an unchained Ghost Dog away from monsters on the
current floor.
The stone chooses the safest available spot outside secret rooms.


## Dread Lich and Silver Chains

The Dread Lich on floor ten has **78 HP** and a listed **10–14 damage range**.
Alongside his existing grave-fire and curses, each active turn has a **10% chance** to chain a freed, living Ghost Dog
within **five tiles**, with a clear line of sight. Casting the chain takes the Lich's attack for that turn.
Freezing, stunning, or driving him into flight prevents the chain attempt until he can act again.

Chained Ash changes from **D** to **d** and cannot move, attack, dodge, or follow you. **Walk onto his tile to break
the silver chain**, spending one ordinary movement turn. He keeps his current health and can act again from that turn.
Rescuing him takes priority over a reach attack past him, and does not repeat the rewards or history of his first
rescue.

He remains vulnerable to enemies and can receive passive healing or ward protection while on your floor.
Blink Stones leave him in place and cannot land you on his tile; recall scrolls cannot release a living companion.
If you leave by stairs, he stays at the same position on that floor until you return. Living Ash remains chained even
if you kill the Lich. If he falls, his spirit disperses and ordinary revival mechanics apply.

Free Ash before using the escape stairs to leave together. Escaping while he is chained counts as leaving alone,
including the ending and safe-companion challenge. Saves preserve the chain, anchor floor, health and RNG exactly.


## Weapons and Armor

Open **i** to select an item and read its description and comparison. Use arrows and Enter on Unix, or enter the item
number in command-and-Enter mode. Confirm with Enter or **e** to equip, **u** to use a consumable, or **d** to drop.
Press **i** to return to the list or Esc/`map` to close. Browsing and cancelling are free; a successful action spends
one turn. Advanced equip/drop commands remain available. The `details` command (`:details` on Unix) shows the full
pack report, including companion rules and descriptions of items you do not currently carry.

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
wake, and normally act once every three turns, letting you retreat. They alternate normal attacks and signatures.
Normal attacks reach two tiles and use half the signature damage range before defenses. Moving off signature tiles
does not protect you from normal attacks while you stay in range. Adjacent normal attacks retain their melee powers;
distance-two pressure does not. God armor or an Artifact helps, but careful use of reach,
consumables, and positioning still matters. Epics never return as stairwell guardians, and killing one is not needed
to defeat the Dread Lich or escape. The Oracle Crown prevents the Frost Monarch's binding and Dusk Tyrant's hex;
spell wards reduce the Storm Leviathan's lightning damage to you.

Each Epic also has a signature attack: **Hammerfall**, **Grave Cross**, **Cinder Lane**, **Soul Scythe**,
**Ice Bloom**, **Thunder Marks**, **Eclipse**, or **Shard Fan**, respectively. Within two tiles, it can prepare
the attack on its last rest turn before a signature action. **Red X** tiles show where it will strike on its next active turn,
replacing its usual attack or movement. You have one action to move clear; the tiles never follow you.
Your `@` and Ash's `D` turn red when threatened, and the warning names the attack below the map and in **m** inspection.
Ash spends his action dodging rather than attacking, and avoids walking or blinking onto marked tiles.
Freezing or stunning the Epic delays its warning; fleeing or death cancels it. Walls block warnings, and the
Epic does not prepare a pattern if every neighboring escape tile is blocked. Reading screens and saving are free.

Every damaging hit on a surviving Epic builds **rage**, including Ash, consumables, and passive gear damage. Fully
blocked hits do not count. At **4/4**, it can prepare a wide retaliation around you and a nearby Ash, giving **two
actions** to escape. Move immediately and move again; waiting until the final action to begin moving may be fatal.
The warning shows the countdown and fixed tiles. You can also interrupt, use protection, or blink. A legal two-step
escape must exist before preparation; movement holds, occupied squares and reach attacks count as restrictions.
Retaliation pauses the ordinary attack rhythm, and freezing or stunning delays its countdown. Fear cancels it.
Ash starts escaping wide patterns early. Rage, warning tiles, and countdown survive saving and loading exactly.

Defeating any Epic guarantees one elite treasure on its final tile: rare equipment or a powerful consumable. Kills by
Ghost Dog, magic items, and gear powers earn the same reward. Walk onto the tile to collect it; equipment still needs
to be equipped. Rewards persist through save/load. Exact generation and reward rules are in the
[gameplay contracts](gameplay-contracts.md#relics-and-epic-encounters).


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
to discover each nearby hidden trap. Each discovered trap has its own symbol, shown beside its name in inspection.
Stand beside it and press **y** to disarm it safely, spending one turn. Stepping onto it before disarming still triggers
it. Ash does not trigger traps.

| Symbol | Trap | Effect |
| --- | --- | --- |
| `_` | Falling stone block | Heavy physical damage from the ceiling. |
| `0` | Collapsing pit | Physical damage; your next attempted move climbs out instead. |
| `"` | Dart volley | Physical damage from hidden wall slits. |
| `/` | Flame jet | A burst of fire damage; spell wards apply. |
| `\` | Frost rune | Spell damage and an ice hold; Oracle Crown prevents the hold. |
| `` ` `` | Grave bell | Spell damage and a hex weakening your next strike. |
| `~` | Choking gas | Two poison damage on each of three action turns, including the trigger turn; armor cannot block it. |
| `-` | Repulsion rune | Spell damage and a push back onto your previous tile if it is clear. |

The Oracle Crown cannot remove a physical pit hold. Already inhaled poison continues after leaving the crypt;
reading, saving, and failed actions do not advance it. Discovered, triggered, and disarmed traps, the guardian,
loot, and both doors persist exactly through save/load. Completing the crypt is optional and does not finish the run.


## Book of Ghost Dog and Endings

Your inventory includes the **Book of Ghost Dog** from the start. Press **B** or type `book` in line-input mode to read
collected chapters and side stories without spending a turn. The ten numbered chapters tell Ash's main story.
After the victory message and the bowl of fresh rain water, the escape ending prints just one randomly selected
reminiscence from collected side stories or Ash's return after a fall. The dream and unfinished song reflect whether
Ash escapes with you. Uncollected stories remain undisclosed; if no memory is eligible, only the bowl appears.
The choice stays the same after save/load or reprinting the ending. Your Adventure, challenges, personal records,
record celebrations, and the final dungeon seed follow as usual.

Living Ash has occasional personality moments at his rescue, fountains, keepsake, and return from the mist.
During quiet exploration he may react to the floor or your injuries, at most once per floor and at least fifteen
turns after his last moment. These moments add flavor without changing combat stats or spending extra turns or RNG.
When you escape or die, **Your Adventure** recounts the floors and secrets reached, collected stories, Ash's rescue
and falls, defeated Epics, completed objectives, and final weapon. It uses recorded events. Quitting an unfinished
run does not produce a chronicle.


## Movement Effects

Movement effects name their cause: spider silk webs you, Frostbound Hounds freeze your feet, Chain Imps pin you with
silver chains, and Rift Weavers bind you with rift magic. Each costs one attempted movement turn; other actions and
reading screens keep their existing behavior. The Oracle Crown prevents all four effects and clears them when equipped.


## Example Game Output

These examples include the campaign goal and content that can appear in a run.


### Startup

```text
$ ./bin/ghostdog --seed 42

Ghost Dog Dungeon version 1.0.4
Dungeon seed: 42

=== Personal Records ===
Fewest turns record: no escape recorded yet.
Ghost Dog safe escape record: not achieved yet.
Most monsters killed record: no finished run recorded yet.
Most difficulty defeated record: no finished run recorded yet.

=== Ghost Dog Dungeon: Level 1/10 ===
ruined chapel — pillared halls
HP 46/46 | Armor 2 (block 1) | Turns 0
Weapon Rusty Knife (3-7) | Reach 1
Ghost dog: not yet freed
######################################
######################################
######################################
######################################
######################################
######################################
######################################
######################################
######################################
##############..[.....>###############
##############t...@.......############
#############?t.!.....)#%#.###########
##################.#####...###########
########################...###########
######################################
######################################
######################################
######################################
Legend: @ you | # wall | . floor
Nearby: [ armor | t Tunnel Bat | ! potion | ) weapon
| > down
Here: i: inventory | m: inspect | h: help
Dungeon events:
- After coming upon an old chapel, you find an entrance to a dark cellar in the
  chapel yard. Your adventurous spirit drives you to explore.
- The wind howls through the bell tower, like a far away hound longing for
  home.
- You climb down into the cellar below the old chapel and the door slams shut
  behind you. Some great evil is afoot and clearly you must face it to escape
  the catacombs.
Move: wasd/qezx | i: Pack | y: Interact | m: Inspect
h: Help | map: Return | quit: Quit (command + Enter)

Command>
```


### Command line help

```text
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

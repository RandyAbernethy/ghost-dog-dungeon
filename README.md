# ghost-dog-dungeon
A nethack like way to waste time in 10 levels

Build and play:

```sh
go build -o ghostdog .
./ghostdog
```

Each run varies the dungeon, treasure, fountain locations, and monster groups. Floors can be winding caves, narrow
crypt passages, connected chambers, or pillared halls, with a chapel, barracks, catacomb, or kennel theme. Themes favor
different enemies and supplies within the floor's usual difficulty range. Monster groups can be scattered patrols,
small packs, treasure guards, or a guard with a spellcaster.

Search suspicious walls to discover secret passages. Each game has 5-8 secret chambers, including the escape sanctum
and any nested chambers. Floors can have zero, one, two, or rarely three chambers. Some chambers are independent;
occasionally one hides inside another, with no fixed floor for nesting. Every chamber contains a fountain and its own
side-story scroll. Every floor has new monster types, while the Dread Lich is always on floor ten. Used fountains refill
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

- Reach weapons trade damage for striking from two tiles away. Press the direction toward an enemy along a clear line to
  attack without moving (you can attack over Ghost Dog).
- Armor guards against incoming damage.
- Runespun armor trades guard for extra protection against damaging spells.
- Hunter's armor trades guard for extra weapon damage.

Your inventory includes the **Book of Ghost Dog** from the start. Press **B** or type `book` in line-input mode to read
collected chapters and side stories without spending a turn. The ten numbered chapters tell Ash's main story.

Press **j** or type `challenges` to see the current run and personal records:

- **Fewest turns:** the fastest successful escape, including the final stair action. Successful actions and turns lost
  to entanglement count; reading, inspecting, saving, and failed actions do not.
- **Escape without Ghost Dog falling:** free Ash and escape with him alive without him ever reaching zero health.
- **Most monsters killed:** the most kills in a run that ends in escape or death. Kills by Ghost Dog and consumables
  count; each monster counts once.

Records are saved automatically in `~/.ghost-dog-data.json` when you die or escape, without using the save command.
Personal records appear at game startup and game end, and are shared across runs and save files. Saves retain equipment,
book contents, and challenge progress.

```sh
./ghostdog --seed 42 --save-file run.json
./ghostdog --load-file run.json
```

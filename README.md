# ghost-dog-dungeon
A nethack like way to waste time in 10 levels

Build and play:

```sh
go build -o ghostdog .
./ghostdog
```

Each run varies the dungeon, treasure, fountain locations, and monster groups. Search a room's walls to discover secret
passages. Every floor has new monster types, while the Dread Lich is always on floor ten. Used fountains refill when you
leave their floor, so you can return for more healing if you are willing to face the stairwell guardians. Drinking from
a fountain restores 4-8 health to you and 8-12 health to Ghost Dog.

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

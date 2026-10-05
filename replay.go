package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var newMonsterKindsByFloor = [...]monsterKind{
	monsterAshBeetle, monsterChainImp, monsterLanternWisp, monsterChapelSentinel,
	monsterGloomArcher, monsterBellRevenant, monsterIronrootBrute, monsterFrostHound,
	monsterRiftWeaver, monsterCrownlessKnight,
}

func newMonsterSpec(kind monsterKind) (monster, bool) {
	var m monster
	switch kind {
	case monsterAshBeetle:
		m = monster{Glyph: 'b', MaxHP: 9, MinDamage: 1, MaxDamage: 3}
	case monsterChainImp:
		m = monster{Glyph: 'i', MaxHP: 12, MinDamage: 2, MaxDamage: 4}
	case monsterLanternWisp:
		m = monster{Glyph: 'l', MaxHP: 10, MinDamage: 2, MaxDamage: 4, Undead: true}
	case monsterChapelSentinel:
		m = monster{Glyph: 'C', MaxHP: 18, MinDamage: 3, MaxDamage: 6}
	case monsterGloomArcher:
		m = monster{Glyph: 'a', MaxHP: 17, MinDamage: 3, MaxDamage: 5}
	case monsterBellRevenant:
		m = monster{Glyph: 'R', MaxHP: 20, MinDamage: 3, MaxDamage: 6, Undead: true}
	case monsterIronrootBrute:
		m = monster{Glyph: 'I', MaxHP: 26, MinDamage: 4, MaxDamage: 7}
	case monsterFrostHound:
		m = monster{Glyph: 'H', MaxHP: 23, MinDamage: 4, MaxDamage: 7, Undead: true}
	case monsterRiftWeaver:
		m = monster{Glyph: 'R', MaxHP: 24, MinDamage: 4, MaxDamage: 7}
	case monsterCrownlessKnight:
		m = monster{Glyph: 'K', MaxHP: 32, MinDamage: 5, MaxDamage: 8, Undead: true}
	default:
		return monster{}, false
	}
	m.Kind, m.Name, m.HP = kind, string(kind), m.MaxHP
	return m, true
}

// These abilities run after frozen/fleeing, adjacent attacks, and dog targeting.
func (g *game) newMonsterPower(m *monster) bool {
	d := distance(m.Pos, g.player.Pos)
	switch m.Kind {
	case monsterChainImp:
		if d <= 3 && g.rng.Intn(100) < 25 {
			g.player.WebbedTurns = 1
			g.addMessage("A Chain Imp hooks your ankles with a silver chain.")
			return true
		}
	case monsterLanternWisp:
		if d <= 3 && g.rng.Intn(100) < 30 {
			g.hurtPlayerSpell(g.randRange(2, 4), "The Lantern Wisp casts a pale spark")
			return true
		}
	case monsterGloomArcher:
		if d <= 5 && g.rng.Intn(100) < 40 {
			g.hurtPlayer(g.randRange(m.MinDamage, m.MaxDamage), "The Gloom Archer fires a barbed arrow")
			return true
		}
	case monsterBellRevenant:
		if d <= 4 && g.rng.Intn(100) < 30 {
			g.player.HexedTurns = 1
			g.hurtPlayerSpell(2, "The Bell Revenant tolls a broken bell")
			return true
		}
	case monsterRiftWeaver:
		if d <= 4 && g.rng.Intn(100) < 30 {
			spots := g.openNeighbors(g.player.Pos, false)
			if len(spots) > 0 {
				m.Pos = spots[g.rng.Intn(len(spots))]
				g.player.WebbedTurns = 1
				g.addMessage("A Rift Weaver steps through a tear in the air and binds your feet.")
				return true
			}
		}
	}
	return false
}

func monsterGuard(kind monsterKind) int {
	switch kind {
	case monsterAshBeetle:
		return 1
	case monsterChapelSentinel:
		return 2
	case monsterIronrootBrute:
		return 3
	default:
		return 0
	}
}

func newMonsterDescription(kind monsterKind) string {
	switch kind {
	case monsterAshBeetle:
		return "Its ash-coated shell blocks 1 damage per hit."
	case monsterChainImp:
		return "Can chain your feet from three tiles away, costing a movement turn."
	case monsterLanternWisp:
		return "Undead; casts sparks from three tiles away. Spell wards reduce their damage."
	case monsterChapelSentinel:
		return "Its chapel shield blocks 2 damage per hit; heavy weapons overcome it."
	case monsterGloomArcher:
		return "Fires physical arrows from five tiles away; guard protects against them."
	case monsterBellRevenant:
		return "Undead; its distant bell deals spell damage and weakens your next strike."
	case monsterIronrootBrute:
		return "Blocks 3 damage per hit but rests every other turn; use reach and timing."
	case monsterFrostHound:
		return "Undead; its freezing bite can cost you a movement turn."
	case monsterRiftWeaver:
		return "Can teleport beside you and bind your feet; save a blink stone."
	case monsterCrownlessKnight:
		return "Undead; its melee strike shatters an active warding charm."
	default:
		return ""
	}
}

var reachWeaponNames = [...]string{
	"Ashwood Spear", "Chapel Lance", "Bone-Tipped Pike", "Silver Glaive", "Ember Lance",
	"Gravewood Pike", "Frostglass Spear", "Stormreach Glaive", "Voidsteel Lance", "Dawnwatch Pike",
}

func (w weapon) effectiveReach() int { return max(1, w.Reach) }

func weaponUndeadBonus(w weapon) int {
	if w.Magic {
		return 2
	}
	return 0
}

func (p *player) spellWard() int {
	return p.HeadArmor.SpellWard + p.BodyArmor.SpellWard + p.LegArmor.SpellWard
}

func (p *player) strikeBonus() int {
	return p.HeadArmor.StrikeBonus + p.BodyArmor.StrikeBonus + p.LegArmor.StrikeBonus
}

func (g *game) hurtPlayerSpell(damage int, source string) {
	g.hurtPlayer(max(0, damage-g.player.spellWard()), source)
}

func (p *player) carryWeapon(w weapon) {
	for _, carried := range p.Weapons {
		if carried == w {
			return
		}
	}
	p.Weapons = append(p.Weapons, w)
}

func (p *player) carryArmor(a armor) {
	for _, carried := range p.Armors {
		if carried == a {
			return
		}
	}
	p.Armors = append(p.Armors, a)
}

func (p *player) ensureCarriedEquipment() {
	p.carryWeapon(p.Weapon)
	for _, a := range []armor{p.HeadArmor, p.BodyArmor, p.LegArmor} {
		p.carryArmor(a)
	}
}

func (g *game) equipCommand(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) != 2 {
		g.addMessage("Use equip NUMBER from the inventory; in immediate-key mode, type :equip NUMBER and Enter.")
		return false
	}
	n, err := strconv.Atoi(fields[1])
	g.player.ensureCarriedEquipment()
	if err != nil || n < 1 || n > len(g.player.Weapons)+len(g.player.Armors) {
		g.addMessage("Choose an equipment number shown in your inventory.")
		return false
	}
	if n <= len(g.player.Weapons) {
		w := g.player.Weapons[n-1]
		if g.player.Weapon == w {
			g.addMessage("You already have that weapon equipped.")
			return false
		}
		g.player.Weapon = w
		g.addMessage(fmt.Sprintf("You equip %s: %d-%d damage, reach %d. Swapping equipment takes a turn.", w.Name, w.Min, w.Max, w.effectiveReach()))
	} else {
		a := g.player.Armors[n-len(g.player.Weapons)-1]
		if g.player.armorForSlot(a.Slot) == a {
			g.addMessage("You already have that armor equipped.")
			return false
		}
		g.player.setArmor(a)
		g.addMessage(fmt.Sprintf("You equip %s. %s Swapping equipment takes a turn.", a.Name, armorTraits(a)))
	}
	return true
}

func (g *game) dropCommand(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) != 2 {
		g.addMessage("Use :drop NUMBER + Enter to drop an item numbered in your inventory.")
		return false
	}
	n, err := strconv.Atoi(fields[1])
	g.player.ensureCarriedEquipment()
	lines := g.inventoryLines()
	gearCount := len(g.player.Weapons) + len(g.player.Armors)
	if err != nil || n < 1 || n > gearCount+len(lines) {
		g.addMessage("Choose an item number shown in your inventory.")
		return false
	}
	var dropped item
	switch {
	case n <= len(g.player.Weapons):
		i := n - 1
		w := g.player.Weapons[i]
		if w == g.player.Weapon {
			g.addMessage("You cannot drop an equipped item. Equip something else first.")
			return false
		}
		dropped = makeWeaponItem(w)
		g.player.Weapons = append(g.player.Weapons[:i], g.player.Weapons[i+1:]...)
	case n <= gearCount:
		i := n - len(g.player.Weapons) - 1
		a := g.player.Armors[i]
		if a == g.player.armorForSlot(a.Slot) {
			g.addMessage("You cannot drop an equipped item. Equip something else first.")
			return false
		}
		dropped = makeArmorItem(a)
		g.player.Armors = append(g.player.Armors[:i], g.player.Armors[i+1:]...)
	default:
		line := lines[n-gearCount-1]
		if *line.quantity <= 0 {
			g.addMessage("You have none of that item to drop.")
			return false
		}
		*line.quantity--
		dropped = item{Kind: line.kind, Name: line.name, Glyph: line.glyph}
	}
	dropped.Pos = g.player.Pos
	g.current().Items = append(g.current().Items, dropped)
	g.addMessage(fmt.Sprintf("You drop %s on the floor. Dropping takes a turn.", dropped.Name))
	return true
}

func (g *game) renderEquipment(w io.Writer) {
	g.player.ensureCarriedEquipment()
	fmt.Fprintln(w, "\nCarried equipment (equip NUMBER or drop NUMBER; in immediate-key mode prefix with : and press Enter):")
	for i, carried := range g.player.Weapons {
		active := ""
		if carried == g.player.Weapon {
			active = " [equipped]"
		}
		magic := ""
		if carried.Magic {
			magic = ", +2 damage against undead"
		}
		fmt.Fprintf(w, "%d. %s: %d-%d damage, reach %d%s%s\n", i+1, carried.Name, carried.Min, carried.Max, carried.effectiveReach(), magic, active)
	}
	for i, carried := range g.player.Armors {
		active := ""
		if carried == g.player.armorForSlot(carried.Slot) {
			active = " [equipped]"
		}
		fmt.Fprintf(w, "%d. %s (%s): %s%s\n", len(g.player.Weapons)+i+1, coloredArmorName(carried), slotLabel(carried.Slot), armorTraits(carried), active)
	}
	fmt.Fprintln(w, "Reach 2: press a movement direction to strike a foe two tiles away along a clear line. Equipping takes one turn.")
}

func armorTraits(a armor) string {
	return fmt.Sprintf("%d guard, %d spell ward, %+d strike damage.", a.Defense, a.SpellWard, a.StrikeBonus)
}

type sideStory struct {
	Title string
	Text  string
}

var ghostDogSideStories = [...]sideStory{
	{"The Stolen Glove", "Mara could never find her left glove on cold mornings. Ash carried it to the chapel steps and sat on it until she came outside. She called him a thief, then stayed to watch the sunrise with him."},
	{"The Chainmaker", "The smith who forged Ash's silver chain filed every link smooth. He thought a kind chain could make captivity gentle. After Mara left, he kept the rough filings in a jar and could not bring himself to throw them away."},
	{"A Light in the Rain", "When the chapel roof leaked, Ash followed each falling drop with his nose. Mara moved a lantern from puddle to puddle while the children laughed. For one stormy evening, the chapel had a hundred little moons."},
	{"The Watchman's Bread", "An old watchman always broke his supper in two. Ash waited for the smaller half without begging. Years later, the empty chair beside the chapel gate still had crumbs tucked into its cracks."},
	{"The Arrow and the Bell", "A village archer once aimed at a crow on the chapel bell. Ash barked and spoiled the shot. The archer grumbled until he saw the nest above it; the next day he brought Ash a strip of smoked meat."},
	{"The Bell That Would Not Ring", "Mara lost her voice during a winter fever. Ash slept beneath the silent bell rope for three days. When she finally rang it again, he howled so loudly that nobody noticed how weak the bell sounded."},
	{"Roots Beneath the Chapel", "Ash dug at the great tree until he uncovered a child's wooden soldier. Mara returned it to a gray-haired farmer who had lost it fifty years before. He cried into Ash's fur and called it dust in his eyes."},
	{"Warmth for a Stranger", "A stray hound appeared at the chapel gate during the first snow. Ash pushed his own blanket toward it and lay across the doorway. By morning there were two dogs under the blanket, and Mara had to step over both."},
	{"The Door in a Dream", "Long after Ash was bound below, Mara dreamed of a door opening onto the chapel yard. A pale dog waited on the other side. She never saw who opened it, but she left a clean bowl beside the waking door."},
	{"The Unfinished Song", "Mara wrote a small song for Ash, but she never found its final line. In the margin she left these words: When the bell rings and the road is open, let him choose where home will be."},
	{"The Room Behind the Room", "Mara hid her keepsakes in a little room behind the chapel pantry. Ash found the loose stone and nudged it open with his nose. Inside lay a ribbon, a wooden bell, and a drawing of a small dog. She tied the ribbon to his collar and left the door open whenever they were alone."},
}

func hasStoryID(ids []int, id int) bool {
	for _, known := range ids {
		if known == id {
			return true
		}
	}
	return false
}

func (g *game) addSecretSideStory(lvl *level) {
	for _, hidden := range lvl.Secret.rooms() {
		g.addRoomSideStory(lvl, hidden)
	}
}

func (g *game) addRoomSideStory(lvl *level, hidden *secretRoom) {
	id := hidden.SideStoryID
	if id == 0 {
		id = lvl.Index + 1
	}
	if id < 1 || id > len(ghostDogSideStories) {
		return
	}
	if hasStoryID(g.player.SideStories, id) {
		return
	}
	for _, it := range lvl.Items {
		if it.Kind == itemSideStoryScroll && it.SideStoryID == id {
			return
		}
	}
	for _, p := range hidden.floorTiles() {
		if g.itemAt(lvl, p) == nil && g.fountainAt(lvl, p) == nil && (!lvl.HasEscapeStair || p != lvl.EscapeStairs) {
			lvl.Items = append(lvl.Items, item{Pos: p, Kind: itemSideStoryScroll, Name: "Side-story scroll: " + ghostDogSideStories[id-1].Title, Glyph: '?', SideStoryID: id})
			return
		}
	}
}

func (g *game) renderBook(w io.Writer) {
	fmt.Fprintln(w, "\n=== Book of Ghost Dog ===")
	fmt.Fprintln(w, "The book keeps every story scroll you have found. Reading takes no turn.")
	chapters := append([]int(nil), g.player.StoryChapters...)
	sort.Ints(chapters)
	fmt.Fprintln(w, "\nAsh's story:")
	seen := map[int]bool{}
	for _, id := range chapters {
		if id >= 1 && id <= len(ghostDogStory) && !seen[id] {
			fmt.Fprintf(w, "Chapter %d/10: %s\n", id, ghostDogStory[id-1])
			seen[id] = true
		}
	}
	if len(seen) == 0 {
		fmt.Fprintln(w, "No chapters found yet.")
	}
	fmt.Fprintln(w, "\nSide stories:")
	seen = map[int]bool{}
	for _, id := range g.player.SideStories {
		if id >= 1 && id <= len(ghostDogSideStories) && !seen[id] {
			story := ghostDogSideStories[id-1]
			fmt.Fprintf(w, "%s\n%s\n\n", story.Title, story.Text)
			seen[id] = true
		}
	}
	if len(seen) == 0 {
		fmt.Fprintln(w, "No side stories found yet.")
	}
}

type runStats struct {
	Turns          int  `json:"turns"`
	MonstersKilled int  `json:"monsters_killed"`
	GhostDogFalls  int  `json:"ghost_dog_falls"`
	HistoryKnown   bool `json:"history_known"`
}

type challengeResult struct {
	Seed    int64 `json:"seed"`
	Turns   int   `json:"turns"`
	Kills   int   `json:"kills"`
	Escaped bool  `json:"escaped"`
}

type challengeRecords struct {
	FewestTurns *challengeResult `json:"fewest_turns,omitempty"`
	SafeEscape  *challengeResult `json:"ghost_dog_never_fell,omitempty"`
	MostKills   *challengeResult `json:"most_monsters_killed,omitempty"`
}

func defaultRecordsFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory for challenge records: %w", err)
	}
	return filepath.Join(home, ".ghost-dog-data.json"), nil
}

func (g *game) challengeRecordsPath() (string, error) {
	if g.recordsFile != "" {
		return g.recordsFile, nil
	}
	return defaultRecordsFile()
}

func readChallengeRecords(path string) (challengeRecords, error) {
	var records challengeRecords
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return records, nil
	}
	if err != nil {
		return records, err
	}
	err = json.Unmarshal(data, &records)
	return records, err
}

func (g *game) recordChallenges() error {
	if !g.stats.HistoryKnown || (!g.won && g.player.HP > 0) {
		return nil
	}
	path, err := g.challengeRecordsPath()
	if err != nil {
		return err
	}
	records, err := readChallengeRecords(path)
	if err != nil {
		return err
	}
	result := challengeResult{Seed: g.seed, Turns: g.stats.Turns, Kills: g.stats.MonstersKilled, Escaped: g.won}
	changed := false
	if records.MostKills == nil || result.Kills > records.MostKills.Kills {
		records.MostKills = &result
		changed = true
	}
	if g.won && (records.FewestTurns == nil || result.Turns < records.FewestTurns.Turns) {
		records.FewestTurns = &result
		changed = true
	}
	if g.won && g.dog.Freed && g.dog.Alive && g.stats.GhostDogFalls == 0 && (records.SafeEscape == nil || result.Turns < records.SafeEscape.Turns) {
		records.SafeEscape = &result
		changed = true
	}
	if !changed {
		return nil
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".ghost-dog-data-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

func (g *game) renderChallenges(w io.Writer) {
	fmt.Fprintln(w, "\n=== Optional Challenges ===")
	fmt.Fprintf(w, "This run: %d turns, %d monsters killed, Ghost Dog fell %d time(s).\n", g.stats.Turns, g.stats.MonstersKilled, g.stats.GhostDogFalls)
	fmt.Fprintln(w, "Fewest turns: escape as quickly as you can. Most kills: counts kills by you and Ghost Dog in a run ending in escape or death.")
	if !g.stats.HistoryKnown {
		fmt.Fprintln(w, "This older save has no complete challenge history. Start a new game to set records.")
	} else if g.stats.GhostDogFalls > 0 {
		fmt.Fprintln(w, "Escape without Ghost Dog falling: failed this run, even if he is revived.")
	} else if g.won && g.dog.Freed && g.dog.Alive {
		fmt.Fprintln(w, "Escape without Ghost Dog falling: achieved this run.")
	} else if g.won || g.player.HP <= 0 {
		fmt.Fprintln(w, "Escape without Ghost Dog falling: this run ended without a safe escape together.")
	} else {
		fmt.Fprintln(w, "Escape without Ghost Dog falling: still possible. Free him and bring him out alive.")
	}
	g.renderChallengeRecords(w)
}

func (g *game) renderChallengeRecords(w io.Writer) {
	fmt.Fprintln(w, "\n=== Personal Records ===")
	path, err := g.challengeRecordsPath()
	if err != nil {
		fmt.Fprintf(w, "Could not read challenge records: %v\n", err)
		return
	}
	records, err := readChallengeRecords(path)
	if err != nil {
		fmt.Fprintf(w, "Could not read challenge records: %v\n", err)
		return
	}
	if records.FewestTurns == nil {
		fmt.Fprintln(w, "Fewest turns record: no escape recorded yet.")
	} else {
		fmt.Fprintf(w, "Fewest turns record: %d turns (seed %d).\n", records.FewestTurns.Turns, records.FewestTurns.Seed)
	}
	if records.SafeEscape == nil {
		fmt.Fprintln(w, "Ghost Dog safe escape record: not achieved yet.")
	} else {
		fmt.Fprintf(w, "Ghost Dog safe escape record: achieved in %d turns (seed %d).\n", records.SafeEscape.Turns, records.SafeEscape.Seed)
	}
	if records.MostKills == nil {
		fmt.Fprintln(w, "Most monsters killed record: no finished run recorded yet.")
	} else {
		fmt.Fprintf(w, "Most monsters killed record: %d (seed %d).\n", records.MostKills.Kills, records.MostKills.Seed)
	}
}

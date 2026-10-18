package main

func (g *game) underLichTurn(m *monster) {
	if !m.Awakened && g.canSee(m.Pos) {
		m.Awakened = true
		g.addMessage("The Under Lich rises from a throne of damp bones. 'No living soul passes my master's crypt.' It braces itself across the only passage.")
	}
	// Its position is the maze's mandatory crossing, including under fear effects.
	m.ActionTurns++
	if m.ActionTurns%2 == 0 {
		return
	}
	if g.dog.Freed && g.dog.Alive && distance(m.Pos, g.dog.Pos) == 1 && distance(m.Pos, g.player.Pos) > 1 {
		g.attackDog(m)
		return
	}
	if distance(m.Pos, g.player.Pos) == 1 {
		g.attackPlayer(m)
		return
	}
	if distance(m.Pos, g.player.Pos) <= 3 && g.canSee(m.Pos) {
		g.hexPlayer()
		before := g.player.HP
		g.hurtPlayerSpell(g.randRange(6, 9), "The Under Lich spits foul grave-fire")
		g.godArmorReprisal(m, before)
	}
}

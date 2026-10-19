package main

const (
	lichChainChance = 10
	lichChainRange  = 5
)

func (g *game) tryLichChain(m *monster) bool {
	if m.Kind != monsterBoss || m.HP <= 0 || g.inCrypt || !g.dog.active() ||
		distance(m.Pos, g.dog.Pos) > lichChainRange || !g.clearEpicLine(m.Pos, g.dog.Pos) {
		return false
	}
	if g.rng.Intn(100) >= lichChainChance {
		return false
	}
	g.dog.Chained, g.dog.ChainFloor = true, g.currentLevel
	g.dogFocus = nil
	g.addMessage("The Dread Lich binds Ghost Dog with a silver chain! He cannot act. Move onto his d tile to break the chain.")
	return true
}

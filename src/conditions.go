package main

type bindingKind string

const (
	bindingWebs   bindingKind = "webs"
	bindingIce    bindingKind = "ice"
	bindingChains bindingKind = "chains"
	bindingRift   bindingKind = "rift"
)

func (kind bindingKind) label() string {
	switch kind {
	case bindingWebs:
		return "webbed"
	case bindingIce:
		return "frozen"
	case bindingChains:
		return "chained"
	case bindingRift:
		return "rift bound"
	default:
		return "bound"
	}
}

func (kind bindingKind) blockedMessage() string {
	switch kind {
	case bindingWebs:
		return "Sticky webs hold you in place for a turn."
	case bindingIce:
		return "Ice locks your feet in place for a turn."
	case bindingChains:
		return "Silver chains pin your ankles in place for a turn."
	case bindingRift:
		return "Rift magic binds your feet in place for a turn."
	default:
		return "A binding holds you in place for a turn."
	}
}

func (g *game) bindPlayer(kind bindingKind) {
	if g.player.hasArmorPower(powerClearMind) {
		switch kind {
		case bindingWebs:
			g.addMessage("Oracle Crown dissolves the webs.")
		case bindingIce:
			g.addMessage("Oracle Crown thaws the ice before it can hold you.")
		case bindingChains:
			g.addMessage("Oracle Crown breaks the silver chains.")
		case bindingRift:
			g.addMessage("Oracle Crown dispels the rift magic.")
		}
		return
	}
	g.player.BoundTurns, g.player.Binding = 1, kind
}

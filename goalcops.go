package main

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// GOALS / POLICÍAS. The police types (patrol, FBI, army, SWAT, Power Rangers) are cards of OBJETIVOS (key "cop:<unit>",
// unit = unitPolice..unitRangers). A card is won by KILLING an officer of that type (stabCop / trapCop -> copDefeated),
// or by reaching the milestone that brings the type to the parties (the kills trophies t_cop1..t_cop4, "police:N" in
// Progress.Owned), or because the save bought a uniform of that force in the old shop. Nothing is rewritten.
// What the type does when it comes (which stars call it) is unchanged: unitForStars (killer.go) still reads police:N.
// The uniforms of every force are won by escaping the police (copsets.go) and worn from VESTUARIO > POLICÍAS.
// ---------------------------------------------------------------------------

func copKey(unit int) string { return fmt.Sprintf("cop:%d", unit) }

// copStarsES: the line under the name of a card.
var copStarsES = []string{"1 estrella", "2 estrellas", "3 estrellas", "4 estrellas", "5 estrellas"}

// copCards: the POLICÍAS cards, one per force.
func copCards() []albumCard {
	var out []albumCard
	for i, f := range copForces {
		out = append(out, albumCard{key: copKey(f.unit), nameES: f.es, roleES: copStarsES[i], slug: f.slot, cop: true, roleKind: f.unit})
	}
	return out
}

// copGot: the type is unlocked in this save (see the header).
func (g *Game) copGot(unit int) bool {
	if g.progress.Album[copKey(unit)]&albumKilled != 0 {
		return true
	}
	if unit > unitPolice && ownedID(fmt.Sprintf("police:%d", unit)) {
		return true
	}
	if unit < 0 || unit >= len(copForces) {
		return false
	}
	pre := copForces[unit].slot + ":"
	for _, id := range g.progress.Owned { // a uniform of this force bought before the redesign (or a trophy uniform)
		if strings.HasPrefix(id, pre) {
			return true
		}
	}
	return false
}

// copDefeated: an officer died by the player's hand or a trap (not the medics).
func (g *Game) copDefeated(c *Cop) {
	if c == nil || c.medic || c.kind < 0 || c.kind >= len(copForces) {
		return
	}
	g.albumMark(copKey(c.kind), albumKilled)
}

// copSpecOf: the look of a police type on its card (the base uniform, not the one worn from the wardrobe).
func copSpecOf(unit int) CharSpec {
	u := unitSpecs()
	switch unit {
	case unitFBI:
		return u.fbi[0]
	case unitMilitary:
		return u.military[0]
	case unitSWAT:
		return u.swat
	case unitRangers:
		return u.rangers[0]
	}
	return copSpec()
}

// copLockedHint: how to win a locked police card.
func copLockedHint(unit int) string {
	if unit <= unitPolice {
		return "Mata a un policía local"
	}
	return fmt.Sprintf("Mata a un agente de este tipo (llega con %d asesinatos)", policeNeed(fmt.Sprintf("police:%d", unit)))
}

// copEscapeLine: the escapes counter shown on the POLICÍAS page.
func (g *Game) copEscapeLine() string {
	if need, ok := copSetNext(g.progress.Escapes); ok {
		return fmt.Sprintf("Escapes: %d (próximo uniforme: %d)", g.progress.Escapes, need)
	}
	return fmt.Sprintf("Escapes: %d (todos los uniformes)", g.progress.Escapes)
}

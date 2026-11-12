package main

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// GOALS / POLICÍAS. The police types (patrol, FBI, army, SWAT, Power Rangers) are cards of OBJETIVOS (key "cop:<unit>",
// unit = unitPolice..unitRangers). The types come STRICTLY in order by career kills (copTypeOpen: local always, FBI
// 500, army 1000, SWAT 1500, rangers 2000); a type that has not come shows "???" and can win nothing. An open type is
// CONSEGUIDO by reaching its milestone (the kills trophies t_cop1..t_cop4, "police:N" in Progress.Owned) and
// ELIMINADO once the player KILLS one of its officers (stabCop / trapCop -> copDefeated). The local police is won by
// killing one (or by a local uniform the save bought in the old shop). A uniform of another force (trophy rewards
// "army:tr_gala", "swat:tr_plata", the escapes sets) never opens its type (bug of 2026-11-11: it did).
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

// copTrophyID: the kills trophy that brings a police type ("" for the local police).
func copTrophyID(unit int) string {
	id := fmt.Sprintf("police:%d", unit)
	for _, t := range trophyList {
		if t.reward.id == id && t.stat == "kills" {
			return t.id
		}
	}
	return ""
}

// copTypeOpenIn: the police type has come to the parties in this save, STRICTLY in order (2026-11-11, owner): the
// local police always; FBI at 500 career kills, the army at 1000, SWAT at 1500, the Pawer Rangers at 2000 (trophies
// t_cop1..t_cop4, policeNeed). A type is open when its milestone was reached (the kills, its trophy or its reward
// "police:N" in owned, which is what brings it to a party: unitForStars / policeGroups) AND every type before it is
// open too. Nothing else opens a type: not a uniform of that force (trophy rewards like "army:tr_gala" or
// "swat:tr_plata", the escapes sets), not a card mark. owned: the purchases (Progress.Owned or shopOwned).
func copTypeOpenIn(p *Progress, unit int, owned func(string) bool) bool {
	if unit <= unitPolice {
		return true
	}
	if unit >= len(copForces) {
		return false
	}
	for u := unitFBI; u <= unit; u++ {
		id := fmt.Sprintf("police:%d", u)
		need := policeNeed(id)
		ok := owned(id) || (need > 0 && careerKills(p) >= need)
		if tid := copTrophyID(u); tid != "" && p.Trophies[tid] {
			ok = true
		}
		if !ok {
			return false
		}
	}
	return true
}

// policeTierOpen: the police type answers calls in a party: its reward "police:N" and the one of every type before it
// are owned (shopOwned), so the order FBI -> army -> SWAT -> rangers can never be skipped (unitForStars, policeGroups).
func policeTierOpen(unit int) bool {
	for u := unitFBI; u <= unit; u++ {
		if !ownedID(fmt.Sprintf("police:%d", u)) {
			return false
		}
	}
	return true
}

// copTypeOpen: copTypeOpenIn for the loaded save (the shop already synced).
func (g *Game) copTypeOpen(unit int) bool {
	return copTypeOpenIn(&g.progress, unit, func(id string) bool { return ownedID(id) || g.hasOwned(id) })
}

// copGot: the card is won (see the header). A type that has not come yet (copTypeOpen) is never won, whatever the
// save holds: its card shows "???".
func (g *Game) copGot(unit int) bool {
	if unit < 0 || unit >= len(copForces) || !g.copTypeOpen(unit) {
		return false
	}
	if g.progress.Album[copKey(unit)]&albumKilled != 0 {
		return true
	}
	if unit > unitPolice { // reaching the milestone wins the card (CONSEGUIDO); killing one makes it ELIMINADO
		return true
	}
	pre := copForces[unit].slot + ":"
	for _, id := range g.progress.Owned { // a local uniform bought before the redesign (or a trophy uniform)
		if strings.HasPrefix(id, pre) {
			return true
		}
	}
	return false
}

// copDefeated: an officer died by the player's hand or a trap (not the medics). A type that has not come yet (only
// a debug switch can bring it: -trexkill, -unlocked) marks nothing.
func (g *Game) copDefeated(c *Cop) {
	if c == nil || c.medic || c.kind < 0 || c.kind >= len(copForces) || !g.copTypeOpen(c.kind) {
		return
	}
	g.albumMark(copKey(c.kind), albumKilled)
}

// reconcileCopCards (progress_migrate.go, CopRev 1): a save whose POLICÍAS cards were marked for a type that its
// career kills have not brought yet loses those marks. Only the "cop:N" album keys of closed types are touched: kills,
// trophies, medals, money, uniforms and every other card stay as they are. Idempotent.
func reconcileCopCards(p *Progress) {
	if p.Album == nil {
		return
	}
	own := map[string]bool{}
	for _, id := range p.Owned {
		own[id] = true
	}
	for _, f := range copForces {
		if !copTypeOpenIn(p, f.unit, func(id string) bool { return own[id] }) {
			delete(p.Album, copKey(f.unit))
		}
	}
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

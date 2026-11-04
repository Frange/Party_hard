package main

import "math"

// ---------------------------------------------------------------------------
// Who do the police go after? The LAST person somebody (a guest, a guard or an
// officer) actually saw killing. The handcuffs mark follows that person: if the
// special killer or an armed guest is seen killing, the player is cleared (the
// cuffs come off); if the player kills again and is seen, they go back on.
// Kills nobody sees change nothing. Everything that decides it lives here.
// ---------------------------------------------------------------------------

type suspectKind int

const (
	suspNone suspectKind = iota
	suspPlayer
	suspSlasher
	suspGuest
)

// lastKill is the last killer that was seen.
type lastKill struct {
	kind  suspectKind
	guest *Guest // for suspGuest
	t     float64
}

// playerCleared: somebody else was seen killing after the player did.
func (g *Game) playerCleared() bool {
	k := g.lastKiller.kind
	return k == suspSlasher || k == suspGuest
}

// playerIsSuspect: the police are after the player (the cuffs mark shows).
func (g *Game) playerIsSuspect() bool { return g.isSuspectP(g.p) }

// markPlayerWanted raises the player's search level unless somebody else is the suspect now.
func (g *Game) markPlayerWanted(v float64) {
	if !g.playerCleared() {
		g.wanted = math.Max(g.wanted, v)
		g.ensureSuspect() // two players: if nobody saw the kill, it is the last one who killed
	}
}

// suspectSeen is the single decision point: kind killed and somebody saw it.
func (g *Game) suspectSeen(kind suspectKind, gu *Guest) {
	prev := g.lastKiller
	g.lastKiller = lastKill{kind: kind, guest: gu, t: g.time}
	switch kind {
	case suspPlayer:
		g.suspect = nil // not the innocent guest, not the killer: this one
		if g.is2P() {
			// two players: the cuffs follow the one who was seen. If somebody else was the suspect
			// (the killer, an armed guest) both players were clear: only he is a suspect again. If the
			// other player was a suspect already, now both are.
			if prev.kind == suspSlasher || prev.kind == suspGuest {
				g.clearSuspects()
			}
			g.markSuspect(g.p)
		}
		if prev.kind == suspSlasher || prev.kind == suspGuest {
			// the cuffs come back at once if the authorities are around
			if g.policeCalled && g.lawActive() {
				g.wanted = math.Max(g.wanted, 45)
			}
			for _, c := range g.cops {
				if !c.medic && (c.state == cSlash || c.state == cArrest) {
					g.releaseCop(c)
					g.copNext(c)
				}
			}
		}
	case suspSlasher:
		g.suspect = nil
		g.clearPlayer()
		g.copsGoFor(kind, nil)
	case suspGuest:
		g.clearPlayer()
		if gu != nil && !gu.out() {
			g.suspect = gu
		}
		g.copsGoFor(kind, gu)
	}
}

// clearPlayer takes the cuffs off the player and sends the officers who were chasing him back to work.
func (g *Game) clearPlayer() {
	g.wanted, g.seenNow = 0, false
	g.clearSuspects() // two players: both lose the cuffs
	for _, c := range g.cops {
		if c.medic || (c.state != cChase && c.state != cTired) {
			continue
		}
		g.releaseCop(c)
		c.chaseT, c.lost = 0, 0
		g.copNext(c)
	}
}

// copsGoFor points the officers already on the scene at the new suspect.
func (g *Game) copsGoFor(kind suspectKind, gu *Guest) {
	for _, c := range g.cops {
		if c.medic || c.state == cLeave || c.state == cArrest && kind == suspGuest {
			continue
		}
		switch kind {
		case suspSlasher:
			s := g.slasher
			if s == nil || s.state != slHunt {
				return
			}
			g.releaseCop(c)
			c.state, c.path, c.repath = cSlash, g.lvl.Path(c.pos, s.pos), 0
		case suspGuest:
			if gu == nil || gu.out() {
				return
			}
			g.releaseCop(c)
			c.state, c.repath = cArrest, 0
		}
	}
}

// updateSuspect keeps the model consistent every frame: a suspect that is gone is forgotten and the
// player can never be both cleared and wanted.
func (g *Game) updateSuspect() {
	switch g.lastKiller.kind {
	case suspGuest:
		if gu := g.lastKiller.guest; gu == nil || gu.out() {
			g.lastKiller = lastKill{}
		}
	case suspSlasher:
		alive := false
		for _, s := range g.allSlashers() { // up to two killers with two players
			if s != nil && s.state != slGone {
				alive = true
			}
		}
		if !alive {
			g.lastKiller = lastKill{}
		}
	}
	if g.playerCleared() && g.wanted > 0 {
		g.wanted = 0
	}
	if g.playerCleared() && g.is2P() {
		g.clearSuspects() // somebody else is the suspect: no cuffs on the players
	}
}

// copSeesAt: any officer (not a paramedic) with the point in front of him.
func (g *Game) copSeesAt(at Vec) bool {
	for _, c := range g.cops {
		if !c.medic && c.state != cLeave && g.sees(c.pos, c.dir, at, copRange, copDot) {
			return true
		}
	}
	return false
}

// suspectProblems lists inconsistencies in the state of the handcuffs (used by the tests).
func (g *Game) suspectProblems() []string {
	var out []string
	if g.playerCleared() && g.wanted > 0 {
		out = append(out, "player cleared but still wanted")
	}
	if g.playerCleared() {
		for _, c := range g.cops {
			if !c.medic && c.state == cChase {
				out = append(out, "an officer still chases a cleared player")
			}
		}
	}
	if g.lastKiller.kind == suspGuest && (g.lastKiller.guest == nil || g.lastKiller.guest.out()) {
		out = append(out, "suspect guest is gone")
	}
	return out
}

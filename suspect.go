package main

import (
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Who do the police go after? The LAST person somebody (a guest, a guard or an
// officer) actually saw killing. The handcuffs mark follows that person: if the
// special killer or an armed guest is seen killing, the player is cleared (the
// cuffs come off); if the player kills again and is seen, they go back on.
// Kills nobody sees change nothing. Everything that decides it lives here.
//
// ALERT vs WANTED (rule of the user, 2026-11-04). Being seen is not enough to wear the cuffs:
//
//  Before (audit of the old flow). witness() (game.go) runs on every kill by a player. Guests with the
//  player in front (canWatch, guestRange) get sawKiller; the nearest one becomes the caller
//  (becomeCaller) and the rest only alarm (!?). A cop with the player in front puts g.wanted = 45 at
//  once; a guard in front only gets sight = 99 (he chases/attacks, no cuffs). The guest in gAlarm then
//  either goes to a phone (gToPhone -> gCall -> callPolice -> raisePolice(informed): g.wanted = 70),
//  or, when an officer is near, walks to him (gReport; on arrival markPlayerWanted(70)), or else just
//  scares (gScared) and forgets. The cuffs mark shows while g.wanted > 0 (isSuspectP), but raisePolice
//  also took "any living witness" as informed, even one who had told nobody.
//
//  Now. Two states per player:
//   - ALERT: at least one guest saw THIS player kill (sawP) and has not warned yet (not guest.warned)
//     and is still on his way to do it: gAlarm with a phone/an officer to go to, gToPhone, gCall or
//     gReport (pendingWitnesses). A "!" mark pulses over the player (drawAlertMark), no cuffs.
//   - WANTED (cuffs): the warning HAS been given: the caller finishes the call (callPolice) or the
//     witness reaches the officer (updateReport) -> guestWarned() -> g.wanted = 70, the alert goes
//     away and the officers chase/arrest as ever. An officer who sees the kill himself is a direct
//     witness: the warning is immediate (wanted = 45, cuffs at once). A guard who sees it chases as
//     before and puts no cuffs on by himself (only the report of a witness or a call does).
//  Getting away: a witness who dies, is knocked out, is gone, or scares/forgets (gScared, gFlee, idle)
//  stops counting. If nobody is pending and nobody warned, the alert disappears and there are no
//  cuffs. With several witnesses the alert lasts until the last one warns (cuffs) or stops counting
//  (free). After a warning, killing the witness does not take the cuffs off: only the old rules do
//  (wardrobe/disguise, somebody else seen killing, case closed).
//  Two players: everything per player (sawP, jail.sus): J2's witness only alerts/cuffs J2.
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

// suspectSeen is the single decision point: kind killed and somebody saw it. The one who saw it is
// the police themselves (direct witness), so the cuffs go on at once.
func (g *Game) suspectSeen(kind suspectKind, gu *Guest) { g.suspectSeenBy(kind, gu, true) }

// suspectSeenBy: direct = an officer saw it. With direct false (a guest or a guard) the player becomes
// the last killer seen but wears no cuffs until a witness warns the police (guestWarned).
func (g *Game) suspectSeenBy(kind suspectKind, gu *Guest, direct bool) {
	if kind == suspPlayer && g.stealth {
		g.stealthDetect() // STEALTH mode: somebody saw the kill = detected (stealth.go)
	}
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
			if direct {
				g.markSuspect(g.p)
			}
		}
		if prev.kind == suspSlasher || prev.kind == suspGuest {
			// the cuffs come back at once if the authorities are around and an officer saw it
			if direct && g.policeCalled && g.lawActive() {
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

// ---- alert (a witness has seen the player and has not warned yet) ----------------------------

// witnessPending: gu saw p kill, has not warned yet and is still on his way to do it.
func (g *Game) witnessPending(gu *Guest, p *Player) bool {
	if gu == nil || gu.out() || gu.tech || !gu.sawKiller || gu.warned || gu.sawP != p {
		return false
	}
	switch gu.state {
	case gToPhone, gCall, gReport:
		return true
	case gAlarm:
		// still reacting: he will go to a phone / an officer if he can (guestReact, updateReport)
		return gu == g.callerGuest || g.nearCop(gu.pos) != nil
	}
	return false
}

// pendingWitnesses counts the witnesses who saw p and are about to warn the police.
func (g *Game) pendingWitnesses(p *Player) int {
	n := 0
	for _, gu := range g.guests {
		if g.witnessPending(gu, p) {
			n++
		}
	}
	return n
}

// playerAlert: the ALERT mark shows over p (witnesses pending, no cuffs yet, not cleared).
func (g *Game) playerAlert(p *Player) bool {
	if p == nil || p.dead || p.jail.held || g.isSuspectP(p) || g.playerCleared() {
		return false
	}
	return g.pendingWitnesses(p) > 0
}

// guestWarned: gu has told the police (call finished or officer reached). The alert turns into the
// cuffs for the player he saw.
func (g *Game) guestWarned(gu *Guest) {
	gu.warned = true
	if gu.sawKiller && gu.sawP != nil {
		g.markSuspect(gu.sawP) // two players: only the one he saw
	}
}

// drawAlertMark: a yellow-red "!" in a rounded badge, pulsing at 4 Hz, over the head; n > 1 shows the
// number of witnesses pending as a tiny digit. Not the cuffs, not the guests' "!?".
func (g *Game) drawAlertMark(dst *ebiten.Image, cx, bottom float64, n int) {
	ph := math.Sin(g.time * 2 * math.Pi * 4)
	hot := ph > 0
	body, edge := rgb(0xffd23a), rgb(0x6a1408)
	if hot {
		body = rgb(0xff4a2a)
	}
	w, h := 9.0, 12.0
	x, y := math.Round(cx-w/2), math.Round(bottom-h-math.Round(ph*1.2))
	fillRect(dst, x-1, y+1, w+2, h-2, edge)
	fillRect(dst, x+1, y-1, w-2, h+2, edge)
	fillRect(dst, x, y, w, h, body)
	ink := rgb(0x3a0a04)
	if hot {
		ink = rgb(0xfff2c0)
	}
	fillRect(dst, cx-1, y+2, 2, 5, ink) // the bar of the "!"
	fillRect(dst, cx-1, y+8, 2, 2, ink) // its dot
	if n > 1 {
		for _, o := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} { // dark outline: it is read on light floors
			g.drawTextC(dst, strconv.Itoa(n), cx+w/2+4+o[0], y+o[1], rgb(0x14102a), 1)
		}
		g.drawTextC(dst, strconv.Itoa(n), cx+w/2+4, y, rgb(0xffffff), 1)
	}
}

// drawAlertMarks draws the alert over every player who has one, above the J1/J2 tag and its cry.
func (g *Game) drawAlertMarks(dst *ebiten.Image) {
	if g.mode != ModePlay {
		return
	}
	for _, p := range g.players() {
		if p.hidden || !g.playerAlert(p) {
			continue
		}
		sc := g.charSc
		if sc <= 0 {
			sc = 1
		}
		top := math.Round(g.wy(p.pos.Y)) - 40*sc - 6 // where the J1/J2 tag sits
		if g.is2P() {
			top -= 12
			if p.msgT > 0 && p.msg != "" {
				top -= 11
			}
		}
		g.drawAlertMark(dst, math.Round(g.wx(p.pos.X)), top, g.pendingWitnesses(p))
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
	g.updateCopMemory() // DEBUG option "police with memory" (copmemory.go)
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

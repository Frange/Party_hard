package main

import (
	"math"
)

// ---------------------------------------------------------------------------
// Two local players (core).
//
// With g.twoPlayers on there is a second Player, g.p2. Almost the whole game was written for
// one player and reads "g.p" everywhere, so instead of rewriting every system the game runs
// them "as" a player:
//
//   g.asPlayer(i, fn)  runs fn with g.p pointing at player i and the input layer reading
//                      that player's keys or pad (inputP). Player 2's whole update (move,
//                      stab, interact, dance, items...) is player 1's code run this way.
//   g.focus(pos)       is for NPCs: while an NPC thinks, g.p is the player nearest to it,
//                      so guests, guards, animals, killers and the T-Rex react to whoever is
//                      closest, and each player is seen separately.
//
// With one player none of this runs and nothing changes.
//
// Rules of the mode (asked by the user):
//   - 20-30 more guests per party (guestCap grows by g.twoExtra).
//   - both start by the entrance of the level, a few steps apart.
//   - the players cannot kill each other: a stab at the other one knocks him down for 3 s
//     (stars over his head) and he cannot be hit again until 5 s more have passed.
//   - traps, poison, fire, the T-Rex, killers... can kill either of them like anyone else.
//   - the party is lost when BOTH are dead; the dead one stays on the floor as a body.
//   - points and statistics are shared: what either of them earns counts for the party,
//     for the wallet and for the trophies.
//   - up to two special killers at the same time (see slasher2 in killer.go).
// ---------------------------------------------------------------------------

const (
	hitDownT   = 3.0 // seconds on the floor after a punch from the other player
	hitSafeT   = 5.0 // seconds more during which he cannot be punched again
	playerGap  = 34  // distance between the two players at the start
	extraGuest = 20  // 20..30 more guests with two players
)

// dbgTwoPlayers is set by the -twop flag (debug): start every party with two players.
var dbgTwoPlayers bool

// players lists the players in the party (one or two).
func (g *Game) players() []*Player {
	if g.is2P() {
		return []*Player{g.first(), g.p2} // g.p may be player 2 while the game runs as him
	}
	return []*Player{g.p}
}

// playerIndex is 0 for the first player and 1 for the second.
func (g *Game) playerIndex(p *Player) int {
	if g.p2 != nil && p == g.p2 {
		return 1
	}
	return 0
}

// otherPlayer is the other one of the two (nil with one player).
func (g *Game) otherPlayer(p *Player) *Player {
	if !g.is2P() {
		return nil
	}
	if p == g.p2 || p.idx == 1 {
		return g.first()
	}
	return g.p2
}

// first is player 1 even while the game runs "as" player 2.
func (g *Game) first() *Player {
	if g.p1 != nil {
		return g.p1
	}
	return g.p
}

// asPlayer runs fn as player i (0 or 1): g.p is that player and the input layer reads his device.
func (g *Game) asPlayer(i int, fn func()) {
	if i == 0 || !g.is2P() {
		fn()
		return
	}
	if g.p1 == nil {
		g.p1 = g.p
	}
	oldP, oldIn := g.p, inputP
	g.p, inputP = g.p2, 1
	defer func() { g.p, inputP = oldP, oldIn }()
	fn()
}

// withPlayer runs fn with g.p pointing at pl (no change of input device).
func (g *Game) withPlayer(pl *Player, fn func()) {
	old := g.p
	g.p = pl
	fn()
	g.p = old
}

// nearestPlayer is the living player nearest to pos (preferring the ones in plain sight).
func (g *Game) nearestPlayer(pos Vec) *Player {
	best, bd := g.p, 1e18
	for _, pl := range g.players() {
		d := dist(pl.pos, pos)
		if pl.dead || pl.jail.held {
			d += 600
		} else if pl.hidden {
			d += 300
		}
		if d < bd {
			best, bd = pl, d
		}
	}
	return best
}

// focus makes g.p the player nearest to pos until the returned function is called:
// `defer g.focus(npc.pos)()`. With one player it does nothing.
func (g *Game) focus(pos Vec) func() {
	if !g.is2P() {
		return noFocus
	}
	old := g.p
	g.p = g.nearestPlayer(pos)
	return func() { g.p = old }
}

func noFocus() {}

// allPlayersDead: the party is lost.
func (g *Game) allPlayersDead() bool {
	for _, pl := range g.players() {
		if !pl.dead && !pl.jail.held { // a prisoner is out of the game until he escapes or is freed
			return false
		}
	}
	return true
}

// livePlayers counts the players still in the party.
func (g *Game) livePlayers() int {
	n := 0
	for _, pl := range g.players() {
		if !pl.dead {
			n++
		}
	}
	return n
}

// is2P: does this party have two players?
func (g *Game) is2P() bool { return g.p2 != nil }

// ringColor is the colour under each player's feet: blue for the first, red for the second.
func ringColor(p *Player) [3]uint8 {
	if p != nil && p.idx == 1 {
		return [3]uint8{255, 90, 90}
	}
	return [3]uint8{110, 170, 255}
}

// setupPlayers builds player 2 (when the party has two players) and puts both at the entrance.
func (g *Game) setupPlayers(def *LevelDef) {
	g.p1, g.p2 = nil, nil
	g.rev = reviveState{} // one call of the ambulance per party
	twoInput = false
	if g.p != nil {
		g.p.idx = 0
	}
	if !g.wantTwoPlayers() {
		return
	}
	playerDev = autoDevices()
	cs := g.playerCS(1) // the look of player 2 (wardrobe.go: red by default)
	g.p2 = &Player{pos: def.Start, dir: DirUp, cs: cs, stamina: g.perkStamina, style: g.rng.Intn(4), idx: 1}
	g.p1 = g.p
	// side by side at the entrance: a few steps apart, each on free ground
	a := g.lvl.NearestFree(Vec{def.Start.X - playerGap/2, def.Start.Y})
	b := g.lvl.NearestFree(Vec{def.Start.X + playerGap/2, def.Start.Y})
	if dist(a, b) < 20 { // a narrow spot: look for a place for the second one around the first
		b = g.lvl.NearestFree(Vec{a.X + playerGap, a.Y})
		if dist(a, b) < 20 {
			b = g.lvl.NearestFree(Vec{a.X, a.Y + playerGap})
		}
	}
	g.p.pos, g.p2.pos = a, b
	g.reviveDebugSetup()
}

// wantTwoPlayers: does this party have two players?
func (g *Game) wantTwoPlayers() bool {
	if g.stealth { // the stealth mode decides by itself (stealth2p.go): never the 2 players switch
		return g.stealth2
	}
	return dbgTwoPlayers || twoPlayerMode
}

// updateTwoPlayers: the timers of the second player and the immunity after a punch, and the
// input mirror (the input layer only splits the keyboard and the pads in play).
func (g *Game) updateTwoPlayers(dt float64) {
	if !g.is2P() {
		return
	}
	g.updateRevive(dt)
	for _, pl := range g.players() {
		if pl.immT > 0 {
			pl.immT -= dt
		}
		if pl.msgT > 0 {
			pl.msgT -= dt
		}
	}
}

// hitPlayer: att punches vic. Nobody dies: he falls for hitDownT and then cannot be punched
// again for hitSafeT. Returns false if the punch did nothing (he is still protected).
func (g *Game) hitPlayer(att, vic *Player) bool {
	if vic == nil || vic.dead || vic.hidden || vic.immT > 0 || vic.knockT > 0 {
		return false
	}
	vic.knockT = hitDownT
	vic.immT = hitDownT + hitSafeT
	vic.moving = false
	vic.msgT, vic.msg = 1.6, hitCries[g.rng.Intn(len(hitCries))]
	if vic.carrying != nil { // the body he was carrying falls beside him
		vic.carrying.carried = false
		vic.carrying.pos = vic.pos
		vic.carrying = nil
	}
	g.snd.Play("thud")
	g.shake = math.Max(g.shake, 0.12)
	g.burst(Vec{vic.pos.X, vic.pos.Y - 20}, 10, 60, rgb(0xf8e040), 0.6, 2, 40)
	if att != nil {
		att.stabT = 0.18
	}
	return true
}

var hitCries = []string{"¡AUCH!", "¡EH!", "¡ZAS!", "¡MI CABEZA!", "¡TÚ!", "¡PERO...!"}

// playerReach: is the other player within a stab of p, in front of him?
func (g *Game) playerInReach(p *Player, reach float64) *Player {
	o := g.otherPlayer(p)
	if o == nil || o.dead || o.hidden {
		return nil
	}
	if dist(p.pos, o.pos) < reach {
		return o
	}
	return nil
}

// playerName is how the messages call a player.
func (g *Game) playerName(p *Player) string {
	if p != nil && p.idx == 1 {
		return "J2"
	}
	return "J1"
}

// ---------------------------------------------------------------------------
// The second special killer. A party with one player has at most one killer at a time
// (as always); with two players there can be two. The game keeps its single-killer code and
// runs it twice: slot2 holds the second killer and its timers, and swapSlasherSlot puts it in
// the place the old code reads (g.slasher, g.slasherDone, g.slasherT, g.slasherN).
// ---------------------------------------------------------------------------

type slasherSlot struct {
	s    *Slasher
	done bool
	t    float64
	n    int
}

func (g *Game) swapSlasherSlot() {
	g.slasher, g.slot2.s = g.slot2.s, g.slasher
	g.slasherDone, g.slot2.done = g.slot2.done, g.slasherDone
	g.slasherT, g.slot2.t = g.slot2.t, g.slasherT
	g.slasherN, g.slot2.n = g.slot2.n, g.slasherN
}

// eachSlasher runs fn for the first killer slot and, with two players, for the second one too.
func (g *Game) eachSlasher(fn func()) {
	fn()
	if g.is2P() {
		g.swapSlasherSlot()
		fn()
		g.swapSlasherSlot()
	}
}

// allSlashers lists the special killers of the party (one or two).
func (g *Game) allSlashers() []*Slasher {
	var out []*Slasher
	if g.slasher != nil {
		out = append(out, g.slasher)
	}
	if g.slot2.s != nil {
		out = append(out, g.slot2.s)
	}
	return out
}

// updateSlashers is updateSlasher for every killer, each one reacting to the player nearest to him.
func (g *Game) updateSlashers(dt float64) {
	g.eachSlasher(func() {
		restore := noFocus
		if g.slasher != nil {
			restore = g.focus(g.slasher.pos)
		}
		g.updateSlasher(dt)
		restore()
	})
}

// hitAnySlasher: a stab at whichever special killer is within reach. Reports whether it hit one.
func (g *Game) hitAnySlasher(p *Player, reach float64) bool {
	hit := false
	g.eachSlasher(func() {
		if !hit && g.slasher != nil && g.slasher.state == slHunt && dist(p.pos, g.slasher.pos) < reach {
			g.hitSlasher()
			hit = true
		}
	})
	return hit
}

// eachPlayer runs fn for every player in the party, with g.p pointing at him (so g.die kills him).
func (g *Game) eachPlayer(fn func(p *Player)) {
	for _, pl := range g.players() {
		pl := pl
		g.withPlayer(pl, func() { fn(pl) })
	}
}

// botMove2 is botMove for the second player (debug: -walkbot with two players).
func (g *Game) botMove2(dt float64) (float64, float64) {
	g.bot2T -= dt
	if g.bot2T <= 0 {
		g.bot2T = 0.8 + g.rng.Float64()*2.2
		a := g.rng.Float64() * 2 * math.Pi
		g.bot2Dx, g.bot2Dy = math.Cos(a), math.Sin(a)
		if g.rng.Float64() < 0.5 {
			g.bot2Dx, g.bot2Dy = math.Round(g.bot2Dx), math.Round(g.bot2Dy)
		}
	}
	return g.bot2Dx, g.bot2Dy
}

// playerWantedNow: are the police after this player? The criminality level is one for the party
// (g.wanted) but the cuffs are personal: see isSuspectP in twojail.go.
func (g *Game) playerWantedNow(pl *Player) bool {
	if pl == nil || pl.dead || pl.disguised {
		return false
	}
	return g.isSuspectP(pl)
}

// disguiseClosesCase: with one player the disguise ends the search as always; with two it ends
// it only when every living player has changed his look (the other one is still being hunted).
func (g *Game) disguiseClosesCase() bool {
	if !g.is2P() {
		return true
	}
	for _, pl := range g.players() {
		if !pl.dead && !pl.disguised && g.isSuspectP(pl) {
			return false // somebody the police are still after has not changed his look
		}
	}
	return true
}

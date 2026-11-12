package main

import (
	"math"
)

// ---------------------------------------------------------------------------
// Level score: completion bonuses, the RETO (challenge) multipliers and the statistics box of the
// end-of-level screen (ModeLevelDone).
//
// HOW A PARTY IS SCORED (points and money are the same thing: g.points of the party are added to the
// wallet when the party ends, times the golden-wallet perk g.perkMoney, see bankPoints in shop.go):
//   - each kill by the player: +30 (guest), +50 (guard / guest killed by a trap or poison, "credit"),
//     +100 (police officer), +500 (the killer), plus small extras (dancing, gags in 2P...).
//   - NEW, STYLE: chain (each kill within chainWindow s of the previous one: kill value x1.2, x1.4... up to x3),
//     and "LIMPIO" +bonusClean for a kill nobody saw in cleanDelay s. Both x3 with the RETO. NOT DONE: the
//     "A LA CÁMARA" bonus (kill in front of a camera and leave unwanted): the camera LOS + "no wanted"
//     outcome is not a single event in camera.go, so it was left out.
//   - NEW, every completed level, with or without the challenge:
//       +bonusComplete   for finishing the party,
//       +bonusUnseen     if no witness ever saw you kill (run.seen == 0),
//       +bonusTimeFast / +bonusTimeMid  if it took less than timeFastS / timeMidS seconds.
//   - NEW, RETO (g.challenge: set by RETO DE HOY, daily.go; the C key is gone):
//       every kill is worth challengeKillMult times (x3, applied live through killVal, so the HUD shows it),
//       and when the level is COMPLETED the whole total (kills + bonuses) is multiplied by challengeFinishMult (x5).
//       Dying, being caught or leaving the party gives no x5 and no completion bonuses (the x3 on the
//       kills already scored is kept, like any point scored before losing).
// Nothing here writes progress.json by itself: the final total goes through the usual bankPoints.
// ---------------------------------------------------------------------------

const (
	bonusComplete       = 200 // finishing a party
	bonusUnseen         = 150 // nobody saw you killing in the whole party
	bonusTimeFast       = 300 // finished in less than timeFastS seconds
	bonusTimeMid        = 150 // ... or in less than timeMidS seconds
	timeFastS           = 180.0
	timeMidS            = 300.0
	challengeKillMult   = 3 // RETO: value of every kill
	challengeFinishMult = 5 // RETO: the total when the level is completed
	// STYLE points
	chainWindow  = 4.0 // a kill within this many seconds of the previous one of the same player continues the chain
	chainStep    = 0.2 // each kill of the chain adds this to the multiplier of the kill: x1, x1.2, x1.4...
	chainMaxMult = 3.0 // ...up to this (the 11th kill of a chain)
	bonusClean   = 50  // "LIMPIO": a kill nobody saw (no witness alert for cleanDelay seconds)
	cleanDelay   = 3.0
)

// killCause: how a kill of the player happened (the statistics box).
type killCause int

const (
	kcNone  killCause = iota // not set: the knife / hands
	kcKnife                  // knife, hands (also guards, officers and the killer)
	kcPoison
	kcTrap  // traps: gas, fireworks, piano, electricity of the pools...
	kcBlast // explosions
	kcN
)

// finishBreak is the points breakdown shown when the level is completed.
type finishBreak struct {
	base       int // the points of the party without the extras below
	chain      int // style: extra of the kill chains (before the RETO)
	clean      int // style: clean kills (before the RETO)
	streak     int // style: extra of the RACHA multiplier (before the RETO, streak.go)
	streakBest int // the best streak of the party (either player)
	killExtra  int // RETO: the extra of the x3 on kills
	complete   int
	unseen     int
	speed      int
	finalExtra int // RETO: the extra of the x5 on completion
	total      int
	wallet     int // what goes to the wallet (total x the golden wallet perk)
	challenge  bool
}

// runStats: what happened in the party, per player (index 0 = J1, 1 = J2). It lives in memory only.
type runStats struct {
	kills                [2][kcN]int
	nonGuest             int // kills counted above that were not guests (guards, officers, the killer)
	seen                 [2]int
	cuffed               [2]int
	wasSus               [2]bool
	killExtra            int        // points added by the RETO x3
	chainExtra, cleanPts int        // style points (before the RETO x3, whose extra goes to killExtra)
	chainN               [2]int     // current chain
	chainLast            [2]float64 // time of the last kill of the chain
	chainMax             [2]int
	cleanAt              [2]float64 // when the last kill becomes a clean one (0 = none pending)
	cleanKills           [2]int
	lastPl               int
	camSeen              int  // times a camera recorded the player
	units                uint // bit per police unit kind that came (1<<unitPolice...)
	medic                bool // an ambulance came
	jailed               int  // 2P: times a player was taken into custody
	wasHeld              [2]bool
	slasher              bool // another killer walked the party
	knifed               bool // the knife (or hands, guards, officers, killer) took a life: no "noknife" trophy
	fin                  finishBreak
	done                 bool // fin is the real final breakdown (the level was completed)
	// MEDALS (medals.go): set by awardMedal when the party is won; the newspaper shows them
	medalOn   bool // this party gives a medal
	medal     int  // the medal these points won (medalNone..medalGold)
	medalNew  bool // ...and it is better than the one saved before
	medalGoal int  // the kill target the thresholds were measured against
	medalSaid int  // the best medal already announced in this party (the toast, medals.go medalLive)
	// RACHA (streak.go): clean kills in a row per player and the extra points of its multiplier
	streakN, streakBest [2]int
	streakExtra         int        // extra of the streak multiplier (before the RETO x3, whose extra goes to killExtra)
	streakPts           [2]int     // ...per player
	streakUndo          [2]int     // extra of the LAST kill (before the RETO): taken back if that kill is seen
	streakUndoT         [2]float64 // when that last kill was scored (only the same frame can take it back)
}

func (r *runStats) killsOf(i int) (n int) {
	for c := kcKnife; c < kcN; c++ {
		n += r.kills[i][c]
	}
	return n + r.kills[i][kcNone]
}

func (r *runStats) total(c killCause) int { return r.kills[0][c] + r.kills[1][c] }

// killVal is the value of one kill: x3 with the RETO (and the extra is remembered for the breakdown).
// The chain multiplier of the player who just killed (countKill ran first) applies too.
func (g *Game) killVal(n int) int {
	if g.stealth {
		return n
	}
	r := &g.run
	m := math.Min(chainMaxMult, 1+chainStep*float64(maxI(r.chainN[r.lastPl]-1, 0)))
	v := int(math.Round(float64(n) * m))
	r.chainExtra += v - n
	v = g.streakApply(v) // RACHA: x1.1 ... x1.5 for a run of clean kills (streak.go)
	if !g.challenge {
		return v
	}
	extra := v * (challengeKillMult - 1)
	r.killExtra += extra
	return v + extra
}

// styleVal: a flat style bonus, x3 with the RETO.
func (g *Game) styleVal(n int) int {
	if g.challenge && !g.stealth {
		g.run.killExtra += n * (challengeKillMult - 1)
		return n * challengeKillMult
	}
	return n
}

// chainNow: the live chain of player i (0 when it has expired).
func (g *Game) chainNow(i int) int {
	if g.run.chainN[i] > 0 && g.time-g.run.chainLast[i] <= chainWindow {
		return g.run.chainN[i]
	}
	return 0
}

// tickClean pays the "LIMPIO" bonus of the kills nobody saw. force pays what is pending now (end of party).
func (g *Game) tickClean(force bool) {
	r := &g.run
	for i := 0; i < 2; i++ {
		if r.cleanAt[i] <= 0 || (!force && g.time < r.cleanAt[i]) {
			continue
		}
		r.cleanAt[i] = 0
		r.cleanKills[i]++
		r.cleanPts += bonusClean
		v := g.styleVal(bonusClean)
		pos := g.p.pos
		if pl := g.players(); i < len(pl) && pl[i] != nil {
			pos = pl[i].pos
		}
		if force {
			g.points += v
		} else {
			g.award(pos, v, "LIMPIO")
		}
	}
}

// statPlayer: which player a kill is credited to. Traps and explosions go to the owner of the last trap.
func (g *Game) statPlayer(c killCause) int {
	if !g.is2P() {
		return 0
	}
	if (c == kcTrap || c == kcBlast) && g.coop.lastTrap != nil && g.time-g.coop.lastTrapT <= 12 {
		if o, ok := g.coop.trapOwner[g.coop.lastTrap]; ok && o >= 0 && o < 2 {
			return o
		}
	}
	return g.playerIndex(g.p)
}

// countKill adds one kill of the player to the statistics.
func (g *Game) countKill(c killCause, guest bool) {
	if c == kcNone {
		c = kcKnife
	}
	pi := g.statPlayer(c)
	r := &g.run
	if c == kcKnife {
		r.knifed = true
	}
	r.kills[pi][c]++
	r.lastPl = pi
	if g.time-r.chainLast[pi] <= chainWindow && r.chainN[pi] > 0 {
		r.chainN[pi]++
	} else {
		r.chainN[pi] = 1
	}
	r.chainLast[pi] = g.time
	r.chainMax[pi] = maxI(r.chainMax[pi], r.chainN[pi])
	g.statMax("chain", r.chainN[pi])
	r.cleanAt[pi] = g.time + cleanDelay
	g.streakKill(pi) // RACHA (streak.go)
	if !guest {
		g.run.nonGuest++
	}
}

// trackCuffs counts the times each player got the handcuffs mark (a rising edge, polled every frame).
func (g *Game) trackCuffs() {
	g.tickClean(false)
	g.streakTick()             // the streak breaks while a player is wanted / alerted / held (streak.go)
	g.medalLive()              // the medal toast when the points reach bronze / silver / gold (medals.go)
	for _, c := range g.cops { // who came (the newspaper)
		if c.medic {
			g.run.medic = true
		} else if c.kind >= 0 && c.kind < 16 {
			g.run.units |= 1 << uint(c.kind)
		}
	}
	if g.slasherN > 0 {
		g.run.slasher = true
	}
	for i, pl := range g.players() {
		if i < 2 && pl != nil {
			if pl.jail.held && !g.run.wasHeld[i] {
				g.run.jailed++
			}
			g.run.wasHeld[i] = pl.jail.held
		}
	}
	escaped := false
	for i, pl := range g.players() {
		if i > 1 {
			break
		}
		s := g.isSuspectP(pl)
		if s && !g.run.wasSus[i] {
			g.run.cuffed[i]++
		}
		if g.escapeTick(pl, g.run.wasSus[i], s) { // the search ended and he is free (copsets.go)
			escaped = true
		}
		g.run.wasSus[i] = s
	}
	if escaped {
		g.cuffEscaped()
	}
}

// speedBonus: the bonus for the time of the party.
func speedBonus(t float64) int {
	switch {
	case t < timeFastS:
		return bonusTimeFast
	case t < timeMidS:
		return bonusTimeMid
	}
	return 0
}

// previewFinish computes the final breakdown from the current state without changing anything.
func (g *Game) previewFinish() finishBreak {
	f := finishBreak{base: g.points - g.run.killExtra - g.run.chainExtra - g.run.cleanPts - g.run.streakExtra, killExtra: g.run.killExtra, chain: g.run.chainExtra, clean: g.run.cleanPts, complete: bonusComplete, challenge: g.challenge,
		streak: g.run.streakExtra, streakBest: maxI(g.run.streakBest[0], g.run.streakBest[1])}
	if g.run.seen[0]+g.run.seen[1] == 0 {
		f.unseen = bonusUnseen
	}
	f.speed = speedBonus(g.endTime)
	sum := g.points + f.complete + f.unseen + f.speed
	f.total = sum
	if g.challenge {
		f.total = sum * challengeFinishMult
		if g.dc != nil {
			f.total += dailyBonus // RETO DE HOY: extra on top (shown inside the x5 line)
		}
		f.finalExtra = f.total - sum
	}
	f.wallet = int(float64(f.total) * g.perkMoney)
	return f
}

// finishScore: the party has been completed. Adds the bonuses (and the x5 of the RETO) to the points.
// Called once, before the points are banked. Not called when the party is lost.
func (g *Game) finishScore() {
	if g.run.done || g.stealth {
		return
	}
	g.endTime = g.time
	g.tickClean(true) // kills nobody saw by now are clean
	g.run.fin = g.previewFinish()
	g.points = g.run.fin.total
	g.run.done = true
}

// levelDoneDemo is the debug flag -leveldonedemo: the end-of-level screen with sample data (frozen progress).
var levelDoneDemo int // any value > 0: the newspaper (the only end screen); 1 or 2 players come from -players

func (g *Game) levelDoneDemoSetup() {
	if levelDoneDemo == 0 || g.time < 0.5 || g.mode != ModePlay {
		return
	}
	levelDoneDemo = 0
	g.run.kills = [2][kcN]int{{kcKnife: 9, kcPoison: 3, kcTrap: 2}, {kcKnife: 4, kcBlast: 5}}
	g.run.nonGuest = 1
	g.run.seen, g.run.cuffed = [2]int{3, 1}, [2]int{1, 0}
	g.points = 1210
	if g.challenge {
		g.run.killExtra = 600
	}
	if g.is2P() {
		g.coopEnsure()
		g.coop.score[0] = coopStats{points: 640, kills: 14, hits: 1, floorT: 3, danceT: 12, fives: 1}
		g.coop.score[1] = coopStats{points: 570, kills: 9, hits: 2, floorT: 9, helps: 2, fives: 1, danceT: 3}
	}
	g.run.chainMax, g.run.cleanKills = [2]int{6, 3}, [2]int{4, 2}
	g.run.chainExtra, g.run.cleanPts = 180, 6*bonusClean
	g.points += g.run.chainExtra + g.run.cleanPts
	g.run.streakBest, g.run.streakPts, g.run.streakExtra = [2]int{9, 4}, [2]int{48, 6}, 54 // RACHA (streak.go)
	g.points += g.run.streakExtra
	g.time = 212
	if !g.is2P() { // one player: nothing belongs to J2
		g.run.kills[1], g.run.seen[1], g.run.cuffed[1], g.run.chainMax[1], g.run.cleanKills[1] = [kcN]int{}, 0, 0, 0, 0
	}
	g.run.slasher, g.run.camSeen, g.run.medic = true, 1, true
	g.run.units = 1<<unitPolice | 1<<unitSWAT
	g.finishScore()
	g.awardMedal() // the medal of the party in the points table (medals.go); the progress is frozen here
	g.mode = ModeLevelDone
}

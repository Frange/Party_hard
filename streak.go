package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// RACHA (idea 237): a run of CLEAN kills in a row multiplies the points of the next kills a little.
//
// THE RULE (one per player, J1 and J2 independent):
//   - every kill of the player (knife, guards, officers, the killer, and his traps / poison / explosions credited
//     to him, levelscore.go countKill) adds 1 to his streak, unless at that moment he is wanted, handcuffed, held,
//     dead or a witness is on the way to warn the police (then it stays at 0);
//   - the streak BREAKS (back to 0) when: a kill of his is SEEN (witnessAct: guests, guards or officers; that kill
//     loses its streak extra too), a security camera films him (camera.go), or while the police want him / a
//     witness alert is over him / he is held or dead (streakTick, every frame);
//   - the multiplier (streakMult) goes on the kill value after the chain multiplier and before the RETO x3:
//     x1.1 from 3 kills, x1.2 from 5, x1.3 from 8, x1.4 from 12 and x1.5 (the top) from 16.
//   - never in STEALTH (its own count). The extra is kept apart (run.streakExtra) for the newspaper row
//     "Racha · mejor N" and the per-player line (streakline.go). Nothing of this is saved in progress.json.
//
// On screen: a small "RACHA N" under the feet of the player once the streak is 2 or more (with its xM from 3).
// ---------------------------------------------------------------------------

// streakTiers: from n clean kills, the multiplier (the first row that fits wins).
var streakTiers = []struct {
	n int
	m float64
}{{16, 1.5}, {12, 1.4}, {8, 1.3}, {5, 1.2}, {3, 1.1}}

const streakShowMin = 2 // the counter shows from this streak on

// streakMult: the multiplier of a streak of n clean kills (1 below 3, 1.5 at most).
func streakMult(n int) float64 {
	for _, t := range streakTiers {
		if n >= t.n {
			return t.m
		}
	}
	return 1
}

// streakBlocked: player i cannot grow a streak now (wanted, alerted, held, dead).
func (g *Game) streakBlocked(i int) bool {
	pl := g.players()
	if i < 0 || i >= len(pl) || pl[i] == nil {
		return true
	}
	p := pl[i]
	return p.dead || p.jail.held || g.isSuspectP(p) || g.playerAlert(p)
}

// streakKill: a kill was credited to player i (countKill).
func (g *Game) streakKill(i int) {
	if g.stealth || i < 0 || i > 1 {
		return
	}
	r := &g.run
	if g.streakBlocked(i) {
		r.streakN[i] = 0
		return
	}
	r.streakN[i]++
	r.streakBest[i] = maxI(r.streakBest[i], r.streakN[i])
}

// streakApply: the kill value v (after the chain) with the streak multiplier of the player who just killed.
// It remembers the extra so that, if this very kill is seen in the same frame, it is taken back (streakSeen).
func (g *Game) streakApply(v int) int {
	r := &g.run
	i := r.lastPl
	if g.stealth || i < 0 || i > 1 {
		return v
	}
	r.streakUndo[i] = 0
	m := streakMult(r.streakN[i])
	if m <= 1 {
		return v
	}
	nv := int(math.Round(float64(v) * m))
	d := nv - v
	r.streakExtra += d
	r.streakPts[i] += d
	r.streakUndo[i], r.streakUndoT[i] = d, g.time
	return nv
}

// streakSeen: a kill of player i was seen. The streak breaks, and the extra of the kill scored in this same frame
// (the one they saw) is taken back (with its RETO x3).
func (g *Game) streakSeen(i int) {
	if i < 0 || i > 1 {
		return
	}
	r := &g.run
	if d := r.streakUndo[i]; d > 0 && r.streakUndoT[i] == g.time {
		r.streakExtra -= d
		r.streakPts[i] -= d
		lost := d
		if g.challenge {
			r.killExtra -= d * (challengeKillMult - 1)
			lost = d * challengeKillMult
		}
		g.points -= lost
	}
	r.streakUndo[i] = 0
	g.streakBreak(i)
}

// streakBreak: player i loses his streak (seen, filmed, wanted...).
func (g *Game) streakBreak(i int) {
	if i >= 0 && i < 2 {
		g.run.streakN[i] = 0
	}
}

// streakDemo: the debug flag -streakdemo N (progress frozen): every player starts the party with a streak of N.
var streakDemo int

// streakTick: every frame of the party (trackCuffs). A wanted / alerted / held / dead player has no streak.
func (g *Game) streakTick() {
	if streakDemo > 0 && g.time > 0.5 && g.mode == ModePlay {
		for i := range g.players() {
			if i < 2 {
				g.run.streakN[i] = streakDemo
				g.run.streakBest[i] = maxI(g.run.streakBest[i], streakDemo)
			}
		}
		streakDemo = 0
		return
	}
	for i := 0; i < 2; i++ {
		if g.run.streakN[i] > 0 && g.streakBlocked(i) {
			g.run.streakN[i] = 0
		}
	}
}

// streakText: the counter of a streak of n ("RACHA 5 x1,2"; "RACHA 2" before the multiplier starts).
func streakText(n int) string {
	m := streakMult(n)
	if m <= 1 {
		return fmt.Sprintf("RACHA %d", n)
	}
	t := int(math.Round(m * 10))
	return fmt.Sprintf("RACHA %d x%d,%d", n, t/10, t%10)
}

// streakLabelRect: where the counter of player p goes: centred under his feet (under the ring), whole pixels,
// kept 16 px inside the screen. ok=false when there is nothing to show.
func (g *Game) streakLabelRect(p *Player, i int) (s string, x, y, w, h float64, ok bool) {
	if p == nil || g.mode != ModePlay || g.stealth || i < 0 || i > 1 || g.run.streakN[i] < streakShowMin || p.dead || g.coopHidden(p) {
		return "", 0, 0, 0, 0, false
	}
	s = streakText(g.run.streakN[i])
	w, h = math.Ceil(g.textW(s, 1)), 8
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	x = math.Round(g.wx(p.pos.X) - w/2)
	y = math.Round(g.wy(p.pos.Y) + 6*sc + 4)
	if y+h+12 > g.streakBottomFree() { // the keys help of the start is at the bottom: over the head instead
		y = math.Round(g.wy(p.pos.Y) - 40*sc - 6 - 12 - h)
		if g.is2P() {
			y -= 12 // above the J1 / J2 tag
		}
	}
	x = clampF(x, 16, screenW-16-w)
	y = clampF(y, 16, screenH-16-h)
	return s, x, y, w, h, true
}

// streakBottomFree: the y where the keys help at the bottom of the screen starts while it is shown (draw.go: one
// line for 1 player during the first 16 s, players.go drawControlsHelp for 2); the screen height otherwise.
func (g *Game) streakBottomFree() float64 {
	switch {
	case twoPlayerMode && g.time <= controlsHelpTime:
		return screenH - 64
	case !twoPlayerMode && g.time < 16:
		return screenH - 26
	}
	return screenH
}

// streakLabel: one counter on screen.
type streakLabel struct {
	i          int
	s          string
	x, y, w, h float64
}

// streakLabels: the counters to draw now. With two players, when both labels would touch (the players stand
// together) J2's moves 4 px above J1's, so they never overlap. A label over the shown HUD panel is left out.
func (g *Game) streakLabels() []streakLabel {
	var out []streakLabel
	for i, p := range g.players() {
		s, x, y, w, h, ok := g.streakLabelRect(p, i)
		if !ok {
			continue
		}
		for _, o := range out {
			if x < o.x+o.w+4 && x+w+4 > o.x && y < o.y+o.h+4 && y+h+4 > o.y {
				y = math.Max(16, o.y-h-4)
				if y < o.y+o.h+4 && y+h+4 > o.y { // no room above (top edge): under it
					y = o.y + o.h + 4
				}
			}
		}
		if g.hudAlpha() > 0 { // the panel is (partly) on screen: the counter never paints over it
			x0, y0, x1, y1 := g.hudHideRect()
			if x < x1 && x+w > x0 && y < y1 && y+h > y0 {
				continue
			}
		}
		out = append(out, streakLabel{i, s, x, y, w, h})
	}
	return out
}

// drawStreaks: the counter of each player (before the HUD panel). 1P: green, gold once it multiplies; 2P: the
// colour of the player's ring, so each one knows his own.
func (g *Game) drawStreaks(dst *ebiten.Image) {
	pl := g.players()
	for _, l := range g.streakLabels() {
		col := rgb(0x9cf0b0)
		if streakMult(g.run.streakN[l.i]) > 1 {
			col = rgb(0xffe060)
		}
		if g.is2P() && l.i < len(pl) {
			c := ringColor(pl[l.i])
			col = rgb(uint32(c[0])<<16 | uint32(c[1])<<8 | uint32(c[2]))
		}
		g.drawText(dst, l.s, l.x, l.y, col, 1)
	}
}

func init() {
	for k, v := range map[string]string{
		"RACHA %d":         "STREAK %d",
		"RACHA %d x%d,%d":  "STREAK %d x%d.%d",
		"Racha · mejor %d": "Streak · best %d",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

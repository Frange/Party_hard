package main

import (
	"flag"
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// PARTY COUNTDOWN (idea 255): in SOME parties (about 1 in 12: cdChance, own dice, rolled when the party loads) the
// neighbours have had enough and the party has a closing time: a line under the HUD panel says "LA FIESTA ACABA EN
// m:ss". The time depends on the people of the party (cdDuration: 7 minutes plus 6 s per guest over 40 of the kill
// target, between 7 and 10 minutes). It runs on the game time (g.time), so the pause and the windows that stop the game
// stop it too; two players share one clock.
//   - finish the party before it reaches 0: nothing happens (no bonus, no penalty);
//   - it reaches 0: the police come in force (a call with the player wanted, three stars and two more cars on top of
//     the first one; nobody dies on the spot) and the party is CLOSED: no more guests arrive (cap.go arrivalsClosed).
//
// Never in STEALTH, the RETO DE HOY (it has its own clock), the island, a level tested from the editor, nor in DEBUG
// (the in-game DEBUG mode or any debug/capture flag: progressFrozen), unless a test or -countdownnow forces it. It writes
// nothing to the progress.
// ---------------------------------------------------------------------------

const (
	cdChance  = 1.0 / 12
	cdMinDur  = 420.0 // 7:00
	cdMaxDur  = 600.0 // 10:00
	cdPerHead = 6.0   // seconds per guest of the kill target over cdBase
	cdBase    = 40
	cdStars   = 3
	cdExtra   = 2 // police cars sent on top of the first one
)

type cdState struct {
	on, fired bool
	dur, end  float64
	extra     int
	extraAt   float64
}

var (
	cdRng  *rand.Rand
	cdSeed int64 // tests: non-zero = deterministic dice
	// cdForce: -countdownnow (debug) or a test: every party that may have a countdown gets one (also in DEBUG);
	// cdForceDur > 0 sets its length in seconds (to see the end without waiting minutes).
	cdForce    bool
	cdForceDur float64
	cdNowF     = flag.Float64("countdownnow", 0, "force the party countdown with this length in seconds (debug; 0 = off)")
)

// cdForced: a test or -countdownnow forces the countdown (and its length, when given).
func cdForced() (bool, float64) {
	if cdNowF != nil && *cdNowF > 0 {
		return true, *cdNowF
	}
	return cdForce, cdForceDur
}

func cdDice() *rand.Rand {
	if cdRng == nil {
		sd := cdSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x0c10c4
		}
		cdRng = rand.New(rand.NewSource(sd))
	}
	return cdRng
}

// cdRoll: does this party get a countdown? (1 in 12)
func cdRoll(r *rand.Rand) bool { return r.Float64() < cdChance }

// cdDuration: the closing time of a party with this kill target: more people, more time (7 to 10 minutes).
func cdDuration(total int) float64 {
	d := cdMinDur + float64(total-cdBase)*cdPerHead
	if d < cdMinDur {
		return cdMinDur
	}
	if d > cdMaxDur {
		return cdMaxDur
	}
	return d
}

// cdAllowed: the parties where a countdown may happen.
func (g *Game) cdAllowed() bool {
	if g.lvl == nil || g.stealth || g.dc != nil || g.levelIdx == specialJurassic || g.fromEditor {
		return false
	}
	force, _ := cdForced()
	return force || cdRollFrozen || !progressFrozen
}

// cdRollFrozen: tests only: roll the real dice although the progress is frozen (the tests never unfreeze it).
var cdRollFrozen bool

// cdLevelReset: called at the end of loadLevel, once the kill target (g.total) is known.
func (g *Game) cdLevelReset() {
	g.cd = cdState{}
	if !g.cdAllowed() {
		return
	}
	force, fdur := cdForced()
	roll := cdRoll(cdDice()) // always rolled: the dice do not depend on the forcing
	if !roll && !force {
		return
	}
	g.cd.on, g.cd.dur = true, cdDuration(g.total)
	if force && fdur > 0 {
		g.cd.dur = fdur
	}
	g.cd.end = g.time + g.cd.dur
}

// cdLeft: seconds left (ok false = no countdown in this party).
func (g *Game) cdLeft() (float64, bool) {
	if !g.cd.on {
		return 0, false
	}
	l := g.cd.end - g.time
	if l < 0 {
		l = 0
	}
	return l, true
}

// cdTick: called every frame of the party (updatePlay).
func (g *Game) cdTick() {
	if !g.cd.on || g.mode != ModePlay {
		return
	}
	if g.cd.fired {
		if g.cd.extra > 0 && g.time >= g.cd.extraAt && !g.dbg.noPolice {
			g.cd.extra--
			g.cd.extraAt = g.time + 4
			g.dispatch(false)
		}
		return
	}
	if g.time < g.cd.end {
		return
	}
	g.cd.fired = true
	g.say("¡Hora de cerrar! La policía disuelve la fiesta")
	g.snd.Play("siren")
	if g.dbg.noPolice {
		return
	}
	at := g.p.pos
	if !g.policeCalled {
		g.raisePolice(at, true)
	} else {
		g.markPlayerWanted(32)
	}
	if g.stars < cdStars {
		g.stars = cdStars
	}
	if g.starFloor < g.stars {
		g.starFloor = g.stars
	}
	g.cd.extra, g.cd.extraAt = cdExtra, g.time+8
}

// cdClosed: the countdown reached 0: the party takes nobody else (cap.go arrivalsClosed).
func (g *Game) cdClosed() bool { return g.cd.on && g.cd.fired }

// cdLineH: the room the countdown line takes in the column under the HUD panel (hud.go hudSide).
func (g *Game) cdLineH() float64 {
	if g.cd.on && g.mode == ModePlay {
		return 14
	}
	return 0
}

// cdText: the (Spanish) text of the line, translated by drawText, and its colour (red in the last minute, blinking by whole steps, never a fade).
func (g *Game) cdText() (string, color.RGBA) {
	left, _ := g.cdLeft()
	if g.cd.fired {
		return "¡FIESTA CERRADA! Llega la policía", rgb(0xff4a3a)
	}
	col := rgb(0xffc040)
	if left < 60 {
		col = rgb(0xff6a4a)
		if int(left*2)%2 == 0 {
			col = rgb(0xffffff)
		}
	}
	return fmt.Sprintf("LA FIESTA ACABA EN %d:%02d", int(left)/60, int(left)%60), col
}

// cdRect: where the line goes (screen pixels): the top of the column under the HUD panel.
func (g *Game) cdRect() (x, y, w, h float64) {
	s, _ := g.cdText()
	return hudX, g.hud.bottom - 3 + hudSideGap, 11 + g.textW(s, 1), 9
}

// drawCountdown: a small clock and the line, under the HUD panel (it follows the panel when a player hides it).
func (g *Game) drawCountdown(dst *ebiten.Image) {
	if !g.cd.on || g.mode != ModePlay || g.hud.hide >= 0.5 {
		return
	}
	s, col := g.cdText()
	x, y, _, _ := g.cdRect()
	dark := rgb(0x1a0c10)
	fillRect(dst, x, y, 7, 7, dark)
	fillRect(dst, x+1, y+1, 5, 5, col)
	fillRect(dst, x+2, y+2, 3, 3, dark)
	fillRect(dst, x+3, y+2, 1, 2, col)
	fillRect(dst, x+3, y+3, 2, 1, col)
	g.drawText(dst, s, x+11, y, col, 1)
}

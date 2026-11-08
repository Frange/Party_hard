package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// STEALTH mode for TWO PLAYERS (DEBUG only for now): "player 1 is the decoy, player 2 sneaks past".
//
// ENTRY POINTS: the DEBUG menu row stealth2pMenuLabel and the flag -stealth2p [-level N] (main.go).
// Not in the main menu. It is the stealth mode of stealth.go with these differences:
//   - detection is PER PLAYER: each one has his own meter and his own cardboard box (E for player 1,
//     the interact key of player 2, input2.go); cameras and vision cones count for each one.
//   - the phase panel (red ALERT / yellow CAUTION) is shared: if either is seen the alert is for both,
//     and the panel says who caused it (sthState.culprit, sthState.by).
//   - THE DECOY: the dance key (F / "/") makes a NOISE (a whistle) instead of dancing. It needs a guard or
//     an officer within sthNoiseNear. The noise falls sthNoiseThrow px in front of the player, and every
//     guard and officer within sthNoiseRange of that point (and not chasing) goes there and stays for
//     sthNoiseTime s. The noise itself is NOT a detection: it only moves the watchers, so the partner
//     can pass behind them. Whoever stands in a cone is still seen as usual.
//   - the party is won as in the normal mode (nobody left alive); the end panel says
//     "SIN SER VISTO" only if neither player was ever detected.
// With one player this file does nothing (g.stealth2 is false).
// ---------------------------------------------------------------------------

const stealth2pMenuLabel = "Sigilo 2 jugadores (J1 señuelo, J2 pasa por detrás)"

// stealthWhistle: the decoy key. True when it was used here (so it does not turn into a dance).
func (g *Game) stealthWhistle(p *Player) bool {
	if !g.stealth || !g.stealth2 || p == nil || p.dead || p.carrying != nil {
		return false
	}
	s := &g.sth
	i := g.playerIndex(p)
	switch {
	case s.of(i).box:
		g.say(fmt.Sprintf("Quita la caja para hacer ruido (%s)", actKeyName(i, actInteract)))
	case g.time < s.cd[i]:
		g.say("Aún no puedes volver a hacer ruido")
	case !g.authorityNear(p.pos, sthNoiseNear):
		g.say("Acércate a un guardia o a un poli para hacer ruido")
	default:
		at := dirVec(p.dir)
		to := g.lvl.NearestFree(Vec{p.pos.X + at.X*sthNoiseThrow, p.pos.Y + at.Y*sthNoiseThrow})
		g.stealthNoiseAt(to, i)
		g.coopSayNow(p, pickOf(g, []string{"¡Psst!", "¡Eh, por aquí!", "*silbido*"}))
	}
	return true
}

// authorityNear: a guard or an officer (not a medic nor one who is leaving) within r of pos.
func (g *Game) authorityNear(pos Vec, r float64) bool {
	for _, gd := range g.guards {
		if !gd.dead && dist(gd.pos, pos) < r {
			return true
		}
	}
	for _, c := range g.cops {
		if !c.medic && c.state != cLeave && dist(c.pos, pos) < r {
			return true
		}
	}
	return false
}

// stealthNoiseAt: a noise falls at `to`, made by player i. It is not a detection.
func (g *Game) stealthNoiseAt(to Vec, i int) {
	s := &g.sth
	s.noisePos, s.noiseT, s.noiseAt, s.noiseFx = to, sthNoiseTime, 0, 1.2
	s.cd[i] = g.time + sthNoiseCD
	s.whistles[i]++
	g.snd.Play("boing")
}

// updateStealthNoise: while the noise lasts, the free guards and officers near it walk to it and stay.
func (g *Game) updateStealthNoise(dt float64) {
	s := &g.sth
	if s.noiseFx > 0 {
		s.noiseFx -= dt
	}
	if !g.stealth2 || s.noiseT <= 0 {
		return
	}
	s.noiseT -= dt
	if s.noiseAt -= dt; s.noiseAt > 0 {
		return
	}
	s.noiseAt = 0.4
	at := s.noisePos
	for _, gd := range g.guards {
		if gd.dead || dist(gd.pos, at) > sthNoiseRange {
			continue
		}
		switch gd.state {
		case gdPatrol, gdWait, gdReturn, gdInvestigate, gdSearch:
		default:
			continue
		}
		if dist(gd.pos, at) > 18 {
			if gd.state != gdInvestigate || len(gd.path) == 0 {
				gd.state = gdInvestigate
				gd.path = g.lvl.Path(gd.pos, at)
			}
		} else {
			gd.state, gd.timer, gd.path = gdWait, 1, nil
			gd.dir = faceTo(gd.pos, at)
		}
		if gd.icon == "" {
			gd.icon, gd.iconT = "?", 1.2
		}
	}
	for _, c := range g.cops {
		if c.medic || dist(c.pos, at) > sthNoiseRange {
			continue
		}
		switch c.state {
		case cGoto, cSweep, cReturn, cSearch:
		default:
			continue
		}
		if dist(c.pos, at) > 18 {
			if c.state != cGoto || len(c.path) == 0 {
				c.state = cGoto
				c.path = g.lvl.Path(c.pos, at)
			}
		} else {
			c.state, c.wait, c.path = cSweep, 1, nil
			c.dir = faceTo(c.pos, at)
		}
		if c.icon == "" {
			c.icon, c.iconT = "?", 1.2
		}
	}
}

// drawStealthNoise: a ring where the noise fell.
func (g *Game) drawStealthNoise(dst *ebiten.Image) {
	s := &g.sth
	if !g.stealth2 || s.noiseFx <= 0 {
		return
	}
	k := 1 - s.noiseFx/1.2
	r := 10 + 36*k
	x, y := g.wx(s.noisePos.X), g.wy(s.noisePos.Y)
	a := uint8(200 * (1 - k))
	for t := 0; t < 24; t++ {
		ang := float64(t) / 24 * 2 * math.Pi
		fillRect(dst, math.Round(x+math.Cos(ang)*r)-1, math.Round(y+math.Sin(ang)*r*0.6)-1, 3, 3, rgba(0xfff2a0, a))
	}
}

// stealthCulpritText: the panel line of the red alert with two players.
func (g *Game) stealthCulpritText() string {
	if g.sth.culprit == 1 {
		return "La alerta la provocó el J2"
	}
	return "La alerta la provocó el J1"
}

func init() {
	for es, en := range map[string]string{
		stealth2pMenuLabel: "Stealth 2 players (P1 decoy, P2 sneaks past)",
		"SIGILO 2J: uno hace de señuelo (%s / %s), el otro pasa": "STEALTH 2P: one is the decoy (%s / %s), the other sneaks past",
		"Quita la caja para hacer ruido (%s)":                    "Take the box off to make noise (%s)",
		"Aún no puedes volver a hacer ruido":                     "You cannot make noise again yet",
		"Acércate a un guardia o a un poli para hacer ruido":     "Get close to a guard or a cop to make noise",
		"¡Psst!":                                "Psst!",
		"¡Eh, por aquí!":                        "Hey, over here!",
		"*silbido*":                             "*whistle*",
		"La alerta la provocó el J1":            "Player 1 caused the alert",
		"La alerta la provocó el J2":            "Player 2 caused the alert",
		"%s: caja · %s o %s: ruido (señuelo)":   "%s: box · %s or %s: noise (decoy)",
		"%s: caja (puesta) · %s o %s: ruido":    "%s: box (on) · %s or %s: noise",
		"Fantasmas perfectos: nadie os detectó": "Perfect ghosts: nobody spotted you",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}

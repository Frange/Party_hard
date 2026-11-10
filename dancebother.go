package main

import (
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// Dancing near people. A guest only comes to hit the player when he INSISTS: every dance within
// danceNear of the same guest adds one point of "annoyance"; the points fade (danceFade per second)
// when the player stops. Only a minority of guests mind (danceTouchy), and even those reach for the fist
// only with a low chance per dance once the annoyance is danceLimit or more. After a reaction the player
// has danceCooldown seconds of peace. Own dice (danceDice): g.rng is not moved.
// ---------------------------------------------------------------------------

const (
	danceNear     = 60.0 // the guest must be this close to the dancer (and see him)
	danceLimit    = 4.0  // dances in a row next to him before he may react
	danceFade     = 0.25 // annoyance lost per second when the player is not bothering him
	danceTouchy   = 0.35 // share of guests who mind at all
	danceHitP     = 0.25 // chance per dance, past the limit, that a bothered guest comes to hit
	danceCooldown = 30.0 // seconds of peace after somebody came
)

var (
	danceRng  *rand.Rand
	danceSeed int64 // tests: non-zero = deterministic dice
)

func danceDice() *rand.Rand {
	if danceRng == nil {
		sd := danceSeed
		if sd == 0 {
			sd = time.Now().UnixNano()
		}
		danceRng = rand.New(rand.NewSource(sd))
	}
	return danceRng
}

// danceAnnoy runs once per dance of player p.
func (g *Game) danceAnnoy(p *Player, r *rand.Rand) {
	var hitter *Guest
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" {
			continue
		}
		near := dist(gu.pos, p.pos) <= danceNear && g.lvl.LOS(gu.pos, p.pos)
		if !near && gu.annoy > 0 { // the annoyance only fades while the player is not bothering him
			if gu.annoy -= (g.time - gu.annoyAt) * danceFade; gu.annoy < 0 {
				gu.annoy = 0
			}
		}
		gu.annoyAt = g.time
		if !near {
			continue
		}
		if gu.touchy == 0 {
			gu.touchy = 2
			if r.Float64() < danceTouchy {
				gu.touchy = 1
			}
		}
		if gu.touchy != 1 {
			continue
		}
		gu.annoy++
		if gu.annoy >= danceLimit && gu.state != gPunch && hitter == nil && g.time >= p.danceCool && r.Float64() < danceHitP {
			hitter = gu
		}
	}
	if hitter != nil {
		hitter.annoy = 0
		p.danceCool = g.time + danceCooldown
		hitter.icon, hitter.iconT = g.line(guestCategory(hitter), "dance_annoyed"), 2
		hitter.state, hitter.timer = gPunch, 12
	}
}

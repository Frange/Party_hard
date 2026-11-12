package main

import (
	"flag"
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// TREATS: the small rewards a player takes from the suitcase (pe_suitcase.go) and the street vendor (pe_vendor.go):
// a few seconds of speed, a party hat or a balloon. They are only for fun and never touch stealth, sight or points:
//   - the speed is a multiplier read by the player's movement (pevSpeedMul, one line in game.go), at most +30 % for
//     a few seconds;
//   - the hat and the balloon are drawings over the player.
// They belong to the party (the level) where they were given: a new level forgets them (pevTreatsFor).
// A reactive party event ("treats") draws them while any is alive: it does not roll dice, does not count for the
// three events of a party and ends by itself when the last treat runs out.
//
// The capture flags: `-pevent id:stage` (e.g. `-pevent suitcase:open`, `-pevent vendor:buy`) forces the event and
// lets it play its stage by itself near the first player (pevDemo); progress is frozen as with every debug flag.
// ---------------------------------------------------------------------------

const (
	treatSpeedK   = 1.3  // speed of the "churro turbo"
	treatSpeedDur = 8.0  // seconds
	treatHatDur   = 20.0 // seconds with the party hat / the balloon
)

type pevTreat struct {
	speedT, hatT, balloonT float64 // game time until which each one lasts
	col                    color.RGBA
}

var pevTreats struct {
	lvl *Level
	p   [2]pevTreat
}

// pevTreatsFor: the treats of player p in this party (a new level starts clean).
func (g *Game) pevTreatsFor(p *Player) *pevTreat {
	if pevTreats.lvl != g.lvl {
		pevTreats.lvl = g.lvl
		pevTreats.p = [2]pevTreat{}
	}
	i := 0
	if p != nil && p.idx == 1 {
		i = 1
	}
	return &pevTreats.p[i]
}

// pevSpeedMul: the movement multiplier of the treats (1 = none).
func (g *Game) pevSpeedMul(p *Player) float64 {
	if p == nil || pevTreats.lvl != g.lvl || g.lvl == nil {
		return 1
	}
	if g.time < g.pevTreatsFor(p).speedT {
		return treatSpeedK
	}
	return 1
}

// pevTreatsLive: some treat still runs (hat, balloon or speed) for some player.
func (g *Game) pevTreatsLive() bool {
	if pevTreats.lvl != g.lvl || g.lvl == nil {
		return false
	}
	for _, t := range pevTreats.p {
		if g.time < t.speedT || g.time < t.hatT || g.time < t.balloonT {
			return true
		}
	}
	return false
}

// pevDemo: the stage asked with `-pevent id:stage` for this event ("" = none).
func pevDemo(id string) string {
	if s, ok := strings.CutPrefix(dbgPartyEvent, id+":"); ok {
		return s
	}
	return ""
}

// pevDebugOn: the event was forced with -pevent (with or without a stage).
func pevDebugOn(id string) bool {
	return dbgPartyEvent == id || strings.HasPrefix(dbgPartyEvent, id+":")
}

// pevNearPlayerSpot: for the captures, an open spot 40-90 px from the first player (falls back to any open spot).
func (g *Game) pevNearPlayerSpot(rad float64, outdoor bool) (Vec, bool) {
	r := partyDice()
	for i := 0; i < 200; i++ {
		a := r.Float64() * 2 * math.Pi
		d := 40 + r.Float64()*50
		p := g.lvl.NearestFree(Vec{g.p.pos.X + math.Cos(a)*d, g.p.pos.Y + math.Sin(a)*d})
		if outdoor && !g.lvl.Outdoor(p) {
			continue
		}
		if dist(p, g.p.pos) > 30 && g.pevClear(p, rad) && g.lvl.PathG(g.p.pos, p) != nil {
			return p, true
		}
	}
	return g.pevSpot(r, rad, outdoor)
}

func init() {
	registerPartyEvent(&partyEvent{ID: "treats", Name: "Premios (gorro, globo, turbo)", Reactive: true, Dur: treatHatDur + 5,
		Eligible: func(g *Game, t partyTraits) bool { return g.pevTreatsLive() },
		Start:    func(g *Game, a *partyAct) bool { return true },
		Update:   func(g *Game, a *partyAct, dt float64) bool { return !g.pevTreatsLive() },
		Items:    treatItems})
}

var treatPartyCols = []color.RGBA{{0xff, 0x4a, 0x8a, 255}, {0x4a, 0xc8, 0xff, 255}, {0xff, 0xd8, 0x40, 255}, {0x8a, 0xe0, 0x5a, 255}}

// drawPartyHatPix: a striped party cone whose brim is centred on (cx, y).
func drawPartyHatPix(plot plotFn, cx, y, px int, col color.RGBA) {
	for r := 0; r < 7; r++ {
		w := 1 + r
		c := col
		if r%3 == 1 {
			c = pevWhite
		}
		plot(cx-(w/2)*px, y-(7-r)*px, w*px, px, c)
	}
	plot(cx-px, y-9*px, 2*px, 2*px, pevGold) // the pompom
	plot(cx-4*px, y, 8*px, px, pevInk)       // the brim's shadow
}

// drawBalloonPix: a balloon on a string; (x, y) is the hand, the balloon floats `up` pixels above.
func drawBalloonPix(plot plotFn, x, y, px, up int, t float64, col color.RGBA) {
	sway := int(math.Round(math.Sin(t*1.7) * 2))
	bx, by := x+sway*px+3*px, y-up*px
	for i := 0; i < up-5; i++ {
		k := float64(i) / float64(up)
		plot(x+int(math.Round(float64(sway+3)*k))*px, y-i*px, px, px, color.RGBA{0xe8, 0xe8, 0xe8, 255})
	}
	plot(bx-2*px, by-8*px, 5*px, 7*px, col)
	plot(bx-3*px, by-7*px, 7*px, 5*px, col)
	plot(bx-px, by-7*px, px, 2*px, pevWhite)
	plot(bx, by-px, px, px, col) // the knot
}

func treatItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for _, pl := range g.players() {
		if pl == nil || pl.dead || pl.hidden {
			continue
		}
		pl := pl
		t := g.pevTreatsFor(pl)
		if g.time < t.hatT {
			items = append(items, drawItem{pl.pos.Y + 0.5, func() {
				cx, cy := g.pevXY(pl.pos)
				drawPartyHatPix(plot, cx, cy-k(38), px, t.col)
			}})
		}
		if g.time < t.balloonT {
			items = append(items, drawItem{pl.pos.Y + 0.6, func() {
				cx, cy := g.pevXY(pl.pos)
				drawBalloonPix(plot, cx+k(6), cy-k(20), px, 26, g.time, t.col)
			}})
		}
		if g.time < t.speedT && int(g.time*10)%3 == 0 { // little speed lines behind him
			items = append(items, drawItem{pl.pos.Y - 0.1, func() {
				cx, cy := g.pevXY(pl.pos)
				for i := 0; i < 3; i++ {
					plot(cx-k(12)-i*2*px, cy-k(8)-i*k(7), 4*px, px, pevWhite)
				}
			}})
		}
	}
	return items
}

// treatSfx registers the small sounds of these events the first time (squeak of the rubber duck, buzz of the drone).
func (g *Game) treatSfx() {
	if g.snd == nil || g.snd.sfx == nil {
		return
	}
	if _, ok := g.snd.sfx["squeak"]; ok {
		return
	}
	g.snd.sfx["squeak"] = mix(tone(1250, 0.12, 0.22, 10, 0), shift(tone(1650, 0.14, 0.2, 12, 0), 0.07))
	g.snd.sfx["buzz"] = mix(lowpass(tone(118, 1.1, 0.10, 1.2, 2), 6), lowpass(tone(236, 1.1, 0.05, 1.2, 1), 8))
}

// pevDebugLog: with -pevent, where the event happens (world and screen pixels, to aim -shotzoom). Only in debug runs
// (and in the tests, which force the events the same way).
func pevDebugLog(g *Game, what string, p Vec) {
	if flag.Lookup("test.v") != nil { // quiet in the tests
		return
	}
	x, y := g.pevXY(p)
	fmt.Printf("pevent %s: world %.0f,%.0f screen %d,%d\n", what, p.X, p.Y, x, y)
	// -shotzoom -1,-1,w,h,z: the crop is aimed at the event (its place is only known once it starts)
	if shotZoom > 1 && shotCrop[0] < 0 {
		w, h := shotCrop[2], shotCrop[3]
		shotCrop[0] = maxI(0, minI(screenW-w, x-w/2))
		shotCrop[1] = maxI(0, minI(screenH-h, y-h*3/5))
	}
}

package main

import (
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "APAGON": a general power cut in the party. VERY rare.
//
//   - chance per level: blackoutChance (about 1 party in 16). Never in the STEALTH mode. The decision is taken when the
//     level loads with its own random source (blackoutPlan), so it does not move the dice of the rest of the party.
//   - it lasts blackoutMin..blackoutMax seconds (8-12) and starts blackoutAtMin..blackoutAtMax seconds into the party.
//   - the screen goes dark with a vignette; light comes only from the player's halo (always visible, blackoutHaloR),
//     the torches of guards and officers, the phones of some guests, the eyes of the cats and the sparks of the fuse
//     box. Guests panic: some scream and run at random.
//   - WHILE IT LASTS: the security cameras do not work (no cone, no LED, no recording) and everybody's sight is
//     HALVED (blackoutVisK: guests, guards and officers see half as far, see g.sees).
//   - when the light comes back there is a "clac" and a funny message.
//
// Force it with the flag -blackout (debug: freezes the progress, see debugFlagsActive).
// ---------------------------------------------------------------------------

const (
	blackoutChance  = 1.0 / 16 // chance that a party (not stealth) has a blackout
	blackoutMin     = 8.0      // seconds of darkness, at least
	blackoutMax     = 12.0     // ... and at most
	blackoutAtMin   = 25.0     // earliest second of the party it can happen
	blackoutAtMax   = 120.0    // latest
	blackoutVisK    = 0.5      // fraction of the usual sight range while it lasts
	blackoutHaloR   = 62.0     // radius of the minimum light around each player (never fully dark for him)
	blackoutFadeIn  = 0.5      // seconds that the light takes to go out
	blackoutFadeOut = 0.35     // seconds that the light takes to come back
	blackoutDark    = 236      // alpha of the darkness (0..255)
	blackoutScream  = 3        // guests that shout (the bubbles stay few)
	blackoutPanicP  = 0.4      // fraction of the free guests that run at random
	blackoutDemoAt  = 3.0      // second at which -blackout forces it
)

// blackoutState is the event of the party in course.
type blackoutState struct {
	planned bool    // this party will have a blackout
	at      float64 // second of the party at which it starts
	dur     float64 // seconds it lasts
	t       float64 // seconds of darkness left (> 0 = active)
	started bool
	done    bool
	spark   float64 // seconds of the next spark flash
	flash   float64 // seconds left of the visible spark
	panel   Vec     // the fuse box
}

// dbgBlackout: -blackout forces the event in every party (debug).
var dbgBlackout bool

// blackoutPlan decides, with the given dice, whether a party has a blackout and when. Stealth never has one.
func blackoutPlan(r *rand.Rand, stealth bool) (blackoutState, bool) {
	if stealth || r.Float64() >= blackoutChance {
		return blackoutState{}, false
	}
	return blackoutState{
		planned: true,
		at:      blackoutAtMin + r.Float64()*(blackoutAtMax-blackoutAtMin),
		dur:     blackoutMin + r.Float64()*(blackoutMax-blackoutMin),
	}, true
}

// blackoutLevelReset is called when a level loads.
func (g *Game) blackoutLevelReset() {
	g.blk = blackoutState{}
	if g.stealth {
		return
	}
	if dbgBlackout {
		g.blk = blackoutState{planned: true, at: blackoutDemoAt, dur: 10}
		return
	}
	if b, ok := blackoutPlan(rand.New(rand.NewSource(time.Now().UnixNano())), g.stealth); ok {
		g.blk = b
	}
}

// blackoutOn: the power is out right now.
func (g *Game) blackoutOn() bool { return g.blk.t > 0 }

// blackoutK: how dark it is (0 = light, 1 = full darkness), with the fades.
func (g *Game) blackoutK() float64 {
	if g.blk.t <= 0 {
		return 0
	}
	in := (g.blk.dur - g.blk.t) / blackoutFadeIn
	out := g.blk.t / blackoutFadeOut
	return clampF(math.Min(in, out), 0, 1)
}

// blackoutSight: the factor that sight ranges get now (1 normally).
func (g *Game) blackoutSight() float64 {
	if g.blackoutOn() {
		return blackoutVisK
	}
	return 1
}

func (g *Game) updateBlackout(dt float64) {
	b := &g.blk
	if !b.planned || b.done || g.stealth || g.lvl == nil {
		return
	}
	if !b.started {
		if g.mode != ModePlay || g.time < b.at {
			return
		}
		g.startBlackout()
	}
	b.t -= dt
	if b.flash > 0 {
		b.flash -= dt
	}
	b.spark -= dt
	if b.spark <= 0 && b.t > 0 {
		b.spark = 0.5 + g.rng.Float64()*1.4
		b.flash = 0.12
		g.burst(b.panel, 6, 70, rgb(0xffe27a), 0.35, 2, 0)
		g.snd.Play("spark")
	}
	if b.t <= 0 {
		b.t, b.done, b.flash = 0, true, 0
		g.snd.Play("clac")
		g.say(blackoutBack[g.rng.Intn(len(blackoutBack))])
	}
}

var blackoutOut = []string{
	"¡Se ha ido la luz! Seguro que ha sido el DJ con la tostadora.",
	"¡Apagón! Alguien ha enchufado el horno y la rave a la vez.",
}
var blackoutBack = []string{
	"¡Clac! Vuelve la luz. Nadie sabe quién estaba abrazando a quién.",
	"¡Clac! Luz otra vez. Y alguien tiene el sombrero de otro.",
	"¡Clac! Ya hay luz. El fusible pide perdón.",
}
var blackoutScreams = []string{"¡AAAH!", "¡LA LUZ!", "¡MAMÁ!", "¿DÓNDE ESTOY?"}

func (g *Game) startBlackout() {
	b := &g.blk
	b.started, b.t, b.spark = true, b.dur, 0.2
	b.panel = g.lvl.Def.Start
	if len(g.lvl.Lights) > 0 {
		b.panel = g.lvl.Lights[0]
	}
	g.say(blackoutOut[g.rng.Intn(len(blackoutOut))])
	g.snd.Play("clac")
	n := 0
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" {
			continue
		}
		switch gu.state {
		case gIdle, gGoto, gDance, gDrink:
		default:
			continue
		}
		if n < blackoutScream && g.rng.Float64() < 0.2 {
			gu.icon, gu.iconT = blackoutScreams[g.rng.Intn(len(blackoutScreams))], 2.5
			n++
		}
		if g.rng.Float64() < blackoutPanicP {
			gu.state, gu.timer, gu.path = gRun, 1+g.rng.Float64()*2, nil
			gu.runUntil = g.time + b.dur
		}
	}
}

// drawBlackout paints the darkness over the world (below the HUD) and the little lights that remain.
func (g *Game) drawBlackout(dst *ebiten.Image) {
	k := g.blackoutK()
	if k <= 0 || g.lvl == nil {
		return
	}
	a := uint8(float64(blackoutDark) * k)
	if g.as == nil || g.as.dark == nil || g.as.glow == nil {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x00000a, a))
		return
	}
	dark := g.as.dark
	dark.Fill(rgba(0x00000a, a))
	// the minimum light of each player: never dark enough to lose him
	for _, p := range g.players() {
		if p != nil && !p.dead {
			g.cutAt(dark, p.pos, blackoutHaloR*1.3, 1)
		}
	}
	for _, gd := range g.guards { // torches
		if !gd.dead {
			g.cutAt(dark, gd.pos, 78, 0.9)
		}
	}
	for _, c := range g.cops {
		if c.state != cLeave {
			g.cutAt(dark, c.pos, 84, 0.95)
		}
	}
	for i, gu := range g.guests { // phones of some guests
		if i%4 == 0 && !gu.out() {
			g.cutAt(dark, gu.pos, 20, 0.7)
		}
	}
	if g.blk.flash > 0 { // the spark of the fuse box
		g.cutAt(dark, g.blk.panel, 46, 1)
	}
	dst.DrawImage(dark, nil)
	// the eyes of the cats and the screens of the phones, on top of the darkness
	for _, an := range g.animals {
		if an != nil && an.kind == animCat {
			x, y := g.wx(an.pos.X), g.wy(an.pos.Y-8)
			fillRect(dst, x-3, y, 2, 2, rgba(0xa6ff5a, uint8(255*k)))
			fillRect(dst, x+2, y, 2, 2, rgba(0xa6ff5a, uint8(255*k)))
		}
	}
	for i, gu := range g.guests {
		if i%4 == 0 && !gu.out() {
			fillRect(dst, g.wx(gu.pos.X)+4, g.wy(gu.pos.Y-16), 3, 4, rgba(0xbfe4ff, uint8(255*k)))
		}
	}
}

package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// EVENT "fireworks": when a party in the OPEN AIR is completed with the players alive, bursts of rockets rise over the
// party and explode in coloured sparks for about four seconds, over the (night-tinted) scene, right before the
// newspaper. A press (confirm / back / click) skips them, so they never hold the flow. 55 % of the outdoor parties.
//
// Cheap: a fixed table of rockets and sparks (no allocations while it plays), drawn into ONE half-resolution
// RGBA buffer (480x270, premultiplied, uploaded with a single WritePixels and drawn at x2: chunky pixel art).
// Own dice (partyDice); g.rng is never moved. Sound: the existing "spark" at each launch (nothing else).
// Arm: completeLevel -> fireworksArm (game.go). Play: Update of ModeLevelDone -> fireworksUpdate, Draw -> fireworksDraw.
// `-pevent fireworks -level N` completes the party 3 s after the start to see them (reactive entry below).
// ---------------------------------------------------------------------------

const (
	fwW, fwH   = screenW / 2, screenH / 2
	fwMaxP     = 520
	fwMaxR     = 6
	fwChance   = 0.55
	fwShowTime = 3.4 // seconds in which rockets are launched
	fwGrav     = 55.0
)

var fwPalette = []color.RGBA{{0xff, 0x4a, 0x5a, 255}, {0xff, 0xd0, 0x40, 255}, {0x5a, 0xe0, 0xff, 255}, {0x7a, 0xff, 0x6a, 255},
	{0xff, 0x7a, 0xe0, 255}, {0xff, 0x9a, 0x30, 255}, {0xb0, 0x8a, 0xff, 255}, {0xf8, 0xf4, 0xea, 255}}

type fwPart struct {
	x, y, vx, vy float64
	life, max    float64
	col          uint8
	on           bool
}

type fwRocket struct {
	x, y, vx, vy, apex float64
	col                uint8
	kind               uint8
	on                 bool
}

type fwState struct {
	armed, on, shown bool
	endT             float64 // g.endTime of the party that armed it
	t, next          float64
	launched         int
	rng              *rand.Rand
	rk               [fwMaxR]fwRocket
	ps               [fwMaxP]fwPart
	buf              []byte
	img              *ebiten.Image
	op               ebiten.DrawImageOptions
}

func init() {
	registerPartyEvent(&partyEvent{ID: "fireworks", Name: "Fuegos artificiales (al cerrar una fiesta exterior, 55 %)", Reactive: true, Prob: fwChance,
		Eligible: func(g *Game, t partyTraits) bool {
			return dbgPartyEvent == "fireworks" && g.time >= 3 && !g.pev.ran["fireworks"]
		},
		Start: func(g *Game, a *partyAct) bool { // debug only: finish the party now to watch them
			g.completeLevel() // (it arms the show: dbgPartyEvent == "fireworks" forces it)
			return true
		},
		Update: func(g *Game, a *partyAct, dt float64) bool { return true }})
}

// fireworksOK: this party may end with fireworks (open air, alive, not stealth / island / the last parties' endings).
func (g *Game) fireworksOK() bool {
	if g.lvl == nil || g.lvl.Def == nil || g.stealth || g.isIsland() || g.levelIdx < 0 {
		return false
	}
	for _, p := range g.players() {
		if p != nil && p.dead {
			return false
		}
	}
	return partyTraitsOf(g.lvl.Def, g.levelIdx).outdoor
}

// fireworksArm: called by completeLevel (after endTime is set). Rolls the 55 % with the own dice.
func (g *Game) fireworksArm(force bool) {
	f := &g.fwx
	f.armed, f.on, f.shown = false, false, false
	force = force || dbgPartyEvent == "fireworks"
	if !force && (!g.fireworksOK() || partyDice().Float64() >= fwChance) {
		return
	}
	f.armed, f.endT = true, g.endTime
}

// fireworksStart: begins the show (own dice from the party dice, so the same session keeps its own order).
func (g *Game) fireworksStart(seed int64) {
	f := &g.fwx
	f.on, f.shown, f.armed = true, true, false
	f.t, f.next, f.launched = 0, 0.15, 0
	f.rng = rand.New(rand.NewSource(seed))
	f.rk = [fwMaxR]fwRocket{}
	f.ps = [fwMaxP]fwPart{}
	if f.buf == nil {
		f.buf = make([]byte, fwW*fwH*4)
	}
}

// fireworksUpdate: true = the show is on (the press that would continue only skips it).
func (g *Game) fireworksUpdate(dt float64) bool {
	f := &g.fwx
	if !f.on {
		if f.armed && f.endT == g.endTime && g.mode == ModeLevelDone && !g.stealth {
			g.fireworksStart(partyDice().Int63())
		} else {
			return false
		}
	}
	if actConfirm.just() || actBack.just() || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		f.on = false
		return true // the paper shows from the next frame; this press was only the skip
	}
	if fwStep(f, dt, g.snd) {
		f.on = false
	}
	return f.on
}

// fwStep advances the show by dt. true = it is over (time is up and nothing is left in the air).
func fwStep(f *fwState, dt float64, snd *Sounds) bool {
	f.t += dt
	r := f.rng
	// launches
	if f.t < fwShowTime && f.t >= f.next {
		f.next = f.t + 0.22 + r.Float64()*0.5
		n := 1
		if f.launched > 2 && r.Float64() < 0.35 {
			n = 2
		}
		for i := 0; i < n; i++ {
			for k := range f.rk {
				if !f.rk[k].on {
					x := 50 + r.Float64()*(fwW-100)
					f.rk[k] = fwRocket{x: x, y: fwH - 6, vx: (r.Float64() - 0.5) * 14, vy: -(150 + r.Float64()*40), apex: 36 + r.Float64()*80,
						col: uint8(r.Intn(len(fwPalette))), kind: uint8(r.Intn(3)), on: true}
					f.launched++
					if snd != nil && f.launched%2 == 1 {
						snd.Play("spark")
					}
					break
				}
			}
		}
	}
	busy := false
	for k := range f.rk {
		rk := &f.rk[k]
		if !rk.on {
			continue
		}
		busy = true
		rk.x += rk.vx * dt
		rk.y += rk.vy * dt
		rk.vy += 40 * dt
		fwEmit(f, rk.x, rk.y+2, (r.Float64()-0.5)*8, 20+r.Float64()*14, 0.35, 7) // the tail: white-gold sparks
		if rk.y <= rk.apex || rk.vy > -30 {
			fwBurst(f, rk)
			rk.on = false
		}
	}
	for i := range f.ps {
		p := &f.ps[i]
		if !p.on {
			continue
		}
		busy = true
		p.life -= dt
		if p.life <= 0 {
			p.on = false
			continue
		}
		p.vy += fwGrav * dt
		p.vx *= 1 - 0.6*dt
		p.x += p.vx * dt
		p.y += p.vy * dt
	}
	return f.t >= fwShowTime && !busy
}

func fwEmit(f *fwState, x, y, vx, vy, life float64, col uint8) {
	for i := range f.ps {
		if !f.ps[i].on {
			f.ps[i] = fwPart{x: x, y: y, vx: vx, vy: vy, life: life, max: life, col: col, on: true}
			return
		}
	}
}

// fwBurst: the explosion: a sphere, a ring or a willow (slow and long) in one or two colours.
func fwBurst(f *fwState, rk *fwRocket) {
	r := f.rng
	n := 34 + r.Intn(14)
	c2 := uint8((int(rk.col) + 1 + r.Intn(3)) % len(fwPalette))
	for i := 0; i < n; i++ {
		a := float64(i) / float64(n) * 2 * math.Pi
		sp := 38 + r.Float64()*34
		life := 1.0 + r.Float64()*0.5
		switch rk.kind {
		case 1: // ring
			sp = 58
		case 2: // willow
			sp *= 0.75
			life += 0.8
		}
		col := rk.col
		if i%3 == 0 {
			col = c2
		}
		fwEmit(f, rk.x, rk.y, math.Cos(a)*sp, math.Sin(a)*sp, life, col)
	}
}

// fwRaster paints the show into the half-resolution buffer (premultiplied RGBA; the tint of the night first).
func fwRaster(f *fwState, buf []byte) {
	for i := 0; i < len(buf); i += 4 {
		buf[i], buf[i+1], buf[i+2], buf[i+3] = 0, 0, 0, 120
	}
	plot := func(x, y, w, h int, c color.RGBA, a float64) {
		cr, cg, cb, ca := uint8(float64(c.R)*a), uint8(float64(c.G)*a), uint8(float64(c.B)*a), uint8(255*a)
		for yy := y; yy < y+h; yy++ {
			if yy < 0 || yy >= fwH {
				continue
			}
			for xx := x; xx < x+w; xx++ {
				if xx < 0 || xx >= fwW {
					continue
				}
				o := (yy*fwW + xx) * 4
				if cr > buf[o] {
					buf[o] = cr
				}
				if cg > buf[o+1] {
					buf[o+1] = cg
				}
				if cb > buf[o+2] {
					buf[o+2] = cb
				}
				if ca > buf[o+3] {
					buf[o+3] = ca
				}
			}
		}
	}
	for i := range f.ps {
		p := &f.ps[i]
		if !p.on {
			continue
		}
		a := math.Min(1, p.life/p.max*1.6)
		sz := 2
		if p.life/p.max < 0.3 {
			sz = 1
		}
		plot(int(p.x), int(p.y), sz, sz, fwPalette[p.col], a)
	}
	for k := range f.rk {
		if rk := &f.rk[k]; rk.on {
			plot(int(rk.x), int(rk.y), 2, 3, fwPalette[7], 1)
		}
	}
}

// fireworksDraw draws the show over the scene (nothing is allocated here). false = not showing.
func (g *Game) fireworksDraw(screen *ebiten.Image) bool {
	f := &g.fwx
	if !f.on {
		return false
	}
	if f.img == nil {
		f.img = ebiten.NewImage(fwW, fwH)
	}
	fwRaster(f, f.buf)
	f.img.WritePixels(f.buf)
	f.op.GeoM.Reset()
	f.op.GeoM.Scale(2, 2)
	screen.DrawImage(f.img, &f.op)
	return true
}

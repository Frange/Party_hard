package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Fire. The arsonist (pyro.go) sets a prop alight; the flames spread a little to
// the flammable things around it, but never far: every ignition has its own
// budget (a few props, within a short radius of where it started) and the whole
// level has a cap on flames at once. Metal, stone, water and anything standing in
// a pool do not burn. A fire dies by itself after a while (the prop stays charred),
// or the firefighters (firefighters.go) put it out with their hoses.
// People next to the flames panic, and the ones who stay too long die: those
// deaths are nobody's work (no stars, no points) but they count as bodies.
// ---------------------------------------------------------------------------

const (
	maxFires        = 8  // flames alive in the whole level
	fireSpreadMax   = 3  // props one ignition may pass the fire to
	fireSpreadReach = 46 // how far (px) a flame jumps to the next prop
	fireOriginReach = 88 // the farthest from where it all started that anything may catch
	fireLifeMin     = 24 // seconds a fire lasts when left alone
	fireLifeMax     = 36
)

// flammable lists the props that burn (wood, cloth, plants, straw). Anything else —
// cars, fridges, statues, toilets, the DJ booth, water — never does.
var flammable = map[string]bool{
	"sofa": true, "couch": true, "armchair": true, "bed": true, "table": true, "cafetable": true, "longtable": true,
	"stool": true, "bench": true, "bookshelf": true, "closet": true, "shelf": true, "posters": true, "plant": true,
	"planter": true, "bush": true, "hedge": true, "palm": true, "tree": true, "haystack": true, "barrel": true,
	"crate": true, "umbrella": true, "sunbed": true, "tent": true, "bungalow": true, "minibar": true, "stall": true,
	"cart": true, "buffet": true, "counter": true, "bar": true, "rope": true, "cage": true, "stands": true,
}

// isFlammable: the prop can catch fire (and is not already burnt or on fire).
func (g *Game) isFlammable(pr *Prop) bool {
	if pr == nil || pr.gone || pr.wreck || pr.burn > 0 || pr.lit || !flammable[pr.Name] {
		return false
	}
	return !g.lvl.InPool(Vec{pr.X, pr.Y})
}

type fireOrigin struct {
	pos    Vec
	spread int
}

// Fire is one burning prop (or one burning spot).
type Fire struct {
	pos     Vec
	prop    *Prop
	origin  *fireOrigin
	age     float64
	life    float64
	spreadT float64
	w       float64 // width of the flames
	smokeT  float64
	seed    float64
	out     bool
	crackT  float64
}

// fireWorld is everything about fire and the arsonist in the current party.
type fireWorld struct {
	fires    []*Fire
	exposure map[*Guest]float64
	pBurn    float64
	pBurn2   float64 // the second player
	pyroAt   float64 // time the arsonist walks in (0 = none this party)
	pyroDone bool
	lit      int // ignitions so far
	deaths   int
	called   bool
}

func (f *Fire) intensity() float64 {
	if f.out {
		return 0
	}
	in := math.Min(1, f.age/3)
	if rest := f.life - f.age; rest < 4 {
		in *= math.Max(0.15, rest/4)
	}
	return in
}

// putOut: the flames end (the firefighters, or the fire ran out).
func (f *Fire) putOut(g *Game, byHose bool) {
	if f.out {
		return
	}
	f.out = true
	if f.prop != nil {
		f.prop.lit = false
		if byHose {
			f.prop.burn = math.Max(f.prop.burn, 0.5)
		} else {
			f.prop.burn = 1
		}
		if !byHose {
			g.decals = append(g.decals, Decal{pos: Vec{f.pos.X, f.pos.Y + 1}, kind: 1, r: 5 + f.w/6})
		}
	}
	g.burst(Vec{f.pos.X, f.pos.Y - 10}, 10, 36, rgb(0x9a9aa4), 0.9, 3, -26)
}

// igniteProp sets a prop alight. origin is the ignition it belongs to (nil starts a new one).
func (g *Game) igniteProp(pr *Prop, origin *fireOrigin) *Fire {
	if !g.isFlammable(pr) || g.liveFires() >= maxFires {
		return nil
	}
	if origin == nil {
		origin = &fireOrigin{pos: Vec{pr.X, pr.Y}}
	}
	w := 14.0
	if pr.K != nil {
		w = math.Max(14, pr.K.FW*0.7)
	}
	f := &Fire{pos: Vec{pr.X, pr.Y}, prop: pr, origin: origin, life: fireLifeMin + g.rng.Float64()*(fireLifeMax-fireLifeMin), w: math.Min(w, 90), seed: g.rng.Float64() * 10, spreadT: 2.5 + g.rng.Float64()*2}
	pr.lit = true
	g.blaze.fires = append(g.blaze.fires, f)
	g.fireSfx()
	g.snd.Play("ignite")
	g.burst(Vec{f.pos.X, f.pos.Y - 8}, 12, 50, rgb(0xffb030), 0.5, 2, 120)
	return f
}

func (g *Game) liveFires() int {
	n := 0
	for _, f := range g.blaze.fires {
		if !f.out {
			n++
		}
	}
	return n
}

// fireSfx registers the fire sounds the first time they are needed.
func (g *Game) fireSfx() {
	if g.snd == nil || g.snd.sfx == nil {
		return
	}
	if _, ok := g.snd.sfx["ignite"]; ok {
		return
	}
	g.snd.sfx["ignite"] = mix(noise(0.45, 0.55, 5), tone(260, 0.35, 0.3, 6, 0), shift(tone(420, 0.25, 0.25, 8, 0), 0.08))
	g.snd.sfx["crackle"] = toPCM(noise(0.06, 0.3, 30))
	g.snd.sfx["whoosh"] = mix(lowpass(noise(0.5, 0.5, 4), 3), tone(140, 0.4, 0.3, 5, 0))
}

// nearestFlammable: the closest prop that could catch from a flame at p, inside the origin's reach.
func (g *Game) nearestFlammable(p Vec, o *fireOrigin) *Prop {
	var best *Prop
	bd := fireSpreadReach + 1.0
	for _, pr := range g.lvl.Props {
		if !g.isFlammable(pr) {
			continue
		}
		pp := Vec{pr.X, pr.Y}
		if d := dist(p, pp); d < bd && dist(o.pos, pp) <= fireOriginReach {
			best, bd = pr, d
		}
	}
	return best
}

func (g *Game) updateFires(dt float64) {
	if g.lvl == nil || g.mode != ModePlay {
		return
	}
	if g.blaze.exposure == nil {
		g.blaze.exposure = map[*Guest]float64{}
	}
	live := g.blaze.fires[:0]
	for _, f := range g.blaze.fires {
		if f.out {
			continue // the charred prop stays (Prop.burn); the fire itself is forgotten
		}
		f.age += dt
		if f.prop != nil {
			f.prop.burn = math.Max(f.prop.burn, math.Min(1, f.age/7))
		}
		if f.age >= f.life {
			f.putOut(g, false)
			continue
		}
		in := f.intensity()
		// smoke and sparks
		f.smokeT -= dt
		if f.smokeT <= 0 && len(g.parts) < 900 {
			f.smokeT = 0.12 + g.rng.Float64()*0.1
			x := f.pos.X + (g.rng.Float64()-0.5)*f.w
			g.parts = append(g.parts, Particle{pos: Vec{x, f.pos.Y - 10}, vel: Vec{(g.rng.Float64() - 0.5) * 10, -22 - g.rng.Float64()*14}, life: 1.6, max: 1.6, col: rgb(0x70707c), size: 3, grav: -6})
			if g.rng.Float64() < 0.5*in {
				g.parts = append(g.parts, Particle{pos: Vec{x, f.pos.Y - 12}, vel: Vec{(g.rng.Float64() - 0.5) * 30, -40 - g.rng.Float64()*30}, life: 0.7, max: 0.7, col: rgb(0xffa030), size: 1, grav: 60})
			}
		}
		f.crackT -= dt
		if f.crackT <= 0 {
			f.crackT = 0.5 + g.rng.Float64()*1.2
			if in > 0.5 && dist(g.p.pos, f.pos) < 220 {
				g.snd.Play("crackle")
			}
		}
		// it jumps to a flammable neighbour now and then, up to the budget of its ignition
		f.spreadT -= dt
		if f.spreadT <= 0 {
			f.spreadT = 2.5 + g.rng.Float64()*2.5
			if in > 0.6 && f.origin.spread < fireSpreadMax && g.rng.Float64() < 0.5 {
				if pr := g.nearestFlammable(f.pos, f.origin); pr != nil {
					if nf := g.igniteProp(pr, f.origin); nf != nil {
						f.origin.spread++
					}
				}
			}
		}
		g.fireHurt(f, in, dt)
		live = append(live, f)
	}
	g.blaze.fires = live
	// somebody calls the firefighters a few seconds after the first flames
	if g.liveFires() > 0 && !g.blaze.called && g.fireTruck == nil {
		for _, f := range g.blaze.fires {
			if !f.out && f.age > 4 {
				g.blaze.called = true
				g.say("¡Alguien ha llamado a los bomberos!")
				g.sayNews("fire_call", nil)
				break
			}
		}
	}
	if g.liveFires() == 0 && len(g.fireCrew) == 0 {
		g.blaze.called = false
	}
}

// fireHurt: people next to the flames panic and, if they stay, die.
func (g *Game) fireHurt(f *Fire, in, dt float64) {
	if in < 0.3 {
		return
	}
	r := math.Max(14, f.w/2+9)
	at := Vec{f.pos.X, f.pos.Y - 3}
	near := func(p Vec) bool {
		return math.Abs(p.X-at.X) < r && math.Abs(p.Y-at.Y) < r*0.7
	}
	for _, gu := range g.guests {
		if gu.out() || gu.py != nil || gu.state == gLift || !near(gu.pos) {
			continue
		}
		e := g.blaze.exposure[gu]
		if e == 0 {
			g.scare(gu)
			gu.icon, gu.iconT = g.freshPhrase("firecry", fireCries, g.rng.Intn), 1.4
		}
		e += dt
		g.blaze.exposure[gu] = e
		if e > 1.8 && g.rng.Float64() < dt*0.45 {
			g.blaze.deaths++
			g.sayNews("fire_death", map[string]string{"victim": guestLabel(gu)})
			g.killGuest(gu, false)
			g.decals = append(g.decals, Decal{pos: Vec{gu.pos.X, gu.pos.Y + 1}, kind: 1, r: 6})
			g.burst(Vec{gu.pos.X, gu.pos.Y - 12}, 12, 40, rgb(0x50505a), 1, 3, -30)
		} else if e > 1.0 && gu.state != gScared && gu.state != gFlee {
			g.scare(gu)
		}
	}
	// the players (each one burns on his own)
	for _, pl := range g.players() {
		pl := pl
		g.withPlayer(pl, func() {
			p := pl
			burn := &g.blaze.pBurn
			if p.idx == 1 {
				burn = &g.blaze.pBurn2
			}
			if !p.dead && !p.hidden && near(p.pos) && !g.lvl.InPool(p.pos) {
				*burn += dt
				if *burn > 0.6 && p.knockT <= 0 {
					p.knockT = 0.35
					g.burst(Vec{p.pos.X, p.pos.Y - 14}, 8, 50, rgb(0xffa030), 0.5, 2, 80)
				}
				if *burn > 1.7 && !g.invulnerable() {
					if g.is2P() {
						g.die("Te has achicharrado")
					} else {
						g.lose("Te has achicharrado")
					}
				}
			} else if *burn > 0 {
				*burn = math.Max(0, *burn-dt*1.5)
			}
		})
	}
}

var fireCries = []string{"¡FUEGO!", "¡ME QUEMO!", "¡AAAH!", "¡AGUA!", "¡QUE ARDE!", "¡SOCORRO!"}

// ---- drawing -------------------------------------------------------------------

var flameFrames [6]*ebiten.Image

// buildFlames paints six frames of a small pixel-art flame: red at the foot, orange in the middle,
// yellow at the tips, a column per pixel with a different height each frame.
func buildFlames() {
	if flameFrames[0] != nil {
		return
	}
	const w, h = 14, 22
	for f := range flameFrames {
		b := newPix(w, h)
		for x := 0; x < w; x++ {
			cx := float64(x) - float64(w-1)/2
			base := 1 - (cx*cx)/((float64(w)/2)*(float64(w)/2)) // tall in the middle
			wob := math.Sin(float64(x)*1.7+float64(f)*1.3)*2.2 + math.Sin(float64(x)*0.8-float64(f)*2.1)*1.4
			top := (float64(h) - 3) * (0.35 + 0.65*base)
			top += wob
			if top > float64(h)-1 {
				top = float64(h) - 1
			}
			for y := 0; y < int(top); y++ {
				yy := h - 1 - y
				t := float64(y) / top // 0 foot .. 1 tip
				var col color.RGBA
				switch {
				case t < 0.3:
					col = color.RGBA{0xd8, 0x28, 0x14, 255}
				case t < 0.62:
					col = color.RGBA{0xf4, 0x84, 0x1c, 255}
				case t < 0.85:
					col = color.RGBA{0xff, 0xc8, 0x3c, 255}
				default:
					col = color.RGBA{0xff, 0xf0, 0x90, 255}
				}
				// a hot core in the middle of the base
				if t < 0.45 && math.Abs(cx) < 2 {
					col = color.RGBA{0xff, 0xe0, 0x70, 255}
				}
				b.set(x, yy, col)
			}
		}
		flameFrames[f] = b.img()
	}
}

// drawFire paints the flames of one fire and the glow it throws on the floor.
func (g *Game) drawFire(dst *ebiten.Image, f *Fire) {
	in := f.intensity()
	if in <= 0 {
		return
	}
	buildFlames()
	n := int(math.Max(1, math.Round(f.w/12)))
	for i := 0; i < n; i++ {
		fr := (int(g.time*9+f.seed*3) + i*2) % len(flameFrames)
		img := flameFrames[fr]
		sc := in * (0.75 + 0.35*math.Sin(g.time*7+f.seed+float64(i)*1.9))
		if sc < 0.2 {
			continue
		}
		x := g.wx(f.pos.X) + (float64(i)-float64(n-1)/2)*11
		y := g.wy(f.pos.Y) - 1
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-7, -22)
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(math.Round(x), math.Round(y))
		dst.DrawImage(img, op)
	}
	flick := 0.8 + 0.2*math.Sin(g.time*13+f.seed*5)
	g.glowAt(dst, Vec{f.pos.X, f.pos.Y - 8}, 26+f.w*0.4, color.RGBA{255, 140, 40, uint8(95 * in * flick)})
}

// fireItems adds the burning things to the draw list (sorted with everything else).
func (g *Game) fireItems(dst *ebiten.Image, items []drawItem) []drawItem {
	for _, f := range g.blaze.fires {
		f := f
		if f.out {
			continue
		}
		items = append(items, drawItem{f.pos.Y + 1, func() { g.drawFire(dst, f) }})
	}
	return items
}

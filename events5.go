package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// More rare visitors, none of them ever raising the criminality:
//   F1        a famous driver tears through the party in a tiny F1 car
//   stampede  a herd that escaped from somewhere runs across the level
//   parachute a skydiver drops in and joins the party
//   sport     some guests start a ball game among themselves
//   theme     a group in Halloween / Christmas / summer clothes arrives
// ---------------------------------------------------------------------------

type herdRunner struct {
	kind, coat int
	x, y       float64
	ph         float64
}

// clearStrip finds a horizontal band (centre y, half height) that crosses the
// whole level with nothing solid in it, so a runner never goes through a wall
// or a table. ok is false when the level has none.
func (g *Game) clearStrip(half float64) (float64, bool) {
	l := g.lvl
	clear := func(y float64) bool {
		for x := 70.0; x <= l.W-70; x += 8 { // the outer fence is where they come in
			for _, dy := range []float64{-half, 0, half} {
				if !l.Free(Vec{x, y + dy}) {
					return false
				}
			}
		}
		return true
	}
	for try := 0; try < 60; try++ {
		y := 90 + g.rng.Float64()*(l.H-150)
		if clear(y) {
			return y, true
		}
	}
	// the street, when the level has one
	d := l.Def
	if d.Police == "car" && d.Transit == "" && d.CarFrom.Y > 40 && d.CarFrom.Y < l.H-10 {
		for _, off := range []float64{d.CarFrom.Y + g.laneOff(), d.CarFrom.Y - g.laneOff()} {
			if clear(off) {
				return off, true
			}
		}
	}
	return 0, false
}

// ---- F1 --------------------------------------------------------------------

var f1Drivers = []struct {
	name, shout          string
	body, accent, helmet uint32
}{
	{"Fernando Alonso", "¡GP2 ENGINE!", 0x0a7a62, 0xd8e830, 0x2a50c0},
	{"Lewis Hamilton", "¡STILL WE RISE!", 0xd01818, 0xf4f4f4, 0xf0d020},
	{"Max Verstappen", "¡SIMPLY LOVELY!", 0x1c2a5c, 0xe03030, 0xf08020},
	{"Charles Leclerc", "¡SOY TONTO, SOY TONTO!", 0xdc1010, 0xf0d030, 0xe0e0e8},
	{"Carlos Sainz", "¡SMOOTH OPERATOR!", 0x1a58c8, 0xf4f4f4, 0xd02828},
}

func (g *Game) startF1(e *Event) bool {
	amp := 0.0
	var y float64
	ok := false
	for _, a := range []float64{20, 10, 0} {
		if y, ok = g.clearStrip(a + 12); ok {
			amp = a
			break
		}
	}
	if !ok {
		return false
	}
	e.n = g.rng.Intn(len(f1Drivers))
	e.y, e.sp = y, amp
	e.x, e.vx = -60, 250+g.rng.Float64()*40
	if g.rng.Intn(2) == 0 {
		e.x, e.vx = g.lvl.W+60, -e.vx
	}
	e.tgt = []float64{y, g.rng.Float64() * 6}
	d := f1Drivers[e.n]
	g.say("¡" + d.name + " cruza la fiesta con su F1 diminuto!")
	g.snd.Play("alert")
	return true
}

func (g *Game) stepF1(e *Event, dt float64) bool {
	e.x += e.vx * dt
	e.y = e.tgt[0] + e.sp*math.Sin(e.t*3.4+e.tgt[1])
	d := f1Drivers[e.n]
	if g.rng.Float64() < dt*30 {
		g.burst(Vec{e.x - math.Copysign(16, e.vx), e.y - 3}, 1, 20, color.RGBA{190, 190, 200, 200}, 0.7, 3, -10)
	}
	if g.rng.Float64() < dt*14 {
		g.burst(Vec{e.x - math.Copysign(14, e.vx), e.y}, 1, 60, rgb(0xffc040), 0.25, 1, 120)
	}
	if e.phase == 0 && e.t > 0.4 {
		e.phase = 1
		for _, gu := range g.guests {
			if !gu.out() && gu.perf == "" && dist(gu.pos, Vec{e.x, e.y}) < 220 && g.rng.Float64() < 0.4 {
				gu.icon, gu.iconT = []string{"¡UN F1!", "¡VROOOM!", "¡ES ÉL!", "¡CUIDADO!"}[g.rng.Intn(4)], 2
			}
		}
	}
	if e.t > 1.2 && e.phase < 2 && g.rng.Float64() < dt*0.5 {
		e.phase = 2
		g.bubbleAt(Vec{e.x, e.y - 22}, d.shout, 1.6)
	}
	for _, gu := range g.guests {
		if gu.out() || gu.state == gLift {
			continue
		}
		if math.Abs(gu.pos.X-e.x) < 22 && math.Abs(gu.pos.Y-e.y) < 12 {
			g.killGuest(gu, false)
			g.sayNews("kill", map[string]string{"actor": d.name, "victim": guestLabel(gu)})
			g.snd.Play("thud")
			g.checkDone()
		}
	}
	if p := g.p; !p.dead && !p.hidden && math.Abs(p.pos.X-e.x) < 20 && math.Abs(p.pos.Y-e.y) < 10 {
		g.die("Te ha atropellado " + d.name)
	}
	return e.x > -90 && e.x < g.lvl.W+90
}

// bubbleAt: a short speech bubble from an event actor (no guest to own it).
func (g *Game) bubbleAt(pos Vec, txt string, dur float64) {
	g.bubbles = append(g.bubbles, worldBubble{pos: pos, txt: txt, t: dur})
}

var f1Imgs = map[int]*ebiten.Image{}

func f1Sprite(i int) *ebiten.Image {
	if im, ok := f1Imgs[i]; ok {
		return im
	}
	d := f1Drivers[i]
	body, acc, hel := c(d.body), c(d.accent), c(d.helmet)
	b := newPix(40, 16)
	b.ox, b.oy = 1, 1
	b.ellipse(20, 14, 17, 2.2, color.RGBA{0, 0, 0, 80})
	// tyres
	b.rect(5, 6, 8, 8, c(0x16161c))
	b.rect(7, 8, 4, 4, c(0x8a8a96))
	b.rect(27, 8, 6, 6, c(0x16161c))
	b.rect(29, 10, 2, 2, c(0x8a8a96))
	// rear wing, floor, body, nose
	b.rect(1, 2, 6, 2, acc)
	b.rect(3, 4, 2, 5, shade(body, 0.7))
	b.rect(8, 8, 26, 4, shade(body, 0.8))
	b.rect(9, 5, 20, 5, body)
	b.rect(9, 5, 20, 1, shade(body, 1.3))
	b.rect(28, 7, 9, 3, body)
	b.rect(36, 8, 2, 2, shade(body, 0.8))
	b.rect(34, 11, 5, 2, acc)
	b.rect(9, 8, 20, 1, acc)
	// cockpit with the driver's helmet
	b.ellipse(20, 5, 3, 2.6, hel)
	b.rect(18, 3, 5, 1, c(0x202028))
	b.rect(22, 4, 2, 2, c(0x181820))
	im := b.outlined(c(0x0a0a0e)).img()
	f1Imgs[i] = im
	return im
}

func (g *Game) drawF1(dst *ebiten.Image, e *Event) {
	im := f1Sprite(e.n)
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	if e.vx < 0 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w, 0)
	}
	op.GeoM.Translate(math.Round(g.wx(e.x)-w/2), math.Round(g.wy(e.y)-h+4))
	dst.DrawImage(im, op)
}

// ---- stampede -----------------------------------------------------------------

var herdTypes = []struct {
	kind, lo, hi int
	name         string
	speed        float64
	rx, ry       float64
	zoo          string
}{
	{animElephant, 3, 4, "elefantes", 95, 30, 15, "¡Una manada de elefantes se ha escapado del circo!"},
	{animBull, 4, 6, "toros", 120, 24, 12, "¡Una estampida de toros pasa por la fiesta!"},
	{animHorse, 4, 6, "caballos", 130, 22, 11, "¡Unos caballos desbocados cruzan la fiesta!"},
	{animGiraffe, 3, 4, "jirafas", 90, 24, 12, "¡Unas jirafas se han escapado del zoo!"},
	{animKangaroo, 5, 7, "canguros", 125, 16, 10, "¡Una familia de canguros ha escapado de Australia!"},
	{animPig, 9, 13, "cerdos", 95, 12, 8, "¡Una piara de cerdos escapa del corral!"},
	{animDuck, 10, 14, "patos", 75, 10, 7, "¡Una bandada de patos cruza a pie la fiesta!"},
}

func (g *Game) startStampede(e *Event) bool {
	ht := g.rng.Intn(len(herdTypes))
	if g.rng.Float64() < 0.3 {
		ht = 0 // the elephants are the classic
	}
	h := herdTypes[ht]
	y, ok := g.clearStrip(h.ry + 6)
	if !ok {
		return false
	}
	e.n = ht
	e.vx = h.speed
	dir := 1.0
	if g.rng.Intn(2) == 0 {
		dir = -1
	}
	e.vx *= dir
	n := h.lo + g.rng.Intn(h.hi-h.lo+1)
	startX := -70.0
	if dir < 0 {
		startX = g.lvl.W + 70
	}
	coat := g.rng.Intn(3)
	for i := 0; i < n; i++ {
		jy := (g.rng.Float64() - 0.5) * 22
		if h.ry < 11 {
			jy = (g.rng.Float64() - 0.5) * 50
		}
		e.herd = append(e.herd, herdRunner{kind: h.kind, coat: coat, x: startX - dir*float64(i)*(h.rx*1.2+g.rng.Float64()*14), y: y + jy, ph: g.rng.Float64() * 6})
	}
	g.say(h.zoo)
	g.snd.Play("alert")
	return true
}

func (g *Game) stepStampede(e *Event, dt float64) bool {
	h := herdTypes[e.n]
	live := false
	for i := range e.herd {
		r := &e.herd[i]
		r.x += e.vx * dt
		if r.x > -120 && r.x < g.lvl.W+120 {
			live = true
		}
		if g.rng.Float64() < dt*5 {
			g.burst(Vec{r.x - math.Copysign(h.rx*0.6, e.vx), r.y}, 1, 24, color.RGBA{170, 150, 120, 150}, 0.6, 3, -8)
		}
		for _, gu := range g.guests {
			if gu.out() || gu.state == gLift {
				continue
			}
			if math.Abs(gu.pos.X-r.x) < h.rx && math.Abs(gu.pos.Y-r.y) < h.ry*0.8 {
				g.killGuest(gu, false)
				g.sayNews("kill", map[string]string{"actor": "La manada de " + h.name, "victim": guestLabel(gu)})
				g.snd.Play("thud")
				e.t2++
				g.checkDone()
			}
		}
		if p := g.p; !p.dead && !p.hidden && math.Abs(p.pos.X-r.x) < h.rx*0.9 && math.Abs(p.pos.Y-r.y) < h.ry*0.7 {
			g.die("Te ha arrollado una manada de " + h.name)
		}
	}
	if !live && e.t > 1 {
		if e.t2 > 0 {
			g.say(fmt.Sprintf("La manada de %s deja %d víctimas y se pierde a lo lejos", h.name, int(e.t2)))
		}
		return false
	}
	return true
}

func (g *Game) drawStampede(dst *ebiten.Image, e *Event) {
	for _, r := range e.herd {
		frame := fWalk0 + int((e.t+r.ph)*9)%4
		if r.kind == animKangaroo || r.kind == animBull {
			frame = fWalk0 + int((e.t+r.ph)*8)%4
		}
		im := g.as.animals[animalKey(r.kind, r.coat, frame)]
		if im == nil {
			im = g.as.animals[animalKey(r.kind, r.coat, fStand)]
		}
		if im == nil {
			continue
		}
		w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
		x, y := g.wx(r.x), g.wy(r.y)
		fillShadow(dst, x, y)
		op := &ebiten.DrawImageOptions{}
		if e.vx < 0 {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		hop := 0.0
		if r.kind == animKangaroo {
			hop = math.Abs(math.Sin((e.t+r.ph)*7)) * 9
		}
		op.GeoM.Translate(math.Round(x-w/2), math.Round(y-h+2-hop))
		dst.DrawImage(im, op)
	}
}

// ---- parachutist -------------------------------------------------------------------

func (g *Game) startParachute(e *Event) bool {
	var p Vec
	ok := false
	for try := 0; try < 40 && !ok; try++ {
		p = g.lvl.RandomRoam(g.rng)
		ok = g.lvl.Outdoor(p) && !g.lvl.InPool(p)
	}
	if !ok {
		return false
	}
	spec := genericSpec(g.rng)
	spec.Hat, spec.Glasses, spec.Mask = HatHelmet, GlassShades, color.RGBA{}
	spec.Top, spec.Bottom = TopVest, BotPants
	spec.Shirt, spec.Shirt2 = hx(pick(g.rng, []uint32{0xf08020, 0xe03030, 0x30a0e0, 0xf0d020})), hx(0xf4f4f4)
	e.sets = []*CharSet{newCharSet(&spec)}
	g.lvl.NearestFree(p)
	e.x, e.y = p.X, p.Y           // where he will touch down
	e.vx = float64(g.rng.Intn(4)) // canopy colour set
	e.sp = 0
	g.say("¡Un paracaidista salta sobre la fiesta!")
	g.snd.Play("alert")
	return true
}

const parachuteFall = 6.0 // seconds in the air

func (g *Game) parachuteH(e *Event) float64 {
	return math.Max(0, (parachuteFall-e.t)/parachuteFall) * 230
}

func (g *Game) stepParachute(e *Event, dt float64) bool {
	switch e.phase {
	case 0:
		if e.t >= parachuteFall {
			e.phase, e.t = 1, 0
			land := g.lvl.NearestFree(Vec{e.x, e.y})
			e.x, e.y = land.X, land.Y
			if g.total < g.maxTotal() {
				g.addGuestAt(land)
				if gu := g.guests[len(g.guests)-1]; gu != nil {
					gu.cs = e.sets[0]
					gu.icon, gu.iconT = []string{"¡HOLA, FIESTA!", "¡HE ACERTADO!", "¡Qué buen aterrizaje!", "¡Qué altura!"}[g.rng.Intn(4)], 2.6
				}
			}
			g.burst(Vec{e.x, e.y - 4}, 8, 40, color.RGBA{220, 220, 210, 200}, 0.6, 2, 30)
			g.say("El paracaidista aterriza y se une a la fiesta")
		}
	default:
		return e.t < 6
	}
	return true
}

func canopyCol(set, i int) color.RGBA {
	pal := [][2]uint32{{0xe03030, 0xf4f4f4}, {0x3060e0, 0xf0d020}, {0x20a050, 0xf4f4f4}, {0xf08020, 0x2a2a34}}[set%4]
	return hx(pal[i%2])
}

func (g *Game) drawParachute(dst *ebiten.Image, e *Event) {
	set := int(e.vx)
	if e.phase == 1 {
		// the canopy collapses on the ground
		x, y := g.wx(e.x+16), g.wy(e.y+4)
		for dx := -22.0; dx <= 22; dx += 2 {
			hh := math.Sqrt(1-(dx/22)*(dx/22)) * 5
			fillRect(dst, x+dx, y-hh, 2, hh, canopyCol(set, int((dx+22)/8)))
		}
		return
	}
	h := g.parachuteH(e)
	sway := math.Sin(e.t*1.6) * 10 * (h / 230)
	pos := Vec{e.x + sway, e.y}
	g.drawActor(dst, e.sets[0], AnimWave, e.t*4, pos, DirDown, -h, false)
	cx, cy := g.wx(pos.X), g.wy(pos.Y)-h-62
	for dx := -28.0; dx <= 28; dx += 2 {
		hh := math.Sqrt(1-(dx/28)*(dx/28)) * 15
		fillRect(dst, cx+dx, cy-hh, 2, hh+2, canopyCol(set, int((dx+28)/8)))
	}
	for _, sx := range []float64{-28, -14, 14, 28} {
		for k := 0.0; k <= 1; k += 0.05 {
			fillRect(dst, cx+sx*(1-k), cy+2+k*50, 1, 1, color.RGBA{230, 230, 230, 200})
		}
	}
}

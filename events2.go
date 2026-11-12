package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Two more special events: the Ghostbusters chasing a ghost through the party,
// and a giant tyrannosaurus that eats and tramples whoever it meets.
// ---------------------------------------------------------------------------

var ghostbusterSlugs = []string{"gb_venkman", "gb_stantz", "gb_spengler", "gb_winston"}

// startGhost brings the Ghostbusters in; they follow the ghost for a while.
func (g *Game) startGhost(e *Event) bool {
	if g.guestRoom() < 4 {
		return false
	}
	base := g.lvl.RandomRoam(g.rng)
	for i, slug := range ghostbusterSlugs {
		f := presetByName(slug)
		if f == nil {
			continue
		}
		spec := f(g.rng)
		gu := &Guest{pos: g.lvl.NearestFree(Vec{base.X + float64(i)*14 - 20, base.Y + float64(i%2)*10}), dir: DirDown, cs: newCharSet(&spec), name: nameFromSpec(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: 1, role: i % 2, keep: true}
		g.guests = append(g.guests, gu)
		g.total++
		e.crew = append(e.crew, gu)
	}
	gp := g.lvl.RandomRoam(g.rng)
	e.x, e.y = gp.X, gp.Y
	e.bolts = []Vec{g.lvl.RandomRoam(g.rng)}
	g.say("¡Llegan los Cazafantasmos! Hay un fantasma suelto")
	g.snd.Play("alert")
	return true
}

func (g *Game) stepGhost(e *Event, dt float64) bool {
	// the ghost drifts from one corner of the party to another
	wp := e.bolts[0]
	d := dist(Vec{e.x, e.y}, wp)
	if d < 6 {
		e.bolts[0] = g.lvl.RandomRoam(g.rng)
	} else {
		e.x += (wp.X - e.x) / d * 34 * dt
		e.y += (wp.Y - e.y) / d * 34 * dt
	}
	e.boltT -= dt
	if e.boltT <= 0 {
		e.boltT = 0.6
		for i, gu := range e.crew {
			if gu.out() || gu.state == gFlee || gu.state == gRun {
				continue
			}
			tgt := Vec{e.x + float64(i-2)*22, e.y + 40}
			if dist(gu.pos, Vec{e.x, e.y}) > 70 {
				gu.path = g.lvl.PathG(gu.pos, g.lvl.NearestFree(tgt))
				gu.state, gu.next, gu.dur, gu.hurry = gGoto, gIdle, 0.5, true
			} else {
				gu.state, gu.timer = gIdle, 0.7
			}
		}
	}
	if e.t > 20 { // caught in the trap
		g.burst(Vec{e.x, e.y - 20}, 30, 100, rgb(0x90ffb0), 0.8, 2, 0)
		g.say("¡Fantasma atrapado! Los Cazafantasmos se quedan a la fiesta")
		g.snd.Play("win")
		for _, gu := range e.crew {
			if !gu.out() {
				gu.state, gu.timer, gu.hurry = gIdle, 1, false
			}
		}
		return false
	}
	return true
}

func (g *Game) drawGhostBeams(dst *ebiten.Image, e *Event) {
	for _, gu := range e.crew {
		if gu.out() || dist(gu.pos, Vec{e.x, e.y}) > 120 {
			continue
		}
		a := Vec{g.wx(gu.pos.X), g.wy(gu.pos.Y) - 22}
		b := Vec{g.wx(e.x), g.wy(e.y) - 24}
		n := int(dist(a, b)/3) + 1
		for i := 0; i <= n; i++ {
			t := float64(i) / float64(n)
			jx := math.Sin(g.time*30+float64(i)) * 1.5
			col := []color.RGBA{rgb(0xffa030), rgb(0xfff0a0), rgb(0x50d0ff)}[(i+int(g.time*20))%3]
			fillRect(dst, a.X+(b.X-a.X)*t+jx, a.Y+(b.Y-a.Y)*t, 2, 2, col)
		}
	}
}

func (g *Game) drawGhost(dst *ebiten.Image, e *Event) {
	x := g.wx(e.x)
	y := g.wy(e.y) - 26 + math.Sin(e.t*3)*4
	g.glowAt(dst, Vec{e.x, e.y - 26}, 38, color.RGBA{90, 255, 160, 60})
	body := rgba(0xe8fff0, 210)
	filledEllipse(dst, x, y-4, 12, 13, body)
	fillRect(dst, x-12, y-4, 24, 14, body)
	for i := 0.0; i < 4; i++ { // wavy hem
		filledEllipse(dst, x-9+i*6, y+10+math.Sin(e.t*8+i)*1.5, 3.2, 3.6, body)
	}
	fillRect(dst, x-6, y-8, 3, 4, rgb(0x14141c))
	fillRect(dst, x+3, y-8, 3, 4, rgb(0x14141c))
	filledEllipse(dst, x, y+1, 3, 3.5, rgb(0x14141c))
}

// ---- T-Rex --------------------------------------------------------------

func (g *Game) startRex(e *Event) bool {
	if g.isIsland() || pirateBay(g.lvl.Def) || len(g.liveGuests()) < 6 { // the island has its own rex; none wades out to the pirate ship
		return false
	}
	p := g.lvl.RandomRoam(g.rng)
	e.y = math.Max(120, math.Min(g.lvl.H-60, p.Y))
	e.x, e.vx = -150, 46
	if g.rng.Intn(2) == 0 {
		e.x, e.vx = g.lvl.W+150, -46
	}
	g.say("¡Un tiranosaurio gigante irrumpe en la fiesta!")
	g.snd.Play("boom")
	return true
}

func (g *Game) stepRex(e *Event, dt float64) bool {
	e.x += e.vx * dt
	dir := math.Copysign(1, e.vx)
	head := Vec{e.x + dir*52, e.y}
	e.boltT -= dt
	g.shake = math.Max(g.shake, 0.06+0.04*math.Abs(math.Sin(e.t*5)))
	for _, gu := range g.guests {
		if gu.out() || gu.perf == "dj" {
			continue
		}
		switch {
		case e.boltT <= 0 && e.n < 5 && dist(gu.pos, head) < 30:
			e.boltT = 1.1
			e.n++
			gu.gone = true
			g.burst(Vec{gu.pos.X, gu.pos.Y - 14}, 16, 90, rgb(0xb01818), 0.6, 2, 200)
			g.say("El T-Rex se ha comido a " + guestLabel(gu))
			g.snd.Play("stab")
		case dist(gu.pos, Vec{e.x, e.y}) < 17 || dist(gu.pos, Vec{e.x - dir*20, e.y}) < 15:
			g.killGuest(gu, false)
			g.snd.Play("thud")
		}
	}
	g.eachPlayer(func(p *Player) {
		if !p.dead && !p.hidden && (dist(p.pos, head) < 26 || dist(p.pos, Vec{e.x, e.y}) < 16) {
			g.die("Un T-Rex te ha pisoteado")
		}
	})
	g.checkDone()
	return e.x > -200 && e.x < g.lvl.W+200
}

func genRexFrame(f int) *pixBuf {
	// the same T-Rex as the island's (rexart.go), walking, looking RIGHT (the art looks left)
	b := newPix(160, 99)
	b.ox, b.oy = 22, 13
	drawRexArt(b, []int{rexPoseWalkA, rexPoseWalkB}[f%2])
	o := newPix(b.w, b.h)
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			copy(o.p[(y*o.w+x)*4:(y*o.w+x)*4+4], b.p[(y*b.w+(b.w-1-x))*4:(y*b.w+(b.w-1-x))*4+4])
		}
	}
	return o
}

func (g *Game) rexSprite(f int) *ebiten.Image {
	if g.rexImg[f%2] == nil {
		g.rexImg[f%2] = genRexFrame(f).outlinedSel(0.45).img()
	}
	return g.rexImg[f%2]
}

func (g *Game) drawRex(dst *ebiten.Image, e *Event) {
	im := g.rexSprite(int(e.t * 3))
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	if e.vx < 0 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w, 0)
	}
	op.GeoM.Translate(g.wx(e.x)-w/2, g.wy(e.y)-h+6)
	dst.DrawImage(im, op)
}

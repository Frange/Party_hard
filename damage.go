package main

import (
	"image"
	"math"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// DAÑO VISUAL DEL ESCENARIO: after explosions, fires, piano drops and blows, the ground keeps subtle
// marks until the party ends: soot rays and a broken rim around a blast, cracks, burnt specks, scuffs;
// next to a prop that was caught by a blast, cracks run out from its base. Pure decoration: no
// collisions, no cover, no gameplay, no new props.
//
// It reads what the game already does: every new scorch decal (Decal kind 1) of the party is stamped
// (poll, `dmg.seen`), and a few events call g.scar directly. The marks are painted ONCE into a pixel
// buffer (dmg.buf, premultiplied) the size of the level, uploaded with one WritePixels only when a new
// mark appears, and drawn with ONE DrawImage per frame: nothing is allocated per frame. Its own random
// numbers (g.rng is not touched). At most dmgMax marks per party. PH_DAMAGE=0 disables it.
// ---------------------------------------------------------------------------

type scarKind uint8

const (
	scarBlast scarKind = iota // crater rim + soot rays + cracks
	scarBurn                  // soot patch and charcoal specks (fire)
	scarCrack                 // a few jagged cracks (a piano, a heavy blow)
	scarScuff                 // scratches (a punch, a knock)
)

const dmgMax = 90

var damageOn = os.Getenv("PH_DAMAGE") != "0"
var demoDamage = os.Getenv("PH_DAMAGEDEMO") == "1"

type dmgState struct {
	lvl   *Level
	w, h  int
	buf   *image.RGBA
	img   *ebiten.Image
	dirty bool
	seen  int // g.decals already looked at
	n     int // marks stamped
	t     float64
	rng   *rand.Rand
	hurt  map[*Prop]bool
	demo  bool
}

var dmg dmgState

// dmgReset prepares the layer for the party on screen (a new level, or a restart).
func (g *Game) dmgReset() {
	l := g.lvl
	w, h := int(math.Ceil(l.W*wk)), int(math.Ceil(l.H*wk))
	if l.bg != nil {
		b := l.bg.Bounds()
		w, h = b.Dx(), b.Dy()
	}
	if w <= 0 || h <= 0 || w > 4096 || h > 4096 {
		w, h = screenW, screenH
	}
	if dmg.buf == nil || dmg.w != w || dmg.h != h {
		dmg.buf, dmg.w, dmg.h = image.NewRGBA(image.Rect(0, 0, w, h)), w, h
		if dmg.img != nil {
			dmg.img.Deallocate()
		}
		dmg.img = nil
	} else {
		for i := range dmg.buf.Pix {
			dmg.buf.Pix[i] = 0
		}
	}
	dmg.lvl, dmg.seen, dmg.n, dmg.dirty, dmg.t, dmg.demo = l, 0, 0, false, g.time, false
	dmg.rng = rand.New(rand.NewSource(int64(len(l.Def.Title))*7919 + int64(w)))
	dmg.hurt = map[*Prop]bool{}
}

// dmgPx: "over" blend of a straight-alpha colour into the premultiplied buffer.
func dmgPx(x, y int, r, gr, b, a uint8) {
	if !(image.Pt(x, y).In(dmg.buf.Rect)) || a == 0 {
		return
	}
	i := dmg.buf.PixOffset(x, y)
	p := dmg.buf.Pix[i : i+4 : i+4]
	ia := 255 - uint32(a)
	p[0] = uint8(uint32(r)*uint32(a)/255 + uint32(p[0])*ia/255)
	p[1] = uint8(uint32(gr)*uint32(a)/255 + uint32(p[1])*ia/255)
	p[2] = uint8(uint32(b)*uint32(a)/255 + uint32(p[2])*ia/255)
	p[3] = uint8(uint32(a) + uint32(p[3])*ia/255)
}

// dmgLine: a one-pixel jagged line from (x,y) heading `ang`, `n` pixels long.
func dmgLine(x, y, ang float64, n int, a uint8) {
	for i := 0; i < n; i++ {
		if dmg.rng.Intn(5) == 0 {
			ang += (dmg.rng.Float64() - 0.5) * 1.1
		}
		x += math.Cos(ang)
		y += math.Sin(ang) * 0.55 // the ground is seen from above at an angle
		fade := uint8(float64(a) * (1 - float64(i)/float64(n)*0.6))
		dmgPx(int(math.Round(x)), int(math.Round(y)), 18, 14, 12, fade)
	}
}

// scar stamps a mark at the world position (x, y) with size r (world pixels).
func (g *Game) scar(k scarKind, x, y, r float64) {
	if !damageOn || g.lvl == nil || dmg.buf == nil || dmg.lvl != g.lvl || dmg.n >= dmgMax {
		return
	}
	dmg.n++
	dmg.dirty = true
	cx, cy := int(math.Round(x*wk)), int(math.Round(y*wk))
	R := math.Max(3, r*wk)
	switch k {
	case scarBlast:
		for yy := -int(R * 0.6); yy <= int(R*0.6); yy++ { // soft soot disc
			for xx := -int(R); xx <= int(R); xx++ {
				d := math.Hypot(float64(xx)/R, float64(yy)/(R*0.6))
				if d <= 1 {
					dmgPx(cx+xx, cy+yy, 12, 10, 10, uint8(34*(1-d)+6))
				}
			}
		}
		for i := 0; i < 90; i++ { // the broken rim: brownish chips on the edge
			a := dmg.rng.Float64() * 2 * math.Pi
			rr := R * (0.78 + dmg.rng.Float64()*0.3)
			dmgPx(cx+int(math.Cos(a)*rr), cy+int(math.Sin(a)*rr*0.6), 150, 128, 104, 120)
		}
		n := 6 + dmg.rng.Intn(4)
		for i := 0; i < n; i++ { // rays of soot and cracks
			a := float64(i)/float64(n)*2*math.Pi + dmg.rng.Float64()*0.5
			dmgLine(float64(cx)+math.Cos(a)*R*0.4, float64(cy)+math.Sin(a)*R*0.25, a, int(R*(0.8+dmg.rng.Float64()*0.9)), 150)
		}
		for i := 0; i < 14; i++ { // debris specks around
			a := dmg.rng.Float64() * 2 * math.Pi
			rr := R * (1 + dmg.rng.Float64()*0.8)
			dmgPx(cx+int(math.Cos(a)*rr), cy+int(math.Sin(a)*rr*0.6), 170, 150, 126, 130)
		}
	case scarBurn:
		for i := 0; i < int(R*2.2); i++ {
			a := dmg.rng.Float64() * 2 * math.Pi
			rr := R * math.Sqrt(dmg.rng.Float64())
			dmgPx(cx+int(math.Cos(a)*rr), cy+int(math.Sin(a)*rr*0.55), 14, 12, 12, uint8(60+dmg.rng.Intn(50)))
		}
	case scarCrack:
		n := 3 + dmg.rng.Intn(3)
		for i := 0; i < n; i++ {
			a := float64(i)/float64(n)*2*math.Pi + dmg.rng.Float64()*0.8
			dmgLine(float64(cx), float64(cy), a, int(R*(0.9+dmg.rng.Float64())), 150)
		}
	case scarScuff:
		for i := 0; i < 3; i++ {
			a := -0.6 + dmg.rng.Float64()*1.2
			dmgLine(float64(cx-3+i*3), float64(cy-1+i), a, 4+dmg.rng.Intn(4), 100)
		}
	}
}

// crackProps: the solid props next to a blast get cracks running out from their base (once per prop).
func (g *Game) crackProps(x, y, r float64) {
	for _, pr := range g.lvl.Props {
		if pr.gone || pr.K == nil || pr.K.NoSolid || pr.K.Trap != "" || dmg.hurt[pr] || dmg.n >= dmgMax {
			continue
		}
		fx, fy := pr.Foot.X+pr.Foot.W/2, pr.Foot.Y+pr.Foot.H
		if pr.Foot.W <= 0 || math.Abs(fx-x) > r*2.2 || math.Abs(fy-y) > r*1.2 {
			continue
		}
		dmg.hurt[pr] = true
		g.scar(scarCrack, fx+(dmg.rng.Float64()-0.5)*pr.Foot.W*0.6, fy, math.Min(16, 7+pr.Foot.W/6))
	}
}

// updateDamage: call once per update of the party.
func (g *Game) updateDamage() {
	if !damageOn || g.lvl == nil {
		return
	}
	if dmg.lvl != g.lvl || g.time < dmg.t || len(g.decals) < dmg.seen {
		g.dmgReset()
	}
	dmg.t = g.time
	if demoDamage && !dmg.demo && g.time > 0.3 { // PH_DAMAGEDEMO=1 (screenshots only): a few blasts and fires around the start
		dmg.demo = true
		s := g.lvl.Def.Start
		g.decals = append(g.decals, Decal{pos: Vec{s.X - 150, s.Y - 170}, kind: 1, r: 22}, Decal{pos: Vec{s.X + 30, s.Y - 150}, kind: 1, r: 16},
			Decal{pos: Vec{s.X - 60, s.Y - 120}, kind: 1, r: 9}, Decal{pos: Vec{s.X + 110, s.Y - 190}, kind: 1, r: 18})
		g.scar(scarScuff, s.X-100, s.Y-100, 6)
	}
	for ; dmg.seen < len(g.decals); dmg.seen++ {
		d := g.decals[dmg.seen]
		if d.kind != 1 || d.r < 5 {
			continue
		}
		switch {
		case d.r >= 14:
			g.scar(scarBlast, d.pos.X, d.pos.Y, d.r*1.3)
			g.crackProps(d.pos.X, d.pos.Y, d.r*1.3)
		case d.r >= 7:
			g.scar(scarBurn, d.pos.X, d.pos.Y, d.r)
		}
	}
}

// drawDamage: the accumulated layer, one DrawImage (right after the decals, under props and actors).
func (g *Game) drawDamage(dst *ebiten.Image) {
	if !damageOn || dmg.lvl != g.lvl || dmg.n == 0 || dmg.buf == nil {
		return
	}
	if dmg.img == nil {
		dmg.img = ebiten.NewImage(dmg.w, dmg.h)
		dmg.dirty = true
	}
	if dmg.dirty {
		dmg.img.WritePixels(dmg.buf.Pix)
		dmg.dirty = false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.camI.X, g.camI.Y)
	dst.DrawImage(dmg.img, op)
}

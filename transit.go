package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Places with no road (the cruise ship, the space station): police, ambulances,
// the limousine with new guests, the hearse, the technicians and the kids all
// arrive by boat or by little spaceship, each in the colour of its car.
// They dock along the same lane (CarFrom..CarTo) the cars would use.
// ---------------------------------------------------------------------------

// transitStyle: body and trim colours (and size) for a vehicle of a transit level.
func transitStyle(v *Vehicle) (body, trim color.RGBA, long bool, label string, labCol color.RGBA) {
	body, trim = hx(0x2a50c0), hx(0xf4f4f8)
	switch v.kind {
	case 0:
		switch v.unit {
		case unitFBI:
			body, trim, label, labCol = hx(0x1c2036), hx(0xb842f5), "FBI", color.RGBA{184, 66, 245, 255}
		case unitMilitary:
			body, trim, label, labCol = hx(0x4a5a30), hx(0x3cc846), "MILITAR", color.RGBA{60, 200, 70, 255}
		case unitSWAT:
			body, trim, label, labCol = hx(0x3a3c44), hx(0xffca26), "SWAT", color.RGBA{255, 202, 38, 255}
		case unitRangers:
			body, trim, label, labCol = hx(0xc02030), hx(0xf4f4f8), "POWER RANGERS", color.RGBA{255, 60, 60, 255}
		}
	case 1:
		body, trim = hx(0xf4f4f8), hx(0xd02a2a)
	case 4:
		body, trim = hx(0x241c2c), hx(0x8a6aa8)
	case 6:
		body, trim, long = hx(0x14141c), hx(0xd0d0e0), true
	case 7:
		body, trim = hx(0xd02a2a), hx(0xf4f4f8)
	case 8:
		body, trim = hx(0xf0c828), hx(0x2a4ab8)
	case 9:
		switch v.alt {
		case 0:
			body, trim = hx(0xe83a3a), hx(0xf4f4f8)
		case 1:
			body, trim = hx(0x3aa050), hx(0xf0d020)
		default:
			body, trim = hx(0xf08a20), hx(0x2a4ab8)
		}
	}
	return
}

// genBoat paints a motor launch facing right, 90 px wide (112 for the long one).
func genBoat(body, trim color.RGBA, long bool) *pixBuf {
	w := 90
	if long {
		w = 112
	}
	b := newPix(w, 42)
	b.ellipse(float64(w)/2, 35, float64(w)/2-3, 4, color.RGBA{10, 30, 60, 90})
	b.rect(4, 26, w-14, 9, body)
	for i := 0; i < 9; i++ { // the bow
		b.rect(w-10, 26+i, 10-i, 1, body)
	}
	b.rect(4, 29, w-14, 2, trim)
	b.rect(4, 34, w-14, 1, shade(body, 0.55))
	b.rect(4, 26, w-14, 1, shade(body, 1.3))
	cab := 34
	if long {
		cab = 56
	}
	b.rect(22, 14, cab, 12, shade(body, 1.12))
	for x := 25; x < 22+cab-4; x += 14 {
		b.rect(x, 17, 11, 6, c(0x9ad0ea))
	}
	b.rect(20, 12, cab+4, 3, trim)
	b.rect(0, 26, 5, 10, c(0x2a2a30)) // outboard motor
	b.rect(24, 6, 3, 6, c(0x6a6a74))  // beacon
	b.rect(23, 4, 5, 2, c(0xff4040))
	for x := 6; x < w-8; x += 5 { // foam along the waterline
		b.set(x, 35, c(0xe8f4ff))
	}
	return b
}

// genShip paints a little shuttle facing right.
func genShip(body, trim color.RGBA, long bool) *pixBuf {
	w := 96
	if long {
		w = 116
	}
	b := newPix(w, 42)
	fw := float64(w)
	b.ellipse(fw/2, 38, fw/2-6, 3, color.RGBA{0, 0, 10, 90})
	b.ellipse(fw/2, 24, fw/2-4, 11, shade(body, 0.7))
	b.ellipse(fw/2, 22, fw/2-4, 10, body)
	b.ellipse(fw/2, 20, fw/2-9, 6, shade(body, 1.2))
	b.ellipse(fw/2, 27, fw/2-12, 1.6, trim)
	b.ellipse(fw-30, 17, 12, 7, c(0x7ad0f0)) // canopy
	b.ellipse(fw-33, 15, 5, 3, c(0xc8f4ff))
	for i := 0; i < 11; i++ { // tail fin
		b.rect(10+i, 5+i, 4, 1, trim)
	}
	b.rect(10, 16, 4, 6, trim)
	b.rect(26, 30, 34, 3, shade(body, 0.6)) // wing
	b.ellipse(4, 24, 4, 6, c(0xff9a30))     // engine glow
	b.ellipse(3, 24, 2, 3.5, c(0xfff0a0))
	b.rect(32, 33, 2, 5, c(0x3a3a46)) // landing legs
	b.rect(62, 33, 2, 5, c(0x3a3a46))
	for x := 30; x < w-44; x += 9 { // portholes
		b.rect(x, 21, 3, 3, c(0xfff0b0))
	}
	return b
}

var transitImgs = map[string]*ebiten.Image{}

func (g *Game) drawTransit(dst *ebiten.Image, v *Vehicle) {
	body, trim, long, label, labCol := transitStyle(v)
	sea := g.lvl.Def.Transit == "sea"
	key := fmt.Sprintf("%v%x%x%x%x%v", sea, body.R, body.G, body.B, trim.R+trim.G+trim.B, long)
	im := transitImgs[key]
	if im == nil {
		var pb *pixBuf
		if sea {
			pb = genBoat(body, trim, long)
		} else {
			pb = genShip(body, trim, long)
		}
		im = pb.outlinedSel(0.42).img()
		transitImgs[key] = im
	}
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	bob := math.Sin(g.time*2+v.x*0.05) * 1.3
	if !sea {
		bob = math.Sin(g.time*3)*1.5 - 4
	}
	op := &ebiten.DrawImageOptions{}
	left := g.wx(v.x) - w/2
	if v.state != 2 { // sailing in from the right: bow to the left
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w, 0)
	}
	op.GeoM.Translate(math.Round(left), math.Round(g.wy(v.y)-h+8+bob))
	dst.DrawImage(im, op)
	if !sea { // exhaust glow
		dx := -w/2 + 4
		if v.state != 2 {
			dx = w/2 - 4
		}
		g.glowAt(dst, Vec{g.wx(v.x) + dx - g.camI.X, g.wy(v.y) - 14 + bob - g.camI.Y}, 14, color.RGBA{255, 150, 40, 110})
	}
	if label != "" {
		g.drawText(dst, label, math.Round(g.wx(v.x))-g.textW(label, 1)/2, math.Round(g.wy(v.y))-h+2+bob, labCol, 1)
	}
}

// parkSlot: where the next vehicle of the level stops, and the lane it drives in.
func (g *Game) parkSlot() (tx, ly float64) {
	def := g.lvl.Def
	n := 0
	for _, v := range g.veh {
		if v.state != 2 {
			n++
		}
	}
	tx = def.CarTo.X - float64(n)*100
	if def.ParkX1 > def.ParkX0 {
		if def.CarFrom.X > def.CarTo.X {
			tx = math.Max(def.ParkX0, def.ParkX1-float64(n)*100)
		} else {
			tx = math.Min(def.ParkX1, def.ParkX0+float64(n)*100)
		}
	}
	ly = def.CarFrom.Y + g.laneOff()
	if def.CarFrom.X > def.CarTo.X {
		ly = def.CarFrom.Y - g.laneOff()
	}
	return
}

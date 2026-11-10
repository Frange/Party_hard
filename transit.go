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

var transitImgs = map[string]*ebiten.Image{}

func (g *Game) drawTransit(dst *ebiten.Image, v *Vehicle) {
	body, trim, long, label, labCol := transitStyle(v)
	sea := g.lvl.Def.Transit == "sea"
	laneRemap(g.lvl.Def, v)
	if v.lane { // the lane runs up or down the screen: bow or stern to the camera
		g.drawTransitView(dst, v, sea, body, trim, long, label, labCol)
		return
	}
	style := transitCraft(v)
	frame := int(g.time*3) % 2 // the pennant waves
	if !sea {
		frame = 0
	}
	key := fmt.Sprintf("%v%x%x%x%x%v%d%d", sea, body.R, body.G, body.B, trim.R+trim.G+trim.B, long, style, frame)
	im := transitImgs[key]
	if im == nil {
		var pb *pixBuf
		if sea {
			pb = genBoat(body, trim, long, style, frame) // vessels.go
		} else {
			pb = genShip(body, trim, long, style)
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
	if !sea { // the cyan glow of the engine
		dx := -w/2 + 3
		if v.state != 2 {
			dx = w/2 - 3
		}
		g.glowAt(dst, Vec{g.wx(v.x) + dx - g.camI.X, g.wy(v.y) - h + 30 + bob - g.camI.Y}, 14, color.RGBA{60, 210, 255, 110})
	} else if v.state != 1 { // the wake behind the stern while it sails
		sx, dir := g.wx(v.x)+w/2, 1.0
		if v.state == 2 {
			sx, dir = g.wx(v.x)-w/2, -1.0
		}
		for i := 0; i < 4; i++ {
			age := math.Mod(g.time*2+float64(i)/4, 1)
			filledEllipse(dst, math.Round(sx+dir*(4+age*26)), math.Round(g.wy(v.y)-6+bob), math.Round(6+age*6), 2, rgba(0xe8f4ff, uint8(150*(1-age))))
		}
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

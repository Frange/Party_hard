package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Front and rear views of the flying and sailing vehicles. The side sprites
// (props2.go genHeliPal, vessels.go genBoat / genShip) are only right while a
// vehicle moves left or right; when it comes down or goes up the screen the
// bow (or the nose) points at the camera or away from it, so it is painted
// from the front (it comes down) or from behind (it goes up).
// ---------------------------------------------------------------------------

// ---- helicopters ----------------------------------------------------------------------

// heliLivery: body, trim and style per responder kind (police .. rangers) and medical at [5].
var heliLivery = [6][3]uint32{
	{0xdcdce8, 0x2a4a9a, 0}, {0x20202c, 0x5a2a9a, 1}, {0x5a6a3a, 0x2c3a1c, 2},
	{0x30323a, 0x20222a, 3}, {0xf4f4f8, 0xd02a5a, 4}, {0xf8f8fc, 0xd02828, 5},
}

// heliViewAnchor: the centre of the fuselage in the 64x64 view (the rotor hangs 25 px above it).
const heliViewCX, heliViewCY = 32, 34

// genHeliView paints the helicopter from the front (front=true, coming down the screen)
// or from behind (going up). The rotor is drawn on top while playing, as for the side view.
func genHeliView(body, trim uint32, style int, front bool) *pixBuf {
	b := newPix(64, 64)
	b.ox, b.oy = 1, 1
	dark := c(0x3a3a46)
	b.rect(14, 52, 36, 2, dark) // skids
	b.rect(20, 44, 2, 8, dark)
	b.rect(42, 44, 2, 8, dark)
	b.ellipse(32, 36, 15, 14, c(trim))
	b.ellipse(32, 33, 14, 13, c(body))
	b.rect(25, 18, 14, 9, c(body)) // engine cowling
	b.rect(25, 18, 14, 2, c(trim))
	b.rect(30, 10, 4, 9, dark) // mast and hub
	b.rect(28, 8, 8, 3, dark)
	if front {
		b.ellipse(32, 31, 11, 9, c(0x7ab0d8)) // wind-screen
		b.ellipse(32, 29, 8, 5, c(0xa8d8f0))
		b.rect(31, 22, 2, 17, c(trim)) // frame
		b.rect(29, 44, 6, 2, c(0xf0e070))
		b.rect(31, 14, 2, 4, c(trim)) // tail fin peeping above
	} else {
		b.rect(24, 24, 16, 2, c(body)) // stabiliser
		b.rect(26, 21, 4, 3, dark)     // exhausts
		b.rect(35, 21, 4, 3, dark)
		b.ellipse(32, 28, 5, 5, shade(c(body), 0.8)) // the tail boom comes at the viewer
		b.rect(31, 12, 3, 14, c(trim))               // fin
		b.rect(22, 33, 4, 5, c(0x7ab0d8))            // side windows
		b.rect(39, 33, 4, 5, c(0x7ab0d8))
	}
	switch style {
	case 1: // FBI
		b.rect(18, 36, 28, 2, c(0xb842f5))
		for i := 0; i < 3; i++ {
			b.rect(24+i*8, 40, 5, 3, c(0xf0f0f8))
		}
	case 2: // military
		for _, p := range [][4]int{{20, 30, 5, 3}, {39, 34, 5, 3}, {28, 40, 6, 3}} {
			b.rect(p[0], p[1], p[2], p[3], c(0x2c3a1c))
		}
		b.rect(12, 40, 7, 3, c(0x1c2214))
		b.rect(45, 40, 7, 3, c(0x1c2214))
	case 3: // SWAT
		b.rect(20, 38, 24, 2, c(0xffca26))
		b.rect(26, 20, 12, 2, c(0xffca26))
	case 4: // rangers
		for i, cc := range []uint32{0xe03030, 0x3060e0, 0xf0d020, 0xf070b0, 0x30c050} {
			b.rect(19+i*5, 36, 5, 5, c(cc))
		}
	case 5: // medical
		b.rect(18, 37, 28, 2, c(0xd02828))
		b.rect(29, 39, 6, 5, c(0xd02828))
		b.rect(27, 40, 10, 2, c(0xd02828))
	}
	return b
}

var heliViewImgs = map[int]*ebiten.Image{}

// heliViewImg: the cached front/rear view of livery idx (0..5), scaled like the side sprite.
func heliViewImg(idx int, front bool) *ebiten.Image {
	key := idx * 2
	if front {
		key++
	}
	if im := heliViewImgs[key]; im != nil {
		return im
	}
	l := heliLivery[clampI(idx, 0, 5)]
	im := genHeliView(l[0], l[1], int(l[2]), front).scaled(wk).outlinedSel(0.45).img()
	heliViewImgs[key] = im
	return im
}

// heliLiveryIdx: which livery a helicopter wears (see heliImg in draw.go).
func heliLiveryIdx(v *Vehicle) int {
	switch {
	case v.kind == 3:
		return 5
	case v.kind == 2 && v.unit >= 0 && v.unit < 5:
		return v.unit
	}
	return 0
}

// heliFlying reports whether the helicopter is moving up or down the screen (state 0 comes
// down, state 2 goes up); only then the front/rear view replaces the side one.
func heliFlying(v *Vehicle) (flying, front bool) {
	switch v.state {
	case 0:
		return true, true
	case 2:
		return true, false
	}
	return false, false
}

// sameSprite reports whether two sprites are the same picture: a few edge pixels may differ
// (the exported copy of a sprite was outlined by an older build), a redrawn one does not match.
func sameSprite(a, b *pixBuf) bool {
	if a == nil || b == nil || a.w != b.w || a.h != b.h {
		return false
	}
	used, diff := 0, 0
	for i := 0; i < len(a.p); i += 4 {
		if a.p[i+3] == 0 && b.p[i+3] == 0 {
			continue // fully transparent pixels may carry any colour
		}
		used++
		if a.p[i+3] != b.p[i+3] || a.p[i] != b.p[i] || a.p[i+1] != b.p[i+1] || a.p[i+2] != b.p[i+2] {
			diff++
		}
	}
	return used > 0 && diff*100 <= used*12
}

// ---- boats and shuttles ---------------------------------------------------------------

// genBoatView and genShipView (the bow / nose and stern / engine views) live in vessels.go.

var transitViewImgs = map[string]*ebiten.Image{}

// transitView: the cached front or rear view of a boat or shuttle.
func transitView(sea bool, body, trim color.RGBA, long, front bool, style int) *ebiten.Image {
	key := fmt.Sprintf("v%v%x%x%x%x%v%v%d", sea, body.R, body.G, body.B, int(trim.R)+int(trim.G)+int(trim.B), long, front, style)
	if im := transitViewImgs[key]; im != nil {
		return im
	}
	var pb *pixBuf
	if sea {
		pb = genBoatView(body, trim, long, front, style)
	} else {
		pb = genShipView(body, trim, long, front, style)
	}
	im := pb.outlinedSel(0.42).img()
	transitViewImgs[key] = im
	return im
}

// drawTransitView paints a boat or shuttle that moves up or down the screen.
func (g *Game) drawTransitView(dst *ebiten.Image, v *Vehicle, sea bool, body, trim color.RGBA, long bool, label string, labCol color.RGBA) {
	front := v.vdir > 0 != (v.state == 2) // coming down: bow to us; leaving (going back up): stern
	im := transitView(sea, body, trim, long, front, transitCraft(v))
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	bob := math.Sin(g.time*2+v.y*0.05) * 1.3
	if !sea {
		bob = math.Sin(g.time*3)*1.5 - 4
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(g.wx(v.x)-w/2), math.Round(g.wy(v.y)-h+8+bob))
	dst.DrawImage(im, op)
	if !sea && !front {
		g.glowAt(dst, Vec{g.wx(v.x) - g.camI.X, g.wy(v.y) - h + 30 + bob - g.camI.Y}, 14, color.RGBA{60, 210, 255, 110})
	}
	if label != "" {
		g.drawText(dst, label, math.Round(g.wx(v.x))-g.textW(label, 1)/2, math.Round(g.wy(v.y))-h+2+bob, labCol, 1)
	}
}

// ---- lanes that run up or down the screen --------------------------------------------

// laneVertical: a boat or space level whose lane (CarFrom..CarTo) goes up or down the screen.
func laneVertical(def *LevelDef) bool {
	return def != nil && def.Transit != "" &&
		math.Abs(def.CarTo.Y-def.CarFrom.Y) > math.Abs(def.CarTo.X-def.CarFrom.X)+1
}

// laneDir: +1 when vehicles arrive moving down the screen, -1 when they move up.
func laneDir(def *LevelDef) float64 {
	if def.CarTo.Y >= def.CarFrom.Y {
		return 1
	}
	return -1
}

// laneRemap turns a vehicle that was set up for a horizontal lane (dispatch, kids, hearse... all
// start at CarFrom.X+n with a lane offset in y) into one that drives along a vertical lane.
func laneRemap(def *LevelDef, v *Vehicle) {
	if v.lane || !laneVertical(def) {
		return
	}
	v.lane = true
	d := laneDir(def)
	v.vdir = d
	slots := math.Round(math.Abs(def.CarTo.X-v.tx) / 100)
	if def.ParkX1 > def.ParkX0 {
		slots = math.Round(math.Abs(def.ParkX1-v.tx) / 100)
	}
	slots = math.Max(0, math.Min(slots, 4))
	lat := v.ty - def.CarFrom.Y // the lane offset (the two directions use opposite sides)
	v.x = def.CarFrom.X + lat
	v.tx = v.x
	v.y = def.CarFrom.Y - d*40
	v.ty = def.CarTo.Y - d*slots*100
}

// laneStep moves a remapped vehicle one frame: arriving (state 0) or leaving (state 2).
// It reports whether the vehicle has left the screen for good.
func laneStep(def *LevelDef, v *Vehicle, dt float64) (gone bool) {
	switch v.state {
	case 0:
		v.y += v.vdir * 130 * dt
		if (v.y-v.ty)*v.vdir >= 0 {
			v.y, v.state = v.ty, 1
		}
	case 2:
		v.y -= v.vdir * 130 * dt
		if (def.CarFrom.Y-v.vdir*80-v.y)*v.vdir >= 0 {
			return true
		}
	}
	return false
}

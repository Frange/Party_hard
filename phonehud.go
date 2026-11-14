package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The ringing phone in the top right corner: a blue handset (phone_art.go) with
// red sound dashes and label while someone is calling the police, yellow while
// someone calls the technicians. It jumps and rattles like a comic-strip phone.

// callingKind: 0 nobody, 1 police, 2 technicians.
func (g *Game) callingKind() int {
	k := 0
	for _, gu := range g.guests {
		if gu.state == gCall && !gu.out() {
			if !gu.tech {
				return 1
			}
			k = 2
		}
	}
	return k
}

func (g *Game) drawCallHud(dst *ebiten.Image) {
	if g.mode != ModePlay {
		return
	}
	k := g.callingKind()
	if k == 0 {
		return
	}
	label, ring := "¡RIIING!", color.RGBA{255, 90, 80, 255} // police: red dashes and label
	if k == 2 {
		label, ring = "¡TRIIIN!", color.RGBA{255, 220, 60, 255} // technicians: yellow
	}
	im := phoneHandset(phoneTintForCall(g.lvl, k)) // the colour of the booth (phone_art.go): sharp, whole scale
	t := g.time
	jump := math.Round(math.Abs(math.Sin(t*11)) * 4)
	shake := float64(int(t*26)%2)*2 - 1 // it rattles one pixel each side
	const sc = 2
	w, h := float64(im.Bounds().Dx()*sc), float64(im.Bounds().Dy()*sc)
	cx, cy := float64(screenW)-52, 80-jump // 12+ px under the trophy box
	x0, y0 := math.Round(cx-w/2+shake*sc), math.Round(cy-h/2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(x0, y0)
	dst.DrawImage(im, op)
	// the sound dashes beside the ear cup and the mouth cup, in the colour of who is called, blinking in turn
	dark := color.RGBA{30, 10, 10, 255}
	for i := 0; i < 3; i++ {
		if int(t*10+float64(i))%4 == 0 {
			continue
		}
		dx := float64(10 + i*2)
		for _, grp := range [2][2]float64{{x0 + 8, y0 + 2}, {x0 + 0, y0 + 26}} {
			x, y := grp[0]-dx-8, grp[1]+float64(i*6)
			if i == 1 {
				x -= 3 // the middle dash sticks out
			}
			fillRect(dst, x-1, y-1, 10, 4, dark)
			fillRect(dst, x, y, 8, 2, ring)
		}
	}
	g.bigText(dst, label, cx, 118, ring, color.RGBA{40, 0, 0, 255}, 1.5)
}

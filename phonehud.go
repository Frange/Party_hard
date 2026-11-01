package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The ringing phone in the top right corner: red while someone is calling the
// police, yellow while someone calls the technicians. It jumps and shakes like
// a comic-strip phone.

var phoneIcons = map[uint32]*ebiten.Image{}

func phoneIcon(col uint32) *ebiten.Image {
	if im, ok := phoneIcons[col]; ok {
		return im
	}
	body, dark, hi := c(col), shade(c(col), 0.55), shade(c(col), 1.35)
	b := newPix(22, 16)
	// handset: an arched grip with a cup at each end, mouth and ear down
	b.rect(6, 1, 10, 3, body)
	b.rect(3, 3, 4, 3, body)
	b.rect(15, 3, 4, 3, body)
	b.rect(1, 6, 6, 8, body)
	b.rect(15, 6, 6, 8, body)
	b.rect(6, 1, 10, 1, hi)
	b.rect(1, 6, 6, 1, hi)
	b.rect(15, 6, 6, 1, hi)
	b.rect(6, 3, 10, 1, dark)
	b.rect(1, 13, 6, 1, dark)
	b.rect(15, 13, 6, 1, dark)
	for _, x := range []int{3, 5} {
		b.set(x, 11, dark)
		b.set(x+14, 11, dark)
	}
	im := b.outlined(c(0x101010)).img()
	phoneIcons[col] = im
	return im
}

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
	col, label, ring := uint32(0xe02828), "¡RIIING!", color.RGBA{255, 90, 80, 255}
	if k == 2 {
		col, label, ring = 0xf4c820, "¡TRIIIN!", color.RGBA{255, 220, 60, 255}
	}
	im := phoneIcon(col)
	t := g.time
	jump := math.Abs(math.Sin(t*11)) * 9
	ang := math.Sin(t*26) * 0.38
	cx, cy := float64(screenW)-52, 52-jump
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(2.4, 2.4)
	op.GeoM.Rotate(ang)
	op.GeoM.Translate(cx, cy)
	dst.DrawImage(im, op)
	// sound arcs on both sides, blinking
	for i := 0; i < 3; i++ {
		if int(t*10+float64(i))%3 == 0 {
			continue
		}
		r := 30 + float64(i)*7
		a := ring
		a.A = uint8(255 - i*60)
		for _, side := range []float64{-1, 1} {
			for dy := -9.0; dy <= 9; dy += 3 {
				fillRect(dst, cx+side*(r-math.Abs(dy)*0.5)-1, cy+dy-1, 3, 3, a)
			}
		}
	}
	g.bigText(dst, label, cx, 80, ring, color.RGBA{40, 0, 0, 255}, 1.5)
}

package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// The T-Rex, painted by code (no image files): a chunky, friendly tyrannosaur with a big head and big white eyes, an
// orange-brown hide with dark tiger stripes, a cream belly and jaw, a pink mouth full of teeth and dark claws.
//
// One buffer of 160x92 (b.ox, b.oy = 22, 6 so the head has room to bend down on the left), feet at y = 80:
//
//	0 walk (right foot forward)   1 head down, jaws open    2 head down, biting   3 head up, swallowing
//	4 head up, roaring            5 walk (left foot forward) 10 head up, chewing
//	6, 7 seen from the front (walking towards the camera)    8, 9 seen from behind (walking away)
//
// The side views look LEFT; the game mirrors them.
// ---------------------------------------------------------------------------

const (
	rexPoseWalkA = 0
	rexPoseWalkB = 5
	rexPoseChew  = 10
	rexPoseFront = 6 // 6, 7
	rexPoseBack  = 8 // 8, 9
	rexPoses     = 11
)

var (
	rxOrange = c(0xb8652c)
	rxLight  = c(0xcf7d3a)
	rxStripe = c(0x6a3420)
	rxBelly  = c(0xe6c28c)
	rxBellyD = c(0xc9a06a)
	rxMouth  = c(0x8e3040)
	rxTongue = c(0xdc7080)
	rxTeeth  = c(0xf8f2de)
	rxClaw   = c(0x453840)
	rxEyeW   = c(0xffffff)
	rxPupil  = c(0x14101a)
	// the scrap of meat in the mouth (poses 2, 3, 10); the cake scene paints it in the tones of the mouth
	// (rexscene.go, rexScenePropImg) so it does not look like a red blob on the cake
	rxScrap    = c(0xb01818)
	rxScrapBit = c(0xe0b088)
)

// rexHeadPose: where the head of a side pose is (art coordinates), how far it bends down (rad) and how open the
// jaws are. The cinematics read it too (rexscene.go) to put the party hat on the head of the very same sprite.
type rexHeadPose struct{ hx, hy, down, jaw float64 }

var rexHeadPoses = [rexPoses]rexHeadPose{0: {26, 18, 0.05, 0.55}, 1: {13, 54, 1.0, 1}, 2: {13, 57, 1.0, 0}, 3: {30, 11, -0.5, 0},
	4: {31, 13, -0.6, 1}, 5: {26, 19, 0.05, 0.55}, 10: {30, 12, -0.45, 0.5}}

// rexClaws: three dark claws under a foot at (x, y); dir -1/+1 spreads them to one side, 0 downwards.
func rexClaws(b *pixBuf, x, y float64, dir int) {
	for k := -1; k <= 1; k++ {
		cx := int(x) + k*3
		b.rect(cx, int(y), 2, 3, rxClaw)
		if dir != 0 {
			b.set(cx+dir, int(y)+3, rxClaw)
		}
	}
}

// rexBands paints dark stripes that follow a line: n bands from (x0,y0) to (x1,y1), each a short dash of length ln.
func rexBands(b *pixBuf, x0, y0, x1, y1 float64, n int, dx, dy, r0, r1 float64) {
	for i := 0; i < n; i++ {
		t := (float64(i) + 0.5) / float64(n)
		x, y := x0+(x1-x0)*t, y0+(y1-y0)*t
		capsule(b, x, y, x+dx, y+dy, r0, r1, rxStripe)
	}
}

func drawRexArt(b *pixBuf, v int) {
	switch {
	case v == rexPoseFront || v == rexPoseFront+1:
		drawRexFront(b, v-rexPoseFront)
	case v == rexPoseBack || v == rexPoseBack+1:
		drawRexBack(b, v-rexPoseBack)
	default:
		drawRexSide(b, v)
	}
}

func drawRexSide(b *pixBuf, v int) {
	col, dark := rxOrange, shade(rxOrange, 0.62)
	b.ellipse(62, 83, 40, 3.4, color.RGBA{0, 0, 0, 90})
	// the step: which foot is forward and which one is in the air
	nearOff, farOff, nearUp, farUp := 0.0, 0.0, 0.0, 0.0
	switch v {
	case 0:
		nearOff, farOff, farUp = -6, -6, 3
	case 5:
		nearOff, farOff, nearUp = 5, 5, 3
	}
	// tail: long, tapering, striped, cream underneath
	capsule(b, 80, 48, 100, 56, 11, 7.5, dark)
	capsule(b, 98, 56, 122, 62, 7.5, 1.4, dark)
	capsule(b, 80, 46.5, 100, 54.5, 9.6, 6.2, col)
	capsule(b, 98, 54.5, 122, 61, 6.2, 0.8, col)
	capsule(b, 82, 51, 100, 59, 4, 3, rxBelly)
	capsule(b, 100, 59, 112, 62, 2.4, 1.2, rxBelly)
	for k := 0; k < 6; k++ {
		x := 86 + float64(k)*6.2
		y := 46 + float64(k)*2.5
		capsule(b, x, y-4-float64(5-k)*0.5, x+1.5, y+4-float64(k)*0.3, 1.1, 0.6, rxStripe)
	}
	// far leg
	capsule(b, 82, 56, 84+farOff, 76-farUp, 8, 4.6, dark)
	b.ellipse(86+farOff, 79-farUp, 9, 3.4, dark)
	rexClaws(b, 80+farOff, 79-farUp, -1)
	// near leg: a big thigh and a thinner shin
	b.ellipse(68, 58, 13, 15, shade(col, 0.82))
	b.ellipse(67, 56, 11, 12, shade(col, 0.95))
	capsule(b, 64, 66, 60+nearOff, 76-nearUp, 6.6, 4.6, shade(col, 0.8))
	b.ellipse(57+nearOff, 79-nearUp, 11, 3.8, shade(col, 0.72))
	b.ellipse(56+nearOff, 78-nearUp, 9, 2.6, shade(col, 0.9))
	rexClaws(b, 49+nearOff, 79-nearUp, -1)
	// torso
	dinoBody(b, 60, 43, 31, 18, col)
	b.ellipse(52, 52, 23, 8.5, rxBelly)
	for k := 0; k < 3; k++ { // belly folds
		capsule(b, 36+float64(k)*3, 51+float64(k)*2.4, 62+float64(k)*3, 52+float64(k)*2.4, 0.5, 0.5, rxBellyD)
	}
	rexBands(b, 38, 27, 84, 32, 9, 3, 11, 1.7, 0.7)
	rexBands(b, 44, 29, 78, 33, 7, 2, 6, 1.1, 0.5)
	rexBands(b, 62, 40, 86, 44, 5, 2, 7, 1.2, 0.5) // flank
	// tiny arms with two claws
	capsule(b, 38, 47, 31, 57, 3, 1.8, shade(col, 0.7))
	capsule(b, 38, 46, 32, 56, 2.2, 1.4, col)
	b.rect(28, 57, 2, 3, rxClaw)
	b.rect(31, 58, 2, 3, rxClaw)

	ps := rexHeadPoses[v%rexPoses]
	th := -ps.down
	cs, sn := math.Cos(th), math.Sin(th)
	at := func(x, y float64) (float64, float64) { return ps.hx + x*cs - y*sn, ps.hy + x*sn + y*cs }
	bx, by := at(14, 6)
	capsule(b, 46, 38, bx, by, 12.5, 10.5, dark) // neck
	capsule(b, 46, 36.5, bx, by-1.4, 11.5, 9.5, col)
	cap2 := func(x0, y0, x1, y1, r0, r1 float64, cc color.RGBA) {
		ax, ay := at(x0, y0)
		bx2, by2 := at(x1, y1)
		capsule(b, ax, ay, bx2, by2, r0, r1, cc)
	}
	ell := func(x, y, rx, ry float64, cc color.RGBA) {
		ax, ay := at(x, y)
		b.ellipse(ax, ay, rx, ry, cc)
	}
	jx := -19.0
	jy := 10 + 14*ps.jaw
	if ps.jaw > 0.25 { // the open mouth: dark inside, a pink tongue
		cap2(-3, 6, jx, jy-2, 5.4, 3, rxMouth)
		cap2(-3, 6.5, jx+1, jy-3, 3.6, 2, rxTongue)
	}
	cap2(-2, 10, jx, jy, 4.4, 3.2, shade(rxBelly, 0.82)) // lower jaw, cream
	cap2(-2, 9.2, jx, jy-0.6, 3.5, 2.5, rxBelly)
	ell(0, 0, 17, 12, dark) // skull
	ell(0, -1, 16, 11, col)
	ell(-3, -5, 9, 3.4, shade(col, 1.18))
	cap2(-4, 1, -21, 3, 9, 6.8, dark) // broad snout
	cap2(-4, -0.4, -21, 1.6, 8, 5.8, col)
	for k := 0; k < 2; k++ { // dark stripes on the cheek and over the eye
		cap2(2+float64(k)*4, -9, 3+float64(k)*4, -3, 0.9, 0.5, rxStripe)
	}
	cap2(4, 4, 9, 7, 0.7, 0.5, rxStripe)
	for k := 0; k < 6; k++ { // upper teeth (triangles) and lower ones
		ax, ay := at(-9-float64(k)*2.5, 9)
		b.rect(int(ax), int(ay), 1, 3, rxTeeth)
		b.set(int(ax), int(ay)+3, rxTeeth)
		if ps.jaw > 0.25 {
			bx3, by3 := at(-9-float64(k)*2.5, jy-3.6-float64(k)*0.15)
			b.rect(int(bx3), int(by3), 1, 2, rxTeeth)
		}
	}
	// the big white eye
	ex, ey := at(3, -5)
	b.ellipse(ex, ey, 3.5, 3.6, rxEyeW)
	b.rect(int(ex)-2, int(ey)-1, 3, 3, rxPupil)
	b.set(int(ex)-2, int(ey)-1, rxEyeW)
	nx, ny := at(-21, -1)
	b.set(int(nx), int(ny), dark)
	b.set(int(nx)+1, int(ny), dark)
	if v == 3 || v == 2 || v == rexPoseChew { // a scrap in the mouth
		qx, qy := at(-16, 9)
		b.rect(int(qx)-2, int(qy), 5, 3, rxScrap)
		b.rect(int(qx)+1, int(qy)+2, 2, 3, rxScrapBit)
	}
	if v == 3 { // the throat swells as it swallows
		tx, ty := at(8, 14)
		b.ellipse(tx, ty, 4.5, 4, shade(rxBelly, 1.08))
	}
}

// rexLegsFront: the two legs seen from the front or the back; the foot that is up alternates with the step.
func rexLegsFront(b *pixBuf, step int, back bool) {
	dark := shade(rxOrange, 0.62)
	for i, hx := range []float64{46, 70} {
		up := 0.0
		if (i+step)%2 == 0 {
			up = 3
		}
		capsule(b, hx, 56, hx+float64(2*i-1)*2, 76-up, 10, 6.5, dark)
		capsule(b, hx, 55, hx+float64(2*i-1)*2, 75-up, 8.6, 5.6, shade(rxOrange, 0.88))
		b.ellipse(hx+float64(2*i-1)*3, 79-up, 10, 3.6, shade(rxOrange, 0.7))
		b.ellipse(hx+float64(2*i-1)*3, 78-up, 8.4, 2.6, shade(rxOrange, 0.9))
		if !back {
			rexClaws(b, hx+float64(2*i-1)*3-1, 79-up, 0)
		}
	}
}

func drawRexFront(b *pixBuf, step int) {
	col, dark := rxOrange, shade(rxOrange, 0.62)
	b.ellipse(58, 83, 28, 3.4, color.RGBA{0, 0, 0, 90})
	rexLegsFront(b, step, false)
	dinoBody(b, 58, 47, 24, 23, col)
	b.ellipse(58, 54, 13, 17, rxBelly)
	for k := 0; k < 4; k++ { // belly folds
		capsule(b, 51, 47+float64(k)*4.4, 65, 47+float64(k)*4.4, 0.5, 0.5, rxBellyD)
	}
	for _, sgn := range []float64{-1, 1} { // stripes on the shoulders and flanks
		for k := 0; k < 4; k++ {
			y := 34 + float64(k)*8
			capsule(b, 58+sgn*(21-float64(k)*1.5), y, 58+sgn*(15-float64(k)*0.6), y+3, 1.3, 0.6, rxStripe)
		}
		capsule(b, 58+sgn*18, 52, 58+sgn*14, 61, 3.4, 2, shade(col, 0.7)) // arm
		capsule(b, 58+sgn*18, 51, 58+sgn*14.5, 59.5, 2.5, 1.5, col)
		b.rect(int(58+sgn*14)-1+int((1-sgn)/2*-1), 61, 2, 3, rxClaw)
		b.rect(int(58+sgn*17)-1, 61, 2, 3, rxClaw)
	}
	// the head, big, looking at us with the mouth open
	b.ellipse(58, 22, 20, 15, dark)
	b.ellipse(58, 21, 19, 14, col)
	b.ellipse(58, 15, 12, 4, shade(col, 1.18))
	capsule(b, 58, 8, 58, 14, 1, 0.6, rxStripe)
	for _, sgn := range []float64{-1, 1} {
		capsule(b, 58+sgn*6, 8, 58+sgn*6.5, 12, 0.9, 0.5, rxStripe)
		capsule(b, 58+sgn*12, 10, 58+sgn*13, 14, 0.8, 0.5, rxStripe)
	}
	b.ellipse(58, 28, 14, 9, shade(col, 0.86)) // snout
	b.ellipse(58, 38, 12, 5, shade(rxBelly, 0.9))
	b.ellipse(58, 38, 11, 4, rxBelly) // lower jaw
	b.ellipse(58, 33, 10.5, 6.2, rxMouth)
	b.ellipse(58, 36, 6.5, 3, rxTongue)
	for k := -5; k <= 5; k++ { // teeth
		if k%2 == 0 {
			b.rect(58+k*2-1, 28, 2, 3, rxTeeth)
		}
		if k%2 != 0 {
			b.rect(58+k*2-1, 37, 2, 2, rxTeeth)
		}
	}
	b.set(55, 27, shade(col, 0.5)) // nostrils
	b.set(61, 27, shade(col, 0.5))
	for _, sgn := range []float64{-1, 1} { // the big white eyes
		ex := 58 + sgn*9
		b.ellipse(ex, 17, 3.9, 4.2, rxEyeW)
		b.rect(int(ex-sgn*1)-1, 17, 3, 3, rxPupil)
		b.set(int(ex-sgn*1)-1, 17, rxEyeW)
	}
}

func drawRexBack(b *pixBuf, step int) {
	col, dark := rxOrange, shade(rxOrange, 0.62)
	b.ellipse(58, 83, 28, 3.4, color.RGBA{0, 0, 0, 90})
	rexLegsFront(b, step, true)
	dinoBody(b, 58, 47, 24, 23, col)
	// back stripes: bars across the back that shrink down to the tail
	for k := 0; k < 8; k++ {
		y := 29 + float64(k)*4.4
		w := 15 - float64(k)*1.1
		capsule(b, 58-w, y+1, 58, y, 1.1, 0.8, rxStripe)
		capsule(b, 58, y, 58+w, y+1, 0.8, 1.1, rxStripe)
	}
	capsule(b, 58, 26, 58, 68, 0.8, 0.8, shade(rxStripe, 0.8))
	// head seen from behind, a bit smaller; the white of one eye peeps out on each side
	b.ellipse(58, 21, 16, 13, dark)
	b.ellipse(58, 20, 15, 12, col)
	b.ellipse(58, 15, 9, 3.4, shade(col, 1.18))
	for k := 0; k < 4; k++ {
		capsule(b, 52+float64(k)*4, 11, 52.5+float64(k)*4, 17, 0.8, 0.5, rxStripe)
	}
	b.rect(41, 18, 2, 3, rxEyeW)
	b.rect(74, 18, 2, 3, rxEyeW)
	// tail toward us, tapering to a point; it sways with the step
	sway := float64(step*2 - 1)
	capsule(b, 58, 58, 58+sway, 72, 10, 5, dark)
	capsule(b, 58, 57, 58+sway, 71, 8.6, 4.2, col)
	capsule(b, 58+sway, 71, 58+sway*2, 84, 4.4, 0.7, col)
	capsule(b, 58, 62, 58+sway, 74, 2, 1.4, rxBelly)
	for k := 0; k < 5; k++ {
		y := 63 + float64(k)*4.2
		w := 6.5 - float64(k)*1.2
		capsule(b, 58+sway*float64(k)/4-w, y, 58+sway*float64(k)/4+w, y, 0.9, 0.9, rxStripe)
	}
}

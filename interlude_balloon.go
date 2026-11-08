package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The balloon interlude (after party 15): "PRRRRR". A hot-air balloon, a dawn sky and a very short flight:
//
//	1. the knife cuts the mooring rope and the balloon rises through the clouds (in 2P the red one
//	   wanted to untie the knot),
//	2. a bird pecks the envelope: POP, the air hisses out and the balloon starts to fall,
//	3. they throw everything out to lose weight (a sandbag, the cooler, the anchor) and brush the tops of the trees,
//	4. the balloon is a flat rag now, and it takes off like a party balloon you let go of: zigzag,
//	   loops and farts across the sky, the two of them holding on, towards the next party.
//
// About 13 seconds. The rig (envelope, ropes, basket and passengers) is drawn on its own canvas and then placed,
// tilted and moved as a whole. The passengers never leave the basket (interlude_balloon_test.go).
// ---------------------------------------------------------------------------

// the timeline, in seconds
const (
	bnCut      = 1.5  // the knife cuts the rope
	bnPeak     = 4.8  // it is as high as it gets
	bnPeck     = 4.9  // the bird pecks the envelope
	bnFall0    = 5.4  // it starts to lose height
	bnThrow0   = 6.0  // first thing overboard
	bnThrow1   = 6.8  // second
	bnThrow2   = 7.6  // third
	bnTrees    = 8.3  // the treetops
	bnZip      = 9.3  // it flies off, deflated
	bnFade     = 12.0 // fade to black
	balloonLen = 13.2
)

// the rig, in its own canvas: the basket stands on (bnRX, bnRY)
const (
	bnRX     = 480.0 // centre of the basket
	bnRY     = 430.0 // its bottom
	bnBW     = 200.0 // width of the basket
	bnBH     = 70.0  // height of the basket
	bnOY     = 40.0  // the rig stands this far down on the screen (the basket sits on the ground at y = 470)
	bnGround = bnRY + bnOY
)

// where each passenger stands, in pixels from the centre of the basket (negative = left)
var bnSeatDX = [2]float64{-48, 46}

// balloonAlt: how high above the field the balloon is (pixels).
func balloonAlt(t float64) float64 {
	switch {
	case t < bnCut:
		return 0
	case t < bnPeak:
		return 520 * introEase((t-bnCut)/(bnPeak-bnCut))
	case t < bnFall0:
		return 520
	case t < bnTrees+0.1:
		u := (t - bnFall0) / (bnTrees + 0.1 - bnFall0)
		return 520 - 480*math.Pow(u, 1.4)
	}
	return 40
}

// balloonDeflate: how flat the envelope is, 0 full ... 1 a rag.
func balloonDeflate(t float64) float64 {
	if t < bnPeck {
		return 0
	}
	return introEase((t - bnPeck) / (bnZip - bnPeck))
}

// balloonSeat: the feet of passenger k, in the rig's own coordinates (basket centre plus the seat).
func balloonSeat(k int) (x, y float64) { return bnRX + bnSeatDX[k], bnRY - 16 }

// balloonPose: what passenger k does at time t: feet (in the rig), animation, phase, facing, lean and hop. He never
// leaves the basket: at most 12 px from his seat and a small hop.
func balloonPose(t float64, k int, two bool) (x, y float64, anim int, ph float64, dir Dir, lean, hop float64) {
	x, y = balloonSeat(k)
	dir = DirRight
	anim = AnimIdle
	throwing := func(at float64) float64 { return sceneWindow(t, at-0.4, at) * (1 - sceneWindow(t, at, at+0.35)) }
	first := !two || k == 0
	switch {
	case t < bnCut-0.5:
		if !first { // the red one fumbles with the knot
			anim, ph = AnimPanic, t*7
		}
	case t < bnCut+0.4:
		if first {
			anim, ph = AnimStab, (t-bnCut+0.5)*9
		} else {
			anim, ph = AnimPanic, t*7
		}
	case t < bnPeck-0.5:
		anim, ph = AnimCheer, t*8
		hop = math.Abs(math.Sin(t*7)) * 4
		if !first {
			anim = AnimWave
		}
	case t < bnThrow0-0.4: // the bird
		anim, ph = AnimPoint, t*6
		if !first {
			anim, ph = AnimPanic, t*10
		}
	case t < bnTrees-0.3: // overboard with everything
		anim, ph = AnimPanic, t*9
		for i, at := range []float64{bnThrow0, bnThrow1, bnThrow2} {
			if k == balloonThrower(i, two) { // 1P: the blue one throws it all; 2P: they take turns
				if w := throwing(at); w > 0 || sceneIn(t, at, at+0.4) {
					anim, ph = AnimIdle, 0
					lean = -0.25*w + 0.3*sceneWindow(t, at, at+0.2)*(1-sceneWindow(t, at+0.2, at+0.4))
					hop = 3 * w
				}
			}
		}
	case t < bnZip:
		anim, ph = AnimCrouch, t*6 // ducking the branches
	default:
		anim, ph = AnimPanic, t*12
		lean = 0.08 * math.Sin(t*30)
	}
	y -= hop
	return
}

// balloonThrower: who throws item i (alone in 1P, in turns in 2P).
func balloonThrower(i int, two bool) int {
	if two {
		return i % 2
	}
	return 0
}

// balloonRig: where the rig is on the screen and how it is tilted: translation, rotation.
func balloonRig(t float64) (tx, ty, rot float64) {
	ty = bnOY
	if t < bnCut { // tugging at the rope
		return 0, ty, 0.01 * math.Sin(t*5)
	}
	// breathing and sway in the air
	h := sceneWindow(t, bnCut, bnPeck)
	tx = 6 * math.Sin(t*0.9) * h
	ty += 4 * math.Sin(t*1.7) * h
	rot = 0.03 * math.Sin(t*1.3) * h
	if t > bnPeck { // the hole: it wobbles as it loses air
		w := sceneWindow(t, bnPeck, bnZip)
		rot += 0.06 * math.Sin(t*6) * w
		ty -= 22 * w * (1 - sceneWindow(t, bnThrow2, bnTrees)) // lighter: it holds up a bit
	}
	if sceneIn(t, bnTrees, bnTrees+0.9) { // the branches
		ty += math.Round(math.Sin(t*50) * 3)
		tx += math.Round(math.Cos(t*43) * 3)
	}
	if t >= bnZip { // let go like a party balloon: zigzag, loops and farts
		// it is still on the screen when the fade starts and has flown out of it before the end of the scene
		u := (t - bnZip) / 4.6
		wob := (t - bnZip) * 6.7 // the zigzag keeps its own pace
		tx = 880 * math.Pow(u, 1.5)
		ty = bnOY - 50*u - 70*math.Sin(wob)*(1-0.3*u) + 150*u*u
		rot = 1.0 * math.Sin(wob+0.3)
	}
	return
}

// ---- update ------------------------------------------------------------------------------------------

func (g *Game) updateBalloon(in *interState) {
	if !in.music {
		in.music = true
		g.sceneMusic(sceneInterA, ambInterA)
	}
	t := in.t
	v := math.Min(1, t/1.5)
	if t > bnFade-1.5 {
		v *= 1 - sceneWindow(t, bnFade-1.5, balloonLen-0.3)
	}
	g.snd.SetSceneVolume("base", 0.55*v)
	once := func(key string, at float64, name string) {
		if t >= at && !in.did[key] {
			in.did[key] = true
			g.snd.Play(name)
		}
	}
	once("cut", bnCut, "spark")
	once("up", bnCut+0.3, "boom")
	once("peck", bnPeck, "alert")
	once("t0", bnThrow0, "thud")
	once("t1", bnThrow1, "thud")
	once("t2", bnThrow2, "thud")
	once("trees", bnTrees, "thud")
	once("zip", bnZip, "boom")
	once("zip2", bnZip+1.2, "splash")
	once("zip3", bnZip+2.2, "spark")
	if t >= balloonLen {
		g.finishInterlude()
	}
}

// ---- drawing -----------------------------------------------------------------------------------------

func (g *Game) drawBalloonScene(dst *ebiten.Image, in *interState) {
	if in.canvas == nil {
		in.canvas = ebiten.NewImage(screenW, screenH)
	}
	if in.rig == nil {
		in.rig = ebiten.NewImage(screenW, screenH) // the rig, on its own
	}
	c, rig := in.canvas, in.rig
	t := in.t
	alt := balloonAlt(t)
	g.balloonWorld(c, t, alt)
	// the rig on its canvas, then placed
	rig.Clear()
	g.balloonRigDraw(rig, in)
	tx, ty, rot := balloonRig(t)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-bnRX, -bnRY+bnBH/2)
	op.GeoM.Rotate(rot)
	op.GeoM.Translate(bnRX+tx, bnRY-bnBH/2+ty)
	c.DrawImage(rig, op)
	g.balloonFront(c, in, alt)
	dst.DrawImage(c, nil)
	g.balloonCaption(dst, in)
}

// balloonWorld: the dawn sky, the clouds and the countryside, which slides down as the balloon rises.
func (g *Game) balloonWorld(c *ebiten.Image, t, alt float64) {
	for y := 0.0; y < screenH; y += 4 {
		fillRect(c, 0, y, screenW, 4, mixc(rgb(0x5a9cf0), rgb(0xffd8b0), math.Pow(y/screenH, 1.4)))
	}
	sy := 300 + alt*0.35
	filledEllipse(c, 220, sy, 84, 84, rgba(0xfff0b0, 70))
	filledEllipse(c, 220, sy, 46, 46, rgb(0xfff4c8))
	D := 36 * t         // the wind
	G := bnGround + alt // the ground line on the screen
	if G < screenH+120 {
		// far mountains
		for k := 0; k < 9; k++ {
			mx := math.Round(math.Mod(float64(k)*170-D*0.1+3000, screenW+340)) - 170
			h := 90 + jpHash(k, 3, 201)*70
			for r := 0.0; r < h; r += 3 {
				w := (h - r) * 1.3
				fillRect(c, mx-w/2, G-r-150+alt*0.0, w, 3, mixc(rgb(0x7a74b8), rgb(0xa89ad0), r/h))
			}
		}
		// fields
		for y := G - 40; y < screenH; y += 4 {
			k := (y - (G - 40)) / 160
			fillRect(c, 0, y, screenW, 4, mixc(rgb(0x6cc05a), rgb(0x3a8a3a), math.Min(1, k)))
		}
		for k := 0; k < 10; k++ { // patchwork
			fx := math.Round(math.Mod(float64(k)*150-D*0.5+4000, screenW+300)) - 150
			fy := G - 30 + float64(k%3)*34
			col := []uint32{0xd8c850, 0x9ad060, 0xc89848}[k%3]
			fillRect(c, fx, fy, 120, 22, rgba(col, 200))
			for ly := 0.0; ly < 22; ly += 6 {
				fillRect(c, fx, fy+ly, 120, 1, rgba(0x000000, 40))
			}
		}
		// a windmill and a village
		wx := math.Round(math.Mod(640-D*0.6+4000, screenW+400)) - 200
		fillRect(c, wx-14, G-100, 28, 70, rgb(0xe8dcc8))
		fillRect(c, wx-18, G-110, 36, 14, rgb(0xb84a3a))
		ang := t * 1.6
		for q := 0; q < 4; q++ {
			a := ang + float64(q)*math.Pi/2
			for s := 8.0; s < 56; s += 3 {
				fillRect(c, wx+math.Cos(a)*s-2, G-100+math.Sin(a)*s-2, 4, 4, rgb(0xf4efe4))
			}
		}
		for k := 0; k < 4; k++ {
			hx := math.Round(math.Mod(float64(k)*70+160-D*0.6+4000, screenW+300)) - 100
			fillRect(c, hx, G-36, 40, 28, rgb(0xf0e4cc))
			for r := 0.0; r < 14; r++ {
				fillRect(c, hx-4+r, G-50+r, 48-2*r, 1, rgb(0xc84a3a))
			}
			fillRect(c, hx+16, G-22, 8, 14, rgb(0x6a4a2c))
		}
		// trees
		for k := 0; k < 12; k++ {
			tx := math.Round(math.Mod(float64(k)*88+30-D*0.9+4000, screenW+200)) - 100
			ty := G + 12 + float64(k%4)*18
			fillRect(c, tx-3, ty-30, 6, 30, rgb(0x5a3a22))
			filledEllipse(c, tx, ty-44, 20, 24, rgb(0x2e7a34))
			filledEllipse(c, tx-6, ty-50, 12, 14, rgb(0x4a9a48))
		}
	}
	// clouds: small ones up in the sky, and the thick layer the balloon goes through
	for k := 0; k < 6; k++ {
		cx := math.Round(math.Mod(float64(k)*230+60-D*0.3+3000, screenW+300)) - 150
		cy := math.Mod(60+float64(k)*77+alt*0.5, 620) - 70
		filledEllipse(c, cx, cy, 70, 16, rgba(0xffffff, 235))
		filledEllipse(c, cx+30, cy-12, 38, 14, rgba(0xffffff, 235))
		filledEllipse(c, cx-34, cy-6, 30, 11, rgba(0xf4f6ff, 235))
	}
	band := bnGround + alt - 330 // the top of the thick layer of clouds, 330 px over the field (and 100 px thick)
	if band < screenH+100 && band > -200 {
		for y := math.Max(band, 0); y < band+110 && y < screenH; y += 4 {
			fillRect(c, 0, y, screenW, 4, rgba(0xfafcff, uint8(math.Min(255, 60+(y-band)*2))))
		}
		for k := 0; k < 9; k++ {
			cx := math.Round(math.Mod(float64(k)*150-D*0.4+3000, screenW+260)) - 130
			filledEllipse(c, cx, band+6, 90, 34, rgb(0xfafcff))
			filledEllipse(c, cx+50, band-8, 56, 28, rgb(0xffffff))
			filledEllipse(c, cx-50, band-2, 50, 24, rgb(0xf0f4ff))
		}
	}
}

// balloonEnvelope: the balloon proper, in the rig's coordinates; d is how flat it is.
func (g *Game) balloonEnvelope(c *ebiten.Image, t, d float64) {
	neckY := bnRY - bnBH - 148.0 // the skirt, over the heads
	ry := 120 * (1 - 0.66*d)
	rx := 118 * (1 - 0.3*d)
	top := neckY - 2*ry
	sag := d * 26 * math.Sin(t*5) // the empty top flops to the side
	cols := [8]uint32{0xd8302a, 0xf4d03a, 0x2a6ad8, 0xffffff, 0xd8302a, 0xf4d03a, 0x2a6ad8, 0xffffff}
	for y := top; y < neckY; y += 3 {
		v := (y - top) / (neckY - top)
		var hw float64
		if v < 0.62 {
			q := (0.62 - v) / 0.62
			hw = rx * math.Sqrt(math.Max(0, 1-q*q))
		} else {
			hw = rx + (40-rx)*math.Pow((v-0.62)/0.38, 1.5)
		}
		cx := bnRX + sag*(1-v)*(1-v)
		for gI := 0; gI < 8; gI++ {
			x0 := cx - hw + 2*hw*float64(gI)/8
			w := 2 * hw / 8
			col := rgb(cols[gI])
			if gI >= 5 {
				col = shade(col, 0.82)
			}
			if gI >= 7 {
				col = shade(col, 0.82)
			}
			fillRect(c, math.Round(x0), y, math.Round(w)+1, 3, col)
			if gI > 0 {
				fillRect(c, math.Round(x0), y, 1, 3, rgba(0x000000, 50))
			}
		}
	}
	fillRect(c, bnRX-42, neckY-2, 84, 8, rgb(0x6a4a2c)) // the skirt
	// the hole the bird made
	if t >= bnPeck {
		hx, hy := bnRX+rx*0.55+sag*0.2, top+ry*0.7
		filledEllipse(c, hx, hy, 10, 7, rgb(0x2a1a18))
		for i := 0; i < 9; i++ { // air hissing out
			age := math.Mod(t*2.2+float64(i)/9, 1)
			filledEllipse(c, hx+14+age*70, hy+age*12+math.Sin(float64(i)*4)*6, 5+age*9, 4+age*6, rgba(0xffffff, uint8(200*(1-age))))
		}
	}
}

func (g *Game) balloonRigDraw(c *ebiten.Image, in *interState) {
	t := in.t
	d := balloonDeflate(t)
	g.balloonEnvelope(c, t, d)
	bx, by := bnRX-bnBW/2, bnRY-bnBH
	// ropes: from the corners of the basket to the skirt
	neckY := bnRY - bnBH - 148.0
	rope := func(x0, y0, x1, y1 float64) {
		for s := 0.0; s <= 1; s += 1.0 / 40 {
			fillRect(c, math.Round(x0+(x1-x0)*s), math.Round(y0+(y1-y0)*s), 2, 2, rgb(0x4a3220))
		}
	}
	rope(bx+4, by, bnRX-40, neckY+6)
	rope(bx+bnBW-4, by, bnRX+40, neckY+6)
	rope(bnRX-20, by, bnRX-22, neckY+6)
	rope(bnRX+20, by, bnRX+22, neckY+6)
	// the burner and its flame, between the two of them
	flame := 1.0
	if t > bnPeck {
		flame = 0.35 + 0.3*math.Sin(t*12)
	}
	if t >= bnCut-1.2 && t < bnZip {
		fh := 70 * flame * (0.8 + 0.2*math.Sin(t*30))
		filledEllipse(c, bnRX, by-34-fh/2, 9, fh/2, rgb(0xff8a1c))
		filledEllipse(c, bnRX, by-30-fh/3, 5, fh/3, rgb(0xffe060))
	}
	fillRect(c, bnRX-5, by-34, 10, 34, rgb(0x6a6a74)) // the burner
	// the back wall of the basket, the passengers, then the front wall
	fillRect(c, bx, by, bnBW, bnBH, rgb(0x3a2412))
	nb := 1
	if in.two {
		nb = 2
	}
	for k := 0; k < nb; k++ {
		px, py, anim, ph, dir, lean, hop := balloonPose(t, k, in.two)
		cs := in.hero
		tint := [3]float32{1, 1, 1}
		if in.two {
			cs = in.heroes[k]
			tint = sceneTint(k, in.heroes[0], in.heroes[1])
		}
		g.sceneChar(c, cs, anim, dir, ph, px, py+hop, interScale, 1, hop, lean, tint)
		if sceneIn(t, bnCut-0.6, bnCut+0.3) && (k == 0 || !in.two) { // the knife flash
			g.drawKnifeFlash(c, px-60, py-30)
		}
		// what they throw out is in their hands until it goes
		for i, at := range []float64{bnThrow0, bnThrow1, bnThrow2} {
			if k == balloonThrower(i, in.two) && sceneIn(t, at-1.2, at) {
				balloonItem(c, i, px+30, py-70-hop)
			}
		}
	}
	// the front wall of the basket (wicker) over their feet
	fillRect(c, bx, by+16, bnBW, bnBH-16, rgb(0xa8743c))
	for yy := by + 16; yy < bnRY; yy += 8 {
		fillRect(c, bx, yy, bnBW, 2, rgb(0x7a5028))
	}
	for xx := bx; xx < bx+bnBW; xx += 12 {
		fillRect(c, xx, by+16, 2, bnBH-16, rgb(0x8c5a2a))
	}
	fillRect(c, bx-4, by+10, bnBW+8, 8, rgb(0xc89458)) // the rim
	// four sandbags hanging outside
	if t < bnThrow0 {
		for q := 0; q < 4; q++ {
			sx := bx + 8 + float64(q)*(bnBW-40)/3
			filledEllipse(c, sx+10, by+bnBH-4, 11, 9, rgb(0xb89a5a))
			fillRect(c, sx+8, by+bnBH-16, 4, 6, rgb(0x6a4a2c))
		}
	}
	// the rope that moors it, until it is cut; then the piece that dangles
	if t < bnCut {
		for s := 0.0; s <= 1; s += 1.0 / 30 {
			fillRect(c, math.Round(bx+4-90*s), math.Round(bnRY-14+8*s), 3, 3, rgb(0xd8c090))
		}
	} else {
		for s := 0.0; s < 1; s += 1.0 / 20 {
			fillRect(c, math.Round(bx+4+math.Sin(t*4+s*3)*6*s), math.Round(bnRY-14+s*70), 3, 3, rgb(0xd8c090))
		}
	}
	// the bird
	if t >= bnPeck-1.0 && t < bnPeck+1.6 {
		g.balloonBird(c, t)
	}
}

// balloonItem draws the things they throw: 0 a sandbag, 1 a cooler, 2 an anchor.
func balloonItem(c *ebiten.Image, kind int, x, y float64) {
	x, y = math.Round(x), math.Round(y)
	switch kind {
	case 0:
		filledEllipse(c, x, y, 16, 13, rgb(0xb89a5a))
		fillRect(c, x-3, y-18, 6, 7, rgb(0x6a4a2c))
	case 1:
		fillRect(c, x-18, y-10, 36, 24, rgb(0x2a6ad8))
		fillRect(c, x-18, y-10, 36, 8, rgb(0xf4f4f8))
		fillRect(c, x-6, y-14, 12, 4, rgb(0x9a9aa4))
	default:
		fillRect(c, x-2, y-18, 4, 30, rgb(0x4a4a58)) // an anchor
		fillRect(c, x-10, y-18, 20, 3, rgb(0x4a4a58))
		fillRect(c, x-14, y+8, 28, 3, rgb(0x4a4a58))
		fillRect(c, x-14, y+2, 4, 8, rgb(0x4a4a58))
		fillRect(c, x+10, y+2, 4, 8, rgb(0x4a4a58))
	}
}

// balloonBird: a big white bird that flies in from the right, pecks the envelope, and leaves with a patch of cloth.
func (g *Game) balloonBird(c *ebiten.Image, t float64) {
	var x, y float64
	tx, ty := bnRX+118*0.55*0.9+20, bnRY-bnBH-148-2*120+120*0.7-10
	switch {
	case t < bnPeck:
		u := introEase((t - (bnPeck - 1.0)) / 1.0)
		x, y = tx+520*(1-u), ty-90*(1-u)
	case t < bnPeck+0.5:
		x, y = tx, ty+math.Round(math.Sin((t-bnPeck)*30)*6) // the pecking
	default:
		u := (t - bnPeck - 0.5) / 1.1
		x, y = tx+260*u, ty-40*u-60*u*u
	}
	x, y = math.Round(x), math.Round(y)
	w := math.Round(math.Sin(t*22) * 14)
	filledEllipse(c, x, y, 26, 14, rgb(0xf8f8fc))
	filledEllipse(c, x-24, y-8, 11, 10, rgb(0xf8f8fc))
	fillRect(c, x-46, y-10, 22, 7, rgb(0xf0a030)) // the beak
	fillRect(c, x-46, y-4, 18, 3, rgb(0xd88020))
	fillRect(c, x-26, y-12, 3, 3, rgb(0x10101a))
	fillRect(c, x-6, y-24-w, 22, 6, rgb(0xdcdce8)) // wings
	fillRect(c, x+8, y-18-w, 24, 5, rgb(0xcacad8))
	fillRect(c, x+24, y-2, 18, 6, rgb(0xdcdce8)) // tail
	if t >= bnPeck+0.5 {                         // the patch
		fillRect(c, x-52, y+2, 14, 12, rgb(0xd8302a))
	}
}

// balloonFront: what is in front of the rig: the things thrown overboard, the trees, the farts.
func (g *Game) balloonFront(c *ebiten.Image, in *interState, alt float64) {
	t := in.t
	// whiteout in the clouds
	if f := 1 - math.Abs(alt-280)/45; f > 0 && t > bnCut { // a short flash of cloud, not a long wash
		fillRect(c, 0, 0, screenW, screenH, rgba(0xfafcff, uint8(130*math.Min(1, f*1.5))))
	}
	// things thrown out: they leave the basket and fall
	for i, at := range []float64{bnThrow0, bnThrow1, bnThrow2} {
		if t < at {
			continue
		}
		u := t - at
		sx, sy := balloonSeat(balloonThrower(i, in.two))
		tx, ty, _ := balloonRig(at)
		x := sx + tx + 30 + []float64{170, 210, 140}[i]*u
		y := sy + ty - 70 - 260*u + 520*u*u
		if y > screenH+40 || x > screenW+40 {
			continue
		}
		balloonItem(c, i, x, y)
		if u < 0.5 {
			g.sceneBang(c, []string{"¡FUERA!", "¡FUERA!", "¡FUERA!"}[i], x+20, y-50, u/0.5, 0xffe060, 1.6)
		}
	}
	// the treetops that brush the basket
	if sceneIn(t, bnTrees-0.4, bnTrees+1.1) {
		u := (t - bnTrees + 0.4) / 1.5
		for k := 0; k < 3; k++ {
			x := math.Round(1200 - u*1500 + float64(k)*150)
			if x < -120 || x > screenW+120 {
				continue
			}
			fillRect(c, x-5, 460, 10, 90, rgb(0x5a3a22))
			filledEllipse(c, x, 440, 70, 52, rgb(0x2e7a34))
			filledEllipse(c, x-14, 424, 44, 32, rgb(0x4a9a48))
			filledEllipse(c, x+24, 450, 30, 22, rgb(0x246a2c))
		}
		for i := 0; i < 10; i++ { // leaves
			age := math.Mod(t*1.6+float64(i)/10, 1)
			fillRect(c, math.Round(520+float64(i)*37-age*90), math.Round(400+age*120+math.Sin(float64(i))*30), 6, 4, rgb(0x5ab04a))
		}
		g.sceneBang(c, "¡FROOOSH!", 480, 330, math.Min(1, (t-bnTrees)/0.9), 0x9aff8a, 2.4)
	}
	// farts: the trail of the balloon flying off
	if t >= bnZip {
		tx, ty, _ := balloonRig(t)
		for i := 0; i < 14; i++ {
			age := math.Mod(t*2.4+float64(i)/14, 1)
			px := bnRX + tx - 80 - age*130
			py := bnRY - bnBH - 150 + ty + math.Sin(float64(i)*5+t*6)*30
			filledEllipse(c, px, py, 8+age*18, 7+age*14, rgba(0xf4f0e0, uint8(190*(1-age))))
		}
		if sceneIn(t, bnZip, bnZip+0.8) {
			g.sceneBang(c, "¡PRRRRR!", 470, 200, (t-bnZip)/0.8, 0xffffff, 3.4)
		}
		if sceneIn(t, bnZip+1.4, bnZip+2.0) {
			g.sceneBang(c, "¡PFRRRR!", 560, 300, (t-bnZip-1.4)/0.6, 0xffe060, 2.6)
		}
	}
	if sceneIn(t, bnPeck, bnPeck+0.7) {
		g.sceneBang(c, "¡PLOP!", 640, 140, (t-bnPeck)/0.7, 0xffe060, 2.6)
	}
	if sceneIn(t, bnPeck+0.7, bnPeck+2.2) {
		g.sceneBang(c, "¡PSSSSS!", 520, 90, (t-bnPeck-0.7)/1.5, 0xffffff, 2.2)
	}
}

// ---- captions and the talk -------------------------------------------------------------------------------------

func (g *Game) balloonCaption(dst *ebiten.Image, in *interState) {
	t := in.t
	var line string
	switch {
	case sceneIn(t, 0.4, bnCut+0.8):
		line = "El medio de transporte más fiable."
	case sceneIn(t, 2.8, bnPeak):
		line = "Qué altura. Qué silencio."
	case sceneIn(t, 5.4, 8.6):
		line = "Un pájaro con opiniones."
	case sceneIn(t, 9.4, bnFade-0.4):
		line = "Propulsión a pedorreta."
	}
	if line != "" {
		w := g.textW(tr(line), 1.5) + 40
		fillRect(dst, 480-w/2, 488, w, 32, rgba(0x000000, 170))
		g.drawTextC(dst, line, 480, 495, rgb(0xf4f0ff), 1.5)
	}
	g.balloonSay(dst, in)
	if t < 0.8 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-t/0.8))))
	}
	if t > bnFade {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*math.Min(1, (t-bnFade)/(balloonLen-bnFade-0.2)))))
	}
	g.interSkipHint(dst, in)
}

// balloonSay: one bubble at a time, over the head of whoever talks (until it flies off).
func (g *Game) balloonSay(dst *ebiten.Image, in *interState) {
	t := in.t
	if t >= bnZip {
		return
	}
	say := func(k int, s string) {
		x, y, _, _, _, _, _ := balloonPose(t, k, in.two)
		tx, ty, _ := balloonRig(t)
		g.speechBubble(dst, x+tx+10, y+ty-34*interScale-4, s)
	}
	if !in.two {
		switch {
		case sceneIn(t, 0.6, bnCut-0.1):
			say(0, "Despegue con estilo.")
		case sceneIn(t, bnCut+0.2, 2.8):
			say(0, "¡Cuerda cortada!")
		case sceneIn(t, 3.2, 4.5):
			say(0, "Qué vistas... qué paz.")
		case sceneIn(t, 5.2, 6.1):
			say(0, "¿Eso era un pájaro?")
		case sceneIn(t, 6.2, 7.5):
			say(0, "¡Tira peso, tira peso!")
		case sceneIn(t, 7.7, bnTrees+0.2):
			say(0, "¡Vienen árboles!")
		}
		return
	}
	switch {
	case sceneIn(t, 0.6, 1.2):
		say(0, "Despegue con estilo.")
	case sceneIn(t, 1.2, 2.2):
		say(1, "¡Corta ya la cuerda!")
	case sceneIn(t, 2.3, 3.1):
		say(0, "Ya está. Cortada.")
	case sceneIn(t, 3.3, 4.5):
		say(0, "Qué vistas. Qué paz.")
	case sceneIn(t, 5.2, 6.1):
		say(1, "¡¡UN PÁJARO!!")
	case sceneIn(t, 6.2, 7.1):
		say(0, "Tiremos lastre. Con calma.")
	case sceneIn(t, 7.2, 8.0):
		say(1, "¡¡Tira la nevera!!")
	case sceneIn(t, 8.0, 9.0):
		say(0, "Árboles. Agáchate.")
	}
}

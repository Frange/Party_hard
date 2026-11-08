package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The boat interlude (after party 10): "THE LEAK AND THE SHARK". A rowing boat on a sunny sea, nothing
// like the night street of the car:
//
//	1. calm: he rows (in 2P the blue one rows, the red one is anxious); a plank gives way and a geyser
//	   squirts up in the middle of the boat while the water level rises over their shins,
//	2. he bails with a bucket (the water goes up as fast as it goes out), and a fin appears and circles,
//	3. BONK: the shark bumps the boat from below (which closes the hole), jumps out of the sea above them
//	   and swallows the bucket he threw up in panic; it lands with the bucket stuck on its snout,
//	4. blind and embarrassed, the shark comes round to the stern and pushes the boat full speed
//	   (the water runs out over the stern); they wave goodbye and the next party begins.
//
// About 14 seconds. Everything is procedural. The passengers never leave the hull (interlude_boat_test.go).
// ---------------------------------------------------------------------------

// the timeline, in seconds
const (
	boatLeak   = 1.7  // a plank gives way: the geyser
	boatBail0  = 3.0  // the bucket
	boatFin    = 4.8  // the fin appears
	boatBump   = 6.6  // BONK: it bumps the boat from below (and closes the hole)
	boatJump   = 7.4  // the shark leaps over the boat
	boatApex   = 7.95 // ...and swallows the bucket at the top
	boatLand   = 8.5  // it lands with the bucket stuck on its nose
	boatTurn   = 9.7  // it has crossed the sea and turns round
	boatPush   = 10.3 // it pushes the boat
	boatFade   = 12.7
	boatLen    = 13.8
	boatBX     = 300.0 // the stern of the boat on the screen (the sea moves, the boat stays)
	boatBaseY  = 372.0 // its waterline
	boatHorzn  = 250.0 // the horizon
	interBoatW = 300.0 // the hull: width...
	// ...and height of the gunwale above the waterline
	interBoatTop = 46.0
)

// boatScale: the passengers are drawn at 3x (whole number, no stretching): at 4x the slim hero towered over the little boat
const boatScale = 3.0

// where each passenger sits, in pixels from the stern (0 the blue one, 1 the red one)
var boatSeatDX = [2]float64{62, 214}

// boatWater: how high the water is inside the boat (pixels).
func boatWater(t float64) float64 {
	switch {
	case t < boatLeak:
		return 0
	case t < boatBail0:
		return (t - boatLeak) * 8
	case t < boatBump:
		return math.Min(24, (boatBail0-boatLeak)*8+(t-boatBail0)*2.2) // the bucket only keeps up with the leak
	case t < boatPush:
		return math.Min(24, (boatBail0-boatLeak)*8+(boatBump-boatBail0)*2.2) - (t-boatBump)*1.2
	}
	w := math.Min(24, (boatBail0-boatLeak)*8+(boatBump-boatBail0)*2.2) - (boatPush-boatBump)*1.2
	return math.Max(0, w-(t-boatPush)*7) // it runs out over the stern
}

// boatY is the waterline of the boat at time t: it bobs, sinks with the water, jumps at the bump, and lifts its bow when pushed.
func boatY(t float64) float64 {
	bob := math.Round(math.Sin(t*2.4) * 2)
	y := boatBaseY + bob + math.Round(boatWater(t)*0.35)
	if t > boatBump {
		y += math.Round(7 * math.Exp(-(t-boatBump)*5) * math.Cos((t-boatBump)*18))
	}
	if t > boatPush {
		y -= math.Round(math.Min(5, (t-boatPush)*4))
	}
	return y
}

// boatSpeed is how fast the sea goes by (px/s).
func boatSpeed(t float64) float64 {
	return 22 + 340*sceneWindow(t, boatPush, boatPush+1.3)
}

// boatDist: how far the sea has moved (the integral of boatSpeed).
func boatDist(t float64) float64 {
	d := 22 * t
	if t > boatPush {
		u := math.Min(t-boatPush, 1.3)
		d += 340*u*u/(2*1.3) + 340*math.Max(0, t-boatPush-1.3)
	}
	return d
}

// boatSeat: where the feet of passenger k are at time t (the boat's own position and bobbing plus a 1 px sway).
func boatSeat(t float64, k int) (x, y float64) {
	return boatBX + boatSeatDX[k], boatY(t) - 32 + math.Round(math.Sin(t*3.1+float64(k)*2))
}

// boatBailer: who has the bucket (the blue one alone, or the red one in 2P).
func boatBailer(two bool) int {
	if two {
		return 1
	}
	return 0
}

// boatPose: what passenger k does at time t: feet, animation, phase, facing, lean and hop. The feet stay within
// 40 px of the seat and the hop is small, so he never leaves the hull.
func boatPose(t float64, k int, two bool) (x, y float64, anim int, ph float64, dir Dir, lean, hop float64) {
	x, y = boatSeat(t, k)
	dir = DirRight
	anim = AnimIdle
	stroke := math.Sin(t * 3.2)
	switch {
	case !two || k == 0:
		switch {
		case t < boatLeak+0.5: // rows
			lean = 0.1 * stroke
		case t < boatBail0: // "is that water?"
			if two {
				lean = 0.1 * stroke
			} else {
				anim, ph = AnimPoint, t*6
			}
		case t < boatFin:
			if two {
				lean = 0.1 * stroke
			} else { // bails
				lean = 0.16 * math.Sin(t*9)
				hop = math.Abs(math.Sin(t*9)) * 4
			}
		case t < boatBump:
			if two { // the red one panics and he sips his drink
				anim, ph = AnimDrink, t*5
			} else {
				anim, ph = AnimPoint, t*6
			}
		case t < boatApex:
			if two {
				lean = 0.1 * stroke
			} else {
				anim, ph = AnimPanic, t*9
			}
		case t < boatPush+0.6:
			if !two {
				anim, ph = AnimPanic, t*9
			}
		default:
			anim, ph = AnimWave, t*8
		}
		if two && t >= boatPush {
			anim, ph, lean = AnimWave, t*8, 0
		}
	default: // the red one: nervous
		switch {
		case t < boatLeak:
			anim, ph = AnimTalk, t*5
		case t < boatFin: // bails like mad
			lean = 0.18 * math.Sin(t*11)
			hop = math.Abs(math.Sin(t*11)) * 5
		case t < boatBump: // hides behind the blue one
			u := introEase((t - boatFin) / 0.6)
			x -= 34 * u
			anim, ph = AnimPanic, t*10
		case t < boatPush:
			x -= 34
			anim, ph = AnimPanic, t*10
		default:
			x -= 34 * (1 - introEase((t-boatPush)/0.8))
			anim, ph = AnimCheer, t*8
		}
	}
	y -= hop
	return
}

// boatHand: where the hands of passenger k are (the oar, the bucket).
func boatHand(x, y float64) (hx, hy float64) { return x + 6*boatScale, y - 14.5*boatScale }

// ---- update ------------------------------------------------------------------------------------------

func boatFadeVol(t float64) float64 {
	v := math.Min(1, t/1.5)
	if t > boatFade-1.5 {
		v *= 1 - sceneWindow(t, boatFade-1.5, boatLen-0.3)
	}
	return v
}

func (g *Game) updateBoat(in *interState) {
	if !in.music {
		in.music = true
		g.sceneMusic(sceneInterB, ambInterB)
	}
	g.snd.SetSceneVolume("base", 0.55*boatFadeVol(in.t))
	once := func(key string, at float64, name string) {
		if in.t >= at && !in.did[key] {
			in.did[key] = true
			g.snd.Play(name)
		}
	}
	once("leak", boatLeak, "splash")
	once("fin", boatFin, "alert")
	once("bump", boatBump, "thud")
	once("jump", boatJump, "splash")
	once("chomp", boatApex, "boom")
	once("land", boatLand, "splash")
	once("push", boatPush, "thud")
	once("win", boatPush+1.5, "win")
	if in.t >= boatLen {
		g.finishInterlude()
	}
}

// ---- drawing -----------------------------------------------------------------------------------------

func (g *Game) drawBoatScene(dst *ebiten.Image, in *interState) {
	if in.canvas == nil {
		in.canvas = ebiten.NewImage(screenW, screenH)
	}
	c := in.canvas
	t := in.t
	g.boatSea(c, t)
	sharkBehind := g.boatShark(c, t, true)
	g.boatDrawBoat(c, in)
	if !sharkBehind {
		g.boatShark(c, t, false)
	}
	g.boatCaption(dst, c, in)
}

// boatSea: the sky and the sea, going by with the distance travelled.
func (g *Game) boatSea(c *ebiten.Image, t float64) {
	H := boatHorzn
	D := boatDist(t)
	for y := 0.0; y < H; y += 4 {
		fillRect(c, 0, y, screenW, 4, mixc(rgb(0x2a78d8), rgb(0xb4e4ff), y/H))
	}
	filledEllipse(c, 790, 78, 70, 70, rgba(0xfff4a0, 60))
	filledEllipse(c, 790, 78, 44, 44, rgb(0xfff08a))
	for k := 0; k < 5; k++ { // clouds, passing by
		cx := math.Round(math.Mod(float64(k)*260+80-D*0.12, 1200)) - 120
		cy := 48 + float64(k*41%120)
		filledEllipse(c, cx, cy, 64, 14, rgba(0xffffff, 235))
		filledEllipse(c, cx+30, cy-10, 34, 13, rgba(0xffffff, 235))
		filledEllipse(c, cx-30, cy-6, 28, 10, rgba(0xf0f8ff, 235))
	}
	for k := 0; k < 3; k++ { // gulls
		gx := math.Round(math.Mod(float64(k)*340+120-t*30, screenW+80)) - 40
		gy := math.Round(70 + float64(k)*34 + math.Sin(t*1.3+float64(k))*8)
		w := math.Round(math.Sin(t*9+float64(k)*2) * 3)
		col := rgb(0xffffff)
		fillRect(c, gx-8, gy+w, 5, 2, col)
		fillRect(c, gx-3, gy+w/2, 4, 2, col)
		fillRect(c, gx+1, gy, 2, 2, col)
		fillRect(c, gx+3, gy+w/2, 4, 2, col)
		fillRect(c, gx+7, gy+w, 5, 2, col)
	}
	for y := H; y < screenH; y += 4 { // the sea
		fillRect(c, 0, y, screenW, 4, mixc(rgb(0x3ab8d8), rgb(0x0a4a96), math.Min(1, (y-H)/280)))
	}
	for y := H; y < H+110; y += 6 { // the glitter of the sun
		k := (y - H) / 110
		w := 12 + k*60 + math.Sin(t*3+y)*8
		fillRect(c, math.Round(790-w/2+math.Sin(t*2+y*0.3)*6), y, math.Round(w), 2, rgba(0xfffadc, uint8(210*(1-k*0.5))))
	}
	for l := 0; l < 4; l++ { // waves in layers: the nearer, the bigger and the faster
		y0 := H + 14 + float64(l)*66
		sp := 0.35 + float64(l)*0.3
		gap := 150 - float64(l)*20
		for k := 0; k < 12; k++ {
			wx := math.Round(math.Mod(float64(k)*gap-D*sp+jpHash(k, l, 111)*60+2400, screenW+80)) - 40
			wy := math.Round(y0 + jpHash(k, l+4, 113)*44 + math.Sin(t*1.6+float64(k))*2)
			w := math.Round(14 + float64(l)*9 + jpHash(k, l, 115)*10)
			fillRect(c, wx, wy, w, 2, rgba(0xffffff, uint8(90+l*30)))
			fillRect(c, wx+3, wy+3, w-6, 2, rgba(0x08306a, 70))
		}
	}
}

// the hull: brown planks, wedge pointing right. Drawn in two parts: the inside and the back of the rim before the
// passengers, the front wall after them.
func (g *Game) boatHull(c *ebiten.Image, x, y float64) {
	for r := 0.0; r < 40; r++ {
		w := 292 - r
		off := r * 0.3
		col := rgb(0xa8743c)
		if int(r/8)%2 == 1 {
			col = rgb(0x8c5a2a)
		}
		if r > 34 {
			col = rgb(0x4a2c16)
		}
		fillRect(c, x+off, y-44+r, w, 1, col)
	}
	fillRect(c, x, y-46, 300, 4, rgb(0xe8d4a8)) // the rim
	fillRect(c, x+26, y-34, 200, 3, rgb(0xd8402a))
	fillRect(c, x+26, y-34, 200, 1, rgb(0xf0705a))
	for i := 0.0; i < 280; i += 36 { // nails
		fillRect(c, x+10+i, y-40, 2, 2, rgb(0x3a2410))
	}
}

func (g *Game) boatDrawBoat(c *ebiten.Image, in *interState) {
	t := in.t
	x, y := boatBX, boatY(t)
	// wake behind the shark's push, or the little ripples round the hull
	for i := 0; i < 8; i++ {
		age := math.Mod(t*0.9+float64(i)/8, 1)
		filledEllipse(c, x+60+float64(i)*28, y+8, 22+age*4, 3, rgba(0xe8faff, uint8(120*(1-age))))
	}
	if t >= boatPush {
		for i := 0; i < 12; i++ {
			age := math.Mod(t*2.2+float64(i)/12, 1)
			filledEllipse(c, x+20+age*300, y+10+age*6, 10+age*12, 3, rgba(0xf4fcff, uint8(190*(1-age))))
		}
	}
	// the back rim and the dark inside
	fillRect(c, x+8, y-66, 280, 4, rgb(0xc8a870))
	fillRect(c, x+8, y-62, 280, 22, rgb(0x3a2412))
	// passengers
	nb := 1
	if in.two {
		nb = 2
	}
	var hands [2][2]float64
	for k := 0; k < nb; k++ {
		px, py, anim, ph, dir, lean, hop := boatPose(t, k, in.two)
		cs := in.hero
		if in.two {
			cs = in.heroes[k]
		}
		tint := [3]float32{1, 1, 1}
		if in.two {
			tint = sceneTint(k, in.heroes[0], in.heroes[1])
		}
		g.sceneChar(c, cs, anim, dir, ph, px, py+hop, boatScale, 1, hop, lean, tint)
		hands[k][0], hands[k][1] = boatHand(px, py+hop)
	}
	// water inside, in front of their legs
	if w := boatWater(t); w > 0 {
		wy := y - 44 - w // (the water shows over their legs, above the near wall)
		fillRect(c, x+10, wy, 276, w+14, rgba(0x2a8ad8, 190))
		fillRect(c, x+10, wy, 276, 2, rgba(0xdff4ff, 220))
		for i := 0.0; i < 276; i += 34 {
			fillRect(c, x+8+i+math.Mod(t*20, 34), wy+4, 12, 2, rgba(0xffffff, 120))
		}
	}
	// the geyser of the leak
	if t >= boatLeak && t < boatBump {
		h := 120 * math.Min(1, (t-boatLeak)/0.3)
		gx := x + 138
		for i := 0; i < 26; i++ {
			age := math.Mod(t*2.4+float64(i)/26, 1)
			px := gx + math.Sin(float64(i)*7.3)*(6+age*26)
			py := y - 34 - h*math.Sin(math.Pi*age)
			fillRect(c, math.Round(px), math.Round(py), 5, 5, rgba(0xdff6ff, uint8(240*(1-age*0.5))))
		}
		col := math.Round(h * 0.8) // the jet itself: a column that shoots up, then the drops fall round it
		fillRect(c, gx-4, y-34-col, 8, col, rgba(0xcdeeff, 170))
		fillRect(c, gx-2, y-34-col, 3, col, rgba(0xffffff, 200))
	}
	// the water running out over the stern, pushed by the shark
	if t >= boatPush && boatWater(t) > 0.5 {
		for i := 0.0; i < 40; i += 4 {
			fillRect(c, x-2-i*0.4, y-32+i*1.3, 5, 5, rgba(0xbfe8ff, 220))
		}
	}
	// the front wall of the hull, over their feet
	g.boatHull(c, x, y)
	// the oar of the one that rows
	if rows := in.two && !sceneIn(t, boatFin, boatBump) || t < boatLeak+0.5; rows && t < boatPush {
		g.boatOar(c, hands[0][0], hands[0][1], y, t)
	}
	// the bucket
	bk := boatBailer(in.two)
	if t >= boatBail0 && t < boatJump+0.15 {
		bx, by := hands[bk][0]+10, hands[bk][1]-4+math.Round(math.Sin(t*9)*8)
		if t >= boatBail0 && t < boatFin+1.5 { // the water it throws, over the side
			g.boatBailArc(c, bx, by, y, t)
		}
		boatBucket(c, bx, by, false)
	} else if t >= boatJump+0.15 && t < boatApex { // thrown up in panic
		u := sceneWindow(t, boatJump+0.15, boatApex)
		sx, sy := hands[bk][0]+10, hands[bk][1]-4
		ax, ay := boatBX+150.0, 175.0
		boatBucket(c, sx+(ax-sx)*u, sy+(ay-sy)*u-math.Sin(math.Pi*u)*40, false)
	}
	// the bump from below
	if t >= boatBump && t < boatBump+0.9 {
		u := (t - boatBump) / 0.9
		g.sceneBang(c, "¡BONK!", boatBX+150, y-120, u, 0xffe060, 2.6)
		for i := 0; i < 7; i++ { // bubbles
			fillRect(c, boatBX+60+float64(i)*28, y+14+math.Mod(float64(i)*13+u*50, 40), 4, 4, rgba(0xffffff, 200))
		}
	}
	if t >= boatBump && t < boatBump+1.2 { // stars over the one who rows
		px, py, _, _, _, _, _ := boatPose(t, 0, in.two)
		g.sceneStars(c, px, py-34*boatScale-6, t, 1)
	}
	if t >= boatApex && t < boatApex+0.6 {
		g.sceneBang(c, "¡ÑAM!", boatBX+150, 90, (t-boatApex)/0.6, 0xff6a6a, 3)
	}
	if t >= boatLand && t < boatLand+0.7 {
		g.sceneBang(c, "¡SPLASH!", boatBX+400, 330, (t-boatLand)/0.7, 0xdff4ff, 2.4)
	}
}

// boatOar: the oar goes from the hands down into the water and swings with the stroke.
func (g *Game) boatOar(c *ebiten.Image, hx, hy, waterY, t float64) {
	phi := 1.15 + 0.3*math.Sin(t*3.2+0.6)
	dx, dy := math.Cos(phi), math.Sin(phi)
	L := 150.0
	for s := -20.0; s < L; s += 2 {
		fillRect(c, math.Round(hx+dx*s), math.Round(hy+dy*s), 4, 4, rgb(0x7a5230))
	}
	tx, ty := hx+dx*L, hy+dy*L
	for s := 0.0; s < 30; s += 2 { // the blade
		fillRect(c, math.Round(tx+dx*s)-3, math.Round(ty+dy*s), 9, 3, rgb(0xb88650))
	}
	filledEllipse(c, tx+dx*14, math.Max(ty, waterY+4)+6, 22, 4, rgba(0xf4fcff, 150))
}

// boatBailArc: the water thrown by the bucket, in drops that fly over the side and fall into the sea.
func (g *Game) boatBailArc(c *ebiten.Image, x, y, waterY, t float64) {
	for i := 0; i < 10; i++ {
		age := math.Mod(t*1.45+float64(i)/10, 1)
		px := x + age*270
		py := y - 150*age*(1-age)*4 + age*age*(waterY+14-y)
		fillRect(c, math.Round(px), math.Round(py), 5, 5, rgba(0xcdeeff, uint8(240*(1-age*0.4))))
	}
}

// boatBucket: a tin bucket, mouth up (or upside down on a shark's nose).
func boatBucket(c *ebiten.Image, x, y float64, down bool) {
	x, y = math.Round(x), math.Round(y)
	if down {
		for r := 0.0; r < 20; r++ {
			fillRect(c, x-12+r*0.25, y+r-20, 24-r*0.5, 1, rgb(0x8a9ab0))
		}
		fillRect(c, x-14, y-22, 28, 4, rgb(0xb8c8dc))
		return
	}
	for r := 0.0; r < 22; r++ {
		fillRect(c, x-14+r*0.3, y+r-10, 28-r*0.6, 1, rgb(0x8a9ab0))
	}
	fillRect(c, x-15, y-12, 30, 4, rgb(0xb8c8dc))
	fillRect(c, x-14, y-9, 28, 2, rgba(0x2a8ad8, 220))
	fillRect(c, x-16, y-18, 2, 8, rgb(0x5a6070)) // handle
	fillRect(c, x+14, y-18, 2, 8, rgb(0x5a6070))
	fillRect(c, x-16, y-20, 32, 2, rgb(0x5a6070))
}

// ---- the shark -----------------------------------------------------------------------------------------

var boatSharkImgs [2]*ebiten.Image // 0 closed mouth, 1 open

// boatSharkSprite: the shark, 200 x 124, nose to the right; the middle of its body is the middle of the sprite.
func boatSharkSprite(open bool) *ebiten.Image {
	i := 0
	if open {
		i = 1
	}
	if boatSharkImgs[i] != nil {
		return boatSharkImgs[i]
	}
	m := ebiten.NewImage(200, 124)
	body, belly, dark := rgb(0x6a86a8), rgb(0xe4ecf4), rgb(0x4a6488)
	filledEllipse(m, 100, 62, 84, 24, body)
	filledEllipse(m, 108, 72, 70, 12, belly)
	for r := 0.0; r < 34; r++ { // the tail: a fork
		fillRect(m, 6+r*0.2, 62-r-2, 20-r*0.4, 1, dark)
		fillRect(m, 6+r*0.2, 62+r*0.8, 20-r*0.4, 1, dark)
	}
	for r := 0.0; r < 30; r++ { // the dorsal fin, leaning back
		fillRect(m, 100-r*0.6, 40-r, 26-r*0.8, 1, dark)
	}
	for r := 0.0; r < 22; r++ { // the pectoral fin
		fillRect(m, 96+r*0.4, 78+r*0.9, 22-r, 1, dark)
	}
	if open { // mouth: dark, with two rows of teeth
		for r := 0.0; r < 26; r++ {
			fillRect(m, 140+r*0.9, 56+r*0.4, 56-r*1.5, 1, rgb(0x5a1020))
		}
		for q := 0.0; q < 6; q++ {
			fillRect(m, 150+q*7, 54+q*1.8, 4, 6, rgb(0xffffff))
			fillRect(m, 150+q*7, 68+q*1.2, 4, 6, rgb(0xffffff))
		}
	} else {
		fillRect(m, 150, 68, 40, 2, rgb(0x2a3a58))
		for q := 0.0; q < 5; q++ {
			fillRect(m, 156+q*8, 70, 3, 4, rgb(0xffffff))
		}
	}
	filledEllipse(m, 152, 52, 6, 6, rgb(0xffffff)) // the eye
	fillRect(m, 153, 50, 4, 5, rgb(0x10101a))
	for q := 0.0; q < 3; q++ { // gills
		fillRect(m, 118+q*5, 56+q*0.5, 2, 12, dark)
	}
	boatSharkImgs[i] = m
	return m
}

// boatSharkPos: where the shark is at time t (centre of its sprite), if it is out of the water as a whole shark.
// The mode says how it is drawn: 1 leaping (rotated), 2 swimming left, 3 swimming right to the stern, 4 pushing.
func boatSharkPos(t float64) (x, y, ang float64, mode int, flip bool) {
	switch {
	case t >= boatJump && t < boatLand:
		u := (t - boatJump) / (boatLand - boatJump)
		x = boatBX - 100 + u*500
		y = 440 - 1250*u*(1-u)
		ang = math.Atan2(-1250*(1-2*u), 500)
		return x, y, ang, 1, false
	case t >= boatLand && t < boatTurn:
		u := introEase((t - boatLand) / (boatTurn - boatLand))
		return boatBX + 400 + (boatBX-190-(boatBX+400))*u, 446 + 4*math.Sin(t*5), 0, 2, true
	case t >= boatTurn && t < boatPush:
		u := introEase((t - boatTurn) / (boatPush - boatTurn))
		return boatBX - 190 + (boatBX-112-(boatBX-190))*u, 446 + (boatBaseY-2-446)*u, 0, 3, false
	case t >= boatPush:
		return boatBX - 112 + math.Round(math.Sin(t*14)), boatY(t) - 2, 0, 4, false
	}
	return
}

// boatShark draws the shark in two passes: back=true draws what goes behind the boat (the fin when it is on the far
// side of its circle) and reports whether it did. back=false draws the rest (in front of the boat).
func (g *Game) boatShark(c *ebiten.Image, t float64, back bool) bool {
	// the fin circling
	if t >= boatFin && t < boatBump {
		r := 1 - 0.45*sceneWindow(t, boatFin, boatBump)
		a := math.Pi + (t-boatFin)*2.6
		fx := boatBX + 150 + 270*r*math.Cos(a)
		fy := 412 + 48*r*math.Sin(a)
		isBack := math.Sin(a) < 0
		if isBack != back {
			return isBack && back
		}
		sink := 1.0
		if t > boatBump-0.35 {
			sink = (boatBump - t) / 0.35
		}
		dirx := -math.Sin(a) // its direction of travel
		boatFinDraw(c, fx, fy, dirx >= 0, sink, t)
		return isBack
	}
	if back {
		return false
	}
	if t >= boatBump && t < boatJump { // bubbles where it gathers speed
		for i := 0; i < 6; i++ {
			fillRect(c, boatBX-120+float64(i)*10, 420+math.Mod(float64(i)*9+t*40, 24), 4, 4, rgba(0xffffff, 190))
		}
	}
	x, y, ang, mode, flip := boatSharkPos(t)
	if mode == 0 {
		return false
	}
	open := mode == 1 && t > boatApex-0.35 && t < boatApex+0.05
	img := boatSharkSprite(open)
	sw, sh := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	const scale = 1.15
	src := img
	if mode >= 2 { // swimming: only what is over the water
		src = img.SubImage(image.Rect(0, 0, int(sw), 72)).(*ebiten.Image)
	}
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(sw, 0)
	}
	op.GeoM.Translate(-sw/2, -sh/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Rotate(ang)
	op.GeoM.Translate(math.Round(x), math.Round(y))
	c.DrawImage(src, op)
	// the bucket on its nose, upside down, once it has swallowed it
	if t >= boatApex {
		dir := 1.0
		if flip {
			dir = -1
		}
		nx, ny := dir*scale*76, -scale*4
		boatBucket(c, x+nx*math.Cos(ang)-ny*math.Sin(ang), y+nx*math.Sin(ang)+ny*math.Cos(ang)+10, true)
	}
	if mode >= 2 { // foam at the waterline
		filledEllipse(c, x, y+12, 100*scale, 5, rgba(0xf4fcff, 170))
	}
	if t >= boatLand-0.1 && t < boatLand+0.5 { // the splash of the landing
		u := (t - boatLand + 0.1) / 0.6
		for i := 0; i < 8; i++ {
			a := float64(i) * 0.7
			fillRect(c, boatBX+380+math.Cos(a)*80*u, 440-math.Abs(math.Sin(a))*210*u*(1-u), 6, 6, rgba(0xf4fcff, uint8(220*(1-u))))
		}
	}
	return false
}

// boatFinDraw: a fin cutting the water, with its wake.
func boatFinDraw(c *ebiten.Image, x, y float64, right bool, sink, t float64) {
	x, y = math.Round(x), math.Round(y)
	d := 1.0
	if !right {
		d = -1
	}
	for i := 1; i <= 6; i++ { // the V of the wake
		age := float64(i) / 6
		filledEllipse(c, x-d*age*70, y+6+age*4, 14+age*14, 3, rgba(0xf4fcff, uint8(170*(1-age))))
	}
	h := 38 * sink
	for r := 0.0; r < h; r++ {
		w := (h - r) * 0.55
		off := (h - r) * 0.35 * d
		fillRect(c, x-w/2-off*0.0+off*0.0, y-h+r, w+4, 1, rgb(0x5a748f))
	}
	fillRect(c, x-2, y-h, 4, 3, rgb(0x34506f))
	filledEllipse(c, x, y+3, 22, 4, rgba(0xf4fcff, 190))
}

// ---- captions and the talk -----------------------------------------------------------------------------------

func (g *Game) boatCaption(dst, c *ebiten.Image, in *interState) {
	t := in.t
	// a little shake with the bump and when the shark lands
	ox, oy := 0.0, 0.0
	if t > boatBump && t < boatBump+0.5 {
		ox, oy = math.Round(math.Sin(t*60)*4), math.Round(math.Cos(t*50)*3)
	}
	if t > boatLand && t < boatLand+0.3 {
		oy = math.Round(math.Sin(t*55) * 3)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox, oy)
	dst.DrawImage(c, op)
	var line string
	switch {
	case sceneIn(t, 0.6, 3.0):
		line = "Una barca, un mar y cero planes."
	case sceneIn(t, 3.1, 6.4):
		line = "El casco tiene otra opinión."
	case sceneIn(t, 6.8, 9.6):
		line = "Y llega la visita."
	case sceneIn(t, 10.6, boatFade-0.4):
		line = "Taxi tiburón, servicio exprés."
	}
	if line != "" {
		w := g.textW(tr(line), 1.5) + 40
		fillRect(dst, 480-w/2, 488, w, 32, rgba(0x000000, 170))
		g.drawTextC(dst, line, 480, 495, rgb(0xf4f0ff), 1.5)
	}
	g.boatSay(dst, in)
	if t < 0.8 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-t/0.8))))
	}
	if t > boatFade {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*math.Min(1, (t-boatFade)/(boatLen-boatFade-0.2)))))
	}
	g.interSkipHint(dst, in)
}

// boatSay: one bubble at a time. 1P: only the blue one talks; 2P: the blue one is calm, the red one is not.
func (g *Game) boatSay(dst *ebiten.Image, in *interState) {
	t := in.t
	say := func(k int, s string) {
		x, y, _, _, _, _, _ := boatPose(t, k, in.two)
		g.speechBubble(dst, x+10, y-34*boatScale-4, s)
	}
	if !in.two {
		switch {
		case sceneIn(t, 0.9, 1.9):
			say(0, "Qué paz, qué calma...")
		case sceneIn(t, 2.0, 3.1):
			say(0, "¿Eso... es agua?")
		case sceneIn(t, 3.3, 4.7):
			say(0, "¡Achica, achica!")
		case sceneIn(t, 5.0, 6.5):
			say(0, "¿Eso es una aleta?")
		case sceneIn(t, 8.5, 9.8):
			say(0, "¡Se ha comido mi cubo!")
		case sceneIn(t, 10.6, 11.7):
			say(0, "¿Nos empuja? ¡Qué majo!")
		case sceneIn(t, 11.8, boatFade):
			say(0, "¡Gracias, tiburón!")
		}
		return
	}
	switch {
	case sceneIn(t, 0.9, 1.7):
		say(0, "Con calma. Remo yo.")
	case sceneIn(t, 2.0, 3.0):
		say(1, "¡¡Hay una fuga!!")
	case sceneIn(t, 3.2, 4.6):
		say(0, "Tú achica, yo remo.")
	case sceneIn(t, 4.9, 6.0):
		say(1, "¡¡TIBURÓN!!")
	case sceneIn(t, 6.1, 6.6):
		say(0, "Será un delfín.")
	case sceneIn(t, 8.5, 9.8):
		say(1, "¡Mi cubo! ¡Era nuevo!")
	case sceneIn(t, 10.6, 11.6):
		say(0, "Nos empuja. Qué amable.")
	case sceneIn(t, 11.7, boatFade):
		say(1, "¡Gracias, tiburón!")
	}
}

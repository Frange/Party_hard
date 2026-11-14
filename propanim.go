package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// MOVING PROPS (feedback of v1.4.0: "los objetos de los niveles deben tener movimiento"). A few big props of the
// levels get a short loop of frames: the ferris wheel and the carousel turn, the balloons sway, the crowd of the
// stands moves (heads bob, arms up, a little jump), the mirror ball turns its mirrors, the fireplace flickers, the
// water of the fountain runs, the gondolier rows, the slot machines blink (and spin their reels while somebody
// plays: propplay.go) and the roulette wheel turns with its ball.
//
// CHEAP: the frames are drawn ONCE (the first time the prop is drawn, through the same pipeline as every prop sprite:
// detail, outline, the muted palette) and kept in propAnimCache; every frame of the game only picks one of them
// (drawProp asks propAnimImg). No allocation per frame, no drawing of pixels per frame.
//
// DETERMINISTIC: the frame is a function of the game clock g.time (stops in the pause) and of the prop's place, so a
// capture at a given tick is always the same.
//
// The sprites of these props may come from the asset folder (the editor exports them). The animation is used only
// when the file is the STOCK export (pixel for pixel the built-in drawing): an object the user painted in the editor
// keeps his drawing, still. propAnimRemember (called by applyObjects before it replaces the built-in kinds) keeps the
// built-in kinds to compare with.
//
// Collisions are untouched: only the drawn image changes (same size and anchor as the still sprite); the footprint,
// the solids and the navigation of the level are the same.
// ---------------------------------------------------------------------------

type propAnim struct {
	n     int     // frames of the loop
	fps   float64 // frames per second
	draw  func(b *pixBuf, k *PropKind, v, f int)
	frame func(t, ph float64, busy bool) int // optional: picks the frame (default int(t*fps+ph) % n)
}

// propAnims: the animated props (by name).
var propAnims = map[string]*propAnim{
	"ferris":     {n: 9, fps: 3, draw: drawFerrisFrame},
	"carousel":   {n: 24, fps: 6, draw: drawCarouselFrame},
	"balloons":   {n: 8, fps: 3, draw: drawBalloonsFrame},
	"stands":     {n: 12, fps: 3, draw: drawStandsFrame},
	"mirrorball": {n: 3, fps: 3, draw: drawMirrorBallFrame},
	"fireplace":  {n: hearthFrames, fps: 9, draw: drawFireplaceFrame},
	"fountain":   {n: 4, fps: 6, draw: drawFountainFrame},
	"gondola":    {n: 6, fps: 2.5, draw: drawGondolaFrame},
	"slot":       {n: 8, fps: 2.5, draw: drawSlotFrame, frame: slotFrame},
	"roulette":   {n: 16, fps: 8, draw: drawRouletteFrame},
}

// propAnimOff: tests switch the animation off to compare with the still sprites.
var propAnimOff bool

// propAnimBuiltin: the built-in kinds of the animated props, kept before the asset objects replace them.
var propAnimBuiltin = map[string]*PropKind{}

// propAnimRemember keeps the built-in kinds (called by applyObjects before it registers the asset objects).
func propAnimRemember() {
	for n := range propAnims {
		if _, ok := propAnimBuiltin[n]; ok {
			continue
		}
		if k := propKinds[n]; k != nil && k.Native == nil && k.Draw != nil {
			propAnimBuiltin[n] = k
		}
	}
}

// propAnimSet: the frames of one prop, [variant][frame]. base is the still sprite they were built for (the cache is
// rebuilt when the sprites are rebuilt: the editor reloads the objects). off = the user painted his own sprite.
type propAnimSet struct {
	base   *ebiten.Image
	frames [][]PropImg
	off    bool
}

var propAnimCache = map[string]*propAnimSet{}

// propAnimKind: the built-in kind to draw the frames with, or nil when the sprite in use is not the stock one.
func propAnimKind(name string, k *PropKind) *PropKind {
	if k == nil {
		return nil
	}
	if k.Native == nil {
		if k.Draw == nil {
			return nil
		}
		return k
	}
	bk := propAnimBuiltin[name]
	if bk == nil {
		return nil
	}
	for i, f := range k.Native { // the asset must be the stock export of every variant
		r := renderPropSprite(name, bk, i)
		if r.w != f.w || r.h != f.h || string(r.p) != string(f.p) {
			return nil
		}
	}
	return bk
}

// renderPropFrame draws frame f of variant v like renderPropSprite draws a still sprite (same seed: the noise of
// the detail does not flicker between frames).
func renderPropFrame(name string, k *PropKind, a *propAnim, v, f int) *pixBuf {
	b := newPix(k.W+2, k.H+2)
	b.ox, b.oy = 1, 1
	a.draw(b, k, v, f)
	seed := int64(len(name)*31+v) + int64(name[0])*977
	return b.scaled(wk).detail(seed, 0.035).outlinedSel(0.45)
}

func buildPropAnim(name string, a *propAnim, imgs []PropImg) *propAnimSet {
	s := &propAnimSet{base: imgs[0].img}
	bk := propAnimKind(name, propKinds[name])
	if bk == nil {
		s.off = true
		return s
	}
	for v := range imgs {
		row := make([]PropImg, a.n)
		for f := 0; f < a.n; f++ {
			row[f] = PropImg{img: stylizeSprite(renderPropFrame(name, bk, a, v, f)).img(), ox: imgs[v].ox, oy: imgs[v].oy}
		}
		s.frames = append(s.frames, row)
	}
	return s
}

// propAnimPhase: a fixed offset per prop (two wheels of the same fair do not turn in step).
func propAnimPhase(pr *Prop) float64 {
	return float64((int(pr.X)*7 + int(pr.Y)*3) % 97)
}

// propAnimFrameAt: the frame of an animation at game time t.
func propAnimFrameAt(a *propAnim, t, ph float64, busy bool) int {
	if a.frame != nil {
		return a.frame(t, ph, busy) % a.n
	}
	f := int(t*a.fps+ph) % a.n
	if f < 0 {
		f += a.n
	}
	return f
}

// propAnimImg: the image of an animated prop now (ok false = draw the still sprite).
func (g *Game) propAnimImg(pr *Prop, v int) (PropImg, bool) {
	if propAnimOff || pr.wreck {
		return PropImg{}, false
	}
	a := propAnims[pr.Name]
	if a == nil || g.as == nil {
		return PropImg{}, false
	}
	imgs := g.as.props[pr.Name]
	if len(imgs) == 0 {
		return PropImg{}, false
	}
	s := propAnimCache[pr.Name]
	if s == nil || s.base != imgs[0].img {
		s = buildPropAnim(pr.Name, a, imgs)
		propAnimCache[pr.Name] = s
	}
	if s.off || len(s.frames) == 0 {
		return PropImg{}, false
	}
	row := s.frames[v%len(s.frames)]
	return row[propAnimFrameAt(a, g.time, propAnimPhase(pr), g.machBusy(pr))], true
}

// ---- the frames ------------------------------------------------------------------------------------------------

// drawFerrisFrame: the wheel turned by f/9 of the angle between two spokes (9 frames = one spoke: the loop closes);
// the cabins hang at the tips of the spokes.
func drawFerrisFrame(b *pixBuf, k *PropKind, v, f int) {
	cx, cy, r := 56.0, 62.0, 50.0
	rot := float64(f) / 9 * (2 * math.Pi / 10)
	b.line(30, 130, 56, 62, c(0x8a8a98))
	b.line(82, 130, 56, 62, c(0x8a8a98))
	b.line(31, 130, 57, 62, c(0x6a6a78))
	b.line(81, 130, 55, 62, c(0x6a6a78))
	for a := 0.0; a < 2*math.Pi; a += 0.03 {
		b.set(int(cx+math.Cos(a)*r), int(cy+math.Sin(a)*r), c(0xe8e8f4))
		b.set(int(cx+math.Cos(a)*(r-1)), int(cy+math.Sin(a)*(r-1)), c(0x9a9aac))
		b.set(int(cx+math.Cos(a)*(r*0.55)), int(cy+math.Sin(a)*(r*0.55)), c(0xc8c8d8))
	}
	for i := 0; i < 10; i++ {
		a := float64(i)/10*2*math.Pi + rot
		b.line(int(cx), int(cy), int(math.Round(cx+math.Cos(a)*r)), int(math.Round(cy+math.Sin(a)*r)), c(0xb8b8c8))
	}
	for i := 0; i < 10; i++ { // the cabins over the spokes
		a := float64(i)/10*2*math.Pi + rot
		gx, gy := int(math.Round(cx+math.Cos(a)*r)), int(math.Round(cy+math.Sin(a)*r))
		b.rect(gx-3, gy, 7, 6, c([]uint32{0xe04a5a, 0xf0c030, 0x40b0e0, 0x70d060, 0xe880c0}[i%5]))
		b.rect(gx-3, gy, 7, 1, c(0xf8f8f8))
	}
	b.ellipse(cx, cy, 4, 4, c(0xe04a5a))
	b.rect(26, 128, 60, 4, c(0x4a4a58))
}

// drawCarouselFrame: the horses go round (24 frames = half a turn: the colours repeat every four horses) and
// bob up and down on their poles.
func drawCarouselFrame(b *pixBuf, k *PropKind, v, f int) {
	b.ellipse(46, 72, 44, 11, c(0x6a2a34))
	b.ellipse(46, 70, 44, 11, c(0xd8b050))
	b.ellipse(46, 69, 38, 8.5, c(0x8a3a46))
	cols := []uint32{0xe04a5a, 0xf0e8d0}
	horse := []uint32{0xf4f4f0, 0xd08040, 0x6a4a30, 0xe8a0c0}
	rot := float64(f) / 24 * math.Pi
	type hp struct{ px, py, hy, i int }
	var back, front []hp
	for i := 0; i < 8; i++ {
		a := float64(i)/8*2*math.Pi + 0.4 + rot
		px := 46 + int(math.Round(math.Cos(a)*30))
		py := 66 + int(math.Round(math.Sin(a)*6))
		bob := int(math.Round(math.Sin(float64(i)*math.Pi/2+float64(f)*2*math.Pi/12) * 2))
		h := hp{px, py, py - 22 + bob, i}
		if math.Sin(a) > -0.3 {
			front = append(front, h)
		} else {
			back = append(back, h)
		}
	}
	for _, h := range back {
		b.rect(h.px, h.py-30, 1, 30, c(0xf0d070))
	}
	for _, h := range front {
		b.rect(h.px, h.py-30, 1, 30, c(0xf0d070))
		hx := h.px - 3
		col := c(horse[h.i%4])
		b.rect(hx, h.hy, 7, 5, col)
		b.rect(hx+5, h.hy-3, 3, 4, col)
		b.rect(hx, h.hy+5, 1, 3, c(0x3a2a20))
		b.rect(hx+5, h.hy+5, 1, 3, c(0x3a2a20))
	}
	b.rect(44, 14, 4, 52, c(0xd8b050))
	for y := 8; y < 34; y++ {
		t := float64(y-8) / 26
		w := int(5 + 40*t)
		for x := 46 - w; x < 46+w; x++ {
			kk := (x - (46 - w)) * 8 / (2*w + 1)
			b.set(x, y, c(cols[kk%2]))
		}
	}
	for x := 4; x < 88; x += 6 { // the bulbs of the rim blink in turn
		cc := c(0xf0d070)
		if (x/6+f/3)%2 == 0 {
			cc = c(0xfff4c0)
		}
		b.rect(x, 33, 3, 2, cc)
	}
	b.rect(45, 2, 2, 7, c(0xf0d070))
	b.ellipse(46, 3, 2, 2, c(0xff4060))
}

// drawBalloonsFrame: every balloon sways a pixel on its string, each with its own beat.
func drawBalloonsFrame(b *pixBuf, k *PropKind, v, f int) {
	cols := []uint32{0xe04a5a, 0xf0d030, 0x40a0e0, 0x60c070, 0xe880c0, 0xf0a030}
	ph := float64(f) / 8 * 2 * math.Pi
	for i := 0; i < 8; i++ {
		x := 6 + (i*9)%24 + int(math.Round(math.Sin(ph+float64(i)*1.7)*1.2))
		y := 8 + (i*13)%22 + int(math.Round(math.Sin(2*ph+float64(i)*2.3)*0.7))
		b.ellipse(float64(x), float64(y), 5, 6.4, c(cols[i%6]))
		b.set(x-2, y-3, c(0xffffff))
		b.line(x, y+6, 17, 50, c(0xd8d8e0))
	}
	b.rect(14, 50, 6, 4, c(0x6a6a76))
}

// standsFig: what one figure of the painted crowd does in frame f: 0 sits, 1 bobs up, 2 arms up, 3 jumps.
func standsFig(x, t, v, f int) int {
	h := (x*73 + t*151 + v*37) % 97
	switch p := (f + h) % 12; {
	case h%7 == 0 && p < 2:
		return 3
	case h%3 == 0 && p < 3:
		return 2
	case h%2 == 0 && (p == 4 || p == 9):
		return 1
	}
	return 0
}

// drawStandsFrame: the stands with a crowd that moves: some heads bob, some raise their arms, a few jump.
func drawStandsFrame(b *pixBuf, k *PropKind, v, f int) {
	b.rect(0, 14, 210, 36, c(0x4a4a58))
	for t := 0; t < 4; t++ {
		y := 10 + t*9
		b.rect(0, y, 210, 3, c(0x8a8a98))
		b.rect(0, y+3, 210, 6, c(0x5a5a68))
		for x := 3 + (t*5)%6; x < 206; x += 8 {
			skin := c([]uint32{0xe8b48a, 0xc98a5e, 0x8a5a3a, 0xf7d9c0}[(x/8+t)%4])
			shirt := c([]uint32{0xe04a5a, 0x40a0e0, 0xf0c030, 0x60c070, 0xf4f4f0, 0x8a2aa8}[(x/8*3+t*2+v)%6])
			up := 0
			st := standsFig(x, t, v, f)
			switch st {
			case 1, 2:
				up = 1
			case 3:
				up = 2
			}
			b.rect(x, y-2-up, 4, 3, skin)
			b.rect(x-1, y+1-up, 6, 6+up, shirt)
			if st >= 2 { // the arms up
				b.rect(x-2, y-5-up, 1, 5, skin)
				b.rect(x+5, y-5-up, 1, 5, skin)
			}
		}
	}
	b.rect(0, 46, 210, 4, c(0x3a3a46))
	b.rect(0, 0, 210, 6, c(0x9a2a2a))
	for x := 0; x < 210; x += 14 {
		b.rect(x, 0, 7, 6, c(0xf0e0b0))
	}
}

// drawMirrorBallFrame: the mirrors turn (the tiles slide a pixel per frame: 3 frames = one tile, then the loop
// starts again with other brightnesses, the flicker of a turning ball) and the sparkles jump.
func drawMirrorBallFrame(b *pixBuf, k *PropKind, v, f int) {
	b.rect(13, 0, 1, 14, c(0xb8bcc8)) // chain
	cx, cy, r := 13.0, 29.0, 12.0
	for y := -12; y <= 12; y++ {
		for x := -12; x <= 12; x++ {
			if float64(x*x+y*y) > r*r {
				continue
			}
			xs := x + 16 + f // the tiles slide sideways a pixel per frame
			gx, gy := xs/3, (y+16)/3
			kk := 0.6 + 0.4*float64((gx*7+gy*13+v*5)%5)/4
			if x < 0 && y < 0 {
				kk += 0.18
			}
			if x > 4 || y > 6 {
				kk -= 0.22
			}
			base := c(0xc8d0e0)
			if xs%3 == 0 || (y+16)%3 == 0 {
				base = c(0x6a7284)
			}
			b.set(int(cx)+x, int(cy)+y, shade(base, math.Max(0.4, math.Min(1.3, kk))))
		}
	}
	spark := [][2]int{{4, 20}, {20, 24}, {8, 38}, {18, 14}, {6, 30}, {21, 33}}
	for i := 0; i < 3; i++ {
		p := spark[(i*2+f)%len(spark)]
		b.set(p[0], p[1], c([]uint32{0xfff4b0, 0xffffff, 0xb0e8ff, 0xffd0f0}[(i+v)%4]))
		if (i+f)%2 == 0 { // a little cross: the flash of a mirror
			b.set(p[0]-1, p[1], c(0xffffff))
			b.set(p[0]+1, p[1], c(0xffffff))
		}
	}
}

// drawFireplaceFrame: the built-in fireplace with the flames drawn again in each frame (the tongues rise and fall).
func drawFireplaceFrame(b *pixBuf, k *PropKind, v, f int) {
	k.Draw(b, v)
	drawHearth(b, f) // the same hearth as the still sprite (frame 0), fire_art.go
}

// drawFountainFrame: the fountain with running water: the drops go down the jets and the glints of the basin turn.
func drawFountainFrame(b *pixBuf, k *PropKind, v, f int) {
	stone, stoneD, stoneL := c(0xb8b4aa), c(0x7a766e), c(0xdcd8cc)
	b.ellipse(50, 54, 49, 12, stoneD)
	b.ellipse(50, 51, 49, 12, stone)
	b.ellipse(50, 50, 45, 10, stoneL)
	b.ellipse(50, 51, 42, 9, c(0x4a90c8))
	b.ellipse(50, 51, 36, 7, c(0x68b0e0))
	for i := 0; i < 14; i++ {
		a := float64(i)*0.9 + float64(f)*0.225
		b.set(50+int(math.Cos(a)*30), 51+int(math.Sin(a)*5), c(0xc8ecff))
	}
	for i := 0; i < 3; i++ { // rings where the jets fall into the basin
		a := float64(i)*2.1 + float64(f)*0.4
		b.set(50+int(math.Cos(a)*18), 51+int(math.Sin(a)*3), c(0xe8f8ff))
	}
	b.rect(46, 22, 8, 26, stone)
	b.rect(46, 22, 3, 26, stoneL)
	b.rect(52, 22, 2, 26, stoneD)
	b.ellipse(50, 30, 18, 6, stone)
	b.ellipse(50, 29, 15, 4.4, c(0x68b0e0))
	b.set(44+f*3, 29, c(0xc8ecff))
	b.rect(49, 8, 2, 16, c(0xc8ecff))
	for _, o := range []int{-8, -5, 5, 8} {
		b.line(50, 10, 50+o, 26, c(0x9ad0f0))
		for j := 0; j < 2; j++ { // two drops running down each jet
			t := (float64(f) + float64(j)*2) / 4
			y := 10 + int(t*16)
			b.set(50+int(math.Round(float64(o)*t)), y, c(0xf4fcff))
		}
	}
	b.set(49, 8+f*4, c(0xf4fcff)) // the water rising in the spout
	b.ellipse(50, 9, 3, 2, stoneL)
}

// drawGondolaFrame: the gondola of the gondolier (variant 1): the oar sweeps back and forth and he leans into it.
func drawGondolaFrame(b *pixBuf, k *PropKind, v, f int) {
	if v%2 == 0 {
		k.Draw(b, v)
		return
	}
	const W = 104
	flip := func(x int) int { return W - 1 - x }
	sw := math.Cos(float64(f) / 6 * 2 * math.Pi) // every frame differs from the next
	// draw the boat without its gondolier: the plain boat of variant 0, mirrored like variant 1
	tmp := newPix(b.w, b.h)
	tmp.ox, tmp.oy = b.ox, b.oy
	k.Draw(tmp, 0)
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			sx := W - 1 - x + 2*b.ox // mirror around the design width (W) inside the padded buffer
			if sx < 0 || sx >= b.w {
				continue
			}
			i, j := (y*b.w+x)*4, (y*tmp.w+sx)*4
			copy(b.p[i:i+4], tmp.p[j:j+4])
		}
	}
	lean := 0
	if sw > 0.5 {
		lean = 1
	}
	set := func(x, y int, cc color.RGBA) { b.set(flip(x), y, cc) }
	for y := 3; y < 21; y++ {
		cc := rgb(0xf0ece0)
		if y >= 6 && y <= 13 && (y/2)%2 == 0 {
			cc = rgb(0x1a2a5a)
		}
		if y > 13 {
			cc = rgb(0x1a1a22)
		}
		dx := 0
		if y < 12 {
			dx = lean
		}
		set(14+dx, y, cc)
		set(15+dx, y, cc)
	}
	set(14+lean, 2, rgb(0xe8c890))
	set(15+lean, 2, rgb(0xe8c890))
	for x := 12; x < 18; x++ {
		set(x+lean, 1, rgb(0xe8d070))
	}
	set(13+lean, 0, rgb(0xc83a30))
	ex := 30 + int(math.Round(sw*5)) // the blade in the water, forward and back
	b.line(flip(16+lean), 9, flip(ex), 30, rgb(0x8a6a40))
	if sw < -0.3 { // a ripple where the blade pushes
		b.set(flip(ex+2), 31, rgb(0xc8e8f0))
	}
}

// slotFrame: frames 0-3 waiting (the marquee chases), 4-7 playing (the reels spin, the lights run fast).
func slotFrame(t, ph float64, busy bool) int {
	if busy {
		return 4 + int(t*12+ph)%4
	}
	return int(t*2.5+ph) % 4
}

// drawSlotFrame: the slot machine with its chasing lights and, in the playing frames, the reels spinning.
func drawSlotFrame(b *pixBuf, k *PropKind, v, f int) {
	body := []color.RGBA{c(0xb02030), c(0x2a50b8)}[v%2]
	b.ellipse(12, 39, 11, 2.4, color.RGBA{0, 0, 0, 80})
	b.rect(2, 4, 18, 36, body)
	b.rect(2, 4, 18, 2, shade(body, 1.35))
	b.rect(2, 6, 2, 34, shade(body, 1.2))
	b.rect(18, 6, 2, 34, shade(body, 0.7))
	b.rect(4, 8, 14, 5, c(0xf0d040)) // marquee
	for i, x := 0, 5; x < 17; i, x = i+1, x+3 {
		cc := c(0xa02020)
		if (i+f)%2 == 0 {
			cc = c(0xff5050)
		}
		b.set(x, 10, cc)
	}
	b.rect(4, 15, 14, 10, c(0x101018)) // reels
	spin := f >= 4
	for i, col := range []uint32{0xf04040, 0xf0e040, 0x40d060} {
		b.rect(5+i*4, 16, 3, 8, c(0xf4f4f0))
		if spin {
			y := 16 + (f*3+i*2)%8 // the symbols run down the reel
			b.rect(6+i*4, y, 1, 2, c(col))
			b.rect(6+i*4, 16+(f*3+i*2+4)%8, 1, 2, c([]uint32{0x40d060, 0xf04040, 0xf0e040}[i]))
			b.rect(5+i*4, 19, 3, 1, c(0xd8d8d0))
		} else {
			b.rect(6+i*4, 18, 1, 3, c(col))
		}
	}
	if f%4 == 1 || spin { // the screen glow
		b.rect(4, 15, 14, 1, c(0x40406a))
	}
	b.rect(4, 27, 14, 3, c(0x2a2a34)) // coin slot
	b.rect(8, 28, 6, 1, c(0xf0d040))
	b.rect(4, 34, 14, 4, c(0x14141c)) // payout tray
	lever := 12
	if spin && f%2 == 1 {
		lever = 16 // the lever pulled down
	}
	b.rect(21, lever, 2, 24-lever, c(0x9a9aa6))
	b.ellipse(22, float64(lever-1), 2, 2, c(0xe03030))
}

// drawRouletteFrame: the roulette table; the wheel turns (red and black pockets, the gold cross) and the white ball
// runs round the other way (16 frames: the ball goes once round, the wheel a quarter of a turn).
func drawRouletteFrame(b *pixBuf, k *PropKind, v, f int) {
	b.ellipse(35, 40, 33, 4, color.RGBA{0, 0, 0, 80})
	b.rect(4, 30, 62, 10, c(0x5a3a22)) // wooden edge
	b.ellipse(35, 28, 33, 14, c(0x7a4a28))
	b.ellipse(35, 26, 31, 12, c(0x1e7a3e)) // green felt
	const wx, wy = 22.0, 25.0
	b.ellipse(wx, wy, 10, 7, c(0x2a1a10)) // the bowl
	rot := float64(f) / 16 * (math.Pi / 2)
	for y := -6; y <= 6; y++ { // the pockets: a ring of 16 alternating red and black, one green
		for x := -9; x <= 9; x++ {
			nx, ny := float64(x)/8.5, float64(y)/5.6
			d := nx*nx + ny*ny
			if d > 1 || d < 0.42 {
				continue
			}
			a := math.Atan2(ny, nx) - rot
			p := int(math.Floor(a/(2*math.Pi)*16+64)) % 16
			cc := c(0xd02828)
			switch {
			case p == 0:
				cc = c(0x20a050)
			case p%2 == 1:
				cc = c(0x14141a)
			}
			b.set(int(wx)+x, int(wy)+y, cc)
		}
	}
	b.ellipse(wx, wy, 3.6, 2.4, c(0x8a5a20)) // the hub
	for i := 0; i < 2; i++ {                 // the gold cross turning with the wheel
		a := rot*2 + float64(i)*math.Pi/2
		dx, dy := math.Cos(a)*3.4, math.Sin(a)*2.2
		b.line(int(math.Round(wx-dx)), int(math.Round(wy-dy)), int(math.Round(wx+dx)), int(math.Round(wy+dy)), c(0xe8c040))
	}
	ba := -float64(f) / 16 * 2 * math.Pi // the ball, the other way round, on the outer track
	bx, by := wx+math.Cos(ba)*8.6, wy+math.Sin(ba)*5.8
	b.set(int(math.Round(bx)), int(math.Round(by)), c(0xffffff))
	b.set(int(math.Round(bx))+1, int(math.Round(by)), c(0xe0e0e8))
	for x := 40; x < 62; x += 6 { // number grid
		b.rect(x, 21, 5, 3, c(0xf4f4f0))
		b.rect(x, 25, 5, 3, c(0xd02828))
	}
	for _, p := range [][3]int{{44, 31, 0xe83030}, {47, 30, 0x3050e0}, {52, 32, 0xf0f0f0}, {56, 30, 0x20b050}} {
		b.ellipse(float64(p[0]), float64(p[1]), 2.4, 1.4, c(uint32(p[2])))
	}
}

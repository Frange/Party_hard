package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// partyColors cycle in the neighbours' disco.
var partyColors = []uint32{0xe03a8a, 0x3a6ae0, 0x3ad0a8, 0xf0c030, 0x9a4ae0, 0xe05a30}

// drawBedroom is the title backdrop: our hero tries to sleep in his bed,
// tossing and turning, while the house across the street throws a party that
// can be seen through the window.
func (g *Game) drawBedroom(dst *ebiten.Image) {
	t := g.time
	quiet := g.bedQuiet     // the ending: the neighbours are asleep too
	gold := g.allTrophies() // everything won: a golden room, the party in the window is dead and he sleeps in peace
	wall, floor := rgb(0x16202c), rgb(0x0d1420)
	if gold {
		wall, floor = rgb(0x3a2c10), rgb(0x241a08)
	}
	fillRect(dst, 0, 0, screenW, 300, wall)
	for y := 0.0; y < 300; y += 3 {
		if int(y/3)%4 == 0 {
			fillRect(dst, 0, y, screenW, 1, shade(wall, 0.92))
		}
	}
	fillRect(dst, 0, 300, screenW, screenH-300, floor)
	fillRect(dst, 0, 298, screenW, 3, rgb(0x080c14))
	for x := 0.0; x < screenW; x += 40 { // floorboards
		fillRect(dst, x, 301, 1, screenH-301, shade(floor, 0.8))
	}

	// ---- window onto the neighbours' party -------------------------------
	ox := -212.0 // the window sits over the bed, in the middle of the room
	fr := func(x, y, w, h float64, c color.RGBA) { fillRect(dst, x+ox, y, w, h, c) }
	fe := func(x, y, rx, ry float64, c color.RGBA) { filledEllipse(dst, x+ox, y, rx, ry, c) }
	frameCol := rgb(0x08101a)
	if gold {
		frameCol = rgb(0xc8921c)
	}
	fr(552, 22, 356, 190, frameCol)
	gx, gy, gw, gh := 566.0, 36.0, 328.0, 162.0
	for y := 0.0; y < gh; y += 2 {
		fr(gx, gy+y, gw, 2, mixc(rgb(0x1a1440), rgb(0x0d1428), y/gh))
	}
	for i := 0; i < 26; i++ { // stars
		sx, sy := gx+float64((i*97)%int(gw)), gy+float64((i*53)%60)
		a := uint8(90 + 80*math.Abs(math.Sin(t*1.3+float64(i))))
		fr(sx, sy, 1, 1, rgba(0xdce4ff, a))
	}
	fe(618, 66, 9, 9, rgb(0xe8e6d4)) // moon
	fe(614, 63, 8, 8, rgb(0x1a1440))
	// the house across the street
	fr(586, 92, 290, 106, rgb(0x1c1a2c))
	fr(580, 84, 302, 10, rgb(0x2e2a44))
	fr(580, 84, 302, 2, rgb(0x46406a))
	for i := 0; i < 5; i++ { // small windows upstairs, some lit
		wx := 600.0 + float64(i)*56
		col := rgb(0x2a2a44)
		if (i+int(t*2))%3 == 0 {
			col = rgb(0xe8c860)
		}
		fr(wx, 98, 14, 10, col)
	}
	// the big living-room window with the disco inside
	rx, ry, rw, rh := 606.0, 112.0, 250.0, 80.0
	fr(rx-3, ry-3, rw+6, rh+6, rgb(0x0a0814))
	fr(rx, ry, rw, rh, rgb(0x2a1038))
	ci := int(t * 2.4)
	for cy := 0; cy < 3; cy++ { // flashing floor
		for cx := 0; cx < 12; cx++ {
			c := partyColors[(cx+cy*2+ci)%len(partyColors)]
			fr(rx+float64(cx)*rw/12, ry+rh-24+float64(cy)*8, rw/12-1, 7, rgba(c, 200))
		}
	}
	bx := rx + rw/2 // disco ball
	fr(bx, ry, 1, 12, rgb(0x8a8aa0))
	fe(bx, ry+18, 7, 7, rgb(0xd8dcec))
	for k := 0; k < 8; k++ {
		a := t*3 + float64(k)
		fr(bx+math.Cos(a)*(11+float64(k%3)*6), ry+18+math.Sin(a)*(7+float64(k%2)*8), 2, 2, rgba(partyColors[(k+ci)%len(partyColors)], 220))
	}
	for i := 0; i < 6; i++ { // dancers
		dx := rx + 24 + float64(i)*42
		if gold { // the neighbours are all dead on the floor
			shirt := shade(rgb(partyColors[(i*2+1)%len(partyColors)]), 0.6)
			ly := ry + rh - 8
			fr(dx-7, ly-3, 12, 6, shirt)
			fr(dx-12, ly-2, 5, 4, rgb(0x14141c))
			fe(dx+8, ly, 3.2, 3.2, rgb(0xc8a078))
			fr(dx-3, ly-8, 6, 1, rgba(0xb01818, 200)) // a pool of blood
			continue
		}
		bob := math.Abs(math.Sin(t*6+float64(i)*1.7)) * 4
		by := ry + rh - 12 - bob
		shirt := rgb(partyColors[(i*2+1)%len(partyColors)])
		fr(dx-3, by-12, 6, 9, shirt)
		fr(dx-3, by-3, 2, 6, rgb(0x14141c))
		fr(dx+1, by-3, 2, 6, rgb(0x14141c))
		fe(dx, by-16, 3.2, 3.4, rgb(0xe0b088))
		up := math.Sin(t*6+float64(i)*1.7) > 0
		if up {
			fr(dx-6, by-20, 2, 10, shirt)
			fr(dx+4, by-14, 2, 8, shirt)
		} else {
			fr(dx-6, by-14, 2, 8, shirt)
			fr(dx+4, by-20, 2, 10, shirt)
		}
	}
	for _, sx := range []float64{rx + 4, rx + rw - 18} { // speakers
		fr(sx, ry+rh-40, 14, 30, rgb(0x14121c))
		pul := 2 + 2*math.Abs(math.Sin(t*8))
		fe(sx+7, ry+rh-18, pul+2, pul+2, rgb(0x3a3648))
	}
	for k, c := range []uint32{0xe03a8a, 0x3ad0a8, 0xf0c030} { // balloons
		fe(872+float64(k)*3, 70+float64(k)*14+math.Sin(t*2+float64(k))*3, 7, 9, rgba(c, 230))
		fr(872+float64(k)*3, 79+float64(k)*14, 1, 20-float64(k)*8, rgba(0xc8c8d8, 140))
	}
	if quiet { // the house across the street: lights out, music off
		fr(586, 92, 290, 106, rgb(0x14122a))
		fr(580, 84, 302, 10, rgb(0x1e1a34))
		for i := 0; i < 5; i++ {
			fr(600+float64(i)*56, 98, 14, 10, rgb(0x1c1a30))
		}
		fr(rx-3, ry-3, rw+6, rh+6, rgb(0x0a0814))
		fr(rx, ry, rw, rh, rgb(0x120a1c))
		fr(rx+rw/2-26, ry+rh-30, 52, 26, rgb(0x1a1226))
		fr(rx+rw/2-22, ry+rh-26, 44, 18, rgb(0x2a2a4a)) // a pile of tired people on the sofa, asleep
	}
	// colour spill of the party into the night
	pc := partyColors[ci%len(partyColors)]
	// curtains and frame
	for k := 0.0; k < 30; k += 6 {
		fr(gx+k, gy, 5, gh, shade(rgb(0x2e1c3a), 0.85+0.2*math.Sin(k)))
		fr(gx+gw-30+k, gy, 5, gh, shade(rgb(0x2e1c3a), 0.85+0.2*math.Cos(k)))
	}
	fr(728, 36, 4, 162, rgb(0x08101a))
	fr(540, 208, 380, 10, rgb(0x2a1e18))
	fr(540, 208, 380, 2, rgb(0x4a3528))

	// ---- the bed, in the middle, with the hero tossing and turning --------
	const bx0, bx1, by0, by1 = 332.0, 712.0, 268.0, 440.0
	fillRect(dst, bx0-52, 330, 40, 70, rgb(0x2e2119)) // bedside table
	fillRect(dst, bx0-52, 330, 40, 4, rgb(0x4a3528))
	fillRect(dst, bx0-44, 352, 24, 16, rgb(0x14100c))
	fillRect(dst, bx0-44, 306, 24, 18, rgb(0x0a0a0e)) // alarm clock
	clock := "3:47"
	if quiet {
		clock = "6:30"
	}
	g.drawText(dst, clock, bx0-42, 310, rgb(0xff3030), 0.9)
	fillRect(dst, bx0-48, 398, 6, 20, rgb(0x2a1e18))
	fillRect(dst, bx0-22, 398, 6, 20, rgb(0x2a1e18))
	fillRect(dst, bx0-12, 250, 14, 190, rgb(0x3a2a1e)) // headboard
	fillRect(dst, bx0-12, 250, 14, 3, rgb(0x5a4030))
	fillRect(dst, bx0+2, by0, bx1-bx0, by1-by0, rgb(0xaab4cc)) // mattress
	fillRect(dst, bx0+2, by0, bx1-bx0, 6, rgb(0xc4cce0))
	fillRect(dst, bx0+2, by1-6, bx1-bx0, 6, rgb(0x6a7690))
	fillRect(dst, bx1, 300, 12, 150, rgb(0x3a2a1e)) // footboard
	fillRect(dst, bx0-10, by1+4, 8, 18, rgb(0x2a1e18))
	fillRect(dst, bx1, by1+4, 8, 18, rgb(0x2a1e18))
	fillRect(dst, bx0+8, by0+10, 58, by1-by0-20, rgb(0xa2aabb)) // pillow, as wide as the bed
	fillRect(dst, bx0+8, by0+10, 58, 4, rgb(0xc4ccdc))
	fillRect(dst, bx0+8, by1-14, 58, 4, rgb(0x8a92a4))
	fillRect(dst, bx0+62, by0+10, 4, by1-by0-20, rgb(0x8a92a4))
	if !g.bedEmpty {
		g.drawSleeper(dst, bx0-30, (by0+by1)/2, t)
	}
	// blanket over the legs, moving with every turn
	wob := math.Sin(t*2) * 1.5
	fillRect(dst, 600, 320+wob, bx1-600, 110, rgb(0x3a6aa0))
	fillRect(dst, 600, 320+wob, bx1-600, 4, rgb(0x6a9ac8))
	for x := 624.0; x < bx1-10; x += 44 {
		fillRect(dst, x, 326+wob, 2, 98, rgb(0x2c5688))
	}
	fillRect(dst, 600, 424, bx1-600, 6, rgb(0x244a74))

	if gold {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x2a1800, 70))
		for i := 0; i < 46; i++ { // golden glitter drifting over the room
			gx2 := math.Mod(float64(i*137)+t*6*(1+float64(i%3)), screenW)
			gy2 := math.Mod(float64(i*71)+t*(10+float64(i%5)*4), screenH)
			a := uint8(70 + 150*math.Abs(math.Sin(t*2+float64(i))))
			fillRect(dst, gx2, gy2, 2, 2, rgba(0xffe070, a))
		}
	} else {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x02040a, 100))
	}
	if !quiet {
		fillRect(dst, bx0+2, by0, bx1-bx0, by1-by0, rgba(pc, 16)) // the party's lights on the sheets
	}
	dst.DrawImage(g.as.vignette, nil)
	if !g.bedEmpty && !g.bedNoTalk {
		g.drawSleeperTalk(dst)
	}
}

// drawSleeper draws the hero lying in bed; every few seconds he flips over
// from his back to his front and mutters, never falling asleep.
func (g *Game) drawSleeper(dst *ebiten.Image, cx, cy, t float64) {
	cs := g.as.playerCS
	if cs == nil {
		return
	}
	const period, turn = 6.8, 0.9
	n := int(t / period)
	ph := math.Mod(t, period)
	calm := g.allTrophies() || g.bedQuiet
	if calm {
		n, ph = 0, 0 // he lies on his back, not turning
	}
	flip := 1.0
	lift := 0.0
	view := vFront
	if n%2 == 1 {
		view = vBack
	}
	if ph > period-turn { // turning over
		u := (ph - (period - turn)) / turn
		flip = math.Cos(math.Pi * u)
		lift = math.Sin(math.Pi*u) * 14
		if u > 0.5 {
			if n%2 == 0 {
				view = vBack
			} else {
				view = vFront
			}
		}
	}
	im := cs.frameImg(AnimSleep, view, 0)
	if im == nil {
		return
	}
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	long := math.Max(w, h)
	sc := math.Max(2, math.Floor(340/long)) // nearly as long as the bed
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(sc, sc)
	if h > w { // a standing sprite: lay it down with the head on the pillow
		op.GeoM.Scale(flip, 1)
		op.GeoM.Rotate(-math.Pi / 2)
	} else {
		op.GeoM.Scale(1, flip)
	}
	op.GeoM.Translate(cx+long*sc/2, cy-lift+math.Sin(t*1.6)*1.5)
	dst.DrawImage(im, op)
	// above the head: first a few Zs, then he wakes up and complains; drawn
	// last, over the night tint, so it stays sharp
	g.talkX, g.talkY, g.talkT, g.talkN = cx+14*sc, cy-lift, t, n
	ph2 := math.Mod(t, period)
	switch {
	case calm:
		g.talkKind = 1
	case ph2 < 2.3:
		g.talkKind = 1
	case ph2 < period-turn:
		g.talkKind = 2
	default:
		g.talkKind = 3
	}
}

func (g *Game) drawSleeperTalk(dst *ebiten.Image) {
	hx0, hy0, t := g.talkX, g.talkY, g.talkT
	switch g.talkKind {
	case 1:
		for i := 0; i < 3; i++ {
			f := math.Mod(t*0.55+float64(i)/3, 1)
			g.drawText(dst, "Z", math.Round(hx0-8+f*34+float64(i)*3), math.Round(hy0-44-f*40), rgba(0xe8f0ff, uint8(240*(1-f))), float64(2+i))
		}
	case 2:
		g.speechBubble(dst, hx0, hy0-48, sleeperLines[g.talkN%len(sleeperLines)])
	case 3:
		g.speechBubble(dst, hx0, hy0-48, "¡¡Grr!!")
	}
}

var sleeperLines = []string{
	"quiero dormiiiiiiir...", "¡que paren la música ya!", "así no hay quien se duerma", "tengo sueño...",
	"son las 3:47 de la mañana...", "mañana tengo que madrugar", "¿es que no tienen vecinos?", "¡un poco de respeto, por favor!",
	"ni con la almohada en la cabeza", "voy a llamar a la policía...", "cuento ovejas y están bailando",
}

// speechBubble draws a small comic balloon centred over (x, y).
func (g *Game) speechBubble(dst *ebiten.Image, x, y float64, s string) {
	w := math.Round(g.textW(s, 2)) + 24
	bx := math.Round(math.Max(8, math.Min(screenW-8-w, x-w/2)))
	by := math.Round(y - 34)
	fillRect(dst, bx-2, by-2, w+4, 34, rgb(0x14121c))
	fillRect(dst, bx, by, w, 30, rgb(0xfffaf0))
	fillRect(dst, math.Round(x)-4, by+30, 9, 4, rgb(0xfffaf0)) // tail towards the head
	fillRect(dst, math.Round(x)-2, by+34, 5, 3, rgb(0xfffaf0))
	g.drawTextC(dst, s, bx+w/2, by-1, rgb(0x14102a), 2)
}

var _ = color.RGBA{}

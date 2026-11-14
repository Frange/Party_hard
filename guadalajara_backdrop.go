package main

import "math"

// ---------------------------------------------------------------------------
// The painted backdrop of GUADALAJARA (guadalajara.go): 960 x gdlBackH, made once. Dusk sky (purple above, pink and
// orange on the mountains), the mountains, and the front row standing on the square:
//   x   0-250  a wooded hill; in front of it the pink house of the
//              "Pastelería Mariam" (sign, striped awning, lit shop window) and the orange house of "Miel de la
//              Alcarria" (sign with a honey jar, balconies)
//   x 258-498  the renaissance palace: red roof, the gallery of arches under it, the stone facade studded with
//              diamond points, framed windows and the decorated portal with its crest
//   x 462-624  behind them, the green hill with the black bull silhouette on top, against the sky (the top left of
//              the screen is under the HUD, so the bull stands here)
//   x 556-702  the stone church with its two little towers (red pyramid roofs, bells), rose window and portal
//   x 708-822  the pantheon: pale stone, the grey dome with its lantern, the rose window and the steps
//   x 826-960  the purple stadium up on its terrace, with two floodlights and the purple crest, and the diagonal
//              flight of steps that climbs to it from the square
// The bull is only a silhouette (no brand). Drawn with the muted dusk palette of the game; no picture is copied.
// ---------------------------------------------------------------------------

var (
	gdlSkyTop  = rgb(0x3a2866)
	gdlSkyMid  = rgb(0xb8648a)
	gdlSkyLow  = rgb(0xe8a064)
	gdlMount   = rgb(0x6a4e86)
	gdlMountD  = rgb(0x4e3a6e)
	gdlHill    = rgb(0x3e6a3e)
	gdlHillD   = rgb(0x2c4e30)
	gdlStone   = rgb(0xdcc298)
	gdlStoneD  = rgb(0xa88e6a)
	gdlStoneL  = rgb(0xf0dcb4)
	gdlPale    = rgb(0xe6dece)
	gdlPink    = rgb(0xd894b4)
	gdlOrange  = rgb(0xd88a4a)
	gdlPurple  = rgb(0x6a3a9a)
	gdlPurpleD = rgb(0x46246a)
	gdlPurpleL = rgb(0x9a6ac8)
)

func guadalajaraBackdrop() *pixBuf {
	if gdlBackBuf != nil {
		return gdlBackBuf
	}
	W, H := 960, int(gdlBackH)
	b := newPix(W, H)
	// the sky: purple above, pink, then orange towards the mountains, with streaks of cloud
	for y := 0; y < H; y++ {
		t := math.Min(1, float64(y)/70)
		c := mixc(gdlSkyTop, gdlSkyMid, math.Min(1, t*1.6))
		if t > 0.62 {
			c = mixc(gdlSkyMid, gdlSkyLow, (t-0.62)/0.38)
		}
		b.rect(0, y, W, 1, c)
	}
	for i := 0; i < 16; i++ {
		x, y := phHash(i, 23)%W, 6+phHash(i, 29)%42
		w := 26 + phHash(i, 31)%80
		b.rect(x, y, w, 1, mixc(rgb(0xf0a080), gdlSkyTop, float64(y)/90))
		b.rect(x+5, y+1, w-10, 1, mixc(rgb(0xd07890), gdlSkyTop, float64(y)/90))
	}
	// the mountains
	for x := 0; x < W; x++ {
		fx := float64(x)
		h1 := 52 + 6*math.Sin(fx/66+0.5) + 4*math.Sin(fx/21) + 2*math.Sin(fx/8)
		h2 := 64 + 5*math.Sin(fx/47+1) + 3*math.Sin(fx/15)
		b.rect(x, int(h1), 1, H, gdlMount)
		b.rect(x, int(h2), 1, H, gdlMountD)
	}
	// the far city: little roofs and dark treetops
	for i := 0; i < 70; i++ {
		x := phHash(i, 37) % W
		y := 72 + phHash(i, 41)%14
		if i%3 == 0 {
			b.ellipse(float64(x), float64(y), 7, 6, alcTreeDark)
			continue
		}
		w := 14 + phHash(i, 43)%16
		b.rect(x, y+4, w, 30, shade(alcWhite, 0.6))
		b.rect(x-1, y, w+2, 4, shade(alcRoof, 0.7))
	}
	b.rect(0, 92, W, H-92, shade(alcWhite, 0.52))

	gdlLeftHill(b)
	gdlBullHill(b)
	gdlCakeShop(b)
	gdlHoneyShop(b)
	gdlPalace(b)
	alcCypress(b, 548, 146)
	gdlChurch(b)
	alcCypress(b, 704, 146)
	gdlPantheon(b)
	gdlStadium(b)
	// the kerb where the facades meet the square
	b.rect(0, H-4, W, 4, gdlStoneD)
	b.rect(0, H-4, W, 1, gdlStoneL)
	gdlBackBuf = b
	return b
}

// the wooded hill on the left, behind the houses, with a path
func gdlLeftHill(b *pixBuf) {
	for x := 0; x < 262; x++ {
		fx := float64(x)
		top := 26 + 0.0016*(fx-56)*(fx-56) + 2*math.Sin(fx/9)
		if top > 90 {
			top = 90
		}
		for y := int(top); y < 120; y++ {
			c := gdlHill
			if (x+y)%7 == 0 || y > int(top)+30 {
				c = gdlHillD
			}
			b.set(x, y, c)
		}
		b.set(x, int(top), shade(gdlHill, 1.25))
	}
	for i := 0; i < 40; i++ { // trees on the slope
		x := phHash(i, 47) % 250
		top := 26 + 0.0016*float64((x-56)*(x-56))
		y := int(top) + 6 + phHash(i, 53)%30
		b.ellipse(float64(x), float64(y), 5, 4, alcTreeDark)
		b.ellipse(float64(x)-1, float64(y)-1, 3, 2, shade(gdlHill, 1.15))
	}
	for i := 0; i < 30; i++ { // the winding path
		x := 120 + i*4
		y := 52 + i + int(3*math.Sin(float64(i)/3))
		b.rect(x, y, 3, 1, rgb(0xb89a78))
	}
}

// the green hill that shows between the palace and the church (the top left of the screen is under the HUD), with the
// black bull silhouette standing on its top against the sky
func gdlBullHill(b *pixBuf) {
	for x := 462; x < 624; x++ {
		dx := float64(x - 528)
		top := 32 + 0.0045*dx*dx + 1.5*math.Sin(float64(x)/7)
		if top > 100 {
			top = 100
		}
		for y := int(top); y < 124; y++ {
			c := gdlHill
			if (x*3+y)%9 == 0 || y > int(top)+24 {
				c = gdlHillD
			}
			b.set(x, y, c)
		}
		b.set(x, int(top), shade(gdlHill, 1.25))
	}
	for i := 0; i < 14; i++ { // a few trees on its slopes
		x := 470 + phHash(i, 59)%140
		dx := float64(x - 528)
		y := int(32+0.0045*dx*dx) + 8 + phHash(i, 61)%20
		b.ellipse(float64(x), float64(y), 5, 4, alcTreeDark)
		b.ellipse(float64(x)-1, float64(y)-1, 3, 2, shade(gdlHill, 1.15))
	}
	// the bull: body, head lowered, horns, legs and tail, all black
	bull := rgb(0x0e0c10)
	bx, by := 508, 10                                      // the top-left corner of its box (40 x 22), standing on the hilltop
	b.ellipse(float64(bx+18), float64(by+10), 13, 6, bull) // body
	b.ellipse(float64(bx+28), float64(by+8), 6, 5, bull)   // the hump
	b.ellipse(float64(bx+35), float64(by+11), 4, 4, bull)  // the head
	b.rect(bx+36, by+13, 3, 3, bull)                       // the muzzle
	b.line(bx+33, by+7, bx+31, by+3, bull)                 // the horns
	b.line(bx+37, by+7, bx+39, by+3, bull)
	for _, x := range []int{bx + 8, bx + 12, bx + 24, bx + 28} { // legs
		b.rect(x, by+14, 2, 7, bull)
	}
	b.line(bx+5, by+8, bx+2, by+16, bull) // the tail
	b.rect(bx+1, by+16, 2, 2, bull)
}

// the pink house of the cake shop: the "Pastelería Mariam" sign, the striped awning and the lit shop window
func gdlCakeShop(b *pixBuf) {
	x0, x1 := 0, 112
	alcRoofBand(b, x0, x1, 58, 9)
	b.rect(84, 50, 8, 9, rgb(0xc8b8a4)) // a chimney
	alcWall(b, x0, 68, x1, 146, gdlPink, 91)
	for i := 0; i < 3; i++ {
		x := 12 + i*34
		alcWindow(b, x, 74, 10, 12, i == 1, gdlStoneL)
		alcBalcony(b, x-2, 86, 14)
	}
	// the sign: two lines on a light board with a dark frame (low on the facade: the HUD covers the top left)
	b.rect(10, 110, 92, 16, rgb(0x5a2a48))
	b.rect(11, 111, 90, 14, rgb(0xf4e0ec))
	gdlTextC(b, "PASTELERÍA", 56, 113, rgb(0x6a2a5a)) // (its accent sits on row 111, inside the board)
	gdlTextC(b, "MARIAM", 56, 119, rgb(0x6a2a5a))
	// the awning: pink and white stripes with a scalloped hem
	for x := 6; x < 106; x++ {
		c := rgb(0xf4ecf0)
		if (x/5)%2 == 0 {
			c = rgb(0xb84a8a)
		}
		b.rect(x, 127, 1, 6, c)
		if x%5 < 3 {
			b.set(x, 133, c)
		}
	}
	// the shop window with the cakes, lit, and the door
	b.rect(8, 134, 66, 12, rgb(0x5a3a3a))
	b.rect(10, 135, 62, 10, alcLit)
	for i := 0; i < 6; i++ {
		x := 13 + i*10
		b.rect(x, 139, 7, 5, rgb([]uint32{0xf4f0e8, 0xc86a8a, 0x8a5a3a, 0xf0c060}[i%4]))
		b.set(x+3, 138, rgb(0xd83a3a))
	}
	b.rect(80, 134, 18, 12, alcWood)
	b.rect(82, 135, 14, 5, alcLit)
}

// the orange house of the honey shop: the "Miel de la Alcarria" sign with a honey jar, balconies, door and window
func gdlHoneyShop(b *pixBuf) {
	x0, x1 := 116, 252
	alcRoofBand(b, x0, x1, 54, 9)
	alcWall(b, x0, 64, x1, 146, gdlOrange, 97)
	for i := 0; i < 4; i++ {
		x := x0 + 12 + i*32
		alcWindow(b, x, 70, 10, 12, i == 2, gdlStoneL)
		b.rect(x-3, 70, 2, 12, rgb(0x3a6a4a)) // shutters
		b.rect(x+11, 70, 2, 12, rgb(0x3a6a4a))
		alcBalcony(b, x-3, 82, 16)
	}
	// the sign: a dark board, a honey jar and the name (low on the facade: the HUD covers the top left)
	b.rect(x0+8, 110, 120, 15, rgb(0x5a3418))
	b.rect(x0+9, 111, 118, 13, rgb(0x8a5424))
	b.rect(x0+13, 113, 8, 9, rgb(0xd8901e)) // the jar
	b.rect(x0+13, 113, 2, 9, rgb(0xf0c050))
	b.rect(x0+13, 112, 8, 2, rgb(0xf0ece0))
	gdlText(b, "MIEL DE LA ALCARRIA", x0+26, 115, rgb(0xf8d878))
	// the shop: a window, lit, with jars on its shelves, and the door
	b.rect(x0+10, 127, 70, 19, rgb(0x5a3a2a))
	b.rect(x0+12, 128, 66, 18, alcLit)
	for i := 0; i < 8; i++ { // jars on the shelves
		x := x0 + 15 + i*8
		b.rect(x, 130, 5, 6, rgb(0xd8901e))
		b.rect(x, 139, 5, 6, rgb(0xc07818))
	}
	b.rect(x0+12, 137, 66, 1, alcWood)
	b.rect(x0+96, 124, 20, 22, alcWood)
	b.ellipse(float64(x0+106), 124, 10, 4, alcWood)
}

// the renaissance palace: red roof, the gallery of arches, the facade studded with diamond points, framed windows,
// corner pilasters and the decorated portal with its crest
func gdlPalace(b *pixBuf) {
	x0, x1 := 258, 498
	mid := (x0 + x1) / 2
	alcRoofBand(b, x0+4, x1-4, 26, 9)
	alcWall(b, x0, 36, x1, 146, gdlStone, 103)
	// the gallery of small arches under the roof
	b.rect(x0, 36, x1-x0, 2, gdlStoneL)
	for x := x0 + 6; x < x1-8; x += 12 {
		b.rect(x, 42, 8, 10, alcDark)
		b.ellipse(float64(x+4), 42, 4, 3, alcDark)
		b.rect(x+3, 44, 2, 8, rgb(0x5a4038)) // the little column behind
	}
	b.rect(x0, 53, x1-x0, 2, gdlStoneL)
	b.rect(x0, 55, x1-x0, 1, gdlStoneD)
	// the diamond points all over the facade (a light and a dark side each)
	for y := 60; y < 140; y += 6 {
		for x := x0 + 4 + (y/6%2)*3; x < x1-3; x += 6 {
			b.set(x, y, gdlStoneL)
			b.set(x+1, y, shade(gdlStone, 1.05))
			b.set(x, y+1, shade(gdlStone, 0.92))
			b.set(x+1, y+1, gdlStoneD)
		}
	}
	// corner pilasters
	for _, x := range []int{x0, x1 - 6} {
		b.rect(x, 36, 6, 110, gdlStoneL)
		b.rect(x+5, 36, 1, 110, gdlStoneD)
	}
	// two rows of framed windows (the portal axis left free)
	for i := 0; i < 9; i++ {
		x := x0 + 18 + i*25
		if x > mid-30 && x < mid+22 {
			continue
		}
		alcWindow(b, x, 64, 9, 15, phHash(i, 107)%3 == 0, gdlStoneL)
		b.rect(x-3, 61, 15, 2, gdlStoneD)
		alcWindow(b, x, 100, 9, 16, phHash(i, 109)%4 == 0, gdlStoneL)
		b.rect(x-3, 97, 15, 2, gdlStoneD)
		alcBalcony(b, x-2, 116, 13)
	}
	// the portal: columns, the carved frieze, the crest held over it and the round-arched door
	b.rect(mid-24, 62, 48, 84, gdlStoneL)
	for _, x := range []int{mid - 24, mid + 20} {
		b.rect(x, 62, 4, 84, shade(gdlStone, 1.1))
		b.rect(x+3, 62, 1, 84, gdlStoneD)
	}
	b.rect(mid-20, 92, 40, 6, gdlStoneD) // the frieze
	for x := mid - 18; x < mid+18; x += 4 {
		b.rect(x, 93, 2, 4, gdlStoneL)
	}
	b.ellipse(float64(mid), 76, 10, 11, gdlStoneD) // the crest
	b.ellipse(float64(mid), 76, 8, 9, rgb(0xb83a3a))
	b.rect(mid-1, 69, 2, 14, rgb(0xf0c040))
	b.rect(mid-6, 75, 12, 2, rgb(0xf0c040))
	b.rect(mid-12, 104, 24, 42, alcWood)
	b.ellipse(float64(mid), 104, 12, 8, alcWood)
	b.rect(mid, 100, 1, 46, shade(alcWood, 0.7))
	for i := 0; i < 3; i++ { // the steps
		b.rect(mid-28-i*6, 140+i*2, 56+i*12, 2, shade(gdlStoneL, 1-float64(i)*0.06))
	}
}

// the stone church with its two little towers: red pyramid roofs, bells, the red gable, rose window and portal
func gdlChurch(b *pixBuf) {
	x0, x1 := 556, 702
	mid := (x0 + x1) / 2
	stone := rgb(0xc8a47a)
	// the nave: a red tiled gable over the front
	alcWall(b, x0+26, 58, x1-26, 146, stone, 113)
	for y := 38; y < 60; y++ {
		hw := (y - 38) * 50 / 22
		c := mixc(alcRoof, alcRoofD, float64(y-38)/22)
		b.rect(mid-hw, y, 2*hw, 1, c)
		if y%3 == 0 {
			b.rect(mid-hw, y, 2*hw, 1, shade(c, 0.85))
		}
	}
	b.rect(mid-1, 28, 2, 10, gdlStoneD) // the cross
	b.rect(mid-4, 31, 8, 2, gdlStoneD)
	// the two towers
	for _, tx := range []int{x0, x1 - 28} {
		alcWall(b, tx, 26, tx+28, 146, shade(stone, 1.05), 117+tx)
		for y := 10; y < 26; y++ { // the red pyramid roof
			hw := (y - 10) * 16 / 16
			b.rect(tx+14-hw, y, 2*hw, 1, mixc(alcRoof, alcRoofD, float64(y-10)/16))
		}
		b.rect(tx-1, 26, 30, 2, gdlStoneL)
		b.rect(tx+8, 32, 12, 16, alcDark) // the bell opening
		b.ellipse(float64(tx+14), 32, 6, 4, alcDark)
		b.rect(tx+11, 38, 6, 6, rgb(0xb08a40))
		b.rect(tx-1, 50, 30, 2, gdlStoneL)
		alcWindow(b, tx+11, 70, 6, 14, false, gdlStoneL)
		alcWindow(b, tx+11, 104, 6, 14, tx == x0, gdlStoneL)
	}
	// the rose window and the portal with its archivolts
	b.ellipse(float64(mid), 76, 11, 11, gdlStoneD)
	b.ellipse(float64(mid), 76, 9, 9, rgb(0x5a3a5a))
	for a := 0.0; a < 2*math.Pi; a += math.Pi / 4 {
		for r := 2.0; r < 9; r++ {
			b.set(mid+int(math.Round(math.Cos(a)*r)), 76+int(math.Round(math.Sin(a)*r)), gdlStoneL)
		}
	}
	b.ellipse(float64(mid), 76, 2, 2, alcLit)
	for i := 2; i >= 0; i-- {
		w := 12 + i*3
		b.rect(mid-w, 112-i*2, 2*w, 34+i*2, shade(stone, 0.85+float64(i)*0.07))
		b.ellipse(float64(mid), float64(112-i*2), float64(w), float64(8+i), shade(stone, 0.85+float64(i)*0.07))
	}
	b.rect(mid-9, 112, 18, 34, alcWood)
	b.ellipse(float64(mid), 112, 9, 6, alcWood)
	b.rect(mid, 108, 1, 38, shade(alcWood, 0.7))
}

// the pantheon: pale stone, a grey dome on its drum with a lantern and a cross, corner turrets, the rose window
func gdlPantheon(b *pixBuf) {
	x0, x1 := 712, 822
	mid := (x0 + x1) / 2
	grey, greyD, greyL := rgb(0x7a7a8e), rgb(0x52526a), rgb(0xa4a4b8)
	// the dome (ribbed) on its drum
	b.rect(mid-30, 50, 60, 14, gdlPale)
	for x := mid - 28; x < mid+28; x += 8 {
		b.rect(x, 53, 3, 8, alcDark)
	}
	for y := 18; y < 50; y++ {
		t := float64(50-y) / 32
		hw := int(32 * math.Sqrt(math.Max(0, 1-t*t)))
		b.rect(mid-hw, y, 2*hw, 1, grey)
		b.rect(mid-hw, y, hw/2, 1, greyL)
		b.rect(mid+hw-hw/3, y, hw/3, 1, greyD)
	}
	for _, o := range []int{-18, -6, 6, 18} { // ribs
		for y := 22; y < 50; y++ {
			t := float64(50-y) / 32
			hw := 32 * math.Sqrt(math.Max(0, 1-t*t))
			b.set(mid+int(float64(o)*hw/32), y, greyD)
		}
	}
	b.rect(mid-4, 10, 8, 9, gdlPale) // the lantern and the cross
	b.ellipse(float64(mid), 10, 4, 3, grey)
	b.rect(mid, 1, 1, 7, gdlStoneD)
	b.rect(mid-2, 3, 5, 1, gdlStoneD)
	// the body with its corner turrets
	alcWall(b, x0+8, 62, x1-8, 146, gdlPale, 121)
	for _, tx := range []int{x0, x1 - 16} {
		alcWall(b, tx, 48, tx+16, 146, shade(gdlPale, 0.96), 123+tx)
		b.ellipse(float64(tx+8), 48, 8, 7, grey)
		b.ellipse(float64(tx+6), 46, 3, 3, greyL)
		b.rect(tx+7, 36, 2, 6, gdlStoneD)
		alcWindow(b, tx+5, 64, 6, 14, false, gdlStoneL)
		alcWindow(b, tx+5, 98, 6, 14, false, gdlStoneL)
	}
	b.rect(x0+8, 62, x1-x0-16, 2, gdlStoneL)
	// the rose window, the portal and the steps
	b.ellipse(float64(mid), 82, 12, 12, gdlStoneD)
	b.ellipse(float64(mid), 82, 10, 10, rgb(0x4a4a6a))
	for a := 0.0; a < 2*math.Pi; a += math.Pi / 6 {
		for r := 3.0; r < 10; r++ {
			b.set(mid+int(math.Round(math.Cos(a)*r)), 82+int(math.Round(math.Sin(a)*r)), gdlPale)
		}
	}
	b.ellipse(float64(mid), 82, 2.5, 2.5, alcLit)
	b.rect(mid-14, 102, 28, 44, shade(gdlPale, 0.92))
	b.rect(mid-9, 112, 18, 34, alcDark)
	b.ellipse(float64(mid), 112, 9, 7, alcDark)
	for i := 0; i < 3; i++ {
		b.rect(mid-24-i*5, 140+i*2, 48+i*10, 2, shade(gdlStoneL, 1-float64(i)*0.06))
	}
}

// the purple stadium up on its terrace (stands, pillars, the purple crest, two floodlights) and the diagonal steps
func gdlStadium(b *pixBuf) {
	x0, x1 := 826, 960
	// the floodlights: masts with a panel of lamps (lit)
	for _, mx := range []int{846, 948} {
		b.rect(mx, 10, 2, 50, rgb(0x3a3448))
		b.rect(mx-7, 2, 16, 9, rgb(0x2a2438))
		for gx := 0; gx < 4; gx++ {
			for gy := 0; gy < 2; gy++ {
				b.rect(mx-6+gx*4, 3+gy*4, 3, 3, rgb(0xfff2c0))
			}
		}
	}
	// the stadium: the curved rim of the stands, then the facade with white pillars and purple panels
	for x := x0; x < x1; x++ {
		dx := float64(x-893) / 70
		top := 34 + int(6*dx*dx)
		b.rect(x, top, 1, 5, gdlPurpleL)
		b.rect(x, top+5, 1, 50-top, gdlPurple)
		if x%3 == 0 {
			b.set(x, top+6, rgb(0xd8d0e8)) // the crowd's heads over the rim
		}
	}
	alcWall(b, x0, 50, x1, 94, gdlPurpleD, 131)
	for x := x0 + 2; x < x1; x += 14 {
		b.rect(x, 50, 4, 44, rgb(0xe6e0f0))
		b.rect(x+3, 50, 1, 44, rgb(0xb0a8c8))
	}
	for x := x0 + 7; x < x1-4; x += 14 {
		b.rect(x, 78, 8, 12, alcDark) // the gates
		b.rect(x, 56, 8, 8, gdlPurple)
	}
	// the crest: a purple shield with a white border and a golden band
	cx := 896
	b.rect(cx-10, 52, 20, 18, rgb(0xf4f0e8))
	b.rect(cx-8, 54, 16, 14, gdlPurple)
	for y := 70; y < 78; y++ {
		hw := 10 - (y-70)*10/8
		b.rect(cx-hw, y, 2*hw, 1, rgb(0xf4f0e8))
		if hw > 2 {
			b.rect(cx-hw+2, y, 2*hw-4, 1, gdlPurple)
		}
	}
	b.rect(cx-8, 60, 16, 3, rgb(0xf0c040))
	b.rect(cx-1, 55, 2, 12, rgb(0xf4f0e8))
	// the terrace: a stone wall up to the stadium, bushes, and the diagonal steps with their balustrade
	alcWall(b, x0, 94, x1, 146, gdlStoneD, 137)
	b.rect(x0, 94, x1-x0, 2, gdlStoneL)
	for i := 0; i < 24; i++ { // the steps: from the square (bottom left) up to the terrace (top right)
		sx := x0 + 4 + i*5
		sy := 142 - i*2
		b.rect(sx, sy, 20, 2, gdlStoneL)
		b.rect(sx, sy+2, 20, 1, gdlStoneD)
	}
	for i := 0; i < 24; i++ { // the balustrade along the steps
		sx := x0 + 2 + i*5
		sy := 136 - i*2
		b.rect(sx, sy, 5, 1, gdlPale)
		if i%2 == 0 {
			b.rect(sx, sy, 1, 6, gdlPale)
		}
	}
	for _, p := range [][2]int{{836, 104}, {852, 110}, {944, 128}, {930, 138}} {
		b.ellipse(float64(p[0]), float64(p[1]), 6, 5, alcTreeDark)
		b.ellipse(float64(p[0]-1), float64(p[1]-1), 4, 3, shade(gdlHill, 1.1))
	}
}

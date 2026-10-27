package main

import "math"

// Props for the new scenarios: plazas, funfair, supermarket, stadium, cinema, pub.

func init() {
	pk := propKinds

	pk["fountain"] = &PropKind{W: 100, H: 66, FW: 90, FH: 38, Draw: func(b *pixBuf, v int) {
		stone, stoneD, stoneL := c(0xb8b4aa), c(0x7a766e), c(0xdcd8cc)
		b.ellipse(50, 54, 49, 12, stoneD)
		b.ellipse(50, 51, 49, 12, stone)
		b.ellipse(50, 50, 45, 10, stoneL)
		b.ellipse(50, 51, 42, 9, c(0x4a90c8))
		b.ellipse(50, 51, 36, 7, c(0x68b0e0))
		for i := 0; i < 14; i++ {
			a := float64(i) * 0.9
			b.set(50+int(math.Cos(a)*30), 51+int(math.Sin(a)*5), c(0xc8ecff))
		}
		b.rect(46, 22, 8, 26, stone)
		b.rect(46, 22, 3, 26, stoneL)
		b.rect(52, 22, 2, 26, stoneD)
		b.ellipse(50, 30, 18, 6, stone)
		b.ellipse(50, 29, 15, 4.4, c(0x68b0e0))
		b.rect(49, 8, 2, 16, c(0xc8ecff))
		for _, o := range []int{-8, -5, 5, 8} {
			b.line(50, 10, 50+o, 26, c(0x9ad0f0))
		}
		b.ellipse(50, 9, 3, 2, stoneL)
	}}
	pk["eiffel"] = &PropKind{W: 130, H: 200, FW: 84, FH: 14, Draw: func(b *pixBuf, v int) {
		iron, ironD, ironL := c(0x6a4a38), c(0x3e2a20), c(0x8a6a52)
		cx := 65.0
		half := func(y float64) float64 { // half width of the tower at row y (top 8 .. bottom 198)
			t := (y - 8) / 190
			return 2 + 56*math.Pow(t, 1.9)
		}
		for y := 8; y < 198; y++ {
			h := half(float64(y))
			l, r := int(cx-h), int(cx+h)
			b.rect(l, y, 3, 1, iron)
			b.rect(r-2, y, 3, 1, ironD)
			if y%6 == 0 && h > 4 {
				b.rect(l+2, y, int(2*h)-3, 1, ironL)
			}
		}
		// inner lattice (zigzag)
		for y := 20; y < 188; y += 12 {
			h1, h2 := half(float64(y)), half(float64(y+12))
			b.line(int(cx-h1)+2, y, int(cx+h2)-2, y+12, ironD)
			b.line(int(cx+h1)-2, y, int(cx-h2)+2, y+12, ironD)
		}
		// platforms
		for _, py := range []int{70, 124} {
			h := half(float64(py))
			b.rect(int(cx-h)-3, py, int(2*h)+7, 4, ironL)
			b.rect(int(cx-h)-3, py+3, int(2*h)+7, 1, ironD)
		}
		// base arch: hollow centre between the legs
		for y := 150; y < 198; y++ {
			h := half(float64(y))
			a := math.Sqrt(math.Max(0, 1-math.Pow((float64(y)-198)/48, 2)))
			w := int((h - 8) * a)
			for x := int(cx) - w; x <= int(cx)+w; x++ {
				b.set(x, y, c(0x000000))
				b.p[(y*b.w+x)*4+3] = 0
			}
		}
		b.rect(int(cx)-1, 0, 2, 10, ironD)
		b.set(int(cx), 0, c(0xff4040))
		b.rect(int(cx-half(198))-2, 196, int(2*half(198))+5, 3, ironD)
	}}
	pk["carousel"] = &PropKind{W: 92, H: 84, FW: 72, FH: 26, Draw: func(b *pixBuf, v int) {
		b.ellipse(46, 72, 44, 11, c(0x6a2a34))
		b.ellipse(46, 70, 44, 11, c(0xd8b050))
		b.ellipse(46, 69, 38, 8.5, c(0x8a3a46))
		cols := []uint32{0xe04a5a, 0xf0e8d0}
		for i := 0; i < 8; i++ {
			a := float64(i)/8*2*math.Pi + 0.4
			px := 46 + int(math.Cos(a)*30)
			py := 66 + int(math.Sin(a)*6)
			b.rect(px, py-30, 1, 30, c(0xf0d070))
			if math.Sin(a) > -0.3 {
				hx := px - 3
				hy := py - 22 + int(math.Sin(float64(i)*1.3)*3)
				b.rect(hx, hy, 7, 5, c([]uint32{0xf4f4f0, 0xd08040, 0x6a4a30, 0xe8a0c0}[i%4]))
				b.rect(hx+5, hy-3, 3, 4, c([]uint32{0xf4f4f0, 0xd08040, 0x6a4a30, 0xe8a0c0}[i%4]))
				b.rect(hx, hy+5, 1, 3, c(0x3a2a20))
				b.rect(hx+5, hy+5, 1, 3, c(0x3a2a20))
			}
		}
		b.rect(44, 14, 4, 52, c(0xd8b050))
		for y := 8; y < 34; y++ {
			t := float64(y-8) / 26
			w := int(5 + 40*t)
			for x := 46 - w; x < 46+w; x++ {
				k := (x - (46 - w)) * 8 / (2*w + 1)
				b.set(x, y, c(cols[k%2]))
			}
		}
		b.rect(2, 33, 88, 2, c(0xd8b050))
		b.rect(45, 2, 2, 7, c(0xf0d070))
		b.ellipse(46, 3, 2, 2, c(0xff4060))
	}}
	pk["ferris"] = &PropKind{W: 112, H: 132, FW: 46, FH: 10, Draw: func(b *pixBuf, v int) {
		cx, cy, r := 56.0, 62.0, 50.0
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
			a := float64(i) / 10 * 2 * math.Pi
			b.line(int(cx), int(cy), int(cx+math.Cos(a)*r), int(cy+math.Sin(a)*r), c(0xb8b8c8))
			gx, gy := int(cx+math.Cos(a)*r), int(cy+math.Sin(a)*r)
			b.rect(gx-3, gy, 7, 6, c([]uint32{0xe04a5a, 0xf0c030, 0x40b0e0, 0x70d060, 0xe880c0}[i%5]))
			b.rect(gx-3, gy, 7, 1, c(0xf8f8f8))
		}
		b.ellipse(cx, cy, 4, 4, c(0xe04a5a))
		b.rect(26, 128, 60, 4, c(0x4a4a58))
	}}
	pk["bumpercar"] = &PropKind{W: 26, H: 20, FW: 22, FH: 8, Vars: 4, Draw: func(b *pixBuf, v int) {
		col := c([]uint32{0xe04a5a, 0x40a0e0, 0xf0c030, 0x60d070}[v%4])
		b.rect(2, 8, 22, 9, shade(col, 0.7))
		b.rect(2, 7, 22, 8, col)
		b.rect(2, 7, 22, 1, shade(col, 1.3))
		b.rect(6, 3, 12, 6, shade(col, 0.85))
		b.rect(8, 4, 8, 3, c(0xa8d8f0))
		b.rect(0, 14, 26, 3, c(0x2a2a30))
		b.rect(11, 0, 1, 4, c(0x3a3a46))
		b.set(11, 0, c(0xf0f040))
	}}
	pk["coaster"] = &PropKind{W: 120, H: 84, FW: 14, FH: 8, Draw: func(b *pixBuf, v int) {
		red, white := c(0xd83a3a), c(0xf0f0f0)
		for x := 0; x < 120; x++ {
			y := 30 + int(math.Sin(float64(x)/120*math.Pi*2)*22)
			b.rect(x, y, 1, 3, red)
			b.set(x, y-1, white)
			if x%10 == 0 {
				b.line(x, y+3, x, 80, c(0x8a8a98))
				b.line(x, y+12, x+5, y+3, c(0xb0b0c0))
			}
		}
		b.rect(0, 80, 120, 3, c(0x4a4a58))
		// a little train
		b.rect(52, 5, 16, 6, c(0x40a0e0))
		b.rect(52, 5, 16, 1, white)
	}}
	pk["tent"] = &PropKind{W: 66, H: 54, FW: 60, FH: 18, Vars: 4, Draw: func(b *pixBuf, v int) {
		pal := [][2]uint32{{0xe04a5a, 0xf4f0e0}, {0x40a0e0, 0xf4f0e0}, {0xf0c030, 0xf4f0e0}, {0x60c070, 0xf4f0e0}}[v%4]
		for y := 4; y < 34; y++ {
			t := float64(y-4) / 30
			w := int(4 + 29*t)
			for x := 33 - w; x < 33+w; x++ {
				k := (x - (33 - w)) * 6 / (2*w + 1)
				b.set(x, y, c(pal[k%2]))
			}
		}
		b.rect(5, 34, 56, 18, c(pal[0]))
		b.rect(5, 34, 56, 2, shade(c(pal[0]), 0.7))
		b.rect(26, 38, 14, 14, c(0x24202c))
		b.rect(32, 38, 2, 14, c(0x14121a))
		b.rect(32, 0, 2, 6, c(0x6a4a30))
		b.rect(34, 0, 7, 4, c(pal[0]))
	}}
	pk["foodtruck"] = &PropKind{W: 84, H: 46, FW: 80, FH: 20, Vars: 4, Draw: func(b *pixBuf, v int) {
		col := c([]uint32{0xe04a5a, 0x40b0a0, 0xf0b030, 0x6a7ae0}[v%4])
		b.cube(0, 6, 84, 40, 8, shade(col, 1.15), col)
		b.rect(4, 22, 44, 12, c(0x2a2a34))
		b.rect(4, 20, 44, 2, c(0xf4f0e0))
		for x := 4; x < 48; x += 8 {
			b.rect(x, 17, 4, 4, c(0xf4f0e0))
			b.rect(x+4, 17, 4, 4, shade(col, 0.8))
		}
		b.rect(8, 26, 6, 3, c(0xf0d070))
		b.rect(54, 14, 22, 14, c(0xa8d8f0))
		b.rect(54, 14, 22, 1, c(0xf4f4f8))
		b.ellipse(16, 44, 6, 5, c(0x1a1a20))
		b.ellipse(66, 44, 6, 5, c(0x1a1a20))
		b.ellipse(16, 44, 2.4, 2.4, c(0x9a9aa8))
		b.ellipse(66, 44, 2.4, 2.4, c(0x9a9aa8))
		b.rect(10, 36, 30, 2, c(0xf4f0e0))
	}}
	pk["shelf"] = &PropKind{W: 92, H: 42, FW: 90, FH: 10, Vars: 4, Draw: func(b *pixBuf, v int) {
		b.rect(0, 6, 92, 36, c(0x8a8a96))
		b.rect(0, 6, 92, 2, c(0xc8c8d4))
		rows := [][]uint32{
			{0xe04a5a, 0xf0d030, 0x40a0e0, 0x60c070, 0xe880c0},
			{0xf0a030, 0x6a7ae0, 0xe04a5a, 0xf4f0e0, 0x40b0a0},
			{0x60c070, 0xe04a5a, 0xf0d030, 0x6a7ae0, 0xf0a030},
		}
		for r := 0; r < 3; r++ {
			y := 9 + r*10
			b.rect(0, y+8, 92, 2, c(0x6a6a76))
			for x := 2; x < 90; x += 5 {
				h := 5 + (x*7+r*3+v)%4
				b.rect(x, y+8-h, 4, h, c(rows[(r+v)%3][(x/5)%5]))
				b.rect(x, y+8-h, 4, 1, c(0xf8f8f8))
			}
		}
		b.rect(0, 38, 92, 4, c(0x5a5a66))
	}}
	pk["freezer"] = &PropKind{W: 76, H: 40, FW: 74, FH: 12, Draw: func(b *pixBuf, v int) {
		b.cube(0, 4, 76, 36, 6, c(0xe4eef4), c(0xb8ccd8))
		b.rect(3, 12, 70, 22, c(0x9ad0e8))
		b.rect(3, 12, 70, 2, c(0xf4fcff))
		for x := 6; x < 70; x += 8 {
			b.rect(x, 18, 5, 6, c([]uint32{0xe04a5a, 0xf0d030, 0x40a0e0, 0x60c070}[(x/8)%4]))
			b.rect(x, 26, 5, 5, c([]uint32{0x6a7ae0, 0xf0a030, 0xe880c0, 0xf4f0e0}[(x/8)%4]))
		}
		b.rect(0, 36, 76, 2, c(0x40e0ff))
	}}
	pk["checkout"] = &PropKind{W: 48, H: 32, FW: 44, FH: 12, UseKind: "drink", Use: []Vec{{0, 12}}, Draw: func(b *pixBuf, v int) {
		b.cube(2, 10, 44, 22, 6, c(0xd8d8e0), c(0x6a7ae0))
		b.rect(6, 12, 26, 3, c(0x2a2a34))
		b.rect(34, 2, 10, 10, c(0x3a3a46))
		b.rect(36, 4, 6, 3, c(0x70e0a0))
		b.rect(36, 9, 6, 1, c(0xf0f0f0))
		b.rect(12, 6, 8, 5, c(0xe04a5a))
	}}
	pk["cart"] = &PropKind{W: 22, H: 22, FW: 18, FH: 6, Draw: func(b *pixBuf, v int) {
		for x := 2; x < 18; x += 3 {
			b.rect(x, 5, 1, 8, c(0xc8c8d4))
		}
		b.rect(2, 5, 17, 1, c(0xe4e4ee))
		b.rect(2, 12, 15, 1, c(0xc8c8d4))
		b.line(17, 5, 21, 1, c(0xe4e4ee))
		b.ellipse(5, 18, 2, 2, c(0x2a2a34))
		b.ellipse(15, 18, 2, 2, c(0x2a2a34))
		b.rect(4, 13, 12, 4, c(0xa0a0b0))
	}}
	pk["screen"] = &PropKind{W: 170, H: 70, FW: 170, FH: 8, Vars: 3, Draw: func(b *pixBuf, v int) {
		b.rect(0, 0, 170, 70, c(0x6a1a28))
		b.rect(0, 0, 170, 4, c(0x8a2a3a))
		for x := 0; x < 14; x++ {
			b.rect(x*2, 4, 1, 64, c(0x7a2232))
			b.rect(156+x*2, 4, 1, 64, c(0x7a2232))
		}
		scr := [][2]uint32{{0x7ab4e8, 0xf0e0b0}, {0xe88a5a, 0x3a2a5a}, {0x3a8a5a, 0xe8f0a0}}[v%3]
		for y := 8; y < 62; y++ {
			t := float64(y-8) / 54
			b.rect(24, y, 122, 1, mixc(c(scr[0]), c(scr[1]), t))
		}
		b.ellipse(100, 30, 12, 8, c(0xf8f4e0))
		b.rect(50, 44, 70, 18, shade(c(scr[1]), 0.55))
		b.rect(24, 60, 122, 2, c(0xf4f4f8))
		b.rect(0, 64, 170, 6, c(0x3a1018))
	}}
	pk["popcorn"] = &PropKind{W: 28, H: 38, FW: 24, FH: 8, Draw: func(b *pixBuf, v int) {
		b.cube(2, 14, 24, 24, 5, c(0xe84a4a), c(0xb83030))
		b.rect(4, 4, 20, 12, c(0xa8d8f0))
		b.rect(4, 4, 20, 1, c(0xf4f8ff))
		for i := 0; i < 14; i++ {
			b.rect(5+(i*5)%16, 8+(i*3)%7, 2, 2, c(0xf8f0c0))
		}
		b.rect(4, 22, 20, 3, c(0xf8f0d0))
		b.rect(6, 28, 3, 3, c(0xf8f0d0))
		b.rect(11, 28, 3, 3, c(0xe84a4a))
	}}
	pk["goal"] = &PropKind{W: 76, H: 36, FW: 76, FH: 5, Draw: func(b *pixBuf, v int) {
		b.rect(2, 4, 3, 30, c(0xf4f4f8))
		b.rect(71, 4, 3, 30, c(0xf4f4f8))
		b.rect(2, 4, 72, 3, c(0xf4f4f8))
		for x := 5; x < 72; x += 4 {
			b.rect(x, 7, 1, 24, c(0xc8d0d8))
		}
		for y := 9; y < 31; y += 4 {
			b.rect(5, y, 66, 1, c(0xc8d0d8))
		}
		b.rect(2, 32, 72, 2, c(0xd8d8e0))
	}}
	pk["bench"] = &PropKind{W: 38, H: 22, FW: 36, FH: 8, UseKind: "lounge", Use: []Vec{{-9, 12}, {9, 12}}, Draw: func(b *pixBuf, v int) {
		b.rect(2, 4, 34, 3, c(0x8a5a30))
		b.rect(2, 4, 34, 1, c(0xb27a48))
		b.rect(2, 9, 34, 4, c(0x7a4e28))
		b.rect(2, 9, 34, 1, c(0xa06a3a))
		b.rect(4, 13, 3, 7, c(0x2a2a34))
		b.rect(31, 13, 3, 7, c(0x2a2a34))
		b.rect(4, 5, 2, 9, c(0x2a2a34))
		b.rect(32, 5, 2, 9, c(0x2a2a34))
	}}
	pk["cafetable"] = &PropKind{W: 40, H: 44, FW: 14, FH: 8, Vars: 3, Draw: func(b *pixBuf, v int) {
		col := c([]uint32{0xe04a5a, 0x40a0e0, 0xf0c030}[v%3])
		b.rect(19, 12, 2, 30, c(0x6a6a76))
		b.ellipse(20, 36, 7, 2.6, c(0x4a4a56))
		b.ellipse(20, 30, 13, 4, c(0xe8e0d0))
		b.ellipse(20, 29, 11, 3, c(0xf8f4ec))
		b.rect(16, 26, 2, 3, c(0xe04040))
		for y := 2; y < 14; y++ {
			t := float64(y-2) / 12
			w := int(3 + 17*t)
			for x := 20 - w; x < 20+w; x++ {
				k := (x - (20 - w)) * 6 / (2*w + 1)
				if k%2 == 0 {
					b.set(x, y, col)
				} else {
					b.set(x, y, c(0xf4f0e0))
				}
			}
		}
		b.rect(19, 0, 2, 3, c(0x6a6a76))
	}}
	pk["statue"] = &PropKind{W: 28, H: 62, FW: 22, FH: 10, Vars: 2, Draw: func(b *pixBuf, v int) {
		st, stD, stL := c(0xb8b4aa), c(0x7a766e), c(0xdcd8cc)
		b.cube(2, 38, 24, 24, 5, stL, st)
		b.rect(2, 56, 24, 2, stD)
		b.rect(10, 20, 8, 20, st)
		b.rect(10, 20, 3, 20, stL)
		b.ellipse(14, 14, 4.4, 5, st)
		b.set(13, 13, stD)
		b.set(16, 13, stD)
		if v == 0 {
			b.line(18, 22, 24, 10, st)
			b.line(10, 24, 6, 32, st)
		} else {
			b.line(10, 22, 4, 12, st)
			b.line(18, 24, 22, 32, st)
			b.rect(3, 6, 3, 7, stD)
		}
	}}
	pk["jukebox"] = &PropKind{W: 26, H: 40, FW: 24, FH: 8, Light: true, Draw: func(b *pixBuf, v int) {
		b.cube(1, 8, 24, 32, 5, c(0xe0a030), c(0x8a4a1a))
		b.ellipse(13, 12, 11, 8, c(0xf0c850))
		b.ellipse(13, 13, 8, 6, c(0x40e0e0))
		b.rect(5, 22, 16, 8, c(0x2a1a14))
		for i := 0; i < 4; i++ {
			b.rect(6+i*4, 24, 3, 4, c([]uint32{0xe04a5a, 0xf0e040, 0x40e0ff, 0xe04a5a}[i]))
		}
		b.rect(3, 34, 20, 2, c(0xf0e040))
	}}
	pk["pooltable"] = &PropKind{W: 66, H: 38, FW: 62, FH: 22, Draw: func(b *pixBuf, v int) {
		b.cube(2, 6, 62, 32, 8, c(0x2a8a50), c(0x5a3418))
		b.rect(2, 6, 62, 3, c(0x7a4a28))
		b.rect(2, 6, 3, 24, c(0x7a4a28))
		b.rect(61, 6, 3, 24, c(0x7a4a28))
		for i, col := range []uint32{0xe04a5a, 0xf0d030, 0x40a0e0, 0xf4f4f0, 0x1a1a20, 0xe88a30} {
			b.rect(34+(i%3)*4, 14+(i/3)*4, 3, 3, c(col))
		}
		b.rect(14, 18, 3, 3, c(0xf8f8f8))
		b.line(4, 30, 22, 20, c(0xe8d0a0))
		b.rect(3, 30, 3, 3, c(0x14100c))
	}}
	pk["stands"] = &PropKind{W: 210, H: 50, FW: 206, FH: 14, Vars: 2, Draw: func(b *pixBuf, v int) {
		b.rect(0, 14, 210, 36, c(0x4a4a58))
		for t := 0; t < 4; t++ {
			y := 10 + t*9
			b.rect(0, y, 210, 3, c(0x8a8a98))
			b.rect(0, y+3, 210, 6, c(0x5a5a68))
			for x := 3 + (t*5)%6; x < 206; x += 8 {
				skin := c([]uint32{0xe8b48a, 0xc98a5e, 0x8a5a3a, 0xf7d9c0}[(x/8+t)%4])
				shirt := c([]uint32{0xe04a5a, 0x40a0e0, 0xf0c030, 0x60c070, 0xf4f4f0, 0x8a2aa8}[(x/8*3+t*2+v)%6])
				b.rect(x, y-2, 4, 3, skin)
				b.rect(x-1, y+1, 6, 6, shirt)
			}
		}
		b.rect(0, 46, 210, 4, c(0x3a3a46))
		b.rect(0, 0, 210, 6, c(0x9a2a2a))
		for x := 0; x < 210; x += 14 {
			b.rect(x, 0, 7, 6, c(0xf0e0b0))
		}
	}}
	pk["ticketbooth"] = &PropKind{W: 36, H: 46, FW: 32, FH: 12, Draw: func(b *pixBuf, v int) {
		b.cube(2, 10, 32, 36, 6, c(0xe04a5a), c(0xb83040))
		b.rect(6, 18, 24, 10, c(0xa8d8f0))
		b.rect(6, 18, 24, 1, c(0xf4f8ff))
		b.rect(2, 8, 32, 4, c(0xf4f0e0))
		for x := 2; x < 34; x += 8 {
			b.rect(x, 8, 4, 4, c(0xe04a5a))
		}
		b.rect(10, 29, 16, 2, c(0xf0d070))
		b.rect(8, 0, 20, 8, c(0xf0d070))
		b.rect(10, 2, 16, 3, c(0x6a1a28))
	}}
	pk["cottoncandy"] = &PropKind{W: 30, H: 42, FW: 22, FH: 8, Draw: func(b *pixBuf, v int) {
		b.cube(3, 20, 24, 22, 5, c(0xf0e8f0), c(0xe880c0))
		b.rect(3, 26, 24, 2, c(0x40b0e0))
		for i, col := range []uint32{0xf4a0d0, 0x90d0f8, 0xf4a0d0} {
			b.ellipse(8+float64(i)*7, 12, 5, 6, c(col))
			b.rect(8+i*7, 16, 1, 6, c(0xf4f0e0))
		}
		b.rect(4, 14, 22, 2, c(0xf4f0e0))
	}}
	pk["balloons"] = &PropKind{W: 34, H: 56, FW: 12, FH: 6, NoSolid: false, Draw: func(b *pixBuf, v int) {
		cols := []uint32{0xe04a5a, 0xf0d030, 0x40a0e0, 0x60c070, 0xe880c0, 0xf0a030}
		for i := 0; i < 8; i++ {
			x := 6 + (i*9)%24
			y := 8 + (i*13)%22
			b.ellipse(float64(x), float64(y), 5, 6.4, c(cols[i%6]))
			b.set(x-2, y-3, c(0xffffff))
			b.line(x, y+6, 17, 50, c(0xd8d8e0))
		}
		b.rect(14, 50, 6, 4, c(0x6a6a76))
	}}
	pk["hedge"] = &PropKind{W: 64, H: 24, FW: 62, FH: 12, Draw: func(b *pixBuf, v int) {
		b.rect(2, 8, 60, 16, c(0x2a6a30))
		for x := 2; x < 62; x += 4 {
			b.rect(x, 6+(x*7)%4, 4, 4, c(0x3a8a40))
		}
		b.rect(2, 8, 60, 2, c(0x4aa050))
		b.rect(2, 20, 60, 4, c(0x1e4e24))
	}}
	pk["planter"] = &PropKind{W: 44, H: 22, FW: 40, FH: 10, Vars: 3, Draw: func(b *pixBuf, v int) {
		b.cube(2, 8, 40, 14, 4, c(0xb8b4aa), c(0x8a867e))
		b.rect(4, 8, 36, 4, c(0x5a3a22))
		cols := []uint32{0xe04a5a, 0xf0c030, 0xe880c0}
		for x := 5; x < 40; x += 4 {
			b.rect(x, 3+(x*5)%4, 3, 5, c(0x2a8a38))
			b.rect(x, 1+(x*5)%4, 3, 3, c(cols[(x/4+v)%3]))
		}
	}}
	pk["obelisk"] = &PropKind{W: 26, H: 96, FW: 18, FH: 10, Draw: func(b *pixBuf, v int) {
		st, stD, stL := c(0xc8c0b0), c(0x8a8478), c(0xe4dccc)
		b.cube(2, 82, 22, 14, 4, stL, st)
		for y := 8; y < 84; y++ {
			t := float64(y-8) / 76
			w := 3 + int(6*t)
			b.rect(13-w, y, w, 1, stL)
			b.rect(13, y, w, 1, stD)
		}
		for y := 0; y < 8; y++ {
			b.rect(13-(y/2), y, y/2*2+1, 1, st)
		}
		b.rect(12, 0, 2, 3, c(0xe8c040))
	}}
	pk["cage"] = &PropKind{W: 34, H: 56, FW: 28, FH: 10, Light: true, Draw: func(b *pixBuf, v int) {
		b.rect(2, 44, 30, 8, c(0x3a3a46))
		b.rect(2, 44, 30, 1, c(0x8a8a98))
		for x := 4; x < 32; x += 6 {
			b.rect(x, 8, 1, 38, c(0xc8c8d8))
		}
		b.rect(2, 8, 30, 2, c(0xe8e8f4))
		b.rect(2, 30, 30, 1, c(0xa8a8b8))
		b.rect(8, 2, 18, 6, c(0xf040c0))
		b.rect(10, 3, 14, 3, c(0x40f0e0))
	}}
	pk["truss"] = &PropKind{W: 120, H: 30, FW: 4, FH: 4, NoSolid: true, Draw: func(b *pixBuf, v int) {
		b.rect(0, 10, 120, 3, c(0x8a8a98))
		b.rect(0, 16, 120, 2, c(0x6a6a78))
		for x := 0; x < 120; x += 8 {
			b.line(x, 11, x+4, 17, c(0xb0b0c0))
			b.line(x+4, 17, x+8, 11, c(0xb0b0c0))
		}
		for i, col := range []uint32{0xff4060, 0x40e0ff, 0xf0e040, 0x60f080, 0xc060f0} {
			x := 12 + i*24
			b.rect(x, 18, 5, 5, c(0x24242c))
			b.rect(x+1, 22, 3, 3, c(col))
		}
	}}
	pk["posters"] = &PropKind{W: 60, H: 34, FW: 56, FH: 4, NoSolid: true, Vars: 3, Draw: func(b *pixBuf, v int) {
		for i := 0; i < 3; i++ {
			x := 3 + i*19
			cols := [][2]uint32{{0xe04a5a, 0xf0d070}, {0x40a0e0, 0xf4f0e0}, {0x60c070, 0xe880c0}}[(i+v)%3]
			b.rect(x, 2, 15, 26, c(0x14121a))
			b.rect(x+1, 3, 13, 24, c(cols[0]))
			b.rect(x+3, 8, 9, 9, c(cols[1]))
			b.rect(x+2, 21, 11, 2, c(0xf4f4f8))
		}
	}}
}

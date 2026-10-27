package main

import (
	"math"
)

func init() {
	pk := propKinds

	pk["lift"] = &PropKind{W: 64, H: 66, FW: 62, FH: 10, Draw: func(b *pixBuf, v int) {
		b.cube(0, 0, 64, 66, 6, c(0x4a4e58), c(0x2c3038))
		b.rect(10, 18, 44, 48, c(0x8a929c))
		b.rect(10, 18, 44, 2, c(0xb4bcc6))
		b.rect(31, 18, 2, 48, c(0x4a4e58))
		b.rect(10, 18, 2, 48, c(0x5a626c))
		b.rect(18, 7, 28, 8, c(0x101810))
		b.rect(21, 9, 4, 4, c(0x40e060))
		b.rect(28, 9, 4, 4, c(0x40e060))
		b.rect(35, 9, 4, 4, c(0x2a6a38))
		b.rect(10, 62, 44, 4, c(0x6a727c))
	}}
	pk["ac"] = &PropKind{W: 46, H: 38, FW: 44, FH: 14, Draw: func(b *pixBuf, v int) {
		b.cube(0, 4, 46, 34, 10, c(0xc4cad2), c(0x8a929c))
		b.ellipse(23, 12, 12, 6, c(0x3a4048))
		b.ellipse(23, 12, 10, 4.6, c(0x14181c))
		for i := 0; i < 4; i++ {
			a := float64(i) * math.Pi / 4
			b.line(int(23-math.Cos(a)*9), int(12-math.Sin(a)*3.5), int(23+math.Cos(a)*9), int(12+math.Sin(a)*3.5), c(0x5a626c))
		}
		b.rect(4, 26, 38, 1, c(0x6a727c))
		b.rect(4, 30, 38, 1, c(0x6a727c))
	}}
	pk["bungalow"] = &PropKind{W: 178, H: 84, FW: 168, FH: 16, UseKind: "drink", Use: []Vec{{-56, 18}, {-18, 18}, {18, 18}, {56, 18}}, Draw: func(b *pixBuf, v int) {
		for x := 0; x < 178; x++ {
			h := 30 + (x*7)%5
			b.rect(x, 0, 1, h, c(0x8a5a2a))
			if x%3 == 0 {
				b.rect(x, 2, 1, h-2, c(0x6a4220))
			}
			if x%5 == 1 {
				b.rect(x, 4, 1, 8, c(0xa87238))
			}
		}
		b.rect(0, 0, 178, 3, c(0x6a4220))
		b.rect(2, 30, 4, 54, c(0x5a3a1a))
		b.rect(172, 30, 4, 54, c(0x5a3a1a))
		b.rect(56, 34, 66, 11, c(0x2a1a10))
		b.rect(58, 36, 62, 7, c(0xe8802a))
		b.rect(62, 38, 54, 3, c(0x2a1a10))
		cols := []uint32{0x3a9a5a, 0xd0a030, 0xc04040, 0x3a7ac0, 0xe0e0e0}
		for i, x := 0, 10; x < 168; x, i = x+9, i+1 {
			b.rect(x, 48, 3, 9, c(cols[i%5]))
		}
		b.cube(6, 58, 166, 26, 10, c(0xb27a48), c(0x5a3520))
		b.rect(6, 80, 166, 2, c(0xc9a640))
	}}
	pk["liberty"] = &PropKind{W: 36, H: 72, FW: 20, FH: 8, Draw: func(b *pixBuf, v int) {
		b.cube(8, 56, 20, 16, 4, c(0xb4b4c0), c(0x8a8a98))
		b.rect(12, 28, 12, 28, c(0x5aa890))
		b.rect(12, 28, 3, 28, c(0x7ac8b0))
		b.ellipse(18, 22, 5.2, 5.4, c(0x5aa890))
		for i := 0; i < 5; i++ {
			a := -2.4 + float64(i)*0.6
			b.line(18, 18, 18+int(math.Cos(a)*8), 18+int(math.Sin(a)*8), c(0x7ac8b0))
		}
		b.line(24, 32, 30, 8, c(0x5aa890))
		b.line(25, 32, 31, 8, c(0x5aa890))
		b.rect(28, 2, 5, 6, c(0xf0c030))
		b.rect(29, 0, 3, 3, c(0xff8a20))
		b.rect(4, 32, 8, 3, c(0x4a8a78))
	}}
	pk["slide"] = &PropKind{W: 190, H: 100, FW: 4, FH: 4, NoSolid: true, SortOff: -60, Draw: func(b *pixBuf, v int) {
		pts := [][2]float64{{186, 12}, {140, 12}, {90, 34}, {44, 62}, {26, 84}, {44, 92}, {64, 78}, {52, 64}, {34, 76}}
		for i := 0; i+1 < len(pts); i++ {
			for w := 0; w < 7; w++ {
				col := c(0xd8a418)
				if w == 0 {
					col = c(0xf8d848)
				}
				if w >= 5 {
					col = c(0xa87c10)
				}
				thickLine(b, pts[i][0], pts[i][1]+float64(w)-3, pts[i+1][0], pts[i+1][1]+float64(w)-3, 3, col)
			}
		}
		b.rect(170, 4, 18, 14, c(0xd8a418))
		b.rect(172, 6, 14, 10, c(0x3a2a08))
		b.rect(182, 18, 3, 30, c(0x8a8a96))
	}}
	pk["stage"] = &PropKind{W: 276, H: 124, FW: 268, FH: 56, SortOff: -90, Draw: func(b *pixBuf, v int) {
		b.rect(0, 0, 276, 8, c(0x2a2a36))
		for x := 4; x < 272; x += 8 {
			b.line(x, 0, x+4, 8, c(0x4a4a5a))
			b.line(x+4, 8, x+8, 0, c(0x4a4a5a))
		}
		b.rect(2, 0, 6, 108, c(0x2a2a36))
		b.rect(268, 0, 6, 108, c(0x2a2a36))
		for _, lx := range []int{26, 70, 138, 206, 250} {
			b.ellipse(float64(lx), 11, 5, 3, c(0x20202a))
			cc := []uint32{0xff3090, 0x30c0ff, 0xa040ff, 0xf0e040, 0x40ff90}[(lx/7)%5]
			b.ellipse(float64(lx), 12, 3, 1.8, c(cc))
		}
		b.cube(0, 62, 276, 62, 14, c(0x504e60), c(0x2a2838))
		b.rect(18, 24, 240, 36, c(0x14141e))
		for x := 20; x < 256; x++ {
			y := 42 + int(8*math.Sin(float64(x)/9))
			b.rect(x, y, 1, 2, c(0xff3090))
			b.rect(x, y+5, 1, 1, c(0x30c0ff))
		}
		// amp stacks and speakers
		for _, sx := range []int{14, 240} {
			b.cube(sx, 40, 22, 42, 5, c(0x3a3a46), c(0x1e1e26))
			b.ellipse(float64(sx)+11, 56, 6, 6, c(0x4a4a58))
			b.ellipse(float64(sx)+11, 56, 4, 4, c(0x101016))
			b.ellipse(float64(sx)+11, 71, 6, 6, c(0x4a4a58))
			b.ellipse(float64(sx)+11, 71, 4, 4, c(0x101016))
		}
		// drum kit
		b.ellipse(138, 70, 14, 5, c(0xd8d8e0))
		b.ellipse(138, 69, 12, 3.6, c(0xa02828))
		b.ellipse(116, 66, 8, 3, c(0xe8c030))
		b.ellipse(160, 66, 8, 3, c(0xe8c030))
		b.ellipse(124, 76, 7, 3, c(0xd8d8e0))
		b.ellipse(152, 76, 7, 3, c(0xd8d8e0))
	}}
	pk["portapotty"] = &PropKind{W: 28, H: 48, FW: 26, FH: 8, Hide: true, Vars: 2, Draw: func(b *pixBuf, v int) {
		col := []uint32{0x3a7ac8, 0x3aa070}[v%2]
		b.cube(0, 0, 28, 48, 6, c(col+0x101010), c(col))
		b.rect(4, 12, 20, 32, shade(c(col), 0.8))
		b.rect(4, 12, 20, 1, shade(c(col), 1.3))
		b.rect(18, 26, 3, 3, c(0xd8d8e0))
		b.rect(8, 16, 12, 5, c(0xe8e8f0))
	}}
	pk["surfboard"] = &PropKind{W: 18, H: 54, FW: 10, FH: 5, Vars: 3, Draw: func(b *pixBuf, v int) {
		cc := []uint32{0xe84a8a, 0x30b0e0, 0xf0c030}[v%3]
		b.ellipse(9, 26, 7, 25, c(cc))
		b.rect(8, 2, 2, 50, c(0xf4f4f4))
		b.ellipse(9, 14, 3, 4, shade(c(cc), 1.3))
		b.rect(6, 36, 6, 2, shade(c(cc), 0.7))
	}}
	pk["lifeguard"] = &PropKind{W: 72, H: 100, FW: 46, FH: 12, Draw: func(b *pixBuf, v int) {
		b.rect(8, 40, 4, 60, c(0x8a5a2a))
		b.rect(58, 40, 4, 60, c(0x8a5a2a))
		for y := 56; y < 96; y += 12 {
			b.line(10, y, 60, y+10, c(0x6a4220))
			b.line(60, y, 10, y+10, c(0x6a4220))
		}
		b.cube(4, 18, 62, 28, 8, c(0xd8d0c0), c(0xb04040))
		b.rect(8, 28, 54, 8, c(0x5a8ac0))
		b.rect(0, 10, 70, 8, c(0xc02828))
		b.rect(0, 10, 70, 2, c(0xe85050))
		b.rect(30, 0, 2, 12, c(0x8a8a96))
		b.rect(32, 0, 8, 5, c(0xe8e830))
		b.rect(28, 84, 14, 16, c(0x8a5a2a))
	}}
	pk["booth"] = &PropKind{W: 168, H: 104, FW: 164, FH: 34, Vars: 2, Draw: func(b *pixBuf, v int) {
		b.cube(0, 0, 168, 104, 14, c(0x2a2040), c(0x1a1228))
		b.rect(0, 0, 168, 2, c(0xb050ff))
		b.rect(10, 22, 148, 62, c(0x14101e))
		for y := 24; y < 82; y++ {
			for x := 12; x < 156; x += 1 {
				if ((x/12)+(y/10)+v)%3 == 0 {
					b.set(x, y, c(0x2a60b0))
				} else if ((x/9)+(y/7))%4 == 0 {
					b.set(x, y, c(0xf0a020))
				} else {
					b.set(x, y, c(0x6a2a90))
				}
			}
		}
		b.rect(10, 22, 148, 2, c(0x40e0ff))
		b.rect(10, 82, 148, 2, c(0xff4090))
		for x := 12; x < 160; x += 14 {
			b.rect(x, 88, 3, 3, c([]uint32{0xff4090, 0x40e0ff, 0xf0e040}[(x/14)%3]))
		}
	}}
	pk["horse"] = &PropKind{W: 56, H: 48, FW: 32, FH: 8, Draw: func(b *pixBuf, v int) {
		b.ellipse(30, 22, 18, 9, c(0x7a3a1a))
		b.ellipse(26, 20, 12, 5, c(0x8a4a24))
		for _, lx := range []int{16, 22, 36, 42} {
			b.rect(lx, 28, 3, 16, c(0x6a2e14))
			b.rect(lx, 42, 3, 3, c(0x14100c))
		}
		b.line(44, 18, 52, 8, c(0x7a3a1a))
		b.line(45, 20, 53, 10, c(0x7a3a1a))
		b.line(46, 22, 54, 12, c(0x7a3a1a))
		b.ellipse(52, 9, 4, 3, c(0x7a3a1a))
		b.rect(54, 10, 2, 2, c(0x2a1a10))
		b.rect(44, 10, 3, 8, c(0x2a1a10))
		b.line(14, 18, 6, 30, c(0x2a1a10))
		b.line(13, 20, 5, 32, c(0x2a1a10))
	}}
	pk["hen"] = &PropKind{W: 14, H: 14, FW: 4, FH: 3, NoSolid: true, Vars: 2, Draw: func(b *pixBuf, v int) {
		b.ellipse(7, 8, 5, 4, c(0xf4f0e8))
		b.rect(1, 5, 3, 3, c(0xe8e0d0))
		b.rect(10, 3+v*2, 3, 3, c(0xf4f0e8))
		b.rect(10, 1+v*2, 2, 2, c(0xe03030))
		b.set(13, 4+v*2, c(0xf0b020))
		b.rect(5, 12, 1, 2, c(0xf0b020))
		b.rect(8, 12, 1, 2, c(0xf0b020))
	}}
	pk["haystack"] = &PropKind{W: 64, H: 48, FW: 54, FH: 14, Draw: func(b *pixBuf, v int) {
		b.ellipse(32, 32, 30, 14, c(0xc8a028))
		b.ellipse(32, 26, 24, 14, c(0xe0bc3c))
		b.ellipse(30, 20, 14, 8, c(0xf0d058))
		for i := 0; i < 26; i++ {
			x, y := 6+(i*13)%52, 14+(i*7)%26
			b.line(x, y, x+4, y-2, c(0xa88418))
		}
	}}
	pk["fruitstand"] = &PropKind{W: 92, H: 50, FW: 88, FH: 14, Draw: func(b *pixBuf, v int) {
		b.cube(0, 14, 92, 36, 10, c(0x8a5a2a), c(0x5a3a1a))
		cols := []uint32{0xd03030, 0xe88a20, 0x40a040, 0xe8d030, 0x8a3a90}
		for i := 0; i < 5; i++ {
			for k := 0; k < 4; k++ {
				b.ellipse(float64(8+i*17+(k%2)*4), float64(12+k*2), 4, 2.4, c(cols[(i+k)%5]))
			}
		}
		b.rect(4, 0, 2, 14, c(0x5a3a1a))
		b.rect(86, 0, 2, 14, c(0x5a3a1a))
		b.rect(2, 0, 88, 3, c(0xe8e0d0))
		b.rect(2, 0, 88, 1, c(0xd03030))
	}}
	pk["arch"] = &PropKind{W: 120, H: 70, FW: 4, FH: 4, NoSolid: true, SortOff: 80, Draw: func(b *pixBuf, v int) {
		b.rect(6, 20, 6, 50, c(0x6a4220))
		b.rect(108, 20, 6, 50, c(0x6a4220))
		b.rect(2, 10, 116, 12, c(0x8a5a2a))
		b.rect(2, 10, 116, 2, c(0xb27a40))
		b.ellipse(60, 12, 16, 11, c(0xf4f0e0))
		b.ellipse(60, 12, 12, 8, c(0xd03030))
		b.ellipse(60, 12, 8, 5, c(0xf4f0e0))
		b.ellipse(60, 12, 4, 2.6, c(0xd03030))
		b.rect(10, 14, 36, 4, c(0xe8d8a0))
		b.rect(74, 14, 36, 4, c(0xe8d8a0))
	}}
	pk["motorbike"] = &PropKind{W: 52, H: 30, FW: 44, FH: 8, Vars: 2, Draw: func(b *pixBuf, v int) {
		cc := []uint32{0x8a3ad0, 0xc82a2a}[v%2]
		b.ellipse(10, 24, 7, 5, c(0x14141a))
		b.ellipse(10, 24, 3, 2.4, c(0x8a8a96))
		b.ellipse(42, 24, 7, 5, c(0x14141a))
		b.ellipse(42, 24, 3, 2.4, c(0x8a8a96))
		b.rect(14, 14, 24, 8, c(cc))
		b.rect(14, 14, 24, 2, shade(c(cc), 1.4))
		b.rect(34, 8, 8, 8, c(cc))
		b.rect(38, 4, 8, 3, c(0x3a3a46))
		b.rect(12, 20, 20, 4, c(0x2a2a34))
		b.rect(2, 22, 6, 2, c(0xe8e8f0))
	}}
	pk["rope"] = &PropKind{W: 124, H: 28, FW: 4, FH: 4, NoSolid: true, SortOff: -4, Draw: func(b *pixBuf, v int) {
		for _, x := range []int{4, 60, 118} {
			b.rect(x, 6, 3, 20, c(0xc0c8d4))
			b.ellipse(float64(x)+1.5, 5, 3, 3, c(0x30e0ff))
		}
		for _, seg := range [][2]int{{4, 60}, {60, 118}} {
			for x := seg[0]; x < seg[1]; x++ {
				t := float64(x-seg[0]) / float64(seg[1]-seg[0])
				y := 8 + int(5*math.Sin(t*math.Pi))
				b.set(x, y, c(0x30c0e0))
				b.set(x, y+1, c(0x1a8aa8))
			}
		}
	}}
	pk["buffet"] = &PropKind{W: 52, H: 114, FW: 46, FH: 96, Poison: true, Vars: 2, UseKind: "drink", Use: []Vec{{-34, -30}, {34, -30}, {-34, -66}, {34, -66}}, Draw: func(b *pixBuf, v int) {
		b.cube(2, 10, 48, 104, 12, c(0xa8c4e0), c(0x5a7aa0))
		b.rect(2, 10, 48, 1, c(0xd0e4f4))
		col := c(0xe04040)
		if v == 1 {
			col = c(0x5ae05a)
		}
		b.ellipse(26, 50, 14, 7, c(0xcfe4f0))
		b.ellipse(26, 49, 11.5, 5, col)
		for i := 0; i < 14; i++ {
			x, y := 8+(i*11)%34, 16+(i*17)%80
			b.rect(x, y, 3, 3, c([]uint32{0xe04040, 0xf0f0f0, 0x40a0e0}[i%3]))
		}
		b.rect(6, 104, 4, 10, c(0x3a4a60))
		b.rect(42, 104, 4, 10, c(0x3a4a60))
	}}
	pk["bathtub"] = &PropKind{W: 62, H: 42, FW: 60, FH: 24, Draw: func(b *pixBuf, v int) {
		b.cube(0, 6, 62, 36, 14, c(0xf0f4f8), c(0xd0d8e0))
		b.rect(4, 10, 54, 8, c(0x7ac0e8))
		b.rect(4, 10, 54, 2, c(0xa8dcf4))
		b.rect(56, 2, 4, 8, c(0xa0a8b4))
		b.rect(54, 1, 6, 2, c(0xa0a8b4))
		b.rect(4, 40, 4, 2, c(0x8a929c))
		b.rect(54, 40, 4, 2, c(0x8a929c))
	}}
	pk["fireplace"] = &PropKind{W: 60, H: 54, FW: 58, FH: 14, Draw: func(b *pixBuf, v int) {
		b.cube(0, 4, 60, 50, 8, c(0x8a8480), c(0x5a5650))
		for y := 14; y < 52; y += 6 {
			b.rect(0, y, 60, 1, c(0x44403c))
		}
		b.rect(12, 20, 36, 28, c(0x1a1210))
		b.rect(26, 34, 8, 12, c(0xff7a20))
		b.rect(22, 38, 16, 8, c(0xffb030))
		b.rect(28, 28, 4, 8, c(0xffd860))
		b.rect(4, 4, 52, 3, c(0x6a4a2a))
	}}
	pk["crate"] = &PropKind{W: 26, H: 26, FW: 24, FH: 8, Draw: func(b *pixBuf, v int) {
		b.cube(0, 2, 26, 24, 8, c(0xb48a50), c(0x8a6232))
		b.rect(0, 12, 26, 2, c(0x6a4a22))
		b.rect(11, 10, 3, 14, c(0x6a4a22))
	}}
	pk["sofa"] = &PropKind{W: 28, H: 62, FW: 26, FH: 56, UseKind: "lounge", Use: []Vec{{-22, -20}, {-22, -40}}, Vars: 2, Draw: func(b *pixBuf, v int) {
		base := []uint32{0xe8e8f0, 0xb89a60}[v%2]
		b.cube(0, 0, 28, 62, 8, shade(c(base), 1.1), c(base))
		b.rect(2, 6, 10, 50, shade(c(base), 0.85))
		b.rect(12, 6, 14, 24, shade(c(base), 1.15))
		b.rect(12, 32, 14, 24, shade(c(base), 1.15))
	}}
	pk["barrel"] = &PropKind{W: 22, H: 28, FW: 18, FH: 8, Draw: func(b *pixBuf, v int) {
		b.rect(2, 6, 18, 20, c(0x7a4a28))
		b.ellipse(11, 7, 9, 3.4, c(0xa06a38))
		b.ellipse(11, 25, 9, 3, c(0x5a3418))
		b.rect(2, 11, 18, 2, c(0x2a2a30))
		b.rect(2, 20, 18, 2, c(0x2a2a30))
	}}
	pk["sign"] = &PropKind{W: 30, H: 40, FW: 6, FH: 4, NoSolid: true, Draw: func(b *pixBuf, v int) {
		b.rect(14, 12, 3, 28, c(0x8a8a96))
		b.rect(2, 2, 26, 14, c(0xd03030))
		b.rect(5, 8, 20, 3, c(0xf4f4f4))
	}}
}

// helicopter body (rotor is drawn dynamically)
func genHeli() *pixBuf {
	b := newPix(112, 64)
	b.ox, b.oy = 1, 1
	b.ellipse(40, 34, 30, 16, c(0x2a4a9a))
	b.ellipse(40, 31, 28, 13, c(0xdcdce8))
	b.ellipse(40, 38, 27, 8, c(0x2a4a9a))
	b.ellipse(26, 29, 12, 9, c(0x7ab0d8))
	b.ellipse(26, 27, 9, 6, c(0xa8d8f0))
	b.rect(52, 28, 40, 8, c(0xdcdce8))
	b.rect(52, 33, 40, 3, c(0x2a4a9a))
	b.rect(88, 18, 5, 18, c(0xdcdce8))
	b.rect(88, 18, 5, 4, c(0x2a4a9a))
	b.rect(34, 26, 24, 8, c(0xf4f4f8))
	b.rect(36, 28, 3, 4, c(0x2a4a9a))
	b.rect(41, 28, 3, 4, c(0x2a4a9a))
	b.rect(46, 28, 3, 4, c(0x2a4a9a))
	b.rect(10, 50, 60, 2, c(0x3a3a46))
	b.rect(18, 44, 2, 6, c(0x3a3a46))
	b.rect(56, 44, 2, 6, c(0x3a3a46))
	b.rect(38, 8, 4, 10, c(0x3a3a46))
	return b
}

func genBodyBag() *pixBuf {
	b := newPix(34, 16)
	b.ox, b.oy = 1, 1
	b.ellipse(16, 8, 15, 6, c(0x20202a))
	b.ellipse(16, 7, 14, 4.6, c(0x34343e))
	b.rect(4, 7, 24, 1, c(0x8a8a96))
	b.rect(4, 5, 24, 1, c(0x4a4a58))
	return b
}

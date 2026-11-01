package main

import "image/color"

// Road visitors that are not plain traffic: a monster truck that drives over
// the other cars, and three cars straight out of Rocket League (Octane,
// Fennec, Dominus) in blue and orange, boosting down the road.

func bigWheel(b *pixBuf, cx, cy, r float64, rim color.RGBA) {
	b.ellipse(cx, cy, r, r, c(0x121216))
	b.ellipse(cx, cy, r*0.62, r*0.62, c(0x2a2a30))
	b.ellipse(cx, cy, r*0.38, r*0.38, rim)
	b.ellipse(cx, cy, r*0.14, r*0.14, c(0x121216))
}

func rlWheel(b *pixBuf, cx, cy float64, rim color.RGBA) {
	b.ellipse(cx, cy, 5.2, 5.2, c(0x121216))
	b.ellipse(cx, cy, 3.2, 3.2, rim)
	b.ellipse(cx, cy, 1.2, 1.2, c(0x121216))
}

func rlTeam(v int) (body, trim color.RGBA) {
	if v%2 == 0 {
		return c(0x2a7ae0), c(0x9ad0ff)
	}
	return c(0xf08020), c(0xffd090)
}

func init() {
	pk := propKinds
	add := func(name string, w, h int, speed float64, vars int, draw func(b *pixBuf, v int)) {
		pk[name] = &PropKind{W: w, H: h, FW: float64(w) - 4, FH: 12, NoSolid: true, Vars: vars, Ambient: speed, Category: "vehicle", Draw: draw}
	}
	add("monstertruck", 100, 64, 80, 3, func(b *pixBuf, v int) {
		body := c([]uint32{0x28a840, 0xd02828, 0x2858d0}[v%3])
		flame := c(0xffb020)
		b.ellipse(50, 61, 46, 3.5, color.RGBA{0, 0, 0, 90})
		// body sits high on the axles
		b.rect(10, 26, 80, 14, body)
		b.rect(10, 26, 80, 2, shade(body, 1.3))
		b.rect(10, 37, 80, 3, shade(body, 0.65))
		b.rect(58, 14, 26, 13, shade(body, 0.92)) // cab
		b.rect(60, 16, 18, 8, c(0x7aa0d0))
		b.rect(62, 17, 3, 6, c(0xc8e0f8))
		b.rect(84, 28, 6, 6, c(0xfff0b0)) // headlight
		b.rect(10, 30, 5, 5, c(0xe03030))
		// flame decals
		for i := 0; i < 5; i++ {
			b.rect(16+i*7, 31, 5-i%2, 2, flame)
			b.rect(18+i*7, 29, 2, 2, c(0xe85020))
		}
		// axles and huge wheels
		b.rect(18, 40, 6, 12, c(0x5a5a64))
		b.rect(74, 40, 6, 12, c(0x5a5a64))
		bigWheel(b, 21, 49, 14, c(0xd8d8e0))
		bigWheel(b, 77, 49, 14, c(0xd8d8e0))
	})
	add("octane", 64, 34, 150, 2, func(b *pixBuf, v int) {
		body, trim := rlTeam(v)
		b.ellipse(32, 31, 28, 3.4, color.RGBA{0, 0, 0, 90})
		b.rect(6, 17, 54, 11, body)
		b.rect(6, 17, 54, 1, shade(body, 1.35))
		b.rect(10, 26, 46, 3, shade(body, 0.6))
		b.rect(16, 9, 26, 9, shade(body, 0.92))  // cabin
		b.rect(20, 11, 18, 6, c(0x182034))       // glass
		b.rect(42, 13, 14, 5, shade(body, 0.95)) // wedge hood
		b.rect(56, 19, 6, 6, shade(body, 0.85))  // nose
		b.rect(2, 11, 8, 2, trim)                // wing
		b.rect(4, 13, 2, 5, shade(body, 0.7))
		b.rect(14, 22, 38, 2, trim)
		b.rect(58, 20, 3, 2, c(0xfff0b0))
		rlWheel(b, 17, 29, trim)
		rlWheel(b, 49, 29, trim)
	})
	add("fennec", 58, 34, 155, 2, func(b *pixBuf, v int) {
		body, trim := rlTeam(v)
		b.ellipse(29, 31, 25, 3.4, color.RGBA{0, 0, 0, 90})
		b.rect(6, 12, 46, 16, body) // boxy
		b.rect(6, 12, 46, 2, shade(body, 1.35))
		b.rect(8, 26, 42, 3, shade(body, 0.6))
		b.rect(12, 14, 32, 7, c(0x182034)) // wide windows
		b.rect(44, 15, 6, 6, shade(body, 0.85))
		b.rect(14, 22, 28, 2, trim)
		b.rect(50, 21, 3, 3, c(0xfff0b0))
		b.rect(3, 10, 6, 2, trim)
		rlWheel(b, 15, 29, trim)
		rlWheel(b, 43, 29, trim)
	})
	add("dominus", 68, 32, 150, 2, func(b *pixBuf, v int) {
		body, trim := rlTeam(v)
		b.ellipse(34, 30, 30, 3, color.RGBA{0, 0, 0, 90})
		b.rect(4, 19, 60, 8, body) // long and low
		b.rect(4, 19, 60, 1, shade(body, 1.35))
		b.rect(8, 25, 52, 3, shade(body, 0.6))
		b.rect(20, 12, 24, 8, shade(body, 0.92)) // cabin set far back
		b.rect(23, 13, 17, 5, c(0x182034))
		b.rect(44, 16, 14, 4, shade(body, 0.95))
		b.rect(58, 20, 6, 4, shade(body, 0.8))
		b.rect(2, 14, 8, 2, trim)
		b.rect(10, 22, 44, 2, trim)
		b.rect(60, 21, 3, 2, c(0xfff0b0))
		rlWheel(b, 16, 28, trim)
		rlWheel(b, 52, 28, trim)
	})
}

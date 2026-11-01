package main

import "image/color"

// Street vehicles beyond the plain car: vans, trucks, an ice-cream truck, the
// school bus, taxis and a few famous cars. All are ambient props (they drive
// along the road lanes, see updateAmbient).

func vehWheels(b *pixBuf, xs ...int) {
	for _, x := range xs {
		b.ellipse(float64(x), 31, 6, 5, c(0x15151b))
		b.ellipse(float64(x), 31, 3, 2.6, c(0x9a9aa6))
	}
}

func vehShadow(b *pixBuf, w int) {
	b.ellipse(float64(w)/2, 32, float64(w)/2-3, 4, color.RGBA{0, 0, 0, 90})
}

func init() {
	pk := propKinds
	add := func(name string, w, h int, speed float64, vars int, draw func(b *pixBuf, v int)) {
		pk[name] = &PropKind{W: w, H: h, FW: float64(w) - 4, FH: 12, NoSolid: true, Vars: vars, Ambient: speed, Category: "vehicle", Draw: draw}
	}
	// plain van, three colours
	add("van", 78, 38, 95, 3, func(b *pixBuf, v int) {
		body := c([]uint32{0xf2f2f6, 0x3a6ac0, 0xc03a3a}[v%3])
		vehShadow(b, 78)
		b.rect(2, 8, 74, 23, body)
		b.rect(2, 8, 74, 1, shade(body, 1.25))
		b.rect(2, 28, 74, 3, shade(body, 0.7))
		b.rect(56, 11, 16, 11, c(0x7aa0c0))
		b.rect(10, 12, 38, 8, shade(body, 0.85))
		b.rect(2, 21, 3, 3, c(0xe03030))
		b.rect(73, 22, 3, 3, c(0xfff0b0))
		vehWheels(b, 16, 62)
	})
	add("boxtruck", 100, 44, 78, 2, func(b *pixBuf, v int) {
		box := c([]uint32{0xe8e8ec, 0x4a7ac8}[v%2])
		vehShadow(b, 100)
		b.rect(2, 4, 66, 27, box)
		b.rect(2, 4, 66, 2, shade(box, 1.2))
		b.rect(2, 27, 66, 4, shade(box, 0.72))
		b.rect(10, 12, 50, 6, shade(box, 0.82))
		b.rect(68, 12, 28, 19, c(0xc03a3a))
		b.rect(84, 15, 11, 9, c(0x7aa0c0))
		b.rect(68, 12, 28, 1, c(0xe06060))
		b.rect(93, 24, 3, 3, c(0xfff0b0))
		vehWheels(b, 14, 50, 82)
	})
	add("icetruck", 84, 46, 62, 1, func(b *pixBuf, v int) {
		vehShadow(b, 84)
		cream := c(0xf6e8d0)
		b.rect(2, 10, 78, 21, cream)
		b.rect(2, 10, 78, 1, c(0xffffff))
		b.rect(2, 24, 78, 3, c(0xf08aa8)) // pink stripe
		b.rect(2, 28, 78, 3, shade(cream, 0.7))
		b.rect(60, 13, 16, 10, c(0x7aa0c0))
		b.rect(14, 13, 28, 9, c(0x2a2a34)) // serving window
		b.rect(14, 13, 28, 2, c(0xf0508a)) // awning
		// the cone on the roof
		b.rect(34, 6, 3, 4, c(0xd8a860))
		b.ellipse(35.5, 4, 5, 4, c(0xf8c8d8))
		b.ellipse(35.5, 1, 3, 3, c(0xe85088))
		vehWheels(b, 16, 62)
	})
	add("schoolbus", 112, 42, 70, 1, func(b *pixBuf, v int) {
		vehShadow(b, 112)
		y := c(0xf0b818)
		b.rect(2, 6, 106, 25, y)
		b.rect(2, 6, 106, 1, c(0xffe060))
		b.rect(2, 22, 106, 2, c(0x2a2a30))
		b.rect(2, 27, 106, 4, shade(y, 0.7))
		for x := 8; x < 80; x += 11 {
			b.rect(x, 10, 8, 9, c(0x7aa0c0))
		}
		b.rect(92, 10, 12, 9, c(0x7aa0c0))
		b.rect(82, 10, 8, 18, c(0x2a2a34)) // door
		b.rect(104, 20, 3, 3, c(0xfff0b0))
		b.rect(2, 20, 2, 3, c(0xe03030))
		vehWheels(b, 22, 90)
	})
	add("taxi", 66, 38, 105, 1, func(b *pixBuf, v int) {
		genCar2(b, 0xf0c818)
		b.rect(26, 1, 14, 5, c(0xf4f4f8))
		b.rect(28, 2, 10, 2, c(0x2a2a34))
		b.rect(2, 22, 62, 2, c(0x2a2a34))
	})
	add("batmobile", 84, 34, 150, 1, func(b *pixBuf, v int) {
		vehShadow(b, 84)
		k := c(0x14141c)
		b.rect(2, 20, 80, 11, k)
		b.rect(2, 20, 80, 1, c(0x4a4a58))
		b.rect(8, 14, 30, 7, k)
		b.rect(28, 11, 20, 10, c(0x24242e)) // canopy
		b.rect(30, 12, 14, 7, c(0x2a3a58))
		b.rect(54, 17, 26, 4, k)
		b.rect(0, 8, 8, 14, k) // fins
		b.rect(2, 4, 5, 6, k)
		b.rect(66, 22, 14, 2, c(0xe8c030))
		b.rect(2, 24, 3, 3, c(0xff4a20))
		vehWheels(b, 18, 64)
	})
	add("delorean", 72, 36, 100, 2, func(b *pixBuf, v int) {
		vehShadow(b, 72)
		s := c(0xb8bcc6)
		b.rect(2, 18, 66, 12, s)
		b.rect(2, 18, 66, 1, c(0xe8ecf4))
		b.rect(2, 27, 66, 3, shade(s, 0.65))
		b.rect(12, 8, 34, 11, shade(s, 0.9))
		b.rect(14, 10, 14, 8, c(0x4a6080))
		b.rect(30, 10, 14, 8, c(0x4a6080))
		b.rect(2, 20, 6, 6, c(0x3a3a44)) // rear vents
		b.rect(60, 22, 6, 2, c(0xfff0b0))
		b.rect(34, 6, 18, 3, c(0x6a6a74)) // gull-wing door, half open
		vehWheels(b, 16, 54)
	})
	add("homercar", 66, 38, 90, 1, func(b *pixBuf, v int) {
		genCar2(b, 0xf08aa8)
		b.rect(2, 22, 62, 2, c(0xffc8d8))
	})
	add("wackymean", 80, 34, 135, 1, func(b *pixBuf, v int) {
		vehShadow(b, 80)
		p := c(0x7a30c0)
		b.rect(2, 20, 76, 11, p)
		b.rect(2, 20, 76, 1, c(0xb070f0))
		b.rect(34, 12, 18, 9, shade(p, 0.85))
		b.rect(38, 8, 8, 5, c(0xf0c8a0)) // driver
		b.rect(0, 10, 7, 12, p)
		b.rect(70, 24, 10, 3, c(0xe8c030))
		b.rect(62, 14, 3, 7, c(0x2a2a34)) // exhausts
		b.rect(66, 14, 3, 7, c(0x2a2a34))
		vehWheels(b, 14, 62)
	})
	add("wackyboulder", 70, 38, 90, 1, func(b *pixBuf, v int) {
		vehShadow(b, 70)
		g := c(0x8a8a90)
		b.ellipse(34, 17, 30, 14, g)
		b.ellipse(30, 13, 18, 8, shade(g, 1.15))
		b.rect(18, 8, 4, 2, shade(g, 0.6))
		b.rect(42, 20, 8, 2, shade(g, 0.6))
		b.rect(26, 6, 14, 5, c(0x7aa0c0))
		vehWheels(b, 14, 54)
	})
	add("wackypink", 66, 38, 125, 1, func(b *pixBuf, v int) {
		genCar2(b, 0xf860a8)
		b.rect(26, 6, 14, 3, c(0xffe0f0))
		b.rect(6, 17, 54, 1, c(0xffc0e0))
	})
}

// genCar2 is genCar's body with an arbitrary colour.
func genCar2(b *pixBuf, col uint32) {
	body := c(col)
	b.ellipse(33, 31, 30, 4, color.RGBA{0, 0, 0, 90})
	b.rect(2, 16, 62, 15, body)
	b.rect(2, 16, 62, 1, shade(body, 1.25))
	b.rect(2, 28, 62, 3, shade(body, 0.7))
	b.rect(14, 5, 36, 13, shade(body, 0.9))
	b.rect(14, 5, 36, 1, shade(body, 1.25))
	b.rect(16, 7, 14, 9, c(0x7aa0c0))
	b.rect(33, 7, 15, 9, c(0x7aa0c0))
	b.rect(31, 5, 2, 12, shade(body, 0.7))
	b.rect(2, 20, 3, 3, c(0xfff0b0))
	b.rect(61, 20, 3, 3, c(0xe03030))
	vehWheels(b, 15, 51)
}

package main

import "image/color"

// ---------------------------------------------------------------------------
// Murray, the evil talking skull, on a stake (see murray.go). Three frames:
// mouth shut, open and wide open (he alternates them while he talks).
// ---------------------------------------------------------------------------

func init() {
	propKinds["murray"] = &PropKind{W: 24, H: 46, FW: 8, FH: 4, Vars: 3, Draw: drawMurray}
}

func drawMurray(b *pixBuf, v int) {
	bone, boneD, boneL := c(0xe8e0c8), c(0xb4ab90), c(0xfff8e4)
	dark := c(0x160c12)
	wood, woodD := c(0x7a5230), c(0x4e3220)
	open := []int{0, 3, 6}[v%3]
	// a little mound of earth and the stake
	b.ellipse(12, 44, 9, 2.4, c(0x3a2c20))
	b.ellipse(12, 43.4, 7, 1.8, c(0x56422e))
	b.rect(10, 24+open, 4, 20-open, wood)
	b.rect(13, 24+open, 1, 20-open, woodD)
	b.rect(10, 28+open, 4, 1, woodD)
	b.rect(10, 36+open, 4, 1, woodD)
	// horns
	for i := 0; i < 5; i++ {
		b.rect(3-i/2, 6-i, 2, 1, c(0x8a2a2a))
		b.rect(19+i/2, 6-i, 2, 1, c(0x8a2a2a))
	}
	b.set(1, 1, c(0xb83a3a))
	b.set(22, 1, c(0xb83a3a))
	// cranium
	b.ellipse(12, 12, 9, 8, boneD)
	b.ellipse(11.5, 11.5, 8.4, 7.4, bone)
	b.ellipse(9, 8, 4.5, 3, boneL)
	// eyes: black sockets with a red glow
	for _, ex := range []float64{8, 16} {
		b.ellipse(ex, 12, 3, 3.6, dark)
		b.ellipse(ex, 12.4, 1.7, 2.2, c(0xff2a1a))
		b.set(int(ex), 12, c(0xffd0a0))
		b.set(int(ex)-3, 9, c(0x9a2418))
		b.set(int(ex)+2, 9, c(0x9a2418))
	}
	// nose and cheekbones
	b.set(11, 16, dark)
	b.set(12, 16, dark)
	b.set(11, 17, dark)
	b.rect(4, 15, 2, 2, boneD)
	b.rect(18, 15, 2, 2, boneD)
	// upper teeth
	b.rect(6, 18, 12, 2, bone)
	for x := 7; x <= 17; x += 2 {
		b.set(x, 19, boneD)
	}
	// the jaw drops when he talks
	jy := 20 + open
	if open > 0 {
		b.rect(7, 20, 10, open, dark)
		b.rect(9, 20+open-1, 6, 1, c(0x7a1a2a)) // tongue
	}
	b.rect(6, jy, 12, 4, boneD)
	b.rect(7, jy, 10, 3, bone)
	for x := 8; x <= 16; x += 2 {
		b.set(x, jy, boneD)
	}
	b.rect(7, jy+3, 10, 1, boneD)
	_ = color.RGBA{}
}

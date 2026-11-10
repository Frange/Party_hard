package main

import "image/color"

// ---------------------------------------------------------------------------
// Murray, the grumpy talking skull, on a short stake (see murray.go). A cartoon
// skull: white bone with a thick black outline, a big cranium that bulges to one
// side, large black sockets with angry brows and a white glint, a small nose
// hole, square upper teeth and a separate lower jaw that hangs a bit loose with
// crooked teeth. Three frames: jaw barely open, open and wide open (he swaps
// them while he talks).
// ---------------------------------------------------------------------------

func init() {
	propKinds["murray"] = &PropKind{W: 28, H: 42, FW: 8, FH: 4, Vars: 3, Draw: drawMurray}
}

func drawMurray(b *pixBuf, v int) {
	ink := c(0x0a0a0c)
	bone, boneS, boneL := c(0xf6f2e6), c(0xcfc8b4), c(0xffffff)
	wood, woodD := c(0x7a5230), c(0x4e3220)
	open := []int{1, 3, 5}[v%3]

	// a little mound of earth and the stake under the jaw
	b.ellipse(13, 40.5, 8, 2, c(0x3a2c20))
	b.ellipse(13, 40, 6, 1.5, c(0x56422e))
	b.rect(11, 26, 4, 14, ink)
	b.rect(12, 26, 2, 14, wood)
	b.rect(13, 26, 1, 14, woodD)

	// the skull is painted on its own layers so each piece gets a solid black contour
	paste := func(s *pixBuf) {
		s = s.outlined(ink)
		for y := 0; y < s.h; y++ {
			for x := 0; x < s.w; x++ {
				if i := (y*s.w + x) * 4; s.p[i+3] != 0 {
					b.set(x-b.ox, y-b.oy, rgbaAt(s.p[i:i+4]))
				}
			}
		}
	}
	layer := func() *pixBuf {
		s := newPix(b.w, b.h)
		s.ox, s.oy = b.ox, b.oy
		return s
	}
	// the lower jaw goes first (behind): it hangs loose, narrower, and droops to the left
	jy := 22 + open
	if open > 0 {
		b.rect(6, 20, 11, jy-19, ink) // the dark inside of the mouth
	}
	j := layer()
	j.rect(6, jy, 11, 3, bone)
	j.rect(5, jy+1, 2, 3, bone)
	j.rect(7, jy+3, 9, 1, boneS)
	j.rect(15, jy+2, 2, 1, boneS)
	paste(j)

	s := layer()
	// cranium: round, bulging up and back to the right (lopsided)
	s.ellipse(11.5, 10, 9, 8.5, bone)
	s.ellipse(17.5, 7, 8.5, 6.5, bone)
	s.ellipse(20.5, 11, 5, 5, bone)
	// maxilla / cheeks, narrower than the cranium
	s.rect(4, 13, 14, 6, bone)
	s.rect(5, 19, 13, 2, bone)
	// shading on the underside and the back of the cranium, a highlight on top
	s.ellipse(22, 12.5, 3, 3, boneS)
	s.rect(17, 16, 5, 1, boneS)
	s.ellipse(10, 4, 4, 1.8, boneL)
	paste(s)

	// big black sockets with a white glint
	b.ellipse(7, 12, 3.1, 3.6, ink)
	b.ellipse(15.5, 12, 3.3, 3.8, ink)
	b.rect(6, 11, 2, 2, boneL)
	b.rect(14, 11, 2, 2, boneL)
	// angry brows: thick black ridges sloping down towards the nose
	b.line(3, 6, 9, 9, ink)
	b.line(3, 7, 9, 10, ink)
	b.line(20, 6, 13, 9, ink)
	b.line(20, 7, 13, 10, ink)
	// small nose hole
	b.rect(10, 16, 2, 1, ink)
	b.set(11, 15, ink)
	b.set(10, 17, ink)
	// square upper teeth on a black gum line
	b.rect(5, 18, 13, 1, ink)
	for x := 5; x <= 14; x += 3 {
		b.rect(x+1, 19, 2, 2, bone)
		b.set(x, 19, ink)
		b.set(x, 20, ink)
	}
	b.rect(17, 19, 1, 2, ink)
	b.rect(5, 21, 13, 1, ink)
	// crooked lower teeth: uneven heights and one missing
	for i, x := range []int{7, 9, 11, 13, 15} {
		h := []int{2, 1, 0, 2, 1}[i]
		if h > 0 && h <= open+1 {
			b.rect(x, jy-h, 1+i%2, h, bone)
		}
	}
}

func rgbaAt(p []byte) (c color.RGBA) { return color.RGBA{p[0], p[1], p[2], p[3]} }

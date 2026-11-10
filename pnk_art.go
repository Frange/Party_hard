package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Pop-punk bands (Sum 41, Blink-182, The Offspring): our own pixel-art archetypes of STYLE, not
// portraits of anyone: hair (spikes, crests), clothes, prints, studded belts, tattoos, beer cans
// and drumsticks. Overlay hooks that genChar calls (head, torso, arms). Visual only: hitbox, speed
// and stats do not change. The members show their real name when it is known for sure (bandRealNames in bands.go),
// else "<band> · <role>".
//   Sum 41:        cantante (white cap, black tee with a winged print, jeans, studded belt)
//                  guitarra (black tee with a white print), bajo (yellow spikes, black tee with a
//                  rude-hand print, khaki trousers), batería (orange-red spiky crest, white tee with
//                  a framed print). All four hold a beer can.
//   Blink-182:     bajo (short dark hair, skate tee), guitarra (short hair, tattooed arms),
//                  batería (black mohawk with spikes, black vest, tattoos, drumsticks).
//   The Offspring: cantante (short blond hair, black polo), guitarra (black spiky hair, dark
//                  glasses, black and white check shirt), bajo and batería; all wear the band's
//                  flaming skull (a yellow ring with a red field and a yellow skull) somewhere.
// Everything is painted into the head (part 1), torso (2) and arms (3, 4), so the exported pictures
// of special_people keep it when the game animates the rig.
// ---------------------------------------------------------------------------

const (
	pnkNone = iota
	pnkSumVoz
	pnkSumGuit
	pnkSumBajo
	pnkSumBat
	pnkBlkBajo
	pnkBlkGuit
	pnkBlkBat
	pnkOffVoz
	pnkOffGuit
	pnkOffBajo
	pnkOffBat
)

// pnkSlugKind: the slugs (kept from the old saves) and their look.
var pnkSlugKind = map[string]int{
	"deryck": pnkSumVoz, "dave_baksh": pnkSumGuit, "jason_mc": pnkSumBajo, "frank_zummo": pnkSumBat,
	"mark_hoppus": pnkBlkBajo, "tom_delonge": pnkBlkGuit, "travis_barker": pnkBlkBat,
	"off_cantante": pnkOffVoz, "off_guitarra": pnkOffGuit, "off_bajo": pnkOffBajo, "off_bateria": pnkOffBat,
}

func pnkKind(s *CharSpec) int { return pnkSlugKind[s.Name] }

var (
	pnkInk     = hx(0x26304e) // tattoo ink
	pnkInk2    = hx(0x8a2a3a)
	pnkStud    = hx(0xd8dce4)
	pnkBelt    = hx(0x16161c)
	pnkWhite   = hx(0xf4f4f4)
	pnkBlack   = hx(0x141418)
	pnkStick   = hx(0xe8d4a0)
	pnkStickD  = hx(0xb89a60)
	pnkLogoY   = hx(0xf0c818) // the ring
	pnkLogoF   = hx(0xffe860) // flames
	pnkLogoR   = hx(0xd82020) // the red field
	pnkLogoS   = hx(0xfff2a8) // the skull
	pnkBlond   = hx(0xf4d81c)
	pnkBlondL  = hx(0xfff27a)
	pnkBlondD  = hx(0xc8a010)
	pnkOrange  = hx(0xe8501a)
	pnkOrangeL = hx(0xff8a3a)
	pnkOrangeD = hx(0xb03810)
	pnkCrest   = hx(0x18181e)
	pnkCrestL  = hx(0x4a4a5a)
	pnkSkate   = hx(0xe8b040)
	pnkWing    = hx(0x7ad0f0)
	pnkCheckW  = hx(0xe2e2e2)
	pnkCheckK  = hx(0x1c1c22)
)

// pnkLogo5: the flaming skull, 5x5, with its top left corner at (x, y).
func pnkLogo5(b *pixBuf, x, y int) {
	rows := [5]string{".YYY.", "YFRFY", "YSSSY", "YRSRY", ".YYY."}
	pnkPaint(b, x, y, rows[:])
}

// pnkLogo3: a small embroidered version, 3x3.
func pnkLogo3(b *pixBuf, x, y int) {
	rows := [3]string{"FRF", "SSS", "RSR"}
	pnkPaint(b, x, y, rows[:])
}

func pnkPaint(b *pixBuf, x, y int, rows []string) {
	for dy, r := range rows {
		for dx, ch := range r {
			var c color.RGBA
			switch ch {
			case 'Y':
				c = pnkLogoY
			case 'F':
				c = pnkLogoF
			case 'R':
				c = pnkLogoR
			case 'S':
				c = pnkLogoS
			case 'K':
				c = pnkBlack
			case 'W':
				c = pnkWhite
			default:
				continue
			}
			b.set(x+dx, y+dy, c)
		}
	}
}

// pnkSpikes: a crown of hair spikes over the head, one column each, heights[i] tall above top.
// Columns follow x0, x0+1...; the tips are lighter.
func pnkSpikes(b *pixBuf, x0, top int, heights []int, col, tip, dark color.RGBA) {
	for i, h := range heights {
		x := x0 + i
		if h <= 0 {
			continue
		}
		c := col
		if i%2 == 1 {
			c = dark
		}
		b.rect(x, top-h+1, 1, h+1, c)
		b.set(x, top-h+1, tip)
	}
}

// pnkHead: spikes, crests and blond hair.
func pnkHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := pnkKind(s)
	if k == pnkNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	hw := int(math.Floor(rx + 0.5))
	side, back := vw == vSide, vw == vBack
	switch k {
	case pnkSumBajo: // a crown of bright yellow spikes all over the head
		hs := []int{2, 4, 2, 5, 3, 5, 2, 4, 2}
		if side {
			hs = []int{3, 2, 4, 2, 5, 3, 4, 1}
		}
		n := minI(len(hs), 2*hw+1)
		pnkSpikes(b, ihx-n/2, top, hs[:n], pnkBlond, pnkBlondL, pnkBlondD)
		b.rect(ihx-hw, top+1, 2*hw+1, 1, pnkBlond)
		if back {
			b.rect(ihx-hw, top+1, 2*hw+1, ey+1-top, pnkBlondD)
			b.rect(ihx-hw+1, top+1, 2*hw-1, 2, pnkBlond)
		}
	case pnkSumBat: // an orange-red crest of spikes, tallest in the middle
		hs := []int{1, 3, 2, 5, 4, 6, 4, 5, 2, 3, 1}
		if side {
			hs = []int{2, 4, 3, 6, 4, 5, 3, 2}
		}
		n := minI(len(hs), 2*hw+3)
		pnkSpikes(b, ihx-n/2, top, hs[(len(hs)-n)/2:(len(hs)-n)/2+n], pnkOrange, pnkOrangeL, pnkOrangeD)
		b.rect(ihx-hw, top+1, 2*hw+1, 1, pnkOrange)
		if back {
			b.rect(ihx-hw, top+1, 2*hw+1, ey+1-top, pnkOrangeD)
		}
	case pnkBlkBat: // shaved sides and a tall black mohawk of spikes
		b.cur = 1
		if side {
			hs := []int{2, 4, 3, 5, 3, 5, 3, 4}
			n := minI(len(hs), 2*hw+1)
			pnkSpikes(b, ihx-hw, top, hs[:n], pnkCrest, pnkCrestL, pnkCrest)
			b.rect(ihx-hw, top+1, n, 1, pnkCrest)
			b.rect(ihx-hw, top+2, 2, ey-top-1, shade(s.Skin, 0.7)) // shaved nape
		} else {
			hs := []int{5, 6, 4}
			pnkSpikes(b, ihx-1, top, hs, pnkCrest, pnkCrestL, pnkCrest)
			b.rect(ihx-1, top+1, 3, 1, pnkCrest)
			b.rect(ihx-hw, top+2, 1, 2, shade(s.Skin, 0.72)) // shaved sides
			b.rect(ihx+hw, top+2, 1, 2, shade(s.Skin, 0.72))
			if back {
				b.rect(ihx, top+1, 1, ey+1-top, pnkCrest) // the strip down the nape
			}
		}
	case pnkOffVoz: // short bleached blond hair, a little spiky
		hs := []int{1, 2, 1, 2, 1, 2, 1, 2, 1}
		n := minI(len(hs), 2*hw+1)
		pnkSpikes(b, ihx-n/2, top, hs[:n], s.Hair, shade(s.Hair, 1.12), shade(s.Hair, 0.85))
	case pnkOffGuit: // black hair standing up in spikes
		hs := []int{2, 3, 1, 4, 2, 4, 1, 3, 2}
		if side {
			hs = []int{1, 3, 2, 4, 2, 3, 1}
		}
		n := minI(len(hs), 2*hw+1)
		pnkSpikes(b, ihx-n/2, top, hs[:n], pnkCrest, pnkCrestL, pnkCrest)
	}
}

// pnkTorso: prints, the flaming skull, polo collars, check shirts, studded belts and tank tattoos.
func pnkTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := pnkKind(s)
	if k == pnkNone {
		return
	}
	b.cur = 2
	tw := x1 - x0
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	studs := func() { // a black belt with silver studs on the trousers
		b.rect(x0, y1-1, tw, 1, pnkBelt)
		for x := x0; x < x1; x += 2 {
			b.set(x, y1-1, pnkStud)
		}
	}
	switch k {
	case pnkSumVoz: // a winged print on the black tee
		if front {
			b.set(mid-2, y0+2, pnkWing)
			b.set(mid+2, y0+2, pnkWing)
			b.set(mid-1, y0+3, pnkWing)
			b.set(mid+1, y0+3, pnkWing)
			b.set(mid, y0+3, pnkLogoR)
			b.set(mid, y0+4, pnkLogoY)
			b.set(mid, y0+5, pnkLogoR)
		}
		studs()
	case pnkSumGuit: // a white print with a line of letters on the black tee
		if front {
			b.rect(mid-1, y0+2, 3, 1, pnkWhite)
			b.set(mid-1, y0+4, pnkWhite)
			b.set(mid+1, y0+4, pnkWhite)
			b.rect(mid, y0+5, 1, 2, pnkWhite)
		} else if back {
			b.rect(mid-1, y0+2, 3, 1, shade(pnkWhite, 0.8))
		} else if side {
			b.set(x1-2, y0+2, pnkWhite)
			b.set(x1-2, y0+4, pnkWhite)
		}
	case pnkSumBajo: // the rude hand print on the black tee, khaki trousers with studs
		if front {
			b.rect(mid, y0+2, 1, 2, pnkWhite)
			b.rect(mid-1, y0+4, 3, 2, pnkWhite)
			b.set(mid, y0+5, shade(pnkWhite, 0.75))
		}
		studs()
	case pnkSumBat: // a framed print on the white tee, studded belt
		if front {
			b.rect(mid-1, y0+2, 3, 4, pnkBlack)
			b.set(mid, y0+3, hx(0x9a9aa4))
			b.set(mid, y0+4, hx(0x6a6a74))
		}
		studs()
	case pnkBlkBajo: // a skateboard on the tee
		if front {
			b.rect(mid-2, y0+4, 5, 1, pnkSkate)
			b.set(mid-2, y0+3, pnkSkate)
			b.set(mid+2, y0+3, pnkSkate)
			b.set(mid-1, y0+5, pnkWhite)
			b.set(mid+1, y0+5, pnkWhite)
		} else if back { // the brand across the shoulders
			b.rect(mid-1, y0+2, 3, 1, pnkSkate)
		} else if side {
			b.set(x1-2, y0+4, pnkSkate)
		}
	case pnkBlkGuit: // a white star print on the dark red tee
		if front {
			b.set(mid, y0+2, pnkWhite)
			b.rect(mid-1, y0+3, 3, 1, pnkWhite)
			b.set(mid-1, y0+4, pnkWhite)
			b.set(mid+1, y0+4, pnkWhite)
		}
	case pnkBlkBat: // tattoos on the shoulders and the chest above the black vest
		b.set(x0, y0, pnkInk)
		b.set(x1-1, y0+1, pnkInk)
		if front {
			b.set(mid, y0, pnkInk2)
			b.set(mid-1, y0, s.Skin)
			b.set(mid+1, y0, s.Skin)
		}
		if back {
			b.rect(mid-1, y0, 3, 1, s.Skin)
			b.set(mid, y0, pnkInk)
		}
	case pnkOffVoz: // a black polo: a collar, a button and the skull embroidered on the chest
		if front {
			b.set(mid-1, y0, hx(0x3a3a44))
			b.set(mid+1, y0, hx(0x3a3a44))
			b.set(mid, y0, s.Skin)
			b.set(mid, y0+1, hx(0x5a5a64))
			pnkLogo3(b, mid+(tw/2)-3, y0+2)
		} else if back {
			b.rect(mid-1, y0, 3, 1, hx(0x3a3a44))
		} else if side {
			b.set(x1-1, y0, hx(0x3a3a44))
		}
	case pnkOffGuit: // a black and white check shirt, open on a black tee; the skull on the back
		for y := y0; y < y1-1; y++ {
			for x := x0; x < x1; x++ {
				c := pnkCheckK
				if ((x-x0)>>1+(y-y0)>>1)%2 == 0 {
					c = pnkCheckW
				}
				if y == y0 || x == x1-1 {
					c = shade(c, 0.85)
				}
				b.set(x, y, c)
			}
		}
		switch vw {
		case vFront:
			b.rect(mid, y0, 1, y1-1-y0, pnkBlack) // the tee in the opening
			b.set(mid, y0+3, pnkLogoY)
			b.set(mid, y0+4, pnkLogoR)
		case vBack:
			pnkLogo5(b, mid-2, y0+2)
		case vSide:
			b.rect(x1-1, y0, 1, y1-1-y0, pnkBlack)
		}
	case pnkOffBajo, pnkOffBat: // the flaming skull on the black tee or vest
		switch vw {
		case vFront:
			pnkLogo5(b, mid-2, y0+2)
		case vBack:
			if k == pnkOffBajo {
				pnkLogo5(b, mid-2, y0+2)
			}
		case vSide:
			b.set(x1-2, y0+3, pnkLogoY)
			b.set(x1-2, y0+4, pnkLogoR)
		}
	}
}

// pnkArm: tattoos, check sleeves and drumsticks (called from drawArm, same path as thickLine).
func pnkArm(b *pixBuf, s *CharSpec, vw int, sx, sy, mx, my, ex, ey float64, armW int, dark bool, id uint8) {
	k := pnkKind(s)
	if k == pnkNone {
		return
	}
	b.cur = id
	walk := func(x0, y0, x1, y1 float64, f func(i, x, y, dx int)) {
		steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))*2) + 1
		for i := 0; i <= steps; i++ {
			t := float64(i) / float64(steps)
			x := int(math.Floor(x0 + (x1-x0)*t))
			y := int(math.Floor(y0 + (y1-y0)*t))
			for dx := 0; dx < armW; dx++ {
				f(i, x, y, dx)
			}
		}
	}
	ink := pnkInk
	if dark {
		ink = shade(ink, 0.8)
	}
	switch k {
	case pnkBlkGuit, pnkBlkBat: // tattooed arms: ink along the forearm (and the bare upper arm)
		walk(mx, my, ex, ey, func(i, x, y, dx int) {
			if (i+dx)%3 == 1 {
				b.set(x+dx, y, ink)
			}
		})
		if k == pnkBlkBat {
			walk(sx, sy+1, mx, my, func(i, x, y, dx int) {
				if (i+dx)%2 == 0 {
					c := ink
					if i%4 == 0 {
						c = pnkInk2
					}
					b.set(x+dx, y, c)
				}
			})
		}
	case pnkOffGuit: // check sleeves down to the wrist
		paint := func(i, x, y, dx int) {
			c := pnkCheckK
			if ((x+dx)>>1+y>>1)%2 == 0 {
				c = pnkCheckW
			}
			if dark {
				c = shade(c, 0.8)
			}
			b.set(x+dx, y, c)
		}
		walk(sx, sy, mx, my, paint)
		walk(mx, my, ex, ey, paint)
	}
	if k == pnkBlkBat || k == pnkOffBat { // a drumstick in each hand
		x, y := int(math.Floor(ex+0.5)), int(math.Floor(ey+0.5))
		d := 1
		switch {
		case vw == vSide:
			d = 1
		case ex < sx-0.3:
			d = -1
		case ex <= sx+0.3 && id == 3:
			d = -1
		}
		st, sd := pnkStick, pnkStickD
		if dark {
			st, sd = shade(st, 0.85), shade(sd, 0.85)
		}
		ox := x + armW/2
		// held low and forward, the tips away from the body
		for i, o := range [][2]int{{1, 1}, {1, 2}, {2, 3}, {2, 4}, {3, 5}, {3, 6}} {
			c := st
			if i >= 4 {
				c = sd
			}
			b.set(ox+d*o[0], y+o[1]-1, c)
		}
	}
}

// pogo: the punks jump up and down with both feet together when they dance (gait.go).
func init() {
	gaitDefs["pogo"] = gaitDef{dance: AnimHop}
	for _, s := range []string{"jason_mc", "frank_zummo", "tom_delonge", "travis_barker", "off_guitarra", "off_bateria"} {
		gaitSlugs[s] = "pogo"
	}
}

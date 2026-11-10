package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Our own pixel-art takes on the look of four rock groups (slugs of bands.go: "bt_*", "lp_*",
// "des_*", "ps_*"), as overlay hooks that genChar calls (head, torso, arms). Only STYLE
// archetypes: haircuts, colours, clothes and generic tattoos; nobody's face or name. Visual only:
// hitbox, speed and stats do not change. The base specs live in bands.go (every random colour is
// fixed there, so each member always draws the same and the exported pictures can follow the code).
//   - Billy Talent: all in black. Singer with long black hair and a fringe over one eye; guitarist
//     with a tall stiff black crest and white wristbands; bassist with a shaved head, black tee and
//     tattooed arms; drummer with short hair, white tee and black braces.
//   - Lost Prophets (six): emo-rock fringes, black tees with a printed emblem, worn orange-brown
//     jeans with a belt, tattoos and wristbands; the drummer has a beard. Two new members here:
//     "lp_ritmo" (rhythm guitar) and "lp_corista" (backing vocals).
//   - Despistaos: casual indie-pop; the singer now wears SHOCKING PINK hair over a dark beard and a
//     dark green shirt; one with shades and a grey tee, one bearded with a printed white tee, one
//     curly with glasses, a printed white tee and a tattoo.
//   - Punk Sailor (an archetype, "sailor punk"): the captain (dark captain's cap with a golden
//     band and badge, black leather jacket over a navy and white striped tee, silver chain), one in
//     a white shirt with black VERTICAL stripes, one in a black sleeveless waistcoat with shades and
//     tattoos, and one with a long mane, a goatee and a white jacket.
// ---------------------------------------------------------------------------

const (
	bsbNone = iota
	bsbBtSing
	bsbBtGuitar
	bsbBtBass
	bsbBtDrum
	bsbLpSing
	bsbLpGuitar
	bsbLpBass
	bsbLpDrum
	bsbLpRhythm
	bsbLpChoir
	bsbDesSing
	bsbDesGuitar
	bsbDesBass
	bsbDesDrum
	bsbPsSing
	bsbPsGuitar
	bsbPsBass
	bsbPsDrum
)

// bsbKinds: the slugs drawn here.
var bsbKinds = map[string]int{
	"bt_cantante": bsbBtSing, "bt_guitarra": bsbBtGuitar, "bt_bajo": bsbBtBass, "bt_bateria": bsbBtDrum,
	"lp_cantante": bsbLpSing, "lp_guitarra": bsbLpGuitar, "lp_bajo": bsbLpBass, "lp_bateria": bsbLpDrum,
	"lp_ritmo": bsbLpRhythm, "lp_corista": bsbLpChoir,
	"des_cantante": bsbDesSing, "des_guitarra": bsbDesGuitar, "des_bajo": bsbDesBass, "des_bateria": bsbDesDrum,
	"ps_cantante": bsbPsSing, "ps_guitarra": bsbPsGuitar, "ps_bajo": bsbPsBass, "ps_bateria": bsbPsDrum,
}

func bsbKind(s *CharSpec) int { return bsbKinds[s.Name] }

var (
	bsbWhite   = hx(0xf4f4f4)
	bsbBlack   = hx(0x101014)
	bsbGrey    = hx(0x5a5a64)
	bsbSilver  = hx(0xc8ccd4)
	bsbSilverL = hx(0xf0f2f6)
	bsbGold    = hx(0xe8c040)
	bsbGoldL   = hx(0xfff0a0)
	bsbNavy    = hx(0x1e2a58)
	bsbCapNavy = hx(0x161a28)
	bsbInk     = hx(0x2e5a78) // tattoo ink
	bsbBand    = hx(0x8a3ab8) // a purple rubber wristband
	bsbBelt    = hx(0x2a1e16)
	bsbBeard   = hx(0x3a2414) // the pink-haired singer keeps a dark beard
	bsbPrintR  = hx(0xc83a3a)
)

// the presets of the two new Lost Prophets members. A fixed random source of their own, so the
// shared random stream of builtinChars (export.go) is not moved for the characters after them.
func presetLpRitmo(*rand.Rand) CharSpec {
	return def("lp_ritmo", skinc(0xf0d0b4), hairc(0xa07a48, HairSide), hatc(HatNone, 0), tops(TopTee, 0x18181c, 0xf0f0f0),
		pantsc(0x9a5a2e), shoesc(0x16161c)).F(rand.New(rand.NewSource(2004)))
}

func presetLpCorista(*rand.Rand) CharSpec {
	return def("lp_corista", skinc(0xe0b894), hairc(0x16121a, HairSpiky), hatc(HatNone, 0), tops(TopTee, 0x18181c, 0xc83a3a),
		pantsc(0xa0582a), shoesc(0xf0f0f0)).F(rand.New(rand.NewSource(2006)))
}

// bsbNewSlugs: the members added by this file (the others were already in bands.go).
var bsbNewSlugs = []string{"lp_ritmo", "lp_corista"}

func init() {
	famousPresets = append(famousPresets, presetLpRitmo, presetLpCorista)
	specialPool = append(specialPool, bsbNewSlugs...) // so they have their card in OBJETIVOS / PERSONAJES
	displayNameOverrides["lp_ritmo"] = bandMemberName("lp_ritmo", "Lost Prophets", "rítmica")
	displayNameOverrides["lp_corista"] = bandMemberName("lp_corista", "Lost Prophets", "corista")
	for i := range bandList {
		if bandList[i].slug == "lostprophets" { // six on the stage: the two new ones beside the drummer
			bandList[i].members = append(row("lp_cantante", "lp_guitarra", "lp_bajo", "lp_bateria"),
				bandMember{"lp_ritmo", "guitar", -30, -20}, bandMember{"lp_corista", "sing", 30, -20})
		}
	}
	// walking styles (only drawing; the players on the stage keep their instrument poses)
	for slug, g := range map[string]string{"bt_cantante": "hop", "ps_cantante": "stagger", "lp_cantante": "hips", "des_cantante": "hop"} {
		if _, ok := gaitSlugs[slug]; !ok {
			gaitSlugs[slug] = g
		}
	}
	for k, v := range map[string]string{"Rítmica": "Rhythm", "Corista": "Backing vox"} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

// bsbHead: fringes, crest, shaved head, the dark beard under the pink hair, the captain's cap.
func bsbHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := bsbKind(s)
	if k == bsbNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	hw := int(math.Floor(rx + 0.5))
	nose := int(math.Floor(hxp + rx))
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	hairC := s.Hair
	// fringe: hair over the face from the top down to the eyes, on one side
	fringe := func(left bool, low int) {
		switch {
		case front:
			x0, x1 := ihx-hw, ihx
			if !left {
				x0, x1 = ihx, ihx+hw
			}
			for y := top + 1; y <= low; y++ {
				for x := x0; x <= x1; x++ {
					if b.get(x, y).A != 0 {
						b.set(x, y, hairC)
					}
				}
			}
		case side:
			for y := top + 1; y <= low; y++ {
				for x := nose - 2; x <= nose; x++ {
					if b.get(x, y).A != 0 {
						b.set(x, y, hairC)
					}
				}
			}
		}
	}
	switch k {
	case bsbBtSing:
		fringe(true, ey) // covers one eye
		if front {
			b.set(ihx+1, top+1, hairC)
		}
	case bsbLpGuitar:
		fringe(false, ey)
	case bsbLpSing, bsbLpRhythm:
		fringe(true, ey-1)
	case bsbBtGuitar:
		// a tall stiff black crest: a quiff standing straight up, narrowing to ragged tips
		hi := shade(hairC, 3.2)
		dx := 0
		if side {
			dx = 1 // it leans forward a little
		}
		b.rect(ihx-2+dx, top-1, 5, 1, hairC)
		b.rect(ihx-1+dx, top-2, 3, 1, hairC)
		b.rect(ihx-1+2*dx, top-3, 3, 1, hairC)
		b.set(ihx-1+2*dx, top-4, hairC)
		b.set(ihx+1+2*dx, top-4, hairC)
		b.set(ihx+2*dx, top-5, hairC)
		if !back {
			b.set(ihx+dx, top-2, hi)
			b.set(ihx+2*dx, top-4, hi)
		}
	case bsbBtBass:
		// shaved head: a fine stubble shadow on the scalp
		sh := shade(s.Skin, 0.78)
		last := ey - 2 // down to just above the brows
		if back {
			last = top + 4
		}
		for y := top; y <= last; y++ {
			for x := ihx - hw; x <= ihx+hw; x++ {
				if c := b.get(x, y); c.A != 0 && (y < last || !back || (x+y)%2 == 0) {
					b.set(x, y, sh)
				}
			}
		}
	case bsbDesSing:
		// the beard stays dark under the pink hair
		if !back {
			for y := int(math.Floor(hyp + ry*0.2)); y <= int(math.Floor(hyp+ry+1)); y++ {
				for x := int(math.Floor(hxp - rx)); x <= int(math.Floor(hxp+rx)); x++ {
					if b.get(x, y) == hairC {
						b.set(x, y, bsbBeard)
					}
				}
			}
		}
	case bsbPsSing:
		// the captain's cap: a flat dark crown wider than the head, golden band and badge, a black peak
		b.rect(ihx-hw-1, top-3, 2*hw+3, 4, bsbCapNavy)
		b.rect(ihx-hw-2, top-3, 2*hw+5, 1, shade(bsbCapNavy, 1.5))
		b.rect(ihx-hw-1, top+1, 2*hw+3, 1, bsbGold)
		switch {
		case front:
			b.rect(ihx-hw, top+2, 2*hw+1, 1, bsbBlack)
			b.rect(ihx-1, top-2, 3, 2, bsbGold)
			b.set(ihx, top-2, bsbGoldL)
		case side:
			b.rect(ihx, top+2, hw+3, 1, bsbBlack)
			b.set(nose-1, top-1, bsbGold)
		}
	}
}

// bsbTorso: prints, belts, collars, the leather jacket over the stripes, vertical stripes, buttons.
func bsbTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := bsbKind(s)
	if k == bsbNone {
		return
	}
	b.cur = 2
	tw := x1 - x0
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	belt := func(c [3]uint32) {
		b.rect(x0, y1-1, tw, 1, hx(c[0]))
		if !back {
			bx := mid
			if side {
				bx = x1 - 2
			}
			b.rect(bx, y1-1, 2, 1, hx(c[1]))
			if c[2] != 0 {
				for x := x0; x < x1; x += 2 {
					if x != bx && x != bx+1 {
						b.set(x, y1-1, hx(c[2]))
					}
				}
			}
		}
	}
	switch k {
	case bsbBtSing, bsbBtGuitar:
		if front { // shirt collar and buttons
			b.set(mid-1, y0, bsbGrey)
			b.set(mid+1, y0, bsbGrey)
			for y := y0 + 2; y < y1-1; y += 2 {
				b.set(mid, y, bsbGrey)
			}
		}
	case bsbBtBass:
		belt([3]uint32{0x101014, 0xa8acb4, 0})
	case bsbBtDrum:
		if back { // the braces cross on the back
			for i := 0; i < y1-y0; i++ {
				x := x0 + 1 + i*(tw-3)/maxI(1, y1-y0-1)
				b.set(x, y0+i, s.Susp)
				b.set(x1-2-(x-x0-1), y0+i, s.Susp)
			}
		}
		belt([3]uint32{0x101014, 0x101014, 0})
	case bsbLpSing, bsbLpGuitar, bsbLpBass, bsbLpDrum, bsbLpRhythm, bsbLpChoir, bsbDesBass, bsbDesDrum:
		if front { // the printed emblem
			ink := s.Shirt2
			switch k {
			case bsbLpSing: // a winged skull-ish badge
				b.rect(mid-1, y0+2, 3, 2, ink)
				b.set(mid-2, y0+2, ink)
				b.set(mid+2, y0+2, ink)
				b.set(mid, y0+4, ink)
			case bsbLpGuitar: // a band print
				b.rect(mid-2, y0+2, 5, 1, ink)
				b.rect(mid-1, y0+3, 3, 2, shade(ink, 0.8))
			case bsbLpRhythm: // wings
				b.rect(mid-3, y0+2, 7, 1, ink)
				b.rect(mid-2, y0+3, 5, 1, shade(ink, 0.8))
				b.set(mid, y0+4, ink)
			case bsbLpChoir:
				b.rect(mid-1, y0+2, 3, 3, ink)
				b.set(mid, y0+3, s.Shirt)
			case bsbDesBass: // a big round black drawing on the white tee
				b.rect(mid-2, y0+2, 5, 3, bsbBlack)
				b.set(mid-2, y0+2, s.Shirt)
				b.set(mid+2, y0+2, s.Shirt)
				b.set(mid, y0+3, bsbGrey)
			case bsbDesDrum: // lines of black lettering
				b.rect(mid-2, y0+1, 5, 1, bsbBlack)
				b.rect(mid-2, y0+3, 2, 1, bsbBlack)
				b.rect(mid+1, y0+3, 2, 2, bsbBlack)
			}
		}
		switch k {
		case bsbLpSing, bsbLpRhythm:
			belt([3]uint32{0x16161c, 0xd0d4dc, 0x9a9ea8}) // studded belt
		case bsbLpBass:
			belt([3]uint32{0xe8e8e8, 0xa8acb4, 0})
		case bsbLpGuitar, bsbLpDrum, bsbLpChoir:
			belt([3]uint32{0x2a1e16, 0xc8ccd4, 0})
		}
	case bsbDesSing:
		// a short-sleeved dark green shirt: lighter collar, buttons, a pocket
		if !back {
			b.set(mid-1, y0, shade(s.Shirt, 1.6))
			b.set(mid+1, y0, shade(s.Shirt, 1.6))
			if front {
				b.set(mid, y0, s.Skin)
				for y := y0 + 2; y < y1-1; y += 2 {
					b.set(mid, y, shade(s.Shirt, 0.6))
				}
				b.rect(mid+2, y0+2, 2, 2, shade(s.Shirt, 0.8))
			}
		} else {
			b.rect(x0+1, y0, tw-2, 1, shade(s.Shirt, 1.4))
		}
	case bsbPsSing:
		// black leather jacket open over a navy and white striped tee, a silver chain
		switch {
		case front:
			for y := y0; y < y1-1; y++ {
				c := bsbWhite
				if (y-y0)%2 == 1 {
					c = bsbNavy
				}
				b.rect(mid-2, y, 5, 1, c)
			}
			b.rect(mid-3, y0, 1, y1-y0-1, shade(s.Shirt, 1.8)) // the lapels shine
			b.rect(mid+3, y0, 1, y1-y0-1, shade(s.Shirt, 1.8))
			b.set(mid-2, y0, bsbSilver) // the chain
			b.set(mid-1, y0+1, bsbSilverL)
			b.set(mid, y0+2, bsbSilver)
			b.set(mid+1, y0+1, bsbSilverL)
			b.set(mid+2, y0, bsbSilver)
		case side:
			for y := y0; y < y1-1; y++ {
				c := bsbWhite
				if (y-y0)%2 == 1 {
					c = bsbNavy
				}
				b.set(x1-1, y, c)
			}
			b.set(x1-1, y0+1, bsbSilver)
		case back:
			b.rect(mid, y0+1, 1, y1-y0-2, shade(s.Shirt, 1.5)) // seam
		}
	case bsbPsGuitar:
		// white shirt with black vertical stripes, open at the neck
		for y := y0; y < y1-1; y++ {
			for x := x0; x < x1; x++ {
				if (x-x0)%2 == 1 && b.get(x, y).A != 0 {
					b.set(x, y, bsbBlack)
				}
			}
		}
		if front {
			b.rect(mid-1, y0, 3, 1, s.Skin)
			b.set(mid, y0+1, s.Skin)
		}
	case bsbPsBass:
		// black sleeveless waistcoat: silver buttons and pockets
		if front {
			for y := y0 + 1; y < y1-1; y += 2 {
				b.set(mid, y, bsbSilver)
			}
			b.set(mid-3, y0+2, bsbGrey)
			b.set(mid+3, y0+2, bsbGrey)
		}
		belt([3]uint32{0x101014, 0xc8ccd4, 0})
	case bsbPsDrum:
		belt([3]uint32{0x101014, 0x101014, 0})
	}
}

// bsbArm: tattoos, wristbands and the striped sleeves.
func bsbArm(b *pixBuf, s *CharSpec, vw int, sx, sy, mx, my, ex, ey float64, armW int, dark bool, id uint8) {
	k := bsbKind(s)
	if k == bsbNone {
		return
	}
	b.cur = id
	fl := func(f float64) int { return int(math.Floor(f)) }
	lerp := func(a, c, t float64) float64 { return a + (c-a)*t }
	sh := func(c color.RGBA) color.RGBA {
		if dark {
			return shade(c, 0.8)
		}
		return c
	}
	near := (vw == vFront && id == 4) || (vw == vBack && id == 3) || (vw == vSide && id == 4)
	tattooFore := func() {
		for _, t := range []float64{0.25, 0.6} {
			b.set(fl(lerp(mx, ex, t)), fl(lerp(my, ey, t)), sh(bsbInk))
		}
		if armW > 1 {
			b.set(fl(lerp(mx, ex, 0.45))+1, fl(lerp(my, ey, 0.45)), sh(bsbInk))
		}
	}
	tattooUpper := func() {
		for _, t := range []float64{0.3, 0.7} {
			b.set(fl(lerp(sx, mx, t))+armW-1, fl(lerp(sy, my, t)), sh(bsbInk))
		}
		b.set(fl(lerp(sx, mx, 0.5)), fl(lerp(sy, my, 0.5)), sh(bsbInk))
	}
	band := func(c color.RGBA) { b.rect(fl(ex), fl(ey)-1, armW, 1, sh(c)) }
	switch k {
	case bsbBtSing:
		if id == 3 {
			band(bsbBand)
		}
	case bsbBtGuitar:
		band(bsbWhite)
	case bsbBtBass:
		tattooFore()
	case bsbLpSing:
		tattooFore()
		if near {
			band(bsbBlack)
		}
	case bsbLpGuitar:
		if near {
			tattooFore()
		}
	case bsbLpRhythm:
		band(bsbBlack)
		if !near {
			tattooFore()
		}
	case bsbLpChoir:
		if near {
			band(bsbBlack)
		}
	case bsbDesDrum:
		if near {
			tattooFore()
		}
	case bsbPsBass:
		tattooUpper()
		tattooFore()
	case bsbPsGuitar:
		// black stripes down the white sleeves
		b.line(fl(sx)+armW-1, fl(sy), fl(mx)+armW-1, fl(my), sh(bsbBlack))
		b.line(fl(mx)+armW-1, fl(my), fl(lerp(mx, ex, 0.85))+armW-1, fl(lerp(my, ey, 0.85)), sh(bsbBlack))
	}
}

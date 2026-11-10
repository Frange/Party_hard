package main

import "math"

// ---------------------------------------------------------------------------
// Small own drawing layers that make some famous guests (names I-P) easier to recognise: the gold
// visor of the green space marine, the T visor of the bounty hunter, the floppy caps of the little
// blue goblins and of the green hero, braids, headbands, a turtle shell, a long druid beard...
// genChar calls ciHead at the end of the head and ciBody after the torso. They are parodies drawn
// with a few pixels of our own (party touches, own palette), not copies of any reference.
// ---------------------------------------------------------------------------

const (
	ciNone = iota
	ciChief
	ciMando
	ciBravo
	ci456
	ciKratos
	ciLeon
	ciLink
	ciRoshi
	ciNegan
	ciOnce
	ciDruid
	ciSmurf
	ciPan
	ciWed
	ciClaws
	ciPopeye
	ciNinja
	ciMonk
	ciMarty
)

var ciKinds = map[string]int{
	"jefe_maestro": ciChief, "mandaloriano": ciMando, "johnny_bravo": ciBravo, "jugador_456": ci456,
	"leon_k": ciLeon, "link": ciLink, "maestro_roshi": ciRoshi, "negan": ciNegan, // kratos: ktm_art.go
	"once": ciOnce, "panoramix": ciDruid, "pitufo": ciSmurf, "papa_pitufo": ciSmurf, "pitufina": ciSmurf,
	"peter_pan": ciPan, "miercoles": ciWed, "lobezno": ciClaws, "popeye": ciPopeye, "naruto": ciNinja,
	"krilin": ciMonk, "marty_mcfly": ciMarty,
}

func ciKind(s *CharSpec) int { return ciKinds[s.Name] }

var (
	ciGold   = hx(0xe8b828)
	ciGoldL  = hx(0xfff0a0)
	ciInk    = hx(0x16161c)
	ciWhite  = hx(0xf6f6f2)
	ciRed    = hx(0xc8242a)
	ciBrown  = hx(0x6a4422)
	ciBlueHB = hx(0x2a3a8a)
	ciSteel  = hx(0xc8ccd4)
)

// ciSwing: a one-pixel sway for caps, braids and beards while moving.
func ciSwing(p Pose) int {
	switch p.Anim {
	case AnimWalk, AnimRun, AnimHop:
		return p.F % 2
	}
	return 0
}

// ciHead paints the head details of the character (after hats and the stock face).
func ciHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := ciKind(s)
	if k == ciNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	chin := int(math.Floor(hyp + ry))
	rxi := int(math.Floor(rx))
	sw := ciSwing(p)
	hair := s.Hair
	switch k {
	case ciChief: // a wide gold visor with a shine
		switch vw {
		case vFront:
			b.rect(ihx-2, ey-1, 5, 2, ciGold)
			b.set(ihx-2, ey-1, ciGoldL)
			b.set(ihx-1, ey-1, ciGoldL)
		case vSide:
			b.rect(ihx, ey-1, rxi+1, 2, ciGold)
			b.set(ihx+rxi, ey-1, ciGoldL)
		}
	case ciMando: // a black T visor
		switch vw {
		case vFront:
			b.rect(ihx-2, ey-1, 5, 1, ciInk)
			b.rect(ihx, ey, 1, 2, ciInk)
		case vSide:
			b.rect(ihx+1, ey-1, rxi, 1, ciInk)
			b.rect(ihx+rxi, ey, 1, 2, ciInk)
		}
	case ciBravo: // a tall blond quiff
		hl, hd := shade(hair, 1.2), shade(hair, 0.75)
		tq := maxI(top, 5) // squashed at the top of a jump so it stays inside the sprite
		switch vw {
		case vFront:
			b.rect(ihx-rxi, tq-2, 2*rxi+1, 3, hair)
			b.rect(ihx-rxi, tq-4, 2*rxi+2, 2, hair)
			b.set(ihx, tq-4, hl)
			b.set(ihx+1, tq-3, hl)
			b.rect(ihx-rxi, tq+1, 1, 2, hd)
			b.rect(ihx+rxi, tq+1, 1, 2, hd)
		case vSide:
			b.rect(ihx-rxi, tq-2, 2*rxi+1, 3, hair)
			b.rect(ihx-rxi+1, tq-4, 2*rxi+2, 2, hair)
			b.set(ihx+rxi+1, tq-2, hd)
			b.set(ihx+1, tq-4, hl)
		case vBack:
			b.rect(ihx-rxi, tq-2, 2*rxi+1, ey-tq+3, hair)
			b.rect(ihx-rxi+1, tq-4, 2*rxi-1, 2, hair)
			b.rect(ihx, tq-1, 1, 3, hd)
		}
	case ciKratos: // the red stripe over the head and the eye
		switch vw {
		case vFront:
			b.rect(ihx-1, top, 1, ey-top, ciRed)
			b.rect(ihx-1, ey+1, 1, chin-ey-1, ciRed)
		case vSide:
			b.rect(ihx-1, top, 2, 2, ciRed)
			b.rect(ihx+1, top+1, 1, ey-top-1, ciRed)
		case vBack:
			b.rect(ihx-1, top, 1, chin-top, ciRed)
		}
	case ciLink: // floppy green cap, pointy ears, blond fringe
		cap, capD := s.HatCol, shade(s.HatCol, 0.75)
		switch vw {
		case vFront:
			b.set(ihx-rxi-1, ey-1, s.Skin)
			b.set(ihx+rxi+1, ey-1, s.Skin)
			b.set(ihx-rxi-2, ey-2, s.Skin)
			b.set(ihx+rxi+2, ey-2, s.Skin)
			b.rect(ihx-rxi, top-1, 2*rxi+1, 2, cap)
			b.rect(ihx-rxi+1, top-2, 2*rxi-1, 1, cap)
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, hair)
			b.set(ihx-rxi, top+2, hair)
			b.set(ihx+rxi, top+2, hair)
			b.rect(ihx+rxi+1, top-1+sw, 1, 3, capD) // the tip hangs behind
		case vSide:
			b.set(ihx-1, ey-1, s.Skin)
			b.set(ihx-2, ey-2, s.Skin)
			b.rect(ihx-rxi, top-1, 2*rxi+1, 2, cap)
			b.rect(ihx-rxi+1, top-2, 2*rxi-1, 1, cap)
			b.set(ihx+rxi, top+1, hair)
			b.rect(ihx-rxi-1, top, 2, 2, cap)
			b.rect(ihx-rxi-2-sw, top+2, 2, 2, cap)
			b.set(ihx-rxi-3-sw, top+4, capD)
		case vBack:
			b.rect(ihx-rxi, top-1, 2*rxi+1, 3, cap)
			b.rect(ihx-rxi+1, top-2, 2*rxi-1, 1, cap)
			b.rect(ihx-1+sw, top+2, 3, 3, capD)
			b.rect(ihx+sw, top+5, 1, 2, capD)
		}
	case ciRoshi, ciDruid: // a long white beard (to the belt for the druid)
		beard := hx(0xf4f4f0)
		bd := hx(0xd4d4d8)
		long := 3
		if k == ciDruid {
			long = 6
		}
		switch vw {
		case vFront:
			b.rect(ihx-2, ey+1, 5, chin-ey+1, beard)
			b.rect(ihx-1, chin+1, 3, long, beard)
			b.set(ihx+sw, chin+1+long, bd)
			b.set(ihx, chin, bd)
			b.set(ihx-1, ey+1, bd) // moustache tips
			b.set(ihx+1, ey+1, bd)
		case vSide:
			b.rect(ihx, ey+1, rxi+1, chin-ey+1, beard)
			b.rect(ihx+1, chin+1, 2, long, beard)
			b.set(ihx+1+sw, chin+1+long, bd)
		}
	case ciOnce: // a nosebleed from her powers
		if vw == vFront {
			b.set(ihx-1, ey+1, ciRed)
		} else if vw == vSide {
			b.set(ihx+rxi, ey+2, ciRed)
		}
	case ciSmurf: // the soft cap that flops forward
		cap, capD := s.HatCol, shade(s.HatCol, 0.8)
		switch vw {
		case vFront:
			b.rect(ihx-rxi, top-1, 2*rxi+1, 3, cap)
			b.rect(ihx-rxi+1, top-2, 2*rxi-1, 1, cap)
			b.rect(ihx-1, top-3, 3, 1, cap)
			b.rect(ihx+1, top-3, 2, 1, capD)
			b.set(ihx+3, top-2+sw, capD)
			b.rect(ihx-rxi, top+2, 2*rxi+1, 1, capD)
		case vSide:
			b.rect(ihx-rxi, top-1, 2*rxi+1, 3, cap)
			b.rect(ihx-rxi+1, top-2, 2*rxi, 1, cap)
			b.rect(ihx, top-3, 3, 1, cap)
			b.rect(ihx+rxi-1, top-2+sw, 2, 2, capD)
			b.rect(ihx-rxi, top+2, 2*rxi+1, 1, capD)
		case vBack:
			b.rect(ihx-rxi, top-1, 2*rxi+1, 4, cap)
			b.rect(ihx-rxi+1, top-2, 2*rxi-1, 1, cap)
			b.rect(ihx-1, top-3, 3, 1, capD)
		}
		if s.Female && vw != vFront { // her long blond hair under the cap
			b.rect(ihx-rxi, top+2, 2*rxi+1, 1, hair)
		}
	case ciPan: // a pointy green cap with a red feather
		cap := s.HatCol
		feather := hx(0xe0302a)
		switch vw {
		case vFront, vBack:
			b.rect(ihx-rxi, top-1, 2*rxi+1, 2, cap)
			b.rect(ihx-1, top-2, 3, 1, cap)
			b.set(ihx, top-3, cap)
			b.line(ihx+rxi, top-1, ihx+rxi+2, top-4-sw, feather)
		case vSide:
			b.rect(ihx-rxi, top-1, 2*rxi+1, 2, cap)
			b.rect(ihx-rxi-1, top-2, 3, 1, cap)
			b.set(ihx-rxi-2, top-3, cap)
			b.line(ihx-rxi+1, top-1, ihx-rxi-2, top-5-sw, feather)
		}
	case ciWed: // two long black braids
		hd := shade(hair, 1.6)
		switch vw {
		case vFront:
			for i := 0; i < 6; i++ {
				c := hair
				if i%2 == 1 {
					c = hd
				}
				b.set(ihx-rxi-1, ey+i, c)
				b.set(ihx+rxi+1, ey+i, c)
			}
		case vSide:
			for i := 0; i < 6; i++ {
				c := hair
				if i%2 == 1 {
					c = hd
				}
				b.set(ihx-rxi, ey+1+i, c)
			}
		case vBack:
			for i := 0; i < 6; i++ {
				c := hair
				if i%2 == 1 {
					c = hd
				}
				b.set(ihx-2, chin-1+i, c)
				b.set(ihx+2, chin-1+i, c)
			}
		}
	case ciClaws: // the two pointed tufts
		switch vw {
		case vFront, vBack:
			b.rect(ihx-rxi-1, top-1, 2, 2, hair)
			b.set(ihx-rxi-2, top-2, hair)
			b.rect(ihx+rxi, top-1, 2, 2, hair)
			b.set(ihx+rxi+2, top-2, hair)
		case vSide:
			b.rect(ihx-rxi, top-1, 3, 2, hair)
			b.set(ihx-rxi-1, top-2, hair)
		}
	case ciPopeye: // a corncob pipe
		pipe := hx(0xc89a4a)
		switch vw {
		case vFront:
			b.set(ihx+1, ey+2, ciBrown)
			b.set(ihx+2, ey+2, ciBrown)
			b.rect(ihx+3, ey, 1, 3, pipe)
		case vSide:
			b.set(ihx+rxi, ey+2, ciBrown)
			b.set(ihx+rxi+1, ey+2, ciBrown)
			b.rect(ihx+rxi+2, ey, 1, 3, pipe)
		}
	case ciNinja: // blue headband with a steel plate, whiskers
		switch vw {
		case vFront:
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, ciBlueHB)
			b.rect(ihx-1, top+1, 3, 1, ciSteel)
			b.set(ihx-2, ey+1, shade(s.Skin, 0.6))
			b.set(ihx+2, ey+1, shade(s.Skin, 0.6))
		case vSide:
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, ciBlueHB)
			b.rect(ihx+rxi-1, top+1, 2, 1, ciSteel)
			b.set(ihx-rxi-1, top+2+sw, ciBlueHB) // the knot tails
			b.set(ihx-rxi-2, top+3, ciBlueHB)
		case vBack:
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, ciBlueHB)
			b.rect(ihx-1+sw, top+2, 1, 3, ciBlueHB)
			b.rect(ihx+1+sw, top+2, 1, 2, ciBlueHB)
		}
	case ciMonk: // the dots on the forehead
		dot := hx(0x7a4a2a)
		if vw == vFront {
			for _, dx := range []int{-2, 0, 2} {
				b.set(ihx+dx, top+1, dot)
			}
			b.set(ihx-1, top+2, dot)
			b.set(ihx+1, top+2, dot)
		} else if vw == vSide {
			b.set(ihx+1, top+1, dot)
			b.set(ihx+2, top+2, dot)
		}
	}
}

// ciBody paints the torso details (after the shirt): collars, scarves, belts, patches, a shell.
func ciBody(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := ciKind(s)
	if k == ciNone {
		return
	}
	tw := x1 - x0
	switch k {
	case ci456: // white collar and the number patch
		b.rect(x0, y0, tw, 1, ciWhite)
		if vw == vFront {
			b.rect(x0+1, y0+2, 3, 2, ciWhite)
			b.set(x0+1, y0+3, ciInk)
			b.set(x0+3, y0+3, ciInk)
		} else if vw == vBack {
			b.rect(mid-2, y0+2, 5, 3, ciWhite)
			b.set(mid-1, y0+3, ciInk)
			b.set(mid, y0+3, ciInk)
			b.set(mid+1, y0+3, ciInk)
		}
	case ciKratos: // the red tattoo from the shoulder down the chest
		switch vw {
		case vFront:
			b.line(x0+1, y0, x0+1, y1-3, ciRed)
			b.line(x0+1, y0+1, mid-1, y0+3, ciRed)
		case vBack:
			b.line(x1-2, y0, x1-2, y1-3, ciRed)
		case vSide:
			b.rect(mid, y0, 1, 4, ciRed)
		}
	case ciLeon: // a sheepskin collar
		fur := hx(0xe8dcbc)
		b.rect(x0, y0, tw, 1, fur)
		if vw == vFront {
			b.set(x0, y0+1, fur)
			b.set(x1-1, y0+1, fur)
			b.set(mid, y0+1, hx(0x2a2a30))
		}
	case ciLink: // a brown belt with a buckle
		b.rect(x0, y1-2, tw, 1, ciBrown)
		if vw == vFront {
			b.set(mid, y1-2, ciGold)
		}
		if vw == vBack { // the shield on his back
			b.rect(mid-2, y0+1, 5, 5, hx(0x2a4ab0))
			b.rect(mid-1, y0+2, 3, 2, hx(0xd8d8e0))
		}
	case ciRoshi: // a turtle shell on the back
		shell, rim := hx(0x3a8a3a), hx(0xe8c870)
		switch vw {
		case vBack:
			b.rect(x0-1, y0, tw+2, y1-y0, rim)
			b.rect(x0, y0+1, tw, y1-y0-2, shell)
			b.set(mid, y0+2, shade(shell, 0.7))
			b.set(mid-1, y0+4, shade(shell, 0.7))
			b.set(mid+1, y0+4, shade(shell, 0.7))
		case vSide:
			b.rect(x0-2, y0+1, 2, y1-y0-2, shell)
			b.set(x0-2, y0+1, rim)
		}
	case ciNegan: // a red scarf
		b.rect(x0, y0, tw, 1, ciRed)
		if vw == vFront {
			b.rect(mid-1, y0+1, 1, 3+ciSwing(p), ciRed)
		}
	case ciDruid: // golden sickle at the belt
		if vw != vBack {
			b.set(x1-1, y1-3, ciGold)
			b.set(x1, y1-4, ciGold)
			b.set(x1, y1-2, ciBrown)
		}
	case ciWed: // the white collar
		if vw == vFront {
			b.rect(mid-2, y0, 2, 1, ciWhite)
			b.rect(mid+1, y0, 2, 1, ciWhite)
			b.set(mid-2, y0+1, ciWhite)
			b.set(mid+2, y0+1, ciWhite)
		} else if vw == vSide {
			b.rect(mid, y0, 2, 1, ciWhite)
		}
	case ciPopeye: // red collar and gold buttons
		if vw == vFront {
			b.rect(x0, y0, tw, 1, ciRed)
			b.set(mid, y0+2, ciGold)
			b.set(mid, y0+4, ciGold)
		} else {
			b.rect(x0, y0, tw, 1, ciRed)
		}
	case ciMonk: // the blue undershirt and belt of the training suit
		b.rect(x0, y1-2, tw, 1, ciBlueHB)
		if vw == vFront {
			b.rect(mid-1, y0, 3, 1, ciBlueHB)
			b.set(mid, y0+1, ciBlueHB)
		}
	case ciMarty: // a red puffer vest over the denim jacket
		vest, seam := hx(0xd8302a), hx(0xa01e1c)
		for y := y0; y < y1; y++ {
			c := vest
			if (y-y0)%2 == 1 {
				c = seam
			}
			b.rect(x0, y, tw, 1, c)
		}
		if vw == vFront {
			b.rect(mid, y0, 1, y1-y0, hx(0xe8e0d0)) // open zip, the shirt under it
		}
	case ciMando: // a bandolier
		if vw != vSide {
			b.line(x0, y0+1, x1-1, y1-2, ciBrown)
		}
	}
}

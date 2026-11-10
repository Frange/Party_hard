package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// EL DOCTOR PICO (THE BEAK DOCTOR): a party killer (one of the ASESINOS) drawn very close to the user's
// plague-doctor pixel-art reference: a black wide-brimmed hat, a long bone-coloured bird beak mask with a
// round dark red lens, a black hood, a long blue-black robe down to the feet with a small cross on the
// chest, a brown leather belt with a buckle, wide sleeves, dark brown gloves and boots, a brown satchel on
// the hip and a long walking staff.
//
// He follows every rule of the other killers (killerreg.go, goalkillers.go): he only comes when the party
// plan sends a killer (never two parties in a row, never in STEALTH), he is one of the cards of the ASESINOS
// tab (unlocked by killing him) and counts in "Asesinos vencidos". He never comes to the secret island
// (plagueFits). What is his own:
//   - he walks slowly and calmly (plagueSpeedK) and takes plagueHP stabs;
//   - a guest he reaches is not stabbed: he touches him with his staff and the guest falls SICK (coughs,
//     retches, staggers for plagueSickT seconds) and then collapses dead (the doctor's kill, not the
//     player's). Meanwhile the doctor stands and watches; the player and the guards get the usual blow;
//   - he says his line when he arrives.
// -killer -killeras doctor_pico forces him (terminator_killer.go).
// genChar calls plagueHead / plagueTorso / plagueArm / plagueLeg / plagueHeld (chars.go).
// ---------------------------------------------------------------------------

const (
	plagueSlug   = "doctor_pico"
	plagueHP     = 4
	plagueSpeedK = 0.75 // of the usual killer walk
	plagueSickT  = 3.2  // s a touched guest is sick before he falls
)

// plagueLine: his catchphrase (ES; EN in lang_data via init).
const plagueLine = "Respire hondo, por favor."

type plagueState struct {
	sick map[*Guest]float64 // touched guests -> seconds left
}

var (
	pdRobe    = hx(0x1e2232)
	pdRobeL   = hx(0x2e3448)
	pdRobeD   = hx(0x12141e)
	pdHood    = hx(0x18181e)
	pdHat     = hx(0x101014)
	pdHatL    = hx(0x2a2a32)
	pdBeak    = hx(0xe6d8b0)
	pdBeakL   = hx(0xfff2d4)
	pdBeakD   = hx(0xb4a27a)
	pdLens    = hx(0x8a2418)
	pdLensL   = hx(0xd04a30)
	pdRim     = hx(0x2a1a12)
	pdBelt    = hx(0x6a4024)
	pdBuckle  = hx(0xd8b048)
	pdCross   = hx(0xd8cfb8)
	pdGlove   = hx(0x3e2818)
	pdGloveL  = hx(0x5a3c24)
	pdBag     = hx(0x7a5030)
	pdBagL    = hx(0x9a6a40)
	pdStaff   = hx(0x5a3a20)
	pdStaffL  = hx(0x8a6038)
	pdSickCol = hx(0x8ab04a)
)

func plagueSpec() CharSpec {
	e := def(plagueSlug, skinc(0xe6d8b0), hairs(HairBald), tops(TopRobe, 0x1e2232, 0x1e2232), pantsc(0x1e2232),
		shoesc(0x3e2818), bodyc(BodyNormal), stat(-1), weaponc(HoldNone))
	sp := e.F(rand.New(rand.NewSource(1347))) // own source: builtinChars' shared stream is not moved
	sp.Hat, sp.Glasses, sp.Beard, sp.Carry, sp.Weapon = HatNone, GlassNone, BeardNone, HoldNone, HoldNone
	sp.Cape.A, sp.Mask.A = 0, 0
	return sp
}

func init() {
	killerPresets = append(killerPresets, presetEntry{plagueSlug, func(*rand.Rand) CharSpec { return plagueSpec() }})
	for k, v := range map[string]string{
		plagueLine:                      "Breathe deeply, please.",
		"Bastón":                        "Staff",
		"Cof, cof...":                   "Cough, cough...",
		"El Doctor Pico te ha recetado": "The Beak Doctor wrote you off",
		"¡El Doctor Pico! No dejes que te toque.": "The Beak Doctor! Don't let him touch you.",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

func isPlague(s *Slasher) bool { return s != nil && s.plague != nil }

func isPlagueSpec(s *CharSpec) bool { return s != nil && s.Name == plagueSlug }

// plagueRole: the line under his name on the ASESINOS card ("" for the others).
func plagueRole(slug string) string {
	if slug == plagueSlug {
		return "Bastón"
	}
	return ""
}

// plagueFits: may this killer come to this party? The doctor never comes to the island.
func (g *Game) plagueFits(name string) bool {
	return name != plagueSlug || g.levelIdx != specialJurassic
}

// plagueSpawn: his own state and toughness.
func (g *Game) plagueSpawn(s *Slasher) {
	if s == nil || s.name != plagueSlug {
		return
	}
	s.plague = &plagueState{sick: map[*Guest]float64{}}
	s.hp = plagueHP
	s.chainsaw = false
}

// plagueArrive: his line when he steps out of the car.
func (g *Game) plagueArrive(s *Slasher) {
	if !isPlague(s) {
		return
	}
	s.icon, s.iconT = plagueLine, 2.5
	g.say("¡El Doctor Pico! No dejes que te toque.")
}

// plagueSpeed: he walks slowly and calmly.
func plagueSpeed(s *Slasher, speed float64) float64 {
	if !isPlague(s) {
		return speed
	}
	return speed * plagueSpeedK
}

// plagueTick runs every tick while he is at the party: the sick guests cough and stagger, then fall.
func (g *Game) plagueTick(s *Slasher, dt float64) {
	if !isPlague(s) {
		return
	}
	for gu, left := range s.plague.sick {
		if gu.out() {
			delete(s.plague.sick, gu)
			continue
		}
		left -= dt
		if left <= 0 {
			delete(s.plague.sick, gu)
			g.killGuest(gu, false)
			g.sayNews("kill", map[string]string{"actor": slasherActorName(s), "victim": guestLabel(gu)})
			g.slasherWitness(s, gu.pos)
			continue
		}
		s.plague.sick[gu] = left
		// sick: he stops, retches and staggers on the spot
		gu.state, gu.path, gu.partner, gu.timer = gVomit, nil, nil, left
		gu.moving = false
		if gu.iconT <= 0.2 {
			gu.icon, gu.iconT = "Cof, cof...", 1
		}
		if math.Mod(left, 0.5) < dt {
			g.burst(Vec{gu.pos.X, gu.pos.Y - 18}, 2, 14, pdSickCol, 0.5, 2, -30)
		}
	}
}

// plagueTouch runs instead of the usual strike: a guest in reach is touched with the staff and falls sick
// (the doctor then stands and watches him). It reports true when the tick is used (no stab).
func (g *Game) plagueTouch(s *Slasher) bool {
	if !isPlague(s) || s.tgtG == nil {
		return false
	}
	v := s.tgtG
	if _, sick := s.plague.sick[v]; sick {
		s.path = nil // he waits beside him, calmly
		s.dir = faceTo(s.pos, v.pos)
		return true
	}
	if v.out() || dist(s.pos, v.pos) >= 14 {
		return false
	}
	s.stabT, s.pause = 0.35, 0.8
	s.plague.sick[v] = plagueSickT
	g.snd.Play("glug")
	g.burst(Vec{v.pos.X, v.pos.Y - 18}, 6, 20, pdSickCol, 0.7, 2, -30)
	s.repath = 0
	return true
}

// ---------------------------------------------------------------------------
// The art (genChar). The base spec is a plain robe (TopRobe) with brown boots; these layers add the rest.
// ---------------------------------------------------------------------------

func pdDark(c color.RGBA, dark bool) color.RGBA {
	if dark {
		return shade(c, 0.78)
	}
	return c
}

// plagueHead: hood, wide-brimmed hat and the beak mask with its round lenses.
func plagueHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	if !isPlagueSpec(s) {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	chin := int(math.Floor(hyp + ry))
	rxi := int(math.Floor(rx))
	row := func(y, x0, x1 int, c color.RGBA) { b.rect(ihx+x0, y, x1-x0+1, 1, c) }
	lens := pdLens
	if p.Anim == AnimStab {
		lens = pdLensL
	}
	// the hood covers the whole head and falls over the shoulders
	for y := top; y <= chin+1; y++ {
		w := rxi
		if y > chin-1 {
			w = rxi + 1
		}
		row(y, -w, w, pdHood)
	}
	// the hat: a flat crown and a wide brim
	brim := top + 1
	switch vw {
	case vFront, vBack:
		row(top-3, -2, 2, pdHat)
		row(top-2, -3, 3, pdHat)
		row(top-1, -3, 3, pdHat)
		row(top-1, -3, -2, pdHatL)
		row(top, -3, 3, pdRobeL) // band
		row(brim, -rxi-3, rxi+3, pdHat)
		row(brim, -rxi-3, -rxi-2, pdHatL)
	case vSide:
		row(top-3, -2, 1, pdHat)
		row(top-2, -3, 2, pdHat)
		row(top-1, -3, 2, pdHat)
		row(top-1, -3, -3, pdHatL)
		row(top, -3, 2, pdRobeL)
		row(brim, -rxi-3, rxi+3, pdHat)
		row(brim, rxi+2, rxi+3, pdHatL)
	}
	switch vw {
	case vFront:
		// the mask: a bone face under the brim, two round lenses and the beak coming down towards us
		row(brim+1, -2, 2, pdBeakD)
		row(ey-1, -2, 2, pdBeak)
		row(ey, -2, 2, pdBeak)
		b.set(ihx-2, ey-1, pdRim)
		b.set(ihx+2, ey-1, pdRim)
		b.set(ihx-2, ey, lens)
		b.set(ihx+2, ey, lens)
		b.set(ihx-1, ey-1, pdBeakD)
		b.set(ihx+1, ey-1, pdBeakD)
		// the beak: wide at the top, pointing down to the chest
		row(ey+1, -1, 1, pdBeak)
		row(ey+2, -1, 1, pdBeak)
		b.set(ihx-1, ey+1, pdBeakL)
		b.set(ihx+1, ey+2, pdBeakD)
		row(ey+3, 0, 0, pdBeak)
		b.set(ihx, ey+4, pdBeakD)
		b.set(ihx, ey+5, pdBeakD)
	case vSide:
		// face to +x: bone face, a big round lens, the long curved beak far forward
		b.rect(ihx, ey-2, rxi+1, 4, pdBeak)
		b.set(ihx, ey-2, pdBeakD)
		b.rect(ihx+1, ey-1, 2, 2, pdRim)
		b.set(ihx+1, ey-1, lens)
		b.set(ihx+2, ey, lens)
		b.set(ihx+2, ey-1, pdLensL)
		// beak: three steps forward and down, light top, shadowed underside
		bx := ihx + rxi + 1
		b.rect(bx, ey, 3, 2, pdBeak)
		b.rect(bx, ey, 3, 1, pdBeakL)
		b.rect(bx+3, ey+1, 2, 2, pdBeak)
		b.set(bx+3, ey+1, pdBeakL)
		b.set(bx+5, ey+2, pdBeak)
		b.set(bx+5, ey+3, pdBeakD)
		b.rect(bx, ey+2, 3, 1, pdBeakD)
		b.set(bx+4, ey+3, pdBeakD)
		b.set(ihx+rxi, ey+1, pdBeakD) // strap
		b.set(ihx-1, ey-1, pdBelt)
	case vBack:
		// the hood and the knot of the mask strap
		row(ey-1, -rxi, rxi, pdHood)
		row(ey, -1, 1, pdBelt)
		b.set(ihx-rxi, ey, pdBelt)
		b.set(ihx+rxi, ey, pdBelt)
		b.set(ihx-rxi-1, ey+1, pdBeakD) // the tip of the beak shows on the sides
	}
}

// plagueTorso: the robe with its light fold, the small cross, the belt with its buckle and the satchel.
func plagueTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	if !isPlagueSpec(s) {
		return
	}
	b.cur = 2
	tw := x1 - x0
	// the hood's cape over the shoulders
	b.rect(x0-1, y0, tw+2, 2, pdHood)
	b.rect(x0, y0+2, tw, 1, pdHood)
	belt := y1 - 2
	if belt < y0+5 {
		belt = y0 + 5
	}
	switch vw {
	case vFront:
		b.rect(x0+1, y0+3, 1, belt-y0-3, pdRobeL) // fold
		// small cross on the chest
		b.rect(mid, y0+3, 1, 3, pdCross)
		b.rect(mid-1, y0+4, 3, 1, pdCross)
		b.rect(x0-1, belt, tw+2, 1, pdBelt)
		b.set(mid, belt, pdBuckle)
		b.set(mid+1, belt, pdBuckle)
		// satchel on his left hip
		b.rect(x1-1, belt+1, 3, 3, pdBag)
		b.rect(x1-1, belt+1, 3, 1, pdBagL)
		// the robe goes down to the feet: a dark line in the middle
		b.rect(mid, belt+1, 1, 13, pdRobeD)
	case vSide:
		b.rect(x0-1, belt, tw+2, 1, pdBelt)
		b.set(x1, belt, pdBuckle)
		b.rect(x0-2, belt+1, 3, 4, pdBag) // the satchel hangs at his back hip
		b.rect(x0-2, belt+1, 3, 1, pdBagL)
		b.set(x0-1, belt-1, pdBelt) // strap
		b.rect(x1-1, y0+3, 1, belt-y0-3, pdRobeL)
	case vBack:
		b.rect(x0-1, belt, tw+2, 1, pdBelt)
		b.rect(mid, y0+3, 1, belt-y0-3, pdRobeD) // seam
		b.rect(x0-2, belt+1, 2, 3, pdBag)
	}
}

// plagueArm: wide sleeves that flare at the wrist and dark brown gloves.
func plagueArm(b *pixBuf, s *CharSpec, vw int, sx, sy, mx, my, ex, ey float64, armW int, dark bool, id uint8) {
	if !isPlagueSpec(s) {
		return
	}
	b.cur = id
	thickLine(b, sx, sy, mx, my, armW+1, pdDark(pdRobe, dark))
	thickLine(b, mx, my, ex, ey-1, armW+1, pdDark(pdRobe, dark))
	b.rect(pdFl(lerp(mx, ex, 0.8))-1, pdFl(lerp(my, ey, 0.8)), armW+2, 1, pdDark(pdRobeL, dark)) // the cuff flares
	b.rect(pdFl(ex), pdFl(ey), armW, 2, pdDark(pdGlove, dark))
	b.set(pdFl(ex), pdFl(ey), pdDark(pdGloveL, dark))
}

// plagueLeg: dark brown boots under the robe.
func plagueLeg(b *pixBuf, s *CharSpec, vw int, hxp, hipY, mx, my, ex, ey float64, sxx, syy, legW int, dark bool) {
	if !isPlagueSpec(s) {
		return
	}
	if vw == vSide {
		b.rect(sxx-1, syy-1, legW+2, 1, pdDark(pdGlove, dark))
		b.rect(sxx-1, syy+1, legW+2, 1, pdDark(pdRim, dark))
	} else {
		b.rect(sxx, syy-1, legW, 1, pdDark(pdGlove, dark))
		b.set(sxx, syy, pdDark(pdGloveL, dark))
	}
}

// plagueHeld: his long walking staff (instead of a knife or an empty hand). True when it painted.
func plagueHeld(b *pixBuf, s *CharSpec, kind, vw int, hx0, hy0 float64) bool {
	if !isPlagueSpec(s) || (kind != HoldNone && kind != HoldKnife) {
		return false
	}
	x := pdFl(hx0)
	y := pdFl(hy0)
	top, bot := y-12, y+9
	if top < 1 {
		top = 1
	}
	if bot > charH-1 {
		bot = charH - 1
	}
	if x < 0 {
		x = 0
	}
	if x > charW-1 {
		x = charW - 1
	}
	b.rect(x, top, 1, bot-top+1, pdStaff)
	b.rect(x, top, 1, 2, pdStaffL)
	b.set(x, y-1, pdStaffL)
	return true
}

func pdFl(f float64) int { return int(math.Floor(f)) }

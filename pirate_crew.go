package main

import (
	"math/rand"
)

// ---------------------------------------------------------------------------
// The crew of the BARCO PIRATA (v1.4.0, tanda H): almost every guest comes as a pirate, many with a peg leg, a hook,
// an eye patch or a parrot on the shoulder, and some dressed like the crews of a famous pirate cartoon and of a famous
// pirate adventure game (only the clothes: no names). The security guards of the ship are pirates too.
// The bits are drawn by genChar (chars.go) through pirPatch / pirArm / pirPeg; CharSpec.pir carries them (unexported:
// it never goes to a file).
// ---------------------------------------------------------------------------

const (
	pirBitPeg    uint8 = 1 << iota // the right leg is a wooden peg
	pirBitHook                     // the right hand is an iron hook
	pirBitParrot                   // a parrot on the left shoulder
)

// pirPeg: is this leg (sign > 0 = the right one) a peg?
func pirPeg(s *CharSpec, sign float64) bool { return s.pir&pirBitPeg != 0 && sign > 0 }

// pirPatch: the eye patch over one eye with its strap round the head (GlassPatch).
func pirPatch(b *pixBuf, vw int, hxp, rx float64, ihx, ey int) {
	ink := hx(0x0c0c10)
	b.rect(int(hxp-rx), ey-2, int(2*rx), 1, ink) // the strap
	if vw == vSide {
		b.rect(ihx+1, ey-1, 2, 3, ink)
		return
	}
	b.rect(ihx-2, ey-1, 2, 3, ink)
}

// pirArm: the hook instead of the right hand, the parrot sitting on the left shoulder.
func pirArm(b *pixBuf, s *CharSpec, vw int, sx, sy, ex, ey float64, armW int, sign float64) {
	if s.pir == 0 {
		return
	}
	if s.pir&pirBitHook != 0 && sign > 0 {
		iron, ironL := hx(0x8a8e98), hx(0xd0d4dc)
		x, y := int(ex), int(ey)
		b.rect(x, y, armW, 2, hx(0x3a2a20)) // the cuff
		b.rect(x+armW/2, y+2, 1, 2, iron)
		b.set(x+armW/2+1, y+4, iron)
		b.set(x+armW/2+2, y+3, ironL)
	}
	if s.pir&pirBitParrot != 0 && sign < 0 && vw != vBack {
		x, y := int(sx)-1, int(sy)-4
		green, red, yel := hx(0x2aa84a), hx(0xd83a2a), hx(0xf0d030)
		b.rect(x, y, 3, 4, green)
		b.rect(x, y-1, 2, 2, red) // the head
		b.set(x-1, y, yel)        // the beak
		b.set(x+2, y+3, hx(0x2a5ad0))
		b.rect(x+1, y+4, 1, 2, red) // the tail
	}
}

// pirateCrewSpec: one pirate of the crowd (venuecrowd.go "pirate"): the classic outfits plus the parodies, and the
// pirate bits rolled on top. The rolls use the crowd's own dice.
func pirateCrewSpec(s CharSpec, r *rand.Rand) CharSpec {
	switch r.Intn(9) {
	case 0: // the deckhand: a striped shirt and a bandana (a red or black cap pulled tight)
		s.Top, s.Shirt, s.Shirt2, s.Pattern = TopTee, hx(0xf0ece0), hx(pick(r, []uint32{0xb83a30, 0x1a2a5a, 0x2a2a30})), 2
		s.Hat, s.HatCol = HatBeanie, hx(pick(r, []uint32{0xb83a30, 0x2a2a30}))
		s.Bottom, s.Pants = BotShorts, hx(pick(r, []uint32{0x3a3a52, 0x5a4a3a}))
	case 1: // the captain: tricorn, long coat, beard
		s.Hat, s.HatCol = HatPirate, hx(0x16141a)
		s.Top, s.Shirt, s.Shirt2 = TopLong, hx(pick(r, []uint32{0x7a2a2a, 0x2a2a4a, 0x3a4a2a})), hx(0xe8e0c8)
		s.Bottom, s.Pants = BotPants, hx(0x2a2018)
		if !s.Female {
			s.Beard = []int{BeardFull, BeardGoatee, BeardMustache}[r.Intn(3)]
		}
	case 2: // the buccaneer: a vest over a white shirt, a tricorn of another colour, a mug of grog
		s.Top, s.Shirt, s.Shirt2 = TopVest, hx(0xf0ece0), hx(pick(r, []uint32{0x5a3a26, 0x2a2a30, 0x6a2a3a}))
		s.Hat, s.HatCol = HatPirate, hx(pick(r, []uint32{0x5a3a26, 0x3a2a1a}))
		s.Carry = HoldBeer
	case 3: // the cartoon captain: a straw hat, an open red vest, blue shorts and sandals
		s.Hat, s.HatCol = HatCowboy, hx(0xe8c870)
		s.Top, s.Shirt, s.Shirt2 = TopVest, s.Skin, hx(0xc8282a)
		s.Bottom, s.Pants = BotShorts, hx(0x2a4a9a)
		s.Hair, s.Hair2 = hx(0x14100c), HairMessy
	case 4: // the cartoon swordsman: green hair, white shirt, dark green sash and trousers, a sword
		s.Hair, s.Hair2, s.Hat = hx(0x3a9a4a), HairSpiky, HatNone
		s.Top, s.Shirt, s.Shirt2 = TopTee, hx(0xf0f0ec), hx(0xf0f0ec)
		s.Bottom, s.Pants = BotPants, hx(0x1e3a22)
		s.Carry, s.CarryCol = HoldSaber, hx(0xb8bcc8) // a steel cutlass, not a glowing blade
	case 5: // the would-be mighty pirate of the game: a blond ponytail, a loose white shirt, blue trousers
		s.Hair, s.Hair2, s.Hat = hx(0xe8c060), HairPony, HatNone
		s.Top, s.Shirt, s.Shirt2 = TopLong, hx(0xf4f0e4), hx(0xf4f0e4)
		s.Bottom, s.Pants = BotPants, hx(0x2a3a6a)
		s.Carry, s.CarryCol = HoldSaber, hx(0xb8bcc8) // a steel cutlass, not a glowing blade
	case 6: // the ghost pirate captain of the game: pale green skin, a black beard, a big coat and a tricorn
		s.Skin = hx(0x9ad0a8)
		s.Hat, s.HatCol = HatPirate, hx(0x16141a)
		s.Beard, s.Hair = BeardFull, hx(0x14100c)
		s.Top, s.Shirt, s.Shirt2 = TopLong, hx(0x1a2a5a), hx(0xe8e0c8)
		s.Bottom, s.Pants = BotPants, hx(0x1a1a22)
	case 7: // the cook with a wooden leg
		s.Hat, s.HatCol = HatBeanie, hx(0x2a2a30)
		s.Top, s.Shirt, s.Shirt2 = TopTee, hx(0xf0ece0), hx(0xf0ece0)
		s.Bottom, s.Pants = BotPants, hx(0x4a3a2a)
		s.pir |= pirBitPeg
	default: // a guest in a plain sailor's shirt with a drink
		s.Top, s.Shirt, s.Shirt2, s.Pattern = TopLong, hx(0xf4f4f0), hx(0x1a2a5a), 2
		s.Carry = HoldCup
	}
	s.Shoes = hx(pick(r, []uint32{0x2a1a10, 0x14141a}))
	if r.Intn(4) == 0 {
		s.pir |= pirBitPeg
	}
	if r.Intn(4) == 0 && s.Carry == HoldNone {
		s.pir |= pirBitHook
	}
	if r.Intn(5) == 0 {
		s.pir |= pirBitParrot
	}
	if r.Intn(3) == 0 {
		s.Glasses = GlassPatch
	}
	return s
}

// pirateGuardSpec: a security guard of the ship: a black tricorn, a red sash coat, an eye patch.
func pirateGuardSpec(r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	s.Hat, s.HatCol = HatPirate, hx(0x101014)
	s.Top, s.Shirt, s.Shirt2 = TopVest, hx(0x14141a), hx(0xb82a2a)
	s.Bottom, s.Pants, s.Shoes = BotPants, hx(0x1a1a22), hx(0x14141a)
	s.Glasses, s.Beard = GlassPatch, []int{BeardFull, BeardStubble, BeardMustache}[r.Intn(3)]
	s.Carry, s.Weapon, s.Ride = HoldNone, HoldNone, 0
	s.Body = BodyBig
	if r.Intn(3) == 0 {
		s.pir |= pirBitHook
	}
	return s
}

// pirateGuards dresses the ship's security as pirates (called once, when the party starts: venueacts.go).
func (g *Game) pirateGuards() {
	if g.lvl == nil || !pirateBay(g.lvl.Def) {
		return
	}
	r := crowdDice()
	var looks []*CharSet
	for i := 0; i < 3; i++ {
		sp := pirateGuardSpec(r)
		looks = append(looks, newCharSet(&sp))
	}
	for i, gd := range g.guards {
		if gd.special || gd.vip != 0 {
			continue
		}
		gd.cs = looks[i%len(looks)]
	}
}

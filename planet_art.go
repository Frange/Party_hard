package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Captain Planet and his five young friends: our own minimal pixel-art take, flat colours, as
// overlay hooks that genChar calls (torso, legs, hands). Visual only: hitbox, speed and stats
// do not change.
//   - "capitan_planeta" (existing slug, same name, body and stature): light blue skin, a green
//     mohawk, a red suit with bare blue arms, a black belt, a yellow emblem on the chest, red
//     briefs over blue legs and red boots.
//   - five new famous guests with our own generic names (group "Planetarios"), each with a
//     power ring on the right hand (one dot of the colour of his element):
//       planetario_tierra:  black hair, brown vest with cream trim, teal T-shirt, red-brown shorts.
//       planetario_fuego:   black hair, purple jacket over a pink T-shirt, brown shorts.
//       planetario_viento:  dark skin, red T-shirt with a green square, dark green shorts.
//       planetario_agua:    red hair, brown vest, blue-green T-shirt, dark blue trousers.
//       planetario_corazon: blonde ponytail, olive vest, blue T-shirt, brown shorts, yellow
//                           socks and green shoes.
// Every colour is set, so they always draw the same pictures (exported to special_people).
// ---------------------------------------------------------------------------

const (
	plNone = iota
	plCaptain
	plEarth
	plFire
	plWind
	plWater
	plHeart
)

// planetSlugs: the five friends, in the order of the group.
var planetSlugs = []string{"planetario_tierra", "planetario_fuego", "planetario_viento", "planetario_agua", "planetario_corazon"}

func planetKind(s *CharSpec) int {
	switch s.Name {
	case "capitan_planeta":
		return plCaptain
	case "planetario_tierra":
		return plEarth
	case "planetario_fuego":
		return plFire
	case "planetario_viento":
		return plWind
	case "planetario_agua":
		return plWater
	case "planetario_corazon":
		return plHeart
	}
	return plNone
}

var (
	plBlack   = hx(0x18181e)
	plEmblem  = hx(0xf4d020)
	plEmblemD = hx(0xc89a10)
	plRedSuit = hx(0xd02828)
	plRedL    = hx(0xf04a3a)
	plCream   = hx(0xeadcb4)
	plVestE   = hx(0x8a5a30) // Earth: brown vest
	plVestW   = hx(0x6e4424) // Water: darker brown vest
	plVestH   = hx(0x7a7a2e) // Heart: olive vest
	plJacket  = hx(0x5a3a8a) // Fire: purple jacket
	plGreenSq = hx(0x3aa040) // Wind: the square on the T-shirt
	plSock    = hx(0xf0d030)
)

// planetRing: the colour of the power ring of each friend.
func planetRing(k int) (c [2]uint32) {
	switch k {
	case plEarth:
		return [2]uint32{0x3ac040, 0x1e7a24}
	case plFire:
		return [2]uint32{0xff5a1a, 0xb02a0a}
	case plWind:
		return [2]uint32{0xe8f6ff, 0x9ab8d0}
	case plWater:
		return [2]uint32{0x2a8aff, 0x1a50b0}
	case plHeart:
		return [2]uint32{0xff5aa0, 0xb02a68}
	}
	return
}

// planetSpec: the full look of one character of the group (every random colour overwritten).
func planetSpec(r *rand.Rand, slug string) CharSpec {
	s := mk(r, slug)
	s.Mask.A = 0
	s.HatCol = hx(0x000000)
	s.Accent = hx(0x18181e)
	s.Body, s.Stature = BodyNormal, 1
	switch slug {
	case "planetario_tierra":
		s.Skin, s.Hair, s.Hair2 = hx(0xe2b48a), hx(0x16121a), HairShort
		s.Top, s.Shirt, s.Shirt2 = TopTee, hx(0x2a9a8a), hx(0x2a9a8a)
		s.Bottom, s.Pants, s.Shoes = BotShorts, hx(0x8a3424), hx(0x4a3020)
	case "planetario_fuego":
		s.Skin, s.Hair, s.Hair2 = hx(0xe8bc94), hx(0x16121a), HairSide
		s.Top, s.Shirt, s.Shirt2 = TopLong, plJacket, hx(0xec7aac)
		s.Bottom, s.Pants, s.Shoes = BotShorts, hx(0x6a4a2a), hx(0x2a2a34)
	case "planetario_viento":
		s.Skin, s.Hair, s.Hair2 = hx(0x6a4228), hx(0x141016), HairShort
		s.Top, s.Shirt, s.Shirt2 = TopTee, hx(0xc83030), hx(0xc83030)
		s.Bottom, s.Pants, s.Shoes = BotShorts, hx(0x2a5a2a), hx(0xf0f0f0)
	case "planetario_agua":
		s.Skin, s.Hair, s.Hair2 = hx(0xf2c8a6), hx(0xd0602a), HairMessy
		s.Top, s.Shirt, s.Shirt2 = TopTee, hx(0x3a9ab0), hx(0x3a9ab0)
		s.Bottom, s.Pants, s.Shoes = BotPants, hx(0x23306a), hx(0x3a2a20)
		s.Stature = 0
	case "planetario_corazon":
		s.Skin, s.Hair, s.Hair2 = hx(0xf2cca8), hx(0xf0d060), HairPony
		s.Top, s.Shirt, s.Shirt2 = TopTee, hx(0x3a6ad0), hx(0x3a6ad0)
		s.Bottom, s.Pants, s.Shoes = BotShorts, hx(0x7a5434), hx(0x3a9a3a)
		s.Female = true
	}
	return s
}

func init() {
	for _, slug := range planetSlugs {
		slug := slug
		famousPresets = append(famousPresets, func(r *rand.Rand) CharSpec { return planetSpec(r, slug) })
	}
	for slug, v := range map[string][3]string{
		"planetario_tierra":  {"Planetario de la Tierra", "Earth Planetteer", "Tierra"},
		"planetario_fuego":   {"Planetario del Fuego", "Fire Planetteer", "Fuego"},
		"planetario_viento":  {"Planetario del Viento", "Wind Planetteer", "Viento"},
		"planetario_agua":    {"Planetario del Agua", "Water Planetteer", "Agua"},
		"planetario_corazon": {"Planetaria del Corazón", "Heart Planetteer", "Corazón"},
	} {
		displayNameOverrides[slug] = v[0]
		albumShortName[slug] = v[2]
		if _, ok := langDataEN[v[0]]; !ok {
			langDataEN[v[0]] = v[1]
		}
	}
	for k, v := range map[string]string{"Tierra": "Earth", "Fuego": "Fire", "Viento": "Wind", "Agua": "Water", "Corazón": "Heart"} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for slug, d := range map[string][2]string{
		"planetario_tierra":  {"Planta un árbol en cada maceta", "Plants a tree in every flowerpot"},
		"planetario_fuego":   {"Enciende la pista con un anillo", "Lights up the floor with a ring"},
		"planetario_viento":  {"Sopla las velas desde la puerta", "Blows out the candles from the door"},
		"planetario_agua":    {"Rellena los vasos sin grifo", "Refills the glasses with no tap"},
		"planetario_corazon": {"Hace que el gato baile con todos", "Gets the cat dancing with everyone"},
	} {
		charDescs[slug] = d
		if _, ok := langDataEN[d[0]]; !ok {
			langDataEN[d[0]] = d[1]
		}
	}
	starGroups = append(starGroups, append([]string{"capitan_planeta"}, planetSlugs...))
}

// planetVest: an open vest (or jacket) of colour c over the T-shirt; trim is its edge colour.
func planetVest(b *pixBuf, vw, x0, x1, y0, y1 int, c, trim color.RGBA) {
	tw := x1 - x0
	w := 1
	if tw >= 6 {
		w = 2
	}
	h := y1 - y0
	switch vw {
	case vFront:
		b.rect(x0, y0, w, h, c)
		b.rect(x1-w, y0, w, h, c)
		b.rect(x0+w, y0+1, 1, h-1, trim)
		b.rect(x1-w-1, y0+1, 1, h-1, trim)
	case vBack:
		b.rect(x0, y0, tw, h, c)
		b.rect(x0, y0, tw, 1, trim)
	case vSide:
		b.rect(x0, y0, tw-1, h, c)
		b.rect(x1-2, y0+1, 1, h-1, trim)
	}
}

// planetTorso: suits, vests, jackets and the square on the T-shirt.
func planetTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := planetKind(s)
	if k == plNone {
		return
	}
	b.cur = 2
	tw := x1 - x0
	front, side := vw == vFront, vw == vSide
	switch k {
	case plCaptain:
		// red suit, red briefs over the blue legs, the black belt and the yellow emblem
		b.rect(x0, y1-1, tw, 3, plRedSuit)
		b.rect(x0, y1+1, tw, 1, shade(plRedSuit, 0.8))
		b.rect(x0, y1-2, tw, 1, plBlack)
		if front {
			b.set(mid, y1-2, plEmblemD) // the buckle
			b.rect(mid-1, y0+2, 3, 3, plEmblem)
			b.set(mid, y0+3, plRedSuit)
			b.set(mid+1, y0+4, plEmblemD)
			b.rect(x0, y0, 1, 1, plRedL)
		} else if side {
			b.rect(x1-2, y0+2, 2, 3, plEmblem)
			b.set(x1-1, y0+3, plRedSuit)
		}
	case plEarth:
		planetVest(b, vw, x0, x1, y0, y1, plVestE, plCream)
	case plWater:
		planetVest(b, vw, x0, x1, y0, y1, plVestW, shade(plVestW, 1.3))
	case plHeart:
		planetVest(b, vw, x0, x1, y0, y1, plVestH, shade(plVestH, 1.3))
	case plFire:
		// the purple jacket is the shirt; the pink T-shirt shows in the middle
		if front {
			b.rect(mid-1, y0, 3, y1-y0-1, s.Shirt2)
			b.rect(mid-1, y0, 3, 1, shade(s.Shirt2, 1.15))
		} else if side {
			b.rect(x1-1, y0, 1, y1-y0-1, s.Shirt2)
		}
	case plWind:
		// a green square on the red T-shirt
		if front {
			b.rect(mid-1, y0+2, 3, 3, plGreenSq)
			b.set(mid, y0+3, shade(plGreenSq, 1.3))
		} else if side {
			b.rect(x1-2, y0+2, 2, 3, plGreenSq)
		}
	}
	_ = p
}

// planetLeg: the Captain's red boots and the yellow socks of the Heart friend.
func planetLeg(b *pixBuf, s *CharSpec, vw int, sxx, syy, legW int, dark bool) {
	k := planetKind(s)
	switch k {
	case plCaptain:
		boot := plRedSuit
		if dark {
			boot = shade(boot, 0.8)
		}
		x, w := sxx, legW
		if vw == vSide {
			x, w = sxx-1, legW+2
		}
		b.rect(sxx, syy-4, legW, 4, boot)
		b.rect(sxx, syy-4, legW, 1, shade(boot, 1.25)) // the top of the boot
		b.rect(x, syy, w, 2, boot)
	case plHeart:
		sock := plSock
		if dark {
			sock = shade(sock, 0.85)
		}
		b.rect(sxx, syy-2, legW, 2, sock)
	}
}

// planetHand: one dot of the colour of the friend's power ring on the right hand.
func planetHand(b *pixBuf, s *CharSpec, vw int, ex, ey float64, armW int, dark bool, id uint8) {
	k := planetKind(s)
	if k == plNone || k == plCaptain || id != 4 {
		return
	}
	rc := planetRing(k)
	c := hx(rc[0])
	if dark {
		c = hx(rc[1])
	}
	x, y := int(math.Floor(ex)), int(math.Floor(ey))
	b.set(x, y, c)
	_ = vw
	_ = armW
}

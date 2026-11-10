package main

import (
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// Guests dressed for the party. In the birthday party many wear a cone hat, on the rink they wear helmets, team
// shirts and skates (some carry a stick), and in the 70s disco they have afros, sideburns, shades and bell-bottoms.
// Chosen with their own dice (never g.rng, so the rest of the party is rolled as always); a guest that is dressed
// replaces the random one (no name, never a famous face).
// ---------------------------------------------------------------------------

var (
	crowdRng  *rand.Rand
	crowdSeed int64 // tests: non-zero = deterministic
)

func crowdDice() *rand.Rand {
	if crowdRng == nil {
		sd := crowdSeed
		if sd == 0 {
			sd = time.Now().UnixNano()
		}
		crowdRng = rand.New(rand.NewSource(sd))
	}
	return crowdRng
}

// crowdKind: which dressing the level asks for ("" = none).
func crowdKind(def *LevelDef) string {
	if def == nil {
		return ""
	}
	switch def.Icon {
	case "cake", "hockey", "disco70", "italy":
		return def.Icon
	}
	return ""
}

// crowdChance: share of the guests that arrive dressed for the party.
var crowdChance = map[string]float64{"cake": 0.5, "hockey": 0.45, "disco70": 0.6, "italy": 0.45}

// crowdSpec dresses one guest.
func crowdSpec(kind string, r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	switch kind {
	case "cake":
		s.Hat, s.HatCol = HatPartyCone, hx(pick(r, []uint32{0xe04a5a, 0xf0d030, 0x40a0e0, 0x60c070, 0xe880c0}))
		if r.Intn(4) == 0 {
			s.Carry, s.CarryCol = HoldBalloons, hx(pick(r, []uint32{0xe04a5a, 0x40a0e0, 0xf0d030}))
		} else if r.Intn(5) == 0 {
			s.Carry = HoldCup
		}
	case "hockey":
		team := pick(r, []uint32{0xc03840, 0x2c5aa8, 0x2a8a5a, 0xd8a020, 0x6a3a9a})
		s.Top, s.Shirt, s.Shirt2, s.Pattern = TopLong, hx(team), hx(0xf4f4f8), 2
		s.Bottom, s.Pants = BotPants, hx(0x20222c)
		s.Shoes = hx(0xe8eef4) // white skates
		if r.Intn(2) == 0 {
			s.Hat, s.HatCol = HatHelmet, hx(team)
		} else if r.Intn(3) == 0 {
			s.Hat, s.HatCol = HatBeanie, hx(team)
		}
		if r.Intn(3) == 0 {
			s.Carry = HoldStick
		}
	case "disco70":
		s.Hair2 = HairAfro
		if s.Female {
			s.Hair = hx(pick(r, []uint32{0x14100c, 0x2a1810, 0x5a3418, 0xb06a28}))
		} else if r.Intn(3) == 0 { // some men go for a long mane or a plain cut instead
			s.Hair2 = HairLong
		}
		if r.Intn(2) == 0 {
			s.Glasses = []int{GlassShades, GlassBig, GlassRound}[r.Intn(3)]
		}
		if !s.Female {
			s.Beard = []int{BeardSideburns, BeardSideburns, BeardMustache, BeardNone}[r.Intn(4)]
		}
		s.Bottom, s.Pants = BotFlare, hx(pick(r, []uint32{0xd8761e, 0x7a3e1a, 0x5a2a6a, 0xf0e8d0, 0x2a7a8a}))
		s.Top = []int{TopHawaiian, TopLong, TopTee}[r.Intn(3)]
		s.Shirt = hx(pick(r, []uint32{0xf0d030, 0xe880c0, 0x40c0b0, 0xf4f0e0, 0xd05a30}))
		s.Shirt2 = hx(pick(r, []uint32{0xd8761e, 0xb040a0, 0x2a7a8a}))
		s.Shoes = hx(pick(r, []uint32{0xf4f0e0, 0x5a2a6a, 0xc89a30}))
	case "italy": // ITALIA (italy.go): gondolier stripes, chefs, shades, pizza, gelato cups and mandolins
		switch r.Intn(4) {
		case 0: // gondolier: striped shirt
			s.Top, s.Shirt, s.Shirt2, s.Pattern = TopTee, hx(0xf4f0e8), hx(pick(r, []uint32{0x1a2a5a, 0xc03840})), 2
			s.Bottom, s.Pants = BotPants, hx(0x1a1a22)
		case 1: // pizzaiolo
			s.Hat, s.HatCol = HatChef, hx(0xf4f4f4)
			s.Top, s.Shirt = TopTee, hx(0xf4f0e8)
			s.Carry = HoldPizza
		case 2: // la dolce vita: linen, shades, a hat
			s.Shirt = hx(pick(r, []uint32{0xf0e8d0, 0x8ab8d8, 0xe8a0a0, 0x3a7a4a}))
			s.Glasses = GlassShades
			if r.Intn(2) == 0 {
				s.Hat, s.HatCol = HatFedora, hx(pick(r, []uint32{0xe8d8a8, 0x3a3a40}))
			}
			s.Carry = HoldCup // a gelato cup
		default: // a street musician or just green, white and red
			s.Shirt = hx(pick(r, []uint32{0x2a8a4a, 0xf4f4f0, 0xc83a30}))
			if r.Intn(3) == 0 {
				s.Carry = HoldGuitar
			}
		}
	}
	return s
}

// venueGuestCS: a dressed guest for this level, or nil when this one stays as the dice made him.
func (g *Game) venueGuestCS(def *LevelDef) *CharSet {
	kind := crowdKind(def)
	if kind == "" {
		return nil
	}
	r := crowdDice()
	if r.Float64() >= crowdChance[kind] {
		return nil
	}
	spec := crowdSpec(kind, r)
	return newCharSet(&spec)
}

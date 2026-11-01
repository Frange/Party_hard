package main

import (
	"math/rand"
)

func pick(r *rand.Rand, a []uint32) uint32 { return a[r.Intn(len(a))] }

var skinTones = []uint32{0xf4c9a4, 0xe8b48a, 0xc98a5e, 0x9c6a43, 0x6b4630, 0xf7d9c0, 0xeec0a0, 0xd9a07a}
var hairTones = []uint32{0x2b1d12, 0x4a2c17, 0x8a5a2b, 0xd9a441, 0xe8d6a0, 0x1a1a22, 0xb03a2e, 0x6a6a74, 0xd94a9a, 0x3aa0d0, 0x5ac05a, 0x9a4adc, 0xf0f0f0}
var shirtTones = []uint32{0xe04040, 0x3aa0e0, 0x45c45a, 0xf0c030, 0xb050d0, 0xff8030, 0x20c0b0, 0xf060a0, 0xe8e8f0, 0x707090, 0x90e040, 0xff5050, 0x1e1e2a, 0x2a8a8a, 0xe0e060, 0xc03a8a, 0x8a4a2a, 0x5a5a8a}
var pantsTones = []uint32{0x2a3a6a, 0x1e1e2a, 0x4a3a2a, 0x5a5a66, 0x263a4a, 0x7a2a4a, 0xc8b88a, 0x3a5a3a, 0x6a3a2a, 0x3a5aa8}
var shoeTones = []uint32{0x1a1a22, 0x3a2a1a, 0xe8e8f0, 0x7a2a2a, 0x2a3a6a}

func baseSpec(r *rand.Rand) CharSpec {
	return CharSpec{
		Skin: hx(pick(r, skinTones)), Hair: hx(pick(r, hairTones)), Shirt: hx(pick(r, shirtTones)), Shirt2: hx(pick(r, shirtTones)),
		Pants: hx(pick(r, pantsTones)), Shoes: hx(pick(r, shoeTones)), Accent: hx(0xa02030), HatCol: hx(pick(r, shirtTones)),
	}
}

func stature(r *rand.Rand) int {
	switch n := r.Intn(100); {
	case n < 55:
		return 0
	case n < 68:
		return -1
	case n < 76:
		return -2
	case n < 86:
		return 1
	case n < 94:
		return 2
	default:
		return 3
	}
}

// genericSpec: a random party-goer; the body is similar but height, build, hair, skin,
// clothes, hats, masks and beards vary a lot.
func genericSpec(r *rand.Rand) CharSpec {
	s := baseSpec(r)
	s.Female = r.Intn(2) == 0
	s.Stature = stature(r)
	if s.Female {
		s.Hair2 = []int{HairLong, HairLong, HairBob, HairBun, HairPony, HairShort, HairCurly, HairBuns}[r.Intn(8)]
		switch r.Intn(7) {
		case 0, 1:
			s.Top = TopDress
		case 2:
			s.Top = TopTank
		case 3:
			s.Top = TopTee
			s.Bottom = BotShorts
		case 4:
			s.Top = TopHoodie
		case 5:
			s.Top = TopTee
			s.Bottom = BotSkirt
			s.Pants = hx(pick(r, shirtTones))
		default:
			s.Top = TopTee
		}
		if r.Intn(8) == 0 {
			s.Glasses = GlassRound
		}
		s.Body = []int{BodyNormal, BodyNormal, BodySlim, BodyBig, BodyFat, BodyObese}[r.Intn(6)]
		if r.Intn(100) < 50 {
			decorateWoman(&s, r)
		}
	} else {
		s.Hair2 = []int{HairShort, HairShort, HairShort, HairBald, HairMohawk, HairCurly, HairPompadour, HairSpiky, HairComb, HairMessy}[r.Intn(10)]
		s.Top = []int{TopTee, TopTee, TopTee, TopLong, TopTank, TopHoodie, TopHawaiian, TopSuit}[r.Intn(8)]
		if r.Intn(5) == 0 {
			s.Bottom = BotShorts
		}
		s.Beard = []int{BeardNone, BeardNone, BeardNone, BeardStubble, BeardFull, BeardMustache, BeardGoatee}[r.Intn(7)]
		if r.Intn(7) == 0 {
			s.Glasses = []int{GlassRound, GlassShades}[r.Intn(2)]
		}
		s.Body = []int{BodyNormal, BodyNormal, BodyNormal, BodySlim, BodyBig, BodyFat, BodyGiant, BodyObese}[r.Intn(8)]
	}
	if r.Intn(12) == 0 {
		s.Head = HeadBig
	}
	s.Pattern = r.Intn(3)
	if r.Intn(6) == 0 && s.Hair2 != HairMohawk && s.Hair2 != HairPompadour {
		s.Hat = []int{HatCap, HatBeanie, HatCap, HatBucket, HatFedora}[r.Intn(5)]
	}
	if !s.Female && r.Intn(24) == 0 {
		// wrestler mask
		s.Mask = hx(pick(r, []uint32{0xe03030, 0x3060e0, 0x30b050, 0xf0c030, 0xb040d0, 0x202028}))
		s.Hair2, s.Hat, s.Beard, s.Glasses = HairBald, HatNone, BeardNone, GlassNone
		s.Top, s.Bottom = TopBare, BotTrunks
		s.Pants = hx(pick(r, shirtTones))
		s.Shoes = s.Pants
		s.Body = BodyBig
	}
	if s.Top == TopSuit {
		s.Shirt = hx(pick(r, []uint32{0x1e1e2a, 0x3a3a50, 0x5a3a2a}))
		s.Shirt2 = hx(0xe8e8f0)
		s.Pants = shade(s.Shirt, 0.9)
		s.Accent = hx(pick(r, shirtTones))
	}
	return s
}

func mk(r *rand.Rand, name string) CharSpec {
	s := baseSpec(r)
	s.Name = name
	s.Hair2 = HairShort
	return s
}

func presetElvis(r *rand.Rand) CharSpec {
	s := mk(r, "elvis")
	s.Skin, s.Hair = hx(0xf0c8a8), hx(0x14101a)
	s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopSuit, hx(0xf6f6fa), hx(0xe8c840), hx(0xe8c840), hx(0xf6f6fa)
	s.Hair2, s.Beard, s.Glasses = HairPompadour, BeardSideburns, GlassShades
	return s
}

func presetAlien(r *rand.Rand) CharSpec {
	s := mk(r, "alien")
	s.Skin = hx(0x6ad04a)
	s.Top, s.Shirt = TopBare, hx(0x6ad04a)
	s.Bottom, s.Pants = BotShorts, hx(0x7a8aa0)
	s.Shoes = hx(0x5ab03a)
	s.Hair2, s.Head, s.Eyes, s.Body = HairBald, HeadBig, EyesAlien, BodySlim
	return s
}

func presetET(r *rand.Rand) CharSpec {
	s := mk(r, "et")
	s.Skin = hx(0x9a7a50)
	s.Top, s.Shirt = TopBare, hx(0x9a7a50)
	s.Bottom, s.Pants = BotNone, hx(0x9a7a50)
	s.Shoes = hx(0x8a6a40)
	s.Hair2, s.Head, s.Body, s.Stature = HairBald, HeadBig, BodySlim, 3
	return s
}

func presetHomer(r *rand.Rand) CharSpec {
	s := mk(r, "homer")
	s.Skin, s.Hair = hx(0xf8d838), hx(0x2a2020)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopTee, hx(0xf4f4f8), hx(0x4a6ab0), hx(0x6a6a7a)
	s.Hair2, s.Body = HairHomer, BodyFat
	s.Head = HeadBig
	return s
}

func presetBart(r *rand.Rand) CharSpec {
	s := mk(r, "bart")
	s.Skin, s.Hair = hx(0xf8d838), hx(0xf8d838)
	s.Top, s.Shirt, s.Bottom, s.Pants, s.Shoes = TopTee, hx(0xf06a20), BotShorts, hx(0x4a6ab0), hx(0xe8e8f0)
	s.Hair2, s.Body, s.Stature, s.Head = HairSpiky, BodySlim, 4, HeadBig
	return s
}

func presetMarge(r *rand.Rand) CharSpec {
	s := mk(r, "marge")
	s.Head = HeadBig
	s.Skin, s.Hair = hx(0xf8d838), hx(0x4a8ad8)
	s.Top, s.Shirt, s.Pants, s.Shoes, s.Female = TopDress, hx(0x4aa860), hx(0x4aa860), hx(0xd04040), true
	s.Hair2 = HairBeehive
	return s
}

func presetLisa(r *rand.Rand) CharSpec {
	s := mk(r, "lisa")
	s.Skin, s.Hair = hx(0xf8d838), hx(0xf8d838)
	s.Top, s.Shirt, s.Pants, s.Shoes, s.Female = TopDress, hx(0xf06a20), hx(0xf06a20), hx(0x4a4aa8), true
	s.Hair2, s.Body, s.Stature, s.Head = HairStar, BodySlim, 3, HeadBig
	return s
}

func presetBurns(r *rand.Rand) CharSpec {
	s := mk(r, "burns")
	s.Head = HeadBig
	s.Skin, s.Hair = hx(0xf8d838), hx(0xd8d8d8)
	s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopSuit, hx(0x2a2a30), hx(0xf0f0f0), hx(0x2a8a40), hx(0x2a2a30)
	s.Body, s.Hair2 = BodySlim, HairBald
	s.Stature = 1
	return s
}

func presetFlanders(r *rand.Rand) CharSpec {
	s := mk(r, "flanders")
	s.Head = HeadBig
	s.Skin, s.Hair = hx(0xf8d838), hx(0x3a2a1a)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0x4aa860), hx(0x4a6ab0), hx(0x3a2a1a)
	s.Beard, s.Glasses = BeardMustache, GlassRound
	return s
}

func presetFry(r *rand.Rand) CharSpec {
	s := mk(r, "fry")
	s.Head = HeadBig
	s.Skin, s.Hair = hx(0xf0c8a4), hx(0xf08a20)
	s.Top, s.Shirt, s.Shirt2, s.Pants, s.Shoes = TopLong, hx(0xc82a2a), hx(0xf4f4f8), hx(0x3a5a9a), hx(0xe8e8f0)
	s.Hair2 = HairFry
	return s
}

func presetLeela(r *rand.Rand) CharSpec {
	s := mk(r, "leela")
	s.Head = HeadBig
	s.Skin, s.Hair = hx(0xf0c8d0), hx(0x8a3ac8)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopTank, hx(0xf0f0f0), hx(0x4a4a8a), hx(0x2a2a4a)
	s.Hair2, s.Eyes, s.Female = HairPony, EyesCyclops, true
	return s
}

func presetBender(r *rand.Rand) CharSpec {
	s := mk(r, "bender")
	s.Body, s.Hat = BodyRobot, HatAntenna
	s.Hair2 = HairBald
	s.Shoes = hx(0x5a6270)
	return s
}

func presetFarnsworth(r *rand.Rand) CharSpec {
	s := mk(r, "farnsworth")
	s.Head = HeadBig
	s.Skin, s.Hair = hx(0xf0d0b0), hx(0xf4f4f4)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0xf4f4f8), hx(0x3a3a50), hx(0x2a2a2a)
	s.Hair2, s.Glasses, s.Body, s.Stature = HairMessy, GlassBig, BodySlim, 1
	return s
}

func presetZoidberg(r *rand.Rand) CharSpec {
	s := mk(r, "zoidberg")
	s.Skin, s.Hair = hx(0xe05a40), hx(0xe05a40)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0xf4f4f8), hx(0x3a3a50), hx(0x2a2a2a)
	s.Head, s.Hair2, s.Beard = HeadBig, HairBald, BeardFull
	s.Hair = hx(0xc03a28)
	return s
}

func presetPeter(r *rand.Rand) CharSpec {
	s := mk(r, "peter")
	s.Head = HeadBig
	s.Skin, s.Hair = hx(0xf4d0b0), hx(0x5a3a1a)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopTee, hx(0xf4f4f8), hx(0x4a8a4a), hx(0x3a2a1a)
	s.Glasses, s.Body, s.Beard = GlassBig, BodyFat, BeardGoatee
	return s
}

func presetLois(r *rand.Rand) CharSpec {
	s := mk(r, "lois")
	s.Head = HeadBig
	s.Skin, s.Hair = hx(0xf4d0b0), hx(0xd06a20)
	s.Top, s.Shirt, s.Pants, s.Shoes, s.Female = TopTee, hx(0xf4f4f8), hx(0x4a8a4a), hx(0xe8e8f0), true
	s.Hair2 = HairBob
	return s
}

func presetStewie(r *rand.Rand) CharSpec {
	s := mk(r, "stewie")
	s.Skin, s.Hair = hx(0xf4d0b0), hx(0x3a2a1a)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopTee, hx(0xd8c030), hx(0xd03030), hx(0x2a2a4a)
	s.Hair2, s.Head, s.Body, s.Stature = HairBald, HeadHuge, BodySlim, 4
	return s
}

func presetCartman(r *rand.Rand) CharSpec {
	s := mk(r, "cartman")
	s.Skin = hx(0xf0d0a8)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0xd83a2a), hx(0xd8c830), hx(0x3a2a1a)
	s.Hat, s.HatCol = HatBeanie, hx(0x58b8e8)
	s.Body, s.Head, s.Stature = BodyFat, HeadHuge, 7
	return s
}

func presetKenny(r *rand.Rand) CharSpec {
	s := mk(r, "kenny")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopHoodie, hx(0xf08a20), hx(0xf08a20), hx(0x3a2a1a)
	s.Mask, s.MaskKind = hx(0xf08a20), MaskHood
	s.Body, s.Head, s.Stature = BodyBig, HeadHuge, 8
	return s
}

func presetKyle(r *rand.Rand) CharSpec {
	s := mk(r, "kyle")
	s.Skin = hx(0xf0d0a8)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0xf07a20), hx(0x3a5a3a), hx(0x3a2a1a)
	s.Hat, s.HatCol = HatUshanka, hx(0x2a9a3a)
	s.Body, s.Head, s.Stature = BodyBig, HeadHuge, 8
	return s
}

func presetStan(r *rand.Rand) CharSpec {
	s := mk(r, "stan")
	s.Skin = hx(0xf0d0a8)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0x7a4a2a), hx(0x3a4a8a), hx(0x3a2a1a)
	s.Hat, s.HatCol = HatBeanie, hx(0x2a4ab8)
	s.Body, s.Head, s.Stature = BodyBig, HeadHuge, 8
	return s
}

func presetVader(r *rand.Rand) CharSpec {
	s := mk(r, "vader")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopArmor, hx(0x1a1a20), hx(0x14141a), hx(0x0a0a0e)
	s.Hat, s.HatCol, s.Cape = HatVader, hx(0x16161c), hx(0x0e0e12)
	s.Body, s.Stature = BodyBig, -2
	s.Carry, s.CarryCol = HoldSaber, hx(0xff3030)
	return s
}

func presetTrooper(r *rand.Rand) CharSpec {
	s := mk(r, "trooper")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopArmor, hx(0xf0f0f4), hx(0xe4e4ea), hx(0x2a2a30)
	s.Hat, s.HatCol, s.Stature = HatTrooper, hx(0xf4f4f8), -1
	return s
}

func presetChewbacca(r *rand.Rand) CharSpec {
	s := mk(r, "chewbacca")
	s.Skin, s.Hair = hx(0x7a4a28), hx(0x6a3e20)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopTee, hx(0x7a4a28), hx(0x7a4a28), hx(0x6a3e20)
	s.Hair2, s.Body, s.Stature, s.Beard = HairMessy, BodyBig, -3, BeardFull
	return s
}

func presetYoda(r *rand.Rand) CharSpec {
	s := mk(r, "yoda")
	s.Skin = hx(0x9ac860)
	s.Top, s.Shirt, s.Shoes = TopRobe, hx(0xb89a60), hx(0x6a4a2a)
	s.Hair2, s.Ears, s.Stature, s.Head = HairBald, 3, 5, HeadBig
	s.Carry, s.CarryCol = HoldSaber, hx(0x50e050)
	return s
}

func presetJedi(r *rand.Rand) CharSpec {
	s := mk(r, "jedi")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopRobe, hx(0xb89a70), hx(0x7a5a3a), hx(0x4a3a2a)
	s.Hair, s.Beard = hx(0x6a4a2a), BeardStubble
	s.Carry, s.CarryCol = HoldSaber, hx(0x40a0ff)
	return s
}

func presetLeia(r *rand.Rand) CharSpec {
	s := mk(r, "leia")
	s.Top, s.Shirt, s.Pants, s.Shoes, s.Female = TopDress, hx(0xf6f6fa), hx(0xf6f6fa), hx(0xe8e8e8), true
	s.Hair, s.Hair2 = hx(0x4a2c17), HairBuns
	return s
}

func presetKirk(r *rand.Rand) CharSpec {
	s := mk(r, "kirk")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0xd8b030), hx(0x1a1a22), hx(0x1a1a22)
	s.Hair, s.Hair2 = hx(0x5a3a1a), HairComb
	return s
}

func presetSpock(r *rand.Rand) CharSpec {
	s := mk(r, "spock")
	s.Skin, s.Hair = hx(0xe8d09a), hx(0x14101a)
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0x2a60c8), hx(0x1a1a22), hx(0x1a1a22)
	s.Hair2, s.Ears = HairBowl, 2
	return s
}

func presetRedshirt(r *rand.Rand) CharSpec {
	s := mk(r, "redshirt")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0xd02a2a), hx(0x1a1a22), hx(0x1a1a22)
	return s
}

func presetIndy(r *rand.Rand) CharSpec {
	s := mk(r, "indy")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0x7a4a22), hx(0xb8a070), hx(0x4a2a12)
	s.Hat, s.HatCol, s.Beard = HatFedora, hx(0x8a5a2a), BeardStubble
	return s
}

func presetBond(r *rand.Rand) CharSpec {
	s := mk(r, "bond")
	s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants, s.Shoes = TopSuit, hx(0x14141a), hx(0xf4f4f8), hx(0x14141a), hx(0x14141a), hx(0x0a0a0e)
	s.Hair, s.Hair2 = hx(0x3a2a1a), HairComb
	return s
}

func presetPerroflauta(r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name = "perroflauta"
	s.Female = false
	s.Hair2, s.Hair = HairDreads, hx(pick(r, []uint32{0x4a2c17, 0x8a5a2b, 0x2b1d12}))
	s.Top, s.Pattern, s.Shirt, s.Shirt2 = TopLong, 2, hx(pick(r, shirtTones)), hx(pick(r, shirtTones))
	s.Pants, s.Shoes, s.Beard = hx(pick(r, []uint32{0x6a4a8a, 0x3a6a4a, 0xc8a050})), hx(0x6a4a2a), BeardFull
	s.Hat = HatBeanie
	if r.Intn(2) == 0 {
		s.Hat = HatNone
	}
	s.Carry = HoldBeer
	s.Body = BodySlim
	return s
}

func presetTorero(r *rand.Rand) CharSpec {
	s := mk(r, "torero")
	s.Top, s.Shirt, s.Shirt2, s.Pattern, s.Pants, s.Shoes = TopLong, hx(0xe85a9a), hx(0xe8c840), 1, hx(0xe85a9a), hx(0x14141a)
	s.Hat, s.HatCol, s.Cape, s.Hair, s.Beard = HatMontera, hx(0x14141a), hx(0xd02a2a), hx(0x14101a), BeardMustache
	s.Body = BodySlim
	return s
}

func presetSevillana(r *rand.Rand) CharSpec {
	s := mk(r, "sevillana")
	s.Female = true
	s.Top, s.Shirt, s.Shirt2, s.Pattern, s.Shoes = TopDress, hx(0xd8282a), hx(0xf8f8f8), 3, hx(0x7a2a2a)
	s.Hair, s.Hair2 = hx(0x14101a), HairBun
	return s
}

func presetFrench(r *rand.Rand) CharSpec {
	s := mk(r, "french")
	s.Top, s.Shirt, s.Shirt2, s.Pattern, s.Pants, s.Shoes = TopLong, hx(0xf4f4f8), hx(0x2a3a7a), 2, hx(0x1a1a22), hx(0x14141a)
	s.Hat, s.HatCol, s.Beard = HatBeret, hx(0x14141a), BeardMustache
	s.Carry = HoldBaguette
	return s
}

func presetChinese(r *rand.Rand) CharSpec {
	s := mk(r, "chinese")
	s.Skin = hx(0xf0d890)
	s.Top, s.Shirt, s.Shirt2, s.Pants, s.Shoes = TopLong, hx(0xd8282a), hx(0xe8c840), hx(0x14141a), hx(0x14141a)
	s.Hat, s.HatCol, s.Hair2, s.Hair = HatCone, hx(0xd8c070), HairPony, hx(0x14101a)
	s.Beard = BeardMustache
	return s
}

func presetKimono(r *rand.Rand) CharSpec {
	s := mk(r, "kimono")
	s.Female = true
	s.Skin = hx(0xf8e4cc)
	s.Top, s.Shirt, s.Shoes = TopRobe, hx(0xe8508a), hx(0x6a3a2a)
	s.Hair, s.Hair2 = hx(0x14101a), HairBun
	return s
}

func presetJapanTourist(r *rand.Rand) CharSpec {
	s := mk(r, "japan")
	s.Skin = hx(0xf0d8a0)
	s.Top, s.Shirt, s.Shirt2, s.Pattern, s.Bottom, s.Pants = TopTee, hx(0x4aa0e0), hx(0xf4f4f8), 1, BotShorts, hx(0xc8b88a)
	s.Hat, s.HatCol, s.Glasses, s.Hair = HatBucket, hx(0xf4f4f0), GlassRound, hx(0x14101a)
	s.Carry, s.Stature = HoldCamera, 1
	return s
}

func presetAmerican(r *rand.Rand) CharSpec {
	s := mk(r, "american")
	s.Top, s.Shirt, s.Shirt2, s.Pattern, s.Bottom, s.Pants = TopTee, hx(0xf4f4f8), hx(0xd02a2a), 2, BotShorts, hx(0x3a5aa8)
	s.Hat, s.HatCol, s.Body, s.Stature = HatCap, hx(0xd02a2a), BodyGiant, -1
	s.Carry = HoldBigCup
	s.Shoes = hx(0xf0f0f0)
	return s
}

func presetGerman(r *rand.Rand) CharSpec {
	s := mk(r, "german")
	s.Skin, s.Hair = hx(0xf4d0b0), hx(0xe8d070)
	s.Top, s.Shirt, s.Bottom, s.Pants, s.Susp, s.Shoes = TopTee, hx(0xf4f4f8), BotShorts, hx(0x6a4a2a), hx(0x2a6a3a), hx(0x3a2a1a)
	s.Hat, s.HatCol, s.Beard = HatTyrol, hx(0x3a6a4a), BeardMustache
	s.Carry = HoldBeer
	s.Body = BodyBig
	return s
}

func presetGermanF(r *rand.Rand) CharSpec {
	s := mk(r, "germanf")
	s.Female = true
	s.Skin, s.Hair = hx(0xf8dcc0), hx(0xf0d870)
	s.Top, s.Shirt, s.Shirt2, s.Pattern, s.Shoes = TopDress, hx(0x3a60c0), hx(0xf4f4f8), 2, hx(0x3a2a1a)
	s.Hair2 = HairPony
	s.Carry = HoldBeer
	return s
}

func presetFinn(r *rand.Rand) CharSpec {
	s := mk(r, "finn")
	s.Skin, s.Hair = hx(0xffeee8), hx(0xf8f0d8)
	s.Top, s.Bottom, s.Pants, s.Shoes = TopBare, BotTrunks, hx(0x2a60c8), hx(0xe8e8f0)
	s.Hair2, s.Stature = HairMessy, 1
	s.Carry = HoldBeer
	return s
}

func presetItalian(r *rand.Rand) CharSpec {
	s := mk(r, "italian")
	s.Top, s.Shirt, s.Shirt2, s.Pants, s.Shoes = TopLong, hx(0xf4f4f8), hx(0x3aa040), hx(0xd8282a), hx(0x1a1a22)
	s.Hat, s.Beard, s.Hair = HatChef, BeardMustache, hx(0x14101a)
	s.Carry = HoldPizza
	return s
}

func presetTribal(r *rand.Rand) CharSpec {
	s := mk(r, "tribal")
	s.Skin, s.Hair = hx(0x5a3420), hx(0x14101a)
	s.Top, s.Bottom, s.Pants, s.Shoes = TopBare, BotSkirt, hx(0xc8a040), hx(0x5a3420)
	s.Hair2, s.Accent = HairCurly, hx(0xf0f0f0)
	return s
}

func presetLuchador(r *rand.Rand) CharSpec {
	s := mk(r, "luchador")
	col := hx(pick(r, []uint32{0xe03030, 0x3060e0, 0x30b050, 0xf0c030, 0xb040d0}))
	s.Mask = col
	s.Top, s.Bottom, s.Pants, s.Shoes = TopBare, BotTrunks, hx(pick(r, []uint32{0xf0c030, 0xe03030, 0x30b050, 0x3060e0})), col
	s.Hair2, s.Body = HairBald, BodyBig
	return s
}

func presetWaiter(r *rand.Rand) CharSpec {
	s := mk(r, "waiter")
	s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants, s.Shoes = TopSuit, hx(0x1a1a22), hx(0xf4f4f8), hx(0xd02a2a), hx(0x1a1a22), hx(0x0a0a0e)
	s.Carry = HoldTray
	s.Beard = []int{BeardNone, BeardMustache}[r.Intn(2)]
	return s
}

func presetMario(r *rand.Rand) CharSpec {
	s := mk(r, "mario")
	s.Top, s.Shirt, s.Pants, s.Susp, s.Shoes = TopLong, hx(0xd82a2a), hx(0x2a4ac8), hx(0x2a4ac8), hx(0x6a3a1a)
	s.Hat, s.HatCol, s.Beard, s.Body, s.Stature = HatCap, hx(0xd82a2a), BeardMustache, BodyBig, 2
	return s
}

func presetLuigi(r *rand.Rand) CharSpec {
	s := presetMario(r)
	s.Name = "luigi"
	s.Shirt, s.HatCol = hx(0x2aa040), hx(0x2aa040)
	s.Stature = 0
	s.Body = BodyNormal
	return s
}

func presetDracula(r *rand.Rand) CharSpec {
	s := mk(r, "dracula")
	s.Skin, s.Hair = hx(0xe8e0e0), hx(0x14101a)
	s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopSuit, hx(0x14101a), hx(0xf4f4f8), hx(0xa01020), hx(0x14101a)
	s.Cape, s.Hair2 = hx(0x7a1020), HairComb
	return s
}

func presetSanta(r *rand.Rand) CharSpec {
	s := mk(r, "santa")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopLong, hx(0xd82a2a), hx(0xd82a2a), hx(0x14101a)
	s.Hat, s.HatCol, s.Beard, s.Hair, s.Body = HatBeanie, hx(0xd82a2a), BeardFull, hx(0xf4f4f4), BodyFat
	return s
}

func presetFreddie(r *rand.Rand) CharSpec {
	s := mk(r, "freddie")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopTank, hx(0xf4f4f8), hx(0xe8e8f0), hx(0xf4f4f4)
	s.Hair, s.Beard, s.Hair2 = hx(0x14101a), BeardMustache, HairShort
	s.Carry = HoldMic
	return s
}

func presetBiker(r *rand.Rand) CharSpec {
	s := mk(r, "biker")
	s.Top, s.Shirt, s.Pants, s.Shoes = TopVest, hx(0x1c1c22), hx(0x22222c), hx(0x0c0c10)
	s.Body, s.Beard, s.Hair2, s.Glasses = BodyBig, BeardFull, HairBald, GlassShades
	s.Hair = hx(0x2a1a14)
	return s
}

func presetMexican(r *rand.Rand) CharSpec {
	s := mk(r, "mexican")
	s.Top, s.Shirt, s.Shirt2, s.Pattern = TopTee, hx(0x30a050), hx(0xe04040), 2
	s.Hat, s.HatCol, s.Beard = HatSombrero, hx(0xd8b060), BeardMustache
	s.Hair = hx(0x1a1a22)
	return s
}

func presetCowboy(r *rand.Rand) CharSpec {
	s := mk(r, "cowboy")
	s.Top, s.Shirt, s.Shirt2, s.Pattern = TopLong, hx(0xb03a2a), hx(0x6a1a12), 2
	s.Pants, s.Shoes = hx(0x3a5a9a), hx(0x5a3418)
	s.Hat, s.HatCol, s.Beard = HatCowboy, hx(0x8a5a2a), BeardStubble
	return s
}

func presetHawaiian(r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name = "hawaiian"
	s.Female = false
	s.Top, s.Bottom, s.Pattern = TopHawaiian, BotShorts, 1
	s.Shirt = hx(pick(r, []uint32{0xe04a4a, 0x2ab0e0, 0xf0b020, 0x40c070}))
	s.Shirt2 = hx(0xf8f8f0)
	s.Pants = hx(pick(r, []uint32{0xf0e0b0, 0x3a6ab0}))
	s.Mask = hx(0)
	s.Mask.A = 0
	if r.Intn(2) == 0 {
		s.Glasses = GlassShades
	}
	if s.Hair2 == HairBald {
		s.Hair2 = HairShort
	}
	return s
}

func presetSwim(r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name = "swim"
	s.Mask.A = 0
	s.Top, s.Bottom = TopSwim, BotTrunks
	s.Shirt = hx(pick(r, []uint32{0xe04a8a, 0x2ab0e0, 0xf0b020, 0xe84a4a}))
	s.Pants = s.Shirt
	if !s.Female {
		s.Shirt = s.Skin
		s.Pants = hx(pick(r, []uint32{0x2a6ac0, 0xe04a4a, 0x40c070, 0xf0b020}))
	}
	s.Hat = HatNone
	return s
}

func presetRobot(r *rand.Rand) CharSpec {
	s := presetBender(r)
	s.Name = "robot"
	return s
}

// The famous and themed cast. Entries repeated appear more often.
var famousPresets = []func(*rand.Rand) CharSpec{
	presetElvis, presetAlien, presetAlien, presetET,
	presetHomer, presetBart, presetMarge, presetLisa, presetBurns, presetFlanders,
	presetFry, presetLeela, presetBender, presetFarnsworth, presetZoidberg,
	presetPeter, presetLois, presetStewie,
	presetCartman, presetKenny, presetKyle, presetStan,
	presetVader, presetTrooper, presetTrooper, presetChewbacca, presetYoda, presetJedi, presetLeia,
	presetKirk, presetSpock, presetRedshirt, presetIndy, presetBond,
	presetPerroflauta, presetPerroflauta, presetPerroflauta,
	presetTorero, presetSevillana, presetSevillana, presetFrench, presetChinese, presetKimono, presetJapanTourist,
	presetAmerican, presetAmerican, presetGerman, presetGermanF, presetFinn, presetItalian, presetTribal, presetTribal,
	presetLuchador, presetLuchador, presetWaiter, presetWaiter, presetMario, presetLuigi, presetDracula, presetSanta, presetFreddie,
	presetBiker, presetMexican, presetCowboy, presetCowboy, presetHawaiian, presetHawaiian, presetHawaiian, presetSwim, presetSwim, presetRobot,
}

func randomGuestSpec(r *rand.Rand) CharSpec {
	if len(castSlugs) > 0 && r.Intn(100) < 35 { // bought characters come first
		if f := presetByName(castSlugs[r.Intn(len(castSlugs))]); f != nil {
			return f(r)
		}
	}
	if r.Intn(100) < 52 {
		for try := 0; try < 8; try++ {
			sp := famousPresets[r.Intn(len(famousPresets))](r)
			if allowedSlug(sp.Name) {
				return sp
			}
		}
	}
	return genericSpec(r)
}

func playerSpec() CharSpec {
	return CharSpec{Name: "player", Skin: hx(0xe8d0b8), Hair: hx(0x16121c), Shirt: hx(0x2c3c78), Pants: hx(0x24305e),
		Shoes: hx(0x0e0e12), Top: TopLong, Hair2: HairComb, HatCol: hx(0x2c3c78), Shirt2: hx(0x2c3c78), Accent: hx(0x24305e)}
}

func copSpec() CharSpec {
	return CharSpec{Name: "cop", Skin: hx(0xe8b48a), Hair: hx(0x2b1d12), Shirt: hx(0x3b62c0), Shirt2: hx(0xf4f4f8), Pants: hx(0x1b2a5e),
		Shoes: hx(0x0e0e12), Top: TopUniform, Hat: HatPolice, HatCol: hx(0x1b2a5e), Accent: hx(0x14141a), Hair2: HairShort}
}

func medicSpec() CharSpec {
	return CharSpec{Name: "medic", Skin: hx(0xe8b48a), Hair: hx(0x2b1d12), Shirt: hx(0xe8f0f4), Shirt2: hx(0xe03030), Pants: hx(0xc0262a),
		Shoes: hx(0x0e0e12), Top: TopTee, Hat: HatMedic, HatCol: hx(0xe8f0f4), Accent: hx(0xe03030), Hair2: HairShort}
}

func guardSpec(r *rand.Rand) CharSpec {
	return CharSpec{Name: "guard", Skin: hx(pick(r, skinTones)), Hair: hx(0x15151b), Shirt: hx(0x1c1c24), Pants: hx(0x14141a),
		Shoes: hx(0x0a0a0e), Top: TopVest, Body: BodyBig, Glasses: GlassShades, Hair2: []int{HairBald, HairShort}[r.Intn(2)],
		Beard: []int{BeardNone, BeardStubble}[r.Intn(2)], Shirt2: hx(0x1c1c24), Accent: hx(0x1c1c24), HatCol: hx(0x1c1c24), Stature: -1}
}

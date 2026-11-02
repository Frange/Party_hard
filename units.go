package main

// The built-in responders: the police variants, FBI, military, SWAT and the Power Rangers.

type unitSet struct {
	copAlt   []CharSpec
	swat     CharSpec
	fbi      []CharSpec
	military []CharSpec
	rangers  []CharSpec
}

func unitSpecs() unitSet {
	var u unitSet
	a := copSpec()
	a.Name, a.Female, a.Hair2, a.Hair, a.Body = "cop_f", true, HairPony, hx(0x6a3a1a), BodySlim
	b := copSpec()
	b.Name, b.Body, b.Beard, b.Stature, b.Skin = "cop_big", BodyFat, BeardMustache, -1, hx(0x9a6a48)
	u.copAlt = []CharSpec{a, b}
	u.swat = CharSpec{Name: "swat", Skin: hx(0xe0b088), Hair: hx(0x1a1a1a), Shirt: hx(0x22262e), Shirt2: hx(0x22262e), Pants: hx(0x1a1c22), Shoes: hx(0x0a0a0c),
		Top: TopArmor, Hat: HatHelmet, HatCol: hx(0x22262e), Accent: hx(0x22262e), Body: BodyBig, Glasses: GlassShades, Stature: -1}
	u.fbi = []CharSpec{
		{Name: "mulder", Skin: hx(0xe8b48a), Hair: hx(0x3a2a1a), Shirt: hx(0x2a2a34), Shirt2: hx(0xf4f4f8), Pants: hx(0x2a2a34),
			Shoes: hx(0x0e0e12), Top: TopSuit, Accent: hx(0x6a2a3a), Hair2: HairComb},
		{Name: "scully", Female: true, Skin: hx(0xf4d0b4), Hair: hx(0xb04a1a), Shirt: hx(0x22222c), Shirt2: hx(0xf4f4f8), Pants: hx(0x22222c),
			Shoes: hx(0x0e0e12), Top: TopSuit, Accent: hx(0x22222c), Hair2: HairBob},
	}
	u.military = []CharSpec{
		{Name: "military_a", Skin: hx(0xc89868), Hair: hx(0x1a1a1a), Shirt: hx(0x3a4a2a), Shirt2: hx(0x3a4a2a), Pants: hx(0x2a3a22), Shoes: hx(0x14140e),
			Top: TopArmor, Hat: HatHelmet, HatCol: hx(0x3a4a2a), Accent: hx(0x3a4a2a), Body: BodyBig},
		{Name: "military_b", Skin: hx(0x8a6244), Hair: hx(0x0e0e0e), Shirt: hx(0x344028), Shirt2: hx(0x344028), Pants: hx(0x263018), Shoes: hx(0x100f0a),
			Top: TopArmor, Hat: HatHelmet, HatCol: hx(0x344028), Accent: hx(0x344028), Body: BodyBig, Beard: BeardMustache},
	}
	for _, rc := range []struct {
		name string
		col  uint32
	}{{"ranger_red", 0xd01818}, {"ranger_blue", 0x1858d0}, {"ranger_yellow", 0xe0c010}, {"ranger_green", 0x18a030}, {"ranger_pink", 0xe050a8}} {
		u.rangers = append(u.rangers, CharSpec{Name: rc.name, Skin: hx(0xe0b088), Hair: hx(0x1a1a1a), Shirt: hx(rc.col), Shirt2: hx(rc.col), Pants: hx(0x1a1a22), Shoes: hx(0x0a0a0c),
			Top: TopArmor, Hat: HatHelmet, HatCol: hx(rc.col), Accent: hx(0xf4f4f8), Body: BodyBig})
	}
	return u
}

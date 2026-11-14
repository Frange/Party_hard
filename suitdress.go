package main

// ---------------------------------------------------------------------------
// The GOLD SUIT is really put on (2026-11-12, owner: "coger el traje estando esposado: sigues con esposas y no
// cambia la ropa"). Picking it up (putOnSuit, vip.go) now changes the clothes of the player for the rest of the
// party (a gold dinner suit over his own build, like the disguise keeps his build) and, as a change of look, it takes
// the handcuffs mark off him exactly like the disguise does (useDisguise, items.go): he is no longer the suspect and,
// when nobody else is wanted, the case is closed. The 8 s in which nobody sees him are unchanged.
// ---------------------------------------------------------------------------

// suitGoldLook: the gold dinner suit (the same colours as the "Traje dorado de gala" of the shop).
func suitGoldLook(s *CharSpec) {
	s.Top, s.Shirt, s.Shirt2, s.Pants, s.Accent = TopSuit, hx(0xd8b040), hx(0xf4e8a0), hx(0xd8b040), hx(0xa01020)
}

// suitDress: p wears the gold suit (new clothes) and loses the handcuffs mark like with a disguise.
func (g *Game) suitDress(p *Player) {
	if p == nil {
		return
	}
	var sp CharSpec
	switch {
	case p.cs != nil && p.cs.Spec != nil && p.cs.Spec.Skin.A != 0:
		sp = *p.cs.Spec
	default:
		sp = *g.playerSpecFor(p.idx, "", "")
	}
	suitGoldLook(&sp)
	p.cs = newCharSet(&sp)
	if !g.isSuspectP(p) {
		return
	}
	old := g.p
	g.p = p
	defer func() { g.p = old }()
	p.disguised = true
	p.jail.sus = false // the cuffs mark leaves him (two players, twojail.go)
	if g.disguiseClosesCase() {
		g.closeCase()
	}
}

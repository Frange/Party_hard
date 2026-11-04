package main

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// More women of every build and hair colour, and the lifeguards.
// ---------------------------------------------------------------------------

var womanHair = []uint32{
	0x2b1d12, 0x4a2c17, 0x8a5a2b, 0xd9a441, 0xf0d878, 0xf4f0dc, 0x1a1a22, 0xb03a2e, 0xc8501e, 0x8a3a1a,
	0xf060a0, 0x3a8ae0, 0x40c070, 0x9a4adc, 0x30c8c8, 0xff8a2a, 0xb8b8c4, 0xe8305a,
}

var womanSkins = []uint32{0xf7d9c0, 0xf4c9a4, 0xeec0a0, 0xe8b48a, 0xd9a07a, 0xc98a5e, 0x9c6a43, 0x6b4630}

var womanDresses = []uint32{0xe03a5a, 0x1e1e2a, 0xf0f0f4, 0xc03a8a, 0x3a6ae0, 0xe8c040, 0x30a870, 0x8a3ad0, 0xff6a3a, 0xa01830}

// womanKinds: 0 curvy/plus size, 1 model, 2 busty, 3 wide hips, 4 curvy everywhere
func decorateWoman(s *CharSpec, r *rand.Rand) {
	s.Skin = hx(pick(r, womanSkins))
	s.Hair = hx(pick(r, womanHair))
	s.Hair2 = []int{HairLong, HairLong, HairLong, HairPony, HairBob, HairBuns, HairCurly, HairBun}[r.Intn(8)]
	s.Beard, s.Hat = BeardNone, HatNone
	s.Mask.A = 0
	switch r.Intn(9) {
	case 0: // plus size
		s.Body, s.Bust, s.Hips = []int{BodyFat, BodyObese}[r.Intn(2)], 2+r.Intn(2), 2+r.Intn(2)
		s.Stature = []int{-1, 0, 0}[r.Intn(3)]
		s.Top = []int{TopTee, TopTank, TopDress, TopHoodie}[r.Intn(4)]
		if s.Top != TopDress && r.Intn(2) == 0 {
			s.Bottom = BotShorts
		}
	case 1: // model: tall, slim, elegant
		s.Body, s.Bust, s.Hips = BodySlim, 1, 1
		s.Stature = 2 + r.Intn(2)
		s.Top = []int{TopDress, TopDress, TopTank}[r.Intn(3)]
		if s.Top == TopDress {
			s.Shirt = hx(pick(r, womanDresses))
			s.Shirt2 = hx(pick(r, womanDresses))
			s.Pattern = []int{0, 0, 3}[r.Intn(3)]
		} else {
			s.Bottom = BotSkirt
			s.Pants = hx(pick(r, womanDresses))
		}
		s.Shoes = hx(pick(r, []uint32{0xc02030, 0x14141a, 0xe8e8f0, 0xd8a840}))
		if r.Intn(3) == 0 {
			s.Glasses = GlassShades
		}
	case 2: // busty
		s.Body, s.Bust, s.Hips = []int{BodyNormal, BodySlim, BodyBig}[r.Intn(3)], 2, 1
		s.Top = []int{TopTank, TopTee, TopTank}[r.Intn(3)]
		s.Shirt = hx(pick(r, shirtTones))
		if r.Intn(2) == 0 {
			s.Bottom = BotSkirt
			s.Pants = hx(pick(r, womanDresses))
		}
	case 3: // wide hips
		s.Body, s.Bust, s.Hips = []int{BodyNormal, BodyBig}[r.Intn(2)], 1, 2
		s.Top = []int{TopTee, TopTank}[r.Intn(2)]
		s.Bottom = []int{BotShorts, BotSkirt}[r.Intn(2)]
		s.Pants = hx(pick(r, womanDresses))
	case 5, 6: // over the top: huge bust and/or hips
		s.Body = []int{BodyNormal, BodyBig, BodyFat, BodySlim}[r.Intn(4)]
		switch r.Intn(3) {
		case 0:
			s.Bust, s.Hips = 3, 3
		case 1:
			s.Bust, s.Hips = 3, 1
		default:
			s.Bust, s.Hips = 1, 3
		}
		s.Top = []int{TopTank, TopTee, TopDress}[r.Intn(3)]
		s.Shirt = hx(pick(r, womanDresses))
		if s.Top != TopDress {
			s.Bottom = []int{BotShorts, BotSkirt}[r.Intn(2)]
			s.Pants = hx(pick(r, womanDresses))
		}
		s.Makeup = r.Intn(2) == 0
	case 7, 8: // heavy make-up, red lips and a long evening gown
		s.Body, s.Bust, s.Hips = []int{BodySlim, BodyNormal, BodyBig}[r.Intn(3)], 1+r.Intn(2), 1+r.Intn(2)
		s.Stature = r.Intn(3)
		s.Top, s.Pattern, s.Makeup = TopDress, 3, true
		s.Shirt = hx(pick(r, []uint32{0xb0102a, 0x14141a, 0xd8b040, 0x1a7a5a, 0xc0c4d0, 0x5a1a8a, 0xe03a8a}))
		s.Shirt2 = hx(pick(r, []uint32{0xf4f0c0, 0xffffff, 0xe8d8a0}))
		s.Hair2 = []int{HairLong, HairBun, HairBeehive, HairCurly, HairBob}[r.Intn(5)]
		s.Shoes = hx(pick(r, []uint32{0xc02030, 0x14141a, 0xd8b040}))
		s.Glasses = GlassNone
	default: // curvy everywhere
		s.Body, s.Bust, s.Hips = BodyBig, 2, 2
		s.Top = []int{TopDress, TopTank, TopTee}[r.Intn(3)]
		s.Shirt = hx(pick(r, womanDresses))
		if s.Top != TopDress {
			s.Bottom = BotShorts
			s.Pants = hx(pick(r, womanDresses))
		}
	}
}

// lifeguardSpec: swimsuit, a white cap and a ring buoy or rescue board in one hand.
func lifeguardSpec(r *rand.Rand, female bool) CharSpec {
	s := baseSpec(r)
	s.Name, s.DisplayName = "lifeguard", "Socorrista"
	s.Female = female
	s.Skin = hx(pick(r, []uint32{0xd9a07a, 0xc98a5e, 0xe8b48a, 0x9c6a43, 0xf4c9a4}))
	s.Hair = hx(pick(r, womanHair))
	s.Bottom, s.Pants = BotTrunks, hx(0xd02020)
	s.Shoes = s.Skin
	s.Hat, s.HatCol = HatCap, hx(0xf4f4f4)
	s.Carry, s.CarryCol = HoldFloat, hx(0xe03030)
	if r.Intn(2) == 0 {
		s.Carry = HoldBoard
	}
	if female {
		s.Top, s.Shirt, s.Shirt2 = TopSwim, hx(0xd02020), hx(0xd02020)
		s.Hair2 = []int{HairPony, HairLong, HairBun}[r.Intn(3)]
		s.Body, s.Bust, s.Hips = []int{BodySlim, BodyNormal, BodyNormal}[r.Intn(3)], 1+r.Intn(2), 1
		s.Stature = r.Intn(3)
	} else {
		s.Top, s.Shirt, s.Shirt2 = TopBare, s.Skin, s.Skin
		s.Hair2 = []int{HairShort, HairShort, HairSpiky}[r.Intn(3)]
		s.Body = []int{BodyNormal, BodyBig, BodyBig}[r.Intn(3)]
		s.Stature = r.Intn(3)
	}
	if r.Intn(3) == 0 {
		s.Glasses = GlassShades
	}
	return s
}

// addLifeguards puts a few lifeguards around the water of beach and pool parties.
func (g *Game) addLifeguards() {
	def := g.lvl.Def
	if len(g.lvl.Pools) == 0 && !def.Swim && def.Sky != "sea" {
		return
	}
	n := 3
	if len(g.lvl.Pools) > 1 {
		n = 4
	}
	for i := 0; i < n; i++ {
		pos := g.lvl.RandomRoam(g.rng)
		if len(g.lvl.Pools) > 0 {
			pl := g.lvl.Pools[i%len(g.lvl.Pools)]
			pos = g.lvl.NearestFree(Vec{pl.X + g.rng.Float64()*pl.W, pl.Y + pl.H + 14 + g.rng.Float64()*10})
		}
		spec := lifeguardSpec(g.rng, i%2 == 1)
		gu := &Guest{pos: pos, dir: Dir(g.rng.Intn(4)), cs: newCharSet(&spec), name: nameFromSpec(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: g.rng.Float64() * 3, role: i % 2, keep: true}
		g.guests = append(g.guests, gu)
	}
}

// ---------------------------------------------------------------------------
// Weapons lying on the floor: one or two at most. A guest who picks one up
// snaps and goes after a few people; nobody else carries weapons.
// ---------------------------------------------------------------------------

func (g *Game) scatterWeapons() {
	g.weapons = nil
	k := propKinds["weapon"]
	if k == nil {
		return
	}
	r := g.rng.Float64()
	n := 0
	switch {
	case r < 0.2:
		n = 2
	case r < 0.7:
		n = 1
	}
	if fastEv {
		n = 2
	}
	for i := 0; i < n; i++ {
		var pos Vec
		for t := 0; t < 20; t++ {
			pos = g.lvl.RandomRoam(g.rng)
			if dist(pos, g.lvl.Def.Start) > 150 && !g.lvl.InPool(pos) {
				break
			}
		}
		if fastEv && len(g.guests) > 0 {
			pos = g.guests[g.rng.Intn(len(g.guests))].pos
		}
		pr := &Prop{Name: "weapon", K: k, X: pos.X, Y: pos.Y, Var: g.rng.Intn(2), Foot: Rect{pos.X - 4, pos.Y - 2, 8, 3}, Sort: pos.Y - 6}
		g.lvl.Props = append(g.lvl.Props, pr)
		g.weapons = append(g.weapons, pr)
	}
}

// maybePickWeapon: a guest walking past a weapon sometimes picks it up.
func (g *Game) maybePickWeapon(gu *Guest) {
	if gu.armed || gu.perf != "" || len(g.weapons) == 0 {
		return
	}
	switch gu.state {
	case gIdle, gGoto, gLounge, gDrink, gChat:
	default:
		return
	}
	for _, pr := range g.weapons {
		if !pr.gone && dist(gu.pos, Vec{pr.X, pr.Y}) < 12 && g.rng.Float64() < g.pickChance() {
			g.armGuest(gu, pr)
			return
		}
	}
}

func (g *Game) armGuest(gu *Guest, pr *Prop) {
	pr.gone = true
	gu.armed = true
	gu.rampage = 2 + g.rng.Intn(3)
	if gu.cs != nil && gu.cs.Spec != nil {
		hold := HoldKnife
		if pr.Var == 1 {
			hold = HoldGun
		}
		gu.origCS = gu.cs
		gu.cs = armedSet(gu.cs, hold)
	}
	gu.icon, gu.iconT = "!!!", 3
	g.snd.Play("alert")
	g.say("¡Alguien ha cogido un arma del suelo!")
	g.rampageTarget(gu)
}

// rampageTarget: the armed guest picks the closest victim (rarely the player).
func (g *Game) rampageTarget(gu *Guest) bool {
	gu.tgt, gu.tgtPlayer = nil, false
	if p := g.p; !p.dead && !p.hidden && dist(p.pos, gu.pos) < 90 && g.rng.Float64() < 0.35 {
		gu.tgtPlayer = true
	} else {
		bd := 280.0
		for _, o := range g.guests {
			if o == gu || o.out() || o.perf != "" || o.armed || o.state == gLift || o.state == gZombie {
				continue
			}
			if d := dist(o.pos, gu.pos); d < bd {
				gu.tgt, bd = o, d
			}
		}
		if gu.tgt == nil {
			g.disarm(gu)
			return false
		}
	}
	gu.state, gu.timer, gu.repath = gKiller, 40, 0
	return true
}

func (g *Game) disarm(gu *Guest) {
	if gu.origCS != nil {
		gu.cs, gu.origCS = gu.origCS, nil
	}
	gu.armed, gu.rampage = false, 0
	gu.state, gu.timer = gIdle, 3
}

func (g *Game) pickChance() float64 {
	if fastEv {
		return 0.3
	}
	return 0.015
}

// ---------------------------------------------------------------------------
// Underwear parties, exhibitionists, hippies with banners.
// ---------------------------------------------------------------------------

var underwearCols = []uint32{0xf4f4f8, 0xe03a5a, 0x3a6ae0, 0x14141a, 0xf060a0, 0x40c0f0, 0xf0c030, 0x8a3ad0, 0xc01830, 0x30a870}

func presetUnderwearM(r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	s.Female, s.Beard = false, []int{BeardNone, BeardStubble, BeardMustache}[r.Intn(3)]
	s.Top, s.Bottom = TopBare, BotTrunks
	s.Shirt, s.Shirt2 = s.Skin, s.Skin
	s.Pants = hx(pick(r, underwearCols))
	s.Hat, s.Glasses, s.Cape.A, s.Susp.A, s.Bust, s.Hips = HatNone, GlassNone, 0, 0, 0, 0
	s.Hair2 = []int{HairShort, HairShort, HairBald, HairSpiky, HairCurly, HairMessy}[r.Intn(6)]
	s.Shoes = hx(pick(r, []uint32{0xf4f4f8, 0x14141a, 0x8a5a2a}))
	return s
}

func presetUnderwearF(r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	s.Female = true
	s.Hair = hx(pick(r, womanHair))
	s.Top, s.Bottom = TopSwim, BotTrunks
	c1 := hx(pick(r, underwearCols))
	s.Shirt, s.Shirt2, s.Pants = c1, c1, c1
	s.Hat, s.Glasses, s.Beard, s.Mask.A = HatNone, GlassNone, BeardNone, 0
	s.Cape.A, s.Susp.A = 0, 0
	s.Bust, s.Hips = 1+r.Intn(3), 1+r.Intn(3)
	s.Makeup = r.Intn(3) == 0
	s.Hair2 = []int{HairLong, HairPony, HairBob, HairBuns, HairCurly}[r.Intn(5)]
	s.Shoes = hx(pick(r, []uint32{0xf4f4f8, 0x14141a, 0xc02030}))
	return s
}

// exhibitionistSpec: nothing on but a pixelated censor bar.
func exhibitionistSpec(r *rand.Rand, female bool) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	s.Female = female
	s.Top, s.Bottom = TopBare, BotNone
	s.Shirt, s.Shirt2 = s.Skin, s.Skin
	s.Hat, s.Glasses, s.Cape.A, s.Susp.A, s.Mask.A = HatNone, GlassNone, 0, 0, 0
	s.Censor = true
	s.Shoes = s.Skin
	if female {
		s.Hair = hx(pick(r, womanHair))
		s.Hair2 = []int{HairLong, HairPony, HairBob, HairCurly}[r.Intn(4)]
		s.Beard = BeardNone
		s.Bust, s.Hips = 1+r.Intn(3), 1+r.Intn(3)
	} else {
		s.Bust, s.Hips = 0, 0
	}
	return s
}

// updateExhibitionists: in about one party out of four a naked man and woman run through the crowd.
func (g *Game) updateExhibitionists() {
	if g.exhibited || !g.exhibitPlan || g.time < g.exhibitT || g.testVictim != nil || g.mode != ModePlay {
		return
	}
	if g.guestRoom() < 2 {
		return
	}
	g.exhibited = true
	for i := 0; i < 2; i++ {
		spec := exhibitionistSpec(g.rng, i == 1)
		pos := g.lvl.NearestFree(Vec{g.lvl.Def.Start.X + float64(i*40-20), g.lvl.Def.Start.Y})
		gu := &Guest{pos: pos, dir: DirUp, cs: newCharSet(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gRun, timer: 3, role: i}
		gu.runUntil = g.time + 22
		gu.icon, gu.iconT = []string{"¡LIBRE!", "¡AAAH!"}[i], 3
		gu.nick = randomPersonName(g.rng, i == 1)
		g.guests = append(g.guests, gu)
		g.total++
	}
	g.say("¡Dos exhibicionistas cruzan la fiesta sin nada puesto!")
	g.snd.Play("alert")
	for _, o := range g.guests {
		if !o.out() && o.perf == "" && g.canWatch(o) && dist(o.pos, g.p.pos) < 400 && g.rng.Float64() < 0.4 {
			o.icon, o.iconT = []string{"JAJAJA", "¡EH!", "¡OLE!", "¡FOTO!"}[g.rng.Intn(4)], 2.5
		}
	}
}

var chants = []string{"¡PAZ Y AMOR!", "¡NO A LA GUERRA!", "¡FLORES, NO BALAS!", "¡LIBERTAD!", "¡SALVA EL PLANETA!", "¡ÚNETE!"}

// spawnHippies: a group of hippies with banners that walks together and chants.
func (g *Game) spawnHippies() {
	if g.rng.Float64() > 0.55 {
		return
	}
	var leader *Guest
	base := g.lvl.RandomRoam(g.rng)
	n := 3 + g.rng.Intn(3)
	for i := 0; i < n; i++ {
		spec := presetHippieGroup(g.rng)
		pos := g.lvl.NearestFree(Vec{base.X + float64(i)*16 - 24, base.Y + float64(i%2)*10})
		gu := &Guest{pos: pos, dir: DirDown, cs: newCharSet(&spec), name: "", style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: g.rng.Float64() * 2, role: i % 2, keep: true}
		gu.chant = true
		if leader == nil {
			leader = gu
		} else {
			gu.leader = leader
		}
		g.guests = append(g.guests, gu)
	}
}

func presetHippieGroup(r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	s.Hair2 = []int{HairLong, HairLong, HairDreads, HairMessy}[r.Intn(4)]
	if s.Female {
		s.Top, s.Bottom = TopDress, BotNone
		s.Pattern = 3
		s.Hair = hx(pick(r, womanHair))
	} else {
		s.Top, s.Pattern = TopTee, 3
		s.Beard = []int{BeardFull, BeardStubble, BeardNone}[r.Intn(3)]
	}
	s.Shirt = hx(pick(r, []uint32{0xf060a0, 0x40c0f0, 0xf0d020, 0x70d040, 0xb050d0, 0xff8030}))
	s.Shirt2 = hx(pick(r, []uint32{0xf060a0, 0x40c0f0, 0xf0d020, 0x70d040, 0xb050d0}))
	s.Hat = HatNone
	s.Carry = HoldSign
	return s
}

// spawnWally: Woof, Wally's dog, wanders the grounds when the gang is around.
func (g *Game) spawnWally() {
	if g.rng.Float64() > 0.7 {
		return
	}
	if pos, ok := g.outdoorSpot(); ok {
		g.animals = append(g.animals, &Animal{kind: animDog, pos: g.lvl.NearestFree(pos), face: 1, area: Rect{pos.X - 100, pos.Y - 60, 200, 120}, seed: g.rng.Float64() * 10, timer: g.rng.Float64() * 3, coat: 2, label: "Woof"})
	}
}

// armedSet is the same character holding a weapon. The guests of the real game are drawn from the
// asset folders (rigs) and their CharSpec is nearly empty: building a new procedural set from it
// would paint an invisible body with only the knife showing.
func armedSet(src *CharSet, hold int) *CharSet {
	cs := *src
	cs.cache = map[int]*ebiten.Image{}
	if cs.rig != nil {
		r := *cs.rig
		r.carry = hold
		cs.rig = &r
		sp := *src.Spec
		cs.Spec = &sp
		return &cs
	}
	spec := *src.Spec
	spec.Carry = hold
	spec.CarryCol = hx(0xe8eef8)
	return newCharSet(&spec)
}

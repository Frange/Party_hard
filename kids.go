package main

// ---------------------------------------------------------------------------
// Three kids on wheels: Sam-sam and César ride scooters, Diego a skateboard.
// They are smaller than everybody and faster than everybody. If they see you
// they bolt, so the only way to catch one is from behind, unseen.
// ---------------------------------------------------------------------------

func ridec(kind int, col uint32) po { return func(s *CharSpec) { s.Ride, s.RideCol = kind, hx(col) } }
func facec(n int) po                { return func(s *CharSpec) { s.Face = n } }

var kidPresets = []presetEntry{
	def("sam_sam", skinc(0xf2c8a0), hairc(0xf2d84a, HairShort), tops(TopTee, 0xe83a3a, 0xf4f4f4), patn(2), bottomc(BotShorts, 0x3a5ab8), shoesc(0xf4f4f4),
		bodyc(BodySlim), headc(HeadBig), stat(7), facec(1), ridec(1, 0xe83a3a)),
	def("cesar", skinc(0xf2c8a0), hairc(0xe8c460, HairSide), tops(TopTee, 0x3aa050, 0x3aa050), bottomc(BotShorts, 0x6a4a2a), shoesc(0xf4f4f4),
		bodyc(BodySlim), headc(HeadBig), stat(7), facec(2), ridec(1, 0x3aa050)),
	def("diego", skinc(0xe0b088), hairc(0xa0743a, HairShort), tops(TopHoodie, 0x2a4ab8, 0x2a4ab8), pantsc(0x3a3a46), shoesc(0xf08a20),
		bodyc(BodySlim), headc(HeadBig), stat(7), ridec(2, 0xf08a20)),
}

func init() {
	famousPresets = append(famousPresets, asFuncs(kidPresets)...)
	for k, v := range map[string]string{"sam_sam": "Sam-sam", "cesar": "César", "diego": "Diego"} {
		displayNameOverrides[k] = v
	}
	specialPool = append(specialPool, "sam_sam", "cesar", "diego")
	quirkLines["kid"] = []string{"¡Wiiii!", "¡Más rápido!", "¡A que no me pillas!", "Je je je...", "¡Mira cómo voy!", "¡Brrrum!"}
	catchphrases["sam_sam"] = []string{"¡Mi patinete es el más rápido!", "¡Ven, corre!"}
	catchphrases["cesar"] = []string{"¿Quién se ha comido mi bocadillo? Je je.", "He hecho una travesura..."}
	catchphrases["diego"] = []string{"¡Mira este truco!", "Mi skate no frena."}
}

const kidSpeedK = 1.8

func isKid(gu *Guest) bool { return gu.quirk == "kid" }

// updateKid: spot the player in front and run for it.
func (g *Game) updateKid(gu *Guest) {
	if g.skateKidDedupe(gu); !isKid(gu) {
		return
	}
	g.kidHopTick(gu)            // a hop on the board now and then (skater_park.go)
	g.kidStreetTick(gu, 1.0/60) // outside: a trick when he stops, and the stuck watchdog (kids_street.go)
	if g.mode != ModePlay || gu.out() || g.p.dead || g.p.hidden || gu.perf != "" {
		return
	}
	if kidRiding(gu) {
		return // busy with his tricks in a bowl: he does not notice the player
	}
	switch gu.state {
	case gSleep, gKiss, gVomit, gLift, gZombie, gKiller, gPunch, gFun, gReport, gAlarm, gToPhone, gCall:
		return
	case gScared, gFlee, gRun:
		if len(gu.path) > 0 {
			return
		}
	}
	if !g.sees(gu.pos, gu.dir, g.p.pos, 135, 0.35) {
		return
	}
	if path, ok := g.kidStreetFleeTo(gu); ok { // a street kid runs away without leaving his open ground
		g.detachPartner(gu)
		gu.state, gu.timer = gScared, 4+g.rng.Float64()*3
		gu.path = path
		gu.icon, gu.iconT = []string{"¡Aaah!", "¡Corre!", "¡Viene!", "¡Socorro!"}[g.rng.Intn(4)], 1.2
		return
	}
	var best Vec
	bd := -1.0
	for i := 0; i < 12; i++ {
		t := g.lvl.RandomRoam(g.rng)
		if dist(t, gu.pos) < 80 {
			continue
		}
		if d := dist(t, g.p.pos) - 0.3*dist(t, gu.pos); d > bd {
			best, bd = t, d
		}
	}
	if bd < 0 {
		best = g.farRoam(gu.pos)
	}
	g.detachPartner(gu)
	gu.state, gu.timer = gScared, 4+g.rng.Float64()*3
	gu.path = g.lvl.PathG(gu.pos, best)
	gu.icon, gu.iconT = []string{"¡Aaah!", "¡Corre!", "¡Viene!", "¡Socorro!"}[g.rng.Intn(4)], 1.2
}

var kidSlugs = []string{"sam_sam", "cesar", "diego"}

type kidPlan struct {
	slug string
	at   float64
	done bool
}

func isKidSlug(name string) bool {
	for _, k := range kidSlugs {
		if k == name {
			return true
		}
	}
	return false
}

func (g *Game) kidPresent(slug string) bool {
	for _, gu := range g.guests {
		if !gu.out() && gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Name == slug {
			return true
		}
	}
	return false
}

// planKids decides which kids turn up after the party has started. Since 2026-11-11 (owner: "1 de cada 2
// pantallas FIJO") no dice decide it: the skate park has the three from the start, every other party of the rule
// (kids_street.go: 1, 3, 5... passing the turn on where they have no room) gets the three riding outside a few
// seconds after the start, and the rest gets none.
func (g *Game) planKids() {
	g.kidPlan = nil
	g.kidStreetSetup()
	skate := g.lvl != nil && len(g.lvl.bowls) > 0
	street := g.kidStreetActive()
	for i, slug := range kidSlugs {
		if skate { // the skate park: the three are always there from the start
			g.skateKidAtStart(slug, i)
			continue
		}
		if g.kidPresent(slug) {
			if street { // already at the party (a famous guest of the start): he is the one, kept by the trims
				for _, o := range g.guests {
					if !o.out() && o.cs != nil && o.cs.Spec != nil && o.cs.Spec.Name == slug {
						o.keep = true
						kidStreetOwn(o)
						break
					}
				}
			}
			continue
		}
		// the old dice (the bought ones always, the others 35 %) are still rolled, so the rest of the party (guests,
		// stars, the arsonist...) keeps the same dice as before; they decide nothing now
		if ownedID("skin:" + slug) {
			g.rng.Float64()
		} else if g.rng.Float64() < 0.35 {
			g.rng.Float64()
		}
		if street {
			g.kidPlan = append(g.kidPlan, kidPlan{slug: slug, at: 4 + float64(i)*3})
		}
	}
}

// skateKidAtStart: the kid is part of the skate park's start (so the trims and the kill target count him like
// anybody): the one already there (a star of the party) or a new one, kept (keep) and first in the list, as the
// start's trims cut plain guests and from the back (trimGuests). With no room at all, he comes soon after.
func (g *Game) skateKidAtStart(slug string, i int) {
	var gu *Guest
	for _, o := range g.guests {
		if !o.out() && o.cs != nil && o.cs.Spec != nil && o.cs.Spec.Name == slug {
			gu = o
			break
		}
	}
	if gu == nil {
		gu = g.addKid(slug, g.lvl.RandomRoam(g.rng))
	}
	if gu == nil {
		g.kidPlan = append(g.kidPlan, kidPlan{slug: slug, at: 1 + float64(i)*1.5})
		return
	}
	gu.keep = true
	for k, o := range g.guests {
		if o == gu {
			copy(g.guests[1:k+1], g.guests[:k])
			g.guests[0] = gu
			break
		}
	}
	for _, o := range g.guests[1:] {
		g.skateKidDedupe(o)
	}
}

// skateKidDedupe: in the skate park each of the three is there once: any other guest dressed as one of them
// (a bought character turning up, a later arrival...) becomes a plain guest, as uniqueFamous does with the famous.
// The one that stays is the first in the list (the start puts the three at the front).
// The same outside (kids_street.go): in a party of the rule the first one stays; in a party WITHOUT the three, any
// of them that came by another route (the random mix, a folder of famous guests) becomes a plain guest, except the
// ones a debug switch brought (-kidsnow, -with).
func (g *Game) skateKidDedupe(gu *Guest) {
	if g.lvl == nil || gu.out() || gu.cs == nil || gu.cs.Spec == nil || !isKidSlug(gu.cs.Spec.Name) {
		return
	}
	if len(g.lvl.bowls) == 0 && !g.kidStreetActive() {
		if kidStreet.lvl == g.lvl && !kidStreet.own[gu] && len(g.dbgWith) == 0 {
			delete(kidSessions, gu)
			gu.cs, gu.name = g.plainGuestCS(), ""
			gu.quirk, gu.quirkInit, gu.gaitInit = "", false, false
			gu.keep = false
		}
		return
	}
	for _, o := range g.guests {
		if o == gu {
			return // he is the first one: he stays
		}
		if !o.out() && o.cs != nil && o.cs.Spec != nil && o.cs.Spec.Name == gu.cs.Spec.Name {
			break
		}
	}
	delete(kidSessions, gu)
	gu.cs, gu.name = g.plainGuestCS(), ""
	gu.quirk, gu.quirkInit, gu.gaitInit = "", false, false
	gu.keep = false
}

func (g *Game) updateKidArrivals() {
	if g.mode != ModePlay || g.testVictim != nil {
		return
	}
	for i := range g.kidPlan {
		k := &g.kidPlan[i]
		if k.done || g.time < k.at {
			continue
		}
		if g.guestRoom() < 1 || g.kidPresent(k.slug) {
			if g.kidPresent(k.slug) {
				k.done = true
			} else if len(g.lvl.bowls) > 0 {
				k.at = g.time + 2 // the skate park waits for a free place: the three must come
			} else {
				k.at = g.time + 8
			}
			continue
		}
		k.done = true
		g.kidArrive(k.slug, i)
	}
}

// addKidAt puts one of the kids into the party.
func (g *Game) addKidAt(slug string, pos Vec) *Guest {
	if g.guestRoom() < 1 {
		return nil
	}
	return g.addKid(slug, pos)
}

// addKid: addKidAt without asking for room: only while the party loads, before the start is trimmed to its
// caps (trimGuests / trimStartTotal drop plain guests first: the kids are kept).
func (g *Game) addKid(slug string, pos Vec) *Guest {
	f := presetByName(slug)
	if f == nil {
		return nil
	}
	spec := f(g.rng)
	gu := &Guest{pos: g.lvl.NearestFree(pos), dir: DirDown, cs: newCharSet(&spec), name: nameFromSpec(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: 0.4, role: g.rng.Intn(2), keep: true}
	gu.icon, gu.iconT = "¡Wiiii!", 1.6
	g.guests = append(g.guests, gu)
	g.total++
	kidStreetOwn(gu)
	return gu
}

func kidIndex(slug string) int {
	for i, k := range kidSlugs {
		if k == slug {
			return i
		}
	}
	return 0
}

func (g *Game) kidArrive(slug string, idx int) {
	def := g.lvl.Def
	name := autoDisplayName(slug)
	if n, ok := displayNameOverrides[slug]; ok {
		name = n
	}
	switch {
	case def.Transit != "":
		tx, ly := g.parkSlot()
		g.veh = append(g.veh, &Vehicle{kind: 9, alt: kidIndex(slug), x: def.CarFrom.X + 60, y: ly, tx: tx, ty: ly})
		if def.Transit == "sea" {
			g.say(name + " llega a la fiesta en su lancha")
		} else {
			g.say(name + " llega a la fiesta en su nave")
		}
	case def.Police == "heli":
		if e := g.makeElevator(1); e != nil {
			e.kid = slug
			g.elevators = append(g.elevators, e)
			g.say("Se oye el ascensor: llega " + name)
			return
		}
		g.kidOnFoot(slug, name)
	default:
		g.kidOnFoot(slug, name)
	}
}

// kidOnFoot: the kid rides in along the pavement and into the party.
func (g *Game) kidOnFoot(slug, name string) {
	def := g.lvl.Def
	x := 24.0
	if def.CarFrom.X < def.CarTo.X {
		x = def.W - 24
	}
	entry, street := g.kidStreetArrivePos(Vec{x, def.CarFrom.Y - 42}) // outside: on his open ground (kids_street.go)
	gu := g.addKidAt(slug, entry)
	if gu == nil {
		return
	}
	if street {
		g.kidStreetPick(gu)
		gu.hurry = true
	} else {
		gu.path = g.lvl.PathG(gu.pos, g.lvl.RandomRoam(g.rng))
		gu.state, gu.next, gu.dur, gu.hurry = gGoto, gIdle, 2, true
	}
	if slug == "diego" {
		g.say(name + " llega en su monopatín por la acera")
	} else {
		g.say(name + " llega en su patinete por la acera")
	}
}

// kidFromVehicle: the little boat or ship has docked and the kid steps out.
func (g *Game) kidFromVehicle(v *Vehicle) {
	slug := kidSlugs[v.alt%len(kidSlugs)]
	if gu := g.addKidAt(slug, Vec{v.x - 12, v.y - 24}); gu != nil {
		if g.kidStreetPick(gu) { // a party of the rule: straight to his open ground (kids_street.go)
			gu.hurry = true
		} else {
			gu.path = g.lvl.PathG(gu.pos, g.lvl.RandomRoam(g.rng))
			gu.state, gu.next, gu.dur, gu.hurry = gGoto, gIdle, 2, true
		}
	}
	g.snd.Play("alert")
}

package main

// ---------------------------------------------------------------------------
// Traps do not choose their victims: whoever is in reach dies (guests, even those on the phone,
// animals, guards, cops, soldiers, medics, the special killers). The guests, guards and cops are
// handled by the trap code itself; this file adds the animals and the killers, which had no case.
// ---------------------------------------------------------------------------

// trapHitOthers kills the animals and the special killers whose position satisfies in.
func (g *Game) trapHitOthers(in func(Vec) bool) {
	for _, a := range append([]*Animal(nil), g.animals...) {
		if !a.dead && in(a.pos) {
			g.killAnimal(a)
		}
	}
	g.trapHitSlashers(in)
}

// trapHitSlashers: a trap kills the special killer outright (no stab, no fight back).
func (g *Game) trapHitSlashers(in func(Vec) bool) {
	for _, s := range g.allSlashers() {
		if s.state == slGone || !in(s.pos) {
			continue
		}
		g.trapKillSlasher(s)
	}
}

func (g *Game) trapKillSlasher(s *Slasher) {
	s.hp, s.state = 0, slGone
	g.stat("killers", 1)
	g.killerDefeated(s.name)
	g.spawnBody(s.pos, s.cs, s.dir)
	g.bodies[len(g.bodies)-1].label = slasherActorName(s)
	g.countKill(kcTrap, false)
	g.points += g.killVal(500)
	g.say("¡Una trampa se ha llevado al asesino! +500")
	g.snd.Play("win")
	g.checkDone()
}

// trapGasOthers: guards, animals and killers choke in the cloud like the guests do.
func (g *Game) trapGasOthers(c gasCloud, dt float64) {
	for _, gd := range g.guards {
		if !gd.dead && dist(gd.pos, c.pos) <= 56 {
			gd.gasT += dt
			if gd.gasT > 2.2 {
				g.trapGuard(gd)
			}
		}
	}
	for _, a := range g.animals {
		if !a.dead && dist(a.pos, c.pos) <= 56 {
			a.gasT += dt
			if a.gasT > 2.2 {
				g.killAnimal(a)
			}
		}
	}
	for _, s := range g.allSlashers() {
		if s.state != slGone && dist(s.pos, c.pos) <= 56 {
			s.gasT += dt
			if s.gasT > 2.2 {
				g.trapKillSlasher(s)
			}
		}
	}
}

// trapCop / trapGuard: a trap kills an officer or a guard. Nobody knows who set it, so (unlike a stab) it
// raises no heat: no witness, no "wanted", no body marked as the player's.
func (g *Game) trapCop(c *Cop) {
	g.snd.Play("stab")
	label := "Agente "
	if c.medic {
		label = "Sanitario "
	}
	g.spawnBody(c.pos, g.copCSFor(c), c.dir)
	g.bodies[len(g.bodies)-1].label = label + g.personalName(c, g.copCSFor(c))
	g.removeCop(c)
	g.copDefeated(c) // POLICÍAS card of OBJETIVOS (goalcops.go)
	g.kills++
	g.countKill(kcTrap, false)
	g.points += g.killVal(100)
}

func (g *Game) trapGuard(gd *Guard) {
	if gd.dead || g.vipTough(gd) {
		return
	}
	gd.dead, gd.icon = true, ""
	g.stat("guards", 1)
	g.spawnBody(gd.pos, gd.cs, gd.dir)
	g.bodies[len(g.bodies)-1].label = "Guardia " + g.personalName(gd, gd.cs)
	g.snd.Play("stab")
	g.kills++
	g.countKill(kcTrap, false)
	g.points += g.killVal(50)
	g.vipOnKill(gd)
	g.checkDone()
}

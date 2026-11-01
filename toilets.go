package main

// Everybody has to go sooner or later (see Guest.need). Every party gets at
// least two toilets: real ones where the level has them, portable ones where
// it does not. Some people do not make it.

// ensureToilets tops the level up to n toilets with portable ones.
func (l *Level) ensureToilets(n int) bool {
	have := len(l.POIs["toilet"])
	if have >= n {
		return false
	}
	k := propKinds["portapotty"]
	before := len(l.Props)
	added := l.ensureBooth("portapotty", n-have+countKind(l, "portapotty"), func(p *Prop) bool { return p.Name == "portapotty" }, l.POIs["toilet"], 34)
	if added {
		for _, p := range l.Props[before:] {
			if p.Name == "portapotty" {
				for _, u := range k.Use {
					l.POIs["toilet"] = append(l.POIs["toilet"], Vec{p.X + u.X, p.Y + u.Y})
				}
			}
		}
	}
	return added
}

func countKind(l *Level, name string) int {
	n := 0
	for _, p := range l.Props {
		if p.Name == name {
			n++
		}
	}
	return n
}

// pottyAt: the portable toilet whose door this guest is standing at.
func (g *Game) pottyAt(pos Vec) *Prop {
	var best *Prop
	bd := 26.0
	for _, p := range g.lvl.Props {
		if p.Name != "portapotty" || p.gone {
			continue
		}
		if d := dist(pos, Vec{p.X, p.Y + 10}); d < bd {
			best, bd = p, d
		}
	}
	return best
}

func (g *Game) pottyBusy(pr *Prop) bool {
	if g.p.hidden && g.p.hideProp == pr {
		return true
	}
	for _, o := range g.guests {
		if o.inside == pr && o.state == gToilet && !o.out() {
			return true
		}
	}
	return false
}

var wcRelief = []string{"¡Qué alivio!", "Ahhh...", "Mejor...", "Uf, necesitaba eso", "Misión cumplida", "¡Ya era hora!", "Esto sí es vida"}
var wcQueue = []string{"¡Hay cola!", "¡Date prisa!", "¡Me muero!", "¿Quién está dentro?", "¡Ocupado!", "¡Tardas un mundo!"}
var wcInside = []string{"PLOP", "PRRRT", "Ejem...", "No mires", "¡Ocupado!", "Fuuu..."}
var wcDisgust = []string{"¡QUÉ ASCO!", "¡GUARRO!", "¡HUELE FATAL!", "JAJAJA", "¡PUAJ!", "¡AIRE!"}

func (g *Game) updateToilet(gu *Guest, dt float64) {
	if !gu.wcInit {
		gu.wcInit = true
		pr := g.pottyAt(gu.pos)
		switch {
		case pr == nil:
			gu.icon, gu.iconT = []string{"Psss...", "AAAH", "Ssshh"}[g.rng.Intn(3)], 2
		case g.pottyBusy(pr):
			gu.queue, gu.wcWait = pr, 7+g.rng.Float64()*3
			gu.icon, gu.iconT = wcQueue[g.rng.Intn(len(wcQueue))], 2
		default:
			gu.inside = pr
		}
	}
	if gu.queue != nil {
		if !g.pottyBusy(gu.queue) {
			gu.inside, gu.queue = gu.queue, nil
			gu.timer = 4
		} else {
			gu.wcWait -= dt
			if gu.wcWait <= 0 {
				// nobody comes out: it ends the way it was always going to
				gu.queue = nil
				g.accident(gu, g.rng.Float64() < 0.35)
			}
			return
		}
	}
	if gu.inside != nil {
		gu.inside.shake = g.time + 0.25
		if gu.iconT <= 0 && g.rng.Float64() < dt*0.5 {
			gu.icon, gu.iconT = wcInside[g.rng.Intn(len(wcInside))], 1.4
		}
	}
	gu.timer -= dt
	if gu.timer <= 0 {
		if gu.inside != nil {
			gu.inside.shake = 0
		}
		gu.inside = nil
		gu.need = 0
		gu.icon, gu.iconT = wcRelief[g.rng.Intn(len(wcRelief))], 2
		gu.state, gu.timer = gIdle, 1.5
	}
}

// accident: they did not make it. poo=false: a puddle where they stand.
func (g *Game) accident(gu *Guest, poo bool) {
	gu.hurry, gu.path, gu.wcDone = false, nil, false
	gu.state = gFun
	if poo {
		gu.fun, gu.timer = 12, 5
		gu.icon, gu.iconT = []string{"¡NO, NO, NO!", "¡AY, NO!", "¡ME LO HE HECHO!", "¡AAAAAH!"}[g.rng.Intn(4)], 2.5
	} else {
		gu.fun, gu.timer = 11, 4
		gu.icon, gu.iconT = []string{"¡NO LLEGO!", "¡AY!", "¡NOOO!", "¡Se me escapa!", "Uy..."}[g.rng.Intn(5)], 2.5
	}
	gu.need = 0
}

// updateAccident: the mess appears a moment after the panic, and the people
// around make it worse.
func (g *Game) updateAccident(gu *Guest, dt float64) {
	if !gu.wcDone && gu.timer < 3 {
		gu.wcDone = true
		kind := 3
		if gu.fun == 12 {
			kind = 4
			g.burst(Vec{gu.pos.X, gu.pos.Y - 10}, 6, 26, rgb(0x6a3e18), 0.5, 2, 90)
			gu.icon, gu.iconT = "PLOP", 1.5
		} else {
			g.burst(Vec{gu.pos.X, gu.pos.Y - 4}, 5, 20, rgb(0xe8d850), 0.4, 2, 120)
		}
		g.decals = append(g.decals, Decal{pos: Vec{gu.pos.X, gu.pos.Y + 2}, kind: kind, r: 6 + g.rng.Float64()*3})
		for _, o := range g.guests {
			if o != gu && !o.out() && o.perf == "" && o.iconT <= 0 && dist(o.pos, gu.pos) < 90 && g.rng.Float64() < 0.5 {
				o.icon, o.iconT = wcDisgust[g.rng.Intn(len(wcDisgust))], 2.2
			}
		}
	}
	if gu.fun == 12 && gu.timer < 2.4 && g.rng.Float64() < dt*5 {
		// the smell: a lazy green cloud
		g.burst(Vec{gu.pos.X + g.rng.Float64()*10 - 5, gu.pos.Y - 22}, 1, 12, rgb(0x8ab040), 0.9, 2, -20)
	}
}

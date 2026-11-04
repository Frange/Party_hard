package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The last trophy, "Leyenda de la fiesta" (all the others), gives four powers
// that are switched with their own keys:
//   immunity (+ stabbing cops, with a golden halo), bringing famous people,
//   sending a killer, and rare events every 10 seconds.
// ---------------------------------------------------------------------------

const legendTrophy = "t_all"

// legendExtras: more rewards that come with the last trophy (also in the shop lists, locked until then).
func legendExtras() []trophyReward {
	return []trophyReward{
		wr(shopAccess, "hat:legend_crown", "Corona legendaria", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatCrown, hx(0xf0c030) }, "De oro macizo y con reflejos que ciegan. Solo para quien lo ha conseguido todo."),
		wr(shopAccess, "cape:legend", "Capa de oro", "cape", func(s *CharSpec) { s.Cape = hx(0xe8b830) }, "Una capa dorada que ondea aunque no haya viento."),
		wr(shopOutfits, "out:legend_suit", "Traje de leyenda", "outfit", func(s *CharSpec) {
			s.Top, s.Shirt, s.Shirt2, s.Pants, s.Accent = TopSuit, hx(0xf0c030), hx(0xfff4c0), hx(0xf0c030), hx(0xfff4c0)
		}, "Dorado de arriba abajo. Con él, hasta los guardias te piden un autógrafo."),
		wr(shopAccess, "haircol:legend", "Pelo de oro", "haircol", func(s *CharSpec) { s.Hair = hx(0xf0c030) }, "Brilla bajo cualquier foco."),
		wr(shopAccess, "shoe:legend", "Zapatos de oro", "shoes", func(s *CharSpec) { s.Shoes = hx(0xf0c030) }, "Se oyen a un kilómetro."),
		wep("legend_sword", "Espada de oro", HoldMachete, "La hoja más cara del mundo. Corta igual, pero queda mejor en las fotos."),
		chr("tr_leyenda", "La Leyenda", "Corona, capa y traje de oro. Se cuela en todas las fiestas, y nadie se atreve a decirle que no."),
	}
}

func init() {
	starPresets = append(starPresets, def("tr_leyenda", hatc(HatCrown, 0xf0c030), hairc(0xf0c030, HairComb), tops(TopSuit, 0xf0c030, 0xfff4c0), accentc(0xfff4c0), pantsc(0xf0c030), shoesc(0xf0c030), capec(0xe8b830), carryc(HoldTrophy), bodyc(BodyBig), stat(-1)))
	shopOnly["tr_leyenda"] = true
	displayNameOverrides["tr_leyenda"] = "La Leyenda"
	trophyList = append(trophyList, Trophy{legendTrophy, "Leyenda de la fiesta", "Consigue todos los demás trofeos.", "all", 1,
		trophyReward{tab: -1, id: "pow:legend", name: "Poderes de leyenda + ropa y objetos dorados",
			desc: "Tienes cuatro poderes con sus teclas (AJUSTES): inmunidad y matar policías con halo dorado, traer gente famosa, enviar asesinos y provocar sucesos raros cada 10 segundos. Además: corona, capa, traje, pelo, zapatos y espada de oro, y el personaje La Leyenda."}})
}

// othersDone: every trophy except the last one has been won.
func (g *Game) othersDone() bool {
	for _, t := range trophyList {
		if t.id != legendTrophy && !g.progress.Trophies[t.id] {
			return false
		}
	}
	return true
}

// allTrophies: the whole list, the last one included (the golden menu).
func (g *Game) allTrophies() bool { return g.progress.Trophies[legendTrophy] && g.othersDone() }

func (g *Game) hasLegend() bool { return g.dbg.powers || g.progress.Trophies[legendTrophy] }

func (g *Game) invulnerable() bool { return g.god || g.powerImmune || g.dbg.immune }

// haloOn: the golden halo shows while the immunity power (or the debug one) is on.
func (g *Game) haloOn() bool { return g.powerImmune || g.dbg.immune }

func (g *Game) canKillCops() bool { return g.powerImmune || g.dbg.killCops }

func (g *Game) needLegend() bool {
	if g.hasLegend() {
		return false
	}
	g.say("Ese poder se consigue con el último trofeo: completa todos los demás")
	return true
}

func (g *Game) updatePowers() {
	if g.mode != ModePlay || g.allPlayersDead() {
		return
	}
	if actImmune.justEither() && !g.needLegend() {
		g.powerImmune = !g.powerImmune
		if g.powerImmune {
			g.say("Inmunidad activada: nada puede contigo y puedes matar policías")
			g.snd.Play("win")
		} else {
			g.say("Inmunidad desactivada")
		}
	}
	if actFamous.justEither() && !g.needLegend() {
		g.powerFamous()
	}
	if actKiller.justEither() && !g.needLegend() {
		g.powerKiller()
	}
	if actPowerEvent.justEither() && !g.needLegend() {
		g.powerEvent()
	}
}

// ---- famous people ----------------------------------------------------------

func (g *Game) famousSlugs() []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if !seen[s] && presetByName(s) != nil {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range specialPool {
		add(s)
	}
	for _, grp := range starGroups {
		for _, s := range grp {
			add(s)
		}
	}
	return out
}

func (g *Game) powerFamous() {
	if g.time-g.pwFamAt < 0.5 {
		return
	}
	if g.guestRoom() < 1 {
		g.say("La fiesta ya no cabe ni un famoso más")
		return
	}
	g.pwFamAt = g.time
	pool := g.famousSlugs()
	if len(pool) == 0 {
		return
	}
	slug := pool[g.rng.Intn(len(pool))]
	spec := presetByName(slug)(g.rng)
	var pos Vec
	for i := 0; i < 20; i++ {
		pos = g.lvl.RandomRoam(g.rng)
		if dist(pos, g.p.pos) > 90 {
			break
		}
	}
	pos = g.lvl.NearestFree(pos)
	gu := &Guest{pos: pos, dir: Dir(g.rng.Intn(4)), cs: newCharSet(&spec), name: nameFromSpec(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: 0.5, role: g.rng.Intn(2), keep: true}
	gu.icon, gu.iconT = "¡Hola!", 1.6
	g.guests = append(g.guests, gu)
	g.total++
	g.burst(Vec{pos.X, pos.Y - 20}, 18, 70, rgb(0xf0c030), 0.8, 2, 80)
	g.snd.Play("alert")
	g.say("¡Llega " + gu.name + "!")
}

// ---- killers and events -----------------------------------------------------

func (g *Game) powerKiller() {
	if s := g.slasher; s != nil && s.state != slGone {
		g.say("Ya hay un asesino suelto en la fiesta")
		return
	}
	g.slasher = nil
	if !g.spawnSlasher() {
		g.say("No hay sitio para que llegue otro asesino")
	}
}

const powerEventGap = 10.0

func (g *Game) powerEvent() {
	if wait := powerEventGap - (g.time - g.pwEvAt); wait > 0 {
		g.say(fmt.Sprintf("Espera %d s para el siguiente suceso", int(wait)+1))
		return
	}
	before := len(g.events)
	for try := 0; try < 14 && len(g.events) == before; try++ {
		g.forceEv = 1 + g.rng.Intn(evKinds)
		g.startEvent()
	}
	g.forceEv = 0
	if len(g.events) == before {
		g.say("No ha pasado nada... inténtalo otra vez")
		return
	}
	g.pwEvAt = g.time
}

// ---- killing the police -----------------------------------------------------

func (g *Game) nearestCop(p Vec, r float64) *Cop {
	var best *Cop
	bd := r
	for _, c := range g.cops {
		if d := dist(c.pos, p); d < bd {
			best, bd = c, d
		}
	}
	return best
}

func (g *Game) stabCop(c *Cop) {
	g.snd.Play("stab")
	label := "Agente "
	if c.medic {
		label = "Sanitario "
	}
	g.killByPlayer = true
	g.spawnBody(c.pos, g.copCSFor(c), c.dir)
	g.killByPlayer = false
	g.bodies[len(g.bodies)-1].label = label + g.personalName(c, g.copCSFor(c))
	g.removeCop(c)
	g.kills++
	g.points += 100
	g.wanted = math.Max(g.wanted, 45)
	g.say("Has acabado con un policía")
}

// ---- the halo -----------------------------------------------------------------

func (g *Game) drawHalo(dst *ebiten.Image) {
	if !g.haloOn() || g.mode != ModePlay {
		return
	}
	for _, pl := range g.players() { // the immunity is for the whole party: every living player gets the halo
		if !pl.dead && !pl.hidden {
			g.drawHaloAt(dst, pl)
		}
	}
}

func (g *Game) drawHaloAt(dst *ebiten.Image, pl *Player) {
	x, y := g.wx(pl.pos.X), g.wy(pl.pos.Y)-42+math.Sin(g.time*3)*1.5
	pulse := 0.5 + 0.5*math.Sin(g.time*4)
	g.glowAt(dst, Vec{x - g.camI.X, y - g.camI.Y}, 20+pulse*4, color.RGBA{255, 210, 70, uint8(80 + pulse*40)})
	for a := 0.0; a < 2*math.Pi; a += 0.16 {
		px, py := x+math.Cos(a)*9, y+math.Sin(a)*3
		col := color.RGBA{255, 214, 70, 255}
		if math.Sin(a) > 0 {
			col = color.RGBA{255, 240, 150, 255} // the near side is brighter
		}
		fillRect(dst, math.Round(px), math.Round(py), 2, 1, col)
		fillRect(dst, math.Round(px), math.Round(py)+1, 2, 1, color.RGBA{200, 140, 20, 160})
	}
}

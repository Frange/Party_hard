package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The famous guests talk to each other (duets: two of them meet and play a
// little scene) and each has now and then a moment of their own.
// ---------------------------------------------------------------------------

type duetStep struct {
	who  int // 0 = the first of the pair speaks, 1 = the second
	line string
	anim int
	fx   uint32 // optional burst of colour
}

type duetDef struct {
	a, b  string
	steps []duetStep
}

func ds(who int, line string, anim int, fx ...uint32) duetStep {
	s := duetStep{who: who, line: line, anim: anim}
	if len(fx) > 0 {
		s.fx = fx[0]
	}
	return s
}

var duetDefs = []duetDef{
	{"homer", "flanders", []duetStep{ds(1, "¡Hola, vecinerino!", AnimWave), ds(0, "¡D'oh! Otra vez tú...", AnimPanic), ds(1, "¡Qué diddly bien está la fiesta!", AnimLaugh), ds(0, "¿Me prestas tu cerveza?", AnimTalk)}},
	{"homer", "burns", []duetStep{ds(1, "Excelente... ¿y usted quién es?", AnimTalk), ds(0, "¡Soy Homer, su empleado!", AnimWave), ds(1, "Smithers, suelte a los perros.", AnimPoint), ds(0, "¡D'oh!", AnimPanic)}},
	{"bart", "lisa", []duetStep{ds(0, "¡Cómete mis pantalones!", AnimLaugh), ds(1, "Eres imposible, Bart.", AnimTalk), ds(0, "¡Ay, caramba!", AnimDance3)}},
	{"fry", "bender", []duetStep{ds(0, "Bender, ¿me das una cerveza?", AnimTalk), ds(1, "¡Muerde mi brillante culo metálico!", AnimLaugh), ds(0, "¡Cállate y toma mi dinero!", AnimWave)}},
	{"rick", "morty", []duetStep{ds(0, "*burp* Morty, nos vamos.", AnimTalk), ds(1, "¡Oh, cielos, Rick!", AnimPanic), ds(0, "Wubba lubba dub dub.", AnimDance4, 0x40ff80)}},
	{"mario", "luigi", []duetStep{ds(0, "¡It's-a me, Mario!", AnimWave), ds(1, "¡Mamma mia, Mario!", AnimPanic), ds(0, "¡Let's-a go!", AnimDance3, 0xffd040)}},
	{"mario", "wario", []duetStep{ds(1, "¡Wahaha! ¡Esa moneda es mía!", AnimLaugh, 0xffd040), ds(0, "¡No, Wario, no!", AnimPanic)}},
	{"sonic", "tails", []duetStep{ds(0, "¡Demasiado lento!", AnimPoint), ds(1, "¡Espérame, Sonic!", AnimPanic), ds(0, "Vale, te espero... diez segundos.", AnimTalk)}},
	{"goku", "vegeta", []duetStep{ds(0, "¡Vegeta! ¡Pelea conmigo!", AnimWave), ds(1, "¡Soy el príncipe de los Saiyans!", AnimPoint), ds(0, "¡Kame hame ha!", AnimPunch, 0x40c0ff), ds(1, "¡Big Bang Attack!", AnimPunch, 0xffe040), ds(0, "Uf... ¡qué hambre me ha dado!", AnimTalk)}},
	{"batman", "superman", []duetStep{ds(0, "Soy Batman.", AnimTalk), ds(1, "Lo sé, Bruce.", AnimTalk), ds(0, "No me llames Bruce aquí...", AnimPanic)}},
	{"thor", "hulk", []duetStep{ds(0, "¿Otra ronda, colega?", AnimTalk), ds(1, "¡HULK APLASTA!", AnimPunch), ds(0, "¡Por Odín! ¡Es una fiesta!", AnimPanic, 0xffee70)}},
	{"vader", "luke", []duetStep{ds(0, "Yo soy tu padre.", AnimPoint), ds(1, "¡¡NOOOOOO!!", AnimPanic), ds(0, "Anda, ven a bailar.", AnimDance3)}},
	{"han_solo", "chewbacca", []duetStep{ds(0, "Tengo un mal presentimiento.", AnimTalk), ds(1, "¡Uuurrr!", AnimWave), ds(0, "Yo también te quiero, amigo.", AnimLove)}},
	{"kirk", "spock", []duetStep{ds(0, "Spock, ¿qué opinas?", AnimTalk), ds(1, "Fascinante. Es ilógico.", AnimTalk), ds(0, "Siempre tan alegre...", AnimLaugh)}},
	{"asterix", "obelix", []duetStep{ds(1, "¡Están locos estos romanos!", AnimLaugh), ds(0, "¡Por Tutatis, Obélix!", AnimPanic), ds(1, "¿Hay jabalí?", AnimTalk)}},
	{"gandalf", "frodo", []duetStep{ds(1, "No quiero este anillo.", AnimTalk), ds(0, "Un mago nunca llega tarde.", AnimPoint), ds(1, "¿Y la cerveza?", AnimWave)}},
	{"harry_potter", "hermione", []duetStep{ds(0, "¿Has visto mi varita?", AnimTalk), ds(1, "Es 'Leviosa', no 'leviosá'.", AnimPoint), ds(0, "¡Expecto patronum!", AnimWave, 0x70e0ff)}},
	{"woody", "buzz", []duetStep{ds(0, "¡Hay una serpiente en mi bota!", AnimPanic), ds(1, "¡Hasta el infinito y más allá!", AnimPoint), ds(0, "Buzz, eso no es volar...", AnimTalk)}},
	{"shrek", "burro_shrek", []duetStep{ds(1, "¿Quieres waffles?", AnimTalk), ds(0, "¡Fuera de mi pantano!", AnimPanic), ds(1, "¡No me abandones!", AnimWave)}},
	{"mortadelo", "filemon", []duetStep{ds(1, "¡Mortadeloooo!", AnimPanic), ds(0, "Jefe, soy un armario.", AnimTalk), ds(1, "¡Te voy a dar una...!", AnimPunch)}},
	{"beavis", "butthead", []duetStep{ds(0, "Huh huh, mola.", AnimLaugh), ds(1, "¡Heh heh! ¡Soy Cornholio!", AnimDance6), ds(0, "Eres idiota, Beavis.", AnimTalk)}},
	{"lennon", "mccartney", []duetStep{ds(0, "¿Tocamos Imagine?", AnimTalk), ds(1, "Mejor Yesterday...", AnimGuitar, 0xf0e0ff), ds(0, "All you need is love.", AnimSing, 0xf0e0ff)}},
	{"doc_brown", "marty_mcfly", []duetStep{ds(1, "¡Esto es pesado, Doc!", AnimPanic), ds(0, "¡Gran Scott! ¡1,21 gigavatios!", AnimWave), ds(1, "¿A dónde vamos?", AnimTalk)}},
	{"cartman", "kyle", []duetStep{ds(0, "¡Respeta mi autoridad!", AnimPoint), ds(1, "¡Eso no es justo!", AnimPanic), ds(0, "¡Me largo a casa!", AnimTalk)}},
	{"peter", "stewie", []duetStep{ds(1, "¡Mundo cruel! Victoria será mía.", AnimPoint), ds(0, "Jejeje. Stewie, qué cosas dices.", AnimLaugh)}},
	{"terminator", "robocop", []duetStep{ds(0, "Volveré.", AnimTalk), ds(1, "Tienes 20 segundos para cumplir.", AnimPoint), ds(0, "Hasta la vista, baby.", AnimPoint)}},
	{"pedro_picapiedra", "pitufo", []duetStep{ds(0, "¡Yabba dabba doo!", AnimWave), ds(1, "¡La la la la la la!", AnimDance3), ds(0, "Qué gente más rara hay aquí.", AnimTalk)}},
	{"espinete", "don_pimpon", []duetStep{ds(0, "¡Hola, Don Pimpón!", AnimWave), ds(1, "¡Hola, Espinete!", AnimWave), ds(0, "¿Jugamos a algo?", AnimTalk)}},
}

var duetGeneric = [][]duetStep{
	{ds(0, "¡Brindemos!", AnimDrink), ds(1, "¡Salud!", AnimDrink), ds(0, "¡Chin chin!", AnimLaugh, 0xffd040)},
	{ds(0, "¿Nos hacemos un selfie?", AnimTalk), ds(1, "¡Claro! ¡Sonríe!", AnimLaugh), ds(0, "¡Clic! Uf, salgo fatal.", AnimTalk)},
	{ds(0, "¿Bailamos?", AnimWave), ds(1, "¡Vamos!", AnimDance3), ds(0, "¡Qué ritmo tienes!", AnimDance4)},
	{ds(0, "¡Choca esos cinco!", AnimWave), ds(1, "¡Toma!", AnimWave, 0xffffff), ds(0, "¡Equipo!", AnimLaugh)},
	{ds(0, "¿Te cuento un secreto?", AnimTalk), ds(1, "Dime, dime...", AnimTalk), ds(0, "Aquí hay un asesino. Shhh.", AnimPanic), ds(1, "¡Qué dices!", AnimPanic)},
}

type duetScene struct {
	a, b  *Guest
	steps []duetStep
	i     int
	t     float64
	meet  bool
	wait  float64
}

func guestSlug(gu *Guest) string {
	if gu.cs != nil && gu.cs.Spec != nil {
		return gu.cs.Spec.Name
	}
	return ""
}

func (g *Game) freeToTalk(gu *Guest) bool {
	if gu.out() || gu.perf != "" || gu.quirk == "kid" || gu.armed {
		return false
	}
	switch gu.state {
	case gIdle, gDance, gLounge, gDrink:
		return true
	}
	return false
}

func (g *Game) updateDuets(dt float64) {
	if g.mode != ModePlay || g.testVictim != nil {
		return
	}
	if sc := g.duet; sc != nil {
		if !g.stepDuet(sc, dt) {
			g.endDuet(sc)
			g.duet = nil
			g.duetT = g.time + 30 + g.rng.Float64()*40
		}
		return
	}
	if g.time < g.duetT {
		return
	}
	g.duetT = g.time + 12
	if fastEv {
		g.duetT = g.time + 2
	}
	named := map[string][]*Guest{}
	var all []*Guest
	for _, gu := range g.guests {
		if sl := guestSlug(gu); sl != "" && g.freeToTalk(gu) {
			named[sl] = append(named[sl], gu)
			all = append(all, gu)
		}
	}
	if len(all) < 2 {
		return
	}
	var cands []int
	for i, d := range duetDefs {
		if len(named[d.a]) > 0 && len(named[d.b]) > 0 {
			cands = append(cands, i)
		}
	}
	var a, b *Guest
	var steps []duetStep
	if len(cands) > 0 && g.rng.Float64() < 0.8 {
		d := duetDefs[cands[g.rng.Intn(len(cands))]]
		a, b, steps = named[d.a][0], named[d.b][0], d.steps
	} else {
		a = all[g.rng.Intn(len(all))]
		for _, o := range all {
			if o != a && dist(o.pos, a.pos) < 260 && (b == nil || g.rng.Float64() < 0.4) {
				b = o
			}
		}
		if b == nil {
			return
		}
		steps = duetGeneric[g.rng.Intn(len(duetGeneric))]
	}
	if a == nil || b == nil || a == b {
		return
	}
	sc := &duetScene{a: a, b: b, steps: steps}
	mid := g.lvl.NearestFree(Vec{(a.pos.X + b.pos.X) / 2, (a.pos.Y + b.pos.Y) / 2})
	for i, gu := range []*Guest{a, b} {
		gu.partner = nil
		slot := g.lvl.NearestFree(Vec{mid.X + float64(i*2-1)*13, mid.Y})
		gu.path = g.lvl.Path(gu.pos, slot)
		gu.state, gu.next, gu.dur, gu.fun = gGoto, gFun, 30, 17
		gu.dAnim = AnimIdle
		gu.hurry, gu.hurryT = false, -99
	}
	g.duet = sc
	if fastEv {
		println("duet start", guestSlug(a), guestSlug(b), len(steps))
	}
}

// stepDuet: false when the scene is over (or broke off).
func (g *Game) stepDuet(sc *duetScene, dt float64) bool {
	for _, gu := range []*Guest{sc.a, sc.b} {
		if gu.out() || gu.fun != 17 || (gu.state != gFun && !(gu.state == gGoto && gu.next == gFun)) {
			return false
		}
	}
	if !sc.meet {
		sc.wait += dt
		if sc.a.state == gFun && sc.b.state == gFun {
			sc.meet = true
			if fastEv {
				println("duet meet")
			}
			sc.a.dir, sc.b.dir = faceTo(sc.a.pos, sc.b.pos), faceTo(sc.b.pos, sc.a.pos)
			sc.t = 0
		} else if sc.wait > 14 {
			return false
		}
		return true
	}
	if sc.i >= len(sc.steps) {
		return false
	}
	if sc.t == 0 {
		s := sc.steps[sc.i]
		sp, ot := sc.a, sc.b
		if s.who == 1 {
			sp, ot = sc.b, sc.a
		}
		sp.icon, sp.iconT = s.line, 2.6
		sp.dAnim, ot.dAnim = s.anim, AnimTalk
		if s.anim == AnimDance3 || s.anim == AnimDance4 || s.anim == AnimDance6 || s.anim == AnimLaugh || s.anim == AnimDrink {
			ot.dAnim = s.anim
		}
		if s.fx != 0 {
			g.burst(Vec{sp.pos.X, sp.pos.Y - 20}, 14, 70, rgb(s.fx), 0.6, 2, 20)
		}
	}
	sc.t += dt
	if sc.t > 2.7 {
		sc.t = 0
		sc.i++
	}
	// keep them in the scene
	sc.a.timer, sc.b.timer = 3, 3
	return true
}

func (g *Game) endDuet(sc *duetScene) {
	if fastEv {
		println("duet end", sc.meet, sc.i)
	}
	for _, gu := range []*Guest{sc.a, sc.b} {
		if gu.fun == 17 && !gu.out() {
			gu.state, gu.timer, gu.fun = gIdle, 1, 0
		}
	}
	if sc.meet {
		g.stat("duets", 1)
	}
}

// ---- moments of their own -----------------------------------------------------------

type beam struct {
	from Vec
	dir  float64
	t    float64
	col  color.RGBA
}

var starSlugs = map[string]bool{"goku": true, "freddie": true, "elvis": true, "santa": true, "thor": true, "sonic": true, "mario": true, "deadpool": true, "hulk": true, "michael_jackson": true}

var deadpoolLines = []string{"¡Hey, tú, el que mueve el ratón!", "¿Sigues jugando? Qué dedicación.", "Esto es un videojuego, ¿lo sabías?", "Cuarta pared: rota.", "Maximum effort... y poca cordura.", "¿Me cobras por salir en pantalla?"}

func (g *Game) updateStar(gu *Guest, dt float64) {
	sl := guestSlug(gu)
	if gu.out() || !starSlugs[sl] || g.mode != ModePlay {
		return
	}
	if gu.starT == 0 {
		gu.starT = g.time + 20 + g.rng.Float64()*50
		if fastEv {
			gu.starT = g.time + 4 + g.rng.Float64()*4
		}
	}
	if gu.starT < 0 { // in the middle of the act
		gu.starAct -= dt
		g.starAct(gu, sl, dt)
		return
	}
	if g.time < gu.starT || !g.freeToTalk(gu) || g.duet != nil && (g.duet.a == gu || g.duet.b == gu) {
		return
	}
	gu.starT = -1
	gu.starAct = 0
	gu.path = nil
	gu.state, gu.fun, gu.timer = gFun, 18, 99
	switch sl {
	case "goku":
		gu.starAct, gu.dAnim = 2.6, AnimPoint
		gu.icon, gu.iconT = "¡KA... ME... HA... ME...!", 2.4
	case "freddie":
		gu.starAct, gu.dAnim = 6, AnimSing
		gu.icon, gu.iconT = "¡AY-OH!", 2
	case "elvis":
		gu.starAct, gu.dAnim = 6, AnimSing
		gu.icon, gu.iconT = "Thank you, thank you very much", 2.4
	case "santa":
		gu.starAct, gu.dAnim = 4, AnimWave
		gu.icon, gu.iconT = "¡JO, JO, JO!", 2.4
	case "thor":
		gu.starAct, gu.dAnim = 2.4, AnimPoint
		gu.icon, gu.iconT = "¡POR ODÍN!", 2.4
	case "sonic":
		gu.starAct = 0.2
		gu.dAnim = AnimRun
		gu.icon, gu.iconT = "¡DEMASIADO LENTO!", 2
	case "mario":
		gu.starAct, gu.dAnim = 3.6, AnimPirouette
		gu.icon, gu.iconT = "¡YAHOO!", 2
	case "deadpool":
		gu.starAct, gu.dAnim = 4.5, AnimTalk
		gu.icon, gu.iconT = deadpoolLines[g.rng.Intn(len(deadpoolLines))], 3
	case "hulk":
		gu.starAct, gu.dAnim = 2.2, AnimPanic
		gu.icon, gu.iconT = "¡HULK... APLASTA!", 2.2
	case "michael_jackson":
		gu.starAct, gu.dAnim = 6, AnimDance10
		gu.icon, gu.iconT = "¡SHAMONE!", 2
	}
	if gu.starAct == 0 {
		gu.starT = g.time + 60
		gu.state = gIdle
	}
}

func (g *Game) starAct(gu *Guest, sl string, dt float64) {
	gu.timer = 3
	near := func(r float64, f func(o *Guest)) {
		for _, o := range g.guests {
			if o != gu && !o.out() && o.perf == "" && g.freeToTalk(o) && dist(o.pos, gu.pos) < r {
				f(o)
			}
		}
	}
	switch sl {
	case "goku":
		if gu.starAct <= 0 && !gu.starDone {
			gu.starDone = true
			dir := 1.0
			var tgt *Guest
			bd := 400.0
			for _, o := range g.guests {
				if o != gu && !o.out() && math.Abs(o.pos.Y-gu.pos.Y) < 30 && math.Abs(o.pos.X-gu.pos.X) < bd {
					tgt, bd = o, math.Abs(o.pos.X-gu.pos.X)
				}
			}
			if tgt != nil && tgt.pos.X < gu.pos.X {
				dir = -1
			} else if tgt == nil && gu.dir == DirLeft {
				dir = -1
			}
			gu.dir = DirRight
			if dir < 0 {
				gu.dir = DirLeft
			}
			g.beams = append(g.beams, beam{from: Vec{gu.pos.X, gu.pos.Y - 18}, dir: dir, t: 0.9, col: color.RGBA{90, 200, 255, 255}})
			g.snd.Play("boom")
			g.shake = 0.4
			for _, o := range g.guests {
				if o != gu && !o.out() && math.Abs(o.pos.Y-(gu.pos.Y)) < 16 && (o.pos.X-gu.pos.X)*dir > 0 && math.Abs(o.pos.X-gu.pos.X) < 260 {
					g.killGuest(o, false)
					g.sayNews("kill", map[string]string{"actor": "Goku con su Kamehameha", "victim": guestLabel(o)})
				}
			}
			g.checkDone()
		}
	case "freddie", "elvis", "michael_jackson":
		if gu.starAct > 4.8 && !gu.starDone {
			gu.starDone = true
			near(170, func(o *Guest) {
				if sl == "elvis" && !(o.cs != nil && o.cs.Spec != nil && o.cs.Spec.Female) {
					return
				}
				o.state, o.fun, o.timer = gFun, 0, 5
				o.icon, o.iconT = map[string]string{"freddie": "¡AY-OH!", "elvis": "♥", "michael_jackson": "¡WOOO!"}[sl], 2
				if sl == "elvis" {
					o.fun = 10
				}
				o.dir = faceTo(o.pos, gu.pos)
			})
		}
	case "santa":
		if gu.starAct < 3.4 && !gu.starDone {
			gu.starDone = true
			g.burst(Vec{gu.pos.X, gu.pos.Y - 24}, 40, 110, rgb(0xe03030), 1.1, 3, 90)
			g.burst(Vec{gu.pos.X, gu.pos.Y - 24}, 30, 100, rgb(0x30c060), 1.1, 3, 90)
			g.burst(Vec{gu.pos.X, gu.pos.Y - 24}, 24, 90, rgb(0xf0d030), 1.1, 3, 90)
			near(140, func(o *Guest) {
				o.icon, o.iconT = []string{"¡Un regalo!", "¡Gracias, Santa!", "¡Qué ilusión!", "¿Carbón? Otra vez..."}[g.rng.Intn(4)], 2.4
			})
		}
	case "thor":
		if gu.starAct <= 0 && !gu.starDone {
			gu.starDone = true
			var v *Guest
			bd := 150.0
			for _, o := range g.guests {
				if o != gu && !o.out() && dist(o.pos, gu.pos) < bd {
					v, bd = o, dist(o.pos, gu.pos)
				}
			}
			if v != nil {
				g.snd.Play("zap")
				g.flash = 0.18
				g.burst(Vec{v.pos.X, v.pos.Y - 14}, 16, 100, rgb(0xfff0a0), 0.4, 2, 0)
				g.killGuest(v, false)
				g.sayNews("kill", map[string]string{"actor": "El rayo de Thor", "victim": guestLabel(v)})
				g.checkDone()
			}
		}
	case "sonic":
		if !gu.starDone {
			gu.starDone = true
			gu.path = g.lvl.Path(gu.pos, g.farRoam(gu.pos))
			gu.state, gu.next, gu.dur, gu.fun = gGoto, gIdle, 2, 0
			gu.hurry, gu.hurryT = true, -99
			gu.starT = g.time + 60 + g.rng.Float64()*60
			gu.starAct = 0
			return
		}
	case "mario":
		if int(gu.starAct*4)%2 == 0 && g.rng.Float64() < dt*8 {
			g.burst(Vec{gu.pos.X, gu.pos.Y - 30}, 3, 40, rgb(0xffd040), 0.5, 2, 60)
		}
	case "deadpool":
		// talks to the player, who is the only one that can hear
	case "hulk":
		if gu.starAct <= 0 && !gu.starDone {
			gu.starDone = true
			g.snd.Play("boom")
			g.shake = 0.5
			g.decals = append(g.decals, Decal{pos: gu.pos, kind: 1, r: 14})
			g.burst(Vec{gu.pos.X, gu.pos.Y - 6}, 20, 90, rgb(0xb8b0a0), 0.6, 3, 80)
			n := 0
			for _, o := range g.guests {
				if o != gu && !o.out() && n < 2 && dist(o.pos, gu.pos) < 34 {
					g.killGuest(o, false)
					g.sayNews("kill", map[string]string{"actor": "Hulk", "victim": guestLabel(o)})
					n++
				}
			}
			g.checkDone()
		}
	}
	if gu.starAct <= 0 && gu.state == gFun {
		gu.state, gu.timer, gu.fun = gIdle, 1, 0
		gu.starT = g.time + 70 + g.rng.Float64()*70
		gu.starDone = false
	}
}

func (g *Game) updateBeams(dt float64) {
	out := g.beams[:0]
	for _, b := range g.beams {
		b.t -= dt
		if g.rng.Float64() < dt*40 {
			g.burst(Vec{b.from.X + b.dir*g.rng.Float64()*260, b.from.Y}, 1, 40, b.col, 0.3, 2, 0)
		}
		if b.t > 0 {
			out = append(out, b)
		}
	}
	g.beams = out
}

func (g *Game) drawBeams(dst *ebiten.Image) {
	for _, b := range g.beams {
		x0 := g.wx(b.from.X)
		w := 260.0
		if b.dir < 0 {
			x0 -= w
		}
		y := g.wy(b.from.Y)
		k := math.Min(1, b.t/0.4)
		fillRect(dst, x0, y-7, w, 14, color.RGBA{60, 160, 255, uint8(110 * k)})
		fillRect(dst, x0, y-4, w, 8, color.RGBA{150, 220, 255, uint8(200 * k)})
		fillRect(dst, x0, y-2, w, 4, color.RGBA{255, 255, 255, uint8(255 * k)})
	}
}

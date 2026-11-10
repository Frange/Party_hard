package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "tvcrew": a TV crew of two arrives: a CAMERAMAN with a shoulder camera (red REC light blinking) and a REPORTER
// with a microphone who talks to the lens (newscast jokes in speech bubbles). They walk to an open spot (near the
// mess when there is one), the guests around turn to the lens and wave or pose, and after ~25 s they leave.
//
// AMBIENCE ONLY: they are not witnesses (nothing in g.sees / the police logic knows them), they do not blame the player.
// Like the pizza boy they can be stabbed (pevStab: a body, a kill, the other one runs off).
// Hook for the newspaper: g.stat("tv_crew") and the id "tvcrew" in pevState.ran ("according to the local television").
//
// WHEN: 14 of the 24 parties (tvIcons), 16 % per party, between 90 and 300 s,
// once. It also waits for the mess: before 150 s it only comes if there are 4+ bodies, police or a siren about; later
// it comes anyway ("by chance"). Group "tv".
// ---------------------------------------------------------------------------

// tvIcons: the parties a TV crew would go to (14 of the 24: lively places and landmarks, not the bullring, the stadium,
// the cemetery, the cinema, the stations, the ranch, the circuit or the snow).
var tvIcons = map[string]bool{"house": true, "beach": true, "neon": true, "eiffel": true, "funfair": true, "disco": true, "festival": true,
	"skate": true, "roof": true, "casino": true, "highway": true, "cake": true, "hockey": true, "disco70": true}

const (
	tvActLen    = 24.0
	tvMessBody  = 4
	tvMessUntil = 150.0
	tvReach     = 24.0
)

var tvLines = pevTexts(
	"En directo desde la fiesta...", "Buenas noches, hay mucho ambiente", "Les habla su reportero favorito",
	"Volvemos con ustedes al estudio", "Hay buen ambiente. Y canapés", "Nos dicen que todo va con normalidad",
	"Un momento: ¿eso es un pato?", "¡Silencio, estamos en el aire!", "Cámara, enfoca la tortilla",
	"Seguimos sin saber quién invitó al pato", "Dicen que el DJ es sospechoso", "Fuentes bien informadas: no hay fuentes",
	"Corten, se me ha movido el pelo", "Estamos en vivo y muy nerviosos", "Esto es periodismo de barra libre",
	"¿Algo que declarar? ¡Hola, mamá!")

var tvMessLines = pevTexts("Se oyen sirenas. Debe ser fiesta", "Hay un charco muy sospechoso aquí", "Todo bajo control... casi todo",
	"La policía no hace declaraciones", "Nos dicen que alguien se ha ido pronto")

var tvWaves = pevTexts("¡Hola, mamá!", "¡Salgo en la tele!", "¡Saludad a la cámara!", "¡Mírame, soy famoso!", "¡Hola, abuela!", "¡Un selfi con la cámara!")

const (
	tvEnter = iota
	tvAct
	tvLeave
)

type tvState struct {
	rep, cam         *pevPerf
	phase            int
	t0               float64
	spot, camSpot    Vec
	entry, camEntry  Vec
	crowd            pevCrowd
	repDead, camDead bool
	next, shout      float64
	line             int
	mess             bool
	say              string // what the reporter says now
}

func (s *tvState) pevHas(gu *Guest) bool { return s.crowd.pevHas(gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "tvcrew", Name: "Equipo de televisión", Group: "tv",
		Prob: 0.16, AtMin: 90, AtMax: 300, Dur: 40,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && t.guests >= 24 && tvIcons[t.icon] },
		Start:    tvStart, Items: tvItems,
		Update: pevGuard(tvUpdate, func(g *Game, a *partyAct) { a.D.(*tvState).crowd.release(g, partyDice()) })})
}

// tvMess: the party is a mess (bodies, police or siren): the news come.
func tvMess(g *Game) bool {
	return len(g.bodies) >= tvMessBody || len(g.cops) > 0 || g.policeCalled
}

func tvRepSpec() CharSpec {
	return CharSpec{Skin: hx(0xe8b890), Hair: hx(0x5a3a22), Shirt: hx(0x2a4a8a), Shirt2: hx(0xf4f2ee), Pants: hx(0x24283a), Shoes: hx(0x16121c),
		Accent: hx(0xd03030), Top: TopSuit, Female: true, Makeup: true, noOverride: true}
}

func tvCamSpec() CharSpec {
	return CharSpec{Skin: hx(0xc89868), Hair: hx(0x2a1a12), Shirt: hx(0x3a3a40), Shirt2: hx(0x3a3a40), Pants: hx(0x55504a), Shoes: hx(0x16121c),
		Accent: hx(0xe0b030), HatCol: hx(0x2a2a30), Hat: HatCap, Top: TopVest, Beard: 1, noOverride: true}
}

func tvStart(g *Game, a *partyAct) bool {
	mess := tvMess(g)
	if !mess && g.time < tvMessUntil && dbgPartyEvent != "tvcrew" {
		return false // it waits for something to happen (the frame retries a few times)
	}
	r := partyDice()
	// the spot: of 30 clear ones, the nearest to the mess (or to the middle of the guests)
	focus := Vec{}
	n := 0
	if len(g.bodies) >= 2 {
		for _, b := range g.bodies {
			focus.X, focus.Y, n = focus.X+b.pos.X, focus.Y+b.pos.Y, n+1
		}
	} else {
		for _, gu := range g.guests {
			if !gu.out() {
				focus.X, focus.Y, n = focus.X+gu.pos.X, focus.Y+gu.pos.Y, n+1
			}
		}
	}
	if n == 0 {
		return false
	}
	focus = Vec{focus.X / float64(n), focus.Y / float64(n)}
	var spot Vec
	found := false
	for i := 0; i < 30; i++ {
		p, ok := g.pevSpot(r, 24, false)
		if ok && (!found || dist(p, focus) < dist(spot, focus)) {
			spot, found = p, true
		}
	}
	if !found {
		return false
	}
	camSpot := g.lvl.NearestFree(Vec{spot.X - 15, spot.Y})
	if !g.lvl.Free(camSpot) || dist(camSpot, spot) < 9 {
		camSpot = g.lvl.NearestFree(Vec{spot.X + 15, spot.Y})
	}
	var entry Vec
	var path, cpath []Vec
	for i := 0; i < 40 && cpath == nil; i++ {
		p := g.lvl.NearestFree(g.lvl.RandomRoam(r))
		if d := dist(p, spot); d < 90 || d > 260 || !g.pevPlayerFar(p, 60) {
			continue
		}
		pa := g.lvl.PathG(p, spot)
		ca := g.lvl.PathG(g.lvl.NearestFree(Vec{p.X - 8, p.Y}), camSpot)
		if pa != nil && ca != nil {
			entry, path, cpath = p, pa, ca
		}
	}
	if cpath == nil {
		return false
	}
	st := &tvState{phase: tvEnter, t0: g.time, spot: spot, camSpot: camSpot, entry: entry, mess: mess}
	st.camEntry = g.lvl.NearestFree(Vec{entry.X - 8, entry.Y})
	st.rep = newPevPerf(tvRepSpec(), entry, DirDown)
	st.cam = newPevPerf(tvCamSpec(), st.camEntry, DirDown)
	st.rep.path, st.cam.path = path, cpath
	st.next = g.time + 1
	a.D = st
	g.stat("tv_crew", 1) // newspaper hook: "according to the local television"
	return true
}

func tvUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*tvState)
	r := partyDice()
	age := g.time - st.t0
	switch st.phase {
	case tvEnter:
		ar := st.rep.step(dt)
		ac := st.cam.step(dt)
		if (ar && ac) || age > 30 {
			st.phase, st.t0 = tvAct, g.time
			st.rep.dir, st.cam.dir = DirDown, DirDown
			st.crowd.gather(g, r, st.rep.pos, 3+r.Intn(3), 30, 48, nil)
		}
	case tvAct:
		tvAct_(g, st, dt)
		if age >= tvActLen || st.repDead && st.camDead {
			tvGoHome(g, st)
		}
	case tvLeave:
		ar, ac := st.rep.step(dt), st.cam.step(dt)
		if ar && ac || age > 30 || st.repDead && st.camDead {
			return true
		}
	}
	return false
}

func tvGoHome(g *Game, st *tvState) {
	st.crowd.release(g, partyDice())
	st.phase, st.t0 = tvLeave, g.time
	st.rep.path = g.lvl.PathG(st.rep.pos, st.entry)
	st.cam.path = g.lvl.PathG(st.cam.pos, st.camEntry)
}

// tvAct_: in front of the lens: the reporter talks, the guests wave and pose in turns.
func tvAct_(g *Game, st *tvState, dt float64) {
	r := partyDice()
	st.crowd.mem = pevPruneMems(st.crowd.mem)
	for _, m := range st.crowd.mem {
		gu := m.gu
		if m.want == gGoto && gu.state == gIdle && gu.timer > 1000 { // he has reached his place in front of the lens
			m.want = gIdle
		}
		switch m.want {
		case gFun:
			m.keep()
			gu.dir = faceTo(gu.pos, st.rep.pos)
		case gIdle:
			m.keep()
			gu.dir = faceTo(gu.pos, st.rep.pos)
		}
		if g.time < m.next {
			continue
		}
		m.next = g.time + 1.8 + r.Float64()*2.6
		if m.want == gIdle && gu.state == gIdle {
			m.pose([]int{AnimWave, AnimCheer, AnimLaugh, AnimWave}[r.Intn(4)])
			if gu.icon == "" && r.Float64() < 0.5 {
				gu.icon, gu.iconT = pevPick(r, tvWaves), 1.8
			}
		} else if m.want == gFun {
			m.hold(gIdle)
		}
	}
	// the reporter talks (a bubble that changes every ~3.5 s)
	if g.time >= st.next && !st.repDead {
		st.next = g.time + 3.3 + r.Float64()*0.8
		pool := tvLines
		if st.mess && r.Float64() < 0.5 {
			pool = tvMessLines
		}
		st.line = r.Intn(len(pool))
		st.say = pool[st.line]
	}
}

// pevStab: the player's blow reaches the reporter or the cameraman (the nearest within maxD).
func (s *tvState) pevStab(g *Game, p *Player, maxD float64) bool {
	if s.phase == tvEnter {
		return false
	}
	lim := math.Min(maxD, tvReach)
	dr, dc := math.Inf(1), math.Inf(1)
	if !s.repDead {
		dr = dist(p.pos, s.rep.pos)
	}
	if !s.camDead {
		dc = dist(p.pos, s.cam.pos)
	}
	switch {
	case dr <= lim && dr <= dc:
		s.repDead = true
		g.pevKillPerf(s.rep, "Reportero")
	case dc <= lim:
		s.camDead = true
		g.pevKillPerf(s.cam, "Cámara")
	default:
		return false
	}
	if s.phase == tvAct {
		tvGoHome(g, s) // the survivor (if any) runs home; the guests are free
	}
	return true
}

// pevKillPerf: the player stabs one of our street characters (same book-keeping as the pizza boy).
func (g *Game) pevKillPerf(pf *pevPerf, label string) {
	g.killByPlayer = true
	g.spawnBody(pf.pos, pf.cs, pf.dir)
	g.bodies[len(g.bodies)-1].label = label
	g.killByPlayer = false
	g.snd.Play("stab")
	g.kills++
	g.countKill(kcKnife, false)
	g.points += g.killVal(20)
	g.witness()
	g.checkDone()
}

// pevStab: a running event may take the blow (the TV crew, the magician).
func (g *Game) pevStab(p *Player, maxD float64) bool {
	if !g.partyActive() {
		return false
	}
	for _, a := range g.pev.act {
		if s, ok := a.D.(interface {
			pevStab(*Game, *Player, float64) bool
		}); ok && s.pevStab(g, p, maxD) {
			return true
		}
	}
	return false
}

// ---- drawing -----------------------------------------------------------------------------------------------------

var (
	tvGrey  = color.RGBA{0x3c, 0x3e, 0x48, 255}
	tvGreyL = color.RGBA{0x6c, 0x70, 0x7c, 255}
	tvRed   = color.RGBA{0xff, 0x30, 0x30, 255}
	tvRedDk = color.RGBA{0x80, 0x18, 0x18, 255}
)

// drawTVCameraPix: the shoulder camera of a figure whose feet are at (cx, cy): body, lens and the REC light (blinks).
func drawTVCameraPix(plot plotFn, cx, cy, px int, k func(float64) int, t float64) {
	x, y := cx+k(2), cy-k(31)
	plot(x-px, y-px, 10*px, 8*px, pevInk)
	plot(x, y, 8*px, 6*px, tvGrey)
	plot(x, y, 8*px, px, tvGreyL)
	plot(x+px, y+6*px, 4*px, 3*px, pevInk) // the lens, pointing down at the guests
	plot(x+2*px, y+6*px, 2*px, 2*px, color.RGBA{0x8a, 0xc8, 0xf0, 255})
	if int(t*1.6)%2 == 0 {
		plot(x+6*px, y+px, 2*px, 2*px, tvRed)
	} else {
		plot(x+6*px, y+px, 2*px, 2*px, tvRedDk)
	}
}

// drawTVMicPix: the reporter's microphone in her hand: handle and a fuzzy ball with a station colour.
func drawTVMicPix(plot plotFn, cx, cy, px int, k func(float64) int) {
	x, y := cx-k(7), cy-k(22)
	plot(x, y, 2*px, 5*px, pevInk)
	plot(x-px, y-3*px, 4*px, 4*px, pevInk)
	plot(x, y-2*px, 2*px, 2*px, color.RGBA{0xf0, 0xc8, 0x30, 255})
}

func tvItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*tvState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	if !st.repDead {
		rp := st.rep
		items = append(items, drawItem{rp.pos.Y, func() {
			rp.draw(g, dst, AnimTalk, g.time*4)
			cx, cy := g.pevXY(rp.pos)
			drawTVMicPix(plot, cx, cy, px, k)
			if st.phase == tvAct && st.say != "" {
				g.bubble(rp, bubGuest, rp.pos, st.say, false, 0)
			}
		}})
	}
	if !st.camDead {
		cm := st.cam
		items = append(items, drawItem{cm.pos.Y, func() {
			cm.draw(g, dst, AnimIdle, g.time*1.2)
			cx, cy := g.pevXY(cm.pos)
			drawTVCameraPix(plot, cx, cy, px, k, g.time)
		}})
	}
	return items
}

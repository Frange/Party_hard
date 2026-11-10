package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EL EXTERMINADOR: a party killer (one of the ASESINOS) drawn as a chrome endoskeleton with glowing
// red eyes and a big rifle (ktm_art.go). The famous guest "terminator" (presets2.go) is still an
// ordinary guest; this is a different, generic killer.
//
// He follows every rule of the other killers (killerreg.go, goalkillers.go): he only comes when the
// party plan sends a killer (never two parties in a row, never in STEALTH), he is one of the cards of
// the ASESINOS tab (unlocked by killing him), he counts in "Asesinos vencidos". He never comes to the
// secret island (termFits). What is his own:
//   - slow but relentless: he walks slower than the others (termSpeedK) and needs termHP stabs;
//   - he shoots: a person he sees in front of him within termRange, with a clear line, is aimed at for
//     termAim seconds (termAimP for a player: time to run or hide) and shot dead;
//   - his eyes glow red, and he says his line when he arrives ("Estaré de vuelta.").
// -killeras slug forces that killer (with -killer, or alone: it implies it), e.g. -killeras el_exterminador.
// ---------------------------------------------------------------------------

const (
	termSlug   = "el_exterminador"
	termHP     = 6
	termSpeedK = 0.7  // of the usual killer walk
	termRange  = 90.0 // px: how far he shoots
	termMinR   = 14.0 // closer than this he just hits (the usual strike)
	termAim    = 0.6  // s aiming at a guest or a guard
	termAimP   = 1.1  // s aiming at a player
	termReload = 1.8  // s between shots
)

// termLine: his catchphrase (a parody, not the film line).
const termLine = "Estaré de vuelta."

type termState struct {
	aim    float64 // seconds aiming at the current target
	reload float64
	flashT float64 // muzzle flash on screen
	from   Vec     // last shot, for the tracer
	to     Vec
}

// dbgKillerAs: -killeras forces this killer to come (debug).
var dbgKillerAs string

func termSpec() CharSpec {
	e := def(termSlug, skinc(0xc4c8d2), hairs(HairBald), tops(TopLong, 0x2a2c34, 0x2a2c34), pantsc(0x2a2c34),
		shoesc(0x8e929e), bodyc(BodyBig), stat(-2), weaponc(HoldRifle))
	// a fixed source of its own: the shared random stream of builtinChars (export.go) is not moved
	return e.F(rand.New(rand.NewSource(1984)))
}

func init() {
	killerPresets = append(killerPresets, presetEntry{termSlug, func(*rand.Rand) CharSpec { return termSpec() }})
	for k, v := range map[string]string{
		termLine:                               "I'll be right back.",
		"Rifle":                                "Rifle",
		"El Exterminador te ha disparado":      "The Exterminator shot you",
		"¡El Exterminador! Corre o escóndete.": "The Exterminator! Run or hide.",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

func isTerm(s *Slasher) bool { return s != nil && s.term != nil }

// termFits: may this killer come to this party? The endoskeleton never comes to the island.
func (g *Game) termFits(name string) bool {
	return name != termSlug || g.levelIdx != specialJurassic
}

// termForced: -killeras picks the killer by hand (from the assets folder if it is there).
func (g *Game) termForced() (*CharSet, bool, string, bool) {
	if dbgKillerAs == "" {
		return nil, false, "", false
	}
	if g.as != nil {
		for _, cs := range g.as.killerCS {
			if cs != nil && cs.Spec != nil && cs.Spec.Name == dbgKillerAs {
				return cs, cs.Spec.Weapon == HoldChainsaw || (cs.rig != nil && cs.rig.carry == HoldChainsaw), dbgKillerAs, true
			}
		}
	}
	for _, e := range killerPresets {
		if e.Name == dbgKillerAs {
			sp := e.F(g.rng)
			return newCharSet(&sp), sp.Weapon == HoldChainsaw, e.Name, true
		}
	}
	return nil, false, "", false
}

// termSpawn: the endoskeleton is tougher than the others.
func (g *Game) termSpawn(s *Slasher) {
	if s == nil || s.name != termSlug {
		return
	}
	s.term = &termState{reload: 1}
	s.hp = termHP
	s.chainsaw = false
}

// termArrive: his line when he steps out of the car.
func (g *Game) termArrive(s *Slasher) {
	if !isTerm(s) {
		return
	}
	s.icon, s.iconT = termLine, 2.5
	g.say("¡El Exterminador! Corre o escóndete.")
}

// termSpeed: he walks slower than the other killers (but he never stops).
func termSpeed(s *Slasher, speed float64) float64 {
	if !isTerm(s) {
		return speed
	}
	return speed * termSpeedK
}

// termTarget: where his current target stands (ok false: none).
func (g *Game) termTarget(s *Slasher) (Vec, bool) {
	switch {
	case s.tgtP:
		return g.p.pos, !g.p.dead && !g.p.hidden
	case s.tgtG != nil && !s.tgtG.out():
		return s.tgtG.pos, true
	case s.tgtD != nil && !s.tgtD.dead:
		return s.tgtD.pos, true
	}
	return Vec{}, false
}

// termShoot runs every tick of his hunt: he aims at the target in front of him and shoots it.
// It reports true when he fired (the caller skips the rest of the tick).
func (g *Game) termShoot(s *Slasher, dt float64) bool {
	if !isTerm(s) {
		return false
	}
	t := s.term
	if t.flashT > 0 {
		t.flashT -= dt
	}
	if t.reload > 0 {
		t.reload -= dt
		return false
	}
	tp, ok := g.termTarget(s)
	d := dist(s.pos, tp)
	if !ok || d < termMinR || d > termRange || !g.lvl.LOS(s.pos, tp) || !g.sees(s.pos, s.dir, tp, termRange, 0.3) {
		t.aim = 0
		return false
	}
	need := termAim
	if s.tgtP {
		need = termAimP
	}
	if t.aim += dt; t.aim < need {
		return false
	}
	t.aim, t.reload, t.flashT = 0, termReload, 0.15
	muzzle := Vec{s.pos.X + dirVec(s.dir).X*10, s.pos.Y - 16}
	t.from, t.to = muzzle, Vec{tp.X, tp.Y - 12}
	g.burst(muzzle, 6, 70, rgb(0xffd060), 0.18, 2, 0)
	for i := 1; i <= 6; i++ { // the tracer
		f := float64(i) / 7
		g.burst(Vec{muzzle.X + (t.to.X-muzzle.X)*f, muzzle.Y + (t.to.Y-muzzle.Y)*f}, 1, 6, rgb(0xff8040), 0.12, 1, 0)
	}
	g.snd.Play("boom")
	g.shake = math.Max(g.shake, 0.12)
	s.stabT, s.pause = 0.3, 0.5
	switch {
	case s.tgtP:
		g.die("El Exterminador te ha disparado")
	case s.tgtG != nil:
		v := s.tgtG
		s.tgtG = nil
		g.killGuest(v, false)
		g.sayNews("kill", map[string]string{"actor": slasherActorName(s), "victim": guestLabel(v)})
		g.slasherWitness(s, v.pos)
	case s.tgtD != nil:
		d := s.tgtD
		s.tgtD = nil
		d.dead, d.icon = true, ""
		g.spawnBody(d.pos, d.cs, d.dir)
		g.bodies[len(g.bodies)-1].label = "Guardia " + g.personalName(d, d.cs)
		g.slasherWitness(s, d.pos)
		g.checkDone()
	}
	s.repath = 0
	return true
}

// termDraw: the red eyes glow (brighter while he aims), and the muzzle flash.
func (g *Game) termDraw(dst *ebiten.Image, s *Slasher) {
	if !isTerm(s) || s.state == slGone {
		return
	}
	a := uint8(90 + 50*math.Sin(g.time*5))
	if s.term.aim > 0 {
		a = 200
	}
	g.glowAt(dst, Vec{s.pos.X, s.pos.Y - 35*g.charSc}, 7, color.RGBA{a, 0, 0, a / 2})
	if s.term.flashT > 0 {
		g.glowAt(dst, s.term.from, 10, color.RGBA{255, 200, 80, 160})
	}
}

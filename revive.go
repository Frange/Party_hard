package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Two players: calling the ambulance to bring a dead partner back.
//
// When one player dies the other can walk to a working red phone ("[E] Llamar a la ambulancia"
// instead of cutting the line) and call an ambulance. It comes by the usual way of the level
// (road, helicopter pad or boat: g.dispatch), two paramedics kneel by the body, the
// defibrillator needs a first shock that fails and a second one that works, and the dead
// player gets up again with a funny cry.
//
// Rules (documented here because the user asked for "only once"):
//   - ONE call per PARTY (per level): g.rev.used. A second death in the same party stays dead.
//   - the HUD shows "Ambulancia: 1" while the call is available and "Ambulancia en N s" while it comes.
//   - the call needs a working phone: a phone whose line was cut cannot be used (pr.used).
//   - deaths that leave nothing to revive: the T-Rex, fire, explosions, zombie bites and the police
//     (reviveBlocked). Anything else (killers, traps, cars, a piano, electricity...) can be revived.
//   - calling the ambulance does not call the police and the paramedics are not witnesses.
//   - if the caller dies too before the paramedics arrive, the party is lost as always.
// ---------------------------------------------------------------------------

const (
	revDelayMin = 4.0 // seconds between the call and the ambulance leaving
	revDelayVar = 3.0
)

// phases of a rescue
const (
	rvWait    = iota // the call has been made, the ambulance is not on its way yet
	rvDrive          // it is coming
	rvWalk           // the paramedics walk to the body
	rvCheck          // kneeling, looking for a pulse (flat line)
	rvCharge         // the defibrillator charges
	rvShock1         // first shock: nothing
	rvCharge2        // charge again
	rvShock2         // second shock: it works
	rvUp             // the partner sits up
	rvBack           // the paramedics go back to the ambulance
	rvDone
)

type reviveState struct {
	used   bool // the call of this party has been made
	done   int  // revivals in this party
	job    *reviveJob
	retryT float64 // no free pad yet: try again
}

type reviveJob struct {
	target, caller *Player
	phase          int
	t              float64 // time since the call
	delay          float64
	eta            float64 // estimated seconds until the paramedics are at the body (HUD)
	v              *Vehicle
	crew           []*Paramedic
	timer          float64
	beepT          float64
	say            string // what the paramedic shouts now
	sayT           float64
	flash          float64
}

// Paramedic is one of the two people of the rescue.
type Paramedic struct {
	pos    Vec
	dir    Dir
	cs     *CharSet
	path   []Vec
	anim   float64
	moving bool
	curSpd float64
	role   int // 0 holds the defibrillator, 1 helps
	kneel  bool
	spot   Vec // where he stands by the body
}

// dbgTwoDead is set by -twodead N: player N (1 or 2) starts dead and the other one calls the ambulance by himself.
var dbgTwoDead int

// reviveBlocked: deaths that cannot be undone by the paramedics.
func reviveBlocked(why string) bool {
	for _, w := range []string{"T-Rex", "achicharrado", "por los aires", "cohete", "zombi", "atrapado"} {
		if strings.Contains(why, w) {
			return true
		}
	}
	return false
}

// reviveTarget is the dead partner that the ambulance could bring back (nil if none).
func (g *Game) reviveTarget() *Player {
	if !g.is2P() {
		return nil
	}
	for _, pl := range g.players() {
		if pl.dead && !reviveBlocked(pl.deathWhy) {
			return pl
		}
	}
	return nil
}

// canCallAmbulance: may this player call the ambulance right now?
func (g *Game) canCallAmbulance(p *Player) bool {
	if p == nil || !g.is2P() || p.dead || p.knockT > 0 || p.carrying != nil || p.hidden {
		return false
	}
	if g.rev.used || g.rev.job != nil || g.mode != ModePlay {
		return false
	}
	t := g.reviveTarget()
	return t != nil && t != p
}

// callAmbulance: the player at the phone calls; the ambulance leaves after a few seconds.
func (g *Game) callAmbulance(p *Player, pr *Prop) {
	t := g.reviveTarget()
	if t == nil || g.rev.used {
		return
	}
	g.rev.used = true
	g.reviveSounds()
	job := &reviveJob{target: t, caller: p, delay: revDelayMin + g.rng.Float64()*revDelayVar}
	job.eta = job.delay + g.reviveTravel()
	g.rev.job = job
	g.snd.Play("ring")
	g.say("Llamas a la ambulancia: ¡aguanta!")
	g.stat("ambulance_calls", 1)
	if pr != nil {
		g.burst(Vec{pr.X, pr.Y - 24}, 8, 40, rgb(0x60ff80), 0.5, 2, 0)
	}
}

// reviveTravel: a rough number of seconds the ambulance takes to arrive (for the HUD countdown).
func (g *Game) reviveTravel() float64 {
	def := g.lvl.Def
	switch {
	case def.Police == "heli":
		return 5
	case def.Transit != "":
		return 8
	}
	return math.Abs(def.CarTo.X-def.CarFrom.X)/130 + 1
}

// dispatchRevive sends the ambulance with the usual arrival of the level. Returns false when it
// could not leave yet (every helicopter pad busy): it is tried again a second later.
func (g *Game) dispatchRevive(job *reviveJob) bool {
	before, pending := len(g.veh), g.pendingAmb
	g.dispatch(true)
	if len(g.veh) > before {
		v := g.veh[len(g.veh)-1]
		v.revive = true
		job.v = v
		job.phase = rvDrive
		return true
	}
	g.pendingAmb = pending // dispatch queued an ordinary paramedic: cancel it, ours retries
	return false
}

// reviveArrives: the ambulance has parked (called from updateVehicles): the paramedics get out.
func (g *Game) reviveArrives(v *Vehicle) {
	job := g.rev.job
	if job == nil || job.v != v {
		v.state = 2 // nobody to rescue any more
		return
	}
	var pos Vec
	if v.isHeli() {
		pos = g.lvl.NearestFree(Vec{v.tx + 14, v.ty + 22})
	} else {
		pos = g.lvl.NearestFree(Vec{v.x - 12, v.y - 24})
	}
	cs := g.as.medicCS
	if cs == nil {
		sp := medicSpec()
		cs = newCharSet(&sp)
	}
	body := job.target.pos
	spots := []Vec{{body.X - 13, body.Y + 2}, {body.X + 13, body.Y + 2}}
	for i := 0; i < 2; i++ {
		m := &Paramedic{pos: g.lvl.NearestFree(Vec{pos.X + float64(i)*10, pos.Y - float64(i)*6}), dir: DirDown, cs: cs, role: i}
		m.spot = g.lvl.NearestFree(spots[i])
		m.path = g.lvl.Path(m.pos, m.spot)
		job.crew = append(job.crew, m)
	}
	job.phase = rvWalk
	g.say("Llega la ambulancia")
}

// reviveSounds registers the two sounds of the rescue the first time they are needed.
func (g *Game) reviveSounds() {
	if g.snd == nil || g.snd.sfx == nil {
		return
	}
	if _, ok := g.snd.sfx["monitor"]; ok {
		return
	}
	g.snd.sfx["monitor"] = mix(tone(1000, 0.16, 0.4, 5, 0))
	g.snd.sfx["flatline"] = mix(tone(1000, 0.9, 0.35, 0.4, 0))
	g.snd.sfx["charge"] = mix(tone(300, 0.5, 0.3, 0.4, 1), shift(tone(600, 0.5, 0.3, 0.4, 1), 0.1), shift(tone(1200, 0.4, 0.25, 0.6, 1), 0.2))
	g.snd.sfx["defib"] = mix(noise(0.25, 0.9, 7), tone(160, 0.3, 0.6, 5, 2), tone(2400, 0.12, 0.3, 14, 1))
}

// updateRevive runs the whole rescue; it is called every frame of a two-player party.
func (g *Game) updateRevive(dt float64) {
	if g.lvl == nil || g.mode != ModePlay {
		return
	}
	g.reviveDebugTick()
	job := g.rev.job
	if job == nil {
		return
	}
	job.t += dt
	job.flash = math.Max(0, job.flash-dt*4)
	if job.sayT > 0 {
		job.sayT -= dt
	}
	// the one that was to be rescued is not what he was (the party restarted...): drop the rescue
	if !job.target.dead && job.phase < rvUp {
		g.reviveEnd(job, false)
		return
	}
	switch job.phase {
	case rvWait:
		if job.t >= job.delay {
			if !g.dispatchRevive(job) {
				job.delay = job.t + 1 // no pad free: one more second
			}
		}
	case rvDrive:
		if job.v != nil && job.v.dead {
			g.reviveEnd(job, false)
		}
	case rvWalk:
		all := true
		for _, m := range job.crew {
			m.moving = false
			if len(m.path) > 0 {
				mv, _ := follow(&m.pos, &m.dir, &m.path, 80*paceK, dt)
				m.moving, m.curSpd = mv, 80
				if mv {
					m.anim += dt * stridePhase(80)
				}
				all = false
			}
		}
		if all || job.t > job.delay+40 {
			job.phase, job.timer = rvCheck, 1.8
			for _, m := range job.crew {
				m.kneel = true
				m.dir = DirUp
				if m.role == 1 {
					m.dir = DirLeft
				}
			}
			job.say, job.sayT = "¿Me oye? ¿Hola?", 1.6
			g.snd.Play("flatline")
		}
	case rvCheck:
		if job.timer -= dt; job.timer <= 0 {
			job.phase, job.timer = rvCharge, 1.3
			job.say, job.sayT = "¡CARGANDO!", 1.2
			g.snd.Play("charge")
		}
	case rvCharge:
		if job.timer -= dt; job.timer <= 0 {
			job.phase, job.timer = rvShock1, 0.9
			job.say, job.sayT = "¡CLEAR!", 0.9
			job.flash = 1
			g.shock(job)
			g.snd.Play("defib")
		}
	case rvShock1:
		if job.timer -= dt; job.timer <= 0 {
			job.phase, job.timer = rvCharge2, 1.2
			job.say, job.sayT = "Nada... otra vez", 1.3
			g.snd.Play("charge")
		}
	case rvCharge2:
		if job.timer -= dt; job.timer <= 0 {
			job.phase, job.timer = rvShock2, 0.8
			job.say, job.sayT = "¡CLEAR!", 0.8
			job.flash = 1
			g.shock(job)
			g.snd.Play("defib")
		}
	case rvShock2:
		if job.timer -= dt; job.timer <= 0 {
			g.reviveNow(job)
			job.phase, job.timer = rvUp, 2.2
		}
	case rvUp:
		for _, m := range job.crew {
			m.kneel = false
		}
		if job.timer -= dt; job.timer <= 0 {
			job.phase = rvBack
			for _, m := range job.crew {
				if job.v != nil {
					m.path = g.lvl.Path(m.pos, g.lvl.NearestFree(Vec{job.v.x - 12, job.v.y - 24}))
				}
			}
		}
	case rvBack:
		gone := 0
		for _, m := range job.crew {
			m.moving = false
			if len(m.path) > 0 {
				mv, _ := follow(&m.pos, &m.dir, &m.path, 80*paceK, dt)
				m.moving, m.curSpd = mv, 80
				if mv {
					m.anim += dt * stridePhase(80)
				}
			}
			if len(m.path) == 0 || (job.v != nil && dist(m.pos, Vec{job.v.x - 12, job.v.y - 24}) < 16) {
				gone++
			}
		}
		if gone == len(job.crew) || job.t > job.delay+80 {
			g.reviveEnd(job, true)
		}
	}
	// the monitor of the defibrillator beeps while they work on him
	if job.phase >= rvCheck && job.phase <= rvShock2 {
		if job.beepT -= dt; job.beepT <= 0 {
			job.beepT = 0.55
			if job.phase == rvCheck {
				g.snd.Play("flatline")
				job.beepT = 1.1
			}
		}
	}
}

// shock: sparks and a jolt of the body.
func (g *Game) shock(job *reviveJob) {
	b := job.target.pos
	g.burst(Vec{b.X, b.Y - 6}, 18, 90, rgb(0xfff4a0), 0.5, 2, 60)
	g.burst(Vec{b.X, b.Y - 6}, 10, 60, rgb(0x80c8ff), 0.4, 2, 40)
	g.shake = math.Max(g.shake, 0.1)
}

// reviveNow: the partner is alive again, at the place where he fell, with full life.
func (g *Game) reviveNow(job *reviveJob) {
	t := job.target
	t.dead = false
	t.deathWhy = ""
	t.pos = g.lvl.NearestFree(t.pos)
	t.knockT = 1.2 // he sits up slowly
	t.hidden, t.carrying = false, nil
	t.msgT, t.msg = 2.8, "¿QUÉ ME HE PERDIDO?"
	t.immT = 2
	g.rev.done++
	g.stat("revives", 1)
	g.snd.Play("win")
	g.burst(Vec{t.pos.X, t.pos.Y - 14}, 16, 70, rgb(0x70ff90), 0.7, 2, -20)
	g.say("¡Tu compañero ha vuelto de entre los muertos!")
	job.say, job.sayT = "¡Respira! ¡Respira!", 1.6
}

// reviveEnd closes the rescue and sends the ambulance away.
func (g *Game) reviveEnd(job *reviveJob, ok bool) {
	if job.v != nil && job.v.state != 2 {
		job.v.state = 2
	}
	g.rev.job = nil
	if !ok && job.target != nil && job.target.dead {
		g.say("La ambulancia se va sin nadie al que salvar")
	}
}

// reviveDebugTick: with -twodead the first living player makes the call by himself after a moment.
func (g *Game) reviveDebugTick() {
	if dbgTwoDead == 0 || g.rev.used || g.time < 1.5 {
		return
	}
	for _, pl := range g.players() {
		if !pl.dead {
			g.callAmbulance(pl, nil)
			return
		}
	}
}

// reviveDebugSetup: -twodead 2 kills player 2 when the party starts (1 kills the first one).
func (g *Game) reviveDebugSetup() {
	if dbgTwoDead == 0 || !g.is2P() {
		return
	}
	pl := g.p2
	if dbgTwoDead == 1 {
		pl = g.first()
	}
	pl.dead, pl.deathWhy = true, "debug"
}

// ---- drawing --------------------------------------------------------------------------

// reviveItems adds the paramedics to the sorted list of things drawn.
func (g *Game) reviveItems(dst *ebiten.Image, items []drawItem) []drawItem {
	job := g.rev.job
	if job == nil {
		return items
	}
	for _, m := range job.crew {
		m := m
		items = append(items, drawItem{m.pos.Y, func() { g.drawParamedic(dst, m, job) }})
	}
	return items
}

func (g *Game) drawParamedic(dst *ebiten.Image, m *Paramedic, job *reviveJob) {
	anim, phase := AnimIdle, g.time*1.2+m.pos.X
	switch {
	case m.moving:
		anim, phase = AnimWalk, m.anim
	case m.kneel && m.role == 0:
		anim, phase = AnimCrouch, g.time*3
	case m.kneel:
		anim, phase = AnimPoint, g.time*2
	}
	if m.cs != nil {
		g.drawActor(dst, m.cs, anim, phase, m.pos, m.dir, 0, false)
	}
	x, y := g.wx(m.pos.X), g.wy(m.pos.Y)
	// the paddles in his hands while he works on the body
	if m.kneel && m.role == 0 && job.phase >= rvCharge && job.phase <= rvShock2 {
		pw := rgb(0xe8e8f0)
		fillRect(dst, x-6, y-16, 4, 3, pw)
		fillRect(dst, x+3, y-16, 4, 3, pw)
		if job.flash > 0.2 {
			c := color.RGBA{255, 244, 160, uint8(255 * job.flash)}
			bx, by := g.wx(job.target.pos.X), g.wy(job.target.pos.Y)
			for i := 0; i < 5; i++ {
				fillRect(dst, bx-8+float64(i)*4+math.Sin(g.time*60+float64(i))*2, by-8-math.Cos(g.time*50+float64(i))*3, 2, 2, c)
			}
		}
	}
	if m.role == 0 && job.sayT > 0 && job.say != "" {
		g.drawParamedicSay(dst, job.say, x, y-52)
	}
}

// drawParamedicSay: a small box with a shout over the paramedic (not a guest bubble: always at most one).
func (g *Game) drawParamedicSay(dst *ebiten.Image, s string, cx, y float64) {
	w := math.Round(g.textW(s, 1)) + 8
	x := clampF(cx-w/2, 2, screenW-w-2)
	fillRect(dst, x, y, w, 13, rgb(0xf4f4f8))
	fillRect(dst, x+1, y+1, w-2, 11, rgb(0x2a8a4a))
	g.drawText(dst, s, x+4, y+1, rgb(0xffffff), 1)
	fillRect(dst, cx-2, y+13, 4, 1, rgb(0xf4f4f8))
}

// drawReviveHud: "Ambulancia: 1" while the call is available, a countdown while it comes.
func (g *Game) drawReviveHud(dst *ebiten.Image) {
	if g.mode != ModePlay || !g.is2P() {
		return
	}
	col := rgb(0x70e0a0)
	txt := "Ambulancia: 1"
	switch job := g.rev.job; {
	case job != nil && job.phase < rvWalk:
		left := math.Max(0, job.eta-job.t)
		txt = fmt.Sprintf("Ambulancia en %d s", int(math.Ceil(left)))
		col = rgb(0xffe060)
	case job != nil:
		txt, col = "Reanimando...", rgb(0xffe060)
	case g.rev.used:
		txt, col = "Ambulancia: 0", rgb(0x7a7a90)
	}
	_, _, ry, _ := g.hudSide()
	g.drawText(dst, txt, hudX, ry, col, 1)
}

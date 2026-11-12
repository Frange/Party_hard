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
// Rules (the user, 2026-11-10: "once per player and per party"):
//   - each PLAYER can be brought back ONCE per party (g.rev.spent, by player index, only in memory: the
//     saved game never changes). A second death of the same player in the same party stays dead; the
//     other player still has his own ambulance. Loading a party resets it (setupPlayers).
//   - the HUD shows "Ambulancia: J1 1 · J2 1" (what is left for each one), "Ambulancia en N s" while it
//     comes, and a box "Tu compañero J2 ha muerto / Llama a la ambulancia en un teléfono [E]" while the dead
//     partner can still be saved (drawReviveNotice), with "AMBULANCIA" over the nearest working phone.
//   - from the moment the paramedics reach the body to the partner getting up there are revWorkT (10) seconds.
//   - the call needs a working phone: a phone whose line was cut cannot be used (pr.used).
//   - deaths that leave nothing to revive: the T-Rex, fire, explosions, zombie bites and the police
//     (reviveBlocked). Anything else (killers, traps, cars, a piano, electricity...) can be revived.
//   - calling the ambulance does not call the police and the paramedics are not witnesses.
//   - if the caller dies too before the paramedics arrive, the party is lost as always.
// ---------------------------------------------------------------------------

const (
	revDelayMin = 4.0 // seconds between the call and the ambulance leaving
	revDelayVar = 3.0

	// the work by the body (the user: "he gets his life back some 10 s after the paramedics reach him");
	// the first shock still fails and the second one works
	revCheckT   = 3.0 // kneeling, looking for a pulse
	revChargeT  = 2.0 // the defibrillator charges
	revShock1T  = 1.5 // first shock: nothing
	revCharge2T = 2.0 // charge again
	revShock2T  = 1.5 // second shock: he breathes
	revWorkT    = revCheckT + revChargeT + revShock1T + revCharge2T + revShock2T
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
	spent  [2]bool // by player index: the ambulance already came (or is coming) for him in this party
	done   int     // revivals in this party
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

// reviveSpent: has this player already had his ambulance in this party?
func (g *Game) reviveSpent(p *Player) bool { return p != nil && g.rev.spent[p.idx&1] }

// reviveLeft: how many ambulances this player still has in this party (0 or 1).
func (g *Game) reviveLeft(p *Player) int {
	if g.reviveSpent(p) {
		return 0
	}
	return 1
}

// reviveTarget is the dead partner that the ambulance could bring back (nil if none).
func (g *Game) reviveTarget() *Player {
	if !g.is2P() {
		return nil
	}
	for _, pl := range g.players() {
		if pl.dead && !reviveBlocked(pl.deathWhy) && !g.reviveSpent(pl) {
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
	if g.rev.job != nil || g.mode != ModePlay {
		return false
	}
	t := g.reviveTarget()
	return t != nil && t != p
}

// callAmbulance: the player at the phone calls; the ambulance leaves after a few seconds.
func (g *Game) callAmbulance(p *Player, pr *Prop) {
	t := g.reviveTarget()
	if t == nil || t == p || g.rev.job != nil {
		return
	}
	g.rev.spent[t.idx&1] = true
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
	// the one that was to be rescued is not what he was (the party restarted...): drop the rescue.
	// Still on the way: it turns back. Already out: the crew stands up and walks back to the
	// ambulance (before, they vanished on the spot and it drove away: 2026-11-11).
	if !job.target.dead && job.phase < rvUp {
		if job.phase < rvWalk || len(job.crew) == 0 {
			g.reviveEnd(job, false)
			return
		}
		job.phase, job.timer = rvUp, 1.2
		job.say, job.sayT = "", 0
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
			job.phase, job.timer = rvCheck, revCheckT
			for _, m := range job.crew {
				m.kneel = true
				m.dir = DirUp
				if m.role == 1 {
					m.dir = DirLeft
				}
			}
			job.say, job.sayT = "¿Me oye? ¿Hola?", 2.2
			g.snd.Play("flatline")
		}
	case rvCheck:
		if job.timer -= dt; job.timer <= 0 {
			job.phase, job.timer = rvCharge, revChargeT
			job.say, job.sayT = "¡CARGANDO!", 1.6
			g.snd.Play("charge")
		}
	case rvCharge:
		if job.timer -= dt; job.timer <= 0 {
			job.phase, job.timer = rvShock1, revShock1T
			job.say, job.sayT = "¡CLEAR!", 1.0
			job.flash = 1
			g.shock(job)
			g.snd.Play("defib")
		}
	case rvShock1:
		if job.timer -= dt; job.timer <= 0 {
			job.phase, job.timer = rvCharge2, revCharge2T
			job.say, job.sayT = "Nada... otra vez", 1.6
			g.snd.Play("charge")
		}
	case rvCharge2:
		if job.timer -= dt; job.timer <= 0 {
			job.phase, job.timer = rvShock2, revShock2T
			job.say, job.sayT = "¡CLEAR!", 1.0
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

// reviveDebugTick: with -twodead 1|2 the first living player makes the call by himself after a moment
// (3..6 leave the call to the player, to see the notice and the phone).
func (g *Game) reviveDebugTick() {
	if dbgTwoDead == 0 || dbgTwoDead > 2 || g.rev.job != nil || g.reviveTarget() == nil || g.time < 1.5 {
		return
	}
	for _, pl := range g.players() {
		if !pl.dead {
			g.callAmbulance(pl, nil)
			return
		}
	}
}

// reviveDebugSetup: -twodead 2 kills player 2 when the party starts (1 kills the first one). 3 and 4 do the
// same but nobody calls (the notice stays on screen); 5 and 6 also put the living one by the nearest phone.
func (g *Game) reviveDebugSetup() {
	if dbgTwoDead == 0 || !g.is2P() {
		return
	}
	pl, other := g.p2, g.first()
	if dbgTwoDead%2 == 1 {
		pl, other = g.first(), g.p2
	}
	pl.dead, pl.deathWhy = true, "debug"
	if dbgTwoDead >= 5 {
		if ph := g.nearestAmbulancePhone(other); ph != nil {
			other.pos = g.lvl.NearestFree(Vec{ph.Foot.X + ph.Foot.W/2, ph.Foot.Y + ph.Foot.H + 4})
		}
	}
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

// drawReviveHud: what is left of each player's ambulance ("Ambulancia: J1 1 · J2 1"), a countdown while it
// comes, the notice box while a dead partner can still be saved, and a short grey line when he cannot.
func (g *Game) drawReviveHud(dst *ebiten.Image) {
	if g.mode != ModePlay || !g.is2P() {
		return
	}
	_, _, ry, _ := g.hudSide()
	if b, ok := g.reviveNoticeBox(ry); ok {
		// the box follows the fade of the HUD panel (hudhide.go); the label over the phone belongs to the map
		g.withHudFade(dst, func(d *ebiten.Image) { g.drawReviveNotice(d, b) })
		g.drawRevivePhoneLabel(dst, b.caller)
		return
	}
	line, col := g.reviveHudLine(), g.reviveHudColor()
	g.withHudFade(dst, func(d *ebiten.Image) { g.drawText(d, line, hudX, ry, col, 1) })
}

// reviveHudLine: the one-line text of the ambulance row of the HUD (no notice box on screen).
func (g *Game) reviveHudLine() string {
	if job := g.rev.job; job != nil {
		if job.phase < rvWalk {
			left := math.Max(0, job.eta-job.t)
			return fmt.Sprintf("Ambulancia en %d s", int(math.Ceil(left)))
		}
		return fmt.Sprintf("Reanimando a %s...", g.playerName(job.target))
	}
	for _, pl := range g.players() {
		if !pl.dead {
			continue
		}
		if reviveBlocked(pl.deathWhy) { // the T-Rex, fire, an explosion, a zombie, the police
			return fmt.Sprintf("%s no se puede reanimar", g.playerName(pl))
		}
		if g.reviveSpent(pl) { // he already had his ambulance in this party
			return fmt.Sprintf("%s ya gastó su ambulancia", g.playerName(pl))
		}
	}
	a, b := g.first(), g.p2
	return fmt.Sprintf("Ambulancia: %s %d · %s %d", g.playerName(a), g.reviveLeft(a), g.playerName(b), g.reviveLeft(b))
}

func (g *Game) reviveHudColor() color.RGBA {
	if g.rev.job != nil {
		return rgb(0xffe060)
	}
	for _, pl := range g.players() {
		if pl.dead {
			return rgb(0x8a8aa0)
		}
	}
	if g.reviveLeft(g.first())+g.reviveLeft(g.p2) == 0 {
		return rgb(0x7a7a90)
	}
	return rgb(0x70e0a0)
}

// reviveHudH: the room the ambulance row takes in the column under the HUD panel (hudSide).
func (g *Game) reviveHudH() float64 {
	if b, ok := g.reviveNoticeBox(0); ok {
		return b.h + hudSideGap
	}
	return 14
}

// reviveNoticeWho: the dead partner who can still be saved and the living one who must call (nil, nil when
// there is nothing to say: one player, nobody dead, the ambulance already coming, a death without remedy...).
func (g *Game) reviveNoticeWho() (dead, caller *Player) {
	if g.mode != ModePlay || !g.is2P() || g.rev.job != nil {
		return nil, nil
	}
	t := g.reviveTarget()
	if t == nil {
		return nil, nil
	}
	for _, pl := range g.players() {
		if pl != t && !pl.dead {
			return t, pl
		}
	}
	return nil, nil
}

// the notice box: >= 6 px of padding around the text, 4 px between its two lines
const (
	revNotePad  = 6.0
	revNoteGap  = 4.0
	revNoteMark = 6.0 // the square in the colour of the dead player
)

type reviveNote struct {
	x, y, w, h     float64
	l1, l2         string
	tx1, ty1, ty2  float64 // where the lines are written (the second one starts at x+pad)
	dead, caller   *Player
	lineH, markGap float64
}

// reviveNoticeBox lays out the notice "Tu compañero J2 ha muerto / Llama a la ambulancia en un teléfono [E]"
// at the top-left corner (hudX, y); ok is false when there is no notice to show.
func (g *Game) reviveNoticeBox(y float64) (b reviveNote, ok bool) {
	dead, caller := g.reviveNoticeWho()
	if dead == nil || g.face == nil { // no font: a game without window (some tests) has nothing to lay out
		return b, false
	}
	b.dead, b.caller = dead, caller
	b.l1 = fmt.Sprintf("Tu compañero %s ha muerto", g.playerName(dead))
	b.l2 = fmt.Sprintf("Llama a la ambulancia en un teléfono [%s]", actionHintFor(actInteract, caller.idx))
	_, lh := textMeasure(g, tr(b.l1))
	b.lineH = math.Ceil(lh)
	b.markGap = revNoteMark + 4
	w1 := math.Ceil(g.textW(b.l1, 1)) + b.markGap
	w2 := math.Ceil(g.textW(b.l2, 1))
	b.x, b.y = hudX, math.Round(y)
	b.w = math.Max(w1, w2) + 2*revNotePad
	b.h = 2*revNotePad + 2*b.lineH + revNoteGap
	b.tx1 = b.x + revNotePad + b.markGap
	b.ty1 = b.y + revNotePad
	b.ty2 = b.ty1 + b.lineH + revNoteGap
	return b, true
}

func (g *Game) drawReviveNotice(dst *ebiten.Image, b reviveNote) {
	g.panel(dst, b.x, b.y, b.w, b.h)
	// a green edge that breathes, so that the eye goes there without a flashing banner
	a := uint8(150 + 100*math.Abs(math.Sin(g.time*3)))
	fillRect(dst, b.x, b.y, 2, b.h, color.RGBA{96, 255, 160, a})
	c := ringColor(b.dead)
	fillRect(dst, b.x+revNotePad, math.Round(b.ty1+b.lineH/2-revNoteMark/2), revNoteMark, revNoteMark, color.RGBA{c[0], c[1], c[2], 255})
	g.drawText(dst, b.l1, b.tx1, b.ty1, rgb(0xffffff), 1)
	g.drawText(dst, b.l2, b.x+revNotePad, b.ty2, rgb(0x60ffa0), 1)
}

// nearestAmbulancePhone: the working phone booth (the red one, "phone") closest to p, nil if none.
func (g *Game) nearestAmbulancePhone(p *Player) *Prop {
	if g.lvl == nil || p == nil {
		return nil
	}
	var best *Prop
	bd := math.Inf(1)
	for _, pr := range g.lvl.Props {
		if !pr.K.Phone || pr.used || pr.gone {
			continue
		}
		if d := distRect(p.pos, pr.Foot); d < bd {
			best, bd = pr, d
		}
	}
	return best
}

// revPromptNear: within this distance of a phone the use prompt "[E] Llamar a la ambulancia" replaces the label.
const revPromptNear = 30.0

// revHudReserved: the top-left corner taken by the HUD panel and the column under it (with the notice box): the
// boxes over a phone keep 12 px away from it.
func (g *Game) revHudReserved() (w, h float64) {
	w, h = hudX+hudW, g.hud.bottom
	_, _, ry, _ := g.hudSide()
	if b, ok := g.reviveNoticeBox(ry); ok {
		w, h = math.Max(w, b.x+b.w), b.y+b.h
	}
	return w + hudSideGap, h + hudSideGap
}

func revFree(x, y, w, h, rw, rh float64) bool {
	return y >= 16 && y+h <= screenH-16 && (x >= rw || y >= rh)
}

// revPhoneBox: a box with text s centred over the green badge of phone pr (drawPhoneMarkers), 12 px above its
// highest bob, >= 16 px from the screen edges and 12 px from the HUD. The x of the text is tx (there is room
// for a mark before it). When it does not fit over the phone (a phone under the HUD, at the top of the map) it
// goes 12 px under the feet of the player p, and if that is not free either, under the HUD column. ok is false
// when no place fits (only possible when below is nil: the label does not move away from its phone).
func (g *Game) revPhoneBox(pr *Prop, s string, mark float64, below *Player) (x, y, w, h, tx float64, ok bool) {
	_, lh := textMeasure(g, tr(s))
	lh = math.Ceil(lh)
	w = math.Ceil(g.textW(s, 1)) + 2*revNotePad + mark
	h = lh + 2*revNotePad
	badgeTop := math.Round(g.wy(pr.Y)-60) - 2 // drawPhoneMarkers bobs it 2 px up and down
	x = math.Round(clampF(math.Round(g.wx(pr.X))-w/2, 16, screenW-16-w))
	rw, rh := g.revHudReserved()
	y = badgeTop - hudSideGap - h
	ok = revFree(x, y, w, h, rw, rh)
	if !ok && below != nil {
		y = math.Round(math.Max(g.wy(pr.Foot.Y+pr.Foot.H), g.wy(below.pos.Y))) + hudSideGap
		if !revFree(x, y, w, h, rw, rh) {
			y = math.Max(y, rh)
		}
		y, ok = math.Round(clampF(y, 16, screenH-16-h)), true
	}
	return x, y, w, h, x + revNotePad + mark, ok
}

// drawRevivePhoneLabel: "AMBULANCIA" over the nearest working phone while the partner can be saved (the use
// prompt takes its place when the caller is next to it).
func (g *Game) drawRevivePhoneLabel(dst *ebiten.Image, caller *Player) {
	pr := g.nearestAmbulancePhone(caller)
	if pr == nil || distRect(caller.pos, pr.Foot) < revPromptNear {
		return
	}
	sx, sy := g.wx(pr.X), g.wy(pr.Y)
	if sx < 0 || sx > screenW || sy < 0 || sy > screenH+40 {
		return
	}
	x, y, w, h, tx, ok := g.revPhoneBox(pr, "AMBULANCIA", 0, nil)
	if !ok {
		return
	}
	g.panel(dst, x, y, w, h)
	g.drawText(dst, "AMBULANCIA", tx, y+revNotePad, rgb(0x60ffa0), 1)
}

// drawAmbulancePrompt: the use prompt of a phone when it calls the ambulance, in a box over the phone so that it
// cannot be missed (the small square says whose key it is).
func (g *Game) drawAmbulancePrompt(dst *ebiten.Image, pr *Prop, txt string, pl *Player) {
	x, y, w, h, tx, _ := g.revPhoneBox(pr, txt, 8, pl)
	g.panel(dst, x, y, w, h)
	a := uint8(150 + 100*math.Abs(math.Sin(g.time*3)))
	fillRect(dst, x, y, 2, h, color.RGBA{96, 255, 160, a})
	fillRect(dst, x+revNotePad, math.Round(y+h/2-2), 4, 4, playerColor(pl.idx))
	g.drawText(dst, txt, tx, y+revNotePad, rgb(0x60ffa0), 1)
}

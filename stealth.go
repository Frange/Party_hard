package main

import (
	"fmt"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// STEALTH mode ("modo SIGILO"): its own mode, apart from the normal game. Only here do these rules apply
// (g.stealth is false everywhere else and every hook starts with that check).
//
// ENTRY POINT (one place, easy to move): startStealth(levelIdx). Today it is reached from the DEBUG menu
// (row stealthMenuLabel, debugmode.go) and from the flag -stealth [-level N] (main.go). To put it in the
// main menu later, just call g.startStealth(i) from there.
//
// RULES
//   - The party is won as usual (nobody left alive) but UNSEEN: if the player was never detected the end
//     panel says "SIN SER VISTO" and a toast shows the badge (it is not saved: the progress is frozen).
//   - DETECTED = somebody has the player inside their vision cone (guests, guards and officers: the same
//     ranges as the normal game) for sthSeeTime seconds, or somebody saw a kill (suspectSeenBy).
//   - Phases after a detection: RED ("ALERTA", sthRedTime s): the police are called and look for him
//     non-stop (g.wanted is held up); being seen again resets the timer. Then YELLOW ("PRECAUCION",
//     sthYellowTime s): the officers keep working but the chase is over; being seen again goes back to
//     red. Then calm again.
//   - THE CARDBOARD BOX (E): with it on, nobody detects the player (guests, guards, officers) and he
//     walks at sthBoxSpeed of his speed and cannot attack. It fails (the box is blown, red phase) when
//     somebody BUMPS into him (anybody closer than sthBoxBump px) or when he stands still for
//     sthBoxStill s in front of an officer or a guard closer than sthBoxNear px. It cannot be put on
//     during the red phase.
//   - Facilitation (only in this mode): the party has sthGuests guests instead of 45-60 (more space),
//     sthHedges extra short hedges that make corridors (stealthDef: a copy of the level, the original is
//     untouched), no random events or killers, and the vision cones of guards and officers are drawn.
//   - One player. With the 2 players switch on, the second player is simply dropped for this party (the
//     switch itself is not touched, the next normal party has two again).
// ---------------------------------------------------------------------------

const (
	sthCalm = iota
	sthRed
	sthYellow
)

const (
	sthRedTime    = 12.0  // seconds of ALERT
	sthYellowTime = 8.0   // seconds of CAUTION
	sthSeeTime    = 0.45  // seconds inside a cone before it counts as detected
	sthBoxSpeed   = 0.5   // speed under the box
	sthBoxBump    = 11.0  // closer than this somebody bumps into the box
	sthBoxNear    = 60.0  // an officer / guard closer than this and in front...
	sthBoxStill   = 2.0   // ...and the player standing still this long: the box is blown
	sthGuests     = 24    // guests of a stealth party
	sthHedges     = 7     // short hedges added to make corridors
	sthNoiseNear  = 130.0 // 2 players: the decoy needs an officer or guard this close to make the noise
	sthNoiseThrow = 90.0  // ...and the noise falls this far in front of him
	sthNoiseRange = 330.0 // guards and officers this close to the noise go to it
	sthNoiseTime  = 6.0   // seconds they stay on it
	sthNoiseCD    = 4.0   // seconds before the same player can make another one
)

const stealthMenuLabel = "Modo SIGILO: ganar sin ser visto (caja de cartón)"

// sthPlayer: what is kept per player (the detection is per player, the phase panel is shared).
type sthPlayer struct {
	meter float64 // 0..1: how long somebody has been looking at him
	box   bool    // wearing the cardboard box
	still float64 // seconds standing still in the box in front of an authority
}

type stealthState struct {
	sthPlayer  // player 1 (the only one in the normal mode): s.meter, s.box, s.still
	phase      int
	t          float64 // seconds left of the red / yellow phase
	detections int     // times detected in this party (both players)
	exT        float64 // seconds left of the big "!"
	exMax      float64
	boxBlown   float64 // seconds left of the "box blown" flash

	// 2 players (-stealth2p, DEBUG): player 2's own state, who caused each detection, and the decoy noise
	two      sthPlayer
	by       [2]int // detections caused by each player
	culprit  int    // who caused the last detection (0 / 1)
	whistles [2]int // noises made by each player
	cd       [2]float64
	noiseT   float64 // seconds left of the noise that guards and officers walk to
	noisePos Vec
	noiseAt  float64 // timer to send them again
	noiseFx  float64 // seconds of the ring drawn where the noise fell
}

// of: the per-player state (0 = player 1, 1 = player 2).
func (s *stealthState) of(i int) *sthPlayer {
	if i == 1 {
		return &s.two
	}
	return &s.sthPlayer
}

// ---- entry / exit -------------------------------------------------------------------------------

// stealthBegin runs at the start of loadLevel: the mode is on only for the level that asked for it.
func (g *Game) stealthBegin() {
	g.stealth, g.stealthNext = g.stealthNext, false
	g.stealth2, g.stealth2Next = g.stealth && g.stealth2Next, false
	g.sth = stealthState{}
}

// startStealth: the single entry point of the mode (level index into levelDefs). The caller has frozen
// the progress (DEBUG session or debug flag).
func (g *Game) startStealth(i int) {
	g.startStealthN(i, false)
}

// startStealth2P: the same mode with two players (DEBUG menu row and -stealth2p): player 1 is the decoy,
// player 2 sneaks past (stealth2p.go).
func (g *Game) startStealth2P(i int) { g.startStealthN(i, true) }

func (g *Game) startStealthN(i int, two bool) {
	g.stealth2Next = two
	if i < 0 || i >= len(levelDefs) {
		i = 0
	}
	g.stealthNext = true
	g.loadLevel(i)
	g.stealthLayer()
	g.mode = ModePlay
	if g.snd != nil {
		g.snd.StartMusic()
	}
	if two {
		g.say(fmt.Sprintf("SIGILO 2J: uno hace de señuelo (%s / %s), el otro pasa", actKeyName(0, actDance), actKeyName(1, actDance)))
		return
	}
	g.say(fmt.Sprintf("SIGILO: que nadie te vea. %s = caja", actKeyName(0, actInteract)))
}

// stealthExit leaves the mode after the party: back to the DEBUG menu (or the main menu).
func (g *Game) stealthExit() {
	g.stealthNext, g.stealth2Next = false, false
	g.loadLevel(0) // stealthBegin puts the mode off
	if g.debugMenu {
		g.mode = ModeDebug
	} else {
		g.mode = ModeMenu
	}
}

// stealthDef: a copy of the level with a few short hedges more, placed on reachable free floor, away from
// the start, the performers and the guards' routes (the original definition is not touched).
func stealthDef(def *LevelDef) *LevelDef {
	if propKinds["hedge"] == nil {
		return def
	}
	cp := *def
	cp.Props = append([]PropInst(nil), def.Props...)
	so := seatsOff // the hedges go first; the seats of seats.go are placed round them afterwards
	seatsOff = true
	tmp := buildLevel(&cp)
	seatsOff = so
	isHedge := func(p *Prop) bool { return p.Name == "hedge" }
	have := 0
	for _, p := range tmp.Props {
		if isHedge(p) {
			have++
		}
	}
	n0 := len(tmp.Props)
	tmp.ensureBooth("hedge", have+sthHedges, isHedge, []Vec{def.Start}, 24)
	for _, pr := range tmp.Props[n0:] {
		if !isHedge(pr) || stealthHedgeBlocks(def, pr) || stealthHedgeTouches(tmp.Props[:n0], pr) {
			continue
		}
		in := PropInst{Kind: "hedge", X: pr.X, Y: pr.Y}
		if def.PropScale > 0 && def.PropScale != 1 {
			in.Scale = def.PropScale
		}
		cp.Props = append(cp.Props, in)
	}
	return &cp
}

// stealthHedgeTouches: a hedge whose footprint (level: the props the level already had) (plus a step) touches another solid prop's footprint: it would close
// the gap between them (the pirate ship's anchor, v1.4.0) even when the pair rules let them stand side by side.
func stealthHedgeTouches(level []*Prop, pr *Prop) bool {
	foot := pr.Foot.Grow(4)
	for _, o := range level {
		if o.K == nil || o.K.NoSolid {
			continue
		}
		f := o.Foot
		if foot.X < f.X+f.W && f.X < foot.X+foot.W && foot.Y < f.Y+f.H && f.Y < foot.Y+foot.H {
			return true
		}
	}
	return false
}

// stealthHedgeBlocks: a hedge that would sit on a guard's route point, a performer or the start.
func stealthHedgeBlocks(def *LevelDef, pr *Prop) bool {
	foot := pr.Foot.Grow(18)
	for _, route := range def.Guards {
		for _, w := range route {
			if foot.Has(w.X, w.Y) {
				return true
			}
		}
	}
	for _, pf := range def.Performers {
		if foot.Has(pf.Pos.X, pf.Pos.Y) {
			return true
		}
	}
	return foot.Has(def.Start.X, def.Start.Y)
}

// stealthLayer: what is done to the party once it is loaded: fewer guests, no random events.
func (g *Game) stealthLayer() {
	g.twoExtra = 0 // no extra guests; one player only unless this is the 2 players stealth (wantTwoPlayers)
	n := 0
	for _, gu := range g.guests {
		if gu.perf != "" || gu.out() || gu.py != nil {
			continue
		}
		if n++; n > sthGuests { // the extra ones (groups too) are gone, like in enforceGuestCap
			gu.gone = true
		}
	}
	g.total = g.aliveCount()
	for _, gu := range g.guests {
		if g.lvl.SolidP(gu.pos.X, gu.pos.Y) {
			gu.pos = g.lvl.NearestFree(gu.pos)
		}
	}
	const never = 1e9
	g.arrivalT, g.slasherT, g.slot2.t, g.eventT, g.murderT, g.cheerT, g.streakT = never, never, never, never, never, never, never
}

// ---- the cardboard box ---------------------------------------------------------------------------

func (g *Game) stealthBoxed(p *Player) bool {
	return g.stealth && p != nil && g.sth.of(g.playerIndex(p)).box
}

// stealthInteract: E puts the box on or takes it off (before any other use of E).
func (g *Game) stealthInteract(p *Player) bool {
	if !g.stealth || p != g.p || p.hidden || p.dead || p.carrying != nil {
		return false
	}
	s := &g.sth
	sp := s.of(g.playerIndex(p))
	switch {
	case sp.box:
		sp.box, sp.still = false, 0
		g.say("Caja fuera")
	case s.phase == sthRed:
		g.say("¡Con la alerta no puedes esconderte!")
	default:
		sp.box, sp.still = true, 0
		g.say("Caja puesta: nadie te ve")
	}
	return true
}

func (g *Game) stealthSpeedMul(p *Player) float64 {
	if g.stealthBoxed(p) {
		p.running = false
		return sthBoxSpeed
	}
	return 1
}

func (g *Game) stealthNoAttack() bool {
	if g.stealthBoxed(g.p) {
		g.say(fmt.Sprintf("Quita la caja para atacar (%s)", actKeyName(g.playerIndex(g.p), actInteract)))
		return true
	}
	return false
}

// ---- detection and phases ------------------------------------------------------------------------

// stealthSeen: somebody has the player inside their vision cone (same ranges as the normal game).
func (g *Game) stealthSeen(p *Player) bool {
	if g.cameraSees(p.pos) { // a security camera has him in its cone (camera.go)
		return true
	}
	for _, gu := range g.guests {
		if gu.out() || gu.state == gSleep || gu.state == gLift || gu.state == gZombie {
			continue
		}
		if g.sees(gu.pos, gu.dir, p.pos, guestRange, guestDot) {
			return true
		}
	}
	for _, gd := range g.guards {
		if !gd.dead && g.sees(gd.pos, gd.dir, p.pos, guardRange, guardDot) {
			return true
		}
	}
	for _, c := range g.cops {
		if !c.medic && c.state != cLeave && g.sees(c.pos, c.dir, p.pos, copRange, copDot) {
			return true
		}
	}
	return false
}

// stealthBoxBusted: under the box nobody sees the player, but somebody can bump into him or he can be
// standing still in front of an officer / a guard for too long.
func (g *Game) stealthBoxBusted(p *Player, dt float64) bool {
	sp := g.sth.of(g.playerIndex(p))
	bump := func(at Vec) bool { return dist(at, p.pos) < sthBoxBump }
	for _, gu := range g.guests {
		if !gu.out() && bump(gu.pos) {
			return true
		}
	}
	near := false
	for _, gd := range g.guards {
		if gd.dead {
			continue
		}
		if bump(gd.pos) {
			return true
		}
		if g.sees(gd.pos, gd.dir, p.pos, sthBoxNear, guardDot) {
			near = true
		}
	}
	for _, c := range g.cops {
		if c.medic || c.state == cLeave {
			continue
		}
		if bump(c.pos) {
			return true
		}
		if g.sees(c.pos, c.dir, p.pos, sthBoxNear, copDot) {
			near = true
		}
	}
	if near && !p.moving {
		sp.still += dt
	} else {
		sp.still = 0
	}
	return sp.still >= sthBoxStill
}

func (g *Game) updateStealth(dt float64) {
	if !g.stealth || g.mode != ModePlay || g.p == nil {
		return
	}
	s := &g.sth
	if s.exT > 0 {
		s.exT -= dt
	}
	if s.boxBlown > 0 {
		s.boxBlown -= dt
	}
	g.updateStealthNoise(dt)
	if g.livePlayers() == 0 {
		return
	}
	for _, p := range g.players() {
		if !p.dead {
			g.stealthWatch(p, dt)
		}
	}
	switch s.phase {
	case sthRed:
		s.t -= dt
		if !g.playerCleared() {
			g.wanted = math.Max(g.wanted, 45) // the police look for them non-stop
		}
		if s.t <= 0 {
			s.phase, s.t = sthYellow, sthYellowTime
			g.clearPlayer() // the chase is over: the officers go back to their search
		}
	case sthYellow:
		s.t -= dt
		if s.t <= 0 {
			s.phase, s.t = sthCalm, 0
		}
	}
}

// stealthWatch: who sees THIS player (each one has his own meter and his own box).
func (g *Game) stealthWatch(p *Player, dt float64) {
	s := &g.sth
	sp := s.of(g.playerIndex(p))
	switch {
	case p.hidden: // in a bush: nobody sees him
		sp.meter = math.Max(0, sp.meter-dt*1.5)
	case sp.box:
		sp.meter = 0
		if g.stealthBoxBusted(p, dt) {
			s.boxBlown = 2.5
			g.stealthDetectBy(p)
		}
	default:
		if g.stealthSeen(p) {
			sp.meter += dt / sthSeeTime
		} else {
			sp.meter = math.Max(0, sp.meter-dt*1.5)
		}
		if sp.meter >= 1 {
			g.stealthDetectBy(p)
		}
	}
}

// stealthDetect: the player was detected. First time (or from caution): the alert sound, the big "!",
// the police are called. During the red phase it only starts the time again.
func (g *Game) stealthDetect() { g.stealthDetectBy(g.p) }

// stealthDetectBy: player p was detected. The red alert is for both players; it remembers who caused it.
func (g *Game) stealthDetectBy(p *Player) {
	if !g.stealth || p == nil || p.dead {
		return
	}
	s := &g.sth
	i := g.playerIndex(p)
	sp := s.of(i)
	sp.meter = 0
	if sp.box {
		sp.box, sp.still = false, 0
		s.boxBlown = 2.5
	}
	if s.phase == sthRed {
		s.t = sthRedTime
		return
	}
	s.phase, s.t = sthRed, sthRedTime
	s.detections++
	s.by[i]++
	s.culprit = i
	s.exT, s.exMax = 1.6, 1.6
	g.stealthSound()
	g.say("¡Te han visto!")
	g.killPos = p.pos
	g.raisePolice(p.pos, true)
	g.withPlayer(p, func() { g.markPlayerWanted(70) })
}

// stealthAlertSfx: the alert sting, synthesised like every sound of the game: three quick rising square
// pulses and a held, buzzing top note over a low thump (short, its own tune).
func stealthAlertSfx() []byte {
	return mix(
		tone(587, 0.09, 0.5, 8, 1),
		shift(tone(880, 0.09, 0.5, 8, 1), 0.10),
		shift(tone(1175, 0.09, 0.5, 8, 1), 0.20),
		shift(tone(1175, 0.40, 0.4, 4, 2), 0.30),
		tone(110, 0.28, 0.6, 9, 0),
	)
}

func (g *Game) stealthSound() {
	s := g.snd
	if s == nil || s.muted || s.sfx == nil {
		return
	}
	if _, ok := s.sfx["stealth_alert"]; !ok {
		s.sfx["stealth_alert"] = stealthAlertSfx()
	}
	s.Play("stealth_alert")
}

// ---- the end -------------------------------------------------------------------------------------

func (g *Game) stealthWin() {
	g.endTime = g.time
	g.snd.Play("win")
	g.mode = ModeLevelDone
	if g.sth.detections == 0 {
		g.toasts = append(g.toasts, trophyToast{name: "SIN SER VISTO", reward: "Modo sigilo"})
	}
}

func (g *Game) stealthDoneTitle(s string) string {
	if g.stealth && g.sth.detections == 0 {
		return "SIN SER VISTO"
	}
	return s
}

func (g *Game) stealthDoneSub(s string) string {
	if !g.stealth {
		return s
	}
	if g.sth.detections == 0 {
		if g.stealth2 {
			return "Fantasmas perfectos: nadie os detectó"
		}
		return "Fantasma perfecto: nadie te detectó"
	}
	return tr("Fiesta limpia, pero te detectaron:") + " " + strconv.Itoa(g.sth.detections)
}

func (g *Game) stealthDoneHint(s string) string {
	if g.stealth {
		return "ENTER: salir del modo sigilo"
	}
	return s
}

// ---- drawing -------------------------------------------------------------------------------------

// stealthDrawBox draws the player as a cardboard box; false when he does not wear it.
func (g *Game) stealthDrawBox(dst *ebiten.Image, p *Player) bool {
	if !g.stealthBoxed(p) || p.dead {
		return false
	}
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	x, y := math.Round(g.wx(p.pos.X)), math.Round(g.wy(p.pos.Y))
	bob := 0.0
	if p.moving {
		bob = math.Round(math.Abs(math.Sin(p.anim*1.2)) * 1.5)
	}
	w, h := math.Round(22*sc), math.Round(24*sc)
	y -= bob
	fillRect(dst, x-w/2-1, y-2, w+2, 4, rgba(0x000000, 70)) // shadow
	fillRect(dst, x-w/2, y-h, w, h, rgb(0xb98a52))          // sides
	fillRect(dst, x-w/2, y-h, w, 3, rgb(0xd8b878))          // lid flaps
	fillRect(dst, x-1, y-h, 2, h, rgb(0xd8b878))            // tape
	fillRect(dst, x-w/2, y-h+3, w, 1, rgb(0x7a5430))
	fillRect(dst, x-w/2, y-4, w, 4, rgb(0x9a6e3e)) // darker base
	fillRect(dst, x-w/2, y-h, 1, h, rgb(0x7a5430)) // edges
	fillRect(dst, x+w/2-1, y-h, 1, h, rgb(0x7a5430))
	fillRect(dst, x-w/2+3, y-h*0.55, w-6, 3, rgb(0x2a1c10)) // the slit he looks through
	return true
}

// stealthCone paints the cone of vision of one watcher as strips along his looking direction, cut where a
// wall stops the sight.
func (g *Game) stealthCone(dst *ebiten.Image, pos Vec, dir Dir, rng, minDot float64, col color.RGBA) {
	if g.visK > 0 {
		rng *= g.visK
	}
	sx, sy := g.wx(pos.X), g.wy(pos.Y)
	if sx < -rng || sx > screenW+rng || sy < -rng || sy > screenH+rng {
		return
	}
	v := dirVec(dir)
	half := math.Sqrt(1-minDot*minDot) / minDot
	eye := Vec{pos.X, pos.Y - 10}
	for s := 6.0; s <= rng; s += 4 {
		at := Vec{eye.X + v.X*s, eye.Y + v.Y*s}
		if !g.lvl.LOS(eye, at) {
			break
		}
		w := math.Min(s*half, math.Sqrt(math.Max(0, rng*rng-s*s)))
		cx, cy := g.wx(at.X), g.wy(at.Y)
		if v.X != 0 {
			fillRect(dst, math.Round(cx)-2, math.Round(cy-w), 4, math.Round(2*w), col)
		} else {
			fillRect(dst, math.Round(cx-w), math.Round(cy)-2, math.Round(2*w), 4, col)
		}
	}
}

var (
	sthColRed    = rgb(0xff4a3a)
	sthColYellow = rgb(0xffd23a)
	sthColCalm   = rgb(0x60e090)
)

func (g *Game) stealthPhaseColor() color.RGBA {
	switch g.sth.phase {
	case sthRed:
		return sthColRed
	case sthYellow:
		return sthColYellow
	}
	return sthColCalm
}

// drawStealth: cones, the detection bar, the big "!" and the phase panel.
func (g *Game) drawStealth(dst *ebiten.Image) {
	if !g.stealth || g.mode != ModePlay || g.lvl == nil || g.p == nil {
		return
	}
	s := &g.sth
	alpha := uint8(34)
	if s.phase == sthRed {
		alpha = 56
	}
	for _, gd := range g.guards {
		if !gd.dead {
			g.stealthCone(dst, gd.pos, gd.dir, guardRange, guardDot, rgba(0xff9628, alpha))
		}
	}
	for _, c := range g.cops {
		if !c.medic && c.state != cLeave {
			g.stealthCone(dst, c.pos, c.dir, copRange, copDot, rgba(0x4696ff, alpha))
		}
	}
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	for i, p := range g.players() {
		px, top := math.Round(g.wx(p.pos.X)), math.Round(g.wy(p.pos.Y))-44*sc
		if m := s.of(i).meter; m > 0.02 && !p.dead { // how close he is to being detected
			bw := 24.0
			fillRect(dst, px-bw/2-1, top-1, bw+2, 5, rgb(0x14102a))
			c := sthColYellow
			if m > 0.6 {
				c = sthColRed
			}
			fillRect(dst, px-bw/2, top, math.Round(bw*math.Min(1, m)), 3, c)
		}
		if s.exT > 0 && i == s.culprit {
			g.drawBigBang(dst, px, top-8, s)
		}
	}
	g.drawStealthNoise(dst)
	g.drawStealthHUD(dst)
}

// drawBigBang: the big exclamation of the detection, popping up over the player's head.
func (g *Game) drawBigBang(dst *ebiten.Image, cx, bottom float64, s *stealthState) {
	age := s.exMax - s.exT
	k := math.Min(1, age/0.12) // pops in
	hot := int(g.time*10)%2 == 0
	body := sthColRed
	if hot {
		body = sthColYellow
	}
	w, h := math.Round(22*k), math.Round(30*k)
	if w < 4 {
		return
	}
	x, y := math.Round(cx-w/2), math.Round(bottom-h-math.Abs(math.Sin(age*9))*3*(1-k*0.5))
	edge := rgb(0x3a0a04)
	fillRect(dst, x-2, y+2, w+4, h-4, edge)
	fillRect(dst, x+2, y-2, w-4, h+4, edge)
	fillRect(dst, x, y, w, h, body)
	ink := rgb(0x3a0a04)
	fillRect(dst, cx-3, y+4, 6, math.Round(h*0.5), ink) // the bar of the "!"
	fillRect(dst, cx-3, y+h-10, 6, 6, ink)              // its dot
}

// drawStealthHUD: phase panel at the top centre: colour, name, seconds left and a bar that empties.
func (g *Game) drawStealthHUD(dst *ebiten.Image) {
	s := &g.sth
	col := g.stealthPhaseColor()
	const px, py, pw, ph = 650.0, 34.0, 290.0, 54.0
	fillRect(dst, px, py, pw, ph, rgba(0x08060f, 215))
	fillRect(dst, px, py, pw, 2, col)
	fillRect(dst, px, py+ph-2, pw, 2, col)
	label, sub := "EN SIGILO", fmt.Sprintf("%s: ponerte la caja", actKeyName(0, actInteract))
	if g.stealth2 {
		sub = fmt.Sprintf("%s: caja · %s o %s: ruido (señuelo)", actKeyName(0, actInteract), actKeyName(0, actDance), actKeyName(1, actDance))
	}
	switch s.phase {
	case sthRed:
		label, sub = "ALERTA", "Los polis te buscan sin parar"
		if g.stealth2 {
			sub = g.stealthCulpritText()
		}
	case sthYellow:
		label, sub = "PRECAUCIÓN", "Siguen buscando: no te dejes ver"
	default:
		if s.box {
			sub = fmt.Sprintf("CAJA PUESTA · %s: quitarla", actKeyName(0, actInteract))
			if g.stealth2 {
				sub = fmt.Sprintf("%s: caja (puesta) · %s o %s: ruido", actKeyName(0, actInteract), actKeyName(0, actDance), actKeyName(1, actDance))
			}
		}
	}
	g.drawText(dst, label, px+10, py+2, col, 2)
	g.drawText(dst, sub, px+10, py+30, rgb(0xd8d0f0), 1)
	if s.phase != sthCalm {
		left := int(math.Ceil(s.t))
		g.drawText(dst, fmt.Sprintf("%ds", left), px+pw-58, py+2, col, 2)
		total := sthRedTime
		if s.phase == sthYellow {
			total = sthYellowTime
		}
		bw := pw - 20
		fillRect(dst, px+10, py+ph-6, bw, 3, rgb(0x2a2440))
		fillRect(dst, px+10, py+ph-6, math.Round(bw*clampF(s.t/total, 0, 1)), 3, col)
	}
}

// strings of the mode in English (added to the dictionary of tr(); lang_data.go stays untouched)
func init() {
	for es, en := range map[string]string{
		"EN SIGILO":                                        "STEALTH",
		"ALERTA":                                           "ALERT",
		"PRECAUCIÓN":                                       "CAUTION",
		"%s: ponerte la caja":                              "%s: put the box on",
		"CAJA PUESTA · %s: quitarla":                       "BOX ON · %s: take it off",
		"SIGILO: que nadie te vea. %s = caja":              "STEALTH: stay unseen. %s = box",
		"Quita la caja para atacar (%s)":                   "Take the box off to attack (%s)",
		"Los polis te buscan sin parar":                    "The cops keep looking for you",
		"Siguen buscando: no te dejes ver":                 "They still search: stay unseen",
		"Caja puesta: nadie te ve":                         "Box on: nobody sees you",
		"Caja fuera":                                       "Box off",
		"¡Con la alerta no puedes esconderte!":             "No hiding during the alert!",
		"¡Te han visto!":                                   "You were seen!",
		"SIN SER VISTO":                                    "UNSEEN",
		"Modo sigilo":                                      "Stealth mode",
		"Fantasma perfecto: nadie te detectó":              "Perfect ghost: nobody spotted you",
		"Fiesta limpia, pero te detectaron:":               "Clean party, but you were spotted:",
		"ENTER: salir del modo sigilo":                     "ENTER: leave the stealth mode",
		stealthMenuLabel:                                   "STEALTH mode: win unseen (cardboard box)",
		"Policía con memoria (el testigo asustado delata)": "Police with memory (a scared witness tells on you)",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}

package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "linterna" (idea 291, GUARDIA CON LINTERNA): in some NIGHT parties (not Def.Day; only the dark, roomy venues of
// flashIcons; about 1 party in 3 of those, own dice, 0-1 per party) one more security guard (the game's own guard sprite)
// comes on patrol with a torch. He walks a route of 3 reachable points (Level.Path; if he does not move for flashStuck
// seconds he skips to the next point or is put back on free ground) and the torch throws a warm cone of light on the floor
// (a triangle fan cut by the walls, see flashRays).
//
//   - DETECTION: a player inside the cone who is really in sight (Level.LOS: no wall or solid between, no smoke, not hidden)
//     makes him shout "¿Quién anda ahí?" and, after flashAlarm seconds of light on him, "¡Alto ahí!": the alarm sound and the
//     ORDINARY guard chase (gdReact -> gdChase, the same leash, endurance and beating as any guard: never harder).
//     Outside the cone he still has the ordinary eyes of a guard (guardRange), nothing more.
//   - THE TRICK TO SWITCH IT OFF (no new systems): come up BEHIND him (outside the cone, within flashOffDist) and press the
//     use key: "clic", the torch goes out. He fumbles with it for a couple of seconds ("¡Eh! ¡Mi linterna!") without turning
//     round and from then on he is an ordinary guard with no torch. A smoke grenade or a cream cake (smogT) makes him drop it
//     too. A use label "[E] Apagar linterna" shows over him while the player is in the right spot.
//   - KILLED: an ordinary guard body (killGuard: the guards' statistics); the torch rolls a little on the floor and stays lit
//     flashDropLife seconds, lighting the ground, then the battery dies.
//   - he COUNTS AS A GUARD (g.guards, g.total + 1, he must go for the party to end), only if the party has room (guestRoom).
//
// Debug: -pevent linterna (forced at 3 s), -pevent linterna:cone (by the player, the cone in view), linterna:seen (the player
// walks into the light), linterna:off (the player switches it off from behind), linterna:dead (killed: the torch on the floor).
// ---------------------------------------------------------------------------

const (
	flashRange    = 150.0 // reach of the beam
	flashHalf     = 0.42  // half the opening of the beam (radians, about 24 degrees)
	flashRays     = 17    // rays of the fan (the cone is cut by the walls ray by ray)
	flashSeeRate  = 1.4   // sight gained per second of light on a player
	flashAlarm    = 1.0   // ... the alarm comes at this much
	flashWho      = 0.3   // ... "¿Quién anda ahí?" at this much
	flashOffDist  = 24.0  // the player switches it off from behind within this distance
	flashFumble   = 2.5   // seconds he fumbles with the dead torch
	flashDropLife = 25.0  // seconds the dropped torch stays lit on the floor
	flashDropR    = 64.0  // reach of the dropped torch's light
	flashStuck    = 5.0   // seconds without moving on patrol before he is helped on
	flashChance   = 0.33  // per eligible party (1 in 3)
)

var (
	flashWhoLines   = pevTexts("¿Quién anda ahí?", "¿Hay alguien ahí?")
	flashAlarmLines = pevTexts("¡Alto ahí!", "¡Alto! ¡Seguridad!")
	flashOffLines   = pevTexts("¡Eh! ¡Mi linterna!", "¿Y la luz? ¡Pilas!")
	flashPrompt     = pevTexts("Apagar linterna")[0]
	flashSayOff     = pevTexts("Le has apagado la linterna al guardia")[0]
)

// flashIcons: night venues with room to walk a patrol in the dark (roomy, mostly open air). Day levels never.
var flashIcons = map[string]bool{"grave": true, "funfair": true, "festival": true, "ranch": true, "skate": true,
	"train": true, "snow": true, "circuit": true}

type flashGuard struct {
	gd      *Guard
	lvl     *Level
	on      bool
	ang     float64 // where the beam points (smoothed)
	sight   float64
	alarms  int
	offWhy  string // "back", "daze", "dead" (tests)
	fumble  float64
	rays    [flashRays]float64
	raysAt  int // g.tick of the last ray cast
	lastPos Vec
	stuckT  float64
	dropAt  Vec // the torch on the floor
	dropV   Vec // its roll
	dropAng float64
	dropT   float64 // g.time when it fell (0 = in his hand)
}

// flashGuards: the torch guards of the party being played (a new level forgets them).
var flashGuards = map[*Guard]*flashGuard{}

func init() {
	registerPartyEvent(&partyEvent{ID: "linterna", Name: "Guardia con linterna", Group: "guard",
		Prob: flashChance, AtMin: 20, AtMax: 110, Dur: 300, MaxRuns: 1, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && flashIcons[t.icon] },
		Start:    flashStart,
		Update:   flashUpdate})
}

func dirAng(d Dir) float64 {
	v := dirVec(d)
	return math.Atan2(v.Y, v.X)
}

func angDiff(a, b float64) float64 {
	d := math.Mod(a-b+3*math.Pi, 2*math.Pi) - math.Pi
	return d
}

// flashOf: the torch state of this guard in the party being played (nil = an ordinary guard).
func (g *Game) flashOf(gd *Guard) *flashGuard {
	if len(flashGuards) == 0 || gd == nil {
		return nil
	}
	s := flashGuards[gd]
	if s == nil || s.lvl != g.lvl {
		return nil
	}
	return s
}

// flashRoute: the spawn and 2 more patrol points, all reachable from each other, far from the start and the players.
func (g *Game) flashRoute() ([]Vec, bool) {
	l := g.lvl
	r := partyDice()
	demo := pevDemo("linterna") != ""
	if demo { // captures: right by the first player
		g.p.pos = l.NearestFree(Vec{g.p.pos.X, g.p.pos.Y - 160}) // into the party, clear of the intro box at the bottom
		p := g.p.pos
		a := l.NearestFree(Vec{p.X - 160, p.Y - 110}) // the player out of the beam (35 degrees off it)
		b := l.NearestFree(Vec{p.X + 160, p.Y - 110})
		if pevDemo("linterna") == "seen" { // he walks down towards the player
			a, b = l.NearestFree(Vec{p.X, p.Y - 150}), l.NearestFree(Vec{p.X, p.Y + 40})
		}
		return []Vec{a, b}, l.Path(a, b) != nil
	}
	ok := func(p Vec) bool {
		return l.Free(p) && !l.SolidP(p.X, p.Y) && !l.badGround(p) && !l.InPool(p)
	}
	for try := 0; try < 40; try++ {
		a := l.RandomRoam(r)
		if !ok(a) || dist(a, l.Def.Start) < 140 || !g.pevPlayerFar(a, 180) || l.Path(l.Def.Start, a) == nil {
			continue
		}
		route := []Vec{a}
		for k := 0; k < 30 && len(route) < 3; k++ {
			b := l.RandomRoam(r)
			last := route[len(route)-1]
			if d := dist(b, last); !ok(b) || d < 110 || d > 320 || l.Path(last, b) == nil {
				continue
			}
			route = append(route, b)
		}
		if len(route) >= 2 {
			return route, true
		}
	}
	return nil, false
}

func flashStart(g *Game, a *partyAct) bool {
	if len(g.as.guardCS) == 0 || g.lvl == nil || g.lvl.Def.Day || g.guestRoom() < 1 { // a guard like any: only with room (cap.go)
		return false
	}
	for gd, s := range flashGuards { // another party's
		if s.lvl != g.lvl {
			delete(flashGuards, gd)
		}
	}
	for _, s := range flashGuards {
		if !s.gd.dead || s.dropT > 0 {
			return false // 0-1 per party
		}
	}
	route, ok := g.flashRoute()
	if !ok {
		return false
	}
	var hx, hy float64
	for _, p := range route {
		hx, hy = hx+p.X, hy+p.Y
	}
	home := Vec{hx / float64(len(route)), hy / float64(len(route))}
	gd := &Guard{pos: route[0], cs: g.as.guardCS[len(g.guards)%len(g.as.guardCS)], route: route, ri: 1 % len(route),
		state: gdWait, timer: 0.5, home: home, leash: 170, dir: faceTo(route[0], route[1%len(route)])}
	g.guards = append(g.guards, gd)
	g.total++ // one more to count (BAJAS x/N), like every guard who comes in later
	s := &flashGuard{gd: gd, lvl: g.lvl, on: true, ang: dirAng(gd.dir), lastPos: gd.pos, raysAt: -99}
	flashGuards[gd] = s
	a.D = s
	pevDemoLog("linterna", gd.pos)
	return true
}

// flashUpdate: the event is only his arrival (he patrols on through flashGuards without taking a place of the running
// events); the captures keep it to stage what they show.
func flashUpdate(g *Game, a *partyAct, dt float64) bool {
	s, _ := a.D.(*flashGuard)
	st := pevDemo("linterna")
	if s == nil || st == "" || st == "cone" {
		return true
	}
	gd := s.gd
	el := g.time - a.t0
	switch st {
	case "seen":
		return el > 6
	case "off":
		if el > 2 && s.on {
			back := dirVec(gd.dir)
			g.p.pos = g.lvl.NearestFree(Vec{gd.pos.X - back.X*16, gd.pos.Y - back.Y*16})
			if el > 3 {
				g.flashInteract(g.p)
			}
		}
		return el > 6
	case "dead":
		if el > 2 && !gd.dead {
			g.killGuard(gd)
		}
		return el > 6
	}
	return true
}

// flashGuardTick (top of updateGuard, before anything else): the beam, the patrol watchdog, the light on the players.
func (g *Game) flashGuardTick(gd *Guard, dt float64) {
	s := g.flashOf(gd)
	if s == nil {
		return
	}
	if gd.dead {
		if s.on { // the torch falls and rolls a little, still lit
			s.on, s.offWhy, s.dropT, s.dropAt, s.dropAng = false, "dead", g.time, gd.pos, s.ang
			s.dropV = Vec{math.Cos(s.ang) * 34, math.Sin(s.ang) * 34}
		}
		return
	}
	// the beam follows where he faces, with a slow sweep while he walks
	want := dirAng(gd.dir)
	if gd.moving && len(gd.path) > 0 && dist(gd.path[0], gd.pos) > 2 { // walking: where he really goes
		want = math.Atan2(gd.path[0].Y-gd.pos.Y, gd.path[0].X-gd.pos.X)
	}
	if gd.state == gdPatrol || gd.state == gdWait {
		want += 0.18 * math.Sin(g.time*1.3)
	}
	s.ang += angDiff(want, s.ang) * math.Min(1, dt*6)
	if s.fumble > 0 {
		s.fumble -= dt
		gd.moving, gd.state, gd.path, gd.timer = false, gdWait, nil, math.Max(gd.timer, 0.2)
	}
	// patrol watchdog: never stuck
	if gd.state == gdPatrol {
		if dist(gd.pos, s.lastPos) < 0.5 {
			s.stuckT += dt
		} else {
			s.stuckT = 0
		}
		if s.stuckT > flashStuck {
			s.stuckT = 0
			if !g.lvl.Free(gd.pos) {
				gd.pos = g.lvl.NearestFree(gd.pos)
			}
			gd.state, gd.timer, gd.path = gdWait, 0.2, nil
		}
	} else {
		s.stuckT = 0
	}
	s.lastPos = gd.pos
	if !s.on {
		return
	}
	if gd.smogT > 0 { // dazed by smoke or cream: he drops it
		s.turnOff(g, "daze")
		return
	}
	if gd.state == gdReact || gd.state == gdChase || gd.state == gdBeat {
		return // the ordinary chase is on
	}
	p := g.p // the player nearest to him (updateGuard runs under g.focus)
	if p != nil && !p.hidden && !p.dead && !p.jail.held && !g.stealthBoxed(p) && g.flashLit(s, p.pos) {
		s.sight += flashSeeRate * dt
		if s.sight >= flashAlarm {
			s.sight, s.alarms = 0, s.alarms+1
			gd.state, gd.timer = gdReact, 0.4
			gd.icon, gd.iconT = pevPick(partyDice(), flashAlarmLines), 1.8
			gd.lastSeen = p.pos
			gd.chaseT, gd.lost, gd.sight = 0, 0, 1.6
			g.snd.Play("alert")
		} else if s.sight >= flashWho && gd.icon == "" {
			gd.icon, gd.iconT = pevPick(partyDice(), flashWhoLines), 1.4
		}
	} else if s.sight > 0 {
		s.sight = math.Max(0, s.sight-0.8*dt)
	}
}

func (s *flashGuard) turnOff(g *Game, why string) {
	if !s.on {
		return
	}
	s.on, s.offWhy, s.sight = false, why, 0
	gd := s.gd
	gd.icon, gd.iconT = pevPick(partyDice(), flashOffLines), 2
	g.snd.Play("clac")
	if why == "back" {
		s.fumble = flashFumble
	}
}

// flashEye: where the beam starts (his hand, a little ahead of his feet).
func (s *flashGuard) eye() Vec {
	return Vec{s.gd.pos.X + math.Cos(s.ang)*6, s.gd.pos.Y + math.Sin(s.ang)*6 - 2}
}

// flashLit: q is inside the beam and in plain sight of him (no wall or solid between, no smoke, no gold suit).
func (g *Game) flashLit(s *flashGuard, q Vec) bool {
	if !s.on || s.gd.dead || g.suitHides(q) {
		return false
	}
	from := s.gd.pos
	if len(g.smk.clouds) > 0 && g.smokeBlocks(from, q) {
		return false
	}
	rng := flashRange
	if g.visK > 0 {
		rng *= g.visK
	}
	rng *= g.dailySight()
	d := dist(from, q)
	if d > rng {
		return false
	}
	if d > 8 && math.Abs(angDiff(math.Atan2(q.Y-from.Y, q.X-from.X), s.ang)) > flashHalf {
		return false
	}
	return g.lvl.LOS(from, q)
}

// flashCast: the length of every ray of the fan, cut by the walls (a few times a second is plenty).
func (g *Game) flashCast(s *flashGuard, from Vec, ang, rng float64) {
	if g.tick-s.raysAt < 3 && g.tick >= s.raysAt {
		return
	}
	s.raysAt = g.tick
	for i := 0; i < flashRays; i++ {
		a := ang - flashHalf + 2*flashHalf*float64(i)/float64(flashRays-1)
		v := Vec{math.Cos(a), math.Sin(a)}
		end := Vec{from.X + v.X*rng, from.Y + v.Y*rng}
		if g.lvl.LOS(from, end) {
			s.rays[i] = rng
			continue
		}
		lo, hi := 0.0, rng
		for k := 0; k < 7; k++ {
			m := (lo + hi) / 2
			if g.lvl.LOS(from, Vec{from.X + v.X*m, from.Y + v.Y*m}) {
				lo = m
			} else {
				hi = m
			}
		}
		s.rays[i] = lo + 4
	}
}

// flashInteract (interact): the use key from behind him switches the torch off. true = done.
func (g *Game) flashInteract(p *Player) bool {
	if p == nil || p.dead || p.carrying != nil || len(flashGuards) == 0 {
		return false
	}
	for _, s := range flashGuards {
		if s.lvl == g.lvl && g.flashBehind(s, p) {
			s.turnOff(g, "back")
			g.say(flashSayOff)
			return true
		}
	}
	return false
}

// flashBehind: the player stands right behind a torch guard, out of the light, and the guard is not after anybody.
func (g *Game) flashBehind(s *flashGuard, p *Player) bool {
	gd := s.gd
	if !s.on || gd.dead || gd.state == gdReact || gd.state == gdChase || gd.state == gdBeat || p.hidden {
		return false
	}
	d := dist(p.pos, gd.pos)
	if d > flashOffDist || g.flashLit(s, p.pos) {
		return false
	}
	v := Vec{math.Cos(s.ang), math.Sin(s.ang)}
	return d < 4 || (v.X*(p.pos.X-gd.pos.X)+v.Y*(p.pos.Y-gd.pos.Y))/d < -0.15
}

// ---- drawing -----------------------------------------------------------------------------------------------------

var (
	flashVs [(flashRays + 1) * 2]ebiten.Vertex
	flashIs [(flashRays - 1) * 3 * 2]uint16
	flashOp = ebiten.DrawTrianglesOptions{Blend: ebiten.BlendLighter}
	flashC  = color.RGBA{0xff, 0xd8, 0x8c, 255} // warm torch light
)

// flashFan draws the cone of light from the eye (world coordinates) with the given ray lengths and brightness k.
func (g *Game) flashFan(dst *ebiten.Image, eye Vec, ang float64, rays []float64, k float64) {
	if px == nil || k <= 0 {
		return
	}
	n := len(rays)
	ex, ey := float32(g.wx(eye.X)), float32(g.wy(eye.Y))
	r, gg, b := float32(flashC.R)/255, float32(flashC.G)/255, float32(flashC.B)/255
	vs := flashVs[:0]
	is := flashIs[:0]
	vs = append(vs, ebiten.Vertex{DstX: ex, DstY: ey, SrcX: 0.5, SrcY: 0.5, ColorR: r * float32(0.34*k), ColorG: gg * float32(0.34*k), ColorB: b * float32(0.34*k), ColorA: 1})
	for i := 0; i < n; i++ {
		a := ang - flashHalf + 2*flashHalf*float64(i)/float64(n-1)
		edge := 1.0
		if i == 0 || i == n-1 {
			edge = 0.4 // soft sides
		}
		kk := float32(0.07 * k * edge)
		vs = append(vs, ebiten.Vertex{DstX: float32(g.wx(eye.X + math.Cos(a)*rays[i])), DstY: float32(g.wy(eye.Y + math.Sin(a)*rays[i])),
			SrcX: 0.5, SrcY: 0.5, ColorR: r * kk, ColorG: gg * kk, ColorB: b * kk, ColorA: 1})
	}
	for i := 1; i < n; i++ {
		is = append(is, 0, uint16(i), uint16(i+1))
	}
	dst.DrawTriangles(vs, is, px, &flashOp)
	// a second, narrower and brighter core
	vs2 := append(flashVs[n+1:n+1], vs[0])
	m := 0
	for i := n / 4; i <= n-1-n/4; i++ {
		a := ang - flashHalf + 2*flashHalf*float64(i)/float64(n-1)
		l := rays[i] * 0.75
		kk := float32(0.05 * k)
		vs2 = append(vs2, ebiten.Vertex{DstX: float32(g.wx(eye.X + math.Cos(a)*l)), DstY: float32(g.wy(eye.Y + math.Sin(a)*l)),
			SrcX: 0.5, SrcY: 0.5, ColorR: r * kk, ColorG: gg * kk, ColorB: b * kk, ColorA: 1})
		m++
	}
	is2 := flashIs[len(is):len(is)]
	for i := 1; i < m; i++ {
		is2 = append(is2, 0, uint16(i), uint16(i+1))
	}
	if len(is2) > 0 {
		dst.DrawTriangles(vs2, is2, px, &flashOp)
	}
}

// drawFlashlights (after the night layer, with the party overlays): the beams of the torch guards, the lit lens in his
// hand, the torch on the floor and the use label for the player who stands right behind him.
func (g *Game) drawFlashlights(dst *ebiten.Image) {
	if len(flashGuards) == 0 || g.lvl == nil || (g.mode != ModePlay && g.mode != ModeLose) {
		return
	}
	for _, s := range flashGuards {
		if s.lvl != g.lvl {
			continue
		}
		gd := s.gd
		if s.on && !gd.dead {
			eye := s.eye()
			g.flashCast(s, eye, s.ang, flashRange)
			g.flashFan(dst, eye, s.ang, s.rays[:], 1)
			lx, ly := math.Round(g.wx(eye.X)), math.Round(g.wy(eye.Y-10))
			if g.as != nil && g.as.glow != nil {
				g.glowAt(dst, Vec{eye.X, eye.Y - 10}, 10, color.RGBA{120, 100, 60, 120})
			}
			fillRect(dst, lx-1, ly-1, 2, 2, rgb(0xfff4c8))
			g.flashLabel(dst, s)
			continue
		}
		if s.dropT > 0 {
			g.drawDroppedTorch(dst, s)
		}
	}
}

// drawDroppedTorch: the torch on the floor, rolled a little, its light fading out at the end of the battery.
func (g *Game) drawDroppedTorch(dst *ebiten.Image, s *flashGuard) {
	el := g.time - s.dropT
	if el > flashDropLife || el < 0 {
		return
	}
	// the roll: it slides with the push of the fall and stops (a wall stops it)
	roll := math.Min(el, 0.6)
	k := roll - roll*roll/1.2 // decelerating
	at := Vec{s.dropAt.X + s.dropV.X*k, s.dropAt.Y + s.dropV.Y*k}
	if g.lvl.SolidP(at.X, at.Y) || !g.lvl.LOS(s.dropAt, at) {
		at = s.dropAt
	}
	ang := s.dropAng + math.Min(el, 0.6)*5 // it turns while it rolls
	lum := 1.0
	if el > flashDropLife-5 {
		lum = (flashDropLife - el) / 5
		if int(el*9)%3 == 0 {
			lum *= 0.5 // the flicker of a dying battery
		}
	}
	var rays [flashRays]float64
	for i := range rays {
		a := ang - flashHalf + 2*flashHalf*float64(i)/float64(flashRays-1)
		rays[i] = flashDropR
		if !g.lvl.LOS(at, Vec{at.X + math.Cos(a)*flashDropR, at.Y + math.Sin(a)*flashDropR}) {
			rays[i] = 12
		}
	}
	g.flashFan(dst, at, ang, rays[:], 0.8*lum)
	// the torch itself: a small dark tube with the lit lens
	x, y := math.Round(g.wx(at.X)), math.Round(g.wy(at.Y))
	dx, dy := math.Round(math.Cos(ang)*3), math.Round(math.Sin(ang)*3)
	fillRect(dst, x-dx-1, y-dy-1, 3, 3, rgb(0x2a2a32))
	fillRect(dst, x-1, y-1, 3, 3, rgb(0x3a3a46))
	fillRect(dst, x+dx-1, y+dy-1, 2, 2, mixc(rgb(0x6a6040), rgb(0xfff4c8), lum))
}

// flashLabel: "[E] Apagar linterna" over him for each player who stands right behind him (like every use label).
func (g *Game) flashLabel(dst *ebiten.Image, s *flashGuard) {
	if g.mode != ModePlay {
		return
	}
	row := 0
	for _, pl := range g.players() {
		if pl == nil || !g.flashBehind(s, pl) {
			continue
		}
		txt := "[" + actionHintFor(actInteract, pl.idx) + "] " + flashPrompt
		w := g.textW(txt, 1)
		x := clampUseTextX(math.Round(g.wx(s.gd.pos.X)), w+map2P(g.is2P()))
		y := math.Round(g.wy(s.gd.pos.Y)) - math.Round(58*g.lifeScale()) - float64(row*14)
		if g.is2P() {
			y = g.clearOfTags(x, y, w+10)
		}
		g.drawTextC(dst, txt, x, y, rgb(0xffd860), 1)
		if g.is2P() {
			fillRect(dst, math.Round(x-w/2)-7, y+2, 4, 4, playerColor(pl.idx))
		}
		row++
	}
}

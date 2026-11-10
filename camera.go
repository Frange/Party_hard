package main

import (
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// SECURITY CAMERAS (idea 6, Metal Gear Solid 1 style). Ceiling cameras bolted to the SOUTH face of an
// opaque wall (in a corner when the wall meets another one, else mid-wall). They are not props: they
// block nothing (the floor below stays free) and are drawn above the people, like a ceiling fixture.
//
// BEHAVIOUR
//   - Each one sweeps about camSweep (+-45 degrees = 90 total) around its axis, resting camPause seconds at
//     each end (camOffset). The vision cone is camHalf wide and camRange long, cut by opaque walls
//     (LOS). It is drawn semi-transparent: green when calm, red for camAlertTime s after it has caught a kill.
//   - NORMAL game: walking through the cone does nothing but a short red LED (camLedTime s), unless LOCATING applies. KILLING while the
//     player stands inside a cone (cameraWitness, called by witness() in game.go) makes the camera an
//     immediate, direct witness: like an officer who sees it (suspectSeenBy direct=true), it calls the
//     police at once (raisePolice informed) and the cuffs go on (BUSCADO), per player with 2 players.
//   - LOCATING (normal game, 2026-11-09, "the police know you were there"): a working camera (not in a
//     blackout) that films a player walking through its cone while (a) he is WANTED (cuffs), (b) he has
//     the ALERT "!" (witnesses on their way), or (c) one of HIS knife bodies, younger than camRecent s,
//     lies in the camera's zone, reports his position (camReport): an officer on the scene walks there,
//     the crew on its way heads there (alertPos) or, with no police yet, they are called (raisePolice,
//     which never makes a second call). (a) also keeps the cuffs fresh (seenNow). (b) and (c) put no
//     cuffs on: the camera locates, it did not see the crime. Only knife kills are registered
//     (camStabs, from cameraWitness): traps, poison and accidents never incriminate. Message
//     "CÁMARA: te ha grabado", once per camera and player every camReportCD s.
//   - The killer and an armed guest killing inside a cone are filmed too (cameraSawKill, called by
//     slasherWitness / the rampage in game.go): they become the suspect and the player is cleared.
//   - STEALTH mode (stealth.go): simply being inside a cone also counts as being seen (stealthSeen asks
//     cameraSees), so it fills the detection meter and fires the red ALERT like a guard would. Under the
//     cardboard box or hidden in a bush the camera sees nothing.
//   - Not implemented (on purpose): blinding / destroying cameras.
//
// PLACEMENT (the plan lives here, levels.go stays untouched): cameraPlan says how many cameras each
// level has by title (cameraPlanNormal) and stealth gives every level that allows it cameraStealthN of them.
// placeCameras looks for wall mounts on the built level and refuses any that touch a prop, a door, a
// performer, the start, or whose cone looks at a closed corner.
// ---------------------------------------------------------------------------

const (
	camRange     = 150.0
	camHalf      = 0.42 // half opening of the cone (about 24 degrees)
	camSweep     = math.Pi / 4
	camSweepT    = 3.4 // seconds of one sweep from end to end
	camPause     = 1.3 // seconds resting at each end
	camLedTime   = 1.2 // red LED after somebody passes through the cone
	camAlertTime = 4.0 // red cone after catching a kill
	camRays      = 36  // rays that cut the cone at walls
	camCell      = 4.0 // drawing cell of the cone (pixel-art blocks, like the stealth cones)
	camSpacing   = 240.0
	camStealthN  = 2 // cameras per level in stealth mode (where the level allows it)
)

// cameraPlanNormal: cameras of the NORMAL game by level title (interiors, shops, stations). The rest of the
// levels have none. Stealth ignores the numbers and puts camStealthN everywhere it fits.
var cameraPlanNormal = map[string]int{
	"HOUSE PARTY":       1,
	"DISCOTECA":         1,
	"CINE":              2,
	"CASINO":            2,
	"ESTACIÓN DE TREN":  2,
	"ESTACIÓN ESPACIAL": 2,
	"CENTRO COMERCIAL":  3, // random mode only
	"PUB":               1, // random mode only
}

type secCam struct {
	mount Vec // centre of the bracket on the wall face
	eye   Vec // where the lens is
	base  float64
	t     float64 // own clock (phase of the sweep)
	ang   float64 // current aim, radians, y down (pi/2 = straight down)
	led   float64 // seconds of red LED left
	alert float64 // seconds of red cone left
	corn  bool
	still bool       // does not sweep (tests)
	cool  [2]float64 // seconds before this camera reports the same player again (per player)
	rays  [camRays]float64
}

// camOffset: -1..1 position in the sweep at time t: rests at both ends, smooth in between.
func camOffset(t float64) float64 {
	p := 2 * (camSweepT + camPause)
	u := math.Mod(t, p)
	if u < 0 {
		u += p
	}
	ease := func(x float64) float64 { return x * x * (3 - 2*x) }
	switch {
	case u < camPause:
		return -1
	case u < camPause+camSweepT:
		return -1 + 2*ease((u-camPause)/camSweepT)
	case u < 2*camPause+camSweepT:
		return 1
	}
	return 1 - 2*ease((u-2*camPause-camSweepT)/camSweepT)
}

func (c *secCam) aim() float64 { return c.base + camSweep*camOffset(c.t) }

// ---- placement ------------------------------------------------------------------------------------

func (g *Game) setupCameras() {
	g.cams, g.camStabs = nil, nil
	if g.lvl == nil || g.lvl.Def == nil {
		return
	}
	n := cameraPlanNormal[g.lvl.Def.Title]
	if g.stealth {
		n = camStealthN
	}
	if n > 0 {
		g.cams = placeCameras(g.lvl, n)
	}
}

// camBodyRect: what a camera covers on screen (bracket, arm and housing, with a margin).
func camBodyRect(m Vec) Rect { return Rect{m.X - 11, m.Y - 6, 22, 20} }

func placeCameras(l *Level, n int) []*secCam {
	var cands []*secCam
	score := map[*secCam]float64{}
	for _, w := range l.sight {
		if w.W < 60 || w.H > 30 || w.H < 4 {
			continue
		}
		y := w.Y + w.H
		type spot struct {
			x    float64
			base float64
			corn bool
		}
		var spots []spot
		for _, s := range l.sight { // a wall that hangs below one end of this one: a corner
			if s.H <= s.W || s.Y > y+6 || s.Y+s.H < y+40 {
				continue
			}
			if s.X+s.W >= w.X-2 && s.X+s.W <= w.X+14 { // wall on the left end
				spots = append(spots, spot{s.X + s.W + 10, math.Pi / 4, true})
			}
			if s.X <= w.X+w.W+2 && s.X >= w.X+w.W-14 { // wall on the right end
				spots = append(spots, spot{s.X - 10, 3 * math.Pi / 4, true})
			}
		}
		if w.W >= 140 {
			spots = append(spots, spot{w.X + w.W/2, math.Pi / 2, false})
		}
		for _, sp := range spots {
			if sp.x < w.X+8 || sp.x > w.X+w.W-8 {
				continue
			}
			c := &secCam{mount: Vec{sp.x, y}, base: sp.base, corn: sp.corn}
			c.eye = Vec{sp.x, y + 6}
			c.ang = c.base
			open, ok := camMountOK(l, c)
			if !ok {
				continue
			}
			sc := open * 100
			if sp.corn {
				sc += 30
			}
			sc += math.Min(dist(c.eye, l.Def.Start), 400) * 0.05
			cands = append(cands, c)
			score[c] = sc
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if score[cands[i]] != score[cands[j]] {
			return score[cands[i]] > score[cands[j]]
		}
		if cands[i].mount.Y != cands[j].mount.Y {
			return cands[i].mount.Y < cands[j].mount.Y
		}
		return cands[i].mount.X < cands[j].mount.X
	})
	var out []*secCam
	for _, c := range cands {
		if len(out) >= n {
			break
		}
		far := true
		for _, o := range out {
			if dist(o.mount, c.mount) < camSpacing {
				far = false
			}
		}
		if far {
			c.t = float64(len(out)) * 2.3 // not all in step
			c.ang = c.aim()
			out = append(out, c)
		}
	}
	return out
}

// camMountOK: the bracket touches nothing and the cone looks at open floor (returns the open fraction).
func camMountOK(l *Level, c *secCam) (float64, bool) {
	body := camBodyRect(c.mount)
	if body.X < 0 || body.X+body.W > l.W || body.Y < 0 {
		return 0, false
	}
	for _, p := range l.Props {
		sc := 1.0
		if p.scale > 0 {
			sc = p.scale
		}
		w, h := float64(p.K.W)*sc, float64(p.K.H)*sc
		if body.Grow(3).Hits(Rect{p.X - w/2, p.Y - h, w, h}) || body.Grow(3).Hits(p.Foot) {
			return 0, false
		}
	}
	for _, d := range l.Def.Doors {
		if body.Grow(6).Hits(d.R) {
			return 0, false
		}
	}
	for _, pf := range l.Def.Performers {
		if body.Grow(30).Has(pf.Pos.X, pf.Pos.Y) {
			return 0, false
		}
	}
	if body.Grow(20).Has(l.Def.Start.X, l.Def.Start.Y) {
		return 0, false
	}
	for _, w := range l.Walls { // no other wall may cut the bracket
		if w.R.Hits(Rect{c.mount.X - 8, c.mount.Y + 1, 16, 12}) {
			return 0, false
		}
	}
	ok, all := 0, 0
	for a := -camSweep; a <= camSweep+1e-9; a += camSweep / 3 {
		for _, r := range []float64{50, 100, 140} {
			all++
			ang := c.base + a
			q := Vec{c.eye.X + math.Cos(ang)*r, c.eye.Y + math.Sin(ang)*r}
			if q.X > 0 && q.Y > 0 && q.X < l.W && q.Y < l.H && l.LOS(c.eye, q) && l.Free(q) {
				ok++
			}
		}
	}
	open := float64(ok) / float64(all)
	return open, open >= 0.5
}

// ---- vision -----------------------------------------------------------------------------------------

// sees: is the point inside this camera's cone right now (range, angle and walls)?
func (c *secCam) sees(g *Game, q Vec) bool {
	if g.suitHides(q) || g.blackoutOn() { // the gold VIP suit (vip.go); no power, no cameras (blackout.go)
		return false
	}
	rng := camRange
	if g.visK > 0 {
		rng *= g.visK
	}
	d := dist(c.eye, q)
	if d > rng {
		return false
	}
	if d > 6 {
		da := math.Atan2(q.Y-c.eye.Y, q.X-c.eye.X) - c.ang
		for da > math.Pi {
			da -= 2 * math.Pi
		}
		for da < -math.Pi {
			da += 2 * math.Pi
		}
		if math.Abs(da) > camHalf {
			return false
		}
	}
	return g.lvl.LOS(c.eye, q)
}

// cameraSees: any camera has the point inside its cone.
func (g *Game) cameraSees(q Vec) bool {
	for _, c := range g.cams {
		if c.sees(g, q) {
			return true
		}
	}
	return false
}

func (g *Game) camWatched(p *Player) bool {
	return p != nil && !p.dead && !p.hidden && !g.stealthBoxed(p) && !p.jail.held
}

// cameraWitness: a kill by g.p was just made; a camera that has him inside its cone is a direct witness.
func (g *Game) cameraWitness() {
	p := g.p
	if len(g.cams) == 0 || g.lvl == nil || p == nil || p.dead || p.hidden {
		return
	}
	if n := len(g.bodies); n > 0 { // the knife body that just fell: filmed walking past it later = located
		if b := g.bodies[n-1]; b.byPlayer && b.born == g.time {
			if g.camStabs == nil {
				g.camStabs = map[*Body]*Player{}
			}
			g.camStabs[b] = p
		}
	}
	seen := false
	for _, c := range g.cams {
		if c.sees(g, p.pos) {
			c.alert, c.led = camAlertTime, camAlertTime
			seen = true
		}
	}
	if !seen {
		return
	}
	g.run.camSeen++
	g.say(camMsg)
	g.killPos = p.pos
	g.suspectSeenBy(suspPlayer, nil, true) // direct witness: the cuffs go on, per player (2P)
	g.raisePolice(p.pos, true)             // and the police are called at once
	g.markPlayerWanted(70)
	if i := g.playerIndex(p); i < 2 {
		for _, c := range g.cams {
			c.cool[i] = camReportCD // the kill itself was the report: no second message at once
		}
	}
}

const (
	camMsg      = "CÁMARA: te ha grabado"
	camRecent   = 25.0 // seconds a knife body counts as fresh for the cameras
	camReportCD = 6.0  // seconds between two reports of one camera about the same player
)

// cameraSawKill: somebody who is not a player (the killer, an armed guest) killed at `at`. Every working
// camera with the spot in its cone turns red; true = filmed.
func (g *Game) cameraSawKill(at Vec) bool {
	seen := false
	for _, c := range g.cams {
		if c.sees(g, at) {
			c.alert, c.led = camAlertTime, camAlertTime
			seen = true
		}
	}
	return seen
}

// camZone: the point lies in the area the camera covers over its whole sweep (range, walls, angle).
func (c *secCam) camZone(g *Game, q Vec) bool {
	if g.blackoutOn() {
		return false
	}
	rng := camRange
	if g.visK > 0 {
		rng *= g.visK
	}
	if dist(c.eye, q) > rng {
		return false
	}
	da := math.Atan2(q.Y-c.eye.Y, q.X-c.eye.X) - c.base
	for da > math.Pi {
		da -= 2 * math.Pi
	}
	for da < -math.Pi {
		da += 2 * math.Pi
	}
	return math.Abs(da) <= camSweep+camHalf && g.lvl.LOS(c.eye, q)
}

// camFreshBody: a knife body of p, younger than camRecent s, still lying in the camera's zone.
func (g *Game) camFreshBody(c *secCam, p *Player) bool {
	if len(g.camStabs) == 0 {
		return false
	}
	for _, b := range g.bodies {
		if g.camStabs[b] == p && !b.bagged && !b.carried && g.time-b.born <= camRecent && c.camZone(g, b.pos) {
			return true
		}
	}
	return false
}

// camLocates: why this camera reports p now ("" = it does not). p is inside its cone.
func (g *Game) camLocates(c *secCam, p *Player) string {
	switch {
	case g.isSuspectP(p):
		return "wanted"
	case g.playerAlert(p):
		return "alert"
	case g.camFreshBody(c, p):
		return "body"
	}
	return ""
}

// camReport: the camera tells the police where p is. Officers on the scene: the nearest free one walks
// there. Crew on its way: it heads there. No police yet: they are called (raisePolice keeps the rule of a
// single call). Only a WANTED player is reported as "informed" (the cuffs stay on); the rest is a location.
func (g *Game) camReport(c *secCam, p *Player, why string) {
	c.alert = math.Max(c.alert, 1.5)
	c.led = math.Max(c.led, camAlertTime)
	if i := g.playerIndex(p); i < 2 {
		c.cool[i] = camReportCD
	}
	g.run.camSeen++
	g.withPlayer(p, func() {
		g.say(camMsg)
		if why == "wanted" {
			g.killPos = p.pos
		}
		if !g.policeCalled {
			g.alertPos = p.pos
			g.raisePolice(p.pos, why == "wanted")
			return
		}
		var best *Cop
		for _, cp := range g.cops {
			if cp.medic || (cp.state != cSweep && cp.state != cGoto && cp.state != cSearch && cp.state != cReturn) {
				continue
			}
			if best == nil || dist(cp.pos, p.pos) < dist(best.pos, p.pos) {
				best = cp
			}
		}
		if best == nil {
			g.alertPos = p.pos // the crew on its way goes for the tip
			return
		}
		g.releaseCop(best)
		best.state, best.wait = cGoto, 0
		best.path = g.lvl.Path(best.pos, p.pos)
		best.icon, best.iconT = g.line(copCategory(best.kind), "understood"), 2
	})
}

func (g *Game) updateCameras(dt float64) {
	if len(g.cams) == 0 || g.lvl == nil {
		return
	}
	rng := camRange
	if g.visK > 0 {
		rng *= g.visK
	}
	for _, c := range g.cams {
		if !c.still {
			c.t += dt
			c.ang = c.aim()
		}
		if c.led > 0 {
			c.led -= dt
		}
		if c.alert > 0 {
			c.alert -= dt
		}
		for i := range c.cool {
			if c.cool[i] > 0 {
				c.cool[i] -= dt
			}
		}
		for _, p := range g.players() {
			if g.camWatched(p) && c.sees(g, p.pos) {
				c.led = math.Max(c.led, camLedTime)
				if g.stealth {
					c.alert = math.Max(c.alert, 0.35)
					continue // stealth has its own detection meter (stealth.go)
				}
				why := g.camLocates(c, p)
				if why == "wanted" {
					g.seenNow = true // filmed: the search does not cool down while he is on screen
					g.killPos = p.pos
				}
				if i := g.playerIndex(p); why != "" && i < 2 && c.cool[i] <= 0 {
					g.camReport(c, p, why)
				}
			}
		}
		for i := range c.rays {
			a := c.ang - camHalf + 2*camHalf*float64(i)/float64(camRays-1)
			dx, dy := math.Cos(a), math.Sin(a)
			at := func(r float64) Vec { return Vec{c.eye.X + dx*r, c.eye.Y + dy*r} }
			if g.lvl.LOS(c.eye, at(rng)) {
				c.rays[i] = rng
				continue
			}
			lo, hi := 0.0, rng
			for k := 0; k < 7; k++ {
				mid := (lo + hi) / 2
				if g.lvl.LOS(c.eye, at(mid)) {
					lo = mid
				} else {
					hi = mid
				}
			}
			c.rays[i] = lo
		}
	}
}

// ---- drawing ----------------------------------------------------------------------------------------

func (g *Game) drawCameras(dst *ebiten.Image) {
	if len(g.cams) == 0 || g.lvl == nil || g.mode != ModePlay || g.blackoutOn() { // no power: no cones
		return
	}
	for _, c := range g.cams {
		g.drawCamCone(dst, c)
	}
	for _, c := range g.cams {
		g.drawCamBody(dst, c)
	}
}

func (g *Game) drawCamCone(dst *ebiten.Image, c *secCam) {
	rng := camRange
	if g.visK > 0 {
		rng *= g.visK
	}
	ex, ey := g.wx(c.eye.X), g.wy(c.eye.Y)
	if ex < -rng || ex > screenW+rng || ey < -rng || ey > screenH+rng {
		return
	}
	hot := c.alert > 0
	base := 40.0
	hex := uint32(0x18e060)
	if hot {
		base, hex = 66, 0xff2a1a
		if int(g.time*8)%2 == 0 {
			base = 52 // a slight flicker while it is alarmed
		}
	}
	for y := -rng; y <= rng; y += camCell {
		for x := -rng; x <= rng; x += camCell {
			d := math.Hypot(x+camCell/2, y+camCell/2)
			if d > rng || d < 4 {
				continue
			}
			da := math.Atan2(y+camCell/2, x+camCell/2) - c.ang
			for da > math.Pi {
				da -= 2 * math.Pi
			}
			for da < -math.Pi {
				da += 2 * math.Pi
			}
			if math.Abs(da) > camHalf {
				continue
			}
			i := int(math.Round((da + camHalf) / (2 * camHalf) * float64(camRays-1)))
			i = int(clampF(float64(i), 0, camRays-1))
			if d > c.rays[i] {
				continue
			}
			a := base * (1 - 0.5*d/rng)
			if math.Abs(da) > camHalf*0.9 || d > c.rays[i]-camCell*1.5 { // brighter rim: the cone reads on any floor
				a = math.Min(a*1.7, 120)
			}
			fillRect(dst, math.Round(ex+x), math.Round(ey+y), camCell, camCell, rgba(hex, uint8(a)))
		}
	}
}

// drawCamBody: bracket on the wall, arm, the housing turned toward where it looks, the lens and a LED that
// blinks dim red slowly (recording) or bright red fast (it has seen somebody).
func (g *Game) drawCamBody(dst *ebiten.Image, c *secCam) {
	mx, my := math.Round(g.wx(c.mount.X)), math.Round(g.wy(c.mount.Y))
	if mx < -30 || mx > screenW+30 || my < -30 || my > screenH+30 {
		return
	}
	f := math.Cos(c.ang) // -1 looks left, +1 looks right, 0 straight down (front)
	dark, steel, hi, shade := rgb(0x14161e), rgb(0x6c7588), rgb(0xa4adc0), rgb(0x474e60)
	fillRect(dst, mx-8, my+9, 16, 2, rgba(0x000000, 50)) // soft shadow on the floor
	fillRect(dst, mx-6, my-5, 12, 5, dark)               // wall plate
	fillRect(dst, mx-5, my-4, 10, 3, shade)
	fillRect(dst, mx-5, my-4, 10, 1, hi)
	fillRect(dst, mx-4, my-3, 1, 1, dark) // screws
	fillRect(dst, mx+3, my-3, 1, 1, dark)
	ax := math.Round(f * 1)
	fillRect(dst, mx-2+ax, my, 4, 3, dark) // swivel arm
	fillRect(dst, mx-1+ax, my, 2, 3, steel)
	side := math.Abs(f) > 0.4
	w := 14.0
	if side {
		w = 12
	}
	hx, hy := math.Round(mx-w/2+f*3), my+3
	fillRect(dst, hx-1, hy-1, w+2, 10, dark) // housing outline
	fillRect(dst, hx, hy, w, 8, steel)
	fillRect(dst, hx, hy, w, 2, hi)
	fillRect(dst, hx, hy+6, w, 2, shade)
	// the lens: round and dark when it looks at us, a short barrel on the side it looks to
	if side {
		lx := hx + w - 3
		if f < 0 {
			lx = hx - 1
		}
		fillRect(dst, lx, hy+1, 4, 7, dark)
		fillRect(dst, lx+1, hy+2, 2, 5, rgb(0x0c1a34))
		fillRect(dst, lx+1, hy+2, 1, 1, rgb(0xe8f4ff))
		vx := hx
		if f < 0 {
			vx = hx + w - 3
		}
		fillRect(dst, vx, hy-2, 3, 2, shade) // sun visor
	} else {
		lx := math.Round(hx + w/2 - 3)
		fillRect(dst, lx-1, hy+1, 8, 7, dark)
		fillRect(dst, lx, hy+2, 6, 5, rgb(0x0c1a34))
		fillRect(dst, lx+1, hy+3, 4, 3, rgb(0x1e4a78))
		fillRect(dst, lx+1, hy+3, 2, 1, rgb(0xe8f4ff)) // glint
		fillRect(dst, hx+1, hy-2, w-2, 2, shade)       // sun visor
	}
	// LED: on the rear top corner
	lx := hx + 1
	if f < 0 {
		lx = hx + w - 3
	}
	var col uint32
	on := false
	switch {
	case c.led > 0:
		col, on = 0xff2a1a, int(g.time*7)%2 == 0
	default: // REC: a slow dim red blink, the camera is on (off in a blackout: not drawn)
		col, on = 0xc02018, math.Mod(g.time*0.9+c.t*0.13, 1) < 0.55
	}
	fillRect(dst, lx, hy-1, 2, 2, dark)
	if on {
		fillRect(dst, lx-1, hy-2, 4, 4, rgba(col, 60)) // glow
		fillRect(dst, lx, hy-1, 2, 2, rgb(col))
	} else {
		fillRect(dst, lx, hy-1, 2, 2, rgb(0x2a3a30))
	}
}

// strings of the cameras in English (same pattern as stealth.go: lang_data.go stays untouched)
func init() {
	for es, en := range map[string]string{
		camMsg: "CAMERA: you're on tape",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}

// debugCamKill (-camkill): puts the player 60 px in front of the first camera (frozen where it looks), a guest
// beside him, and kills the guest as if the player had stabbed him: the lens catches it (red cone, police, BUSCADO).
func (g *Game) debugCamKill() {
	if len(g.cams) == 0 || g.lvl == nil || g.p == nil {
		return
	}
	c := g.cams[0]
	c.still = true
	dx, dy := math.Cos(c.ang), math.Sin(c.ang)
	for _, r := range []float64{70, 60, 50, 40, 30} {
		q := Vec{c.eye.X + dx*r, c.eye.Y + dy*r}
		side := Vec{q.X + 16, q.Y}
		if !g.lvl.Free(q) || g.lvl.SolidP(q.X, q.Y) || !g.lvl.LOS(c.eye, q) || !g.lvl.Free(side) || g.lvl.SolidP(side.X, side.Y) {
			continue
		}
		var gu *Guest
		for _, o := range g.guests {
			if !o.out() {
				gu = o
				break
			}
		}
		if gu == nil {
			return
		}
		g.p.pos, gu.pos = q, side
		g.killGuest(gu, true)
		return
	}
}

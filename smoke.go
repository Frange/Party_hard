package main

import (
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// SMOKE GRENADE (idea 216). Sometimes a party has one or two smoke grenades lying about (next to a table, a
// crate, a wardrobe, a bag...). The player picks one up with the use key (the slot rules of the poison and the
// disguise: one in the hand, the other kind waits in the second slot) and throws it with the same key when there
// is nothing else to use near him.
//
//   - "sometimes": smokeRoll decides with its own dice (session seed + level + party count; g.rng never moves)
//     about one party in three (smokeChance), with 1 or 2 grenades. Never in the STEALTH mode.
//   - the throw: it flies in an arc in the direction the player faces, lands at ~115 px, bounces once and rolls
//     to a stop at ~130-140 px; a wall or a solid prop stops it (it bounces off, never goes through: SolidP).
//   - the cloud: a circle of radius smokeR (80 px) that lasts EXACTLY smokeDur = 6 s. The 6 s include a quick
//     appearance (smokeFadeIn, 0.4 s) and the final fade (smokeFadeOut, the last 0.6 s): the whole 6 s the smoke is
//     ACTIVE (sight blocked, people dazed); after 6 s it is gone at once.
//   - while it lasts NOBODY sees into it, out of it or through it: g.sees (guests, guards, officers, killers) and the
//     security cameras (secCam.sees) are false when the eye or the target is inside a cloud or the line between
//     them crosses one (smokeBlocks; the walls keep blocking as always with LOS). So a kill in the smoke has no
//     witness, no call and no cuffs. The bodies are found later as usual.
//   - guests, guards and officers INSIDE are dazed (Guest/Guard/Cop.smogT): they stand still coughing, turn
//     about at random, with the stars of the VIP stun over the head, for the rest of the smoke + smokeDazeExtra
//     (1 s). No stampede, no call, no chase; then they go back to what they were doing. An officer keeps counting
//     his time on scene and never leaves while dazed (the minimum 20 s rule is untouched).
//   - zombies are not affected (they already see nobody as a witness and a cloud does not stop a zombie); the
//     T-Rex and the animals neither. The players are never affected: they see the cloud as a see-through veil.
//   - not a crime: throwing it raises no search and no stars; nothing is saved or counted.
//   - limits: one grenade in the hand per player, smokeMaxClouds clouds + grenades in the air at once.
//
// Debug: -grenade 1 puts two grenades in every party (one next to the start); -grenade 2 also gives one to the
// player and throws it at the biggest group of guests at ~2 s (to look at it in captures).
// ---------------------------------------------------------------------------

const (
	smokeDur       = 6.0  // seconds the cloud is active (the fades are inside these 6 s)
	smokeFadeIn    = 0.4  // seconds it takes to fill
	smokeFadeOut   = 0.6  // the last seconds, thinning out
	smokeR         = 80.0 // radius of the cloud
	smokeDazeExtra = 1.0  // seconds the dazed stay dazed after the smoke
	smokeMaxClouds = 3    // clouds + grenades in the air at once
	smokeChance    = 1.0 / 3
	smokeSpeed     = 230.0 // ground speed of the throw (px/s)
	smokeLift      = 120.0 // upward speed of the throw (px/s)
	smokeGrav      = 480.0 // gravity of the arc (px/s²): first landing after 0.5 s, ~115 px away, 15 px high
	smokeVeilA     = 0.42  // alpha of the veil at its thickest
	smokeKind      = "smokegrenade"
)

type smokeNade struct {
	pos, vel Vec
	h, vh    float64 // height above the ground and its speed
	bounced  bool
	t        float64
	owner    int
}

type smokeCloud struct {
	pos   Vec
	t     float64 // seconds since it burst
	owner int
}

type smokeState struct {
	nades  []smokeNade
	clouds []smokeCloud
	demoT  float64 // -grenade 2: when the demo throw happens (0 = done / none)
}

var (
	dbgGrenade int   // -grenade N (debug)
	smokeSeed  int64 // tests: non-zero = deterministic session seed
	smokeBase  int64 // the session seed (set once)
	smokeLoads int64 // parties loaded this session
)

func init() {
	propKinds[smokeKind] = &PropKind{W: 12, H: 14, FW: 8, FH: 4, NoSolid: true, Item: "smoke", Draw: func(b *pixBuf, v int) {
		b.ellipse(6, 13, 5, 1.4, color.RGBA{0, 0, 0, 70})
		b.rect(2, 4, 8, 9, c(0x4a5a48)) // the can
		b.rect(2, 4, 2, 9, c(0x6e8068)) // light side
		b.rect(2, 7, 8, 2, c(0xd8d8d0)) // the white band
		b.rect(4, 1, 4, 3, c(0x2a2a30)) // the top
		b.rect(8, 1, 2, 6, c(0x9aa0a8)) // the lever
		b.set(10, 2, c(0xd8c040))       // the ring of the pin
		b.set(11, 3, c(0xd8c040))
		b.set(10, 4, c(0xd8c040))
	}}
}

// smokeRoll: how many grenades a party gets (0, 1 or 2) for the given seed and level. Pure.
func smokeRoll(seed int64, level int) int {
	r := rand.New(rand.NewSource(seed ^ int64(level+1)*0x9E3779B1))
	if r.Float64() >= smokeChance {
		return 0
	}
	if r.Float64() < 0.4 {
		return 2
	}
	return 1
}

// smokeLevelReset is called when a level loads: no smoke, and maybe a grenade or two on the floor.
func (g *Game) smokeLevelReset() {
	g.smk = smokeState{}
	if g.lvl == nil {
		return
	}
	// a second call on the same level (the debug flag) does not pile grenades up
	keep := g.lvl.Props[:0]
	for _, pr := range g.lvl.Props {
		if pr.Name != smokeKind {
			keep = append(keep, pr)
		}
	}
	g.lvl.Props = keep
	if g.stealth {
		return
	}
	if smokeBase == 0 {
		smokeBase = smokeSeed
		if smokeBase == 0 {
			smokeBase = time.Now().UnixNano()
		}
	}
	smokeLoads++
	seed := smokeBase ^ smokeLoads*0x85EBCA77
	n := smokeRoll(seed, g.levelIdx)
	if dbgGrenade > 0 {
		n = 2
		if dbgGrenade >= 2 {
			g.smk.demoT = 2
		}
	}
	if n > 0 {
		g.placeSmoke(n, rand.New(rand.NewSource(seed+7)), dbgGrenade > 0)
	}
}

// smokeAnchor: props next to which a grenade looks forgotten.
var smokeAnchor = map[string]bool{"table": true, "cafetable": true, "clothtable": true, "longtable": true, "buffet": true,
	"bar": true, "counter": true, "minibar": true, "crate": true, "barrel": true, "closet": true, "wardrobe": true,
	"giftable": true, "shelf": true, "bookshelf": true, "disguisebag": true}

// smokeSpotOK: a grenade can lie at p (free, reachable, dry floor, not on another thing, away from other items).
func (l *Level) smokeSpotOK(p Vec, reach []bool) bool {
	k := propKinds[smokeKind]
	if p.X < 40 || p.X > l.W-40 || p.Y < 70 || p.Y > l.H-30 || !l.Free(p) || l.SolidP(p.X, p.Y) || l.InPool(p) || l.badGround(p) {
		return false
	}
	ci := int(p.Y/navCell)*l.cw + int(p.X/navCell)
	if ci < 0 || ci >= len(reach) || !reach[ci] {
		return false
	}
	if l.propClashes(smokeKind, k, p, 0) {
		return false
	}
	for _, o := range l.Props {
		if (o.K.Item != "" || o.K.Phone || o.K.Trap != "") && dist(Vec{o.X, o.Y}, p) < 40 {
			return false
		}
	}
	return true
}

// placeSmoke puts n grenades next to tables, crates, wardrobes, bags... (any free floor if there are none).
// near: the first one goes next to the start (debug).
func (g *Game) placeSmoke(n int, r *rand.Rand, near bool) {
	l := g.lvl
	k := propKinds[smokeKind]
	reach := l.reachFrom(l.Def.Start)
	var anchors []*Prop
	for _, pr := range l.Props {
		if smokeAnchor[pr.Name] && !pr.gone {
			anchors = append(anchors, pr)
		}
	}
	r.Shuffle(len(anchors), func(i, j int) { anchors[i], anchors[j] = anchors[j], anchors[i] })
	add := func(p Vec) {
		l.Props = append(l.Props, &Prop{Name: smokeKind, K: k, X: p.X, Y: p.Y, Foot: Rect{p.X - k.FW/2, p.Y - k.FH, k.FW, k.FH}, Sort: p.Y})
	}
	placed := 0
	if near {
		st := l.Def.Start
		for _, o := range []Vec{{24, 4}, {-24, 4}, {0, 24}, {24, 20}, {-24, 20}, {36, 0}, {-36, 0}, {0, -26}} {
			p := Vec{math.Round(st.X + o.X), math.Round(st.Y + o.Y)}
			if l.Free(p) && !l.SolidP(p.X, p.Y) && !l.InPool(p) {
				add(p)
				placed++
				break
			}
		}
	}
	for _, a := range anchors {
		if placed >= n {
			return
		}
		hw := a.Foot.W/2 + 10
		for _, o := range []Vec{{0, 12}, {-hw, 4}, {hw, 4}, {0, 22}, {-hw, 16}, {hw, 16}} {
			p := Vec{math.Round(a.X + o.X), math.Round(a.Y + o.Y)}
			if dist(p, l.Def.Start) > 80 && l.smokeSpotOK(p, reach) {
				add(p)
				placed++
				break
			}
		}
	}
	for tries := 0; placed < n && tries < 200; tries++ {
		p := l.RandomRoam(r)
		p = Vec{math.Round(p.X), math.Round(p.Y)}
		if dist(p, l.Def.Start) > 80 && l.smokeSpotOK(p, reach) {
			add(p)
			placed++
		}
	}
}

// ---- throwing -------------------------------------------------------------------------------------------------

// smokeHas: the player carries a grenade (in the hand or in the second slot).
func smokeHas(p *Player) bool { return p != nil && (p.item == "smoke" || p.spare == "smoke") }

// smokeThrowOK: the player can throw his grenade now.
func (g *Game) smokeThrowOK(p *Player) bool {
	return smokeHas(p) && !p.dead && !p.hidden && !p.jail.held && p.carrying == nil && g.lvl != nil
}

// throwSmoke: the player throws his grenade where he looks. True = thrown.
func (g *Game) throwSmoke(p *Player) bool {
	if !g.smokeThrowOK(p) {
		return false
	}
	if len(g.smk.clouds)+len(g.smk.nades) >= smokeMaxClouds {
		g.say("Ya hay demasiado humo: espera un poco")
		return false
	}
	if p.item == "smoke" {
		p.item = ""
		p.promoteSpare()
	} else {
		p.spare, p.spareDoses = "", 0
	}
	v := dirVec(p.dir)
	g.smk.nades = append(g.smk.nades, smokeNade{pos: p.pos, vel: Vec{v.X * smokeSpeed, v.Y * smokeSpeed}, vh: smokeLift, owner: p.idx})
	g.snd.Play("boing")
	g.say("¡Granada de humo!")
	return true
}

// stepNade moves a grenade one frame. True = it bursts now.
func (g *Game) stepNade(n *smokeNade, dt float64) bool {
	n.t += dt
	l := g.lvl
	// sideways: never into a solid; a wall sends it back, slower (that is its bounce)
	if nx := n.pos.X + n.vel.X*dt; l.SolidP(nx, n.pos.Y) {
		n.vel.X = -n.vel.X * 0.4
		n.bounced = true
	} else {
		n.pos.X = nx
	}
	if ny := n.pos.Y + n.vel.Y*dt; l.SolidP(n.pos.X, ny) {
		n.vel.Y = -n.vel.Y * 0.4
		n.bounced = true
	} else {
		n.pos.Y = ny
	}
	// up and down
	if n.h > 0 || n.vh > 0 {
		n.vh -= smokeGrav * dt
		n.h += n.vh * dt
		if n.h <= 0 {
			n.h = 0
			if !n.bounced { // the one bounce
				n.bounced = true
				n.vh = -n.vh * 0.35
				n.vel.X, n.vel.Y = n.vel.X*0.4, n.vel.Y*0.4
				g.snd.Play("thud")
			} else {
				n.vh = 0
			}
		}
	} else { // rolling to a stop
		sp := math.Hypot(n.vel.X, n.vel.Y)
		if ns := sp - 300*dt; ns > 0 {
			n.vel.X, n.vel.Y = n.vel.X*ns/sp, n.vel.Y*ns/sp
		} else {
			n.vel = Vec{}
		}
	}
	stopped := n.h <= 0 && n.vh <= 0 && n.vel.X == 0 && n.vel.Y == 0
	return stopped || n.t > 2.5
}

func (g *Game) burstSmoke(at Vec, owner int) {
	g.smk.clouds = append(g.smk.clouds, smokeCloud{pos: at, owner: owner})
	g.snd.Play("ignite")
	g.burst(Vec{at.X, at.Y - 6}, 14, 60, rgb(0xd8dce4), 0.5, 3, -20)
}

// ---- update ---------------------------------------------------------------------------------------------------

func (g *Game) updateSmoke(dt float64) {
	s := &g.smk
	if s.demoT > 0 && g.time >= s.demoT {
		s.demoT = 0
		g.smokeDemoThrow()
	}
	if len(s.nades) > 0 {
		out := s.nades[:0]
		for i := range s.nades {
			n := s.nades[i]
			if g.stepNade(&n, dt) {
				g.burstSmoke(n.pos, n.owner)
				continue
			}
			out = append(out, n)
		}
		s.nades = out
	}
	if len(s.clouds) == 0 {
		return
	}
	co := s.clouds[:0]
	for _, c := range s.clouds {
		c.t += dt
		if c.t < smokeDur-1e-9 {
			co = append(co, c)
		}
	}
	s.clouds = co
	for _, c := range s.clouds {
		g.smokeDazeAround(c)
	}
}

// smokeDazeAround: everybody but the players inside the cloud is dazed until it ends (+1 s).
func (g *Game) smokeDazeAround(c smokeCloud) {
	t := smokeDur - c.t + smokeDazeExtra
	for _, gu := range g.guests {
		if gu.out() || gu.state == gZombie || gu.state == gLift || gu.py != nil || dist(gu.pos, c.pos) > smokeR {
			continue
		}
		if gu.smogT <= 0 && math.Mod(gu.off*7, 3) < 1 && gu.iconT <= 0 { // one in three coughs out loud
			gu.icon, gu.iconT = "*COF COF*", 1.4
		}
		gu.smogT = math.Max(gu.smogT, t)
	}
	for _, gd := range g.guards {
		if !gd.dead && dist(gd.pos, c.pos) <= smokeR {
			if gd.smogT <= 0 {
				gd.icon, gd.iconT = "*COF*", 1.2
			}
			gd.smogT = math.Max(gd.smogT, t)
		}
	}
	for _, o := range g.cops {
		if o.state != cLeave && dist(o.pos, c.pos) <= smokeR {
			if o.smogT <= 0 {
				o.icon, o.iconT = "*COF*", 1.2
			}
			o.smogT = math.Max(o.smogT, t)
		}
	}
}

// smokeDazeDir: the dazed turn about now and then, without walking.
func smokeDazeDir(t, off float64) Dir {
	return Dir(int(t*1.2+off*5) % 4)
}

// smokeDazedGuest: true = the guest is dazed this frame (nothing else to do).
func (g *Game) smokeDazedGuest(gu *Guest, dt float64) bool {
	if gu.smogT <= 0 {
		return false
	}
	gu.smogT -= dt
	gu.moving = false
	gu.dir = smokeDazeDir(g.time, gu.off)
	if gu.iconT > 0 {
		if gu.iconT -= dt; gu.iconT <= 0 {
			gu.icon = ""
		}
	}
	return true
}

func (g *Game) smokeDazedGuard(gd *Guard, dt float64) bool {
	if gd.smogT <= 0 {
		return false
	}
	gd.smogT -= dt
	gd.moving, gd.sight = false, 0
	gd.dir = smokeDazeDir(g.time, gd.home.X*0.01)
	return true
}

func (g *Game) smokeDazedCop(c *Cop, dt float64) bool {
	if c.smogT <= 0 || c.state == cLeave {
		c.smogT = 0
		return false
	}
	c.smogT -= dt
	c.moving = false
	c.dir = smokeDazeDir(g.time, c.lastSeen.X*0.01)
	return true
}

// ---- sight ----------------------------------------------------------------------------------------------------

// smokeBlocks: a cloud stands between a and b (one of them inside, or the line crosses it).
func (g *Game) smokeBlocks(a, b Vec) bool {
	for i := range g.smk.clouds {
		c := g.smk.clouds[i].pos
		if segDist(c, a, b) <= smokeR { // (murray.go)
			return true
		}
	}
	return false
}

// smokeActive: some cloud is up (tests, HUD).
func (g *Game) smokeActive() bool { return len(g.smk.clouds) > 0 }

// ---- debug demo -----------------------------------------------------------------------------------------------

// smokeDemoThrow (-grenade 2): the player gets a grenade and throws it at the biggest group of guests.
func (g *Game) smokeDemoThrow() {
	p := g.first()
	if p == nil || g.lvl == nil {
		return
	}
	var best Vec
	bn := -1
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" {
			continue
		}
		n := 0
		for _, o := range g.guests {
			if !o.out() && dist(o.pos, gu.pos) < 60 {
				n++
			}
		}
		if n > bn {
			best, bn = gu.pos, n
		}
	}
	if bn < 0 {
		return
	}
	for _, o := range []Vec{{-135, 0}, {135, 0}, {0, 135}, {0, -135}, {-100, 0}, {100, 0}, {0, 100}, {0, -100}} {
		q := Vec{best.X + o.X, best.Y + o.Y}
		if g.lvl.Free(q) && !g.lvl.SolidP(q.X, q.Y) && g.lvl.LOS(q, best) {
			p.pos, p.dir = q, faceTo(q, best)
			break
		}
	}
	p.item, p.spare = "smoke", ""
	g.withPlayer(p, func() { g.throwSmoke(p) })
}

// ---- drawing --------------------------------------------------------------------------------------------------

var (
	smokeStarImg   *ebiten.Image // a stun star (as drawVIPStars draws them)
	smokeNadeImg   *ebiten.Image // the grenade in the air
	smokeShadowImg *ebiten.Image // ...and its shadow
	smokeVeilImg   *ebiten.Image // the cloud, built once
	smokePuffImg   *ebiten.Image // a small curl that drifts over it
	smokeOrbit     [6]Vec        // where the curls sit (unit circle, scaled), fixed
)

// smokeImages builds the veil and the curl once: pixel art, premultiplied, no work per frame.
func smokeImages() {
	if smokeVeilImg != nil {
		return
	}
	R := int(smokeR) + 8
	sz := 2 * R
	b := newPix(sz, sz)
	for y := 0; y < sz; y++ {
		for x := 0; x < sz; x++ {
			dx, dy := float64(x-R)+0.5, float64(y-R)+0.5
			d := math.Hypot(dx, dy)
			a := math.Atan2(dy, dx)
			edge := smokeR + 5*math.Sin(a*5) + 3*math.Sin(a*9+1) // a lumpy outline
			if d > edge {
				continue
			}
			k := 1.0
			if d > edge-14 {
				k = 0.55 // the thin rim
			}
			// lighter knots: the volutes inside
			lump := math.Sin(dx*0.11)*math.Sin(dy*0.13) + 0.6*math.Sin((dx+dy)*0.07)
			grey := 196.0
			if lump > 0.55 {
				grey = 222
			} else if lump < -0.5 {
				grey = 168
			}
			al := smokeVeilA * k
			if (x+y)%2 == 0 && k < 1 { // dithered rim: crisp, no blur
				al *= 0.6
			}
			A := uint8(255 * al)
			v := uint8(grey * al)
			i := (y*sz + x) * 4
			b.p[i], b.p[i+1], b.p[i+2], b.p[i+3] = v, v, uint8(math.Min(255, (grey+10)*al)), A
		}
	}
	smokeVeilImg = b.img()
	pr := 14
	pb := newPix(2*pr, 2*pr)
	for y := 0; y < 2*pr; y++ {
		for x := 0; x < 2*pr; x++ {
			d := math.Hypot(float64(x-pr)+0.5, float64(y-pr)+0.5)
			if d > float64(pr) {
				continue
			}
			al := 0.18
			if d > float64(pr)-4 {
				al = 0.1
			}
			v := uint8(232 * al)
			i := (y*2*pr + x) * 4
			pb.p[i], pb.p[i+1], pb.p[i+2], pb.p[i+3] = v, v, v, uint8(255*al)
		}
	}
	smokePuffImg = pb.img()
	st := newPix(7, 7)
	st.rect(0, 2, 7, 3, rgb(0x3a2a00)) // outline: a plus 7 long
	st.rect(2, 0, 3, 7, rgb(0x3a2a00))
	st.rect(1, 3, 5, 1, rgb(0xf8d820))
	st.rect(3, 1, 1, 5, rgb(0xf8d820))
	st.rect(2, 2, 3, 3, rgb(0xf8d820))
	st.set(3, 3, rgb(0xffffff))
	smokeStarImg = st.img()
	nb := newPix(4, 7)
	nb.rect(1, 0, 2, 1, rgb(0x2a2a30))
	nb.rect(0, 1, 4, 6, rgb(0x4a5a48))
	nb.rect(0, 3, 4, 1, rgb(0xd8d8d0))
	smokeNadeImg = nb.img()
	sh := newPix(6, 2)
	sh.rect(0, 0, 6, 2, color.RGBA{0, 0, 0, 90})
	smokeShadowImg = sh.img()
	for i := range smokeOrbit {
		a := float64(i) * 2 * math.Pi / float64(len(smokeOrbit))
		smokeOrbit[i] = Vec{math.Cos(a), math.Sin(a)}
	}
}

// smokeAlpha: how thick a cloud of age t looks (0..1).
func smokeAlpha(t float64) float64 {
	k := math.Min(1, t/smokeFadeIn)
	if left := smokeDur - t; left < smokeFadeOut {
		k = math.Min(k, left/smokeFadeOut)
	}
	return clampF(k, 0, 1)
}

// drawSmoke: the grenades in the air, the clouds as a see-through veil (below the bubbles and the HUD) and the
// stars over the dazed.
func (g *Game) drawSmoke(dst *ebiten.Image) {
	s := &g.smk
	defer g.drawSmokeStars(dst)
	if len(s.nades) == 0 && len(s.clouds) == 0 {
		return
	}
	smokeImages()
	op := &smokeOp
	for i := range s.nades {
		n := &s.nades[i]
		x, y := math.Round(g.wx(n.pos.X)), math.Round(g.wy(n.pos.Y))
		op.GeoM.Reset()
		op.ColorScale.Reset()
		op.GeoM.Translate(x-3, y-1)
		dst.DrawImage(smokeShadowImg, op) // the shadow on the ground
		op.GeoM.Reset()
		op.GeoM.Translate(x-2, y-math.Round(n.h)-7)
		dst.DrawImage(smokeNadeImg, op)
	}
	for i := range s.clouds {
		c := &s.clouds[i]
		k := float32(smokeAlpha(c.t))
		grow := 0.7 + 0.3*math.Min(1, c.t/smokeFadeIn) // it spreads out as it fills
		half := (smokeR + 8) * grow
		op.GeoM.Reset()
		op.ColorScale.Reset()
		op.GeoM.Scale(grow, grow)
		op.GeoM.Translate(math.Round(g.wx(c.pos.X)-half), math.Round(g.wy(c.pos.Y)-half))
		op.ColorScale.Scale(k, k, k, k)
		dst.DrawImage(smokeVeilImg, op)
		// the curls drift slowly round the middle
		for j, o := range smokeOrbit {
			a := c.t*0.5 + float64(j)
			r := smokeR * (0.35 + 0.15*float64(j%3))
			ca, sa := math.Cos(a), math.Sin(a)
			px := (o.X*ca - o.Y*sa) * r * grow
			py := (o.X*sa + o.Y*ca) * r * grow * 0.8
			op.GeoM.Reset()
			op.GeoM.Translate(math.Round(g.wx(c.pos.X)+px-14), math.Round(g.wy(c.pos.Y)+py-14))
			dst.DrawImage(smokePuffImg, op)
		}
	}
}

// smokeOp: the draw options of the clouds, reused (nothing allocated per frame).
var smokeOp ebiten.DrawImageOptions

// drawSmokeStars: the stars over everybody still dazed (also during the extra second after the smoke), the
// same three circling stars as the VIP stun (drawVIPStars) but from a sprite made once.
func (g *Game) drawSmokeStars(dst *ebiten.Image) {
	for _, gu := range g.guests {
		if gu.smogT > 0 && !gu.out() {
			g.smokeStarsAt(dst, gu.pos)
		}
	}
	for _, gd := range g.guards {
		if gd.smogT > 0 && !gd.dead {
			g.smokeStarsAt(dst, gd.pos)
		}
	}
	for _, o := range g.cops {
		if o.smogT > 0 {
			g.smokeStarsAt(dst, o.pos)
		}
	}
}

func (g *Game) smokeStarsAt(dst *ebiten.Image, pos Vec) {
	smokeImages()
	op := &smokeOp
	op.ColorScale.Reset()
	cx, cy := math.Round(g.wx(pos.X)), math.Round(g.wy(pos.Y)-44*g.charSc)
	for i := 0; i < 3; i++ {
		a := g.time*1.5*2*math.Pi + float64(i)*2*math.Pi/3
		op.GeoM.Reset()
		op.GeoM.Translate(math.Round(cx+math.Cos(a)*vipStarRX)-3, math.Round(cy+math.Sin(a)*vipStarRY)-3)
		dst.DrawImage(smokeStarImg, op)
	}
}

// smokeHudSuffix: what the OBJETO line of the HUD adds for a smoke grenade: " x1" when it is in the hand,
// " + GRANADA x1" when it waits in the second slot (short: the line must stay clear of the messages).
func smokeHudSuffix(p *Player) string {
	switch {
	case p.item == "smoke":
		return " x1"
	case p.spare == "smoke":
		return " + GRANADA x1"
	}
	return ""
}

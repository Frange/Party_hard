package main

import (
	"fmt"
	"image/color"
	"math"
	"reflect"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Two players: the funny, cooperative side (the user asked for more of it).
//
//   - the scoreboard at the end of the party (level done / game over), with a silly title each
//   - carrying the partner when he is on the floor (knocked down or dead)
//   - the high five, and the dance duo (both dance together: hearts, points, guests join in)
//   - crossed traps: the traps and the poisoned punch show the colour of who set them, and a trap
//     of the partner only stuns (2 s), it never kills
//   - the cop distraction: "Excuse me, officer, what time is it?" and the partner gets away
//   - small gags: twins, "colega!", "that is mine!", the dance bet, "you will pay for this"
//
// Everything here only runs with two players (g.is2P()); with one player nothing changes.
// The hooks in the rest of the game are one-liners that call into this file:
//   updatePlay      coopTrack / coopUpdate        (stats per player, timers)
//   updatePlayer    coopSpeedMul                  (carrying slows down)
//   stab            coopBlockStab                 (no stabbing while carrying the partner)
//   interact        coopInteract, coopOwn         (carry / high five / distract; trap owner)
//   triggerTrap...  coopTrapFired                 (a trap hurts anybody: no sparing)
//   updateCop       coopCopDistracted             (the officer stops for 3 s)
//   draw            coopHidden, coopDrawWorld, coopDrawScoreboard
// ---------------------------------------------------------------------------

const (
	fiveWindow  = 1.2  // seconds the raised hand waits for the other one
	fiveCool    = 20.0 // seconds between two high fives
	fiveReach   = 26.0
	fivePts     = 25
	carryReach  = 22.0
	distractFor = 3.0 // seconds the officer stands still looking at the distractor
	distractCD  = 15.0
	distractMax = 40.0 // how close to the officer
	duoReach    = 40.0
	duoPtsPerS  = 10.0
	duoPtsMax   = 100
	trapStunT   = 2.0
)

// coopStats are the numbers of one player in the party (for the scoreboard).
type coopStats struct {
	points, kills int
	hits          int     // punches given to the partner
	floorT        float64 // seconds on the floor
	helps         int     // times he carried the partner (the ambulance agent also counts here)
	fives         int     // high fives
	danceT        float64 // seconds dancing
	distracts     int     // cops distracted
}

type coopFx struct {
	kind         int // fxHeart, fxNote, fxStar, fxRing
	x, y, vx, vy float64
	life, max    float64
	col          color.RGBA
}

const (
	fxHeart = iota
	fxNote
	fxStar
	fxRing
)

type coopState struct {
	lvl   *Level
	score [2]coopStats

	// the partner on the shoulders: carry[i] is the player that player i carries
	carry [2]*Player

	interAt  [2]float64 // when each one last pressed interact
	handUp   [2]float64 // raised hand until (g.time)
	fiveNext float64    // next high five allowed at

	distractNext [2]float64
	distract     map[*Cop]float64 // officer stands still until

	trapOwner map[*Prop]int
	lastTrap  *Prop
	lastTrapT float64

	duo      bool
	duoAcc   float64 // points earned in this duet
	duoTotal int     // points of the whole party from duets
	duoJoin  float64 // next time the guests around may join
	duoEnded float64
	danceSt  [2]float64 // when each one started dancing (0 = not dancing)
	betNext  float64
	betOn    bool

	prevKnock [2]float64
	hitBy     [2]float64 // when player i was last punched by the other one
	sayNext   [2]float64
	twinNext  float64
	greetNext float64
	apartT    float64
	mineNext  float64

	pointFrac float64
	demoDone  bool
	fx        []coopFx
}

// coopEnsure resets the state when a new party starts (a new Level).
func (g *Game) coopEnsure() {
	if g.coop.lvl != g.lvl || g.coop.trapOwner == nil {
		g.coop = coopState{lvl: g.lvl, trapOwner: map[*Prop]int{}, distract: map[*Cop]float64{}}
	}
	g.coopSfx()
}

// coopSfx adds the little sounds of this file the first time they are needed.
func (g *Game) coopSfx() {
	if g.snd == nil || g.snd.sfx == nil || g.snd.sfx["clap"] != nil {
		return
	}
	g.snd.sfx["clap"] = mix(noise(0.07, 0.9, 38), tone(1700, 0.05, 0.25, 45, 1), shift(noise(0.06, 0.7, 40), 0.05))
	g.snd.sfx["hype"] = mix(tone(660, 0.07, 0.3, 14, 1), shift(tone(880, 0.07, 0.3, 14, 1), 0.07), shift(tone(1320, 0.12, 0.3, 10, 1), 0.14))
	g.snd.sfx["boing"] = mix(tone(220, 0.25, 0.5, 6, 0), shift(tone(330, 0.2, 0.3, 8, 0), 0.04))
}

// coopSay puts a short cry over a player (not more often than every 2.5 s each).
func (g *Game) coopSay(p *Player, txt string) {
	if p == nil || p.dead {
		return
	}
	i := g.playerIndex(p)
	if g.time < g.coop.sayNext[i] {
		return
	}
	g.coop.sayNext[i] = g.time + 2.5
	p.msg, p.msgT = txt, 1.8
}

// coopSayNow is coopSay without the wait (the answer of the partner).
func (g *Game) coopSayNow(p *Player, txt string) {
	if p == nil || p.dead {
		return
	}
	g.coop.sayNext[g.playerIndex(p)] = g.time + 2.5
	p.msg, p.msgT = txt, 1.8
}

func pickOf(g *Game, list []string) string { return list[g.rng.Intn(len(list))] }

// ---------------------------------------------------------------------------
// stats per player
// ---------------------------------------------------------------------------

// coopTrack runs one player's update and credits to him what the party earned meanwhile:
// points, kills, and the punches he gave to the partner.
func (g *Game) coopTrack(i int, fn func()) {
	if !g.is2P() {
		fn()
		return
	}
	g.coopEnsure()
	pl := g.players()[i]
	o := g.otherPlayer(pl)
	pts, kl := g.points, g.kills
	knock := 0.0
	if o != nil {
		knock = o.knockT
	}
	fn()
	st := &g.coop.score[i]
	if d := g.points - pts; d > 0 {
		st.points += d
	}
	if d := g.kills - kl; d > 0 {
		st.kills += d
	}
	if o != nil && o.knockT >= hitDownT-0.05 && knock < 0.5 && !g.coopIsCarried(o) {
		st.hits++
		g.coop.hitBy[g.playerIndex(o)] = g.time
	}
}

// coopHelp counts a help (carry, revive) for player i: the ambulance code can call it too.
func (g *Game) coopHelp(i int) {
	if g.is2P() && i >= 0 && i < 2 {
		g.coopEnsure()
		g.coop.score[i].helps++
	}
}

// ---------------------------------------------------------------------------
// the update
// ---------------------------------------------------------------------------

func (g *Game) coopUpdate(dt float64) {
	if !g.is2P() {
		return
	}
	g.coopEnsure()
	g.coopDemoSetup()
	c := &g.coop
	pl := g.players()
	for i, p := range pl {
		if p.knockT > 0 && !p.dead && !g.coopIsCarried(p) {
			c.score[i].floorT += dt
		}
		// he stands up after a punch: "you will pay for this"
		if c.prevKnock[i] > 0 && p.knockT <= 0 && !p.dead && g.time-c.hitBy[i] < 10 && c.hitBy[i] > 0 {
			g.coopSayNow(p, pickOf(g, []string{"¡Me las pagarás!", "¡Eso es trampa!", "¡Te vi!", "¡Ya verás!"}))
			c.hitBy[i] = 0
		}
		c.prevKnock[i] = p.knockT
	}
	g.coopCarryUpdate(dt)
	g.coopDance(dt)
	g.coopMeet(dt)
	for cop, until := range c.distract {
		if g.time > until+1 {
			delete(c.distract, cop)
		}
	}
	// the particles
	out := c.fx[:0]
	for _, f := range c.fx {
		f.life -= dt
		if f.life <= 0 {
			continue
		}
		f.x += f.vx * dt
		f.y += f.vy * dt
		out = append(out, f)
	}
	c.fx = out
}

func (g *Game) coopFx(kind int, x, y, vx, vy, life float64, col color.RGBA) {
	if len(g.coop.fx) > 120 {
		return
	}
	g.coop.fx = append(g.coop.fx, coopFx{kind: kind, x: x, y: y, vx: vx, vy: vy, life: life, max: life, col: col})
}

// ---------------------------------------------------------------------------
// carrying the partner
// ---------------------------------------------------------------------------

func (g *Game) coopIsCarried(p *Player) bool {
	if !g.is2P() || p == nil {
		return false
	}
	for _, c := range g.coop.carry {
		if c == p {
			return true
		}
	}
	return false
}

// coopCarrying: does p carry the partner?
func (g *Game) coopCarrying(p *Player) bool {
	return g.is2P() && p != nil && g.coop.carry[g.playerIndex(p)] != nil
}

// coopHidden: the carried partner is not drawn by himself (he is on the shoulders of the other).
func (g *Game) coopHidden(p *Player) bool { return g.coopIsCarried(p) }

func (g *Game) coopSpeedMul(p *Player) float64 {
	if g.coopCarrying(p) {
		return 0.72
	}
	return 1
}

func (g *Game) coopBlockStab(p *Player) bool {
	if g.coopCarrying(p) {
		g.say("Suelta a tu compañero primero (E)")
		return true
	}
	return false
}

func (g *Game) coopPickup(p, o *Player) {
	i := g.playerIndex(p)
	g.coop.carry[i] = o
	if !o.dead {
		o.knockT = math.Min(o.knockT, 0.8) // carried: he wakes up quickly, but stays on the shoulders
		o.immT = math.Max(o.immT, 2)
	}
	p.danceT = 0
	g.snd.Play("thud")
	g.coopSayNow(p, pickOf(g, []string{"¡Pesas!", "¡Ay, mi espalda!", "¡Arriba!", "Ya te tengo"}))
	if !o.dead {
		g.coopSayNow(o, pickOf(g, []string{"¡Y tú, huelas!", "¿Adónde me llevas?", "Qué vistas...", "¡Bájame!"}))
	}
}

func (g *Game) coopDrop(i int) {
	c := g.coop.carry[i]
	if c == nil {
		return
	}
	p := g.players()[i]
	g.coop.carry[i] = nil
	v := dirVec(p.dir)
	c.pos = g.lvl.NearestFree(Vec{p.pos.X + v.X*12, p.pos.Y + v.Y*12})
	if !c.dead {
		c.knockT = 0
		c.immT = math.Max(c.immT, 2)
		g.coopSayNow(c, pickOf(g, []string{"¡Gracias, colega!", "Me debes una", "Mi héroe", "¡Estoy bien!"}))
		g.coopHelp(i)
	}
}

func (g *Game) coopCarryUpdate(dt float64) {
	for i, c := range g.coop.carry {
		if c == nil {
			continue
		}
		p := g.players()[i]
		if p.dead || p.knockT > 0 || p.hidden {
			g.coopDrop(i)
			continue
		}
		c.pos, c.dir, c.moving = p.pos, p.dir, false
		if !c.dead {
			c.knockT = math.Max(c.knockT, 0.3) // held on the shoulders: he cannot move by himself
			c.immT = math.Max(c.immT, 1.5)
		}
	}
}

// ---------------------------------------------------------------------------
// interact: drop / pick up the partner, distract a cop, high five
// ---------------------------------------------------------------------------

// coopInteract is called at the start of interact(). It returns true when it took the press.
func (g *Game) coopInteract(p *Player) bool {
	if !g.is2P() || g.mode != ModePlay {
		return false
	}
	g.coopEnsure()
	i := g.playerIndex(p)
	o := g.otherPlayer(p)
	g.coop.interAt[i] = g.time
	if g.coop.carry[i] != nil { // he is carrying the partner: E puts him down
		g.coopDrop(i)
		return true
	}
	if o == nil || p.hidden || p.dead {
		return false
	}
	d := dist(p.pos, o.pos)
	// the partner is down: pick him up
	if (o.knockT > 0 || o.dead) && !g.coopIsCarried(o) && d <= carryReach && p.carrying == nil && !o.hidden {
		g.coopPickup(p, o)
		return true
	}
	// a cop is chasing the partner: distract him
	if g.coopTryDistract(p, o) {
		return true
	}
	// the high five: both press interact within a second, close together
	if !o.dead && o.knockT <= 0 && !o.hidden && d <= fiveReach && g.time >= g.coop.fiveNext {
		j := g.playerIndex(o)
		if g.time-g.coop.interAt[j] <= fiveWindow && g.time-g.coop.interAt[j] > 0.02 && g.coop.handUp[j] > g.time {
			g.coopHighFive(p, o)
			return true
		}
		g.coop.handUp[i] = g.time + fiveWindow // the hand stays up waiting for the partner
		g.coopSay(p, pickOf(g, []string{"¡Chócala!", "¡Choca esos cinco!", "¡Aquí arriba!"}))
	}
	return false
}

func (g *Game) coopHighFive(p, o *Player) {
	c := &g.coop
	c.fiveNext = g.time + fiveCool
	c.handUp = [2]float64{}
	mid := Vec{(p.pos.X + o.pos.X) / 2, (p.pos.Y + o.pos.Y) / 2}
	g.snd.Play("clap")
	g.shake = math.Max(g.shake, 0.08)
	g.burst(Vec{mid.X, mid.Y - 22}, 14, 70, rgb(0xfff2a0), 0.5, 2, 30)
	g.burst(Vec{mid.X, mid.Y - 22}, 8, 40, rgb(0xffffff), 0.3, 1, 0)
	g.coopFx(fxRing, mid.X, mid.Y-22, 0, 0, 0.4, color.RGBA{255, 240, 140, 255})
	for k := 0; k < 4; k++ {
		g.coopFx(fxStar, mid.X, mid.Y-22, (g.rng.Float64()-0.5)*50, -20-g.rng.Float64()*25, 0.8, color.RGBA{255, 230, 90, 255})
	}
	g.award(mid, fivePts, "choque")
	c.score[0].points += fivePts / 2
	c.score[1].points += fivePts - fivePts/2
	c.score[g.playerIndex(p)].fives++
	c.score[g.playerIndex(o)].fives++
	g.coopSayNow(p, pickOf(g, []string{"¡Chócala!", "¡Equipazo!", "¡Toma ya!"}))
	g.coopSayNow(o, pickOf(g, []string{"¡Choca esos cinco!", "¡Así se hace!", "¡Los mejores!"}))
}

// ---------------------------------------------------------------------------
// cop distraction
// ---------------------------------------------------------------------------

func (g *Game) coopTryDistract(p, o *Player) bool {
	i := g.playerIndex(p)
	if g.time < g.coop.distractNext[i] || o.dead {
		return false
	}
	var best *Cop
	bd := distractMax
	for _, c := range g.cops {
		if c.medic || c.state != cChase || g.coop.distract[c] > g.time {
			continue
		}
		if d := dist(p.pos, c.pos); d < bd {
			best, bd = c, d
		}
	}
	if best == nil {
		return false
	}
	g.coop.distractNext[i] = g.time + distractCD
	g.coop.distract[best] = g.time + distractFor
	g.coop.score[i].distracts++
	g.coopSayNow(p, pickOf(g, []string{"¿Qué hora es, agente?", "Oiga, agente...", "¿Sabe usted dónde...?", "¡Mire, un pájaro!"}))
	best.icon, best.iconT = pickOf(g, []string{"¿Eh? ¿Las cuatro?", "Mmm... ¿Qué?", "¿Un pájaro?", "Un momento..."}), distractFor
	best.dir = faceTo(best.pos, p.pos)
	g.snd.Play("boing")
	g.burst(Vec{p.pos.X, p.pos.Y - 24}, 6, 40, rgb(0xfff2a0), 0.4, 2, 20)
	return true
}

// coopCopDistracted is called by updateCop: the distracted officer stands still, looking at
// whoever distracts him. It returns true while he is distracted.
func (g *Game) coopCopDistracted(c *Cop, dt float64) bool {
	if !g.is2P() || g.coop.distract == nil {
		return false
	}
	until, ok := g.coop.distract[c]
	if !ok || g.time > until {
		return false
	}
	c.moving = false
	c.path = nil
	if o := g.nearestPlayer(c.pos); o != nil {
		// he looks at the nearest one that is NOT being chased... the distractor is the closest
		c.dir = faceTo(c.pos, o.pos)
	}
	return true
}

// ---------------------------------------------------------------------------
// traps: whose they are
// ---------------------------------------------------------------------------

// coopOwn remembers who armed a trap (or poisoned the punch): it shows in his colour.
func (g *Game) coopOwn(pr *Prop, p *Player) {
	if !g.is2P() || pr == nil || p == nil {
		return
	}
	g.coopEnsure()
	g.coop.trapOwner[pr] = g.playerIndex(p)
}

// coopTrapFired is called when a trap goes off.
func (g *Game) coopTrapFired(pr *Prop) {
	if !g.is2P() {
		return
	}
	g.coopEnsure()
	if _, ok := g.coop.trapOwner[pr]; ok {
		g.coop.lastTrap, g.coop.lastTrapT = pr, g.time
	}
}

// ---------------------------------------------------------------------------
// the dance duet and the dance bet
// ---------------------------------------------------------------------------

func (g *Game) coopDance(dt float64) {
	c := &g.coop
	a, b := g.players()[0], g.players()[1]
	for i, p := range g.players() {
		if p.danceT > 0 && !p.dead && p.knockT <= 0 {
			if c.danceSt[i] == 0 {
				c.danceSt[i] = g.time
			}
			c.score[i].danceT += dt
		} else {
			c.danceSt[i] = 0
		}
	}
	both := !a.dead && !b.dead && a.knockT <= 0 && b.knockT <= 0 && !a.hidden && !b.hidden && a.danceT > 0 && b.danceT > 0
	near := both && dist(a.pos, b.pos) <= duoReach
	if near {
		c.betOn = false
		if !c.duo {
			c.duo, c.duoAcc, c.duoJoin = true, 0, g.time+0.6
			g.snd.Play("hype")
			g.coopSayNow(a, pickOf(g, []string{"¡A bailar juntos!", "¡Un dúo!", "¡Vamos allá!"}))
			g.coopSayNow(b, pickOf(g, []string{"¡Ese es mi ritmo!", "¡Dale!", "¡Tú sígueme!"}))
		}
		// same steps and facing each other
		b.style = a.style
		a.dir, b.dir = faceTo(a.pos, b.pos), faceTo(b.pos, a.pos)
		// the duet keeps them dancing a little: the one that started first waits for the other
		if c.duoAcc < duoPtsMax {
			add := duoPtsPerS * dt
			c.duoAcc += add
			c.duoTotal += int(add + 0.5)
			g.pointsAcc(add)
			c.score[0].points += int(add / 2)
			c.score[1].points += int(add / 2)
		}
		if g.rng.Float64() < dt*5 {
			mid := Vec{(a.pos.X + b.pos.X) / 2, (a.pos.Y+b.pos.Y)/2 - 34}
			kind, col := fxHeart, color.RGBA{255, 90, 140, 255}
			if g.rng.Intn(3) == 0 {
				kind, col = fxNote, color.RGBA{255, 240, 120, 255}
			}
			g.coopFx(kind, mid.X+(g.rng.Float64()-0.5)*18, mid.Y, (g.rng.Float64()-0.5)*14, -16-g.rng.Float64()*10, 1.3, col)
		}
		if g.time >= c.duoJoin {
			c.duoJoin = g.time + 1.4
			g.coopDuoJoin(a, b)
		}
		return
	}
	if c.duo {
		c.duo = false
		c.duoEnded = g.time
		if c.duoAcc >= 20 {
			g.coopSayNow(a, pickOf(g, []string{"¡Qué dúo!", "¡Aplausos!", "¡Bis!"}))
		}
	}
	// the dance bet: both dance far from each other; the one that lasts longer wins
	if both && g.time >= c.betNext && !c.betOn && g.time-c.danceSt[0] > 0.3 && g.time-c.danceSt[1] > 0.3 {
		c.betOn = true
		g.coopSay(a, pickOf(g, []string{"¡Yo bailo más!", "¡Mírame!"}))
		g.coopSay(b, pickOf(g, []string{"¡Ni de broma!", "¡Qué va!"}))
	}
	if c.betOn {
		pl := g.players()
		switch {
		case pl[0].danceT <= 0 && pl[1].danceT <= 0 || pl[0].dead || pl[1].dead:
			c.betOn, c.betNext = false, g.time+30 // both gave up: no winner
		case pl[0].danceT <= 0 || pl[1].danceT <= 0:
			w, l := 0, 1
			if pl[0].danceT <= 0 {
				w, l = 1, 0
			}
			c.betOn, c.betNext = false, g.time+60
			g.coopSayNow(pl[w], "¡Gané la apuesta!")
			g.coopSayNow(pl[l], pickOf(g, []string{"¡Me dio un tirón!", "Tengo agujetas"}))
			g.award(pl[w].pos, 15, "apuesta")
			c.score[w].points += 15
		}
	}
}

// pointsAcc adds a fraction of a point to the party (kept as an accumulator).
func (g *Game) pointsAcc(f float64) {
	g.coop.pointFrac += f
	for g.coop.pointFrac >= 1 {
		g.coop.pointFrac--
		g.points++
	}
}

// coopDuoJoin: the guests around the dancing pair are more likely to join in.
func (g *Game) coopDuoJoin(a, b *Player) {
	mid := Vec{(a.pos.X + b.pos.X) / 2, (a.pos.Y + b.pos.Y) / 2}
	n := 0
	for _, gu := range g.guests {
		if n >= 3 || gu.out() || dist(gu.pos, mid) > 110 || !g.lvl.LOS(gu.pos, mid) {
			continue
		}
		switch gu.state {
		case gIdle, gGoto, gDrink, gLounge, gChatGo:
		default:
			continue
		}
		if g.rng.Float64() < 0.45 {
			side := 1.0
			if g.rng.Intn(2) == 0 {
				side = -1
			}
			t := Vec{mid.X + side*(26+g.rng.Float64()*14), mid.Y + (g.rng.Float64()-0.5)*16}
			gu.path = g.lvl.PathG(gu.pos, t)
			gu.state, gu.danceEnd = gDanceWith, g.time+3+g.rng.Float64()*3
			gu.icon, gu.iconT = g.line(guestCategory(gu), "dance_join"), 1.5
			n++
		}
	}
}

// ---------------------------------------------------------------------------
// small gags when they meet
// ---------------------------------------------------------------------------

func (g *Game) coopMeet(dt float64) {
	c := &g.coop
	a, b := g.players()[0], g.players()[1]
	if a.dead || b.dead || a.hidden || b.hidden {
		return
	}
	d := dist(a.pos, b.pos)
	// "colega!": they meet again after being far apart for a while
	if d > 200 {
		c.apartT += dt
	} else if d < 46 && c.apartT > 15 && g.time >= c.greetNext {
		c.apartT = 0
		c.greetNext = g.time + 60
		g.coopSayNow(a, pickOf(g, []string{"¡Colega!", "¡Tú por aquí!", "¿Dónde estabas?"}))
		g.coopSayNow(b, pickOf(g, []string{"¡Por fin!", "¡Te buscaba!", "¡Hola!"}))
	} else if d < 100 {
		c.apartT = math.Max(0, c.apartT-dt)
	}
	// twins: the same look
	if d < 70 && g.time >= c.twinNext && a.cs != nil && b.cs != nil && a.cs.Spec != nil && b.cs.Spec != nil && reflect.DeepEqual(*a.cs.Spec, *b.cs.Spec) {
		c.twinNext = g.time + 90
		g.coopSayNow(a, "¡Gemelos!")
		g.coopSayNow(b, pickOf(g, []string{"¡Mismo estilo!", "¡Qué elegancia!"}))
	}
	// "that is mine!": both reach for the same thing
	if g.time >= c.mineNext && a.item == "" && b.item == "" {
		for _, pr := range g.lvl.Props {
			if pr.gone || pr.K == nil || pr.K.Item == "" {
				continue
			}
			if distRect(a.pos, pr.Foot) < 22 && distRect(b.pos, pr.Foot) < 22 {
				c.mineNext = g.time + 40
				g.coopSayNow(a, "¡Eso es mío!")
				g.coopSayNow(b, pickOf(g, []string{"¡Lo vi yo primero!", "¡Es mío!"}))
				break
			}
		}
	}
}

// ---------------------------------------------------------------------------
// drawing in the world
// ---------------------------------------------------------------------------

func fillRectF(dst *ebiten.Image, x, y, w, h float64, c color.RGBA) { fillRect(dst, x, y, w, h, c) }

var heartPix = []string{
	".X.X.",
	"XXXXX",
	"XXXXX",
	".XXX.",
	"..X..",
}

var notePix = []string{
	"..XX",
	"..X.",
	"..X.",
	"XXX.",
	"XX..",
}

func (g *Game) drawPixels(dst *ebiten.Image, pat []string, x, y float64, c color.RGBA) {
	for j, row := range pat {
		for i, ch := range row {
			if ch == 'X' {
				fillRect(dst, math.Round(x)+float64(i), math.Round(y)+float64(j), 1, 1, c)
			}
		}
	}
}

// coopDrawWorld paints the effects of this file over the party.
func (g *Game) coopDrawWorld(dst *ebiten.Image) {
	if !g.is2P() || g.mode != ModePlay {
		return
	}
	c := &g.coop
	// who set the traps (and the poisoned punch): a small diamond in his colour
	for pr, owner := range c.trapOwner {
		if pr == nil || pr.K == nil || (pr.armT <= 0 && !pr.armed && !pr.armWait) || pr.used || pr.gone {
			continue
		}
		if pr.K.Trap == "" && !pr.K.Poison {
			continue
		}
		rc := ringColor(g.players()[owner])
		col := color.RGBA{rc[0], rc[1], rc[2], 255}
		x := g.wx(pr.X)
		y := g.wy(pr.Y) - float64(pr.K.H) - 6 + math.Round(math.Sin(g.time*5))
		fillRect(dst, x-1, y-3, 3, 1, col)
		fillRect(dst, x-2, y-2, 5, 1, col)
		fillRect(dst, x-3, y-1, 7, 2, col)
		fillRect(dst, x-2, y+1, 5, 1, col)
		fillRect(dst, x-1, y+2, 3, 1, col)
		fillRect(dst, x-1, y-1, 3, 2, color.RGBA{255, 255, 255, 220})
	}
	// the partner on the shoulders
	for i, cp := range c.carry {
		if cp == nil {
			continue
		}
		p := g.players()[i]
		bi := cp.cs.lying(false)
		w, h := bi.Bounds().Dx(), bi.Bounds().Dy()
		g.drawImg(dst, bi, g.wx(p.pos.X), g.wy(p.pos.Y)-38*g.charSc, float64(w)/2, float64(h))
	}
	// the raised hand waiting for a high five
	for i, until := range c.handUp {
		if until > g.time {
			p := g.players()[i]
			rc := ringColor(p)
			x, y := g.wx(p.pos.X)+6, g.wy(p.pos.Y)-34*g.charSc-2-math.Round(math.Sin(g.time*10))
			fillRect(dst, x, y, 4, 5, color.RGBA{255, 220, 180, 255})
			fillRect(dst, x+1, y-2, 1, 2, color.RGBA{255, 220, 180, 255})
			fillRect(dst, x+3, y-2, 1, 2, color.RGBA{255, 220, 180, 255})
			fillRect(dst, x, y+5, 4, 1, color.RGBA{rc[0], rc[1], rc[2], 255})
		}
	}
	for _, f := range c.fx {
		a := f.life / f.max
		col := color.RGBA{f.col.R, f.col.G, f.col.B, uint8(255 * math.Min(1, a*1.6))}
		x, y := g.wx(f.x), g.wy(f.y)
		switch f.kind {
		case fxHeart:
			g.drawPixels(dst, heartPix, x, y, col)
		case fxNote:
			g.drawPixels(dst, notePix, x, y, col)
		case fxStar:
			fillRect(dst, math.Round(x), math.Round(y), 2, 2, col)
			fillRect(dst, math.Round(x)-1, math.Round(y)+0, 4, 1, col)
		case fxRing:
			r := (1 - a) * 18
			for t := 0.0; t < 2*math.Pi; t += 0.25 {
				fillRect(dst, math.Round(x+math.Cos(t)*r), math.Round(y+math.Sin(t)*r*0.6), 1, 1, col)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// the scoreboard
// ---------------------------------------------------------------------------

type coopTitle struct {
	name  string
	score func(me, other coopStats) float64
}

func ratio(a, b float64) float64 {
	d := a + b
	if d <= 0 {
		return 0
	}
	return (a - b) / d
}

var coopTitles = []coopTitle{
	{"Anfitrión del año", func(m, o coopStats) float64 { return ratio(float64(m.points), float64(o.points)) * 1.0 }},
	{"Carnicero con delicadeza", func(m, o coopStats) float64 { return ratio(float64(m.kills), float64(o.kills)) * 1.1 }},
	{"El que lo vio todo", func(m, o coopStats) float64 { return ratio(float64(o.kills), float64(m.kills)) * 0.9 }},
	{"Estorbo oficial", func(m, o coopStats) float64 { return ratio(float64(m.hits), float64(o.hits)) * 1.2 }},
	{"Rey de la pista", func(m, o coopStats) float64 { return ratio(m.danceT, o.danceT) * 1.0 }},
	{"Ángel de la guarda", func(m, o coopStats) float64 {
		return ratio(float64(m.helps+m.distracts), float64(o.helps+o.distracts)) * 1.3
	}},
	{"Saco de boxeo", func(m, o coopStats) float64 { return ratio(m.floorT, o.floorT) * 0.95 }},
	{"Chocador compulsivo", func(m, o coopStats) float64 { return ratio(float64(m.fives), float64(o.fives)) * 0.5 }},
}

// coopTitleFor gives each player a different title: the one he deserves most compared with the partner.
func coopTitleFor(a, b coopStats) [2]string {
	type cand struct {
		p     int
		t     int
		score float64
	}
	var cs []cand
	for ti, t := range coopTitles {
		cs = append(cs, cand{0, ti, t.score(a, b)}, cand{1, ti, t.score(b, a)})
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].score > cs[j].score })
	var out [2]string
	used := map[int]bool{}
	for _, c := range cs {
		if c.score <= 0.05 || out[c.p] != "" || used[c.t] {
			continue
		}
		out[c.p], used[c.t] = coopTitles[c.t].name, true
	}
	fallback := []string{"Compañero ejemplar", "Invitado de honor"}
	for p := 0; p < 2; p++ {
		if out[p] == "" {
			out[p] = fallback[p]
		}
	}
	return out
}

// coopDrawScoreboard paints the duel of the two players under the end-of-party panel.
func (g *Game) coopDrawScoreboard(dst *ebiten.Image) {
	if !g.is2P() {
		return
	}
	g.coopEnsure()
	c := &g.coop
	titles := coopTitleFor(c.score[0], c.score[1])
	fillRect(dst, 0, 362, screenW, 168, rgba(0x000000, 238))
	// crown: more points wins the duel (a tie: none)
	win := -1
	switch {
	case c.score[0].points > c.score[1].points:
		win = 0
	case c.score[1].points > c.score[0].points:
		win = 1
	}
	for i, pl := range g.players() {
		x0 := 40.0 + float64(i)*460
		rc := ringColor(pl)
		col := color.RGBA{rc[0], rc[1], rc[2], 255}
		fillRect(dst, x0, 368, 420, 2, col)
		g.drawText(dst, fmt.Sprintf("JUGADOR %d", i+1), x0, 374, col, 1)
		g.drawText(dst, titles[i], x0+80, 374, rgb(0xf0e060), 1)
		if im, _ := pl.cs.img(AnimIdle, DirDown, 0); im != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(x0+6, 400)
			dst.DrawImage(im, op)
		}
		if win == i {
			cx, cy := x0+24, 396.0
			fillRect(dst, cx, cy, 14, 3, rgb(0xf0d030))
			fillRect(dst, cx, cy-4, 2, 4, rgb(0xf0d030))
			fillRect(dst, cx+6, cy-6, 2, 6, rgb(0xf0d030))
			fillRect(dst, cx+12, cy-4, 2, 4, rgb(0xf0d030))
		}
		st := c.score[i]
		rows := []struct {
			k string
			v string
		}{
			{"Bajas", fmt.Sprint(st.kills)},
			{"Puntos", fmt.Sprint(st.points)},
			{"Golpes al compañero", fmt.Sprint(st.hits)},
			{"Tiempo en el suelo", fmt.Sprintf("%d s", int(st.floorT+0.5))},
			{"Ayudas", fmt.Sprint(st.helps + st.distracts)},
		}
		for r, row := range rows {
			y := 396.0 + float64(r)*18
			g.drawText(dst, row.k, x0+110, y, rgb(0xd0c8e0), 1)
			g.drawTextC(dst, row.v, x0+400, y, rgb(0xffffff), 1)
		}
	}
	g.drawTextC(dst, "VS", screenW/2, 440, rgb(0xff6080), 2)
}

// coopDemo is the debug flag -coopdemo: it sets up a scene of this file for a screenshot.
var coopDemo int

func (g *Game) coopDemoSetup() {
	if coopDemo == 0 || g.coop.demoDone || g.time < 0.5 {
		return
	}
	g.coop.demoDone = true
	a, b := g.players()[0], g.players()[1]
	switch coopDemo {
	case 1:
		g.coop.score[0] = coopStats{points: 480, kills: 12, hits: 1, floorT: 3, danceT: 12, fives: 1}
		g.coop.score[1] = coopStats{points: 360, kills: 7, hits: 2, floorT: 9, helps: 2, fives: 1, danceT: 3}
		g.points, g.endTime = 840, 245
		g.mode = ModeLevelDone
	case 2:
		b.pos = Vec{a.pos.X + 26, a.pos.Y}
		a.danceT, b.danceT = 999, 999
		pk := propKinds["extinguisher"] // an explosive prop of the game, so it draws like a real one
		tp := &Prop{Name: "extinguisher", K: pk, X: a.pos.X - 40, Y: a.pos.Y, Foot: Rect{a.pos.X - 44, a.pos.Y - 4, 8, 6}, armT: 999}
		g.lvl.Props = append(g.lvl.Props, tp)
		g.coop.trapOwner[tp] = 1
	case 3:
		b.pos = Vec{a.pos.X + 10, a.pos.Y}
		b.knockT = 999
		g.coop.carry[0] = b
		g.coop.handUp[1] = g.time + 999
	}
}

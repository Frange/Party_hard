package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The police with two players: personal cuffs, custody and the prison break.
//
// The criminality level of the party (g.wanted, the stars) is ONE for the whole party, but the
// cuffs are personal: a player is a suspect (p.jail.sus) when a witness or an officer saw HIM
// kill (or saw him carrying a body). Rules asked by the user:
//
//   - the police are called: the cuffs go to the one they saw; if they see the other one too (kills
//     in front of somebody, a body on his shoulder), both are suspects.
//   - a special killer, or an armed guest, seen killing clears BOTH of them; whoever kills again
//     and is seen is the only suspect again (suspect.go).
//   - an officer who catches a suspect does not kill him: with the other player free he is held
//     at the police vehicle (jailPlayer). The partner can free him: hold E next to him for 5 s
//     without any officer seeing it (freePlayer): he comes back as he was.
//   - after 1 minute in custody, with the partner still alive, he ESCAPES (escapeJail) in a
//     prisoner suit, dragging a ball and chain: 70 % of his speed, and it is his LAST LIFE: if he is
//     caught again or dies there is no second chance (p.jail.lastLife).
//   - if the free player dies while the other is held, or both are caught: the party is lost.
//
// With one player none of this runs: a catch ends the party as always.
// ---------------------------------------------------------------------------

const (
	jailEscapeT  = 60.0 // seconds in custody before the prison break
	jailRescueT  = 5.0  // seconds of E next to him to free him
	jailReach    = 30.0 // how close the partner must be
	convictSpeed = 0.7  // speed of the one who escaped, with the ball and chain
	chainLen     = 13.0 // pixels between his ankle and the ball
)

// jailHold: somebody is in custody, so the officers stay on the scene (mayLeave).
var jailHold bool

// jailState is what a player carries for the police rules (two players only).
type jailState struct {
	sus      bool    // the police are after him: his cuffs mark shows
	killT    float64 // when he last killed (the suspect if nobody saw it)
	held     bool    // in custody, at the police vehicle
	t        float64 // seconds in custody
	pos      Vec     // where he is held
	free     float64 // progress of the rescue by his partner, 0..jailRescueT
	lastLife bool    // he escaped: no more chances
	ball     Vec     // where the ball of the chain lies
	chainT   float64 // time to the next clink
	holeT    float64 // seconds the broken bars stay where he escaped
	holePos  Vec
}

// speedMul: the one who escaped drags a ball and chain.
func (p *Player) speedMul() float64 {
	if p.jail.lastLife {
		return convictSpeed
	}
	return 1
}

// ---- who is a suspect ---------------------------------------------------------------------

// isSuspectP: the police are after this player (his cuffs show).
func (g *Game) isSuspectP(p *Player) bool {
	if p == nil || p.dead || p.jail.held || g.wanted <= 0 || g.playerCleared() {
		return false
	}
	if !g.is2P() {
		return true
	}
	return p.jail.sus
}

// markSuspect: the police are after p (two players).
func (g *Game) markSuspect(p *Player) {
	if g.is2P() && p != nil && !p.dead && !p.jail.held && !g.playerCleared() {
		p.jail.sus = true
	}
}

// clearSuspects takes the cuffs off both players.
func (g *Game) clearSuspects() {
	for _, pl := range g.players() {
		pl.jail.sus = false
	}
}

// anySuspect: some living free player is a suspect.
func (g *Game) anySuspect() bool {
	for _, pl := range g.players() {
		if !pl.dead && !pl.jail.held && pl.jail.sus {
			return true
		}
	}
	return false
}

// ensureSuspect: the police were called and want somebody but nobody saw the kill: it is the
// last one who killed.
func (g *Game) ensureSuspect() {
	if !g.is2P() || g.anySuspect() {
		return
	}
	var best *Player
	for _, pl := range g.players() {
		if pl.dead || pl.jail.held {
			continue
		}
		if best == nil || pl.jail.killT > best.jail.killT {
			best = pl
		}
	}
	if best != nil {
		best.jail.sus = true
	}
}

// shakeOffPolice: p loses the trail (a wardrobe, a disguise). The other one, if he is a suspect,
// keeps the police on the scene.
func (g *Game) shakeOffPolice(p *Player) {
	if !g.is2P() {
		g.wanted, g.seenNow = 0, false
		return
	}
	p.jail.sus = false
	if !g.anySuspect() {
		g.wanted, g.seenNow = 0, false
	}
}

// copInterest: does this officer care about p? (one player: as always)
func (g *Game) copInterest(p *Player) bool {
	if p.carrying != nil {
		return !p.jail.held
	}
	if !g.is2P() {
		return g.wanted > 0
	}
	return g.isSuspectP(p)
}

// copTarget is the player an officer thinks about: with one player always him; with two, the nearest
// suspect (preferring the ones in plain sight), or the nearest free one when nobody is wanted.
func (g *Game) copTarget(c *Cop) *Player {
	if !g.is2P() {
		return g.p
	}
	rng, dot := copRange, copDot
	if c.state == cChase {
		rng, dot = 150, -0.3
	}
	var best, any *Player
	bd, ad := 1e18, 1e18
	for _, pl := range g.players() {
		if pl.dead || pl.jail.held {
			continue
		}
		d := dist(c.pos, pl.pos)
		if d < ad {
			any, ad = pl, d
		}
		if !g.copInterest(pl) {
			continue
		}
		if !pl.hidden && g.sees(c.pos, c.dir, pl.pos, rng, dot) {
			d -= 1000
		}
		if d < bd {
			best, bd = pl, d
		}
	}
	switch {
	case best != nil:
		return best
	case any != nil:
		return any
	}
	return g.p
}

// ---- the catch -----------------------------------------------------------------------------

// copCatch: an officer reached the player he was chasing.
func (g *Game) copCatch(c *Cop, p *Player) {
	if !g.is2P() {
		g.die("¡La policía te ha atrapado!")
		return
	}
	o := g.otherPlayer(p)
	if p.jail.lastLife || o == nil || o.dead || o.jail.held {
		// no second chance, or nobody left outside to free him: out
		g.withPlayer(p, func() { g.die("¡La policía te ha atrapado!") })
		return
	}
	g.jailPlayer(c, p)
}

// jailPlayer puts p in custody at the vehicle of the officer who caught him (c may be nil).
func (g *Game) jailPlayer(c *Cop, p *Player) {
	at := p.pos
	if c != nil {
		at = g.vehDoor(c)
	}
	p.jail.held, p.jail.t, p.jail.free = true, 0, 0
	p.jail.sus = false
	p.jail.pos = g.lvl.NearestFree(at)
	p.pos = p.jail.pos
	p.dir = DirDown
	p.moving, p.danceT, p.knockT, p.hidden = false, 0, 0, false
	if p.carrying != nil {
		p.carrying.carried = false
		p.carrying.pos = p.pos
		p.carrying = nil
	}
	if c != nil {
		g.releaseCop(c)
		g.copNext(c)
	}
	p.msg, p.msgT = jailCries[g.rng.Intn(len(jailCries))], 2.2
	g.snd.Play("thud")
	g.snd.Play("alert")
	g.shake = math.Max(g.shake, 0.15)
	g.say(g.playerName(p) + " detenido: ¡que " + g.playerName(g.otherPlayer(p)) + " lo saque (E)!") // template "{a} detenido: ¡que {b} lo saque (E)!"
	g.sayNews("jail_held", map[string]string{"victim": g.playerName(p)})
	jailHold = true
}

var jailCries = []string{"¡SOY INOCENTE!", "¡FUE EL MAYORDOMO!", "¡QUIERO MI LLAMADA!", "¡ESTO ES UN MALENTENDIDO!", "¡ME HAN PLANTADO UN CUCHILLO!"}

// ---- every frame -------------------------------------------------------------------------------

// updateJail runs the custody timers, the rescue and the prison break.
func (g *Game) updateJail(dt float64) {
	if !g.is2P() {
		jailHold = false
		return
	}
	g.debugJailStep()
	hold := false
	for _, p := range g.players() {
		if p.jail.holeT > 0 {
			p.jail.holeT -= dt
		}
		if p.dead {
			continue
		}
		if p.jail.held {
			g.jailTick(p, dt)
			hold = hold || p.jail.held // he may have escaped just now
		}
		if p.jail.lastLife && !p.jail.held {
			g.updateBall(p, dt)
		}
	}
	jailHold = hold
}

func (g *Game) jailTick(p *Player, dt float64) {
	p.jail.t += dt
	p.moving = false
	o := g.otherPlayer(p)
	pressing := false
	if o != nil && !o.dead && !o.jail.held && !o.hidden {
		g.asPlayer(o.idx, func() { pressing = actInteract.held() })
	}
	g.jailRescue(p, o, pressing, dt)
	if p.jail.held && p.jail.t >= jailEscapeT {
		g.escapeJail(p)
	}
}

// jailRescue: the partner holds E next to the prisoner, unseen by the officers, and frees him.
func (g *Game) jailRescue(p, o *Player, pressing bool, dt float64) {
	if o == nil || o.dead || o.jail.held {
		p.jail.free = math.Max(0, p.jail.free-dt)
		return
	}
	near := dist(o.pos, p.jail.pos) < jailReach
	seen := g.copSeesAt(o.pos)
	switch {
	case pressing && near && !seen:
		p.jail.free += dt
		if o.msgT <= 0 {
			o.msg, o.msgT = "Shhh...", 1.0
		}
		if p.jail.free >= jailRescueT {
			g.freePlayer(p)
		}
	case pressing && near && seen:
		p.jail.free = math.Max(0, p.jail.free-dt)
		if o.msgT <= 0 {
			o.msg, o.msgT = "¡NOS VEN!", 1.0
		}
	case p.jail.free > 0:
		p.jail.free = math.Max(0, p.jail.free-dt*0.5)
	}
}

// freePlayer: the partner got him out, as he was (no prisoner suit, no ball, no last-life mark).
func (g *Game) freePlayer(p *Player) {
	p.jail.held, p.jail.free, p.jail.t = false, 0, 0
	p.jail.sus = false
	p.pos = g.lvl.NearestFree(Vec{p.jail.pos.X + 12, p.jail.pos.Y + 6})
	p.knockT, p.immT = 0, 3
	p.msg, p.msgT = "¡LIBRE!", 1.8
	g.burst(Vec{p.pos.X, p.pos.Y - 18}, 14, 70, rgb(0xf8e040), 0.6, 2, 40)
	g.snd.Play("win")
	g.say("¡" + g.playerName(p) + " está libre! Sin traje y sin bola")
	g.sayNews("jail_rescue", map[string]string{"victim": g.playerName(p)})
	jailHold = false
}

// escapeJail: a minute has passed and his partner is still alive: prison break, in a striped
// suit and with a ball and chain, slower, on his last life.
func (g *Game) escapeJail(p *Player) {
	p.jail.held, p.jail.free, p.jail.t = false, 0, 0
	p.jail.lastLife = true
	p.cs = g.prisonerCS(p)
	p.pos = g.lvl.NearestFree(Vec{p.jail.pos.X - 12, p.jail.pos.Y + 8})
	p.jail.ball = p.pos
	p.jail.holeT, p.jail.holePos = 9, p.jail.pos
	p.knockT, p.immT = 0, 2
	p.msg, p.msgT = jailBreakCries[g.rng.Intn(len(jailBreakCries))], 2.6
	// the police go after him again if they are still around
	if g.lawActive() {
		g.wanted = math.Max(g.wanted, 30)
		p.jail.sus = true
	}
	g.burst(Vec{p.jail.pos.X, p.jail.pos.Y - 12}, 26, 100, rgb(0xc8c0b0), 0.8, 2, 120)
	g.burst(Vec{p.pos.X, p.pos.Y - 18}, 16, 80, rgb(0xf0e060), 0.5, 2, 40)
	g.snd.Play("siren")
	g.snd.Play("thud")
	g.shake = math.Max(g.shake, 0.3)
	g.say("¡FUGA! " + g.playerName(p) + " se escapa con bola y todo (última vida)")
	g.sayNews("jail_break", map[string]string{"victim": g.playerName(p)})
	jailHold = false
}

var jailBreakCries = []string{"¡LIBRE POR FIN!", "¡ADIÓS, FUNCIONARIOS!", "¡CORRO COMO PUEDO!", "¡ESTA BOLA PESA!", "¡YO NO ESTUVE AQUÍ!"}

// updateBall: the ball follows him at a chain's length and clinks as it drags.
func (g *Game) updateBall(p *Player, dt float64) {
	d := dist(p.jail.ball, p.pos)
	if d > chainLen {
		k := (d - chainLen) / d
		p.jail.ball.X += (p.pos.X - p.jail.ball.X) * k
		p.jail.ball.Y += (p.pos.Y - p.jail.ball.Y) * k
		p.jail.chainT -= dt
		if p.jail.chainT <= 0 {
			p.jail.chainT = 0.32
			if _, ok := g.snd.sfx["chain"]; !ok && g.snd.sfx != nil {
				g.snd.sfx["chain"] = mix(
					tone(2300, 0.03, 0.22, 30, 0), shift(tone(1700, 0.03, 0.2, 30, 0), 0.05),
					shift(tone(2900, 0.025, 0.16, 30, 0), 0.09), noise(0.03, 0.2, 40))
			}
			g.snd.Play("chain")
		}
	}
}

// prisonerCS: the player's own build, in black and white stripes.
func (g *Game) prisonerCS(p *Player) *CharSet {
	return g.shopCS("convict:"+g.equipSigOf(p.idx), func() *CharSpec {
		sp := g.playerSpecFor(p.idx, "", "")
		sp.Top, sp.Shirt, sp.Shirt2, sp.Pattern = TopLong, hx(0xf4f4f6), hx(0x16161c), 2
		sp.Pants = hx(0xe6e6ea)
		sp.Hat, sp.Cape.A, sp.Susp.A, sp.Mask.A = HatNone, 0, 0, 0
		sp.noOverride = true
		return sp
	})
}

// ---- drawing -----------------------------------------------------------------------------------

// drawSuspectCuffs: the cuffs mark over every suspect.
func (g *Game) drawSuspectCuffs(dst *ebiten.Image) {
	if g.mode != ModePlay || g.otherCuffed() {
		return
	}
	for _, p := range g.players() {
		if g.isSuspectP(p) && !p.hidden && !p.dead {
			g.drawCuffs(dst, p.pos)
		}
	}
}

// jailItems adds the prisoners (sitting by the police vehicle) and the broken bars to the scene.
func (g *Game) jailItems(dst *ebiten.Image, items []drawItem) []drawItem {
	if !g.is2P() || g.mode != ModePlay {
		return items
	}
	for _, p := range g.players() {
		p := p
		if p.jail.held {
			items = append(items, drawItem{p.jail.pos.Y + 0.1, func() { g.drawPrisoner(dst, p) }})
		}
		if p.jail.holeT > 0 {
			items = append(items, drawItem{p.jail.holePos.Y - 6, func() { g.drawBrokenBars(dst, p) }})
		}
	}
	return items
}

// drawPrisoner: cuffed by the police vehicle, with the time left and the rescue bar over his head.
func (g *Game) drawPrisoner(dst *ebiten.Image, p *Player) {
	g.withPlayer(p, func() {
		c := ringColor(p)
		col := color.RGBA{c[0], c[1], c[2], 255}
		x, y := math.Round(g.wx(p.jail.pos.X)), math.Round(g.wy(p.jail.pos.Y))
		// the cell: a ring of the police colours under his feet and two bars at his sides
		for a := 0.0; a < 2*math.Pi; a += 0.25 {
			fillRect(dst, math.Round(x+math.Cos(a)*12), math.Round(y-1+math.Sin(a)*5), 1, 1, color.RGBA{230, 60, 60, 230})
		}
		g.drawActor(dst, p.cs, AnimIdle, g.time*2, p.jail.pos, DirDown, 0, false)
		g.drawCuffs(dst, p.jail.pos)
		for _, bx := range []float64{-9, 9} {
			fillRect(dst, x+bx, y-26, 1, 24, color.RGBA{150, 156, 170, 230})
		}
		// the minute: a bar that fills as the prison break approaches
		w := 26.0
		fillRect(dst, x-w/2, y-44, w, 3, rgb(0x2a2a3a))
		fillRect(dst, x-w/2, y-44, w*math.Min(1, p.jail.t/jailEscapeT), 3, col)
		if p.jail.free > 0 {
			fillRect(dst, x-w/2, y-48, w, 3, rgb(0x2a2a3a))
			fillRect(dst, x-w/2, y-48, w*math.Min(1, p.jail.free/jailRescueT), 3, rgb(0x60e080))
		}
		g.drawPlayerTag(dst, p)
	})
}

// drawBrokenBars: the bent bars and the hole where he got out.
func (g *Game) drawBrokenBars(dst *ebiten.Image, p *Player) {
	x, y := math.Round(g.wx(p.jail.holePos.X)), math.Round(g.wy(p.jail.holePos.Y))
	a := uint8(math.Min(1, p.jail.holeT/2) * 255)
	for dx := -9.0; dx <= 9; dx++ {
		for dy := -3.0; dy <= 3; dy++ {
			if dx*dx/81+dy*dy/9 <= 1 {
				fillRect(dst, x+dx, y+dy, 1, 1, color.RGBA{20, 16, 14, a})
			}
		}
	}
	for i, bx := range []float64{-8, -3, 3, 8} {
		h := 14.0 - float64(i%2)*4
		fillRect(dst, x+bx, y-h, 1, h, color.RGBA{150, 156, 170, a})
		fillRect(dst, x+bx+float64(1-2*(i%2)), y-h, 2, 1, color.RGBA{150, 156, 170, a}) // bent tip
	}
}

// drawBallChain: the chain from his ankle to the ball, and the ball.
func (g *Game) drawBallChain(dst *ebiten.Image, p *Player) {
	if !p.jail.lastLife || p.dead || p.jail.held {
		return
	}
	px, py := g.wx(p.pos.X), g.wy(p.pos.Y)-2
	bx, by := g.wx(p.jail.ball.X), g.wy(p.jail.ball.Y)-2
	for i := 1; i < 6; i++ {
		t := float64(i) / 6
		sag := math.Sin(t*math.Pi) * 2
		fillRect(dst, math.Round(px+(bx-px)*t), math.Round(py+(by-py)*t+sag), 1, 1, color.RGBA{170, 176, 190, 255})
	}
	bx, by = math.Round(bx), math.Round(by)
	for i, w := range []float64{3, 5, 5, 5, 3} {
		fillRect(dst, bx-w/2, by-2+float64(i), w, 1, color.RGBA{36, 36, 44, 255})
	}
	fillRect(dst, bx-1, by-2, 1, 1, color.RGBA{120, 124, 140, 255})
	fillRect(dst, bx-1, by-1, 1, 1, color.RGBA{86, 90, 104, 255})
}

// drawJailHud: who is in custody and how long is left, and who is on his last life.
func (g *Game) drawJailHud(dst *ebiten.Image) {
	if !g.is2P() || g.mode != ModePlay {
		return
	}
	y := 124.0
	for _, p := range g.players() {
		c := ringColor(p)
		col := color.RGBA{c[0], c[1], c[2], 255}
		switch {
		case p.dead:
		case p.jail.held:
			left := int(math.Ceil(jailEscapeT - p.jail.t))
			g.drawText(dst, fmt.Sprintf("%s DETENIDO %d:%02d", g.playerName(p), left/60, left%60), 10, y, col, 1)
			y += 12
			g.drawText(dst, "El otro: E junto a él, 5 s sin que le vean", 10, y, rgb(0xb8b0d0), 1)
			y += 12
		case p.jail.lastLife:
			g.drawText(dst, g.playerName(p)+" ÚLTIMA VIDA", 10, y, col, 1)
			y += 12
		}
	}
}

// ---- debug ---------------------------------------------------------------------------------------

// dbgTwoCuff / dbgTwoJail are set by -twocuff and -twojail (debug).
var (
	dbgTwoCuff bool
	dbgTwoJail int // 1: player 1 is caught at 3 s; 2: the same with 55 s of the minute already spent
)

func (g *Game) debugJailStep() {
	if g.time < 3 {
		return
	}
	if dbgTwoCuff {
		dbgTwoCuff = false
		g.raisePolice(g.first().pos, true)
		g.wanted = 70
		for _, p := range g.players() {
			p.jail.sus = true
		}
	}
	if dbgTwoJail > 0 {
		n := dbgTwoJail
		dbgTwoJail = 0
		g.jailPlayer(nil, g.first())
		if n == 2 {
			g.first().jail.t = jailEscapeT - 5
		}
	}
}

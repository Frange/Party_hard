package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Snake: at most one per party. It slithers around outdoors, hisses at
// whoever gets close and then bites. A bite kills a guest after a few
// seconds (they get to scream first) and ends the game for the player. It
// can be stabbed.
// ---------------------------------------------------------------------------

const animSnake = 16

func init() {
	if len(animalKinds) != animSnake {
		panic("animals: the snake index is out of step with animalKinds")
	}
	animalKinds = append(animalKinds, "snake")
}

var snakeCoats = []struct{ body, dark, belly, band c4 }{
	{hx(0x4a9a3a), hx(0x2a5a24), hx(0xd0e090), hx(0x78c04a)},
	{hx(0x9a7a3a), hx(0x4a3418), hx(0xe8d8a0), hx(0x6a4a20)},
	{hx(0xd03a2a), hx(0x14120e), hx(0xf0e070), hx(0xf0d020)}, // coral snake
}

// genSnakeFrame: a side view that slithers (walk frames), rears up to hiss
// (fWarn) or lunges (fKick).
func genSnakeFrame(frame, coat int) *pixBuf {
	b := newPix(40, 22)
	b.ox, b.oy = 1, 1
	cc := snakeCoats[coat%len(snakeCoats)]
	ph := 0.0
	if frame >= fWalk0 && frame <= fWalk3 {
		ph = float64(frame-fWalk0) * math.Pi / 2
	}
	const n = 22
	type pt struct{ x, y, r float64 }
	var body []pt
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n-1) // 0 tail .. 1 head
		x := 3 + t*28
		y := 16 + 2.6*math.Sin(t*7-ph)
		r := 1.2 + 1.5*math.Sin(math.Min(1, t*1.4)*math.Pi*0.62)
		if frame == fWarn { // rears up, an S with the head high
			y = 17 - 9*math.Pow(t, 2.4) + 1.5*math.Sin(t*7)
			x = 3 + t*22 + 3*math.Pow(t, 3)
		}
		if frame == fKick { // stretched out forward and low
			y = 16 - 4*t + 1.2*math.Sin(t*5)
			x = 2 + t*34
		}
		body = append(body, pt{x, y, r})
	}
	for i, p := range body {
		col := cc.body
		switch {
		case coat == 2 && (i/3)%2 == 0:
			col = cc.dark
		case coat == 2 && i%3 == 1:
			col = cc.band
		case coat == 1 && i%4 == 2:
			col = cc.dark
		case coat == 0 && i%5 == 3:
			col = cc.band
		}
		b.ellipse(p.x, p.y, p.r, p.r, col)
		b.ellipse(p.x, p.y+p.r*0.45, p.r*0.7, p.r*0.4, cc.belly)
	}
	h := body[n-1]
	b.ellipse(h.x+2, h.y, 3.4, 2.5, cc.body)
	b.ellipse(h.x+2.5, h.y+0.6, 2.4, 1.1, cc.belly)
	b.set(int(h.x)+2, int(h.y)-1, hx(0x101010))
	b.set(int(h.x)+3, int(h.y)-1, hx(0xf0e020))
	if frame == fWarn || frame == fKick {
		b.line(int(h.x)+5, int(h.y), int(h.x)+8, int(h.y)-1, hx(0xe02030)) // forked tongue
		b.line(int(h.x)+8, int(h.y)-1, int(h.x)+9, int(h.y)-2, hx(0xe02030))
		b.line(int(h.x)+8, int(h.y)-1, int(h.x)+9, int(h.y), hx(0xe02030))
		if frame == fKick {
			b.set(int(h.x)+5, int(h.y)+1, hx(0xffffff)) // fang
			b.set(int(h.x)+5, int(h.y)+2, hx(0xffffff))
		}
	}
	return b
}

func buildSnakeSprites(out AnimalSprites) {
	for coat := 0; coat < 3; coat++ {
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fWarn, fKick} {
			out[animalKey(animSnake, coat, f)] = genSnakeFrame(f, coat).outlinedSel(0.4).img()
		}
	}
}

func snakeFrame(a *Animal) int {
	switch a.state {
	case aWarn:
		return fWarn
	case aKick:
		return fKick
	}
	if a.moving {
		return fWalk0 + int(a.anim*7)%4
	}
	return fStand
}

// startSafeDist: a snake is never born closer than this to the start (its bite reach, 34 px, plus 10 s of its
// 20 px/s walk): the player standing still on Start cannot be bitten in his first ~10 s (QA 2026-11-11, BARCO
// PIRATA: bitten at 4 s). Its wander area (90 px around where it was born) keeps it away afterwards too.
const startSafeDist = 34 + 20*10

// nearStart: too close to where a player starts.
func (g *Game) nearStart(p Vec) bool {
	if dist(p, g.lvl.Def.Start) < startSafeDist {
		return true
	}
	for _, pl := range g.players() {
		if pl != nil && dist(p, pl.pos) < startSafeDist {
			return true
		}
	}
	return false
}

// scatterSnake: one snake, now and then, somewhere outdoors (never next to the start).
func (g *Game) scatterSnake() {
	if !g.dbgSnake && !snakeDecide(snakeDice(), &snakeMisses) { // snakechance.go: own dice
		return
	}
	pos, ok := g.outdoorSpot()
	for i := 0; ok && i < 30 && g.nearStart(pos); i++ {
		pos, ok = g.outdoorSpot()
	}
	if ok && g.nearStart(pos) { // a party with no outdoor spot far enough: no snake this time
		ok = false
	}
	if g.dbgSnake {
		pos, ok = Vec{g.lvl.Def.Start.X + 40, g.lvl.Def.Start.Y - 30}, true
	}
	if !ok {
		return
	}
	g.animals = append(g.animals, &Animal{kind: animSnake, pos: g.lvl.NearestFree(pos), face: 1, area: Rect{pos.X - 90, pos.Y - 50, 180, 100}, seed: g.rng.Float64() * 10, timer: g.rng.Float64() * 2, coat: g.rng.Intn(3)})
}

// snakeTarget: whoever is close enough to bother.
func (g *Game) snakeTarget(a *Animal, reach float64) any {
	if p := g.p; !p.dead && !p.hidden && dist(p.pos, a.pos) < reach {
		return p
	}
	for _, gu := range g.guests {
		if !gu.out() && gu.state != gSleep && dist(gu.pos, a.pos) < reach {
			return gu
		}
	}
	return nil
}

func (g *Game) updateSnake(a *Animal, dt float64) {
	switch a.state {
	case aIdle, aGraze, aWalk:
		if a.state == aWalk {
			a.timer -= dt
			if g.animalWalk(a, 20, dt) || a.timer <= 0 {
				a.state, a.timer = aIdle, 1.5+g.rng.Float64()*2.5
			}
		} else {
			a.timer -= dt
			if a.timer <= 0 {
				a.path = g.lvl.Path(a.pos, g.wanderTarget(a))
				a.state, a.timer = aWalk, 14
			}
		}
		if v := g.snakeTarget(a, 34); v != nil {
			a.warnT += dt
			if a.warnT > 0.4 {
				a.warnT = 0
				a.state, a.timer, a.victim, a.hit, a.path = aWarn, 1.0, v, false, nil
				a.icon, a.iconT = "SSSSS!", 1.2
				if p, ok := v.(*Player); ok {
					a.face = math.Copysign(1, p.pos.X-a.pos.X)
				} else if gu, ok := v.(*Guest); ok {
					a.face = math.Copysign(1, gu.pos.X-a.pos.X)
				}
			}
		} else {
			a.warnT = math.Max(0, a.warnT-dt)
		}
	case aWarn:
		a.timer -= dt
		if a.timer <= 0 {
			if g.snakeTarget(a, 30) != nil && g.rng.Float64() < 0.85 {
				a.state, a.timer, a.hit = aKick, 0.5, false
			} else {
				a.state, a.timer = aIdle, 2
			}
		}
	case aKick:
		a.timer -= dt
		if !a.hit && a.timer < 0.3 {
			a.hit = true
			g.snd.Play("thud")
			switch v := a.victim.(type) {
			case *Player:
				if !v.dead && !v.hidden && dist(v.pos, a.pos) < 36 {
					g.die("Te ha mordido una serpiente")
				}
			case *Guest:
				if !v.out() && dist(v.pos, a.pos) < 36 {
					g.snakeBite(v)
				}
			}
		}
		if a.timer <= 0 {
			a.state, a.timer = aIdle, 3
		}
	}
}

// snakeBite: the venom takes a few seconds; the guest panics meanwhile.
func (g *Game) snakeBite(gu *Guest) {
	gu.state, gu.path, gu.timer = gFun, nil, 3.2
	gu.fun = 14
	gu.bitten = 3.4
	gu.icon, gu.iconT = []string{"¡ME HA MORDIDO!", "¡AY, MI PIERNA!", "¡SERPIENTE!", "¡QUÉ PICOR!"}[g.rng.Intn(4)], 2.6
}

// updateBitten: running in circles, then the end.
func (g *Game) updateBitten(gu *Guest, dt float64) {
	if gu.timer < 0.9 && gu.timer+dt >= 0.9 {
		gu.icon, gu.iconT = "me mareo...", 1
	}
}

// venom: the bite is fatal even if the guest runs off somewhere else.
func (g *Game) venom(gu *Guest, dt float64) {
	gu.bitten -= dt
	if gu.bitten <= 0 {
		gu.bitten = 0
		g.killGuest(gu, false)
		g.sayNews("kill", map[string]string{"actor": "Una serpiente", "victim": guestLabel(gu)})
		g.checkDone()
	}
}

var _ = ebiten.Image{}

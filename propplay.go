package main

import (
	"flag"
	"math"
)

// ---------------------------------------------------------------------------
// PLAYING THE MACHINES (feedback of v1.4.0): in the casino some guests go to play the slot machines ("slot", the
// recreativas) and to the roulette tables, and stay a while playing.
//
// The same pattern as the seats (seats.go): the places to play are built from the props of the level (machBookOf,
// rebuilt when the level or its props change); now and then (in the "rest" choices of guestPick, only in a level that
// has machines) an idle guest books a free place, walks there and plays for 15-35 s in the state gLounge: he is an
// ordinary guest all the time (he can be killed, he witnesses, any alarm takes him away and frees the place). While
// somebody plays a slot machine its reels spin (propanim.go: machBusy); the player is drawn working the buttons, and
// cheers now and then (a short line in a bubble, at most one every 8-14 s per player).
//
// The dice are g.rng only inside a level that has places to play: every other level keeps exactly its old choices.
// ---------------------------------------------------------------------------

type machSpot struct {
	at   Vec // where the player stands (walkable: his gu.pos)
	face Dir
	prop *Prop
	by   *Guest
}

type machSit struct {
	s      *machSpot
	t0     float64 // booked
	at     float64 // arrived (0 = still walking)
	cheer  float64 // game time until which he cheers
	nextLn float64 // next line
}

type machBook struct {
	lvl   *Level
	n     int
	spots []*machSpot
	who   map[*Guest]*machSit
	busy  map[*Prop]bool // props somebody is playing right now
	next  map[*Guest]float64
}

const (
	machMin     = 15.0
	machMax     = 35.0
	machWalkMax = 30.0
	machChance  = 0.6 // of the "rest" choices in a level with machines
)

// machOff: tests switch the players off.
var machOff bool

// machRels: the places round a kind of machine (from the anchor) and where the player looks.
func machRels(name string) []struct {
	dx, dy float64
	face   Dir
} {
	type rel = struct {
		dx, dy float64
		face   Dir
	}
	switch name {
	case "slot":
		return []rel{{0, 8, DirUp}}
	case "roulette":
		return []rel{{-16, 8, DirUp}, {16, 8, DirUp}, {-40, -8, DirRight}, {40, -8, DirLeft}}
	}
	return nil
}

// machBookOf: the places of the current level (built when the level or its props change).
func (g *Game) machBookOf() *machBook {
	b := &g.mach
	l := g.lvl
	if l == nil || (b.lvl == l && b.n == len(l.Props)) {
		return b
	}
	for gu := range b.who { // a new level (or props changed): everybody stops playing
		delete(b.who, gu)
	}
	if b.lvl != l {
		b.next = map[*Guest]float64{}
	}
	if b.who == nil {
		b.who = map[*Guest]*machSit{}
	}
	if b.busy == nil {
		b.busy = map[*Prop]bool{}
	}
	if b.next == nil {
		b.next = map[*Guest]float64{}
	}
	b.lvl, b.n, b.spots = l, len(l.Props), nil
	var reach []bool
	for _, pr := range l.Props {
		rels := machRels(pr.Name)
		if len(rels) == 0 || pr.gone {
			continue
		}
		if reach == nil {
			reach = l.reachFrom(l.NearestFree(l.Def.Start))
		}
		for _, r := range rels {
			at, ok := l.seatApproach(Vec{pr.X + r.dx, pr.Y + r.dy}, reach)
			if !ok {
				continue
			}
			dup := false
			for _, s := range b.spots {
				if dist(s.at, at) < 9 {
					dup = true
					break
				}
			}
			if !dup {
				b.spots = append(b.spots, &machSpot{at: at, face: r.face, prop: pr})
			}
		}
	}
	return b
}

// machBusy: somebody is playing this prop right now (the slot machine spins its reels).
func (g *Game) machBusy(pr *Prop) bool {
	return len(g.mach.busy) > 0 && g.mach.busy[pr]
}

// machPick: called by guestPick in its "rest" choices; true = the guest is on his way to play.
func (g *Game) machPick(gu *Guest) bool {
	if machOff || g.lvl == nil {
		return false
	}
	b := g.machBookOf()
	if len(b.spots) == 0 || b.who[gu] != nil || b.next[gu] > g.time || !seatCanSit(g, gu) {
		return false
	}
	if g.rng.Float64() >= machChance {
		return false
	}
	var best *machSpot
	bs := math.MaxFloat64
	for _, s := range b.spots {
		if s.by != nil || s.prop.gone {
			continue
		}
		d := dist(gu.pos, s.at)
		if d > 420 {
			continue
		}
		if sc := d + g.rng.Float64()*120; sc < bs {
			best, bs = s, sc
		}
	}
	if best == nil {
		return false
	}
	path := g.lvl.PathG(gu.pos, best.at)
	if path == nil {
		b.next[gu] = g.time + 10
		return false
	}
	best.by = gu
	b.who[gu] = &machSit{s: best, t0: g.time}
	gu.path = path
	gu.state, gu.next, gu.dur = gGoto, gLounge, machMin+g.rng.Float64()*(machMax-machMin)
	return true
}

// machFree gives the place back (and starts the guest's cooldown).
func (g *Game) machFree(gu *Guest) {
	b := &g.mach
	st := b.who[gu]
	if st == nil {
		return
	}
	if st.s.by == gu {
		st.s.by = nil
	}
	delete(b.who, gu)
	if b.next == nil {
		b.next = map[*Guest]float64{}
	}
	b.next[gu] = g.time + 50 + float64(int(gu.off*1000)%40)
}

// playing: the guest is at his machine now.
func (g *Game) machPlaying(gu *Guest) *machSit {
	if len(g.mach.who) == 0 {
		return nil
	}
	st := g.mach.who[gu]
	if st == nil || st.at == 0 || gu.state != gLounge {
		return nil
	}
	return st
}

var (
	slotLines     = []string{"¡Vamos, vamos!", "¡Tres cerezas!", "Una más y lo dejo", "¡PREMIO!", "Esta máquina me odia"}
	rouletteLines = []string{"¡Todo al rojo!", "¡Al negro!", "¡El 17, el 17!", "¡He ganado!", "Hagan juego..."}
)

// machNowF: -machnow sends up to ten guests to the machines a second after the start (debug, for captures).
var machNowF = flag.Bool("machnow", false, "debug: up to ten guests go to play the slot machines and the roulette a second after the start")
var machNowLvl *Level

// updateMachines: called every frame after the guests moved.
func (g *Game) updateMachines(dt float64) {
	if g.lvl == nil {
		return
	}
	b := g.machBookOf()
	if *machNowF && machNowLvl != g.lvl && g.time > 1 && len(b.spots) > 0 {
		machNowLvl = g.lvl
		sent := 0
		for _, gu := range g.guests {
			if sent < 10 && !gu.out() && b.who[gu] == nil && (gu.state == gIdle || gu.state == gDance || gu.state == gGoto) && gu.timer < 1000 {
				gu.state, gu.timer, gu.path = gIdle, 0, nil
				for i := 0; i < 4 && b.who[gu] == nil; i++ {
					g.machPick(gu)
				}
				if b.who[gu] != nil {
					sent++
				}
			}
		}
	}
	for p := range b.busy {
		delete(b.busy, p)
	}
	for gu := range b.who { // guests that left the party (no rng here: the map order is random)
		if gu.out() {
			g.machFree(gu)
		}
	}
	for _, gu := range g.guests { // in the order of the guests: the dice stay the same run after run
		st := b.who[gu]
		if st == nil {
			continue
		}
		s := st.s
		if b.lvl != g.lvl || s.by != gu || gu.out() || s.prop.gone {
			g.machFree(gu)
			continue
		}
		if st.at == 0 { // on the way
			switch {
			case gu.state == gLounge:
				if dist(gu.pos, s.at) > 10 {
					g.machFree(gu)
					continue
				}
				st.at = g.time
				gu.pos, gu.dir, gu.path = s.at, s.face, nil
				if gu.timer > machMax {
					gu.timer = machMax
				}
				st.nextLn = g.time + 3 + float64(int(gu.off*100)%5)
			case gu.state != gGoto || gu.next != gLounge:
				g.machFree(gu) // something else took him (a fright, a party event...)
			case g.time-st.t0 > machWalkMax:
				g.machFree(gu)
				gu.state, gu.timer, gu.path = gIdle, 1, nil
			}
			continue
		}
		if gu.state != gLounge || dist(gu.pos, s.at) > 6 { // the game is over, or an alarm took him away
			g.machFree(gu)
			continue
		}
		gu.dir, gu.moving = s.face, false
		b.busy[s.prop] = true
		if g.time >= st.nextLn && gu.icon == "" && gu.iconT <= 0 {
			st.nextLn = g.time + 8 + g.rng.Float64()*6
			if g.rng.Float64() < 0.5 {
				lines := slotLines
				if s.prop.Name == "roulette" {
					lines = rouletteLines
				}
				gu.icon, gu.iconT = lines[g.rng.Intn(len(lines))], 2.4
				st.cheer = g.time + 1.6
			}
		}
	}
}

// machPose: the pose of a guest at his machine (ok false = not playing).
func (g *Game) machPose(gu *Guest) (int, float64, bool) {
	st := g.machPlaying(gu)
	if st == nil {
		return 0, 0, false
	}
	t := g.time
	if t < st.cheer {
		return AnimCheer, t*6 + gu.off, true
	}
	if st.s.prop.Name == "roulette" {
		if int(t/3+gu.off*5)%3 == 0 {
			return AnimTalk, t*5 + gu.off, true
		}
		return AnimIdle, t*1.3 + gu.off, true
	}
	return AnimDJ, t*4 + gu.off, true // the buttons and the lever
}

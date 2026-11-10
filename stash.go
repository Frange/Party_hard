package main

import (
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// STASHES: some containers (crates, barrels, wardrobes, dumpsters, fridges, freezers, gift tables) hide an object
// that the player can take with E. About one container in six (18 %) has one, decided ONCE when the level is first
// looked at (own dice, g.rng is never moved), so it is the same during the whole party.
//
//   - useful: a bottle of poison or a disguise (the existing items: they go through pickItem, so the slots, the
//     doses and the messages are the ones of the normal pick-ups);
//   - funny / rare: a banknote, a cold pizza, a rusty key, underpants with balloons, a plush toy, false teeth, a
//     plastic knife, a golden token. They give a few points (5-30) and a line; none has a gameplay effect.
//
// A faint glint now and then (a 4-pixel star for half a second every ~7 s, only the nearest three containers within
// 150 px of the player) tells where. Wardrobes and dumpsters are also hiding places: the first E searches, the next
// ones hide as always (and a player who carries a body disposes of it: the search needs free hands).
// ---------------------------------------------------------------------------

var stashKinds = map[string]bool{"crate": true, "barrel": true, "closet": true, "dumpster": true, "fridge": true, "freezer": true, "giftable": true}

const (
	stashChance = 0.18
	stashReach  = 16.0
	stashGlintR = 150.0
)

type stashItem struct {
	id     string
	weight int
	pts    int
	say    string
}

var stashTable = []stashItem{
	{"poison", 12, 0, ""},
	{"disguise", 12, 0, ""},
	{"banknote", 10, 30, pevTexts("Un billete de 20 arrugado. ¡A la caja!")[0]},
	{"pizza", 12, 15, pevTexts("Una pizza fría. Aun así, te la comes")[0]},
	{"key", 10, 10, pevTexts("Una llave oxidada. No abre nada... aún")[0]},
	{"undies", 10, 20, pevTexts("¡Calzoncillos con globos! Ni preguntes")[0]},
	{"plush", 8, 10, pevTexts("Un peluche con cara de pocos amigos")[0]},
	{"teeth", 8, 10, pevTexts("Una dentadura postiza. Te sonríe")[0]},
	{"knife", 8, 10, pevTexts("Una navaja... de plástico. Da más pena")[0]},
	{"token", 6, 25, pevTexts("¡Una ficha dorada de casino!")[0]},
}

// stashState: the stashes of the level in play (rolled the first time they are needed).
type stashState struct {
	lvl  *Level
	item map[*Prop]*stashItem
}

var (
	stash     stashState
	stashSeed int64 // tests: non-zero = deterministic dice
)

func stashPick(r *rand.Rand) *stashItem {
	tot := 0
	for _, s := range stashTable {
		tot += s.weight
	}
	n := r.Intn(tot)
	for i := range stashTable {
		if n < stashTable[i].weight {
			return &stashTable[i]
		}
		n -= stashTable[i].weight
	}
	return &stashTable[0]
}

// stashRoll decides the stashes of the level in play (once per level load).
func stashRoll(g *Game) {
	if g.lvl == nil {
		stash = stashState{}
		return
	}
	if stash.lvl == g.lvl && stash.item != nil {
		return
	}
	sd := stashSeed
	if sd == 0 {
		sd = time.Now().UnixNano() ^ 0x57a5b
	}
	r := rand.New(rand.NewSource(sd + int64(g.levelIdx)*7919))
	stash = stashState{lvl: g.lvl, item: map[*Prop]*stashItem{}}
	if g.stealth || g.levelIdx == specialJurassic {
		return
	}
	for _, pr := range g.lvl.Props {
		roll := r.Float64() // always consumed
		it := stashPick(r)
		if stashKinds[pr.Name] && !pr.gone && roll < stashChance {
			stash.item[pr] = it
		}
	}
}

// stashHas: the prop still holds an object.
func (g *Game) stashHas(pr *Prop) *stashItem {
	stashRoll(g)
	return stash.item[pr]
}

// stashInteract: E next to a container with an object takes it.
func (g *Game) stashInteract(p *Player) bool {
	if p == nil || p.dead || p.hidden || p.carrying != nil || p.jail.held || g.lvl == nil || g.mode != ModePlay {
		return false
	}
	stashRoll(g)
	if len(stash.item) == 0 {
		return false
	}
	var best *Prop
	bd := stashReach
	for pr := range stash.item {
		if pr.gone || pr.lit || pr.wreck {
			continue
		}
		if d := distRect(p.pos, pr.Foot); d <= bd {
			best, bd = pr, d
		}
	}
	if best == nil {
		return false
	}
	it := stash.item[best]
	if it.id == "poison" || it.id == "disguise" {
		if !canTake(p, it.id) { // the hands are full of the same: leave it for later
			return false
		}
	}
	delete(stash.item, best)
	g.stashGive(p, best, it)
	return true
}

func (g *Game) stashGive(p *Player, pr *Prop, it *stashItem) {
	g.snd.Play("spark")
	at := Vec{pr.Foot.X + pr.Foot.W/2, pr.Foot.Y}
	switch it.id {
	case "poison", "disguise":
		g.pickItem(&Prop{K: &PropKind{Item: it.id}}) // the normal pick-up: slots, doses and messages
		return
	}
	g.award(at, it.pts, "")
	g.say(it.say)
}

// stashItems: the glints, in the y-sorted list (next to the party events).
func (g *Game) stashItems(dst *ebiten.Image, items []drawItem) []drawItem {
	if g.lvl == nil || g.mode != ModePlay || g.p == nil {
		return items
	}
	stashRoll(g)
	if len(stash.item) == 0 {
		return items
	}
	plot := g.plotter(dst)
	px := g.pevPx()
	n := 0
	for i, pr := range g.lvl.Props { // (props order: stable, no map order -> no flicker)
		if stash.item[pr] == nil || pr.gone || n >= 3 {
			continue
		}
		if g.pevPlayerFar(Vec{pr.Foot.X + pr.Foot.W/2, pr.Foot.Y}, stashGlintR) {
			continue // nobody is within reach of the glint
		}
		if !stashGlint(g.time, i) {
			continue
		}
		n++
		pr := pr
		items = append(items, drawItem{pr.Sort + 0.5, func() {
			cx, cy := g.pevXY(Vec{pr.Foot.X + pr.Foot.W/2, pr.Foot.Y})
			cy -= int(float64(pr.K.H) * 0.7)
			drawGlintPix(plot, cx, cy, px, g.time)
		}})
	}
	return items
}

// stashGlint: is the container twinkling now? Half a second every ~7 s, each one at its own moment.
func stashGlint(t float64, i int) bool {
	period := 7.0 + float64(i%5)*0.6
	ph := t + float64(i)*1.37
	f := ph - period*float64(int(ph/period))
	return f < 0.5
}

// drawGlintPix: a 4-point star of white and gold that grows and shrinks (pixels of the given size).
func drawGlintPix(plot plotFn, cx, cy, px int, t float64) {
	f := t*8 - float64(int(t*8)/3*3) // 0..3 frames
	r := 1
	if f >= 1 && f < 2 {
		r = 2
	}
	plot(cx-px, cy-px, 3*px, 3*px, pevGoldDk)
	plot(cx, cy-(r+1)*px, px, (2*r+3)*px, pevGold)
	plot(cx-(r+1)*px, cy, (2*r+3)*px, px, pevGold)
	plot(cx, cy, px, px, pevWhite)
}

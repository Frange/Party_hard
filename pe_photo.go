package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "fotografo" (idea 272, FOTÓGRAFO DEL EVENTO): sometimes a guest turns up with a camera, a red cap and a waistcoat with
// pockets and goes round asking groups of guests to pose: they line up, smile ("¡Patata!"), FLASH (a short white glow over them
// and a shutter click) and he walks to the next group. Three or four photos and he goes back to the party.
//
// The flash dazzles a player who is in front of the camera, close and in sight: the screen gives the same soft white flash of
// the blasts (g.flash, ~0.4 s fading), and the controls are never blocked. The photographer is an ordinary guest: he can be
// killed (the photos stop and the group goes), he is a witness with the usual rules of guests (nothing more: he is not a
// security camera). Own dice (partyDice), never g.rng.
// ---------------------------------------------------------------------------

const (
	photoShotsMin = 3
	photoShotsMax = 4
	photoPoseLen  = 1.7   // seconds of smiles before the flash
	photoAfter    = 1.3   // seconds after the flash before they go
	photoDazzleR  = 120.0 // a player closer than this to the camera, in front of it and in sight, is dazzled
	photoDazzle   = 0.4   // seconds of white (g.flash)
	photoDist     = 44.0  // the photographer stands this far in front of the group
)

var (
	photoAskLines   = pevTexts("¡Una foto, por favor!", "¡Juntaos un poco!", "¡Para el álbum!", "¡Mirad aquí!")
	photoCheese     = pevTexts("¡Patata!", "¡Decid patata!", "¡Sonreíd!")
	photoGroupLines = pevTexts("¿Salgo bien?", "¡Mi lado bueno!", "¡Patataaa!")
	photoAfterLines = pevTexts("¡Preciosos!", "¡Qué fotón!", "Esta sale en portada")
	photoFlashLine  = pevTexts("¡Flash!")
)

const (
	phWalk = iota
	phPose
	phFlash
	phDone
)

type photoState struct {
	ph      *pevMem
	group   []*pevMem
	centre  Vec
	spot    Vec
	phase   int
	t0      float64
	shots   int
	want    int
	flashAt float64
	dazzled int // times a player was dazzled (tests)
}

func (s *photoState) pevHas(gu *Guest) bool {
	return s.ph != nil && s.ph.gu == gu || pevHasIn(s.group, gu)
}

// photoIcons: the parties with an event photographer (the elegant or festive ones).
var photoIcons = map[string]bool{"house": true, "beach": true, "neon": true, "ranch": true, "boat": true, "disco": true, "roof": true,
	"casino": true, "italy": true, "train": true, "cake": true, "disco70": true}

func init() {
	registerPartyEvent(&partyEvent{ID: "fotografo", Name: "Fotógrafo del evento", Group: "photo",
		Prob: 0.13, AtMin: 30, AtMax: 240, Dur: 60,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && photoIcons[t.icon] },
		Start:    photoStart, Items: photoItems,
		Update: pevGuard(photoUpdate, func(g *Game, a *partyAct) { photoEnd(g, a.D.(*photoState)) })})
}

func photoStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var cand []*Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && gu.name == "" && (g.pevPlayerFar(gu.pos, 60) || pevDebugOn("fotografo")) {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 4 {
		return false
	}
	ph := cand[r.Intn(len(cand))]
	st := &photoState{ph: &pevMem{gu: ph}, want: photoShotsMin + r.Intn(photoShotsMax-photoShotsMin+1)}
	if !g.photoNextGroup(st, r) {
		return false
	}
	a.D = st
	return true
}

// photoNextGroup: picks 2-4 guests standing together, sends them into a row and the photographer in front of it.
func (g *Game) photoNextGroup(st *photoState, r *rand.Rand) bool {
	ph := st.ph.gu
	var seeds []*Guest
	for _, gu := range g.guests {
		if gu != ph && g.pevFreeGuest(gu) && (dist(gu.pos, ph.pos) < 200 || pevDemo("fotografo") != "") {
			seeds = append(seeds, gu)
		}
	}
	r.Shuffle(len(seeds), func(i, j int) { seeds[i], seeds[j] = seeds[j], seeds[i] })
	demo := pevDemo("fotografo") != ""
	if demo { // capture (-pevent fotografo:near): the group nearest to the player, everybody already in place
		for i := 1; i < len(seeds); i++ {
			for j := i; j > 0 && dist(seeds[j].pos, g.p.pos) < dist(seeds[j-1].pos, g.p.pos); j-- {
				seeds[j], seeds[j-1] = seeds[j-1], seeds[j]
			}
		}
	}
	for _, s := range seeds {
		var grp []*Guest
		grp = append(grp, s)
		for _, o := range seeds {
			if o != s && len(grp) < 4 && dist(o.pos, s.pos) < 40 {
				grp = append(grp, o)
			}
		}
		if len(grp) < 2 {
			continue
		}
		c := s.pos
		slots := make([]Vec, len(grp))
		ok := true
		for i := range grp {
			q := Vec{c.X + (float64(i)-float64(len(grp)-1)/2)*11, c.Y}
			if !g.lvl.Free(q) || g.lvl.SolidP(q.X, q.Y) || g.lvl.InPool(q) {
				ok = false
				break
			}
			slots[i] = q
		}
		spot := Vec{c.X, c.Y + photoDist}
		if !ok || !g.lvl.Free(spot) || g.lvl.SolidP(spot.X, spot.Y) || g.lvl.InPool(spot) || !g.lvl.LOS(spot, c) ||
			pevPathLen(g.lvl.PathG(ph.pos, spot), ph.pos) > 240 {
			continue
		}
		var mem []*pevMem
		if demo {
			ph.pos = spot
		}
		for i, gu := range grp {
			if demo {
				gu.pos = slots[i]
			}
			m := &pevMem{gu: gu, pt: slots[i]}
			if g.pevSendTo(m, slots[i]) {
				mem = append(mem, m)
			}
		}
		if len(mem) < 2 {
			pevReleaseAll(mem, r)
			continue
		}
		if !g.pevSendTo(st.ph, spot) {
			pevReleaseAll(mem, r)
			continue
		}
		st.group, st.centre, st.spot = mem, c, spot
		pevDemoLog("fotografo", c)
		st.phase, st.t0 = phWalk, g.time
		if ph.icon == "" {
			ph.icon, ph.iconT = pevPick(r, photoAskLines), 1.8
		}
		return true
	}
	return false
}

func photoUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*photoState)
	r := partyDice()
	if !st.ph.mine() { // killed, frightened, taken by something else: the photos are over
		photoEnd(g, st)
		return true
	}
	st.group = pevPruneMems(st.group)
	age := g.time - st.t0
	ph := st.ph.gu
	switch st.phase {
	case phWalk:
		arrived := ph.state == gIdle && ph.timer > 1000
		in := 0
		for _, m := range st.group {
			gu := m.gu
			if gu.state == gIdle && gu.timer > 1000 {
				m.want = gIdle
				m.keep()
				gu.dir = DirDown
				in++
			}
		}
		if arrived {
			st.ph.want = gIdle
			st.ph.keep()
			ph.dir = DirUp
		}
		if arrived && (in == len(st.group) || age > 7) && in >= 2 {
			for i, m := range st.group {
				if !(m.gu.state == gIdle && m.gu.timer > 1000) {
					pevLetGo(m)
					continue
				}
				m.pose([]int{AnimCheer, AnimLaugh, AnimWave, AnimLove}[(i+st.shots)%4])
				m.gu.dir = DirDown
				if i == 1 && m.gu.icon == "" {
					m.gu.icon, m.gu.iconT = pevPick(r, photoGroupLines), 1.5
				}
			}
			st.group = pevPruneMems(st.group)
			st.ph.pose(AnimIdle)
			ph.dir = DirUp
			ph.icon, ph.iconT = pevPick(r, photoCheese), 1.5
			st.phase, st.t0 = phPose, g.time
		} else if age > 20 { // they never got together: next group (or the end)
			pevReleaseAll(st.group, r)
			st.group = nil
			if !g.photoNextGroup(st, r) {
				photoEnd(g, st)
				return true
			}
		}
	case phPose:
		ph.dir = DirUp
		for _, m := range st.group {
			m.gu.dir = DirDown
		}
		if age >= photoPoseLen {
			st.phase, st.t0, st.flashAt = phFlash, g.time, g.time
			st.shots++
			g.snd.Play("clac")
			g.photoDazzle(st)
		}
	case phFlash:
		ph.dir = DirUp
		if age >= photoAfter {
			pevReleaseAll(st.group, r)
			st.group = nil
			if ph.icon == "" || ph.iconT < 0.5 {
				ph.icon, ph.iconT = pevPick(r, photoAfterLines), 1.8
			}
			if st.shots >= st.want || !g.photoNextGroup(st, r) {
				photoEnd(g, st)
				return true
			}
		}
	}
	return false
}

// photoDazzle: a player in front of the camera, close and in sight, sees white for a moment (never blocked).
func (g *Game) photoDazzle(st *photoState) {
	cam := Vec{st.spot.X, st.spot.Y - 26}
	for _, p := range g.players() {
		if p == nil || p.dead || p.hidden {
			continue
		}
		d := dist(cam, p.pos)
		if d > photoDazzleR || d < 1 {
			continue
		}
		// the camera points up (to the group): the player must be in front of the photographer
		if ds := dist(st.spot, p.pos); ds < 1 || (st.spot.Y-p.pos.Y)/ds < 0.3 || !g.lvl.LOS(cam, p.pos) {
			continue
		}
		g.flash = math.Max(g.flash, photoDazzle)
		st.dazzled++
		if p == g.p && st.dazzled == 1 {
			g.say(photoFlashLine[0])
		}
	}
}

func photoEnd(g *Game, st *photoState) {
	if st.phase == phDone {
		return
	}
	r := partyDice()
	st.phase = phDone
	pevReleaseAll(st.group, r)
	st.group = nil
	pevLetGo(st.ph)
}

// ---- drawing ------------------------------------------------------------------------------------------------

var (
	photoCap   = color.RGBA{0xd0, 0x30, 0x38, 255}
	photoVest  = color.RGBA{0xa8, 0x94, 0x60, 255}
	photoVestD = color.RGBA{0x70, 0x60, 0x38, 255}
	photoBody  = color.RGBA{0x24, 0x24, 0x2c, 255}
)

// drawPhotoGearPix: the cap, the waistcoat and the camera of the photographer whose feet are at (cx, cy). Facing up (his back
// to us) the camera is held up at his eyes and its ends stick out at both sides of his head.
func drawPhotoGearPix(plot plotFn, cx, cy, px int, k func(float64) int, dir Dir, flash bool) {
	top := cy - k(31)
	// waistcoat with pockets
	vy := cy - k(24)
	plot(cx-4*px, vy, 3*px, k(9), photoVest)
	plot(cx+2*px, vy, 3*px, k(9), photoVest)
	plot(cx-4*px, vy+k(5), 3*px, px, photoVestD)
	plot(cx+2*px, vy+k(5), 3*px, px, photoVestD)
	// cap
	plot(cx-3*px, top-px, 7*px, 2*px, photoCap)
	switch dir {
	case DirLeft:
		plot(cx-6*px, top+px, 3*px, px, photoCap)
	case DirRight:
		plot(cx+4*px, top+px, 3*px, px, photoCap)
	case DirDown:
		plot(cx-2*px, top+px, 5*px, px, shade(photoCap, 0.7))
	}
	// camera
	switch dir {
	case DirUp:
		y := cy - k(29)
		plot(cx-6*px, y, 2*px, 3*px, photoBody)
		plot(cx+5*px, y, 2*px, 3*px, photoBody)
		plot(cx-6*px, y-px, 13*px, px, photoBody)
		plot(cx+5*px, y-2*px, 2*px, px, color.RGBA{0xe0, 0xe0, 0xe8, 255}) // the flash on top
		if flash {
			plot(cx+3*px, y-5*px, 6*px, 4*px, pevWhite)
		}
	default:
		x, y := cx-2*px, cy-k(22)
		if dir == DirLeft {
			x -= 3 * px
		} else if dir == DirRight {
			x += 3 * px
		}
		plot(x, y, 5*px, 3*px, photoBody)
		plot(x+px, y+px, 2*px, px, color.RGBA{0x80, 0xa0, 0xd0, 255}) // the lens
		plot(x+3*px, y-px, px, px, color.RGBA{0xe0, 0xe0, 0xe8, 255})
	}
}

func photoItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*photoState)
	if st.phase == phDone || st.ph == nil || !st.ph.mine() {
		return items
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	gu := st.ph.gu
	flashing := st.phase == phFlash && g.time-st.flashAt < 0.18
	items = append(items, drawItem{gu.pos.Y + 0.2, func() {
		cx, cy := g.pevXY(gu.pos)
		dir := gu.dir
		if gu.moving {
			dir = g.gaitDir(gu)
		}
		drawPhotoGearPix(plot, cx, cy, px, k, dir, flashing)
	}})
	if st.phase == phFlash && g.time-st.flashAt < 0.35 {
		u := (g.time - st.flashAt) / 0.35
		for _, m := range st.group {
			m := m
			items = append(items, drawItem{m.gu.pos.Y + 0.3, func() {
				cx, cy := g.pevXY(m.gu.pos)
				al := uint8(200 * (1 - u))
				fillRect(dst, float64(cx-k(9)), float64(cy-k(40)), float64(k(18)), float64(k(40)), color.RGBA{al, al, al, al})
			}})
		}
	}
	return items
}

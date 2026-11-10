package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "karaoke": a GUEST (the main characters want to ruin the party, not to sing) takes a microphone in front of the stage
// and sings out of tune: coloured notes wobble over his head. Three or four guests stop to listen: if he was good they clap, if
// not they cover their ears and complain. At the end a sign shows his score ("¡BIS!" when it went well). Nobody is hurt and
// nothing counts for points. The singer and the audience are let go when it is over (or at once if one is killed).
//
// WHEN: parties with a stage (or an indoor dance floor with a DJ: the spot is the front of the floor), 22 % of them, between 60 and 230 s, once. Group "stage" (never with another show on the stage).
// ---------------------------------------------------------------------------

// karaokeIcons: parties without a stage where a corner of the dance floor does as one.
var karaokeIcons = map[string]bool{"house": true, "neon": true, "disco": true, "disco70": true}

const (
	kGather = iota
	kSing
	kResult
)

const (
	kSingLen    = 14.0
	kResultLen  = 4.6
	kGatherWait = 14.0
)

var karaokeLines = pevTexts("¡Ooooh, mi corazón!", "¡Nanana nanana!", "¡Soy el rey del micro!", "¡Laaa, la-la-laaa!", "¡Que suene, que suene!",
	"¡Aaay, ay, ay, ayyy!")

var karaokeCrowd = pevTexts("¡Mis oídos!", "¡Que alguien pare eso!", "¡Auch!", "¡Eso es un delito!", "¡Qué voz tan... valiente!",
	"¡Bravo!", "¡Otra, otra!", "¡Qué artista!", "¡Cantas genial!", "¡Se me pone la piel de gallina!")

var karaokeSign = pevTexts("PUNTOS", "¡BIS!", "¡AUCH!")

type karaokeState struct {
	singer *pevMem
	gait   string
	crowd  pevCrowd
	spot   Vec
	phase  int
	pt0    float64
	good   bool
	score  int
	lyrAt  float64
	reacts bool
}

func (s *karaokeState) pevHas(gu *Guest) bool {
	return s.singer != nil && s.singer.gu == gu || s.crowd.pevHas(gu)
}

func init() {
	registerPartyEvent(&partyEvent{ID: "karaoke", Name: "Karaoke", Group: "stage",
		Prob: 0.22, AtMin: 60, AtMax: 230, Dur: 40, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool {
			return t.roam && t.guests >= 20 && (t.stage || t.dance && t.dj && karaokeIcons[t.icon])
		},
		Start: karaokeStart, Items: karaokeItems, Over: karaokeOver,
		Update: pevGuard(karaokeUpdate, func(g *Game, a *partyAct) { karaokeEnd(g, a.D.(*karaokeState)) })})
}

func karaokeStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var spot Vec
	found := false
	try := func(p Vec) {
		if !found && g.pevClear(p, 11) && g.pevPlayerFar(p, 60) {
			spot, found = p, true
		}
	}
	for _, pr := range g.lvl.Props {
		if pr.Name != "stage" || pr.gone {
			continue
		}
		for _, dy := range []float64{16, 22, 30} {
			for _, dx := range []float64{0, -22, 22, -44, 44} {
				try(Vec{pr.X + dx, pr.Foot.Y + pr.Foot.H + dy})
			}
		}
	}
	if !found { // no stage: the front of the dance floor
		for _, dr := range g.lvl.Def.Dance {
			for _, dy := range []float64{14, 24, 34} {
				for _, dx := range []float64{0, -30, 30} {
					try(Vec{dr.X + dr.W/2 + dx, dr.Y + dr.H + dy})
				}
			}
		}
	}
	if !found {
		return false
	}
	var best *Guest
	for _, gu := range g.guests {
		if !g.pevFreeGuest(gu) || gu.name != "" || pevFxs[gu] != nil {
			continue
		}
		g.gaitInit(gu)
		if gu.gait != "" {
			continue
		}
		if d := dist(gu.pos, spot); d < 280 && (best == nil || d < dist(best.pos, spot)) && g.lvl.PathG(gu.pos, spot) != nil {
			best = gu
		}
	}
	if best == nil {
		return false
	}
	st := &karaokeState{spot: spot, pt0: g.time, singer: &pevMem{gu: best}}
	if !g.pevSendTo(st.singer, spot) {
		return false
	}
	st.crowd.gather(g, r, spot, 3+r.Intn(2), 28, 42, func(o *Guest) bool { return o == best })
	if len(st.crowd.mem) < 2 {
		pevReleaseAll(st.crowd.mem, r)
		pevRelease(st.singer, r)
		return false
	}
	pevFxPurge(g)
	st.gait = best.gait
	best.gait = "hips"
	st.good = r.Float64() < 0.45
	if st.good {
		st.score = 78 + r.Intn(22)
	} else {
		st.score = 6 + r.Intn(52)
	}
	a.D = st
	return true
}

func karaokeUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*karaokeState)
	r := partyDice()
	if !st.singer.mine() { // he was frightened, hurt or killed: no more show
		karaokeEnd(g, st)
		return true
	}
	gu := st.singer.gu
	age := g.time - st.pt0
	st.crowd.update(g, r, nil, 0, nil)
	switch st.phase {
	case kGather:
		if gu.state == gIdle && gu.timer > 1000 {
			st.singer.want = gIdle
			gu.dir = DirDown
			ready := 0
			for _, m := range st.crowd.mem {
				if m.gu.state == gIdle && m.gu.timer > 1000 {
					ready++
				}
			}
			if ready >= len(st.crowd.mem) || age > kGatherWait {
				st.phase, st.pt0, st.lyrAt = kSing, g.time, g.time+0.5
			}
		} else if age > kGatherWait+8 {
			karaokeEnd(g, st)
			return true
		}
	case kSing:
		st.singer.keep()
		gu.dir = DirDown
		if g.time >= st.lyrAt {
			st.lyrAt = g.time + 3.6
			gu.icon, gu.iconT = pevPick(r, karaokeLines), 3
		}
		if age > 4.5 && !st.reacts {
			st.reacts = true
			lines := karaokeCrowd[:5]
			if st.good {
				lines = karaokeCrowd[5:]
			}
			for _, m := range st.crowd.mem {
				if m.gu.icon == "" && r.Float64() < 0.7 {
					m.gu.icon, m.gu.iconT = pevPick(r, lines), 2.4
				}
			}
		}
		if age >= kSingLen {
			st.phase, st.pt0 = kResult, g.time
			gu.icon = ""
			if st.good {
				g.snd.Play("clap")
			}
		}
	case kResult:
		st.singer.keep()
		if age >= kResultLen {
			karaokeEnd(g, st)
			return true
		}
	}
	return false
}

func karaokeEnd(g *Game, st *karaokeState) {
	r := partyDice()
	if gu := st.singer.gu; gu.gait == "hips" {
		gu.gait = st.gait
	}
	pevLetGo(st.singer)
	st.crowd.release(g, r)
}

// drawBadNotesPix: notes that wobble up and down out of tune, with a flat sign beside the first one.
func drawBadNotesPix(plot plotFn, cx, top, px int, t float64) {
	for i := 0; i < 3; i++ {
		ph := math.Mod(t*0.5+float64(i)/3, 1)
		x := cx + (i-1)*7*px + int(math.Round(math.Sin(t*8+float64(i)*2)*2))*px
		y := top - int(math.Round(ph*14))*px + int(math.Round(math.Sin(t*11+float64(i)*1.7)*2))*px
		drawNotePix(plot, x, y, px, pevNoteCols[i], i != 1)
		if i == 0 { // the flat: a small "b"
			plot(x-3*px, y+2*px, px, 4*px, pevInk)
			plot(x-2*px, y+4*px, 2*px, 2*px, pevInk)
		}
	}
}

func karaokeItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*karaokeState)
	if !st.singer.mine() {
		return items
	}
	gu := st.singer.gu
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	items = append(items, drawItem{gu.pos.Y + 0.3, func() {
		cx, cy := g.pevXY(gu.pos)
		if gu.state == gIdle {
			drawMicPix(plot, cx, cy, px, k)
		}
		if st.phase == kSing {
			if st.good {
				drawNotesPix(plot, cx, cy-k(46), px, g.time, 3)
			} else {
				drawBadNotesPix(plot, cx, cy-k(46), px, g.time)
			}
		}
	}})
	if st.phase != kSing || g.time-st.pt0 < 4.5 {
		return items
	}
	for _, m := range st.crowd.mem {
		o := m.gu
		if !m.mine() || o.state != gIdle || o.timer < 1000 {
			continue
		}
		i := int(math.Abs(o.pos.X))
		items = append(items, drawItem{o.pos.Y + 0.3, func() {
			cx, cy := g.pevXY(o.pos)
			if st.good {
				drawClapPix(plot, cx, cy-k(34), px, g.time, i)
			} else {
				drawEarsPix(plot, cx, cy, px, k)
			}
		}})
	}
	return items
}

// karaokeOver: the sign with the score, over the singer's head (screen space, text on whole pixels).
func karaokeOver(g *Game, dst *ebiten.Image, a *partyAct) {
	st := a.D.(*karaokeState)
	if st.phase != kResult || !st.singer.mine() {
		return
	}
	gu := st.singer.gu
	plot := g.plotter(dst)
	k := g.pevK()
	cx, cy := g.pevXY(gu.pos)
	const w, h = 64, 40
	x := clampI(cx-w/2, 2, screenW-w-2)
	y := clampI(cy-k(50)-h, 2, screenH-h-2)
	back := color.RGBA{0x1c, 0x14, 0x30, 255}
	border := pevGold
	if !st.good {
		border = color.RGBA{0xc8, 0x50, 0x50, 255}
	}
	plot(x, y, w, h, border)
	plot(x+2, y+2, w-4, h-4, back)
	centre := func(s string, yy int, col color.RGBA) {
		tw := int(math.Round(g.textW(s, 1)))
		g.drawText(dst, s, float64(x+(w-tw)/2), float64(yy), col, 1)
	}
	centre("PUNTOS", y+5, pevWhite)
	d1, d2 := st.score/10, st.score%10
	dx := x + (w-16)/2
	drawDigitPix(plot, dx, y+15, 2, d1, clapCol)
	drawDigitPix(plot, dx+8, y+15, 2, d2, clapCol)
	if st.good {
		centre("¡BIS!", y+29, pevGold)
	} else {
		centre("¡AUCH!", y+29, color.RGBA{0xff, 0x90, 0x90, 255})
	}
}

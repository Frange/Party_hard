package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The secret ending: THE PARTY.
//
// Nothing like the quiet ending (grey, silent, short, ghosts walking home): this one is colour, noise
// and a long celebration on the island, in four acts, then colourful credits and a last gag.
//   act 1 (0-8 s)    the celebration: trophies up, bunting, disco ball, the dinosaurs in party hats, the guests
//                    of the whole game dancing round the stage and the cake
//   act 2 (8-20 s)   the conga: everybody (and a cat, a dog and a pig) runs off to line up and comes back in a
//                    conga across the whole island, at dusk
//   act 3 (20-28 s)  night, fireworks, everybody seen from behind looking at the sky
//   act 4 (28-36 s)  the family photo: the flash, and the picture comes out as a polaroid
// ---------------------------------------------------------------------------

const (
	partyLen        = 36.0
	partyCreditsLen = 34.0 // slower credits (36 px/s): the names can be read
)

// the acts
const (
	partyActConga = 8.0
	partyActNight = 20.0
	partyActPhoto = 28.0
	partyFlashAt  = 31.0
	partyCongaV   = 90.0 // px/s
	partyCongaGap = 16.0 // two congas side by side: the whole crowd crosses the island in its act
	partyGuests   = 60   // how many famous guests come to the party (plus the heroes and two rangers)
)

// partyCastNames: the famous faces that are always at the party (the balloons talk with some of them: sonic,
// shrek). The rest of the crowd is picked at random from every famous character of the game (famousNames).
var partyCastNames = []string{"sonic", "guybrush", "shrek", "homer", "bart", "marge", "lisa", "maggie", "bender", "fry",
	"goku", "pantera_rosa", "elvis", "esqueleto_bailarin", "jack_rey_calabaza", "mario", "luigi", "crash", "cell", "ron",
	"kratos", "solid_snake", "neo", "joker", "gb_venkman", "gb_spengler", "gb_stantz", "gb_winston", "chewbacca",
	"gandalf", "frodo", "knuckles", "lechuck", "johnny5", "don_quijote", "sancho_panza", "tortuga_leonardo",
	"tortuga_donatello", "tortuga_michelangelo", "tortuga_raphael", "michael_jackson", "papa"}

type partyState struct {
	built bool
	cast  []*CharSet
	names []string
	photo *ebiten.Image
	done  [10]bool
}

// partyBuild makes the guests of the party (once): everybody who is cheap to draw.
func (g *Game) partyBuild(e *endState) {
	p := &e.x.party
	if p.built {
		return
	}
	p.built = true
	g.sfxOnce("rex_roar", jrRoarSound)
	for i, n := range partyCastPick(g.partySeed()) {
		cs := g.partyGuestCS(n, int64(77+i))
		if cs == nil {
			continue
		}
		p.cast = append(p.cast, cs)
		p.names = append(p.names, n)
	}
	if g.as != nil { // the Power Rangers come too (two of them, not the whole team)
		for i, cs := range g.as.rangerCS {
			if i >= 2 {
				break
			}
			p.cast = append(p.cast, cs)
			p.names = append(p.names, "ranger")
		}
	}
}

// partySeed: a different crowd each time the ending is watched (fixed when there is no random source).
func (g *Game) partySeed() int64 {
	if g.rng != nil {
		return g.rng.Int63()
	}
	return 77
}

// partyCastPick: the names of the guests: the fixed ones first, then the rest of the famous characters
// at random, up to partyGuests. Never a killer, a cop or a kid (famousNames leaves them out).
func partyCastPick(seed int64) []string {
	all := famousNames()
	out := make([]string, 0, partyGuests)
	used := map[string]bool{}
	for _, n := range partyCastNames {
		if all[n] && !used[n] && len(out) < partyGuests {
			out = append(out, n)
			used[n] = true
		}
	}
	var rest []string
	for n := range all {
		if !used[n] {
			rest = append(rest, n)
		}
	}
	sort.Strings(rest)
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	for _, n := range rest {
		if len(out) >= partyGuests {
			break
		}
		out = append(out, n)
	}
	return out
}

// partyGuestCS: the look the game gives that famous guest: the edited one of assets/sprites/special_people
// when there is one, the built-in design otherwise.
func (g *Game) partyGuestCS(name string, seed int64) *CharSet {
	if g.store != nil && g.rigCache != nil {
		for _, ca := range g.store.Chars["special_people"] {
			if ca.Meta.Name == name {
				return g.rigFor(ca)
			}
		}
	}
	return castSet(name, seed)
}

// pMember: somebody of the crowd at an instant.
type pMember struct {
	name      string
	cs        *CharSet
	hero      int // 0 or 1 for the heroes, -1 for the guests
	x, y, hop float64
	anim      int
	dir       Dir
	ph        float64
	alpha     float64
}

// pRow is a stretch of a row of the crowd: the people stand on y, spread between x0 and x1.
type pRow struct{ y, x0, x1 float64 }

// partyRows: the rows of the crowd round the stage (act 1 and the photo) or looking at the sky (act 3),
// cut where the cake, the raptors and the heroes are, so nobody stands on top of them.
func partyRows(watch, two bool) []pRow {
	hx0, hx1 := 216.0, 266.0 // the heroes, in the middle
	if two {
		hx0, hx1 = 206, 278
	}
	if watch {
		return []pRow{{226, 14, 466}, {242, 14, hx0}, {242, hx1, 466}, {258, 14, hx0}, {258, hx1, 466}}
	}
	return []pRow{
		{204, 128, 176}, {204, 208, 272}, {204, 304, 336}, // just in front of the stage, between the raptors
		{218, 16, 56}, {218, 122, 176}, {218, 306, 466},
		{234, 14, 176}, {234, 304, 466},
		{252, 14, hx0}, {252, hx1, 466},
	}
}

// rowSpot spreads n people evenly along the rows (the same gap everywhere) and returns where the j-th is.
func rowSpot(rows []pRow, j, n int) (x, y float64) {
	total := 0.0
	for _, r := range rows {
		total += r.x1 - r.x0
	}
	if n < 1 || total <= 0 {
		return 60, 220
	}
	at := (float64(j) + 0.5) * total / float64(n)
	for _, r := range rows {
		l := r.x1 - r.x0
		if at <= l {
			return math.Round(r.x0 + at), r.y
		}
		at -= l
	}
	last := rows[len(rows)-1]
	return last.x1, last.y
}

func partySlot(j, n int, two bool) (x, y float64) { return rowSpot(partyRows(false, two), j, n) } // act 1

func partyWatch(j, n int, two bool) (x, y float64) { return rowSpot(partyRows(true, two), j, n) } // act 3

func partyPhoto(j, n int, two bool) (x, y float64) { return rowSpot(partyRows(false, two), j, n) } // act 4

// partyCrowd lays the heroes and the guests out at time t. The heroes come first.
func (g *Game) partyCrowd(p *partyState, t float64, two bool) []pMember {
	var out []pMember
	n := len(p.cast)
	nh := 1
	if two {
		nh = 2
	}
	total := nh + n
	for m := 0; m < total; m++ {
		mem := pMember{hero: -1, alpha: 1, dir: DirDown}
		var sx, sy, wx, wy, px, py float64
		if m < nh {
			mem.hero = m
			mem.name = "hero"
			mem.cs = g.sceneHero(m, false)
			base := 240.0
			if two {
				base = 226
			}
			sx, sy = base+float64(m)*28, 236
			wx, wy = base+float64(m)*28, 240
			px, py = base-4+float64(m)*30, 240
		} else {
			j := m - nh
			mem.name, mem.cs = p.names[j], p.cast[j]
			sx, sy = partySlot(j, n, two)
			wx, wy = partyWatch(j, n, two)
			px, py = partyPhoto(j, n, two)
		}
		lane := float64(m % 2) // two congas, one in front of the other
		off := float64(m/2) * partyCongaGap
		switch {
		case t < partyActConga: // act 1: dancing round the stage
			mem.x, mem.y = sx, sy
			mem.anim, mem.ph = AnimDance0+(m*5)%12, t*7+float64(m)
			if mem.hero >= 0 {
				mem.anim, mem.ph = AnimCheer, t*6+float64(m)
				mem.hop = 2 * math.Abs(math.Sin(t*5+float64(m)))
			}
		case t < partyActNight: // act 2: everybody runs left, and comes back in a line
			run := math.Max(sx-270*(t-partyActConga), -60) // they wait for their turn just off the picture
			line := -30 - off + partyCongaV*(t-9.0)
			if run > line {
				mem.x, mem.y = run, sy
				mem.anim, mem.ph, mem.dir = AnimRun, t*11+float64(m), DirLeft
			} else {
				mem.x = math.Min(line, endCW+60) // the head of the conga waits just off the right edge
				mem.y = 222 + 24*lane + math.Round(5*math.Sin(line/34+lane*1.7))
				mem.anim, mem.ph, mem.dir = AnimWalk, t*9+float64(m), DirRight
				mem.hop = 4 * math.Abs(math.Sin(t*9+float64(m)))
			}
		case t < partyActPhoto: // act 3: they are back, looking at the sky
			mem.x, mem.y = wx, wy
			mem.alpha = clampF((t-partyActNight-0.1)/1.0, 0, 1)
			mem.anim, mem.dir, mem.ph = AnimIdle, DirUp, t*1.5
			if t > 26.4 {
				mem.anim, mem.ph = AnimCheer, t*6+float64(m)
				mem.hop = 3 * math.Abs(math.Sin(t*6+float64(m)))
			}
		default: // act 4: the photo
			u := arrSmooth((t - partyActPhoto) / 2.4)
			mem.x, mem.y = lerp(wx, px, u), lerp(wy, py, u)
			if t < partyActPhoto+2.4 {
				mem.anim, mem.ph = AnimWalk, t*8+float64(m)
				mem.dir = DirRight
				if px < wx {
					mem.dir = DirLeft
				}
			} else {
				mem.anim, mem.ph = AnimCheer, t*5+float64(m)
				if t >= partyFlashAt {
					mem.anim, mem.ph = AnimIdle, 0
				}
			}
		}
		out = append(out, mem)
	}
	return out
}

// ---- the world -------------------------------------------------------------------------------------

// partyNight: 0 at dusk, 1 at night.
func partyNight(t float64) float64 { return arrSmooth((t - 12) / 8) }

func (g *Game) partyBackdrop(c *ebiten.Image, t float64) {
	n := partyNight(t)
	topD, topN := rgb(0x2a1a56), rgb(0x070a2e)
	botD, botN := rgb(0xe0703a), rgb(0x20186a)
	for y := 0.0; y < 150; y += 3 {
		k := y / 150
		col := mixc(mixc(topD, botD, k*k), mixc(topN, botN, k*k), n)
		fillRect(c, 0, y, endCW, 3, col)
	}
	if n < 0.9 { // the sun, sinking
		sy := 142 + 70*n
		filledEllipse(c, 300, sy, 54, 54, rgba(0xffd070, uint8(90*(1-n))))
		filledEllipse(c, 300, sy, 40, 40, rgb(0xf8b040))
	}
	if n > 0.3 { // the moon (a disc, no dots)
		a := uint8(255 * clampF((n-0.3)/0.4, 0, 1))
		filledEllipse(c, 410, 40, 15, 15, rgba(0xeae6d4, a))
		filledEllipse(c, 404, 36, 14, 14, rgba(0x20186a, a))
	}
	// the volcano: it throws confetti
	vc := mixc(rgb(0x2a1a3a), rgb(0x0c0a24), n)
	for y := 0; y < 60; y++ {
		half := 6 + float64(y)*1.7
		fillRect(c, 120-half, 96+float64(y), half*2, 1, vc)
	}
	fillRect(c, 108, 92, 24, 3, rgb(0xd04820))
	for i := 0; i < 16; i++ {
		u := math.Mod(t*0.8+float64(i)*0.173, 1)
		fillRect(c, math.Round(120+(hashF(i, 4, 9)-0.5)*70*u), math.Round(92-70*u+60*u*u), 2, 2, rgb(partyColors[i%len(partyColors)]))
	}
	jc := mixc(rgb(0x14301c), rgb(0x061410), n)
	for x := 0.0; x < endCW; x += 6 {
		h := 18 + 14*math.Sin(x/23) + 8*math.Sin(x/7+1)
		fillRect(c, x, 152-h, 6, h+2, jc)
	}
	for y := 150.0; y < endCH; y += 3 {
		fillRect(c, 0, y, endCW, 3, mixc(mixc(rgb(0x3a7a3a), rgb(0x1c4a2a), (y-150)/120), mixc(rgb(0x1c4a30), rgb(0x0c2418), (y-150)/120), n))
	}
	// the bunting between the palms
	for x := 0.0; x < endCW; x += 12 {
		sag := 38 + 12*math.Sin(math.Pi*x/endCW)
		col := rgb(partyColors[int(x/12)%len(partyColors)])
		for i := 0; i < 6; i++ {
			fillRect(c, x+float64(i)*0.5, sag+float64(i), 8-float64(i), 1, col)
		}
		fillRect(c, x, sag-1, 12, 1, rgb(0x6a5a40))
		if n > 0.2 && int(t*3+x/12)%3 != 0 { // little lights
			g.preGlow(c, x+4, sag+2, 12, color.RGBA{255, 220, 120, uint8(60 * n)})
		}
	}
	g.preProp(c, "palm", 0, 22, 150, 0.9, false, 1)
	g.preProp(c, "palm", 1, 458, 150, 0.9, false, 1)
	// the dinosaurs of the background
	g.preProp(c, "jp_brachio", 0, 62+2*math.Sin(t*1.3), 192, 1, false, 1)
	// the stage
	fillRect(c, 150, 186, 180, 8, rgb(0x6a4a2a))
	fillRect(c, 150, 186, 180, 2, rgb(0x9a7a4a))
	for q := 0; q < 9; q++ {
		on := (q+int(t*4))%2 == 0
		lc := rgb(partyColors[q%len(partyColors)])
		if on {
			lc = shade(lc, 0.5)
		}
		fillRect(c, 154+float64(q)*20, 190, 16, 2, lc)
	}
	g.preProp(c, "speaker", 0, 164, 186, 1, false, 1)
	g.preProp(c, "speaker", 0, 316, 186, 1, false, 1)
	g.partyBooth(c, t) // djcameo.go: the giant DJ booth with its parody DJs
	// the disco ball and its lights on the ground
	fillRect(c, 240, 0, 1, 24, rgb(0x8a8a9a))
	filledEllipse(c, 240, 30, 9, 9, rgb(0xc8d0e0))
	for i := -1; i <= 1; i++ {
		fillRect(c, 238+float64(i)*3, 24, 1, 12, rgb(0xf4f8ff))
	}
	for k := 0; k < 9; k++ {
		ang := t*0.8 + float64(k)*0.7
		col := partyColors[k%len(partyColors)]
		filledEllipse(c, 240+math.Cos(ang)*(110+30*math.Sin(float64(k))), 214+math.Sin(ang)*16, 9, 3, rgba(col, 60))
	}
	// the cake
	fillRect(c, 66, 208, 44, 4, rgb(0x8a5a30))
	fillRect(c, 70, 212, 3, 14, rgb(0x6a4220))
	fillRect(c, 103, 212, 3, 14, rgb(0x6a4220))
	fillRect(c, 72, 196, 32, 12, rgb(0xf4d8e0))
	fillRect(c, 72, 196, 32, 3, rgb(0xff6a8a))
	fillRect(c, 78, 186, 20, 10, rgb(0xf0e0b0))
	fillRect(c, 78, 186, 20, 3, rgb(0xe03a8a))
	for q := 0; q < 3; q++ {
		fillRect(c, 82+float64(q)*7, 180, 2, 6, rgb(0x3a9ad8))
		fillRect(c, 82+float64(q)*7, 177-float64(int(t*8+float64(q))%2), 2, 3, rgb(0xffd040))
	}
	recLayer("bg:party")
}

// partyRex draws the T-Rex of the party in his hat; it dances (act 1 and 2) and looks at the sky (act 3).
func (g *Game) partyRex(c *ebiten.Image, t float64) (head Vec) {
	sway := 2 * math.Sin(t*4.4)
	if t >= partyActNight && t < partyActPhoto {
		sway = 0.6 * math.Sin(t*1.2)
	}
	recLayer("actor:rex")
	x, y := 394+sway, 212-math.Abs(sway)*0.6
	g.preProp(c, "jp_trex", 0, x, y, 1, false, 1)
	head = Vec{x - 46, y - 68}
	preHat(c, head.X+4, head.Y-12, 0xf0c030)
	return head
}

func (g *Game) partyAnimals(c *ebiten.Image, t float64, first int) {
	if g.as == nil || len(g.as.animals) == 0 {
		return
	}
	kinds := []int{animCat, animDog, animPig}
	for i, kd := range kinds {
		var x, y float64
		fr := fWalk0 + int(t*8)%4
		flip := false
		switch {
		case t >= partyActConga && t < partyActNight+0.8:
			off := float64(first+i) * partyCongaGap
			x = -30 - off + partyCongaV*(t-9.0)
			y = 236
		case t >= partyActPhoto:
			x, y, fr = 24+float64(i)*22, 266, fStand // in front of the front row
			flip = true
		default:
			continue
		}
		im := g.as.animals[animalKey(kd, 0, fr)]
		if im == nil || x < -40 || x > endCW+40 {
			continue
		}
		recLayer("actor:animal")
		w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		if flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Translate(math.Round(x-w/2), math.Round(y-h+2-2*math.Abs(math.Sin(t*9+float64(i)))))
		fillShadow(c, x, y)
		c.DrawImage(im, op)
	}
}

func (g *Game) partyRaptors(c *ebiten.Image, t float64) {
	recLayer("actor:raptor")
	for i, x := range []float64{192, 288} {
		hop := 6 * math.Abs(math.Sin(t*7+float64(i)*2))
		if t >= partyActNight && t < partyActPhoto {
			hop = 0
		}
		y := 222 - hop
		g.preProp(c, "jp_raptor", i, x, y, 1, i == 1, 1)
		preHat(c, x-14+28*float64(i), y-34, []uint32{0xe03a8a, 0x3ad0a8}[i])
	}
}

func (g *Game) drawTrophy(c *ebiten.Image, x, y, t float64) {
	x, y = math.Round(x), math.Round(y)
	gold, hi := rgb(0xf0c030), rgb(0xfff0a0)
	fillRect(c, x-5, y, 10, 3, gold)
	fillRect(c, x-6, y-9, 12, 9, gold)
	fillRect(c, x-4, y-9, 2, 7, hi)
	fillRect(c, x-9, y-8, 3, 5, gold)
	fillRect(c, x+6, y-8, 3, 5, gold)
	fillRect(c, x-1, y+3, 3, 4, gold)
	fillRect(c, x-4, y+7, 9, 2, rgb(0x8a5a20))
	if int(t*6)%3 == 0 {
		fillRect(c, x+6, y-14, 1, 5, hi)
		fillRect(c, x+4, y-12, 5, 1, hi)
	}
}

// partyFireworks draws the bursts of act 3 (and a few at the end of act 2 and in the photo).
func (g *Game) partyFireworks(c *ebiten.Image, t float64) {
	for k := 0; k < 12; k++ {
		t0 := partyActNight + 0.8 + float64(k)*0.62
		if k >= 9 {
			t0 = 30.2 + float64(k-9)*0.4
		}
		u := t - t0
		if u < 0 || u > 1.9 {
			continue
		}
		cx := 70 + math.Mod(float64(k)*113, 340)
		cy := 44 + math.Mod(float64(k)*41, 56)
		col := partyColors[k%len(partyColors)]
		if u < 0.14 {
			g.preGlow(c, cx, cy, 90, color.RGBA{uint8(col >> 16), uint8(col >> 8), uint8(col), uint8(110 * (1 - u/0.14))})
		}
		n := 28
		for q := 0; q < n; q++ {
			ang := float64(q) / float64(n) * 2 * math.Pi
			sp := 58 + 30*hashF(q, k, 17)
			px := cx + math.Cos(ang)*sp*u*1.2
			py := cy + math.Sin(ang)*sp*u*1.2 + 24*u*u
			a := uint8(255 * math.Min(1, (1.9-u)/0.9))
			sz := 2.0
			if u < 1.0 {
				sz = 3
			}
			fillRect(c, math.Round(px), math.Round(py), sz, sz, rgba(col, a))
			if u > 0.3 {
				fillRect(c, math.Round(px-math.Cos(ang)*3), math.Round(py-math.Sin(ang)*3+1), 1, 1, rgba(0xffffff, a/2))
			}
		}
	}
}

func (g *Game) partyConfetti(c *ebiten.Image, t float64) {
	for k := 0; k < 80; k++ {
		x := hashF(k, 5, 41)*endCW + math.Sin(t*2+float64(k))*6
		y := math.Mod(hashF(k, 6, 41)*endCH+t*(26+hashF(k, 7, 41)*30), endCH+20) - 10
		fillRect(c, math.Round(x), math.Round(y), 2, 3, rgb(partyColors[k%len(partyColors)]))
	}
	for k := 0; k < 14; k++ { // balloons going up
		x := hashF(k, 8, 41)*endCW + 10*math.Sin(t*0.8+float64(k))
		y := endCH + 20 - math.Mod(t*(14+hashF(k, 9, 41)*8)+hashF(k, 10, 41)*320, endCH+60)
		col := rgb(partyColors[(k+2)%len(partyColors)])
		filledEllipse(c, x, y, 6, 7.5, col)
		fillRect(c, x-2, y-4, 2, 2, shade(col, 1.5))
		fillRect(c, x, y+7, 1, 16, rgba(0xc8c8d8, 150))
	}
}

// ---- drawing ---------------------------------------------------------------------------------------

func (g *Game) drawParty(dst *ebiten.Image, e *endState) {
	g.partyBuild(e)
	p := &e.x.party
	if e.canvas == nil {
		e.canvas = ebiten.NewImage(endCW, endCH)
	}
	c := e.canvas
	t := e.t
	two := g.scene2P()
	crowd := g.partyCrowd(p, t, two)
	g.partyBackdrop(c, t)
	// the people, from the back to the front; the back row (by the stage) stands behind the dinosaurs
	const backRow = 212.0
	order := make([]int, len(crowd))
	for i := range order {
		order[i] = i
	}
	for i := 1; i < len(order); i++ { // a tiny insertion sort by depth
		for j := i; j > 0 && crowd[order[j]].y < crowd[order[j-1]].y; j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
	cs0, cs1 := g.sceneHero(0, false), g.sceneHero(1, false)
	g.partyPeople(c, crowd, order, cs0, cs1, func(y float64) bool { return y < backRow })
	head := g.partyRex(c, t)
	g.partyRaptors(c, t)
	if (t >= partyActConga && t < partyActNight+0.8) || t >= partyActPhoto {
		g.partyAnimals(c, t, -4) // they lead the congas
	}
	g.partyPeople(c, crowd, order, cs0, cs1, func(y float64) bool { return y >= backRow })
	if t < partyActConga { // the trophies
		for _, m := range crowd {
			if m.hero >= 0 {
				g.drawTrophy(c, m.x+6, m.y-m.hop-58-3*math.Sin(t*4), t)
			}
		}
	}
	recLayer("front:fx")
	if t >= partyActNight-0.5 && t < partyActPhoto+5 {
		g.partyFireworks(c, t)
	}
	g.partyDrop(c, t)
	g.partyConfetti(c, t)
	if t > partyFlashAt-0.2 && p.photo == nil { // the picture, just before the flash
		p.photo = ebiten.NewImage(endCW, endCH)
		p.photo.DrawImage(c, nil)
	}
	if f := 1 - clampF((t-partyFlashAt)/0.5, 0, 1); t >= partyFlashAt && f > 0 {
		fillRect(c, 0, 0, endCW, endCH, rgba(0xffffff, uint8(255*f)))
	}
	if t < 0.8 {
		fillRect(c, 0, 0, endCW, endCH, rgba(0xfff4d0, uint8(255*(1-t/0.8))))
	}
	g.preBlit(dst, e, 0)
	g.partyTexts(dst, e, crowd, head)
}

// partyPeople draws the members of the crowd (in depth order) whose feet are where `in` says.
func (g *Game) partyPeople(c *ebiten.Image, crowd []pMember, order []int, cs0, cs1 *CharSet, in func(y float64) bool) {
	for _, i := range order {
		m := crowd[i]
		if m.cs == nil || m.alpha <= 0 || !in(m.y) {
			continue
		}
		tint := [3]float32{1, 1, 1}
		if m.hero == 1 {
			tint = sceneTint(1, cs0, cs1)
		}
		if m.hero >= 0 {
			recLayer("actor:hero")
		} else {
			recLayer("actor:guest")
		}
		g.sceneChar(c, m.cs, m.anim, m.dir, m.ph, m.x, m.y, 1, m.alpha, m.hop, 0, tint)
	}
}

// partyLines: the balloons of the party. who: hero 0, hero 1, "rex" or the name of a guest.
type partyLine struct {
	who  string
	a, b float64
	s    string
	two  bool
}

var partyLines = []partyLine{
	{"hero0", 1.4, 3.2, "¡Lo hicimos! ¡Todos los trofeos!", false},
	{"hero1", 3.3, 4.6, "¡Y yo he ayudado!", true},
	{"rex", 4.8, 6.6, "¿Ya me puedo comer la tarta?", false},
	{"shrek", 6.7, 7.9, "Yo me he comido el catering.", false},
	{"hero0", 10.4, 11.8, "¡Conga! ¡Conga!", false},
	{"sonic", 13.5, 14.9, "¡Yo voy delante! ¡Soy rápido!", false},
	{"hero0", 15.2, 16.6, "¡No corras tanto, Zonic!", false},
	{"hero0", 22.0, 23.4, "¡Oooooh!", false},
	{"rex", 24.4, 25.9, "¡Otra! ¡Otra! ¡Otra!", false},
	{"hero1", 26.2, 27.5, "¡Qué bonito!", true},
	{"hero0", 28.9, 29.9, "¡Todos juntos, rápido!", false},
	{"rex", 30.0, 30.9, "¡Digan RAWR!", false},
}

func partyCaption(t float64) string {
	switch {
	case t < partyActConga:
		return "¡Todos los trofeos! Hasta los dinosaurios llevan sombrero."
	case t < partyActNight:
		return "La conga más larga de la historia, con gato, perro y cerdo."
	case t < partyActPhoto:
		return "Fuegos artificiales. Dinosaurios y vecinos, encantados."
	}
	return "Foto de familia. Digan 'patata'... o 'RAWR'."
}

func (g *Game) partyTexts(dst *ebiten.Image, e *endState, crowd []pMember, head Vec) {
	t := e.t
	two := g.scene2P()
	if t >= 0.9 && t < 7.6 { // the banner
		col := rgb(partyColors[int(t*4)%len(partyColors)])
		g.drawTextC(dst, "¡FELIZ FIESTA!", 480, 108+math.Round(5*math.Sin(t*3)), col, 5)
	}
	if t >= 20.8 && t < 27.8 {
		for _, at := range []float64{21.9, 23.1, 24.3, 25.5, 26.7} {
			if t >= at && t < at+0.8 {
				w := []string{"¡BUM!", "¡PUM!", "¡PAF!", "¡BUUM!", "¡PIUM!"}[int(at*3)%5]
				g.sceneBang(dst, w, 150+math.Mod(at*97, 640), 150+math.Mod(at*31, 60), (t-at)/0.8, partyColors[int(at)%len(partyColors)], 4)
			}
		}
	}
	n := 0
	for _, l := range append(append([]partyLine{}, partyLines...), djLines...) {
		if (l.two && !two) || t < l.a || t >= l.b || n >= 2 {
			continue
		}
		var px, py float64
		ok := false
		if l.who == "rex" {
			px, py, ok = head.X*2, head.Y*2-40, true
		} else if l.who == "gabbo" {
			px, py, ok = 0, 0, true
			px, py = djSayPos(t)
		} else {
			for _, m := range crowd {
				if (l.who == "hero0" && m.hero == 0) || (l.who == "hero1" && m.hero == 1) || (m.hero < 0 && m.name == l.who) {
					px, py, ok = m.x*2, (m.y-m.hop-46)*2, m.alpha > 0.5
					break
				}
			}
		}
		if !ok || px < -40 || px > screenW+40 {
			continue
		}
		n++
		g.speechBubble(dst, px, py, l.s)
	}
	g.djTexts(dst, t)
	// the polaroid
	if p := e.x.party.photo; p != nil && t >= partyFlashAt+0.6 {
		u := easeOut((t - partyFlashAt - 0.6) / 1.0)
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x10061c, uint8(170*u)))
		x0, y0 := 220.0, 40.0-math.Round((1-u)*500)
		fillRect(dst, x0, y0, 520, 420, rgb(0xf6f0e4))
		fillRect(dst, x0, y0+418, 520, 2, rgb(0xc8c0b0))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x0+20, y0+16) // the picture at 1x: 480 x 270
		dst.DrawImage(p, op)
		g.drawTextC(dst, "Foto de familia", x0+260, y0+304, rgb(0x2a2230), 3)
		if t > partyFlashAt+2.4 {
			g.drawTextC(dst, "Nadie fue devorado*", x0+260, y0+346, rgb(0x2a2230), 2)
		}
		if t > partyFlashAt+3.6 {
			g.drawTextC(dst, "*Casi nadie.", x0+260, y0+378, rgb(0x8a1a14), 2)
		}
		return
	}
	g.endCaption(dst, partyCaption(t))
}

func (g *Game) partySounds(e *endState) {
	p := &e.x.party
	once := func(slot int, at float64, name string) {
		if e.t >= at && !p.done[slot] {
			p.done[slot] = true
			g.snd.Play(name)
		}
	}
	once(0, 0.2, "win")
	once(1, 21.6, "boom")
	once(2, 23.4, "spark")
	once(3, 25.2, "boom")
	once(4, 26.4, "rex_roar")
	once(5, partyFlashAt, "ring")
	once(6, 30.2, "spark")
	once(7, djDropAt, "boom")
}

// ---- the credits of the party ------------------------------------------------------------------------

// partyCreditsBG: the colourful background of the credits of the secret ending (falling confetti).
func (g *Game) partyCreditsBG(dst *ebiten.Image, e *endState) {
	for y := 0.0; y < screenH; y += 6 {
		fillRect(dst, 0, y, screenW, 6, mixc(rgb(0x2a1050), rgb(0x6a1a5a), y/screenH))
	}
	t := e.total
	for k := 0; k < 70; k++ {
		x := hashF(k, 11, 43)*screenW + math.Sin(t*1.5+float64(k))*12
		y := math.Mod(hashF(k, 12, 43)*(screenH+40)+t*(40+hashF(k, 13, 43)*50), screenH+40) - 20
		col := rgba(partyColors[k%len(partyColors)], 200)
		fillRect(dst, math.Round(x), math.Round(y), 5, 8, col)
	}
}

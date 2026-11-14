package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The story interludes: between parties 5 and 6 (a car), 10 and 11 (a boat), 15 and 16 (a balloon) and, before the space station, the rocket.
//
// The hero walks down the street furious, muttering, steam coming out of his head,
// while the places of the parties he has already finished pass by behind him as
// flashbacks (their real backgrounds, rendered by the game). At the end he gets
// into a car / a boat, the camera follows it and the next party begins.
// ---------------------------------------------------------------------------

const (
	interCar     = 1
	interBoat    = 2 // interlude_boat.go: the leaking boat and the shark (no street)
	interBalloon = 4 // interlude_balloon.go: the balloon that deflates (after party 15); interRocket is 3
)

type interState struct {
	kind    int
	next    int // the level that starts afterwards
	t       float64
	skipArm float64
	canvas  *ebiten.Image
	bgs     []*ebiten.Image // the flashbacks
	names   []string
	hero    *CharSet
	did     map[string]bool
	step    int
	music   bool     // the music of the interlude has been started
	guard   *CharSet // the station guard of the rocket interlude
	guardX  float64
	two     bool          // the two player version (interlude2p.go)
	heroes  [2]*CharSet   // both heroes, each with his knife
	rig     *ebiten.Image // the balloon with its basket, on its own canvas (interlude_balloon.go)
}

// the timeline, in seconds
const (
	itDecel0  = 5.8  // the street slows down
	itStop    = 7.8  // it has stopped: the vehicle is right there
	itBoard0  = 8.0  // he walks to it
	itOpen    = 9.0  // the door opens
	itIn      = 9.6  // he is inside
	itDepart  = 10.8 // it leaves, the camera follows
	itFadeOut = 13.6
	interLen  = 14.8
)

const (
	interGround = 432.0 // the floor line of the street, in screen pixels
	interScale  = 4.0   // the hero is drawn at 4x
	interHeroX  = 330.0
	interV      = 90.0 // street speed while he walks, px/s
)

// interlude places: the campaign levels already beaten
func interludePlaces(kind int) []int {
	switch kind {
	case interBoat, interBalloon:
		return nil // their own scenes: no memories
	case interRocket:
		s := len(campaignLevels) - 1 // the last five parties before the space station (train station, birthday, rink, 70s disco, pirate ship)
		return []int{s - 5, s - 4, s - 3, s - 2, s - 1}
	}
	return []int{0, 1, 2, 3, 4}
}

func (g *Game) startInterlude(kind, next int) {
	g.inter = &interState{kind: kind, next: next, did: map[string]bool{}}
	g.mode = ModeInterlude
	g.sceneReplay = false
	g.snd.StopMusic()
	g.bedQuiet, g.bedEmpty, g.bedNoTalk = false, false, false
	switch kind {
	case interBoat:
		g.markSeen("seen_interlude10")
	case interRocket:
		g.markSeen("seen_interlude20")
	case interBalloon:
		g.markSeen("seen_interlude15")
	default:
		g.markSeen("seen_interlude5")
	}
	g.inter.hero = g.heroWithKnife()
	if g.scene2P() {
		in := g.inter
		in.two = true
		in.heroes[0], in.heroes[1] = g.sceneHero(0, true), g.sceneHero(1, true)
		if in.heroes[0] != nil {
			in.hero = in.heroes[0]
		}
	}
}

// finishInterlude: the real thing goes on to the next party, a replay goes back.
func (g *Game) finishInterlude() {
	in := g.inter
	g.inter = nil
	g.snd.StopThump()
	g.snd.StopScene()
	if g.sceneReplay {
		g.loadLevel(0) // the flashbacks used the level slot
		g.leaveScene()
		return
	}
	if in != nil {
		g.startLevel(in.next)
	}
}

// afterLevel: what comes when a finished party is left behind: maybe a story interlude.
func (g *Game) afterLevel() {
	if g.levelIdx == specialJurassic { // the island has no "next party": back to the list
		g.loadLevel(0)
		g.mode = ModeLevelSelect
		g.menuSel = campaignShown() + 1 // the island's row (campaign_retired.go)
		return
	}
	next := campaignNext(g.levelIdx) // a retired party is skipped (campaign_retired.go)
	if kind, ok := g.interludeAfter(g.levelIdx); ok {
		g.startInterlude(kind, next)
		return
	}
	g.startLevel(next)
}

// interludeAfter: the interlude that follows campaign level i, if it has not been seen yet.
// The 5th, 10th and 15th go by the row of the party in the list (campaignSlotOf: the retired parties do not count).
func (g *Game) interludeAfter(i int) (int, bool) {
	s := campaignSlotOf(i)
	switch {
	case s == 4 && !g.seen("seen_interlude5"):
		return interCar, true
	case s == 9 && !g.seen("seen_interlude10"):
		return interBoat, true
	case s == 14 && !g.seen("seen_interlude15"):
		return interBalloon, true
	case i == len(campaignLevels)-2 && !g.seen("seen_interlude20"): // the party before the space station
		return interRocket, true
	}
	return 0, false
}

// loadFlashback prepares one of the memories (one per frame, while the screen is black).
func (g *Game) loadFlashback(in *interState) {
	places := interludePlaces(in.kind)
	i := len(in.bgs)
	if i >= len(places) {
		return
	}
	idx := places[i]
	if idx >= len(levelDefs) {
		idx = len(levelDefs) - 1
	}
	def := levelDefs[idx]
	in.bgs = append(in.bgs, g.buildFlashback(def, int64(idx)*7919+13))
	in.names = append(in.names, def.Title)
}

// buildFlashback: the party as it was left. The real background, its objects, and what the hero
// did there: bodies, pools of blood, police tape and an abandoned police car; all of it graded
// like an old memory (cold, washed out).
func (g *Game) buildFlashback(def *LevelDef, seed int64) *ebiten.Image {
	save := g.lvl
	g.lvl = buildLevel(def)
	seaSkyStill = true // a still picture: the sun's reflection and the ship baked in (seasky.go)
	g.renderBG()
	seaSkyStill = false
	bg := g.lvl.bg
	g.lvl = save
	m := ebiten.NewImage(screenW, screenH)
	m.DrawImage(bg, nil)
	// the objects of the place
	props := append([]PropInst(nil), def.Props...)
	sortPropsByY(props)
	for _, pr := range props {
		imgs := g.as.props[pr.Kind]
		if len(imgs) == 0 {
			continue
		}
		pi := imgs[pr.Var%len(imgs)]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(pr.X-pi.ox), math.Round(pr.Y-pi.oy))
		m.DrawImage(pi.img, op)
	}
	rng := rand.New(rand.NewSource(seed))
	// the spots where things happened: inside the walkable areas
	var spots []Vec
	for n := 0; n < 60 && len(spots) < 14; n++ {
		var p Vec
		if len(def.Roam) > 0 {
			r := def.Roam[rng.Intn(len(def.Roam))]
			p = Vec{r.X + 12 + rng.Float64()*math.Max(1, r.W-24), r.Y + 12 + rng.Float64()*math.Max(1, r.H-24)}
		} else {
			p = Vec{80 + rng.Float64()*800, 80 + rng.Float64()*380}
		}
		spots = append(spots, p)
	}
	// pools and splashes of blood
	for i, p := range spots {
		r := 7 + rng.Float64()*10
		filledEllipse(m, p.X, p.Y+3, r, r*0.55, rgba(0x6a0c14, 170))
		filledEllipse(m, p.X+2, p.Y+2, r*0.6, r*0.35, rgba(0x8a1420, 190))
		for q := 0; q < 6; q++ {
			a := rng.Float64() * 2 * math.Pi
			d := r * (0.8 + rng.Float64()*1.3)
			fillRect(m, math.Round(p.X+math.Cos(a)*d), math.Round(p.Y+3+math.Sin(a)*d*0.5), 2, 2, rgba(0x7a1018, 200))
		}
		_ = i
	}
	// the bodies
	for i := 0; i < len(spots) && i < 9; i++ {
		spec := genericSpec(rng)
		cs := newCharSet(&spec)
		im := cs.lying(false)
		if im == nil {
			continue
		}
		w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		if rng.Intn(2) == 0 {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Translate(math.Round(spots[i].X-w/2), math.Round(spots[i].Y-h/2))
		m.DrawImage(im, op)
	}
	// police tape between two corners of the place, and the abandoned police car
	tape := func(a, b Vec) {
		d := dist(a, b)
		for s := 0.0; s < d; s += 5 {
			u := s / d
			col := rgb(0xf0d020)
			if int(s/5)%2 == 0 {
				col = rgb(0x14141a)
			}
			fillRect(m, math.Round(a.X+(b.X-a.X)*u), math.Round(a.Y+(b.Y-a.Y)*u), 4, 3, col)
		}
	}
	for i := 0; i+1 < len(spots) && i < 6; i += 2 {
		tape(spots[i], spots[i+1])
	}
	if len(spots) > 0 { // the police car of the game (vehart.go), its light bar still on (blue frame)
		pi := emergencyImg(0, unitPolice, 2)
		p := spots[len(spots)-1]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(p.X-pi.ox), math.Round(p.Y-pi.oy))
		m.DrawImage(pi.img, op)
	}
	// the grade of a memory: washed out, cold, darker
	pix := make([]byte, 4*screenW*screenH)
	m.ReadPixels(pix)
	for i := 0; i < len(pix); i += 4 {
		r, gg, b := float64(pix[i]), float64(pix[i+1]), float64(pix[i+2])
		l := 0.3*r + 0.59*gg + 0.11*b
		nr := l*0.74 + r*0.26
		ng := l*0.86 + gg*0.14
		nb := l*1.05 + b*0.2
		pix[i], pix[i+1], pix[i+2] = clampU8(nr*0.82), clampU8(ng*0.88), clampU8(nb*0.98)
	}
	out := ebiten.NewImage(screenW, screenH)
	out.WritePixels(pix)
	m.Deallocate()
	return out
}

func clampU8(v float64) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

func sortPropsByY(p []PropInst) {
	for i := 1; i < len(p); i++ {
		for j := i; j > 0 && p[j].Y < p[j-1].Y; j-- {
			p[j], p[j-1] = p[j-1], p[j]
		}
	}
}

func (g *Game) updateInterlude() {
	in := g.inter
	if in == nil {
		g.leaveScene()
		return
	}
	if len(in.bgs) < len(interludePlaces(in.kind)) { // still preparing the memories
		g.loadFlashback(in)
		return
	}
	const dt = 1.0 / 60
	in.t += dt
	g.time = in.t
	if in.skipArm > 0 {
		in.skipArm -= dt
	}
	if actBack.just() || actConfirm.just() || ebitenClick() {
		if in.skipArm > 0 {
			g.finishInterlude()
			return
		}
		in.skipArm = 2.2
	}
	switch in.kind {
	case interRocket: // its own timeline (interlude_rocket.go)
		g.updateRocket(in)
		return
	case interBoat:
		g.updateBoat(in)
		return
	case interBalloon:
		g.updateBalloon(in)
		return
	}
	once := func(key string, at float64, name string) {
		if in.t >= at && !in.did[key] {
			in.did[key] = true
			g.snd.Play(name)
		}
	}
	// the music: a cartoon march (or the song of the intros folder) and, in step with it, a sneaky minor version
	// of the same tune that takes over while a memory is on screen
	if !in.music {
		in.music = true
		g.sceneMusic(sceneInterA, ambInterA)
		if !g.snd.SceneIsSong("base") {
			g.snd.StartScene("memory", "memory:"+ambMemA, ambientBuilder(ambMemA), "", true) // same tempo and bar as the base
		}
	}
	g.snd.SetSceneVolume("base", interBaseVol(in.t, len(in.bgs)))
	g.snd.SetSceneVolume("memory", interMemVol(in.t, len(in.bgs)))
	// stomps: one with every step while he is walking
	if in.t < itBoard0 {
		if st := int(g.interPhase(in) * 2 / float64(animFrames[AnimWalk])); st != in.step && in.t > 1 {
			in.step = st
			g.snd.Play("thud")
		}
	}
	once("open", itOpen, "alert")
	once("in", itIn, "thud")
	once("go", itDepart, "boom")
	if in.t >= interLen {
		g.finishInterlude()
	}
}

// interScroll: how far the street has moved by time t.
func interScroll(t float64) float64 {
	const dec = itStop - itDecel0
	base := interV*itDecel0 + interV*dec/2
	switch {
	case t < itDecel0:
		return interV * t
	case t < itStop:
		u := t - itDecel0
		return interV*itDecel0 + interV*u - interV*u*u/(2*dec)
	case t < itDepart:
		return base
	}
	u := t - itDepart
	const a = 150.0
	if u < 3 {
		return base + 0.5*a*u*u
	}
	return base + 0.5*a*9 + a*3*(u-3)
}

// the vehicle waits at this world position; at the stop it is at stopX on the screen
const interStopX = 520.0

func interVehicleX(t float64) float64 {
	base := interScroll(itStop)
	x := interStopX + (base - interScroll(t)) // world position minus scroll
	if t > itDepart {                         // it pulls forward a little while the street rushes by
		x = interStopX + 70*introEase((t-itDepart)/2.5)
	}
	return x
}

func (g *Game) interPhase(in *interState) float64 {
	sp := 30.0
	return interScroll(in.t) / interScale * stridePhase(sp) / sp
}

func (g *Game) drawInterlude(dst *ebiten.Image) {
	in := g.inter
	if in == nil {
		return
	}
	dst.Fill(rgb(0x05060c))
	if len(in.bgs) < len(interludePlaces(in.kind)) {
		g.drawTextC(dst, "...", 480, 250, rgb(0x8a86a0), 2)
		return
	}
	switch in.kind {
	case interRocket:
		g.drawRocketScene(dst, in)
		return
	case interBoat:
		g.drawBoatScene(dst, in)
		return
	case interBalloon:
		g.drawBalloonScene(dst, in)
		return
	}
	if in.canvas == nil {
		in.canvas = ebiten.NewImage(screenW, screenH)
	}
	c := in.canvas
	t := in.t
	c.Fill(rgb(0x070a1c))
	g.interPanel(c, in)
	g.interStreet(c, in)
	if in.two {
		g.interTwoStreetGags(c, in)
	}
	g.interVehicleAndHero(c, in)
	// camera: a little shake with every stomp, a slow zoom when he leaves
	ox, oy := 0.0, 0.0
	z := 1.0
	if t < itBoard0 && t > 1 {
		k := math.Exp(-math.Mod(g.interPhase(in)*2/float64(animFrames[AnimWalk]), 1) * 7)
		oy = k * 3
		ox = math.Sin(t*50) * k
	}
	if t > itIn {
		z = 1 + 0.06*introEase((t-itIn)/3)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-480, -300)
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(480+ox, 300+oy)
	dst.DrawImage(c, op)
	g.interCaption(dst, in)
	if t < 1 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-t))))
	}
	if t > itFadeOut {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*math.Min(1, (t-itFadeOut)/1.4))))
	}
	g.interSkipHint(dst, in)
}

// interSkipHint: the little line at the bottom that says how to skip (twice, with the usual keys).
func (g *Game) interSkipHint(dst *ebiten.Image, in *interState) {
	hint := skipTwiceHint()
	col := rgba(0x8a86a0, 170)
	if in.skipArm > 0 {
		hint, col = "Otra vez para saltar", rgb(0xffe060)
	}
	g.drawTextC(dst, hint, 480, 524, col, 1)
}

// ---- the present and the memories ------------------------------------------------------------

const (
	interSlot   = 1.3  // seconds per memory
	interMem    = 0.62 // the part of each slot that is the memory; the rest is the present
	interPanelH = 336.0
)

// interSky: the present, the night above the street: sky, moon and the city going by.
func (g *Game) interSky(c *ebiten.Image, in *interState) {
	W := interScroll(in.t)
	for y := 0.0; y < interPanelH; y += 4 {
		fillRect(c, 0, y, screenW, 4, mixc(rgb(0x070a1c), rgb(0x1c2a58), y/interPanelH))
	}
	filledEllipse(c, 800, 90, 34, 34, rgb(0xeae6d4)) // moon
	filledEllipse(c, 788, 82, 32, 32, rgb(0x14244c))
	off := W * 0.3
	for k := -1; k < 40; k++ { // far skyline
		bx := float64(k)*52 - math.Mod(off, 52)
		idx := int(math.Floor(off/52)) + k + 1000
		h := 60 + jpHash(idx, 3, 101)*150
		fillRect(c, bx, interPanelH-h, 48, h, rgb(0x0c1226))
		for wy := interPanelH - h + 8; wy < interPanelH-12; wy += 14 {
			for wx := bx + 6; wx < bx+42; wx += 12 {
				if jpHash(int(wx)+idx, int(wy), 103) > 0.9 {
					fillRect(c, wx, wy, 4, 6, rgb(0xe8c860))
				}
			}
		}
	}
	for k := -1; k < 14; k++ { // nearer houses
		hx := float64(k)*150 - math.Mod(W*0.6, 150)
		idx := int(math.Floor(W*0.6/150)) + k + 500
		hh := 110 + jpHash(idx, 9, 105)*40
		col := []uint32{0x2a2438, 0x2a3446, 0x38283a, 0x2a3a32}[((idx%4)+4)%4]
		fillRect(c, hx, interPanelH-hh, 124, hh, rgb(col))
		fillRect(c, hx-6, interPanelH-hh-14, 136, 16, shade(rgb(col), 0.7))
		for r := 0; r < 2; r++ {
			for q := 0; q < 3; q++ {
				wc := rgb(0x161a2c)
				if jpHash(idx*3+q, r, 107) > 0.88 {
					wc = rgb(0x5a4a22)
				}
				fillRect(c, hx+14+float64(q)*34, interPanelH-hh+20+float64(r)*44, 20, 28, wc)
			}
		}
	}
	// the dark band that joins the sky and the street
	for y := 300.0; y < interPanelH; y += 3 {
		fillRect(c, 0, y, screenW, 3, rgba(0x070a1c, uint8(255*math.Min(1, (y-300)/40))))
	}
}

// interMemory: which memory is on screen at time t (of n), how far into it (u) and how opaque it is (a).
func interMemory(t float64, n int) (cur int, u, a float64, ok bool) {
	k := (t - 1) / interSlot
	if k < 0 || int(k) >= n {
		return
	}
	cur = int(k)
	u = k - float64(cur)
	if u >= interMem+0.1 {
		return
	}
	a = math.Min(1, u/0.07) * (1 - math.Max(0, math.Min(1, (u-interMem)/0.1)))
	return cur, u, a, true
}

// interFade: the music comes in during the first seconds and goes away as the vehicle leaves.
func interFade(t float64) float64 {
	v := math.Min(1, t/2.5)
	if t > itBoard0 {
		v *= 1 - introEase((t-itBoard0)/(itFadeOut-itBoard0))
	}
	return v
}

// interBaseVol: the cartoon march of the street. In a memory the music cuts to the sneaky version (quick cross-fade).
func interBaseVol(t float64, n int) float64 {
	v := 0.55
	if _, _, a, ok := interMemory(t, n); ok {
		v *= 1 - a
	}
	return v * interFade(t)
}

// interMemVol: the tense, funny version of the same tune that takes over in the memories: silent in the street.
func interMemVol(t float64, n int) float64 {
	if _, _, a, ok := interMemory(t, n); ok {
		return 0.6 * a * interFade(t)
	}
	return 0
}

// interPanel alternates the present (the night street) with the memories of the parties he has beaten.
func (g *Game) interPanel(c *ebiten.Image, in *interState) {
	t := in.t
	g.interSky(c, in)
	n := len(in.bgs)
	cur, u, a, ok := interMemory(t, n)
	if !ok {
		return
	}
	breath := (0.5 - 0.5*math.Cos(2*math.Pi*t/6)) * a // a slow breath of the memory: no drum here
	uu := math.Min(1, u/interMem)
	z := 1.02 + 0.1*uu + 0.012*breath
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-480, -270)
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(480-(uu-0.5)*28+math.Sin(t*41)*0.8, 238+math.Cos(t*37)*0.8)
	op.ColorScale.ScaleAlpha(float32(a))
	sub := c.SubImage(image.Rect(0, 0, screenW, int(interPanelH))).(*ebiten.Image)
	sub.DrawImage(in.bgs[cur], op)
	// the look of a memory: soft dark edges and a light leak (no grain: no loose white dots)
	for i := 0.0; i < 90; i += 3 { // blurred borders
		al := uint8(230 * a * math.Pow(1-i/90, 2))
		fillRect(sub, i, 0, 3, interPanelH, rgba(0x05060f, al))
		fillRect(sub, screenW-3-i, 0, 3, interPanelH, rgba(0x05060f, al))
		fillRect(sub, 0, i*0.8, screenW, 3, rgba(0x05060f, al))
	}
	vo := &ebiten.DrawImageOptions{}
	vo.GeoM.Scale(1, interPanelH/screenH)
	vo.ColorScale.ScaleAlpha(float32(a))
	sub.DrawImage(g.as.vignette, vo)
	if u < 0.12 { // the flash that opens the memory
		fillRect(sub, 0, 0, screenW, interPanelH, rgba(0xdce8ff, uint8(200*(1-u/0.12))))
	}
	if u > interMem-0.03 && u < interMem+0.08 { // and the one that closes it
		fillRect(sub, 0, 0, screenW, interPanelH, rgba(0xdce8ff, uint8(160*(1-math.Abs(u-interMem-0.025)/0.055))))
	}
	if breath > 0.02 { // the lights of the party, a faint tint that swells and fades slowly
		fillRect(sub, 0, 0, screenW, interPanelH, rgba(partyColors[int(t/6)%len(partyColors)], uint8(34*breath)))
	}
	lk := math.Mod(t*0.7, 1) // a light leak crossing the picture
	fillRect(sub, lk*screenW-60, 0, 90, interPanelH, rgba(0xffd8a0, uint8(26*a)))
	// the film strip and the name of the place
	fillRect(c, 0, 0, screenW, 16, rgba(0x040408, uint8(255*a)))
	fillRect(c, 0, interPanelH-18, screenW, 18, rgba(0x040408, uint8(255*a)))
	for x := -math.Mod(interScroll(t)*0.3, 28); x < screenW; x += 28 {
		fillRect(c, x, 4, 12, 8, rgba(0x20202c, uint8(255*a)))
		fillRect(c, x, interPanelH-14, 12, 8, rgba(0x20202c, uint8(255*a)))
	}
	g.drawText(c, in.names[cur], 24, 268, rgba(0xf4f0ff, uint8(255*a)), 2)
	g.drawText(c, fmt.Sprintf("RECUERDO %d/%d", cur+1, n), 24, 296, rgba(0xf0e060, uint8(230*a)), 1.2)
}

// ---- the street / the pier -----------------------------------------------------------------------

func (g *Game) interStreet(c *ebiten.Image, in *interState) {
	W := interScroll(in.t)
	fillRect(c, 0, interPanelH, screenW, interGround-interPanelH, rgb(0x0a0e24))
	// far trees and lamps (slower than the road)
	for k := -1; k < 8; k++ {
		wx := float64(k)*190 - math.Mod(W*0.55, 190)
		h := 70 + jpHash(k+int(W*0.55/190), 5, 81)*50
		filledEllipse(c, wx, interGround-30-h/2, 34, h/2, rgb(0x0c1a1c))
		fillRect(c, wx-3, interGround-40, 6, 40, rgb(0x120e12))
	}
	for k := -1; k < 5; k++ {
		wx := float64(k)*260 - math.Mod(W, 260) + 90
		fillRect(c, wx, interGround-150, 5, 150, rgb(0x2a2a36))
		fillRect(c, wx-26, interGround-154, 32, 6, rgb(0x2a2a36))
		g.glowAt(c, Vec{wx - 14, interGround - 144}, 90, color.RGBA{255, 214, 120, 70})
		fillRect(c, wx-24, interGround-150, 18, 3, rgb(0xfff0b0))
	}
	// pavement, kerb and road
	fillRect(c, 0, interGround, screenW, 36, rgb(0x4a4658))
	fillRect(c, 0, interGround, screenW, 3, rgb(0x6a6680))
	fillRect(c, 0, interGround+36, screenW, screenH-interGround-36, rgb(0x1e1e28))
	for x := -math.Mod(W, 120); x < screenW; x += 120 {
		fillRect(c, x, interGround+42, 64, 4, rgb(0xb8aa60))
		fillRect(c, x+30, interGround+2, 2, 34, rgb(0x5a566c)) // joints of the paving
	}
	// the litter of the parties he has left behind
	for k := 0; k < 30; k++ {
		wx := float64(k)*173 + jpHash(k, 9, 83)*60 - W
		if wx < -20 || wx > screenW+20 {
			continue
		}
		switch k % 3 {
		case 0:
			fillRect(c, wx, interGround+20, 9, 12, rgb(0xd8503a))
		case 1:
			filledEllipse(c, wx, interGround+26, 9, 6, rgb(0xe03a8a))
		default:
			fillRect(c, wx, interGround+18, 4, 14, rgb(0x3aa050))
		}
	}
}

// ---- hero and vehicle ----------------------------------------------------------------------

func (g *Game) interHeroPos(in *interState) (x, y float64) {
	t := in.t
	x, y = interHeroX, interGround+12
	if t > itBoard0 {
		to := interStopX + 100 // the door of the car
		u := introEase((t - itBoard0) / (itOpen - itBoard0 + 0.4))
		x = interHeroX + (to-interHeroX)*u
	}
	return
}

func (g *Game) interVehicleAndHero(c *ebiten.Image, in *interState) {
	t := in.t
	vx := interVehicleX(t)
	hx, hy := g.interHeroPos(in)
	drawHero := func(alpha float64) {
		if alpha <= 0 {
			return
		}
		cs := in.hero
		if cs == nil {
			cs = g.as.playerCS
		}
		if cs == nil {
			return
		}
		anim, ph, dir := AnimWalk, g.interPhase(in), DirRight
		if t > itBoard0 {
			ph = t * 7
		}
		if t > itIn {
			anim, ph = AnimIdle, 0
		}
		img, flip := cs.img(anim, dir, ph)
		if img == nil {
			return
		}
		bob := 0.0
		if t < itBoard0 { // furious stomping: he bobs and leans forward
			bob = -math.Abs(math.Sin(ph*math.Pi*2/float64(animFrames[AnimWalk]))) * 5
		}
		fillShadow(c, hx, hy)
		op := &ebiten.DrawImageOptions{}
		if flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(charW, 0)
		}
		op.GeoM.Translate(-charCX, -charBase)
		op.GeoM.Scale(interScale, interScale)
		if t < itBoard0 {
			op.GeoM.Rotate(0.07) // leaning into it
		}
		op.GeoM.Translate(math.Round(hx), math.Round(hy+bob))
		op.ColorScale.ScaleAlpha(float32(alpha))
		c.DrawImage(img, op)
		g.interAnger(c, hx, hy+bob, t)
	}
	drawBoth := func(alpha float64) {
		drawHero(alpha)
		if in.two {
			a2 := alpha
			if t > itOpen+0.1 { // the red one is in the car before the blue one
				a2 = 1 - introEase((t-itOpen-0.1)/0.5)
			}
			g.interDrawHero2(c, in, a2)
		}
	}
	alpha := 1.0
	if t > itOpen+0.4 {
		alpha = 1 - introEase((t-itOpen-0.4)/0.7)
	}
	// the car goes first (the hero walks up to its door on the near side)
	g.interCar(c, in, vx, func() { drawBoth(alpha) })
}

// interAnger: the steam and the frown.
func (g *Game) interAnger(c *ebiten.Image, x, y, t float64) {
	if t > itBoard0+0.8 {
		return
	}
	top := y - 31*interScale
	for i := int(t*6) - 7; i <= int(t*6); i++ {
		age := t - float64(i)/6
		if age < 0 || age > 1.2 {
			continue
		}
		side := -1.0
		if i%2 == 0 {
			side = 1
		}
		px := x + side*(14+jpHash(i, 2, 97)*8) + math.Sin(age*5+float64(i))*4
		py := top - 4 - age*55 // the steam rises from the top of his head, never over the face
		r := 6 + age*9
		filledEllipse(c, px, py, r, r*0.8, rgba(0xdce0ee, uint8(170*(1-age/1.2))))
	}
	// the frown: two dark brows slanting down towards the nose (he faces right); no marks on the face
	fillRect(c, math.Round(x+10), math.Round(top-2), 6, 3, rgb(0x10101a))
	fillRect(c, math.Round(x+15), math.Round(top), 6, 3, rgb(0x10101a))
}

// ---- the car ---------------------------------------------------------------------------------

func (g *Game) interCar(c *ebiten.Image, in *interState, x float64, hero func()) {
	t := in.t
	y := interGround + 46 // the wheels touch the road
	if x < -300 || x > screenW+80 {
		hero()
		return
	}
	// a little engine shake while it idles
	if t >= itStop && t < itDepart {
		y += math.Round(math.Sin(t*38) * 1)
	}
	if in.two && t > itOpen+0.3 && t < itDepart { // the two of them fight for the wheel inside: the car rocks
		y += math.Round(math.Sin(t*31) * 2.5)
		x += math.Round(math.Cos(t*23) * 2)
	}
	// the hero stands in front of the car, on the pavement: drawn before the near door
	hero()
	// the car is the game's own car model (vehart.go), with its driver's door and turning wheels (scenecar.go)
	o := 0.0 // the near door, which opens for him
	if t > itOpen {
		o = introEase((t - itOpen) / 0.6)
	}
	if t > itIn+0.6 {
		o = 1 - introEase((t-itIn-0.6)/0.5)
	}
	spin := -1
	if t >= itDepart {
		spin = int(t*14/(2*math.Pi)*sceneCarSpinFr) % sceneCarSpinFr
	}
	drawSceneCar(c, x, y, o, spin)
	// after the door closes, you can see his head in the driver's window
	if t > itIn+1 && in.two {
		g.interTwoCarHeads(c, in, x, y)
	} else if t > itIn+1 {
		filledEllipse(c, x+110, y-65, 7, 8, rgb(0xe0b088))
		fillRect(c, x+104, y-74, 13, 4, rgb(0x2a1a14))
	}
	// exhaust
	if t >= itStop {
		for i := 0; i < 7; i++ {
			age := math.Mod(t*1.6+float64(i)/7, 1)
			filledEllipse(c, x-4-age*60, y-12-age*18, 4+age*8, 4+age*7, rgba(0xc8c8d4, uint8(160*(1-age))))
		}
	}
}

// ---- captions ---------------------------------------------------------------------------------

var interMutters = []string{"Otra fiesta más...", "Ni un día de descanso.", "¿Nadie conoce el botón de silencio?", "Me llaman aguafiestas. Con razón."}

func (g *Game) interCaption(dst *ebiten.Image, in *interState) {
	t := in.t
	var line string
	switch {
	case t < 2:
		line = ""
	case t < itDecel0:
		line = "Cinco fiestas menos. Cada una, un poco peor que la anterior."
	case t < itBoard0+1:
		line = "La siguiente fiesta queda lejos. Toca coche."
	case t < itDepart:
		line = "Que nadie lo toque, que va con mala cara."
	case t < itFadeOut-1:
		line = "Rumbo a la siguiente fiesta."
	}
	if line != "" { // under the street: it never covers the hero or the vehicle
		w := g.textW(tr(line), 1.5) + 40
		fillRect(dst, 480-w/2, 488, w, 32, rgba(0x000000, 170))
		g.drawTextC(dst, line, 480, 495, rgb(0xf4f0ff), 1.5)
	}
	if in.two { // each in his own voice
		g.interTwoSay(dst, in)
		g.interTwoBoardSay(dst, in)
		return
	}
	// he mutters
	if t > 2 && t < itBoard0 {
		const per = 2.9
		k := int((t - 2) / per)
		if u := t - 2 - float64(k)*per; u < 2.3 {
			lines := interMutters
			hx, hy := g.interHeroPos(in)
			g.speechBubble(dst, hx+10, hy-34*interScale-4, lines[k%len(lines)])
		}
	}
}

package main

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// One-lane roads painted in the editor (Muros > Carretera). They are drawn
// into the background and the traffic that runs along them goes in the
// direction chosen when they were painted: right, left, up or down.
// ---------------------------------------------------------------------------

// drawRoads paints the asphalt, the edge lines, the dashed centre and the direction arrows.
func drawRoads(bg *ebiten.Image, def *LevelDef) {
	asphalt := rgb(0x33333e)
	edge := rgb(0x8a8a98)
	dash := rgb(0xe0d8a0)
	arrow := rgb(0x70708a)
	for _, s := range def.Roads {
		r := s.R
		fillRect(bg, r.X, r.Y, r.W, r.H, asphalt)
		// speckle so it does not look flat
		for i := 0; i < int(r.W*r.H/90); i++ {
			px := r.X + math.Mod(float64(i)*37.1, r.W)
			py := r.Y + math.Mod(float64(i)*91.7+float64(i/7)*13, r.H)
			fillRect(bg, px, py, 1, 1, rgb(0x3d3d49))
		}
		horiz := s.horizontal()
		if horiz {
			fillRect(bg, r.X, r.Y+1, r.W, 1, edge)
			fillRect(bg, r.X, r.Y+r.H-2, r.W, 1, edge)
			for x := r.X + 4; x < r.X+r.W-10; x += 22 {
				fillRect(bg, x, r.Y+r.H/2, 11, 1, dash)
			}
		} else {
			fillRect(bg, r.X+1, r.Y, 1, r.H, edge)
			fillRect(bg, r.X+r.W-2, r.Y, 1, r.H, edge)
			for y := r.Y + 4; y < r.Y+r.H-10; y += 22 {
				fillRect(bg, r.X+r.W/2, y, 1, 11, dash)
			}
		}
		// arrows every ~110px
		length := r.W
		if !horiz {
			length = r.H
		}
		for t := 50.0; t < length-30; t += 110 {
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			if horiz {
				cx = r.X + t
			} else {
				cy = r.Y + t
			}
			for f := -8; f < 4; f++ {
				for sd := -1; sd <= 0; sd++ {
					roadPx(bg, s.Dir, cx, cy, float64(f), float64(sd), arrow)
				}
			}
			for k := 0; k < 5; k++ {
				for sd := -(4 - k); sd <= 4-k; sd++ {
					roadPx(bg, s.Dir, cx, cy, float64(4+k), float64(sd), arrow)
				}
			}
		}
	}
}

// roadPx puts one pixel of an arrow drawn in a frame where "forward" is the driving direction.
func roadPx(bg *ebiten.Image, dir int, cx, cy, f, s float64, c color.RGBA) {
	switch dir {
	case roadRight:
		fillRect(bg, cx+f, cy+s, 1, 1, c)
	case roadLeft:
		fillRect(bg, cx-f, cy+s, 1, 1, c)
	case roadDown:
		fillRect(bg, cx+s, cy+f, 1, 1, c)
	default:
		fillRect(bg, cx+s, cy-f, 1, 1, c)
	}
}

type RoadCar struct {
	seg    int
	pos    float64 // along the road (x for horizontal roads, y for vertical)
	lat    float64 // the lane's centre across the road
	speed  float64
	v0     float64
	name   string
	col    int
	length float64
	dead   bool
}

func (g *Game) updateRoads(dt float64) {
	def := g.lvl.Def
	if len(def.Roads) == 0 || g.mode != ModePlay {
		return
	}
	for len(g.roadT) < len(def.Roads) {
		g.roadT = append(g.roadT, 0.5+g.rng.Float64()*3)
	}
	for i, s := range def.Roads {
		g.roadT[i] -= dt
		if g.roadT[i] > 0 {
			continue
		}
		g.roadT[i] = 2.5 + g.rng.Float64()*4.5
		start, lat := roadStart(s)
		free := true
		for _, c := range g.roadCars {
			if c.seg == i && math.Abs(c.pos-start) < 130 {
				free = false
			}
		}
		if !free {
			g.roadT[i] = 0.6
			continue
		}
		name := "car"
		switch r := g.rng.Float64(); {
		case r > 0.9 && propKinds["taxi"] != nil:
			name = "taxi"
		case r > 0.8 && propKinds["van"] != nil:
			name = "van"
		case r > 0.68 && propKinds["motorbike"] != nil:
			name = "motorbike"
		}
		if def.RoadCar == "" {
			name = g.roadSpecialPick(name)
		}
		if def.RoadCar != "" && propKinds[def.RoadCar] != nil {
			name = def.RoadCar // a venue with its own vehicles (the jeeps of the island)
		}
		length := 36.0
		if k := propKinds[name]; k != nil {
			length = float64(k.W)
		}
		switch name {
		case "tank":
			length = 84
		case "tow":
			length = 58
		case "horse":
			length = 50
		}
		if !s.horizontal() {
			if d := vehViewDefs[name]; d != nil {
				length = float64(d.h) // seen head on a vehicle is only as long as it is tall on screen
			}
		}
		g.roadCars = append(g.roadCars, &RoadCar{seg: i, pos: start, lat: lat, v0: 55 + g.rng.Float64()*55, name: name, col: g.rng.Intn(4), length: length})
	}
	out := g.roadCars[:0]
	for _, c := range g.roadCars {
		s := def.Roads[c.seg]
		dir := roadSign(s)
		c.speed = c.v0
		// never drive through the car in front
		for _, o := range g.roadCars {
			if o == c || o.seg != c.seg || o.dead {
				continue
			}
			gap := (o.pos-c.pos)*dir - (o.length+c.length)/2
			if gap > -2 && gap < 70 {
				if gap < 12 {
					c.speed = 0
				} else {
					c.speed = math.Min(c.speed, math.Max(o.speed, 0)*0.95)
				}
			}
		}
		c.pos += c.speed * dt * dir
		g.roadRunOver(c, s)
		if roadAhead(s, c.pos) > 90 {
			continue
		}
		out = append(out, c)
	}
	g.roadCars = out
}

func roadSign(s RoadSeg) float64 {
	if s.Dir == roadRight || s.Dir == roadDown {
		return 1
	}
	return -1
}

func roadStart(s RoadSeg) (start, lat float64) {
	r := s.R
	switch s.Dir {
	case roadRight:
		return r.X - 60, r.Y + r.H/2
	case roadLeft:
		return r.X + r.W + 60, r.Y + r.H/2
	case roadDown:
		return r.Y - 60, r.X + r.W/2
	}
	return r.Y + r.H + 60, r.X + r.W/2
}

// roadAhead: how far beyond the end of the road the car is (negative while it is still on it).
func roadAhead(s RoadSeg, pos float64) float64 {
	r := s.R
	switch s.Dir {
	case roadRight:
		return pos - (r.X + r.W)
	case roadLeft:
		return r.X - pos
	case roadDown:
		return pos - (r.Y + r.H)
	}
	return r.Y - pos
}

func (c *RoadCar) at(s RoadSeg) Vec {
	if s.horizontal() {
		return Vec{c.pos, c.lat}
	}
	return Vec{c.lat, c.pos}
}

// roadRunOver: whoever is under a moving car dies; it never raises the criminality.
func (g *Game) roadRunOver(c *RoadCar, s RoadSeg) {
	if c.speed < 8 {
		return
	}
	at := c.at(s)
	hitX, hitY := c.length/2*0.9, 12.0
	if !s.horizontal() {
		hitX, hitY = 13, c.length/2*0.9
	}
	for _, gu := range g.guests {
		if gu.out() || gu.state == gLift {
			continue
		}
		if math.Abs(gu.pos.X-at.X) < hitX && math.Abs(gu.pos.Y-at.Y) < hitY {
			g.killGuest(gu, false)
			g.sayNews("kill", map[string]string{"actor": "Un coche", "victim": guestLabel(gu)})
			g.snd.Play("thud")
		}
	}
	g.eachPlayer(func(p *Player) {
		if !p.dead && !p.hidden && math.Abs(p.pos.X-at.X) < hitX*0.9 && math.Abs(p.pos.Y-at.Y) < hitY*0.85 {
			g.die("Te ha atropellado un coche")
		}
	})
}

// vehNoseLeft: the sprites of the car family are drawn with the nose on the left; every other
// vehicle (vans, trucks, bus, bikes, the Rocket League cars...) has its nose on the right.
var vehNoseLeft = map[string]bool{"car": true, "taxi": true, "homercar": true, "wackypink": true, "wackymean": true, "tank": true, "tow": true}

// vehMirror: does the sprite of this vehicle have to be mirrored to point where it drives?
func vehMirror(name string, movingRight bool) bool {
	if vehNoseLeft[name] {
		return movingRight
	}
	return !movingRight
}

// roadSpecialPick now and then swaps the plain car for a set piece: the tank (twice per party at
// most), a horse with its rider, the tow truck, or one of the ambient vehicles that have views.
func (g *Game) roadSpecialPick(name string) string {
	switch r := g.rng.Float64(); {
	case r < 0.03 && g.tankN < 2:
		g.tankN++
		return "tank"
	case r < 0.09:
		return "horse"
	case r < 0.13:
		return "tow"
	case r < 0.30:
		var custom []string
		for n, k := range propKinds {
			if k.Ambient > 0 && vehViewDefs[n] != nil && !roadSpecials[n] {
				custom = append(custom, n)
			}
		}
		if len(custom) > 0 {
			sort.Strings(custom)
			return custom[g.rng.Intn(len(custom))]
		}
	}
	return name
}

// drawRoadSpecial draws the tank, the tow truck and the horse on a road. Horizontally they use their
// side sprites; vertically the front or rear view, according to where they are heading.
func (g *Game) drawRoadSpecial(dst *ebiten.Image, c *RoadCar, s RoadSeg, at Vec) {
	bob := math.Round(math.Sin(c.pos * 0.35))
	if !s.horizontal() {
		vp, ok := vehicleView(c.name, c.col, s.Dir == roadDown)
		if !ok {
			return
		}
		if c.name != "horse" {
			bob = 0
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(g.wx(at.X)-vp.ox), math.Round(g.wy(at.Y+10)-vp.oy+bob))
		dst.DrawImage(vp.img, op)
		return
	}
	right := s.Dir != roadLeft
	x, y := g.wx(at.X), g.wy(at.Y+8)
	op := &ebiten.DrawImageOptions{}
	switch c.name {
	case "tank":
		im := g.as.tank
		w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
		if vehMirror("tank", right) {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Translate(math.Round(x-w/2), math.Round(y-h))
		dst.DrawImage(im, op)
	case "tow":
		pi := towSide()
		w := float64(pi.img.Bounds().Dx())
		if vehMirror("tow", right) {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
			op.GeoM.Translate(math.Round(x-(w-pi.ox)), math.Round(y-pi.oy))
		} else {
			op.GeoM.Translate(math.Round(x-pi.ox), math.Round(y-pi.oy))
		}
		dst.DrawImage(pi.img, op)
	case "horse":
		pi := g.as.props["horse"][0]
		w := float64(pi.img.Bounds().Dx())
		if !right {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Translate(math.Round(x-w/2), math.Round(y-pi.oy+bob))
		dst.DrawImage(pi.img, op)
		dir := DirLeft
		if right {
			dir = DirRight
		}
		g.drawActor(dst, g.as.cowCS, AnimIdle, 0, Vec{at.X, at.Y + 8 - 16}, dir, bob, false)
	}
}

func (g *Game) drawRoadCar(dst *ebiten.Image, c *RoadCar) {
	s := g.lvl.Def.Roads[c.seg]
	if roadSpecials[c.name] {
		g.drawRoadSpecial(dst, c, s, c.at(s))
		return
	}
	imgs := g.as.props[c.name]
	if len(imgs) == 0 {
		return
	}
	pi := imgs[c.col%len(imgs)]
	w, h := float64(pi.img.Bounds().Dx()), float64(pi.img.Bounds().Dy())
	at := c.at(s)
	op := &ebiten.DrawImageOptions{}
	if s.horizontal() {
		x, y := g.wx(at.X), g.wy(at.Y+8)
		if vehMirror(c.name, s.Dir != roadLeft) {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
			op.GeoM.Translate(math.Round(x-(w-pi.ox)), math.Round(y-pi.oy))
		} else {
			op.GeoM.Translate(math.Round(x-pi.ox), math.Round(y-pi.oy))
		}
		dst.DrawImage(pi.img, op)
		return
	}
	// vertical road: seen head on when it comes down toward the camera, from behind when it goes up
	if vp, ok := vehicleView(c.name, c.col, s.Dir == roadDown); ok {
		op.GeoM.Translate(math.Round(g.wx(at.X)-vp.ox), math.Round(g.wy(at.Y+10)-vp.oy))
		dst.DrawImage(vp.img, op)
		return
	}
	// no such view: the side view turned a quarter, about the middle of the picture
	op.GeoM.Translate(-w/2, -h/2)
	// the nose of a right-facing sprite goes down with +90 degrees, a left-facing one with -90
	turn := math.Pi / 2
	if s.Dir != roadDown {
		turn = -turn
	}
	if vehNoseLeft[c.name] {
		turn = -turn
	}
	op.GeoM.Rotate(turn)
	op.GeoM.Translate(math.Round(g.wx(at.X)), math.Round(g.wy(at.Y)-6))
	dst.DrawImage(pi.img, op)
}

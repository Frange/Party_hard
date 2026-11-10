package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The railway station: two tracks between two platforms with a level
// crossing in the middle. Trains pass (fast, and deadly to anyone on the
// tracks) or stop and let new guests out.
// ---------------------------------------------------------------------------

func genTrainStation(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "ESTACIÓN DE TREN", Intro: "Andén 2: fiesta. Cuidado con la línea amarilla... y con el expreso.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{6, 6, 20, 70},
		Guests: 44 + rng.Intn(6), CellSize: 16,
		Tracks: []float64{278, 310}, TrainStop: 480,
	}
	d.Floors = []Floor{
		F(0, 20, 960, 94, "tile"),
		F(0, 114, 960, 148, "concrete"),
		F(0, 250, 960, 12, "edge"),
		F(0, 262, 960, 64, "rails"),
		F(0, 326, 960, 12, "edge"),
		F(0, 338, 960, 116, "concrete"),
	}
	d.Walls = walls(
		hw(0, 960, 20, 14, WallBrick, [2]float64{140, 200}, [2]float64{760, 820}),
		vw(0, 20, 444, 10, WallFence), vw(950, 20, 444, 10, WallFence),
		hw(0, 960, 256, 5, WallFence, [2]float64{420, 540}),
		hw(0, 960, 326, 5, WallFence, [2]float64{420, 540}),
		hw(0, 960, 114, 8, WallPlaster, [2]float64{60, 330}, [2]float64{630, 900}),
	)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	frontX := 430 + rng.Float64()*140
	pl.avoid = append(pl.avoid, Vec{frontX, 420}, Vec{frontX, 380}, Vec{480, 294})
	// hall: the DJ plays under the departures board
	pl.at("dj", 480, 76, 0)
	pl.at("speaker", 420, 74, 0)
	pl.at("speaker", 540, 74, 0)
	d.Performers = []Perf{{Vec{480, 62}, "dj", DirDown}}
	pl.at("ticketbooth", 130, 100, 0)
	pl.at("ticketbooth", 830, 100, 0)
	pl.at("posters", 250, 44, 0)
	pl.at("posters", 700, 44, 1)
	pl.at("phone", 40, 104, 0)
	pl.scatter(R(40, 120, 880, 120), 7, 90, "bench", "lamp", "plant")
	pl.scatter(R(40, 345, 880, 100), 7, 90, "bench", "lamp", "plant", "cafetable")
	pl.at("foodtruck", 800, 430, 0)
	genStreet(d, frontX)
	d.Props = props
	d.Roam = []Rect{R(40, 50, 880, 56), R(40, 126, 880, 112), R(436, 258, 88, 72), R(40, 350, 880, 85)}
	d.Dance = []Rect{R(390, 44, 180, 52)}
	d.Animals = []AnimalSpawn{{"dog", Vec{300, 380}, R(100, 345, 400, 90)}}
	d.Guards = [][]Vec{{{60, 160}, {880, 160}}, {{60, 400}, {880, 400}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

type Train struct {
	trackY  float64
	x       float64 // head
	dir     float64 // +1 heads right
	vx      float64
	state   int // 0 warning, 1 arriving, 2 stopped, 3 leaving, 4 express
	t       float64
	cars    int
	col     int
	stops   bool
	boarded bool
	n       int // victims
}

const (
	trainCarLen = 112.0
	trWarn      = 0
	trArrive    = 1
	trStopped   = 2
	trLeave     = 3
	trExpress   = 4
)

func (tr *Train) length() float64 { return float64(tr.cars) * trainCarLen }

func (g *Game) updateTrains(dt float64) {
	def := g.lvl.Def
	if len(def.Tracks) == 0 || g.mode != ModePlay {
		return
	}
	if g.train == nil {
		if g.time >= g.trainT {
			tr := &Train{trackY: def.Tracks[g.rng.Intn(len(def.Tracks))], cars: 4 + g.rng.Intn(3), col: g.rng.Intn(4), stops: g.rng.Float64() < 0.6}
			tr.dir = 1
			if g.rng.Intn(2) == 0 {
				tr.dir = -1
			}
			g.train = tr
			g.say("¡Se acerca un tren! Cuidado con las vías")
			g.snd.Play("alert")
		}
		return
	}
	tr := g.train
	tr.t += dt
	stopX := def.TrainStop
	switch tr.state {
	case trWarn:
		if tr.t > 3.5 {
			tr.state, tr.t = trArrive, 0
			tr.x = -20
			tr.vx = 230 * tr.dir
			if tr.dir < 0 {
				tr.x = g.lvl.W + 20
			}
			if !tr.stops {
				tr.state, tr.vx = trExpress, 340*tr.dir
			}
		}
	case trArrive:
		// the head runs until the whole train is centred on the stop
		headStop := stopX + tr.dir*tr.length()/2
		rem := (headStop - tr.x) * tr.dir
		sp := math.Abs(tr.vx)
		want := math.Min(230, math.Sqrt(math.Max(0, 2*90*rem))+4)
		if rem <= 1 || want < 6 {
			tr.vx, tr.state, tr.t = 0, trStopped, 0
			g.snd.Play("alert")
		} else {
			sp = math.Min(sp+60*dt, want)
			if sp > want {
				sp = want
			}
			tr.vx = sp * tr.dir
		}
	case trStopped:
		if !tr.boarded && tr.t > 1 {
			tr.boarded = true
			n := minI(3+g.rng.Intn(5), g.guestRoom()) // never past the kill target (cap.go)
			py := tr.trackY - 26
			if tr.trackY > g.lvl.Def.Tracks[0] {
				py = tr.trackY + 26
			}
			for i := 0; i < n && g.guestRoom() > 0; i++ {
				x := stopX - tr.length()/2 + 60 + g.rng.Float64()*(tr.length()-120)
				g.addGuestAt(Vec{x, py})
			}
			if n > 0 {
				g.say("El tren trae " + itoa(n) + " pasajeros a la fiesta")
			}
		}
		if tr.t > 8 {
			tr.state, tr.t = trLeave, 0
		}
	case trLeave:
		tr.vx = math.Min(math.Abs(tr.vx)+80*dt, 260) * tr.dir
	case trExpress:
	}
	tr.x += tr.vx * dt
	if tr.state == trWarn {
		return
	}
	// the head of the train kills whatever is on its tracks
	if math.Abs(tr.vx) > 25 {
		x0, x1 := tr.x, tr.x-tr.dir*tr.length()
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		for _, gu := range g.guests {
			if gu.out() || gu.state == gLift || gu.perf != "" {
				continue
			}
			if gu.pos.X > x0 && gu.pos.X < x1 && math.Abs(gu.pos.Y-tr.trackY) < 17 {
				g.killGuest(gu, false)
				g.sayNews("kill", map[string]string{"actor": "El tren", "victim": guestLabel(gu)})
				g.snd.Play("thud")
				tr.n++
				g.checkDone()
			}
		}
		g.eachPlayer(func(p *Player) {
			if !p.dead && !p.hidden && p.pos.X > x0 && p.pos.X < x1 && math.Abs(p.pos.Y-tr.trackY) < 15 {
				g.die("Te ha arrollado el tren")
			}
		})
	}
	// gone: schedule the next one
	if tr.t > 1 && ((tr.dir > 0 && tr.x-tr.length() > g.lvl.W+30) || (tr.dir < 0 && tr.x+tr.length() < -30)) {
		g.train = nil
		g.trainT = g.time + 45 + g.rng.Float64()*45
	}
	if tr.state == trLeave && tr.t > 14 {
		g.train = nil
		g.trainT = g.time + 45 + g.rng.Float64()*45
	}
}

// crossing lights: blinking red while a train is coming or passing
func (g *Game) trainWarning() bool {
	return g.train != nil && (g.train.state == trWarn || math.Abs(g.train.vx) > 25)
}

var trainLooks = []struct{ body, stripe, roof uint32 }{
	{0xe8e8ee, 0xc82040, 0x9a9aa8},
	{0x2a58b8, 0xf4f4f4, 0x1a3878},
	{0xe8c030, 0x3a3a44, 0xa08820},
	{0x2a8a58, 0xe8e0c0, 0x1a5a38},
}

var trainImgs = map[int]*ebiten.Image{}

// trainCar: a side view facing right; kind 0 coach, 1 cab at the right end.
func trainCar(look, kind int) *ebiten.Image {
	key := look*2 + kind
	if im, ok := trainImgs[key]; ok {
		return im
	}
	lk := trainLooks[look%len(trainLooks)]
	body, stripe, roof := c(lk.body), c(lk.stripe), c(lk.roof)
	b := newPix(int(trainCarLen), 46)
	b.ellipse(56, 43, 54, 2.5, color.RGBA{0, 0, 0, 90})
	b.rect(2, 6, 108, 30, body)
	b.rect(2, 4, 108, 3, roof)
	b.rect(2, 6, 108, 1, shade(body, 1.2))
	b.rect(2, 26, 108, 4, stripe)
	b.rect(2, 36, 108, 4, c(0x2a2a32))
	for x := 8; x < 92; x += 16 {
		b.rect(x, 11, 11, 11, c(0x2a3a5a))
		b.rect(x, 11, 11, 2, c(0x6a86b4))
	}
	b.rect(52, 10, 7, 24, shade(body, 0.7)) // door
	if kind == 1 {
		b.rect(96, 8, 14, 12, c(0x2a3a5a))
		b.rect(98, 9, 4, 8, c(0x6a86b4))
		b.rect(106, 24, 4, 3, c(0xfff0b0)) // headlight
		b.rect(108, 20, 2, 16, shade(body, 0.8))
	}
	for _, wx := range []float64{16, 30, 82, 96} {
		b.ellipse(wx, 40, 4.5, 4.5, c(0x16161c))
		b.ellipse(wx, 40, 1.8, 1.8, c(0x8a8a96))
	}
	im := b.outlined(c(0x0a0a0e)).img()
	trainImgs[key] = im
	return im
}

func (g *Game) drawTrain(dst *ebiten.Image) {
	tr := g.train
	if tr == nil || tr.state == trWarn {
		return
	}
	for i := 0; i < tr.cars; i++ {
		kind := 0
		if i == 0 {
			kind = 1
		}
		im := trainCar(tr.col, kind)
		w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
		cx := tr.x - tr.dir*(float64(i)*trainCarLen+trainCarLen/2)
		op := &ebiten.DrawImageOptions{}
		if tr.dir < 0 {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Translate(math.Round(g.wx(cx)-w/2), math.Round(g.wy(tr.trackY)-h+14))
		dst.DrawImage(im, op)
	}
}

// drawCrossingLights: two red lamps blinking at the crossing while a train is about.
func (g *Game) drawCrossingLights(dst *ebiten.Image) {
	if !g.trainWarning() || int(g.time*4)%2 == 0 {
		return
	}
	for _, x := range []float64{410, 550} {
		for _, y := range []float64{252, 330} {
			g.glowAt(dst, Vec{x, y}, 12, color.RGBA{255, 40, 30, 150})
			fillRect(dst, g.wx(x)-2, g.wy(y)-2, 4, 4, color.RGBA{255, 80, 70, 255})
		}
	}
}

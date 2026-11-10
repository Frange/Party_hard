package main

import (
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// 70s DISCO: the dance floor is a grid of tiles (36 x 24, 8 px each) that light up with the music and draw a
// random pixel picture. A new picture every two bars (phrase) and a new palette order every bar; a diagonal wipe
// changes from the old picture to the new one; each tile pulses with the beat.
//
//   - the pulse comes from the BPM of the genre of the party and from g.time (music/musicreact.go has no bass
//     energy, only a volume bump), so it follows the synthesised music; with an mp3 it is not locked to it (not heard)
//   - the pictures are made by generators (heart, star, smiley, spiral, waves, checkerboard, diamonds, mosaic,
//     rings, flower, sun, chevrons, cross) with random size / position / mirroring / palette, from a generator of
//     its own (never g.rng) that is re-seeded in every party; the same kind never comes twice in a row
//   - the layer is ONE pixel buffer of our own written in full each frame and uploaded with ONE WritePixels, then ONE
//     DrawImage: nothing is allocated per frame
// ---------------------------------------------------------------------------

const (
	discoW           = 36  // tiles across
	discoH           = 24  // tiles down
	discoCell        = 8   // pixels per tile
	discoPhraseBeats = 8.0 // a picture lasts two bars of four beats
	discoWipeBeats   = 1.5 // how long the change between two pictures takes
)

type discoPat [discoW * discoH]uint8 // 0 = tile off, 1..3 = colour slot

// 70s palettes (3 colours each); lit tiles use them in an order that rotates every bar
var discoPalettes = [][3][3]uint8{
	{{255, 122, 26}, {166, 60, 216}, {255, 210, 46}},
	{{46, 196, 182}, {255, 79, 160}, {255, 210, 46}},
	{{255, 79, 60}, {255, 160, 30}, {250, 240, 200}},
	{{120, 80, 230}, {255, 79, 160}, {60, 220, 200}},
	{{140, 220, 60}, {255, 122, 26}, {255, 240, 120}},
	{{255, 100, 120}, {80, 160, 255}, {250, 235, 170}},
}

var discoKindNames = []string{"heart", "star", "smiley", "spiral", "waves", "checker", "diamonds", "mosaic", "rings", "flower", "sun", "chevrons", "cross"}

type discoState struct {
	lvl         *Level
	rng         *rand.Rand
	img         *ebiten.Image
	buf         []byte
	cur, prev   discoPat
	kind, pkind int
	hist        [4]int
	phrase      int
	pal         int
	op          ebiten.DrawImageOptions
	count       int
	lit         int // lit tiles of the last frame (tests)
	lastKinds   []int
}

var disco discoState

// discoSeed: tests set it to make the pictures deterministic (0 = from the clock, a new one in every party).
var discoSeed int64

func (st *discoState) reset(l *Level) {
	seed := discoSeed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	st.lvl, st.rng = l, rand.New(rand.NewSource(seed))
	if st.buf == nil {
		st.buf = make([]byte, discoW*discoCell*discoH*discoCell*4)
	}
	if st.img == nil {
		st.img = ebiten.NewImage(discoW*discoCell, discoH*discoCell)
	}
	st.hist = [4]int{-1, -1, -1, -1}
	st.phrase, st.count, st.lastKinds = 0, 0, nil
	st.pal = st.rng.Intn(len(discoPalettes))
	st.kind = st.next(&st.cur)
	st.pkind, st.prev = st.kind, st.cur
}

// next draws a new picture into pat with a kind that was not used in the last four, and returns the kind.
func (st *discoState) next(pat *discoPat) int {
	for try := 0; ; try++ {
		k := st.rng.Intn(len(discoKindNames))
		recent := false
		for _, h := range st.hist {
			recent = recent || h == k
		}
		if recent && try < 40 {
			continue
		}
		*pat = discoPat{}
		discoMake(k, pat, st.rng)
		st.hist[st.count%len(st.hist)] = k
		st.count++
		st.lastKinds = append(st.lastKinds, k)
		return k
	}
}

// discoBeat: the beat count of the music at game time t (the BPM of the genre of the party).
func discoBeat(t, bpm float64) float64 { return t * bpm / 60 }

// discoPulse: how hard the tiles flash at that beat: a kick on every beat (1 -> 0) and a softer one on the off beat.
func discoPulse(beat float64) (kick, off float64) {
	f := beat - math.Floor(beat)
	kick = math.Pow(1-f, 3)
	g := math.Mod(beat*2, 1)
	off = 0.35 * math.Pow(1-g, 4) * (1 - kick*0.5)
	return
}

func discoBPM(g *Game) float64 {
	if g.lvl != nil {
		if b := genreForTitle(g.lvl.Def.Title).bpm; b > 0 {
			return b
		}
	}
	return 116
}

// ---- the picture makers -------------------------------------------------------------------------------------

func (p *discoPat) set(x, y int, v uint8) {
	if x >= 0 && y >= 0 && x < discoW && y < discoH {
		p[y*discoW+x] = v
	}
}

func discoMake(kind int, p *discoPat, r *rand.Rand) {
	cx, cy := float64(discoW-1)/2, float64(discoH-1)/2
	// a random centre offset and scale: the same kind never looks the same twice
	ox, oy := float64(r.Intn(7)-3)*r.Float64(), float64(r.Intn(3)-1)*r.Float64()
	sc := 0.8 + r.Float64()*0.25
	slot := func(i int) uint8 { return uint8(1 + (i%3+3)%3) }
	switch kind {
	case 0: // heart
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				dx, dy := (float64(x)-cx-ox)/(9.5*sc), -(float64(y)-cy-oy)/(9.0*sc)-0.1
				a := dx*dx + dy*dy - 1
				if a*a*a-dx*dx*dy*dy*dy <= 0 {
					v := slot(0)
					if a*a*a-dx*dx*dy*dy*dy > -0.12 {
						v = slot(1) // outline
					}
					p.set(x, y, v)
				}
			}
		}
	case 1: // star with 5 or 6 points, hollow or solid
		n := 5 + r.Intn(2)
		rot := -math.Pi / 2
		outer, inner := 11.0*sc, 4.6*sc+r.Float64()*2
		var px, py [12]float64
		for i := 0; i < 2*n; i++ {
			rad := outer
			if i%2 == 1 {
				rad = inner
			}
			a := rot + float64(i)*math.Pi/float64(n)
			px[i], py[i] = cx+ox+math.Cos(a)*rad*1.0, cy+oy+math.Sin(a)*rad*1.0
		}
		hollow := r.Intn(3) == 0
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				if discoInPoly(float64(x), float64(y), px[:2*n], py[:2*n]) {
					v := slot(0)
					if hollow && discoInPoly(float64(x), float64(y), shrinkPoly(px[:2*n], cx+ox, 0.62), shrinkPoly(py[:2*n], cy+oy, 0.62)) {
						v = slot(2)
					}
					p.set(x, y, v)
				}
			}
		}
	case 2: // smiley: a disc, two eyes and a smile
		rad := 10.3 * sc
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				dx, dy := float64(x)-cx-ox, float64(y)-cy-oy
				d := math.Hypot(dx, dy)
				if d <= rad {
					v := slot(0)
					if d > rad-1.5 {
						v = slot(1)
					}
					p.set(x, y, v)
				}
			}
		}
		ex, ey := int(math.Round(cx+ox)), int(math.Round(cy+oy-rad*0.3))
		for _, s := range []int{-1, 1} {
			for k := 0; k < 3; k++ {
				p.set(ex+s*int(math.Round(rad*0.38)), ey+k, 3)
				p.set(ex+s*int(math.Round(rad*0.38))+1, ey+k, 3)
			}
		}
		for a := 0.25; a < math.Pi-0.25; a += 0.12 {
			p.set(int(math.Round(cx+ox+math.Cos(a)*rad*0.6)), int(math.Round(cy+oy+math.Sin(a)*rad*0.55)), 3)
			p.set(int(math.Round(cx+ox+math.Cos(a)*rad*0.6)), int(math.Round(cy+oy+math.Sin(a)*rad*0.55))+1, 3)
		}
	case 3: // spiral (1 to 3 arms)
		arms := 1 + r.Intn(3)
		tight := 3.2 + r.Float64()*2.4
		dir := 1.0
		if r.Intn(2) == 0 {
			dir = -1
		}
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				dx, dy := float64(x)-cx-ox, float64(y)-cy-oy
				d, th := math.Hypot(dx, dy), math.Atan2(dy, dx)*dir
				ph := math.Mod(d/tight-th*float64(arms)/(2*math.Pi)+100, 1)
				if ph < 0.5 {
					p.set(x, y, slot(int(d/tight)))
				}
			}
		}
	case 4: // waves
		per := 4.0 + float64(r.Intn(3))
		amp := 1.5 + r.Float64()*2.5
		fr := 0.25 + r.Float64()*0.3
		ph := r.Float64() * 6
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				v := math.Mod(float64(y)+amp*math.Sin(float64(x)*fr+ph)+100, per)
				if v < per/2 {
					p.set(x, y, slot(int((float64(y)+amp*math.Sin(float64(x)*fr+ph))/per)))
				}
			}
		}
	case 5: // checkerboard (straight or diagonal)
		s := 2 + r.Intn(3)
		diag := r.Intn(2) == 0
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				a, b := x/s, y/s
				if diag {
					a, b = (x+y)/s, (x-y+discoH)/s
				}
				if (a+b)%2 == 0 {
					p.set(x, y, slot(a/2+b/2))
				}
			}
		}
	case 6: // concentric diamonds / squares
		s := 2.0 + float64(r.Intn(3))
		sq := r.Intn(2) == 0
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				dx, dy := math.Abs(float64(x)-cx-ox), math.Abs(float64(y)-cy-oy)
				d := dx + dy
				if sq {
					d = math.Max(dx, dy*1.4)
				}
				if int(d/s)%2 == 0 {
					p.set(x, y, slot(int(d/s)/2))
				}
			}
		}
	case 7: // mosaic: random cells mirrored both ways (an inkblot of tiles)
		dens := 0.38 + r.Float64()*0.2
		for y := 0; y < discoH/2; y++ {
			for x := 0; x < discoW/2; x++ {
				if r.Float64() < dens {
					v := uint8(1 + r.Intn(3))
					p.set(x, y, v)
					p.set(discoW-1-x, y, v)
					p.set(x, discoH-1-y, v)
					p.set(discoW-1-x, discoH-1-y, v)
				}
			}
		}
	case 8: // rings
		s := 2.0 + float64(r.Intn(2))
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				d := math.Hypot(float64(x)-cx-ox, (float64(y)-cy-oy)*1.15)
				if int(d/s)%2 == 0 {
					p.set(x, y, slot(int(d/s)/2))
				}
			}
		}
	case 9: // flower: petals around a centre
		petals := float64(5 + r.Intn(4))
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				dx, dy := float64(x)-cx-ox, float64(y)-cy-oy
				d, th := math.Hypot(dx, dy), math.Atan2(dy, dx)
				lim := (4 + 8*math.Abs(math.Cos(petals*th/2))) * sc
				if d <= lim {
					v := slot(0)
					if d < 3 {
						v = slot(2)
					} else if d > lim-1.4 {
						v = slot(1)
					}
					p.set(x, y, v)
				}
			}
		}
	case 10: // sun: a disc with rays
		rays := float64(8 + 2*r.Intn(3))
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				dx, dy := float64(x)-cx-ox, float64(y)-cy-oy
				d, th := math.Hypot(dx, dy), math.Atan2(dy, dx)
				switch {
				case d <= 5.2*sc:
					p.set(x, y, slot(0))
				case d <= 11.5*sc && math.Cos(rays*th) > 0.55:
					p.set(x, y, slot(1))
				}
			}
		}
	case 11: // chevrons
		per := 4 + r.Intn(3)
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				v := (y + int(math.Abs(float64(x)-cx-ox))*1) % per
				if v < per/2 {
					p.set(x, y, slot((y+int(math.Abs(float64(x)-cx)))/per))
				}
			}
		}
	default: // a big plus / cross pattern with a border
		t := 3 + r.Intn(3)
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW; x++ {
				dx, dy := math.Abs(float64(x)-cx-ox), math.Abs(float64(y)-cy-oy)
				if dx < float64(t)/2 || dy < float64(t)/2 {
					p.set(x, y, slot(0))
				} else if x < 2 || y < 2 || x >= discoW-2 || y >= discoH-2 {
					p.set(x, y, slot(1))
				} else if int(dx+dy)%7 == 0 {
					p.set(x, y, slot(2))
				}
			}
		}
	}
	// a random mirror (so a heart can end up as two) and a few stray sparkles
	if r.Intn(4) == 0 {
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW/2; x++ {
				p[y*discoW+discoW-1-x] = p[y*discoW+x]
			}
		}
	}
	for i := 0; i < 5+r.Intn(6); i++ {
		x, y := r.Intn(discoW), r.Intn(discoH)
		if p[y*discoW+x] == 0 {
			p[y*discoW+x] = uint8(1 + r.Intn(3))
		}
	}
}

func discoInPoly(x, y float64, px, py []float64) bool {
	in := false
	for i, j := 0, len(px)-1; i < len(px); j, i = i, i+1 {
		if (py[i] > y) != (py[j] > y) && x < (px[j]-px[i])*(y-py[i])/(py[j]-py[i])+px[i] {
			in = !in
		}
	}
	return in
}

// shrinkPoly: the coordinates pulled toward c (a small copy for the hollow centre of a star).
func shrinkPoly(v []float64, c, k float64) []float64 {
	out := make([]float64, len(v))
	for i := range v {
		out[i] = c + (v[i]-c)*k
	}
	return out
}

// ---- drawing --------------------------------------------------------------------------------------------------

func discoHash(x, y, n int) uint32 {
	h := uint32(x)*73856093 ^ uint32(y)*19349663 ^ uint32(n)*83492791
	h ^= h >> 13
	h *= 0x5bd1e995
	return h ^ h>>15
}

// updateDisco brings the picture state to the phrase of time t (called by the draw; pure function of t apart from
// the random pictures that come one after the other).
func (st *discoState) advance(beat float64) {
	ph := int(beat / discoPhraseBeats)
	if ph <= st.phrase {
		return
	}
	st.phrase = ph
	st.prev, st.pkind = st.cur, st.kind
	st.kind = st.next(&st.cur)
}

// render paints the tiles into st.buf for the beat count `beat`; dim = the DJ is dead (lights nearly out).
func (st *discoState) render(beat float64, dim bool) {
	st.advance(beat)
	bar := int(beat / 4)
	kick, off := discoPulse(beat)
	pl := discoPalettes[(st.pal+st.phrase)%len(discoPalettes)]
	wipe := 1.0
	into := (beat - float64(st.phrase)*discoPhraseBeats) / discoWipeBeats
	if st.phrase > 0 && into < 1 {
		wipe = into
	}
	thr := wipe * float64(discoW+discoH+4)
	stride := discoW * discoCell * 4
	lit := 0
	for cy := 0; cy < discoH; cy++ {
		for cx := 0; cx < discoW; cx++ {
			i := cy*discoW + cx
			v := st.cur[i]
			front := float64(cx+cy) - thr
			if front > 0 {
				v = st.prev[i] // not wiped yet
			}
			var r, g, b float64
			if v != 0 && !dim {
				lit++
				c := pl[(int(v)-1+bar)%3]
				k := 0.52 + 0.48*kick + 0.2*off
				if front > -2.5 && front <= 0 { // the bright edge of the wipe
					k = 1.25
				}
				r, g, b = float64(c[0])*k, float64(c[1])*k, float64(c[2])*k
			} else {
				// an unlit tile: a dark purple glass with a faint pulse and the odd twinkle
				k := 0.05 + 0.05*kick*0.6
				if dim {
					k = 0.04
				} else if discoHash(cx, cy, int(beat/2))%29 == 0 {
					k += 0.1 * kick
				}
				c := pl[(cx+cy)%3]
				r, g, b = 22+float64(c[0])*k, 14+float64(c[1])*k*0.8, 30+float64(c[2])*k
			}
			discoTile(st.buf, stride, cx*discoCell, cy*discoCell, r, g, b)
		}
	}
	st.lit = lit
}

func discoClamp(v float64) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

// discoTile writes one 8x8 tile: bevelled glass with a dark grout on the right and bottom edges.
func discoTile(buf []byte, stride, x0, y0 int, r, g, b float64) {
	cr, cg, cb := discoClamp(r), discoClamp(g), discoClamp(b)
	hr, hg, hb := discoClamp(r*1.22+10), discoClamp(g*1.22+10), discoClamp(b*1.22+10)
	sr, sg, sb := discoClamp(r*0.72), discoClamp(g*0.72), discoClamp(b*0.72)
	for py := 0; py < discoCell; py++ {
		o := (y0+py)*stride + x0*4
		for px := 0; px < discoCell; px++ {
			var rr, gg, bb byte
			switch {
			case py == discoCell-1 || px == discoCell-1:
				rr, gg, bb = 6, 4, 10 // grout
			case py == 0 || px == 0:
				rr, gg, bb = hr, hg, hb
			case py == discoCell-2 || px == discoCell-2:
				rr, gg, bb = sr, sg, sb
			default:
				rr, gg, bb = cr, cg, cb
			}
			buf[o], buf[o+1], buf[o+2], buf[o+3] = rr, gg, bb, 255
			o += 4
		}
	}
}

// drawDiscoTiles: the lit dance floor of the 70s disco (floor kind "disco70"), one WritePixels + one DrawImage.
func (g *Game) drawDiscoTiles(dst *ebiten.Image, f Floor) {
	if disco.lvl != g.lvl || disco.rng == nil {
		disco.reset(g.lvl)
	}
	st := &disco
	st.render(discoBeat(g.time, discoBPM(g)), g.djDead)
	st.img.WritePixels(st.buf)
	st.op.GeoM.Reset()
	st.op.GeoM.Translate(g.wx(f.R.X), g.wy(f.R.Y))
	dst.DrawImage(st.img, &st.op)
}

package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The small lit dance floors of the DISCO SETENTERA (v1.4.0, tanda H): like the big one (disco70.go) but each with a
// drawing of its own that moves with the beat: rays turning round the centre, squares growing out of the middle and
// chevrons running along. They are floors of the same kind "disco70" (so every system treats them as the lit floor);
// drawDiscoTiles sends every one that is not the big floor here. One buffer, one WritePixels and one DrawImage each;
// nothing is allocated per frame.
// ---------------------------------------------------------------------------

// discoMinis: the small floors and their drawing (0 rays, 1 squares, 2 chevrons). Multiples of 8 px. The west side of
// the hall and the front: the east side stays free for the VIP zone (vip.go needs a big hole somewhere).
var discoMinis = []struct {
	R    Rect
	kind int
}{
	{Rect{176, 176, 112, 80}, 0},
	{Rect{176, 280, 112, 56}, 1},
	{Rect{408, 336, 144, 40}, 2},
}

type discoMini struct {
	lvl  *Level
	w, h int
	img  *ebiten.Image
	buf  []byte
	op   ebiten.DrawImageOptions
	lit  int // lit tiles of the last frame (tests)
}

var discoMiniSt [3]discoMini

// discoMiniOf: which small floor this rect is (-1 = none).
func discoMiniOf(r Rect) int {
	for i, m := range discoMinis {
		if m.R == r {
			return i
		}
	}
	return -1
}

// discoMiniValue: the colour slot (0 = off) of tile (x,y) of a w x h floor of that kind at beat b.
func discoMiniValue(kind, x, y, w, h int, b float64) uint8 {
	cx, cy := float64(w-1)/2, float64(h-1)/2
	dx, dy := float64(x)-cx, float64(y)-cy
	switch kind {
	case 0: // rays turning round the centre (eight sectors, one in two lit)
		a := math.Atan2(dy, dx*0.8)/(2*math.Pi)*8 + b*0.5
		s := int(math.Floor(a))
		if ((s%2)+2)%2 == 0 {
			return uint8(1 + ((s/2)%3+3)%3)
		}
		if dx*dx+dy*dy < 2.5 {
			return 3
		}
	case 1: // squares growing out of the middle
		d := int(math.Max(math.Abs(dx)*float64(h)/float64(w), math.Abs(dy)))
		ring := d - int(b)
		if ((ring%2)+2)%2 == 0 {
			return uint8(1 + ((d%3)+3)%3)
		}
	case 2: // chevrons running from left to right
		v := int(math.Floor(float64(x) - math.Abs(dy)*1.2 - b*2))
		if ((v%4)+4)%4 < 2 {
			return uint8(1 + (((v/4)%3)+3)%3)
		}
	}
	return 0
}

// drawDiscoMini paints one small floor (called by drawDiscoTiles for every "disco70" floor but the big one).
func (g *Game) drawDiscoMini(dst *ebiten.Image, f Floor) {
	i := discoMiniOf(f.R)
	if i < 0 {
		i = 0
	}
	st := &discoMiniSt[i]
	w, h := int(f.R.W)/discoCell, int(f.R.H)/discoCell
	if st.img == nil || st.w != w || st.h != h {
		st.w, st.h = w, h
		st.buf = make([]byte, w*discoCell*h*discoCell*4)
		if st.img != nil {
			st.img.Deallocate()
		}
		st.img = ebiten.NewImage(w*discoCell, h*discoCell)
	}
	st.lvl = g.lvl
	beat := discoBeat(g.time, discoBPM(g))
	kick, off := discoPulse(beat)
	pl := discoPalettes[(i*2+int(beat/8))%len(discoPalettes)]
	stride := w * discoCell * 4
	lit := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := discoMiniValue(discoMinis[i].kind, x, y, w, h, beat)
			var r, gg, b float64
			if v != 0 && !g.djDead {
				lit++
				c := pl[int(v)-1]
				k := 0.5 + 0.5*kick + 0.2*off
				r, gg, b = float64(c[0])*k, float64(c[1])*k, float64(c[2])*k
			} else {
				c := pl[(x+y)%3]
				k := 0.05 + 0.03*kick
				r, gg, b = 22+float64(c[0])*k, 14+float64(c[1])*k*0.8, 30+float64(c[2])*k
			}
			discoTile(st.buf, stride, x*discoCell, y*discoCell, r, gg, b)
		}
	}
	st.lit = lit
	st.img.WritePixels(st.buf)
	st.op.GeoM.Reset()
	st.op.GeoM.Translate(g.wx(f.R.X), g.wy(f.R.Y))
	dst.DrawImage(st.img, &st.op)
}

package main

import (
	"image/color"
)

// ---------------------------------------------------------------------------
// Front and back views of the road vehicles. On a vertical road the cars that
// come down are drawn from the front and the ones that go up from behind,
// instead of the side sprite turned a quarter. They use the same palettes,
// proportions and art kit (vehart.go) as the side views. The vv* helpers are
// what the special vehicles of vehviews2.go (tank, tow truck, horse) use.
// ---------------------------------------------------------------------------

type vehViewDef struct {
	w, h int // design size (same height as the side sprite)
	draw func(b *pixBuf, v int, front bool)
}

var vehViewDefs = map[string]*vehViewDef{}

var (
	vvDark   = c(0x15151b)
	vvGlass  = c(0x7aa0c0)
	vvLight  = c(0xfff0b0)
	vvTail   = c(0xe03030)
	vvPlate  = c(0xf4f4f8)
	vvGrille = c(0x2a2a34)
)

func vvShadow(b *pixBuf, w int) {
	b.ellipse(float64(w)/2, 32, float64(w)/2-1, 3.5, color.RGBA{0, 0, 0, 90})
}

// vvWheels: the sides of the two wheels that show next to the body, seen head on.
func vvWheels(b *pixBuf, w, top, bot int) {
	for _, x := range []int{0, w - 4} {
		b.rect(x, top, 4, bot-top, vvDark)
		b.rect(x+1, top, 1, bot-top, c(0x34343e))
	}
}

func vvPlateAt(b *pixBuf, w, y int) {
	b.rect(w/2-3, y, 6, 2, vvPlate)
}

type sedanP struct {
	w                                  int
	roofTop, roofBot, bodyTop, bodyBot int
	body, glass                        color.RGBA
	bar, sign, wings, cone, stripe     bool
	trim                               color.RGBA
	dark                               bool // black windows (the racing cars, the Batmobile)
	roof                               color.RGBA
}

// sedanView: any car-shaped vehicle from the front or from behind, with the art kit (vehart.go).
func sedanView(b *pixBuf, front bool, p sedanP) {
	w := p.w
	vvShadow(b, w)
	vaBuild(b, 0, func(t *pixBuf) {
		body := p.body
		roof := p.roof
		if roof.A == 0 {
			roof = shade(body, 0.95)
		}
		gl := p.glass
		if p.dark {
			gl = c(0x1a2030)
		}
		// tyres at both sides, under the body
		vaTyreFront(t, 1, p.bodyBot-7, 4, 10)
		vaTyreFront(t, w-5, p.bodyBot-7, 4, 10)
		// the cabin narrows to the roof
		for y := p.roofTop; y < p.bodyTop; y++ {
			in := 6 + (p.bodyTop-y)*3/maxI(1, p.bodyTop-p.roofTop)
			col := roof
			if y == p.roofTop {
				col = shade(roof, 1.3)
			}
			vaRow(t, y, in, w-1-in, col)
		}
		for y := p.roofTop + 2; y < p.bodyTop-1; y++ {
			in := 6 + (p.bodyTop-y)*3/maxI(1, p.bodyTop-p.roofTop) + 2
			g := gl
			if !front {
				g = shade(gl, 0.82)
			}
			vaRow(t, y, in, w-1-in, g)
		}
		gx := w/2 - 4
		t.set(gx, p.bodyTop-3, vaShineC)
		t.set(gx+1, p.bodyTop-4, vaShineC)
		t.set(gx+2, p.bodyTop-5, vaShineC)
		// the body
		for y := p.bodyTop; y < p.bodyBot; y++ {
			col := p.body
			switch {
			case y == p.bodyTop:
				col = shade(body, 1.28)
			case y >= p.bodyBot-3:
				col = shade(body, 0.8)
			}
			vaRow(t, y, 3, w-4, col)
		}
		vaRow(t, p.bodyBot-1, 4, w-5, vaSkirtC)
		t.rect(1, p.bodyTop+2, 2, 2, shade(roof, 0.8)) // mirrors
		t.rect(w-3, p.bodyTop+2, 2, 2, shade(roof, 0.8))
		if p.stripe {
			t.rect(w/2-2, p.bodyTop, 4, p.bodyBot-p.bodyTop-3, p.trim)
		}
		ly := p.bodyTop + 3
		if front {
			t.rect(4, ly, 4, 2, vaHeadC)
			t.rect(w-8, ly, 4, 2, vaHeadC)
			t.rect(4, ly+2, 2, 1, vaAmberC)
			t.rect(w-6, ly+2, 2, 1, vaAmberC)
			t.rect(9, ly, w-18, 4, c(0x2a2a34)) // grille
			for x := 10; x < w-9; x += 2 {
				t.rect(x, ly+1, 1, 2, c(0x4a4a56))
			}
		} else {
			t.rect(4, ly, 5, 2, vaTailC)
			t.rect(w-9, ly, 5, 2, vaTailC)
			t.rect(w-10, p.bodyBot-2, 3, 1, vaChromeC) // exhaust
		}
		t.rect(3, p.bodyBot-4, w-6, 1, vaChromeC) // bumper
		t.rect(w/2-3, p.bodyBot-6, 6, 2, c(0xf4f4f8))
		if p.bar {
			vaLightbar(t, w/2-7, p.roofTop-3, 14, 0, vaRed, vaBlue)
		}
		if p.sign {
			t.rect(w/2-6, p.roofTop-4, 12, 4, c(0xf4f4f8))
			t.rect(w/2-5, p.roofTop-3, 10, 1, c(0x1c1c22))
			t.rect(w/2-4, p.roofTop-5, 8, 1, vaAmberC)
		}
		if p.wings {
			t.rect(0, p.roofTop-3, 5, 6, c(0x6a6a74))
			t.rect(w-5, p.roofTop-3, 5, 6, c(0x6a6a74))
		}
		if p.cone {
			vaCone(t, w/2, p.roofTop)
		}
	}, nil)
}

// vaCone: the ice cream on the roof, seen head on.
func vaCone(t *pixBuf, cx, top int) {
	t.rect(cx-1, top-4, 3, 4, c(0xd8a860))
	t.ellipse(float64(cx)+0.5, float64(top)-6, 5, 4, c(0xf8c8d8))
	t.ellipse(float64(cx)+0.5, float64(top)-9, 3, 3, c(0xe85088))
}

// boxView: vans, trucks and the buses.
type boxP struct {
	w, top, bodyBot, glassTop, glassBot int
	body, cab                           color.RGBA
	boxBehind                           bool // a taller box behind a lower cab (the box truck)
	box                                 color.RGBA
	stripe                              color.RGBA
	cone                                bool
	bus                                 bool
	ribs                                bool
}

func boxView(b *pixBuf, front bool, p boxP) {
	w := p.w
	vvShadow(b, w)
	vaBuild(b, 0, func(t *pixBuf) {
		vaTyreFront(t, 0, p.bodyBot-7, 4, 10)
		vaTyreFront(t, w-4, p.bodyBot-7, 4, 10)
		fill := func(y0, y1 int, col color.RGBA) {
			for y := y0; y < y1; y++ {
				cc := col
				switch {
				case y == y0:
					cc = shade(col, 1.25)
				case y >= y1-3:
					cc = shade(col, 0.8)
				}
				vaRow(t, y, 2, w-3, cc)
			}
		}
		if p.boxBehind {
			fill(p.top, p.bodyBot, p.box)
			if p.ribs {
				for x := 4; x < w-4; x += 3 {
					t.rect(x, p.top+1, 1, p.bodyBot-p.top-4, shade(p.box, 0.85))
				}
			}
			if front {
				fill(p.glassTop-3, p.bodyBot, p.cab)
			} else {
				t.rect(w/2, p.top+2, 1, p.bodyBot-p.top-6, shade(p.box, 0.55)) // door gap
				t.rect(w/2-3, p.top+8, 2, 6, vaChromeC)
				t.rect(w/2+2, p.top+8, 2, 6, vaChromeC)
			}
		} else {
			fill(p.top, p.bodyBot, p.body)
		}
		vaRow(t, p.bodyBot-1, 3, w-4, vaSkirtC)
		if p.stripe.A > 0 {
			t.rect(2, p.bodyBot-9, w-4, 2, p.stripe)
		}
		if front {
			vaGlassPane(t, 4, p.glassTop, w-8, p.glassBot-p.glassTop, vaGlassC)
			if p.bus {
				t.rect(w/2, p.glassTop, 1, p.glassBot-p.glassTop, shade(p.body, 0.7))
			}
			t.rect(3, p.bodyBot-9, 4, 2, vaHeadC)
			t.rect(w-7, p.bodyBot-9, 4, 2, vaHeadC)
			t.rect(9, p.bodyBot-6, w-18, 3, c(0x2a2a34))
			for x := 10; x < w-9; x += 2 {
				t.rect(x, p.bodyBot-5, 1, 1, c(0x4a4a56))
			}
			t.rect(1, p.glassTop+2, 2, 3, c(0x2a2a30)) // mirrors
			t.rect(w-3, p.glassTop+2, 2, 3, c(0x2a2a30))
		} else {
			if !p.boxBehind {
				vaGlassPane(t, 5, p.glassTop, w-10, p.glassBot-p.glassTop-2, shade(vaGlassC, 0.8))
				if p.bus {
					t.rect(w/2-4, p.glassBot, 8, p.bodyBot-p.glassBot-4, c(0x2a2a34)) // emergency door
				}
			}
			t.rect(3, p.bodyBot-9, 4, 3, vaTailC)
			t.rect(w-7, p.bodyBot-9, 4, 3, vaTailC)
			t.rect(w/2-3, p.bodyBot-5, 6, 2, c(0xf4f4f8))
		}
		t.rect(2, p.bodyBot-3, w-4, 1, vaChromeC)
		if p.cone {
			vaCone(t, w/2, p.top)
		}
	}, nil)
}

func init() {
	reg := func(name string, w, h int, f func(b *pixBuf, v int, front bool)) {
		vehViewDefs[name] = &vehViewDef{w: w, h: h, draw: f}
	}
	reg("car", 32, 36, func(b *pixBuf, v int, front bool) {
		switch v {
		case 4:
			sedanView(b, front, sedanP{w: 32, roofTop: 7, roofBot: 17, bodyTop: 17, bodyBot: 31, body: c(0x22252c), roof: c(0xf0f2f6), glass: vaGlassC, bar: true})
		case 5:
			boxView(b, front, boxP{w: 32, top: 6, bodyBot: 31, glassTop: 10, glassBot: 19, body: c(0xf4f4f6), stripe: c(0xd82828)})
			vaBuild(b, 0, func(t *pixBuf) {
				t.rect(13, 1, 3, 3, vaRed)
				t.rect(16, 1, 3, 3, vaBlue)
				if !front {
					t.rect(15, 8, 2, 7, c(0x3a8ae8)) // the cross on the back doors
					t.rect(13, 10, 6, 3, c(0x3a8ae8))
				}
			}, nil)
		default:
			sedanView(b, front, sedanP{w: 32, roofTop: 7, roofBot: 17, bodyTop: 17, bodyBot: 31, body: vaCarCol(v), glass: vaGlassC})
		}
	})
	reg("taxi", 32, 38, func(b *pixBuf, v int, front bool) {
		sedanView(b, front, sedanP{w: 32, roofTop: 7, roofBot: 17, bodyTop: 17, bodyBot: 31, body: c(0xf2c21a), glass: vaGlassC, sign: true})
	})
	reg("homercar", 32, 38, func(b *pixBuf, v int, front bool) {
		sedanView(b, front, sedanP{w: 32, roofTop: 7, roofBot: 17, bodyTop: 17, bodyBot: 31, body: c(0xf08aa8), glass: vaGlassC})
	})
	reg("wackypink", 32, 38, func(b *pixBuf, v int, front bool) {
		sedanView(b, front, sedanP{w: 32, roofTop: 7, roofBot: 17, bodyTop: 17, bodyBot: 31, body: c(0xf860a8), glass: vaGlassC, stripe: true, trim: c(0xffe0f0)})
	})
	reg("delorean", 32, 36, func(b *pixBuf, v int, front bool) {
		sedanView(b, front, sedanP{w: 32, roofTop: 9, roofBot: 19, bodyTop: 19, bodyBot: 30, body: c(0xb8bcc6), glass: c(0x4a6080), wings: true})
		if !front {
			vaBuild(b, 0, func(t *pixBuf) {
				for y := 22; y < 26; y += 2 { // rear vents
					t.rect(5, y, 7, 1, c(0x3a3a44))
					t.rect(20, y, 7, 1, c(0x3a3a44))
				}
			}, nil)
		}
	})
	rlView := func(boxy bool) func(b *pixBuf, v int, front bool) {
		return func(b *pixBuf, v int, front bool) {
			body, trim := rlTeam(v)
			top, roof := 17, 9
			if boxy {
				top, roof = 13, 8
			}
			w := 32
			vvShadow(b, w)
			vaBuild(b, 0, func(t *pixBuf) {
				vaTyreFront(t, 0, 20, 5, 12) // fat wheels with a light rim
				vaTyreFront(t, w-5, 20, 5, 12)
				t.rect(1, 23, 3, 6, trim)
				t.rect(w-4, 23, 3, 6, trim)
				for y := top; y < 28; y++ {
					col := body
					switch {
					case y == top:
						col = shade(body, 1.3)
					case y >= 25:
						col = shade(body, 0.65)
					}
					vaRow(t, y, 5, w-6, col)
				}
				for y := roof; y < top; y++ {
					in := 8 + (top-y)/3
					vaRow(t, y, in, w-1-in, c(0x1a2030))
				}
				t.set(12, top-2, c(0x8ab0e0))
				t.set(13, top-3, c(0x8ab0e0))
				t.rect(7, top+2, w-14, 1, trim)
				t.rect(w/2-1, top, 2, 28-top-3, trim) // the racing stripe
				if front {
					t.rect(6, top+3, 4, 2, vaHeadC)
					t.rect(w-10, top+3, 4, 2, vaHeadC)
					t.rect(10, top+3, 2, 1, c(0x40e0f0))
					t.rect(w-12, top+3, 2, 1, c(0x40e0f0))
				} else {
					t.rect(6, top+3, 4, 2, vaTailC)
					t.rect(w-10, top+3, 4, 2, vaTailC)
					t.rect(w/2-2, 24, 4, 2, c(0x8a8e98))   // boost
					t.rect(3, roof-2, w-6, 2, c(0x1c1c24)) // the wing
					t.rect(3, roof-2, w-6, 1, trim)
					t.rect(8, roof, 2, top-roof, c(0x2a2a34))
					t.rect(w-10, roof, 2, top-roof, c(0x2a2a34))
				}
			}, nil)
		}
	}
	reg("octane", 32, 34, rlView(false))
	reg("fennec", 32, 34, rlView(true))
	reg("dominus", 32, 34, func(b *pixBuf, v int, front bool) {
		rlView(false)(b, v, front)
		vaBuild(b, 0, func(t *pixBuf) {
			t.rect(11, 12, 10, 2, c(0xf0508a)) // awning
			vaCone(t, 16, 10)
		}, nil)
	})
	reg("batmobile", 34, 34, func(b *pixBuf, v int, front bool) {
		k := c(0x1c1c26)
		w := 34
		vvShadow(b, w)
		vaBuild(b, 0, func(t *pixBuf) {
			vaTyreFront(t, 0, 22, 4, 11)
			vaTyreFront(t, w-4, 22, 4, 11)
			t.rect(2, 20, w-4, 11, k)
			t.rect(2, 20, w-4, 1, c(0x4a4a58))
			t.rect(8, 11, w-16, 10, c(0x24242e)) // canopy
			vaGlassPane(t, 10, 12, w-20, 6, c(0x2a3a58))
			if front {
				t.rect(5, 23, 6, 2, c(0xe8c030))
				t.rect(w-11, 23, 6, 2, c(0xe8c030))
				t.rect(12, 26, w-24, 3, c(0x2a2a34))
			} else {
				t.rect(0, 4, 5, 18, k) // the two fins
				t.rect(w-5, 4, 5, 18, k)
				t.rect(5, 23, 4, 3, c(0xff4a20))
				t.rect(w-9, 23, 4, 3, c(0xff4a20))
			}
		}, nil)
	})
	reg("wackymean", 34, 34, func(b *pixBuf, v int, front bool) {
		p := c(0x7a30c0)
		w := 34
		vvShadow(b, w)
		vaBuild(b, 0, func(t *pixBuf) {
			vaTyreFront(t, 0, 23, 4, 10)
			vaTyreFront(t, w-4, 23, 4, 10)
			t.rect(2, 20, w-4, 11, p)
			t.rect(2, 20, w-4, 1, c(0xb070f0))
			t.rect(10, 12, w-20, 9, shade(p, 0.85))
			t.rect(14, 6, 6, 6, c(0xf0c8a0)) // driver
			t.rect(w/2-1, 2, 2, 5, c(0x2a2a34))
			if front {
				t.rect(4, 24, 4, 2, vaHeadC)
				t.rect(w-8, 24, 4, 2, vaHeadC)
			} else {
				t.rect(8, 14, 3, 7, c(0x2a2a34)) // exhausts
				t.rect(w-11, 14, 3, 7, c(0x2a2a34))
				t.rect(4, 24, 4, 2, vaTailC)
				t.rect(w-8, 24, 4, 2, vaTailC)
			}
		}, nil)
	})
	reg("wackyboulder", 34, 38, func(b *pixBuf, v int, front bool) {
		g := c(0x8a8a90)
		w := 34
		vvShadow(b, w)
		vaBuild(b, 0, func(t *pixBuf) {
			vaTyreFront(t, 0, 24, 4, 10)
			vaTyreFront(t, w-4, 24, 4, 10)
			t.ellipse(17, 18, 14, 13, g)
			t.ellipse(15, 14, 9, 8, shade(g, 1.15))
			if front {
				vaGlassPane(t, 11, 8, 12, 6, vaGlassC)
			} else {
				t.rect(11, 8, 12, 5, shade(vaGlassC, 0.8))
			}
		}, nil)
	})
	reg("van", 34, 38, func(b *pixBuf, v int, front bool) {
		cols := [][2]uint32{{0xf2f2f6, 0x2a5ac8}, {0x2f63c8, 0xf2f2f6}, {0xc83a3a, 0xf0c020}}[v%3]
		boxView(b, front, boxP{w: 34, top: 8, bodyBot: 31, glassTop: 11, glassBot: 21, body: c(cols[0]), stripe: c(cols[1])})
	})
	reg("icetruck", 36, 46, func(b *pixBuf, v int, front bool) {
		boxView(b, front, boxP{w: 36, top: 10, bodyBot: 31, glassTop: 13, glassBot: 22, body: c(0xf6e8d0), stripe: c(0xf08aa8), cone: true})
	})
	reg("boxtruck", 36, 44, func(b *pixBuf, v int, front bool) {
		cab := c([]uint32{0x2f63c8, 0x3a9a48, 0xb8bcc4, 0xc8ccd4}[v%4])
		box := c([]uint32{0xe8ecf0, 0xb8302a, 0x2f58b8, 0x8a9098}[v%4])
		boxView(b, front, boxP{w: 36, top: 4, bodyBot: 31, glassTop: 15, glassBot: 23, body: cab, cab: cab, box: box, boxBehind: true, ribs: v%4 != 2})
	})
	reg("schoolbus", 38, 42, func(b *pixBuf, v int, front bool) {
		body, band := c(0xf0b818), c(0x2a2a30)
		if v%2 == 1 {
			body, band = c(0x46a050), c(0xf0e8b0)
		}
		boxView(b, front, boxP{w: 38, top: 6, bodyBot: 31, glassTop: 9, glassBot: 20, body: body, stripe: band, bus: true})
	})
	reg("monstertruck", 62, 64, func(b *pixBuf, v int, front bool) {
		body := c([]uint32{0x28a840, 0xd02828, 0x2858d0}[v%3])
		w := 62
		b.ellipse(31, 61, 29, 3.5, color.RGBA{0, 0, 0, 90})
		vaBuild(b, 0, func(t *pixBuf) {
			t.rect(18, 40, 5, 12, c(0x5a5a64)) // axles
			t.rect(w-23, 40, 5, 12, c(0x5a5a64))
			for _, cx := range []int{7, w - 8} { // the huge wheels, seen from the front
				vaTyreFront(t, cx-6, 36, 12, 27)
				t.rect(cx-2, 42, 4, 15, c(0x3a3a44))
			}
			t.rect(11, 26, w-22, 14, body)
			t.rect(11, 26, w-22, 2, shade(body, 1.3))
			t.rect(11, 37, w-22, 3, shade(body, 0.65))
			t.rect(17, 14, w-34, 13, shade(body, 0.92))
			if front {
				vaGlassPane(t, 19, 16, w-38, 8, vaGlassC)
				t.rect(13, 29, 7, 5, vaHeadC)
				t.rect(w-20, 29, 7, 5, vaHeadC)
			} else {
				t.rect(19, 16, w-38, 8, shade(vaGlassC, 0.8))
				t.rect(13, 29, 7, 5, vaTailC)
				t.rect(w-20, 29, 7, 5, vaTailC)
			}
			for i := 0; i < 3; i++ { // flame decals along the doors
				t.rect(22+i*7, 33, 5, 2, c(0xffb020))
			}
		}, nil)
	})
	reg("motorbike", 18, 30, func(b *pixBuf, v int, front bool) {
		cc := c([]uint32{0x8a3ad0, 0xd02a2a}[v%2])
		b.ellipse(9, 28, 7, 2.5, color.RGBA{0, 0, 0, 90})
		vaBuild(b, 0, func(t *pixBuf) {
			vaTyreFront(t, 7, 18, 4, 12) // the wheel, thin from this side
			t.rect(5, 10, 8, 10, cc)
			t.rect(5, 10, 8, 2, shade(cc, 1.4))
			t.rect(2, 8, 14, 2, c(0x3a3a46)) // handlebars
			if front {
				t.rect(6, 6, 6, 3, vaGlassC) // windscreen
				t.rect(7, 12, 4, 3, vaHeadC)
			} else {
				t.rect(7, 12, 4, 3, vaTailC)
				t.rect(6, 6, 6, 4, c(0x2a2a34)) // seat
			}
		}, nil)
	})
}

// ---- cache ------------------------------------------------------------------

var vehViewCache = map[string]PropImg{}

// vehViewPix renders one view like any other prop sprite (scaled, detailed, outlined, stylised).
func vehViewPix(name string, v int, front bool) *pixBuf {
	d := vehViewDefs[name]
	if d == nil {
		return nil
	}
	b := newPix(d.w+2, d.h+2)
	b.ox, b.oy = 1, 1
	d.draw(b, v, front)
	seed := int64(len(name)*31+v) + int64(name[0])*977
	if front {
		seed++
	}
	return stylizeSprite(b.scaled(wk).detail(seed, 0.035).outlinedSel(0.45))
}

// vehicleView returns the front (the nose toward the camera) or rear picture of a vehicle,
// anchored at the middle of its base like the side sprites. ok is false for a vehicle without one.
func vehicleView(name string, v int, front bool) (PropImg, bool) {
	d := vehViewDefs[name]
	if d == nil {
		return PropImg{}, false
	}
	if k := propKinds[name]; k != nil && k.Vars > 1 {
		v %= k.Vars
	} else {
		v = 0
	}
	key := name + "#" + string(rune('0'+v))
	if front {
		key += "f"
	} else {
		key += "b"
	}
	if pi, ok := vehViewCache[key]; ok {
		return pi, true
	}
	sb := vehViewPix(name, v, front)
	pi := PropImg{img: sb.img(), ox: float64(d.w/2+1) * wk, oy: float64(d.h+1) * wk}
	vehViewCache[key] = pi
	return pi, true
}

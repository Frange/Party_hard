package main

import (
	"image/color"
)

// ---------------------------------------------------------------------------
// Front and back views of the road vehicles. On a vertical road the cars that
// come down are drawn from the front and the ones that go up from behind,
// instead of the side sprite turned a quarter. They use the same palettes and
// proportions as the side views (props.go, props5.go, props7.go).
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

// vvLamps puts the headlights (front) or the tail lights (rear) and the plate.
func vvLamps(b *pixBuf, w, y int, front bool) {
	col := vvTail
	if front {
		col = vvLight
	}
	b.rect(3, y, 4, 3, col)
	b.rect(w-7, y, 4, 3, col)
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
}

// sedanView: any car-shaped vehicle from the front or from behind.
func sedanView(b *pixBuf, front bool, p sedanP) {
	w := p.w
	vvShadow(b, w)
	vvWheels(b, w, p.bodyBot-8, p.bodyBot+3)
	b.rect(2, p.bodyTop, w-4, p.bodyBot-p.bodyTop, p.body)
	b.rect(2, p.bodyTop, w-4, 1, shade(p.body, 1.25))
	b.rect(2, p.bodyBot-3, w-4, 3, shade(p.body, 0.7))
	// cabin and glass
	b.rect(5, p.roofTop, w-10, p.roofBot-p.roofTop, shade(p.body, 0.9))
	b.rect(5, p.roofTop, w-10, 1, shade(p.body, 1.25))
	gl := p.glass
	if front {
		b.rect(7, p.roofTop+2, w-14, p.roofBot-p.roofTop-3, gl)
		b.rect(7, p.roofTop+2, w-14, 1, shade(gl, 1.3))
	} else {
		b.rect(8, p.roofTop+2, w-16, p.roofBot-p.roofTop-4, shade(gl, 0.82))
	}
	if p.stripe {
		b.rect(w/2-2, p.bodyTop, 4, p.bodyBot-p.bodyTop-3, p.trim)
	}
	vvLamps(b, w, p.bodyTop+4, front)
	if front {
		b.rect(9, p.bodyTop+8, w-18, 3, vvGrille)
	} else {
		b.rect(w-10, p.bodyBot-2, 3, 2, vvDark)
	}
	vvPlateAt(b, w, p.bodyBot-5)
	if p.bar {
		b.rect(w/2-8, p.roofTop-3, 16, 3, c(0x2a2a34))
		b.rect(w/2-8, p.roofTop-3, 8, 3, c(0xe03030))
		b.rect(w/2, p.roofTop-3, 8, 3, c(0x3060ff))
	}
	if p.sign {
		b.rect(w/2-7, p.roofTop-4, 14, 5, vvPlate)
		b.rect(w/2-5, p.roofTop-3, 10, 2, vvGrille)
	}
	if p.wings {
		b.rect(1, p.roofTop-3, 5, 6, c(0x6a6a74))
		b.rect(w-6, p.roofTop-3, 5, 6, c(0x6a6a74))
	}
	if p.cone {
		b.rect(w/2-1, p.roofTop-4, 3, 4, c(0xd8a860))
		b.ellipse(float64(w)/2+0.5, float64(p.roofTop)-6, 5, 4, c(0xf8c8d8))
		b.ellipse(float64(w)/2+0.5, float64(p.roofTop)-9, 3, 3, c(0xe85088))
	}
}

// boxView: vans, trucks and the bus.
type boxP struct {
	w, top, bodyBot, glassTop, glassBot int
	body, cab                           color.RGBA
	boxBehind                           bool // a taller box behind a lower cab (the box truck)
	box                                 color.RGBA
	stripe                              color.RGBA
	cone                                bool
	bus                                 bool
}

func boxView(b *pixBuf, front bool, p boxP) {
	w := p.w
	vvShadow(b, w)
	vvWheels(b, w, p.bodyBot-8, p.bodyBot+3)
	if p.boxBehind {
		// the white box rises above the cab; from behind it is all box
		b.rect(1, p.top, w-2, p.bodyBot-p.top, p.box)
		b.rect(1, p.top, w-2, 2, shade(p.box, 1.2))
		b.rect(1, p.bodyBot-4, w-2, 4, shade(p.box, 0.72))
		if front {
			b.rect(2, p.glassTop-3, w-4, p.bodyBot-p.glassTop+3, p.cab)
			b.rect(2, p.glassTop-3, w-4, 1, shade(p.cab, 1.3))
		} else {
			b.rect(w/2, p.top+2, 1, p.bodyBot-p.top-6, shade(p.box, 0.6)) // door gap
			b.rect(3, p.top+5, w-6, 1, shade(p.box, 0.85))
		}
	} else {
		b.rect(2, p.top, w-4, p.bodyBot-p.top, p.body)
		b.rect(2, p.top, w-4, 1, shade(p.body, 1.25))
		b.rect(2, p.bodyBot-3, w-4, 3, shade(p.body, 0.7))
	}
	if p.stripe.A > 0 {
		b.rect(2, p.bodyBot-9, w-4, 2, p.stripe)
	}
	if front {
		b.rect(4, p.glassTop, w-8, p.glassBot-p.glassTop, vvGlass)
		b.rect(4, p.glassTop, w-8, 1, c(0xa8c8e0))
		if p.bus {
			b.rect(w/2, p.glassTop, 1, p.glassBot-p.glassTop, shade(p.body, 0.7))
		}
		vvLamps(b, w, p.bodyBot-9, true)
		b.rect(9, p.bodyBot-5, w-18, 2, vvGrille)
	} else {
		if !p.boxBehind {
			b.rect(5, p.glassTop, w-10, p.glassBot-p.glassTop-2, shade(vvGlass, 0.8))
			if p.bus {
				b.rect(w/2-4, p.glassBot, 8, p.bodyBot-p.glassBot-4, c(0x2a2a34)) // emergency door
			}
		}
		vvLamps(b, w, p.bodyBot-9, false)
		vvPlateAt(b, w, p.bodyBot-5)
	}
	if p.cone {
		b.rect(w/2-1, p.top-4, 3, 4, c(0xd8a860))
		b.ellipse(float64(w)/2+0.5, float64(p.top)-6, 5, 4, c(0xf8c8d8))
		b.ellipse(float64(w)/2+0.5, float64(p.top)-9, 3, 3, c(0xe85088))
	}
}

func init() {
	reg := func(name string, w, h int, f func(b *pixBuf, v int, front bool)) {
		vehViewDefs[name] = &vehViewDef{w: w, h: h, draw: f}
	}
	carCols := []uint32{0xc03a3a, 0x3a6ac0, 0x3aa05a, 0xe0c040, 0xf2f2f6, 0xf6f6fa}
	reg("car", 32, 36, func(b *pixBuf, v int, front bool) {
		col := carCols[v%len(carCols)]
		sedanView(b, front, sedanP{w: 32, roofTop: 5, roofBot: 18, bodyTop: 16, bodyBot: 31, body: c(col), glass: vvGlass,
			bar: v == 4, stripe: v == 5, trim: c(0xd02828)})
	})
	reg("taxi", 32, 38, func(b *pixBuf, v int, front bool) {
		sedanView(b, front, sedanP{w: 32, roofTop: 5, roofBot: 18, bodyTop: 16, bodyBot: 31, body: c(0xf0c818), glass: vvGlass, sign: true})
	})
	reg("homercar", 32, 38, func(b *pixBuf, v int, front bool) {
		sedanView(b, front, sedanP{w: 32, roofTop: 5, roofBot: 18, bodyTop: 16, bodyBot: 31, body: c(0xf08aa8), glass: vvGlass})
	})
	reg("wackypink", 32, 38, func(b *pixBuf, v int, front bool) {
		sedanView(b, front, sedanP{w: 32, roofTop: 5, roofBot: 18, bodyTop: 16, bodyBot: 31, body: c(0xf860a8), glass: vvGlass})
		b.rect(11, 6, 10, 3, c(0xffe0f0))
	})
	reg("delorean", 32, 36, func(b *pixBuf, v int, front bool) {
		sedanView(b, front, sedanP{w: 32, roofTop: 8, roofBot: 19, bodyTop: 18, bodyBot: 30, body: c(0xb8bcc6), glass: c(0x4a6080), wings: true})
		if !front {
			b.rect(4, 22, 8, 4, c(0x3a3a44)) // rear vents
			b.rect(20, 22, 8, 4, c(0x3a3a44))
		}
	})
	rlView := func(boxy bool) func(b *pixBuf, v int, front bool) {
		return func(b *pixBuf, v int, front bool) {
			body, trim := rlTeam(v)
			top, roof := 17, 9
			if boxy {
				top, roof = 12, 12
			}
			w := 32
			vvShadow(b, w)
			// wide wheels with a rim colour
			for _, x := range []int{0, w - 5} {
				b.rect(x, 21, 5, 11, vvDark)
				b.rect(x+1, 23, 3, 7, trim)
			}
			b.rect(3, top, w-6, 28-top, body)
			b.rect(3, top, w-6, 1, shade(body, 1.35))
			b.rect(3, 25, w-6, 3, shade(body, 0.6))
			b.rect(8, roof, w-16, 9, shade(body, 0.92))
			b.rect(10, roof+2, w-20, 5, c(0x182034))
			b.rect(5, 21, w-10, 2, trim)
			if front {
				vvLamps(b, w, top+3, true)
			} else {
				vvLamps(b, w, top+3, false)
				b.rect(3, 9, w-6, 2, trim) // rear wing
				b.rect(5, 11, 2, 6, shade(body, 0.7))
				b.rect(w-7, 11, 2, 6, shade(body, 0.7))
			}
		}
	}
	reg("octane", 32, 34, rlView(false))
	reg("fennec", 32, 34, rlView(true))
	reg("dominus", 32, 34, func(b *pixBuf, v int, front bool) {
		rlView(false)(b, v, front)
		b.rect(11, 12, 10, 2, c(0xf0508a)) // awning
		b.rect(15, 6, 3, 4, c(0xd8a860))
		b.ellipse(16.5, 4, 5, 4, c(0xf8c8d8))
		b.ellipse(16.5, 1, 3, 3, c(0xe85088))
	})
	reg("batmobile", 34, 34, func(b *pixBuf, v int, front bool) {
		k := c(0x14141c)
		w := 34
		vvShadow(b, w)
		vvWheels(b, w, 22, 33)
		b.rect(2, 20, w-4, 11, k)
		b.rect(2, 20, w-4, 1, c(0x4a4a58))
		b.rect(8, 11, w-16, 10, c(0x24242e)) // canopy
		b.rect(10, 12, w-20, 6, c(0x2a3a58))
		if front {
			b.rect(5, 23, 6, 2, c(0xe8c030))
			b.rect(w-11, 23, 6, 2, c(0xe8c030))
			b.rect(12, 26, w-24, 3, c(0x2a2a34))
		} else {
			b.rect(0, 4, 5, 18, k) // the two fins
			b.rect(w-5, 4, 5, 18, k)
			b.rect(5, 23, 4, 3, c(0xff4a20))
			b.rect(w-9, 23, 4, 3, c(0xff4a20))
		}
	})
	reg("wackymean", 34, 34, func(b *pixBuf, v int, front bool) {
		p := c(0x7a30c0)
		w := 34
		vvShadow(b, w)
		vvWheels(b, w, 23, 33)
		b.rect(2, 20, w-4, 11, p)
		b.rect(2, 20, w-4, 1, c(0xb070f0))
		b.rect(10, 12, w-20, 9, shade(p, 0.85))
		b.rect(14, 6, 6, 6, c(0xf0c8a0)) // driver
		b.rect(w/2-1, 2, 2, 5, c(0x2a2a34))
		if front {
			vvLamps(b, w, 24, true)
		} else {
			b.rect(8, 14, 3, 7, c(0x2a2a34)) // exhausts
			b.rect(w-11, 14, 3, 7, c(0x2a2a34))
			vvLamps(b, w, 24, false)
		}
	})
	reg("wackyboulder", 34, 38, func(b *pixBuf, v int, front bool) {
		g := c(0x8a8a90)
		w := 34
		vvShadow(b, w)
		vvWheels(b, w, 24, 34)
		b.ellipse(17, 18, 15, 14, g)
		b.ellipse(15, 14, 9, 8, shade(g, 1.15))
		if front {
			b.rect(11, 8, 12, 6, vvGlass)
		} else {
			b.rect(11, 8, 12, 5, shade(vvGlass, 0.8))
		}
	})
	reg("van", 34, 38, func(b *pixBuf, v int, front bool) {
		body := c([]uint32{0xf2f2f6, 0x3a6ac0, 0xc03a3a}[v%3])
		boxView(b, front, boxP{w: 34, top: 8, bodyBot: 31, glassTop: 11, glassBot: 22, body: body})
	})
	reg("icetruck", 36, 46, func(b *pixBuf, v int, front bool) {
		boxView(b, front, boxP{w: 36, top: 10, bodyBot: 31, glassTop: 13, glassBot: 23, body: c(0xf6e8d0), stripe: c(0xf08aa8), cone: true})
	})
	reg("boxtruck", 36, 44, func(b *pixBuf, v int, front bool) {
		box := c([]uint32{0xe8e8ec, 0x4a7ac8}[v%2])
		boxView(b, front, boxP{w: 36, top: 4, bodyBot: 31, glassTop: 15, glassBot: 24, body: c(0xc03a3a), cab: c(0xc03a3a), box: box, boxBehind: true})
	})
	reg("schoolbus", 38, 42, func(b *pixBuf, v int, front bool) {
		boxView(b, front, boxP{w: 38, top: 6, bodyBot: 31, glassTop: 9, glassBot: 20, body: c(0xf0b818), stripe: c(0x2a2a30), bus: true})
	})
	reg("monstertruck", 62, 64, func(b *pixBuf, v int, front bool) {
		body := c([]uint32{0x28a840, 0xd02828, 0x2858d0}[v%3])
		w := 62
		b.ellipse(31, 61, 29, 3.5, color.RGBA{0, 0, 0, 90})
		b.rect(11, 26, w-22, 14, body)
		b.rect(11, 26, w-22, 2, shade(body, 1.3))
		b.rect(11, 37, w-22, 3, shade(body, 0.65))
		b.rect(17, 14, w-34, 13, shade(body, 0.92))
		if front {
			b.rect(19, 16, w-38, 8, c(0x7aa0d0))
			b.rect(13, 29, 7, 5, c(0xfff0b0))
			b.rect(w-20, 29, 7, 5, c(0xfff0b0))
		} else {
			b.rect(19, 16, w-38, 8, shade(c(0x7aa0d0), 0.8))
			b.rect(13, 29, 7, 5, c(0xe03030))
			b.rect(w-20, 29, 7, 5, c(0xe03030))
		}
		for i := 0; i < 3; i++ { // flame decals along the doors
			b.rect(22+i*7, 33, 5, 2, c(0xffb020))
		}
		b.rect(18, 40, 5, 12, c(0x5a5a64)) // axles
		b.rect(w-23, 40, 5, 12, c(0x5a5a64))
		for _, cx := range []float64{7, float64(w) - 8} { // the huge wheels, seen from the front
			b.rect(int(cx)-6, 36, 12, 27, c(0x121216))
			b.rect(int(cx)-4, 38, 8, 23, c(0x2a2a30))
			b.rect(int(cx)-2, 42, 4, 15, c(0x3a3a44))
		}
	})
	reg("motorbike", 18, 30, func(b *pixBuf, v int, front bool) {
		cc := c([]uint32{0x8a3ad0, 0xc82a2a}[v%2])
		b.ellipse(9, 28, 7, 2.5, color.RGBA{0, 0, 0, 90})
		b.rect(7, 18, 4, 12, vvDark) // the wheel, thin from this side
		b.rect(8, 18, 1, 12, c(0x6a6a76))
		b.rect(5, 10, 8, 10, cc)
		b.rect(5, 10, 8, 2, shade(cc, 1.4))
		b.rect(2, 8, 14, 2, c(0x3a3a46)) // handlebars
		if front {
			b.rect(7, 12, 4, 3, vvLight)
		} else {
			b.rect(7, 12, 4, 3, vvTail)
			b.rect(6, 6, 6, 5, c(0x2a2a34)) // seat
		}
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

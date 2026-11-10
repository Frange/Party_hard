package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The stretch limousine that brings late guests (arrivals.go). One procedural
// side view, facing left like every other car of the lane (nose = headlight on
// the left): a very long, low body (about 4.7:1), short hood, raked windshield,
// four dark-blue side windows with a glint, door cuts, a light strip along the
// flank, chrome bumpers, head and tail lights, two axles with light rims, and a
// shine on the roof. Two paints: midnight navy (the usual one) and silver.
// Every place that shows the limousine goes through limoImage / drawLimoAt.
// ---------------------------------------------------------------------------

const (
	limoImgW, limoImgH = 144, 36
	limoFront          = 36 // image column that sits on the vehicle x (the nose is at v.x-34, like the old car)
	limoBase           = 33 // image row that sits on the vehicle y (bottom of the tyres)
	limoNose, limoTail = 2, 141
	limoRoofY          = 5
	limoWheelY         = 27
)

// limoWheelX: the two axles (front, rear) in image columns.
var limoWheelX = [2]int{22, 121}

// limoDoorX: the centre of each passenger door, in image columns (between the cuts).
var limoDoorX = []int{50, 72, 93, 114}

// limoSeamX: the vertical door cuts.
var limoSeamX = []int{40, 61, 82, 103, 124}

type limoPaint struct {
	body, dark, light, seam, strip, glass, glassHi, chrome color.RGBA
}

func limoPaintFor(silver bool) limoPaint {
	if silver {
		return limoPaint{
			body: hx(0xa6aec0), dark: hx(0x6e7488), light: hx(0xd6dcea), seam: hx(0x4a5062),
			strip: hx(0xf0f4fc), glass: hx(0x1c2a5c), glassHi: hx(0x5874bc), chrome: hx(0xe2e6ee),
		}
	}
	return limoPaint{
		body: hx(0x1c2442), dark: hx(0x10152a), light: hx(0x3a4672), seam: hx(0x090c18),
		strip: hx(0x9ab2e6), glass: hx(0x3456aa), glassHi: hx(0x86a8e8), chrome: hx(0xb4b8c6),
	}
}

// limoGlassL / limoGlassR: the glass edges of row y (windshield on the left, rear window on the right).
func limoGlassL(y int) int { return 37 - (y-limoRoofY-1)*11/8 }
func limoGlassR(y int) int { return 126 + (y-limoRoofY-1)*6/8 }

// genLimo paints the limousine (no shadow, no outline) in design pixels.
func genLimoBody(silver bool) *pixBuf {
	p := limoPaintFor(silver)
	b := newPix(limoImgW, limoImgH)
	// lower body: a long slab, the nose and the tail rounded off a little
	b.rect(4, 15, 136, 12, p.body)
	b.rect(limoNose, 17, 2, 9, p.body)
	b.rect(140, 16, 2, 10, p.body)
	b.rect(4, 15, 22, 1, p.light) // short hood, catching the light
	b.rect(26, 15, 114, 1, shade(p.light, 0.9))
	b.rect(4, 24, 136, 3, p.dark) // rocker panel
	// greenhouse: raked windshield, long roof, sloped rear window
	for y := limoRoofY + 1; y <= 14; y++ {
		l, r := limoGlassL(y), limoGlassR(y)
		b.rect(l, y, r-l+1, 1, p.body)
	}
	b.rect(limoGlassL(limoRoofY+1)+1, limoRoofY, limoGlassR(limoRoofY+1)-limoGlassL(limoRoofY+1)-1, 1, p.light)
	b.rect(48, limoRoofY, 14, 1, shade(p.light, 1.35)) // roof shine
	b.rect(84, limoRoofY, 7, 1, shade(p.light, 1.2))
	b.rect(limoGlassL(14), 14, limoGlassR(14)-limoGlassL(14)+1, 1, p.chrome) // chrome sill under the glass
	// glass: windshield, then four side windows split by body-coloured pillars
	win := [][2]int{{0, 38}, {41, 59}, {62, 80}, {83, 101}, {104, 200}}
	for i, w := range win {
		for y := 8; y <= 13; y++ {
			x0, x1 := w[0], w[1]
			if i == 0 {
				x0 = limoGlassL(y) + 2
			}
			if i == len(win)-1 {
				x1 = limoGlassR(y) - 2
			}
			if x1 < x0 {
				continue
			}
			col := p.glass
			if y == 8 {
				col = shade(p.glass, 1.25)
			}
			if y == 13 {
				col = shade(p.glass, 0.8)
			}
			b.rect(x0, y, x1-x0+1, 1, col)
			// the glint: a short diagonal streak near the front of each pane
			gx := x0 + 3 + (13 - y)
			if gx+1 <= x1 && y >= 9 {
				b.rect(gx, y, 2, 1, p.glassHi)
			}
		}
	}
	// door cuts and handles
	for i, sx := range limoSeamX {
		b.rect(sx, 15, 1, 9, p.seam)
		if i < len(limoSeamX)-1 {
			b.rect(sx+4, 17, 3, 1, p.chrome)
		}
	}
	// the light strip along the flank
	b.rect(5, 19, 134, 1, p.strip)
	b.rect(5, 20, 134, 1, shade(p.body, 0.75))
	// front: headlight, grille, bumper
	b.rect(limoNose, 17, 3, 3, hx(0xfff0b0))
	b.set(limoNose, 17, hx(0xffffff))
	b.rect(limoNose, 20, 2, 2, p.seam)
	b.rect(limoNose-1, 22, 5, 4, p.chrome)
	b.rect(limoNose-1, 25, 5, 1, shade(p.chrome, 0.7))
	// rear: tail light and bumper
	b.rect(139, 16, 3, 4, hx(0xd42a2a))
	b.set(141, 16, hx(0xff7a6a))
	b.rect(138, 22, 5, 4, p.chrome)
	b.rect(138, 25, 5, 1, shade(p.chrome, 0.7))
	// two axles: dark wheel arches, black tyres, light rims
	for _, wx := range limoWheelX {
		fx := float64(wx) + 0.5
		for y := 19; y <= 26; y++ { // the arch is cut only into the body
			for x := wx - 8; x <= wx+8; x++ {
				dx, dy := (float64(x)+0.5-fx)/7.5, (float64(y)+0.5-27)/7
				if dx*dx+dy*dy <= 1 {
					b.set(x, y, p.seam)
				}
			}
		}
		b.ellipse(fx, limoWheelY+0.5, 5.6, 5.6, hx(0x141418))
		b.ellipse(fx, limoWheelY+0.5, 3, 3, hx(0xc8ccd6))
		b.rect(wx, limoWheelY, 1, 1, hx(0x5a5e6a))
		b.set(wx-1, limoWheelY-1, hx(0xf0f2f6))
	}
	return b
}

// genLimo: the finished sprite: dark outline, soft shadow under it and the
// shared muted finish of the props.
func genLimo(silver bool) *pixBuf {
	car := genLimoBody(silver).outlined(hx(0x07070c))
	out := newPix(limoImgW, limoImgH)
	out.ellipse(float64(limoImgW)/2, float64(limoBase)+0.5, 66, 2.5, color.RGBA{0, 0, 0, 90})
	for i := 0; i < len(car.p); i += 4 {
		if car.p[i+3] != 0 {
			copy(out.p[i:i+4], car.p[i:i+4])
		}
	}
	return stylizeSprite(out)
}

var limoImgs [2]*ebiten.Image

// limoImage: the cached limousine sprite and its anchor (v.x, v.y).
func limoImage(silver bool) PropImg {
	k := 0
	if silver {
		k = 1
	}
	if limoImgs[k] == nil {
		limoImgs[k] = genLimo(silver).img()
	}
	return PropImg{img: limoImgs[k], ox: limoFront, oy: limoBase}
}

// limoSilver: which paint a limousine wears (Vehicle.pad is free on ground vehicles).
func limoSilver(v *Vehicle) bool { return v.pad == 1 }

// drawLimoAt draws the limousine with its anchor on the screen point (x, y).
func drawLimoAt(dst *ebiten.Image, x, y float64, silver bool) {
	pi := limoImage(silver)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(x-pi.ox), math.Round(y-pi.oy))
	dst.DrawImage(pi.img, op)
}

func (g *Game) drawLimo(dst *ebiten.Image, v *Vehicle) {
	drawLimoAt(dst, g.wx(v.x), g.wy(v.y), limoSilver(v))
}

// limoDoorWorld: where the guest behind door k steps out (world x offset from v.x).
func limoDoorWorld(k int) float64 {
	return float64(limoDoorX[k%len(limoDoorX)] - limoFront)
}

// limoCentre: world x of the middle of the limousine (for hits against it).
func limoCentre(v *Vehicle) float64 {
	return v.x + float64((limoNose+limoTail)/2-limoFront)
}

// limoHalfLen: half the length of the limousine in world pixels.
const limoHalfLen = float64(limoTail-limoNose) / 2

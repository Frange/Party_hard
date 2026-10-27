package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

func rgb(hex uint32) color.RGBA {
	return color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255}
}

// rgba returns a premultiplied colour with alpha, as Ebitengine expects.
func rgba(hex uint32, a uint8) color.RGBA {
	c := rgb(hex)
	c.R = uint8(uint16(c.R) * uint16(a) / 255)
	c.G = uint8(uint16(c.G) * uint16(a) / 255)
	c.B = uint8(uint16(c.B) * uint16(a) / 255)
	c.A = a
	return c
}

func shade(c color.RGBA, f float64) color.RGBA {
	cl := func(v float64) uint8 {
		if v > 255 {
			return 255
		}
		if v < 0 {
			return 0
		}
		return uint8(v)
	}
	return color.RGBA{cl(float64(c.R) * f), cl(float64(c.G) * f), cl(float64(c.B) * f), c.A}
}

func mixc(a, b color.RGBA, t float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 255}
}

// ---------------------------------------------------------------------------
// CPU pixel canvas
// ---------------------------------------------------------------------------

type pixBuf struct {
	w, h, ox, oy int
	p            []byte
	pid          []uint8 // optional body-part id per pixel
	cur          uint8
}

func newPix(w, h int) *pixBuf { return &pixBuf{w: w, h: h, p: make([]byte, w*h*4)} }

func (b *pixBuf) set(x, y int, c color.RGBA) {
	x += b.ox
	y += b.oy
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return
	}
	i := (y*b.w + x) * 4
	b.p[i], b.p[i+1], b.p[i+2], b.p[i+3] = c.R, c.G, c.B, c.A
	if b.pid != nil {
		b.pid[y*b.w+x] = b.cur
	}
}

func (b *pixBuf) get(x, y int) color.RGBA {
	x += b.ox
	y += b.oy
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return color.RGBA{}
	}
	i := (y*b.w + x) * 4
	return color.RGBA{b.p[i], b.p[i+1], b.p[i+2], b.p[i+3]}
}

func (b *pixBuf) rect(x, y, w, h int, c color.RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			b.set(xx, yy, c)
		}
	}
}

func (b *pixBuf) ellipse(cx, cy, rx, ry float64, c color.RGBA) {
	for y := int(math.Floor(cy - ry)); y <= int(math.Ceil(cy+ry)); y++ {
		for x := int(math.Floor(cx - rx)); x <= int(math.Ceil(cx+rx)); x++ {
			dx := (float64(x) + 0.5 - cx) / rx
			dy := (float64(y) + 0.5 - cy) / ry
			if dx*dx+dy*dy <= 1 {
				b.set(x, y, c)
			}
		}
	}
}

func (b *pixBuf) line(x0, y0, x1, y1 int, c color.RGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	e := dx + dy
	for {
		b.set(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * e
		if e2 >= dy {
			e += dy
			x0 += sx
		}
		if e2 <= dx {
			e += dx
			y0 += sy
		}
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// cube draws a box seen from slightly above: a lit top face of depth d over a front face.
func (b *pixBuf) cube(x, y, w, h, d int, top, front color.RGBA) {
	b.rect(x, y, w, d, top)
	b.rect(x, y+d, w, h-d, front)
	b.rect(x, y+d, w, 1, shade(front, 1.18))
	b.rect(x+w-1, y+d, 1, h-d, shade(front, 0.78))
	b.rect(x, y, w, 1, shade(top, 1.15))
}

func (b *pixBuf) outlined(c color.RGBA) *pixBuf {
	o := &pixBuf{w: b.w, h: b.h, p: append([]byte(nil), b.p...)}
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				continue
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= b.w || ny >= b.h {
					continue
				}
				if b.p[(ny*b.w+nx)*4+3] != 0 {
					i := (y*b.w + x) * 4
					o.p[i], o.p[i+1], o.p[i+2], o.p[i+3] = c.R, c.G, c.B, c.A
					break
				}
			}
		}
	}
	return o
}

func (b *pixBuf) rotCW() *pixBuf {
	o := newPix(b.h, b.w)
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			si := (y*b.w + x) * 4
			di := (x*o.w + (b.h - 1 - y)) * 4
			copy(o.p[di:di+4], b.p[si:si+4])
		}
	}
	return o
}

func (b *pixBuf) img() *ebiten.Image {
	im := ebiten.NewImage(b.w, b.h)
	im.WritePixels(b.p)
	return im
}

func (b *pixBuf) toRGBA() *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, b.w, b.h))
	copy(im.Pix, b.p)
	return im
}

var outlineCol = rgb(0x17111f)

type Dir int

const (
	DirDown Dir = iota
	DirUp
	DirLeft
	DirRight
)

func dirVec(d Dir) Vec {
	switch d {
	case DirUp:
		return Vec{0, -1}
	case DirLeft:
		return Vec{-1, 0}
	case DirRight:
		return Vec{1, 0}
	}
	return Vec{0, 1}
}

// ---------------------------------------------------------------------------
// Floor textures (32x32, tiled)
// ---------------------------------------------------------------------------

func genFloor(kind string, variant int) *pixBuf {
	b := newPix(32, 32)
	rng := rand.New(rand.NewSource(int64(len(kind)*97+int(kind[0])) + int64(variant)*7919))
	fill := func(c color.RGBA) { b.rect(0, 0, 32, 32, c) }
	noise := func(n int, c color.RGBA) {
		for i := 0; i < n; i++ {
			b.set(rng.Intn(32), rng.Intn(32), c)
		}
	}
	switch kind {
	case "wood":
		fill(rgb(0x7a5333))
		for y := 0; y < 32; y += 8 {
			b.rect(0, y, 32, 1, rgb(0x5e3e26))
			b.rect(0, y+1, 32, 1, rgb(0x8a6040))
			off := (y / 8 * 11) % 32
			b.rect(off, y+2, 1, 6, rgb(0x5e3e26))
			for i := 0; i < 5; i++ {
				b.set(rng.Intn(32), y+2+rng.Intn(6), rgb(0x6a4730))
			}
		}
	case "deck":
		fill(rgb(0xa97c50))
		for x := 0; x < 32; x += 8 {
			b.rect(x, 0, 1, 32, rgb(0x7a5636))
			b.rect(x+1, 0, 1, 32, rgb(0xb98c5e))
			off := (x / 8 * 13) % 32
			b.rect(x+2, off, 6, 1, rgb(0x7a5636))
		}
		noise(30, rgb(0x94683f))
	case "tile":
		for y := 0; y < 32; y += 16 {
			for x := 0; x < 32; x += 16 {
				col := rgb(0xdccfae)
				if (x/16+y/16)%2 == 1 {
					col = rgb(0xc4b48e)
				}
				b.rect(x, y, 16, 16, col)
				b.rect(x, y, 16, 1, shade(col, 1.08))
			}
		}
	case "carpet":
		fill(rgb(0x6b2a3c))
		noise(160, rgb(0x7a3548))
		noise(100, rgb(0x5a2030))
	case "grass":
		fill(rgb(0x2f6a35))
		noise(70, rgb(0x3b7f41))
		noise(50, rgb(0x265a2c))
		for i := 0; i < 8; i++ {
			x, y := rng.Intn(30), rng.Intn(30)
			b.set(x, y, rgb(0x4a9650))
			b.set(x, y+1, rgb(0x4a9650))
		}
	case "pave":
		fill(rgb(0x8d8d98))
		b.rect(0, 0, 32, 1, rgb(0x6f6f7a))
		b.rect(0, 16, 32, 1, rgb(0x6f6f7a))
		b.rect(0, 0, 1, 16, rgb(0x6f6f7a))
		b.rect(16, 16, 1, 16, rgb(0x6f6f7a))
		noise(30, rgb(0x9b9ba6))
	case "asphalt":
		fill(rgb(0x3b3b46))
		noise(90, rgb(0x474752))
		noise(40, rgb(0x30303a))
	case "concrete":
		fill(rgb(0x3a3844))
		noise(60, rgb(0x45434f))
		noise(30, rgb(0x2f2d39))
		b.rect(0, 0, 32, 1, rgb(0x2c2a36))
		b.rect(0, 0, 1, 32, rgb(0x2c2a36))
	case "marble":
		fill(rgb(0xe4e0e6))
		b.rect(0, 0, 32, 1, rgb(0xb4aebc))
		b.rect(0, 0, 1, 32, rgb(0xb4aebc))
		b.line(4, 2, 20, 30, rgb(0xcfc9d6))
		b.line(22, 0, 31, 14, rgb(0xcfc9d6))
	case "parquet":
		fill(rgb(0x8a5a30))
		for y := 0; y < 32; y += 16 {
			for x := 0; x < 32; x += 16 {
				c := rgb(0x7a4e2a)
				if (x/16+y/16)%2 == 0 {
					c = rgb(0x9a6838)
				}
				b.rect(x, y, 16, 16, c)
				b.rect(x, y, 16, 1, rgb(0x5a3a20))
				b.rect(x, y, 1, 16, rgb(0x5a3a20))
			}
		}
	case "sand":
		fill(rgb(0x4e485a))
		noise(110, rgb(0x5a5468))
		noise(80, rgb(0x423e4e))
	case "dirt":
		fill(rgb(0x4e3a28))
		noise(100, rgb(0x5c4632))
		noise(70, rgb(0x3e2e1e))
	case "rooftop":
		fill(rgb(0x1e3a2c))
		b.rect(0, 0, 32, 1, rgb(0x2a4a3a))
		b.rect(0, 0, 1, 32, rgb(0x2a4a3a))
		noise(60, rgb(0x244434))
		noise(30, rgb(0x182e22))
	case "rooftop2":
		fill(rgb(0x0e1a34))
		b.rect(0, 0, 32, 1, rgb(0x1a2c52))
		b.rect(0, 0, 1, 32, rgb(0x1a2c52))
		noise(50, rgb(0x12203e))
	case "boardwalk":
		fill(rgb(0x6a5440))
		for y := 0; y < 32; y += 8 {
			b.rect(0, y, 32, 1, rgb(0x4a382a))
			b.rect(0, y+1, 32, 1, rgb(0x7a634c))
			b.rect((y*5)%32, y+2, 1, 6, rgb(0x4a382a))
		}
	case "cobble":
		fill(rgb(0x8a8478))
		for y := 0; y < 32; y += 8 {
			off := (y / 8 % 2) * 6
			for x := -6; x < 32; x += 12 {
				col := rgb(0xa49e90)
				if (x+y)%3 == 0 {
					col = rgb(0x9a9486)
				}
				b.rect(x+off, y, 11, 7, col)
				b.rect(x+off, y, 11, 1, rgb(0xb8b2a4))
			}
		}
		noise(26, rgb(0x76705f))
	case "pitch":
		if (variant+0)%1 == 0 {
			fill(rgb(0x2c7a38))
			b.rect(0, 0, 32, 16, rgb(0x338a40))
			noise(60, rgb(0x3a9648))
			noise(40, rgb(0x246a2e))
		}
	case "white":
		fill(rgb(0xe8e8e0))
	case "arena":
		fill(rgb(0xb89a64))
		noise(120, rgb(0xc8aa74))
		noise(90, rgb(0xa48854))
		for i := 0; i < 6; i++ {
			x, y := rng.Intn(28), rng.Intn(30)
			b.rect(x, y, 3, 1, rgb(0x94784a))
		}
	case "mall":
		fill(rgb(0xd8d4cc))
		b.rect(0, 0, 32, 1, rgb(0xb8b4aa))
		b.rect(0, 0, 1, 32, rgb(0xb8b4aa))
		for y := 0; y < 32; y += 16 {
			for x := 0; x < 32; x += 16 {
				if (x/16+y/16)%2 == 0 {
					b.rect(x+1, y+1, 15, 15, rgb(0xe4e0d8))
				}
			}
		}
		noise(14, rgb(0xc8c4ba))
	default:
		fill(rgb(0xff00ff))
	}
	return b
}

// radial glow, premultiplied white
func genGlow(n int) *pixBuf {
	b := newPix(n, n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx := (float64(x) + 0.5 - float64(n)/2) / (float64(n) / 2)
			dy := (float64(y) + 0.5 - float64(n)/2) / (float64(n) / 2)
			d := math.Sqrt(dx*dx + dy*dy)
			if d >= 1 {
				continue
			}
			a := math.Pow(1-d, 2)
			v := uint8(a * 255)
			i := (y*n + x) * 4
			b.p[i], b.p[i+1], b.p[i+2], b.p[i+3] = v, v, v, v
		}
	}
	return b
}

// scaled shrinks/enlarges the canvas by k picking the dominant colour of each source cell,
// which keeps pixel art crisp instead of blurring it.
func (b *pixBuf) scaled(k float64) *pixBuf {
	nw, nh := int(math.Ceil(float64(b.w)*k)), int(math.Ceil(float64(b.h)*k))
	o := newPix(nw, nh)
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			x0, x1 := int(math.Floor(float64(x)/k)), int(math.Ceil(float64(x+1)/k))
			y0, y1 := int(math.Floor(float64(y)/k)), int(math.Ceil(float64(y+1)/k))
			counts := map[[4]uint8]int{}
			total, opaque := 0, 0
			for yy := y0; yy < y1 && yy < b.h; yy++ {
				for xx := x0; xx < x1 && xx < b.w; xx++ {
					i := (yy*b.w + xx) * 4
					total++
					if b.p[i+3] > 128 {
						opaque++
						counts[[4]uint8{b.p[i], b.p[i+1], b.p[i+2], 255}]++
					}
				}
			}
			if total == 0 || opaque*2 < total {
				continue
			}
			var best [4]uint8
			bc := 0
			for c, n := range counts {
				if n > bc || (n == bc && c[0]+c[1]+c[2] < best[0]+best[1]+best[2]) {
					best, bc = c, n
				}
			}
			di := (y*nw + x) * 4
			o.p[di], o.p[di+1], o.p[di+2], o.p[di+3] = best[0], best[1], best[2], 255
		}
	}
	return o
}

// detail adds volume and texture to a finished sprite: light from the top left (bevel on the edges)
// and a little deterministic noise, so flat colour areas look like material instead of plastic.
func (b *pixBuf) detail(seed int64, noise float64) *pixBuf {
	o := &pixBuf{w: b.w, h: b.h, p: append([]byte(nil), b.p...)}
	alpha := func(x, y int) uint8 {
		if x < 0 || y < 0 || x >= b.w || y >= b.h {
			return 0
		}
		return b.p[(y*b.w+x)*4+3]
	}
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			i := (y*b.w + x) * 4
			if b.p[i+3] == 0 {
				continue
			}
			f := 1.0
			if alpha(x, y-1) == 0 {
				f *= 1.16
			}
			if alpha(x-1, y) == 0 {
				f *= 1.08
			}
			if alpha(x, y+1) == 0 {
				f *= 0.8
			}
			if alpha(x+1, y) == 0 {
				f *= 0.9
			}
			h := uint32(x*73856093) ^ uint32(y*19349663) ^ uint32(seed*83492791)
			h ^= h >> 13
			f *= 1 + (float64(h%9)-4)/4*noise
			// gentle vertical light gradient
			f *= 1.06 - 0.12*float64(y)/float64(maxI(1, b.h-1))
			cl := func(v float64) uint8 {
				if v > 255 {
					return 255
				}
				return uint8(v)
			}
			o.p[i], o.p[i+1], o.p[i+2] = cl(float64(b.p[i])*f), cl(float64(b.p[i+1])*f), cl(float64(b.p[i+2])*f)
		}
	}
	return o
}

package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// PropKind describes a free-standing object. Props are anchored at the
// bottom-centre of their footprint; the footprint is the part that blocks movement.
type PropKind struct {
	W, H      int
	FW, FH    float64
	Vars      int
	Draw      func(b *pixBuf, v int)
	Hide      bool
	Phone     bool
	TechPhone bool // the yellow phone used only to call the technicians
	Poison    bool
	Trap      string // "explode" | "electric"
	UseKind   string
	Use       []Vec
	Light     bool
	NoSolid   bool
	Solids    []Rect // solid parts relative to the prop anchor, used instead of the foot box when a prop is walkable (a stage you can climb)
	SortOff   float64
	Native    []*pixBuf // sprite frames loaded from assets (already at native scale)
	NAX, NAY  float64
	FPS       float64
	FootDX    float64
	FootDY    float64
	hasFoot   bool
	Ambient   float64
	Category  string
	Item      string // pickup: "poison" or "disguise"
	LightR    float64    // lamps: radius of the light (0 = default)
	LightC    color.RGBA // lamps: colour and strength of the light (A == 0 = default)
}

func c(h uint32) color.RGBA { return rgb(h) }

var propKinds = map[string]*PropKind{}

func init() {
	pk := propKinds

	pk["speaker"] = &PropKind{W: 22, H: 34, FW: 20, FH: 8, Trap: "explode", Draw: func(b *pixBuf, v int) {
		b.cube(0, 0, 22, 34, 6, c(0x4a4a56), c(0x26262e))
		b.ellipse(11, 23, 7, 7, c(0x4e4e5e))
		b.ellipse(11, 23, 5.4, 5.4, c(0x14141a))
		b.ellipse(11, 23, 2.4, 2.4, c(0x34343e))
		b.ellipse(11, 12, 3.6, 3.6, c(0x4e4e5e))
		b.ellipse(11, 12, 2.2, 2.2, c(0x14141a))
		b.set(18, 8, c(0xff3030))
	}}
	pk["couch"] = &PropKind{W: 48, H: 26, FW: 46, FH: 12, UseKind: "lounge", Use: []Vec{{-12, 14}, {12, 14}}, Draw: func(b *pixBuf, v int) {
		base := []uint32{0x7a2f3a, 0x2f5a7a, 0x4a6a3a, 0xe0e0ea}[v%4]
		b.rect(0, 0, 48, 14, shade(c(base), 0.85))
		b.rect(0, 12, 48, 14, shade(c(base), 0.7))
		b.rect(4, 12, 19, 10, shade(c(base), 1.25))
		b.rect(25, 12, 19, 10, shade(c(base), 1.25))
		b.rect(4, 12, 19, 1, shade(c(base), 1.5))
		b.rect(25, 12, 19, 1, shade(c(base), 1.5))
		b.rect(0, 8, 5, 16, c(base))
		b.rect(43, 8, 5, 16, c(base))
		b.rect(0, 22, 48, 4, shade(c(base), 0.55))
	}, Vars: 4}
	pk["armchair"] = &PropKind{W: 26, H: 26, FW: 24, FH: 12, UseKind: "lounge", Use: []Vec{{0, 14}}, Draw: func(b *pixBuf, v int) {
		base := c(0x3a5a8a)
		b.rect(2, 0, 22, 12, shade(base, 0.85))
		b.rect(3, 12, 20, 10, shade(base, 1.2))
		b.rect(0, 6, 5, 16, base)
		b.rect(21, 6, 5, 16, base)
		b.rect(0, 22, 26, 4, shade(base, 0.55))
	}}
	pk["table"] = &PropKind{W: 34, H: 26, FW: 26, FH: 9, Draw: func(b *pixBuf, v int) {
		// barrel table
		b.rect(4, 10, 26, 15, c(0x6a4528))
		b.rect(4, 10, 3, 15, c(0x84593a))
		b.rect(27, 10, 3, 15, c(0x4e3220))
		b.rect(4, 14, 26, 2, c(0x2a2a30))
		b.rect(4, 21, 26, 2, c(0x2a2a30))
		b.ellipse(17, 24, 13, 3, c(0x4e3220))
		b.ellipse(17, 10, 14, 6.5, c(0x8a5a36))
		b.ellipse(17, 9, 12.5, 5.4, c(0xb27a48))
		b.rect(11, 6, 2, 3, c(0xe04040))
		b.rect(21, 8, 2, 3, c(0x40a0e0))
		b.rect(16, 6, 2, 2, c(0xf0f0f0))
	}}
	pk["bar"] = &PropKind{W: 76, H: 38, FW: 74, FH: 14, Vars: 2, UseKind: "drink", Use: []Vec{{-26, 14}, {0, 14}, {26, 14}}, Draw: func(b *pixBuf, v int) {
		b.rect(2, 0, 72, 10, c(0x2a1a12))
		cols := []uint32{0x3a9a5a, 0xd0a030, 0xc04040, 0x3a7ac0, 0xe0e0e0}
		for i, x := 0, 6; x < 70; x, i = x+7, i+1 {
			b.rect(x, 1, 3, 8, c(cols[i%5]))
			b.rect(x+1, 0, 1, 2, c(cols[i%5]))
		}
		if v == 1 {
			b.cube(0, 10, 76, 28, 12, c(0xf4f8ff), c(0xb8c8e8))
			b.rect(0, 34, 76, 2, c(0x40e0ff))
		} else {
			b.cube(0, 10, 76, 28, 12, c(0xb27a48), c(0x5a3520))
			b.rect(0, 34, 76, 2, c(0xc9a640))
		}
		b.rect(10, 14, 4, 3, c(0xe0e0f0))
		b.rect(50, 15, 4, 3, c(0xe04040))
	}}
	pk["punch"] = &PropKind{W: 34, H: 28, FW: 32, FH: 10, UseKind: "drink", Use: []Vec{{0, 14}}, Poison: true, Vars: 2, Draw: func(b *pixBuf, v int) {
		b.cube(1, 12, 32, 16, 7, c(0xf2f2f6), c(0xd4d4de))
		col := c(0xe04a7a)
		if v == 1 {
			col = c(0x5ae05a)
		}
		b.ellipse(17, 11, 11, 6, c(0xcfe4f0))
		b.ellipse(17, 11, 9, 4.6, col)
		b.rect(4, 7, 3, 4, c(0xe0f0f8))
		b.rect(27, 8, 3, 3, c(0xe0f0f8))
	}}
	pk["fridge"] = &PropKind{W: 22, H: 40, FW: 22, FH: 8, Draw: func(b *pixBuf, v int) {
		b.cube(0, 0, 22, 40, 5, c(0xe8eef2), c(0xc8d2d8))
		b.rect(0, 17, 22, 1, c(0x8a959c))
		b.rect(17, 8, 2, 7, c(0x7a858c))
		b.rect(17, 21, 2, 9, c(0x7a858c))
	}}
	pk["counter"] = &PropKind{W: 38, H: 30, FW: 38, FH: 12, Draw: func(b *pixBuf, v int) {
		b.cube(0, 6, 38, 24, 9, c(0xe0e2e6), c(0x8a6a4c))
		b.ellipse(19, 10, 8, 3, c(0x8a95a0))
		b.rect(18, 4, 2, 5, c(0xa0aab4))
		b.rect(15, 3, 6, 2, c(0xa0aab4))
		b.rect(9, 21, 1, 7, c(0x5a432e))
		b.rect(28, 21, 1, 7, c(0x5a432e))
	}}
	pk["stove"] = &PropKind{W: 26, H: 32, FW: 24, FH: 10, Trap: "explode", Draw: func(b *pixBuf, v int) {
		b.cube(1, 6, 24, 26, 9, c(0x6a6a74), c(0x3c3c44))
		b.ellipse(8, 11, 3.5, 2, c(0x15151b))
		b.ellipse(18, 11, 3.5, 2, c(0x15151b))
		b.set(8, 10, c(0x4aa0ff))
		b.set(18, 10, c(0x4aa0ff))
		b.rect(5, 19, 16, 9, c(0x16161c))
		b.rect(6, 20, 14, 2, c(0x5a5a66))
		b.rect(4, 17, 2, 1, c(0xd0d0d8))
		b.rect(20, 17, 2, 1, c(0xd0d0d8))
	}}
	pk["toilet"] = &PropKind{W: 16, H: 22, FW: 14, FH: 8, UseKind: "toilet", Use: []Vec{{0, 12}}, Draw: func(b *pixBuf, v int) {
		b.rect(3, 0, 10, 9, c(0xe6eaee))
		b.rect(3, 0, 10, 1, c(0xffffff))
		b.ellipse(8, 15, 6.5, 5.5, c(0xf4f6f8))
		b.ellipse(8, 15, 4.4, 3.4, c(0xb8c8d8))
		b.rect(5, 19, 6, 3, c(0xd2d8de))
	}}
	pk["stall"] = &PropKind{W: 30, H: 46, FW: 26, FH: 8, Hide: true, Draw: func(b *pixBuf, v int) {
		b.cube(0, 0, 30, 44, 6, c(0xa8bdd2), c(0x7a93ad))
		b.rect(3, 12, 24, 30, c(0x6a86a6))
		b.rect(3, 12, 24, 1, c(0x8aa4c2))
		b.rect(14, 12, 1, 30, c(0x5a7494))
		b.set(21, 26, c(0xe03030))
		b.rect(10, 42, 10, 2, c(0x2a3544))
	}}
	pk["closet"] = &PropKind{W: 28, H: 46, FW: 26, FH: 10, Hide: true, Draw: func(b *pixBuf, v int) {
		b.cube(0, 0, 28, 46, 6, c(0x9a6a40), c(0x6e4529))
		b.rect(13, 7, 1, 38, c(0x3a2414))
		b.rect(2, 9, 10, 34, c(0x7a4f30))
		b.rect(15, 9, 11, 34, c(0x7a4f30))
		b.set(11, 26, c(0xe8c040))
		b.set(16, 26, c(0xe8c040))
	}}
	pk["bed"] = &PropKind{W: 46, H: 40, FW: 46, FH: 24, Draw: func(b *pixBuf, v int) {
		b.rect(0, 0, 46, 12, c(0x5a3a22))
		b.rect(0, 0, 46, 2, c(0x7a5232))
		b.rect(2, 12, 42, 24, c(0xe8e8f0))
		b.rect(5, 13, 16, 8, c(0xffffff))
		b.rect(25, 13, 16, 8, c(0xffffff))
		b.rect(2, 22, 42, 14, c(0x4a6fb3))
		b.rect(2, 22, 42, 2, c(0x6a8fd3))
		b.rect(2, 30, 42, 1, c(0x3a5a98))
		b.rect(0, 36, 46, 4, c(0x5a3a22))
	}}
	pk["tv"] = &PropKind{W: 30, H: 32, FW: 24, FH: 8, UseKind: "lounge", Use: []Vec{{0, 14}}, Draw: func(b *pixBuf, v int) {
		b.cube(3, 22, 24, 10, 3, c(0x4a3322), c(0x2e1e12))
		b.rect(3, 2, 24, 20, c(0x15151b))
		b.rect(5, 4, 20, 16, c(0x4aa8e0))
		b.rect(5, 4, 20, 3, c(0x7accf0))
		b.line(7, 18, 14, 9, c(0x9ad8f8))
	}}
	pk["plant"] = &PropKind{W: 16, H: 28, FW: 8, FH: 6, Draw: func(b *pixBuf, v int) {
		b.rect(4, 19, 8, 9, c(0xb4572e))
		b.rect(3, 19, 10, 2, c(0xc9693a))
		b.ellipse(8, 11, 6, 9, c(0x2e8a3c))
		b.ellipse(5, 13, 3, 6, c(0x3ea04c))
		b.ellipse(11, 12, 3, 6, c(0x276f33))
		b.ellipse(8, 7, 2, 4, c(0x4fb85e))
	}}
	pk["lamp"] = &PropKind{W: 12, H: 52, FW: 6, FH: 4, Light: true, Draw: func(b *pixBuf, v int) {
		b.rect(5, 10, 2, 40, c(0x2a2a34))
		b.rect(2, 4, 8, 7, c(0x3a3a46))
		b.rect(3, 9, 6, 2, c(0xfff0b0))
		b.rect(3, 48, 6, 4, c(0x2a2a34))
	}}
	pk["tree"] = &PropKind{W: 56, H: 72, FW: 10, FH: 6, Draw: func(b *pixBuf, v int) {
		b.rect(24, 48, 8, 24, c(0x6a4228))
		b.rect(24, 48, 2, 24, c(0x8a5a36))
		b.ellipse(28, 42, 20, 6, c(0x1e5028))
		b.ellipse(28, 28, 25, 23, c(0x2a6a34))
		b.ellipse(14, 36, 14, 13, c(0x2e7a3a))
		b.ellipse(42, 36, 14, 13, c(0x2a6a34))
		b.ellipse(22, 20, 13, 10, c(0x3f9448))
		b.ellipse(36, 17, 10, 8, c(0x4aa454))
		b.ellipse(26, 30, 7, 5, c(0x3a8a44))
	}}
	pk["palm"] = &PropKind{W: 48, H: 72, FW: 8, FH: 6, Draw: func(b *pixBuf, v int) {
		for y := 26; y < 72; y++ {
			x := 22 + int(2.5*math.Sin(float64(y)/11))
			b.rect(x, y, 4, 1, c(0x8a6a3c))
			if y%4 == 0 {
				b.rect(x, y, 4, 1, c(0x6a4e2a))
			}
		}
		top := [2]int{24, 26}
		fr := [][2]int{{4, 30}, {10, 14}, {24, 6}, {40, 14}, {46, 32}, {14, 36}, {36, 38}}
		for _, f := range fr {
			b.line(top[0], top[1], f[0], f[1], c(0x2e9a4a))
			b.line(top[0], top[1]+1, f[0], f[1]+1, c(0x237a3a))
			b.line(top[0], top[1]-1, f[0], f[1]-1, c(0x3eb45a))
		}
		b.ellipse(24, 27, 3, 3, c(0x6a4e2a))
	}}
	pk["bush"] = &PropKind{W: 34, H: 28, FW: 26, FH: 9, Hide: true, Draw: func(b *pixBuf, v int) {
		b.ellipse(17, 18, 16, 9, c(0x1e5028))
		b.ellipse(10, 14, 9, 8, c(0x2a6a34))
		b.ellipse(24, 14, 9, 8, c(0x2e7a3a))
		b.ellipse(17, 10, 10, 8, c(0x3a8a44))
		b.ellipse(14, 8, 4, 3, c(0x4aa454))
		b.set(22, 12, c(0xe04a7a))
		b.set(9, 15, c(0xf0e060))
	}}
	pk["dumpster"] = &PropKind{W: 42, H: 32, FW: 40, FH: 12, Hide: true, Draw: func(b *pixBuf, v int) {
		b.rect(0, 4, 42, 6, c(0x1f4a2a))
		b.rect(0, 4, 42, 1, c(0x3a7a4a))
		b.rect(1, 10, 40, 17, c(0x2e7d46))
		for x := 6; x < 40; x += 8 {
			b.rect(x, 10, 1, 17, c(0x24633a))
		}
		b.rect(1, 26, 40, 2, c(0x1a3d26))
		b.rect(4, 28, 6, 4, c(0x15151b))
		b.rect(32, 28, 6, 4, c(0x15151b))
	}}
	pk["bbq"] = &PropKind{W: 28, H: 30, FW: 22, FH: 10, Trap: "explode", Draw: func(b *pixBuf, v int) {
		b.rect(7, 16, 2, 13, c(0x3a3a44))
		b.rect(19, 16, 2, 13, c(0x3a3a44))
		b.ellipse(14, 13, 12, 8, c(0x3a3a44))
		b.ellipse(14, 12, 10, 6, c(0x15151b))
		for i := 0; i < 6; i++ {
			b.set(8+i*2, 11+i%2, c(0xff7a20))
		}
		b.rect(2, 10, 3, 2, c(0x8a8a96))
		b.ellipse(11, 28, 3, 2, c(0x15151b))
	}}
	pk["car"] = &PropKind{W: 66, H: 36, FW: 62, FH: 16, Vars: 6, Draw: func(b *pixBuf, v int) { genCar(b, v) }}
	pk["dj"] = &PropKind{W: 60, H: 36, FW: 58, FH: 14, Draw: func(b *pixBuf, v int) {
		b.cube(0, 12, 60, 24, 9, c(0x34343e), c(0x1c1c24))
		b.rect(2, 32, 56, 2, c(0xe040a0))
		b.ellipse(15, 16, 9, 4, c(0x15151b))
		b.ellipse(15, 16, 6, 2.6, c(0x9a9aaa))
		b.ellipse(45, 16, 9, 4, c(0x15151b))
		b.ellipse(45, 16, 6, 2.6, c(0x9a9aaa))
		b.rect(26, 14, 8, 5, c(0x666678))
		b.rect(22, 3, 16, 9, c(0x8a8a9a))
		b.rect(23, 4, 14, 7, c(0x4ab8f0))
	}}
	pk["sunbed"] = &PropKind{W: 38, H: 20, FW: 32, FH: 9, UseKind: "lounge", Use: []Vec{{0, 12}}, Vars: 3, Draw: func(b *pixBuf, v int) {
		col := []uint32{0xf0f0f0, 0xe85a8a, 0x4ab8e8}[v%3]
		b.rect(2, 8, 2, 10, c(0x888894))
		b.rect(32, 8, 2, 10, c(0x888894))
		b.rect(2, 4, 32, 10, c(col))
		b.rect(2, 4, 32, 1, shade(c(col), 1.2))
		b.rect(2, 9, 32, 1, shade(c(col), 0.8))
		b.rect(3, 1, 9, 5, shade(c(col), 0.9))
	}}
	pk["umbrella"] = &PropKind{W: 52, H: 64, FW: 8, FH: 6, Vars: 3, Draw: func(b *pixBuf, v int) {
		col := []uint32{0xe04a4a, 0x4aa8e0, 0xf0c030}[v%3]
		b.rect(25, 18, 2, 42, c(0xdddde4))
		b.rect(20, 59, 12, 5, c(0x888894))
		b.ellipse(26, 17, 24, 13, c(col))
		for x := 4; x < 48; x += 8 {
			b.line(26, 5, x, 22, shade(c(col), 0.75))
		}
		b.ellipse(26, 11, 14, 5, shade(c(col), 1.2))
		b.rect(25, 2, 2, 3, c(0xdddde4))
	}}
	pk["fusebox"] = &PropKind{W: 18, H: 28, FW: 14, FH: 6, Trap: "electric", Draw: func(b *pixBuf, v int) {
		b.rect(1, 0, 16, 26, c(0x5a6068))
		b.rect(3, 2, 12, 22, c(0x8a9097))
		b.rect(5, 5, 8, 8, c(0xf0c020))
		b.line(9, 6, 7, 10, c(0x15151b))
		b.line(7, 10, 11, 10, c(0x15151b))
		b.line(11, 10, 9, 12, c(0x15151b))
		b.rect(8, 16, 2, 5, c(0xe03030))
		b.rect(3, 26, 12, 2, c(0x3a4048))
	}}
	pk["fountain"] = &PropKind{W: 62, H: 50, FW: 54, FH: 24, Draw: func(b *pixBuf, v int) {
		b.ellipse(31, 38, 30, 12, c(0x9898a6))
		b.ellipse(31, 36, 28, 10.5, c(0xc4c4d0))
		b.ellipse(31, 36, 24, 8.5, c(0x4aa8d8))
		b.ellipse(31, 35, 18, 5, c(0x6ac4ee))
		b.rect(28, 10, 6, 26, c(0xb4b4c2))
		b.ellipse(31, 12, 11, 4, c(0xc4c4d0))
		b.ellipse(31, 11, 8, 2.6, c(0x6ac4ee))
		b.line(31, 4, 31, 10, c(0xaee4fa))
		b.line(31, 4, 25, 12, c(0xaee4fa))
		b.line(31, 4, 37, 12, c(0xaee4fa))
	}}
	pk["piano"] = &PropKind{W: 54, H: 42, FW: 50, FH: 18, Draw: func(b *pixBuf, v int) {
		b.cube(0, 4, 54, 38, 20, c(0x2c2c36), c(0x15151b))
		b.rect(4, 26, 46, 8, c(0xf0f0f4))
		for x := 4; x < 50; x += 3 {
			b.rect(x, 26, 1, 8, c(0x9a9aa6))
		}
		for x := 6; x < 48; x += 6 {
			b.rect(x, 26, 2, 5, c(0x15151b))
		}
		b.rect(3, 38, 3, 4, c(0x15151b))
		b.rect(48, 38, 3, 4, c(0x15151b))
	}}
	pk["bookshelf"] = &PropKind{W: 38, H: 50, FW: 38, FH: 10, Draw: func(b *pixBuf, v int) {
		b.cube(0, 0, 38, 50, 4, c(0x7a4f2e), c(0x4a2c18))
		cols := []uint32{0xc04040, 0x3a7ac0, 0x40a060, 0xd0a030, 0x8a4aa0, 0xe0e0e0}
		for row := 0; row < 4; row++ {
			y := 6 + row*11
			b.rect(2, y+8, 34, 2, c(0x7a4f2e))
			for x, i := 3, row; x < 34; x, i = x+4, i+1 {
				h := 6 + (i*3)%4
				b.rect(x, y+9-h, 3, h, c(cols[i%6]))
			}
		}
	}}
	pk["longtable"] = &PropKind{W: 98, H: 36, FW: 96, FH: 14, Draw: func(b *pixBuf, v int) {
		b.cube(0, 8, 98, 28, 12, c(0xf0ece0), c(0xcfc8b8))
		for x := 8; x < 92; x += 16 {
			b.ellipse(float64(x)+4, 14, 5, 2, c(0xffffff))
			b.ellipse(float64(x)+4, 14, 3, 1.2, c(0xe0dcd0))
		}
		for x := 24; x < 80; x += 32 {
			b.rect(x, 2, 2, 8, c(0xf0e0b0))
			b.set(x, 1, c(0xffa020))
		}
		b.rect(0, 32, 98, 1, c(0xa8a090))
	}}
	pk["phone"] = &PropKind{W: 24, H: 46, FW: 16, FH: 8, Phone: true, Draw: func(b *pixBuf, v int) {
		red, dark := c(0xc82828), c(0x8a1a1a)
		b.rect(1, 0, 22, 4, dark) // roof
		b.rect(2, 0, 20, 1, c(0xe05050))
		b.rect(6, 1, 12, 2, c(0xf0d040))  // TEL sign
		b.rect(2, 4, 20, 38, red)         // frame
		b.rect(4, 6, 16, 32, c(0x86b8d0)) // glass
		b.rect(5, 7, 4, 28, c(0xa8d4e8))
		b.rect(11, 4, 2, 38, red) // mullions
		b.rect(4, 17, 16, 2, red)
		b.rect(4, 28, 16, 2, red)
		b.rect(7, 20, 10, 7, c(0x2a2a34)) // the phone inside
		b.rect(8, 21, 8, 2, c(0xd8d8e0))
		b.rect(9, 24, 2, 2, c(0x8a8a9a))
		b.rect(13, 24, 2, 2, c(0x8a8a9a))
		b.rect(1, 40, 22, 5, dark) // base
	}}
	pk["weapon"] = &PropKind{W: 16, H: 10, FW: 8, FH: 3, NoSolid: true, Draw: func(b *pixBuf, v int) {
		if v == 1 { // pistol
			b.rect(2, 3, 9, 3, c(0x2a2a32))
			b.rect(2, 3, 9, 1, c(0x5a5a68))
			b.rect(3, 6, 3, 4, c(0x3a3a46))
		} else { // knife
			b.rect(1, 5, 4, 2, c(0x4a2c14))
			b.rect(5, 5, 9, 2, c(0xd0d6e0))
			b.rect(5, 5, 9, 1, c(0xf4f8ff))
		}
	}}
	pk["bench"] = &PropKind{W: 38, H: 22, FW: 36, FH: 8, Draw: func(b *pixBuf, v int) {
		b.rect(1, 2, 36, 7, c(0x9a6a3c))
		b.rect(1, 2, 36, 1, c(0xb88a5c))
		b.rect(1, 11, 36, 5, c(0x8a5a30))
		b.rect(3, 16, 3, 5, c(0x2a2a34))
		b.rect(32, 16, 3, 5, c(0x2a2a34))
	}}
	pk["stool"] = &PropKind{W: 12, H: 20, FW: 6, FH: 4, Draw: func(b *pixBuf, v int) {
		b.ellipse(6, 6, 5.5, 3, c(0x8a2a2a))
		b.ellipse(6, 5, 5, 2.4, c(0xb04040))
		b.rect(5, 8, 2, 9, c(0x8a8a96))
		b.ellipse(6, 18, 4, 1.6, c(0x6a6a76))
	}}
	pk["statue"] = &PropKind{W: 20, H: 44, FW: 14, FH: 8, Draw: func(b *pixBuf, v int) {
		b.cube(2, 30, 16, 14, 4, c(0xc4c4d0), c(0x9a9aa8))
		b.ellipse(10, 8, 3.6, 4, c(0xe0e0ea))
		b.rect(5, 12, 10, 14, c(0xd0d0dc))
		b.rect(3, 14, 3, 10, c(0xc0c0cc))
		b.rect(14, 14, 3, 10, c(0xc0c0cc))
		b.rect(6, 26, 8, 4, c(0xb8b8c6))
	}}
}

func genCar(b *pixBuf, v int) {
	cols := []uint32{0xc03a3a, 0x3a6ac0, 0x3aa05a, 0xe0c040, 0xf2f2f6, 0xf6f6fa}
	body := c(cols[v%len(cols)])
	police := v == 4
	b.ellipse(33, 31, 30, 4, color.RGBA{0, 0, 0, 90})
	b.rect(2, 16, 62, 15, body)
	b.rect(2, 16, 62, 1, shade(body, 1.25))
	b.rect(2, 28, 62, 3, shade(body, 0.7))
	b.rect(14, 5, 36, 13, shade(body, 0.9))
	b.rect(14, 5, 36, 1, shade(body, 1.25))
	b.rect(16, 7, 14, 9, c(0x7aa0c0))
	b.rect(33, 7, 15, 9, c(0x7aa0c0))
	b.rect(16, 7, 14, 2, c(0xa8c8e0))
	b.rect(31, 5, 2, 12, shade(body, 0.7))
	b.rect(2, 20, 3, 3, c(0xfff0b0))
	b.rect(61, 20, 3, 3, c(0xe03030))
	b.ellipse(15, 30, 6, 5, c(0x15151b))
	b.ellipse(15, 30, 3, 2.6, c(0x9a9aa6))
	b.ellipse(51, 30, 6, 5, c(0x15151b))
	b.ellipse(51, 30, 3, 2.6, c(0x9a9aa6))
	if v == 5 {
		b.rect(2, 22, 62, 3, c(0xd02828))
		b.rect(30, 16, 8, 6, c(0xd02828))
		b.rect(32, 14, 4, 10, c(0xd02828))
		b.rect(22, 2, 22, 3, c(0x2a2a34))
		b.rect(22, 2, 11, 3, c(0xe03030))
		b.rect(33, 2, 11, 3, c(0xf4f4f8))
	}
	if police {
		b.rect(2, 22, 62, 4, c(0x2a4ab8))
		b.rect(22, 2, 22, 3, c(0x2a2a34))
		b.rect(22, 2, 11, 3, c(0xe03030))
		b.rect(33, 2, 11, 3, c(0x3060ff))
		b.rect(26, 24, 14, 1, c(0xffffff))
	}
}

// PropImg is a prop sprite at the original game's scale with its anchor (bottom-centre).
type PropImg struct {
	img    *ebiten.Image
	ox, oy float64
}

// renderPropSprite draws a prop at design size, scales it to the native resolution and adds detail and outline.
func renderPropSprite(name string, k *PropKind, v int) *pixBuf {
	b := newPix(k.W+2, k.H+2)
	b.ox, b.oy = 1, 1
	k.Draw(b, v)
	seed := int64(len(name)*31+v) + int64(name[0])*977
	return b.scaled(wk).detail(seed, 0.035).outlinedSel(0.45)
}

func buildPropImages() map[string][]PropImg {
	fitAllFootprints()
	out := map[string][]PropImg{}
	for name, k := range propKinds {
		// Both branches go through stylizeSprite so exported, user-painted and
		// procedural props share one muted palette.
		if k.Native != nil {
			for _, f := range k.Native {
				out[name] = append(out[name], PropImg{img: stylizeSprite(f).img(), ox: k.NAX, oy: k.NAY})
			}
			continue
		}
		n := k.Vars
		if n == 0 {
			n = 1
		}
		for v := 0; v < n; v++ {
			sb := stylizeSprite(renderPropSprite(name, k, v))
			ox := (float64(k.W)/2 + 1) * wk
			oy := (float64(k.H) + 1) * wk
			out[name] = append(out[name], PropImg{img: sb.img(), ox: math.Round(ox), oy: math.Round(oy)})
		}
	}
	return out
}

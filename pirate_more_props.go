package main

import "image/color"

// ---------------------------------------------------------------------------
// More things on the deck of the BARCO PIRATA (v1.4.0, tanda H): barrels of gunpowder with sticks of dynamite (a trap:
// they blow up like the speakers, Trap "explode") and heaps of loot spilled on the planks (coins, a goblet, gems; just
// scenery, walkable).
// ---------------------------------------------------------------------------

func init() {
	pk := propKinds
	pk["pdynamite"] = &PropKind{W: 24, H: 32, FW: 18, FH: 8, Trap: "explode", Draw: func(b *pixBuf, v int) {
		pirateBarrel(b, 3, 7, 18, 24, rgb(0x5a3a26))
		b.rect(5, 17, 14, 7, rgb(0xd8c8a0)) // the label: a skull over TNT
		b.rect(10, 18, 4, 3, rgb(0x14141a))
		b.set(11, 19, rgb(0xd8c8a0))
		b.rect(9, 22, 6, 1, rgb(0xb02a1a))
		for i, x := range []int{6, 11, 16} { // the red sticks poking out of the top, with fuses
			b.rect(x, 2+i%2, 3, 8, rgb(0xc8281e))
			b.rect(x, 2+i%2, 1, 8, rgb(0xe85040))
			b.set(x+1, 1+i%2, rgb(0x3a3028))
			b.set(x+2, i%2, rgb(0x3a3028))
		}
	}}
	pk["pgold"] = &PropKind{W: 30, H: 18, FW: 24, FH: 6, NoSolid: true, Vars: 2, Draw: func(b *pixBuf, v int) {
		gold, goldL, goldD := rgb(0xd8b048), rgb(0xf4e090), rgb(0x9a7428)
		b.ellipse(15, 15, 14, 3, color.RGBA{0, 0, 0, 60})
		for i := 0; i < 40; i++ { // the heap: thicker in the middle
			x := 3 + phHash(i+v*50, 17)%24
			dy := phHash(i+v*50, 19) % (2 + 6 - abs(x-15)/3)
			b.rect(x, 15-dy, 2, 1, []color.RGBA{gold, goldL, goldD}[i%3])
		}
		if v == 0 { // a goblet
			b.rect(20, 4, 5, 4, gold)
			b.rect(21, 8, 3, 3, goldD)
			b.rect(20, 11, 5, 1, gold)
			b.set(21, 5, goldL)
		} else { // a string of pearls and a crown
			b.rect(6, 6, 9, 3, gold)
			for _, x := range []int{6, 10, 14} {
				b.rect(x, 4, 1, 2, gold)
			}
			b.set(10, 7, rgb(0xd04060))
			for x := 17; x < 26; x += 2 {
				b.set(x, 11, rgb(0xf4f0f4))
			}
		}
		b.rect(4, 12, 2, 2, rgb(0x40c070)) // gems
		b.rect(25, 13, 2, 2, rgb(0x4070e0))
	}}
}

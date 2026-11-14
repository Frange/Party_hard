package main

import "github.com/hajimehoshi/ebiten/v2"

// The 14x14 picture of ALCALÁ DE HENARES in the list of parties (draw.go drawLevelIcon, levelIconArt in italy_icon.go):
// a stork in its nest on top of a stone belfry against the dusk sky, a red roof below.
// s sky, o the orange of the horizon, W stork white, k black feathers, r beak, n nest, T tower stone, t tower shade,
// d opening, R red roof.
var alcalaIconRows = [12]string{
	"ssssssssssss",
	"ssssWWssssss",
	"ssrrWWssssss",
	"ssssWWWWkkss",
	"sssssWWWkkss",
	"sssnnnnnnnss",
	"oooTTTTTTtoo",
	"oooTddTddtoo",
	"oooTddTddtoo",
	"RRRTTTTTTtRR",
	"RRRTTddTTtRR",
	"RRRTTddTTtRR",
}

var alcalaIconCols = map[byte]uint32{
	's': 0x4a3a72, 'o': 0xd08a6c, 'W': 0xf2eee6, 'k': 0x18181e, 'r': 0xd8502e, 'n': 0x7a5a36,
	'T': 0xd8c2a2, 't': 0xa8927a, 'd': 0x2e2228, 'R': 0xa8503c,
}

func init() {
	levelIconArt["alcala"] = drawAlcalaIcon
}

func drawAlcalaIcon(dst *ebiten.Image, x, y float64, dim bool) {
	fillRect(dst, x, y, 14, 14, rgb(0x0a0a10))
	for r, row := range alcalaIconRows {
		for i := 0; i < len(row); i++ {
			c := rgb(alcalaIconCols[row[i]])
			if dim {
				c = shade(c, 0.4)
			}
			fillRect(dst, x+1+float64(i), y+1+float64(r), 1, 1, c)
		}
	}
}

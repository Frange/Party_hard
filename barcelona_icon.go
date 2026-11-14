package main

import "github.com/hajimehoshi/ebiten/v2"

// The 14x14 picture of BARCELONA in the list of parties (draw.go drawLevelIcon, levelIconArt in italy_icon.go): the
// spires of the basilica against the blue sky, the sea, and the senyera (yellow and red stripes) at the bottom.
// s sky, W spire lit, w spire shade, d opening, p pinnacle, m sea, Y yellow, R red.
var barcelonaIconRows = [12]string{
	"sssssspsssss",
	"ssspsWwsspss",
	"sssWsWwssWss",
	"sssWwWdwsWws",
	"sssWwWwwsWws",
	"ssWdwWdwWdws",
	"mmWwwWwwWwwm",
	"mmWwwddwWwwm",
	"YYYYYYYYYYYY",
	"RRRRRRRRRRRR",
	"YYYYYYYYYYYY",
	"RRRRRRRRRRRR",
}

var barcelonaIconCols = map[byte]uint32{
	's': 0x4a7ad0, 'W': 0xe8d0a6, 'w': 0xa88a62, 'd': 0x3a2a24, 'p': 0xd84a3a, 'm': 0x2a64a8,
	'Y': 0xf4c418, 'R': 0xc8202a,
}

func init() {
	levelIconArt[bcnIcon] = drawBarcelonaIcon
}

func drawBarcelonaIcon(dst *ebiten.Image, x, y float64, dim bool) {
	fillRect(dst, x, y, 14, 14, rgb(0x0a0a10))
	for r, row := range barcelonaIconRows {
		for i := 0; i < len(row); i++ {
			c := rgb(barcelonaIconCols[row[i]])
			if dim {
				c = shade(c, 0.4)
			}
			fillRect(dst, x+1+float64(i), y+1+float64(r), 1, 1, c)
		}
	}
}

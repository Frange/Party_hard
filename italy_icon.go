package main

import "github.com/hajimehoshi/ebiten/v2"

// levelIconArt: parties whose 14x14 icon in the list of parties is a tiny picture instead of the flat colour swatch
// (draw.go: drawLevelIcon). ITALIA: the leaning tower and a piece of the amphitheatre against the evening sky.
var levelIconArt = map[string]func(dst *ebiten.Image, x, y float64, dim bool){
	"italy": drawItalyIcon,
}

// italyIconRows: 12x12, one letter per pixel. s sky, g ground, W marble lit, w marble shade, d dark opening,
// A travertine, a travertine shade, o arch opening, c cypress.
var italyIconRows = [12]string{
	"ssssssWsssss",
	"sssssWWwssss",
	"sssssWdwssss",
	"ssssWWwwssss",
	"ssssWdwdssss",
	"ssssWWwwsAAa",
	"ssscWdwdsAoa",
	"sscWWwwssAAa",
	"sscWdwdsAAoa",
	"sccWWwwsAoAa",
	"gccWwwwgAAAa",
	"gggggggggggg",
}

var italyIconCols = map[byte]uint32{
	's': 0x56658a, 'g': 0x5c5a52, 'W': 0xf6f0e4, 'w': 0x98a0b4, 'd': 0x22242e,
	'A': 0xc8bca6, 'a': 0x8a7e6e, 'o': 0x2a221e, 'c': 0x2a4a2e,
}

func drawItalyIcon(dst *ebiten.Image, x, y float64, dim bool) {
	fillRect(dst, x, y, 14, 14, rgb(0x0a0a10))
	for r, row := range italyIconRows {
		for i := 0; i < len(row); i++ {
			c := rgb(italyIconCols[row[i]])
			if dim {
				c = shade(c, 0.4)
			}
			fillRect(dst, x+1+float64(i), y+1+float64(r), 1, 1, c)
		}
	}
}

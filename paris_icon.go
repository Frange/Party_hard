package main

import "github.com/hajimehoshi/ebiten/v2"

// The 14x14 picture of PARÍS in the list of parties (draw.go drawLevelIcon, levelIconArt in italy_icon.go): the Eiffel
// Tower against the night sky, a zinc roof, and the French tricolour at the bottom.
// s sky, T tower, t tower shade, l light on top, z zinc roof, B blue, W white, R red.
var parisIconRows = [12]string{
	"sssssslsssss",
	"sssssTtsssss",
	"sssssTtsssss",
	"ssssTTttssss",
	"ssssTsstssss",
	"sssTTTtttsss",
	"sssTssssTsss",
	"zzTTzzzzttzz",
	"zTzzzzzzzztz",
	"BBBBWWWWRRRR",
	"BBBBWWWWRRRR",
	"BBBBWWWWRRRR",
}

var parisIconCols = map[byte]uint32{
	's': 0x1a2040, 'T': 0xb89a6a, 't': 0x7a6448, 'l': 0xf0d070, 'z': 0x5a6478,
	'B': 0x1e3c9a, 'W': 0xf4f2ee, 'R': 0xc8202a,
}

func init() {
	levelIconArt["eiffel"] = drawParisIcon
}

func drawParisIcon(dst *ebiten.Image, x, y float64, dim bool) {
	fillRect(dst, x, y, 14, 14, rgb(0x0a0a10))
	for r, row := range parisIconRows {
		for i := 0; i < len(row); i++ {
			c := rgb(parisIconCols[row[i]])
			if dim {
				c = shade(c, 0.4)
			}
			fillRect(dst, x+1+float64(i), y+1+float64(r), 1, 1, c)
		}
	}
}

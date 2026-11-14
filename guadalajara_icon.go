package main

import "github.com/hajimehoshi/ebiten/v2"

// The 14x14 picture of GUADALAJARA in the list of parties (draw.go drawLevelIcon, levelIconArt in italy_icon.go):
// the black bull silhouette on top of the green hill against the dusk sky, the fountain's jet in front.
// s sky, p the pink of the clouds, o the orange of the horizon, k the bull, g hill, G its shade, W water, B basin.
var guadalajaraIconRows = [12]string{
	"ssssssssssss",
	"sspppsssssss",
	"skssssskksss",
	"skkkkkkkkkss",
	"oskkkkkkkooo",
	"oskssskssooo",
	"ggkgggkggggo",
	"gggggWggGGGG",
	"GggggWgGGGGG",
	"GGgWWWWWGGGG",
	"BBBWWWWWBBBG",
	"BBBBBBBBBBBB",
}

var guadalajaraIconCols = map[byte]uint32{
	's': 0x4a3a72, 'p': 0xc87890, 'o': 0xe0a068, 'k': 0x0e0c10, 'g': 0x4a7a44, 'G': 0x2c4e30,
	'W': 0x8ac4f0, 'B': 0xd0c4b0,
}

func init() {
	levelIconArt["guadalajara"] = drawGuadalajaraIcon
}

func drawGuadalajaraIcon(dst *ebiten.Image, x, y float64, dim bool) {
	fillRect(dst, x, y, 14, 14, rgb(0x0a0a10))
	for r, row := range guadalajaraIconRows {
		for i := 0; i < len(row); i++ {
			c := rgb(guadalajaraIconCols[row[i]])
			if dim {
				c = shade(c, 0.4)
			}
			fillRect(dst, x+1+float64(i), y+1+float64(r), 1, 1, c)
		}
	}
}

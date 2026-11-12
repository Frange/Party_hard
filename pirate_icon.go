package main

import "github.com/hajimehoshi/ebiten/v2"

// The 14x14 picture of the BARCO PIRATA in the list of parties (draw.go drawLevelIcon, levelIconArt in italy_icon.go):
// a galleon with its black flag on the sea at sunset.
// s sky, u sun, w water, h hull, H the lit rail, S sail, k black flag, x skull, m mast.
var pirateIconRows = [12]string{
	"ssssmkkkssss",
	"ssssmkxkssss",
	"ssssmkkksuus",
	"sssSmSSssuuu",
	"ssSSmSSSsuuu",
	"ssSSmSSSswww",
	"ssSSmSSSSwww",
	"wwSSmSSSSwww",
	"wHHHHHHHHHHw",
	"whhhhhhhhhww",
	"wwhhhhhhhwww",
	"wwwwwwwwwwww",
}

var pirateIconCols = map[byte]uint32{
	's': 0x5a4a78, 'u': 0xd88068, 'w': 0x2a3a5c, 'h': 0x4a3024, 'H': 0x8a6a50,
	'S': 0xd4c8b0, 'k': 0x16141a, 'x': 0xece8e0, 'm': 0x5a3a26,
}

func init() {
	levelIconArt["pirate"] = drawPirateIcon
}

func drawPirateIcon(dst *ebiten.Image, x, y float64, dim bool) {
	fillRect(dst, x, y, 14, 14, rgb(0x0a0a10))
	for r, row := range pirateIconRows {
		for i := 0; i < len(row); i++ {
			c := rgb(pirateIconCols[row[i]])
			if dim {
				c = shade(c, 0.4)
			}
			fillRect(dst, x+1+float64(i), y+1+float64(r), 1, 1, c)
		}
	}
}

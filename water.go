package main

// The open sea of the beach parties (MIAMI BEACH and the random PLAYA): the water is the band painted by the
// background above the sand (draw.go renderBG, Sky "sea"), with no floor under it, so nothing kept people out
// of it and the player could walk into the sea. addSeaShore puts an invisible solid (WallVoid: never drawn) over
// the whole band, down to a little past the top of the sand, so that the feet stop on the beach and nobody is
// seen standing in the water. Every other water of the game was already closed: the canal of ITALIA (WallVoid),
// the sea around the cruise, the station and the island (voidWalls of their outline); the pools and the lake of
// the island are the swimming places by design (Pools + Swim). water_test.go checks all of it.

// seaShoreGap: how far below the top of the sand the invisible solid ends (the foam reaches the shore line).
const seaShoreGap = 2.0

// seaShoreY: the top of the sand, where the painted sea ends (the same rule as the surf, ambient.go).
func seaShoreY(d *LevelDef) float64 {
	y := 90.0
	for _, f := range d.Floors {
		if f.Kind == "sand" && f.R.Y < y+200 {
			return f.R.Y
		}
	}
	return y
}

// addSeaShore closes the painted sea of a beach level (Sky "sea" without an outline).
func addSeaShore(d *LevelDef) {
	if d.Sky != "sea" || d.Shape != nil {
		return
	}
	d.Walls = append(d.Walls, Wall{R(0, 0, d.W, seaShoreY(d)+seaShoreGap), WallVoid})
}

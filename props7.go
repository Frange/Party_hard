package main

// Road visitors that are not plain traffic: a monster truck that drives over
// the other cars, and three cars straight out of Rocket League (Octane,
// Fennec, Dominus) in blue and orange, boosting down the road.

func init() {
	pk := propKinds
	add := func(name string, w, h int, speed float64, vars int, draw func(b *pixBuf, v int)) {
		pk[name] = &PropKind{W: w, H: h, FW: float64(w) - 4, FH: 12, NoSolid: true, Vars: vars, Ambient: speed, Category: "vehicle", Draw: draw}
	}
	add("monstertruck", 100, 64, 80, 3, vaMonsterTruckSide)
	add("octane", 64, 34, 150, 2, func(b *pixBuf, v int) { vaRLSide(b, vaOctane, v) })
	add("fennec", 58, 34, 155, 2, func(b *pixBuf, v int) { vaRLSide(b, vaFennec, v) })
	add("dominus", 68, 32, 150, 2, func(b *pixBuf, v int) { vaRLSide(b, vaDominus, v) })
}

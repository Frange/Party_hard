package main

// Street vehicles beyond the plain car: vans, trucks, an ice-cream truck, the
// buses, taxis and a few famous cars. All are ambient props (they drive along
// the road lanes, see updateAmbient). They are painted with the vehicle art kit
// (vehart.go), like the car, the police and the ambulance.

func init() {
	pk := propKinds
	add := func(name string, w, h int, speed float64, vars int, draw func(b *pixBuf, v int)) {
		pk[name] = &PropKind{W: w, H: h, FW: float64(w) - 4, FH: 12, NoSolid: true, Vars: vars, Ambient: speed, Category: "vehicle", Draw: draw}
	}
	// plain van, three colours
	add("van", 78, 38, 95, 3, vaVanSide)
	// 0 delivery truck, 1 container truck, 2 bin lorry, 3 grey lorry
	add("boxtruck", 100, 44, 78, 4, vaTruckSide)
	add("icetruck", 84, 46, 62, 1, func(b *pixBuf, v int) { vaIceTruckSide(b) })
	// 0 the yellow school bus, 1 the green city bus (both bring guests)
	add("schoolbus", 112, 42, 70, 2, vaBusSide)
	add("taxi", 66, 38, 105, 1, func(b *pixBuf, v int) { vaTaxiSide(b) })
	add("batmobile", 84, 34, 150, 1, func(b *pixBuf, v int) { vaBatmobileSide(b) })
	add("delorean", 72, 36, 100, 2, func(b *pixBuf, v int) { vaDeloreanSide(b) })
	add("homercar", 66, 38, 90, 1, func(b *pixBuf, v int) {
		vaBuild(b, 66, func(t *pixBuf) {
			vaSedanSide(t, vaCar{body: c(0xf08aa8)})
			t.rect(36, 18, 13, 7, c(0xd87894)) // the odd rear door
			t.rect(2, 22, 62, 1, c(0xffc8d8))
		}, nil)
	})
	add("wackymean", 80, 34, 135, 1, func(b *pixBuf, v int) { vaWackyMeanSide(b) })
	add("wackyboulder", 70, 38, 90, 1, func(b *pixBuf, v int) { vaWackyBoulderSide(b) })
	add("wackypink", 66, 38, 125, 1, func(b *pixBuf, v int) {
		vaBuild(b, 66, func(t *pixBuf) {
			vaSedanSide(t, vaCar{body: c(0xf860a8), rim: c(0xffe0f0)})
			t.rect(26, 5, 14, 2, c(0xffe0f0))
			t.rect(6, 18, 54, 1, c(0xffc0e0))
		}, nil)
	})
}

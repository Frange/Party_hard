package main

// ---------------------------------------------------------------------------
// THE SHORT SHOP. The clothes went from 76 to 41 and the accessories from 135 to 60: only the best and most original
// of each kind stay for sale (the colour variants that only repeat another one, the near-twin hats, the hair cuts that
// look alike...). What left the catalogue is NOT lost: addWear still registers its look (wearFx), and when a save owns
// it (Progress.Owned) the wardrobe lists it and wears it as before -- shopRetired holds the names for that. Nothing is
// rewritten in the progress. The 12 perks, 8 animals and 20 weapons are unchanged.
// ---------------------------------------------------------------------------

// shopKeepWear: the ids still on sale (outfits and accessories). Any other addWear id is retired.
var shopKeepWear = map[string]bool{}

// shopRetired: the items that left the shop (names for the wardrobe), rebuilt with the catalogue.
var shopRetired []shopItem

func init() {
	for _, id := range []string{
		// clothes (41)
		"out:suitnegro", "out:suitblanco", "out:suitrojo", "out:suitdorado",
		"out:hoodgris", "out:hoodrojo", "out:hoodverde",
		"out:trackroja", "out:trackazul", "out:tracknegra",
		"out:teeroja", "out:teenegra", "out:teeblanca",
		"out:tankazul", "out:tankamarilla",
		"out:sweatverde", "out:sweatazul",
		"out:hawaiirosa", "out:hawaiiazul",
		"out:swimazul", "out:swimrojo", "out:swimamarillo",
		"out:robemorada", "out:robenegra", "out:robedorada", "out:robeblanca",
		"out:armorgris", "out:armornegra", "out:armordorada",
		"out:vestamarillo",
		"out:dressrosa", "out:dressazul", "out:dressroja", "out:dressnegro",
		"out:uniform", "out:lab", "out:pijama", "out:prison", "out:zebra", "out:rainbow", "out:gold",
		// accessories (61): hats 26
		"hat:caproja", "hat:beanieazul", "hat:cowboy", "hat:fedora", "hat:bowler", "hat:sombrero", "hat:beret", "hat:ushanka",
		"hat:chef", "hat:tophat", "hat:pirate", "hat:viking", "hat:horns", "hat:wizard", "hat:fez", "hat:hardhat", "hat:partycone",
		"hat:turban", "hat:crown", "hat:bunny", "hat:pumpkin", "hat:dino", "hat:unicorn", "hat:tv", "hat:ninja", "hat:helmet",
		// hair 10, hair colours 5, beards 3, glasses 3, capes 3, shoes 3, skins 4, masks 4
		"hair:long", "hair:bald", "hair:mohawk", "hair:curly", "hair:pompadour", "hair:dreads", "hair:beehive", "hair:star", "hair:bob",
		"hair:baldmane", // v1.4.0: bald on top, a mane down to the floor (hair_mane.go)
		"haircol:rubio", "haircol:pelirrojo", "haircol:blanco", "haircol:azul", "haircol:rosa",
		"beard:full", "beard:mustache", "beard:goatee",
		"gl:shades", "gl:big", "gl:neon",
		"cape:roja", "cape:morada", "cape:dorada",
		"shoe:rojas", "shoe:doradas", "shoe:blancas",
		"skincol:verde alienígena", "skincol:azul", "skincol:dorada", "skincol:gris",
		"mask:hockey", "mask:ghost", "mask:clown", "mask:skull",
	} {
		shopKeepWear[id] = true
	}
}

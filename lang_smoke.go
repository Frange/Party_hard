package main

// English texts of the smoke grenade (smoke.go). Merged into langDataEN without overwriting anything.
func init() {
	for k, v := range map[string]string{
		"Recoger granada de humo":                  "Pick up smoke grenade",
		"GRANADA DE HUMO":                          "SMOKE GRENADE",
		"+ GRANADA x1":                             "+ GRENADE x1",
		"Granada de humo: pulsa %s para lanzarla":  "Smoke grenade: press %s to throw it",
		"¡Granada de humo!":                        "Smoke grenade!",
		"Ya hay demasiado humo: espera un poco":    "Too much smoke already: hang on a bit",
		"Guardas el frasco de veneno para después": "You keep the poison bottle for later",
		"VENENO x%d":                               "POISON x%d", // the OBJETO line of the HUD said VENENO in English too
		"*COF*":                                    "*COUGH*",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

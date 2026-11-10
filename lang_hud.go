package main

// English texts of the HUD panel (hud.go). Merged into langDataEN without overwriting anything lang_data.go has.
func init() {
	for k, v := range map[string]string{
		"GENTE":       "PEOPLE",
		"GUARDIAS %d": "GUARDS %d",
		"BAJAS %d/%d": "KILLS %d/%d",
		"BAJAS":       "KILLS", // the word of the kills row (hud.go), drawn apart from its figures
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

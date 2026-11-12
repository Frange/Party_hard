package main

// English texts of the ambulance that brings a dead partner back (revive.go). Merged into langDataEN without
// overwriting anything that already exists.
func init() {
	for k, v := range map[string]string{
		"Llamar a la ambulancia":                    "Call the ambulance",
		"Tu compañero %s ha muerto":                 "Your partner %s is dead",
		"Llama a la ambulancia en un teléfono [%s]": "Call the ambulance from a phone [%s]",
		"AMBULANCIA":                                    "AMBULANCE",
		"Ambulancia: %s %d · %s %d":                     "Ambulance: %s %d · %s %d",
		"Ambulancia en %d s":                            "Ambulance in %d s",
		"Reanimando a %s...":                            "Reviving %s...",
		"%s no se puede reanimar":                       "%s cannot be revived",
		"%s ya gastó su ambulancia":                     "%s already used his ambulance",
		"Llamas a la ambulancia: ¡aguanta!":             "You call the ambulance: hang in there!",
		"Llega la ambulancia":                           "The ambulance is here",
		"¿Me oye? ¿Hola?":                               "Can you hear me? Hello?",
		"¡CARGANDO!":                                    "CHARGING!",
		"¡CLEAR!":                                       "CLEAR!",
		"Nada... otra vez":                              "Nothing... again",
		"¿QUÉ ME HE PERDIDO?":                           "WHAT DID I MISS?",
		"¡Tu compañero ha vuelto de entre los muertos!": "Your partner is back from the dead!",
		"¡Respira! ¡Respira!":                           "Breathe! Breathe!",
		"La ambulancia se va sin nadie al que salvar":   "The ambulance leaves with nobody to save",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

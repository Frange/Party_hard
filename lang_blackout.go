package main

// English texts of the blackout event (blackout.go). Merged into langDataEN without overwriting anything.
func init() {
	for k, v := range map[string]string{
		"¡Se ha ido la luz! Seguro que ha sido el DJ con la tostadora.":    "The power is out! Surely the DJ and his toaster.",
		"¡Apagón! Alguien ha enchufado el horno y la rave a la vez.":       "Blackout! Somebody plugged in the oven and the rave at once.",
		"¡Clac! Vuelve la luz. Nadie sabe quién estaba abrazando a quién.": "Clack! The lights are back. Nobody knows who was hugging whom.",
		"¡Clac! Luz otra vez. Y alguien tiene el sombrero de otro.":        "Clack! Lights on. And somebody has somebody else's hat.",
		"¡Clac! Ya hay luz. El fusible pide perdón.":                       "Clack! Light at last. The fuse says sorry.",
		"¡AAAH!":        "AAAH!",
		"¡LA LUZ!":      "THE LIGHT!",
		"¡MAMÁ!":        "MOMMY!",
		"¿DÓNDE ESTOY?": "WHERE AM I?",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

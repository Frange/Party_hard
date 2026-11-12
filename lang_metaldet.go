package main

// English text of the metal detector arches (metaldet.go).
// TestPevTexts checks that every Spanish text they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		"¡Alto! ¿Qué llevas ahí?":            "Stop! What have you got there?",
		"¡Eh, tú! ¡Eso ha pitado!":           "Hey, you! That beeped!",
		"¡Control! ¡Quieto ahí!":             "Security check! Freeze!",
		"¿Quién ha hecho pitar eso?":         "Who made that thing beep?",
		"¡Ha pitado el detector de metales!": "The metal detector just went off!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}

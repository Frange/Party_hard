package main

// English texts of the guests sitting at the tables (seats.go). Merged into langDataEN without overwriting anything.
func init() {
	for k, v := range map[string]string{
		"¡Qué bien se está sentado!":         "Sitting down is the best!",
		"Mis pies me lo agradecen":           "My feet thank me",
		"¿Me guardas la silla?":              "Will you save my chair?",
		"Esta fiesta es lo más":              "This party is the best",
		"¿Has visto quién ha venido?":        "Did you see who showed up?",
		"Uf, necesitaba esto":                "Phew, I needed this",
		"Brindemos por el anfitrión":         "A toast to the host",
		"Cinco minutos y vuelvo a bailar":    "Five minutes and back to dancing",
		"¿Y tú de qué conoces al anfitrión?": "So how do you know the host?",
		"¡Qué buena está la música!":         "The music is so good!",
		"No me levanto ni a tiros":           "Not even gunfire gets me up",
		"Otra ronda y me voy":                "One more round and I'm off",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

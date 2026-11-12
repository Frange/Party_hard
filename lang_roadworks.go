package main

// English text of the roadworks and their crane (roadworks.go, crane.go). TestPevTexts checks that every Spanish text
// they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// the workers
		"¡Cuidado, obras!": "Careful, roadworks!", "¡Que no se acerque nadie!": "Nobody come any closer!",
		"¡Por la otra acera!": "Use the other pavement!", "¡Ojo con la zanja!": "Mind the trench!",
		"¡Dale caña al taladro!": "Give that drill some juice!", "Esta tubería no aguanta más": "This pipe has had it",
		"¿Quién ha pedido agua?": "Who ordered water?", "Uf, qué calor": "Phew, it's hot",
		"Media hora y al bocadillo": "Half an hour to sandwich time", "¡Pásame la llave inglesa!": "Pass me the wrench!",
		"¡Han matado a un compañero!": "They killed one of the crew!", "¡Se acabó la obra!": "The job's over!",
		"¡Que venga el capataz!": "Get the foreman!", "Bueno... a trabajar": "Well... back to work",
		"Venga, que hay que acabar": "Come on, we've got to finish", "¡Uy, por poco!": "Whoa, that was close!",
		// the crane
		"Soltar la carga": "Drop the load", "¡Cuidado abajo!": "Heads up below!",
		"¡Se suelta la carga de la grúa!":     "The crane's load comes loose!",
		"Te ha aplastado la carga de la grúa": "The crane's load squashed you",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}

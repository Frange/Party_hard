package main

// English texts of the wild animals (animals6.go) and of the new killer weapons of the shop (shopdata.go). Merged into
// langDataEN without overwriting anything lang_data.go already has.
var langWildEN = map[string]string{
	// animals: hover names, "Has matado a ..." and the newspaper actors
	"Rana": "Frog", "Ardilla": "Squirrel", "Jabalí": "Wild boar", "Cocodrilo": "Crocodile", "Guepardo": "Cheetah", "León": "Lion", "Serpiente": "Snake", "Cobaya": "Guinea pig", "una cobaya": "a guinea pig", "¡WHEEK!": "WHEEK!",
	"una rana": "a frog", "una ardilla": "a squirrel", "un jabalí": "a wild boar", "un cocodrilo": "a crocodile", "un guepardo": "a cheetah", "un león": "a lion",
	"Un cocodrilo": "A crocodile", "Un guepardo": "A cheetah", "Un león": "A lion",
	"Jabalíes": "Wild boars", "Cocodrilos": "Crocodiles", "Guepardos": "Cheetahs", "Leones": "Lions",
	"CROAC!": "RIBBIT!", "¡CHIC CHIC!": "CHITTER!", "¡GRUÑ!": "OINK GRR!", "¡ÑAM!": "CHOMP!", "¡GRRROAR!": "ROOOAR!", "¡FFSSS!": "HISSS!",
	"¡POR LOS PELOS!": "THAT WAS CLOSE!",
	"Jabalíes con muy mal despertar. Hozan el césped y gruñen a los camareros.":                       "Wild boars that woke up grumpy. They dig up the lawn and grunt at the waiters.",
	"Cocodrilos que toman el sol junto al agua. Un bocado y vuelven a la siesta.":                     "Crocodiles sunbathing by the water. One bite and back to their nap.",
	"Guepardos que vienen muy de vez en cuando: un sprint, un susto y se van sin despedirse.":         "Cheetahs that come once in a blue moon: one sprint, one scare, and off they go without a goodbye.",
	"Leones que se cuelan muy rara vez. Rugen, prueban suerte con un invitado y se marchan a dormir.": "Lions that sneak in very rarely. They roar, try their luck with one guest and go off to sleep.",
	// killer weapons
	"Taza de café hirviendo": "Scalding coffee mug", "Micrófono de karaoke": "Karaoke mic", "Móvil sin batería": "Dead phone",
	"Bandeja de canapés": "Canapé tray", "Cámara con flash": "Flash camera", "Copa de litro": "Litre glass",
	"Pompones de animadora": "Cheerleader pompoms", "Flotador salvavidas": "Life ring", "Palo de hockey": "Hockey stick",
	"Ramo de globos": "Bunch of balloons", "Tabla de socorrista": "Lifeguard board", "Extintor": "Fire extinguisher",
}

func init() {
	for k, v := range langWildEN {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for k, v := range map[string]string{
		"animal:boar":    "Jabalíes con muy mal despertar. Hozan el césped y gruñen a los camareros.",
		"animal:croc":    "Cocodrilos que toman el sol junto al agua. Un bocado y vuelven a la siesta.",
		"animal:cheetah": "Guepardos que vienen muy de vez en cuando: un sprint, un susto y se van sin despedirse.",
		"animal:lion":    "Leones que se cuelan muy rara vez. Rugen, prueban suerte con un invitado y se marchan a dormir.",
	} {
		funnyDesc[k] = v
	}
}

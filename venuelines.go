package main

import (
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// FRASES POR NIVEL: guests sometimes say something about the PLACE of the party ("¡qué marea!" on the
// cruise). It plugs into chatterLine (jokes.go): the normal chatter is chosen exactly as before (g.rng
// is not moved) and, with a low chance, replaced by a line of the level. The chance and the pick use a
// generator of their own; the same line never comes twice in a row. Max 48 characters per line.
// ---------------------------------------------------------------------------

const venueChance = 0.14

type venueLine struct{ es, en string }

// venueLines: per level icon (Def.Icon, campaign.go).
var venueLines = map[string][]venueLine{
	"house":    {{"¡Esta casa tiene hasta alfombra de pelo!", "This house even has a shag rug!"}, {"Quién deja un jarrón así en medio...", "Who leaves a vase right there..."}, {"El vecino ya ha tocado el timbre dos veces", "The neighbour has rung twice already"}, {"Mejor fiesta de salón desde mi cumple", "Best living room party since my birthday"}, {"¡No piséis el sofá con zapatos!", "Do not step on the sofa with shoes!"}, {"La nevera se vació en veinte minutos", "The fridge was empty in twenty minutes"}},
	"beach":    {{"Me pica el chiringuito", "The beach bar itches"}, {"¡Hay arena hasta en la cerveza!", "There is sand even in my beer!"}, {"Esa ola venía con intenciones", "That wave had intentions"}, {"Yo con crema 50 y aun así me achicharro", "SPF 50 and I still fry"}, {"¿Quién ha enterrado mis chanclas?", "Who buried my flip-flops?"}, {"Las gaviotas me miran raro", "The seagulls are looking at me funny"}, {"Prefiero el mar a la pista de baile", "I prefer the sea to the dance floor"}},
	"neon":     {{"Estas luces me dan jaqueca", "These lights give me a headache"}, {"Todo brilla, hasta mi mala cara", "Everything glows, even my bad mood"}, {"Mi camisa fluorescente es la mejor", "My fluorescent shirt is the best"}, {"Qué ciudad más ruidosa y bonita", "What a loud and pretty city"}, {"Neón, neón, neón... y más neón", "Neon, neon, neon... and more neon"}, {"No sé si es la luz o he bebido", "Is it the light or did I drink?"}},
	"bull":     {{"¡Olé! Pero desde lejos, por favor", "Olé! But from far away, please"}, {"Ese toro tiene mala leche", "That bull has a bad temper"}, {"Yo de torero solo tengo la chaqueta", "My only bullfighter part is the jacket"}, {"¡Que no me pille el de los cuernos!", "Do not let the horned one get me!"}, {"Cuánta arena y qué poca sombra", "So much sand and so little shade"}, {"Dicen que el rojo enfada. Yo me visto de azul", "Red makes them angry. I wear blue"}},
	"eiffel":   {{"Paris siempre es una buena idea", "Paris is always a good idea"}, {"Qué torre más alta y qué cola más larga", "What a tall tower and what a long queue"}, {"Un croissant y me caso con la fiesta", "One croissant and I marry the party"}, {"Oh la la, qué vistas", "Oh la la, what a view"}, {"Me han cobrado tres euros por el agua", "They charged me three euros for water"}, {"Aquí hasta las palomas visten bien", "Here even the pigeons dress well"}},
	"boat":     {{"¡Qué marea!", "I am so seasick!"}, {"El suelo se mueve y yo no he bebido (mucho)", "The floor moves and I have not (much) drunk"}, {"¿Hay lancha de emergencia? Pregunto", "Is there a lifeboat? Just asking"}, {"Si me caigo al mar, avisad a mi madre", "If I fall in the sea, tell my mother"}, {"Capitán, ¡otra ronda de bebidas!", "Captain, another round of drinks!"}, {"Yo no he sido el del ancla", "I was not the one with the anchor"}},
	"funfair":  {{"Mis tripas siguen en la montaña rusa", "My stomach is still on the roller coaster"}, {"Quiero un algodón de azúcar gigante", "I want a giant cotton candy"}, {"Esta noria va más lenta que mi abuelo", "This wheel is slower than my grandpa"}, {"¡Otro peluche que no me sale en la grúa!", "Another plush I cannot win in the claw!"}, {"Los payasos me dan un poco de miedo", "Clowns scare me a little"}, {"Qué música más pegadiza y repetitiva", "What catchy, repetitive music"}},
	"stadium":  {{"¡Que se retire el árbitro!", "The referee should retire!"}, {"Hoy mi equipo gana. Seguro. Quizá", "My team wins today. Surely. Maybe"}, {"Se me ha caído el bocata en la grada", "I dropped my sandwich in the stands"}, {"¡Fuera de juego! ...Creo", "Offside! ...I think"}, {"Cuánto césped y qué poca gente buena", "So much grass and so few good people"}, {"Mi bufanda de la suerte me mira mal", "My lucky scarf is giving me looks"}},
	"disco":    {{"¡Que suene otra, DJ!", "Play another one, DJ!"}, {"Con este ritmo me salen alas", "With this beat I grow wings"}, {"Mis oídos zumban, pero bailo", "My ears hum, but I dance"}, {"La bola de espejos me hipnotiza", "The mirror ball hypnotizes me"}, {"¿Alguien ha visto mi chaqueta de lentejuelas?", "Has anyone seen my sequin jacket?"}, {"Esto no es una discoteca, es mi hogar", "This is not a club, it is my home"}},
	"festival": {{"Esa banda ya suena a gloria", "That band already sounds divine"}, {"¡Hay barro hasta en el alma!", "There is mud even in my soul!"}, {"Mi pulsera me ha costado un riñón", "My wristband cost me a kidney"}, {"Me han pisado el pie dieciséis veces", "My foot has been stepped on sixteen times"}, {"Prometo ahorrar para el año que viene", "I promise to save for next year"}, {"¿Tocan la de siempre? ¡Que la toquen!", "Do they play that one? Let them play it!"}},
	"skate":    {{"Un ollie y me retiro", "One ollie and I retire"}, {"Esa rampa tiene mi nombre... y mis rodillas", "That ramp has my name... and my knees"}, {"¡Cuidado con la tabla!", "Watch out for the board!"}, {"Me caigo con estilo, que es lo importante", "I fall with style, that is what matters"}, {"Vine a ver trucos, me llevo un chichón", "I came for tricks, I got a bump"}, {"Los pantalones anchos se llevan, ¿no?", "Baggy pants are in, right?"}},
	"grave":    {{"Aquí descansan los que se aburrían", "Here rest those who were bored"}, {"Qué fiesta más de muerte... ya me callo", "Dead good party... I will shut up now"}, {"¿Alguien más oye pasos entre las tumbas?", "Does anyone hear steps among the graves?"}, {"Esa lápida no estaba antes ahí", "That gravestone was not there before"}, {"Prefiero no pisar nada sospechoso", "I would rather not step on anything odd"}, {"Hasta los fantasmas bailan aquí", "Even the ghosts dance here"}},
	"cinema":   {{"¡Shhh! Que empieza lo bueno", "Shhh! The good part is starting"}, {"Otra vez me he sentado detrás del alto", "I sat behind the tall one again"}, {"Las palomitas ya no saben igual", "Popcorn does not taste the same"}, {"Dicen que acaba bien. Mienten", "They say it ends well. They lie"}, {"Esa butaca está pegajosa, ojo", "That seat is sticky, careful"}, {"Hoy repetimos sesión, ¿no?", "We watch it again today, right?"}},
	"circuit":  {{"¡Vroom! A ver quién da más vueltas", "Vroom! Who can do more laps"}, {"Huele a gasolina y a victoria", "Smells like fuel and victory"}, {"Ese piloto frena como mi tía", "That driver brakes like my aunt"}, {"Boxes, ponedme ruedas nuevas", "Pit crew, give me new tyres"}, {"Hoy toca podio, o al menos bocata", "Podium today, or at least a sandwich"}, {"El ruido me ha dejado sordo, ¿qué?", "The noise left me deaf, what?"}},
	"snow":     {{"Mis dedos del pie son un recuerdo", "My toes are just a memory"}, {"¡Chocolate caliente o me rindo!", "Hot chocolate or I give up!"}, {"Me he caído esquiando. Dos veces. Hoy", "I fell skiing. Twice. Today"}, {"La nieve me entra por todas partes", "Snow gets in everywhere"}, {"Qué bonito el paisaje, qué frío mi nariz", "Pretty scenery, cold nose"}, {"Hago un muñeco de nieve y lo llamo Pepe", "I build a snowman and name him Bob"}},
	"roof":     {{"Desde aquí se ve toda la ciudad", "You can see the whole city from here"}, {"No me asomo ni muerto. Perdón", "I will not lean over, not even dead. Sorry"}, {"Hace un poco de viento, ¿no?", "A bit windy, isn't it?"}, {"Las antenas me dan mala espina", "The antennas give me the creeps"}, {"Cuidado con el borde, se escurre", "Careful with the edge, it is slippery"}, {"Qué cerca están las estrellas desde aquí", "The stars are so close from here"}},
	"casino":   {{"Doblo la apuesta y la dignidad", "I double the bet and my dignity"}, {"¡El diecisiete, que es mi día de suerte!", "Seventeen, it is my lucky day!"}, {"Una más y me voy a casa. Mentira", "One more and I go home. Lie"}, {"Esa ruleta me odia, es personal", "That roulette hates me, it is personal"}, {"Me juego la corbata a rojo", "I bet my tie on red"}, {"Las fichas pesan menos que mi cartera", "The chips weigh less than my wallet"}},
	"highway":  {{"Cruzar aquí es de valientes", "Crossing here is for the brave"}, {"Ese camión ha pasado rozando", "That truck just brushed past"}, {"Yo hago autoestop, pero con estilo", "I hitchhike, but with style"}, {"Cuánto ruido de motor, qué paz", "So much engine noise, such peace"}, {"Una gasolinera con baile, ¡pura vida!", "A gas station with dancing, pure life!"}, {"Mi coche está lejos, mi cerveza cerca", "My car is far, my beer is near"}},
	"train":    {{"El tren de las diez llega a las once", "The ten o'clock train arrives at eleven"}, {"¡Vía tres! ...o la cuatro. No sé", "Platform three! ...or four. No idea"}, {"He perdido el tren y ganado una fiesta", "I missed the train and won a party"}, {"Pasajeros, cuiden sus bolsillos y su copa", "Passengers, mind your pockets and your drink"}, {"Esta maleta pesa más que mi conciencia", "This suitcase weighs more than my conscience"}, {"¡Chu, chu! Perdón, no he podido evitarlo", "Choo choo! Sorry, I could not help it"}},
	"station":  {{"Sin gravedad el ponche sale volando", "With no gravity the punch flies away"}, {"Lo bueno del espacio: nadie oye tus bailes", "Good thing about space: nobody hears dancing"}, {"¡Cuidado con la escotilla, es traicionera!", "Mind the hatch, it is treacherous!"}, {"Mi casco es mi mejor complemento", "My helmet is my best accessory"}, {"Desde aquí la Tierra se ve pequeñita", "From here Earth looks so tiny"}, {"Houston, tenemos una fiesta", "Houston, we have a party"}},
}

var venueRng = rand.New(rand.NewSource(time.Now().UnixNano()))
var venueLast string

// venueTalk returns a line about the place with probability venueChance, or "" (the normal chatter stays).
func (g *Game) venueTalk() string {
	if g.lvl == nil || g.lvl.Def == nil || venueRng.Float64() >= venueChance {
		return ""
	}
	lines := venueLines[g.lvl.Def.Icon]
	if len(lines) == 0 {
		return ""
	}
	for try := 0; try < 4; try++ {
		l := lines[venueRng.Intn(len(lines))]
		if l.es != venueLast {
			venueLast = l.es
			return l.es
		}
	}
	return ""
}

func init() {
	for _, ls := range venueLines {
		for _, l := range ls {
			if _, ok := langDataEN[l.es]; !ok {
				langDataEN[l.es] = l.en
			}
		}
	}
}

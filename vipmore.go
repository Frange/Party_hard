package main

// ---------------------------------------------------------------------------
// 39 MORE VIPs (the registry has 50 in total with the 11 of vip.go). All of them are invented parodies of a kind of
// celebrity (a duke, a DJ, a tenor...): no real person, no real name. Each one has a name and a title in Spanish and
// English, a card name that fits the album card (cardES/cardEN), a rank, three lines (talk, talk, alarm), a
// behaviour and a look. The look is procedural: vipLookFor starts from a skin tone, hair tone and shoes taken from
// the palettes by the index of the VIP and then applies the pieces written in the row (hair style, hat, outfit,
// colours, beard, glasses, cape, what he carries), so two VIPs rarely share more than one piece.
// ---------------------------------------------------------------------------

type vipRow struct {
	id, es, en, cardES, cardEN, rankES, rankEN string
	beh, bonus, hp                             int
	l1, l2, l3                                 [2]string
	tip                                        [2]string // only for vbTip
	look                                       []po
}

// carryCol: what he holds, in a colour.
func carryCol(k int, c uint32) po {
	return func(s *CharSpec) { s.Carry, s.CarryCol = k, hx(c) }
}

// vipLookFor builds the spec of the i-th extra VIP.
func vipLookFor(i int, opts []po) CharSpec {
	s := vipBase()
	s.Skin = hx(skinTones[(i*3+1)%len(skinTones)])
	s.Hair = hx(hairTones[(i*5+2)%len(hairTones)])
	s.Shoes = hx(shoeTones[(i*2+1)%len(shoeTones)])
	s.Hair2 = HairShort
	s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopSuit, hx(0x1c1c2c), hx(0xf4f4f8), hx(0xc02030), hx(0x1c1c2c)
	for _, o := range opts {
		o(&s)
	}
	return s
}

func moreVIPs() []vipDef {
	var out []vipDef
	for i, r := range vipExtraRows {
		r, i := r, i
		d := vipDef{id: r.id, es: r.es, en: r.en, cardES: r.cardES, cardEN: r.cardEN, rankES: r.rankES, rankEN: r.rankEN,
			behaviour: r.beh, bonus: r.bonus, hp: r.hp, lines: [][2]string{r.l1, r.l2, r.l3},
			spec: func() CharSpec { return vipLookFor(i, r.look) }}
		if r.beh == vbTip {
			d.tipES, d.tipEN = r.tip[0], r.tip[1]
		}
		out = append(out, d)
	}
	return out
}

// the registry is complete before any init() runs (vip.go's init builds the English texts from it)
var _ = func() bool { vipDefs = append(vipDefs, moreVIPs()...); return true }()

var vipExtraRows = []vipRow{
	{id: "duque", es: "EL DUQUE DE LA TORTILLA", en: "THE DUKE OF OMELETTE", cardES: "DUQUE", cardEN: "DUKE", rankES: "Noble despistado", rankEN: "Clueless noble", beh: vbStand, bonus: 1400, hp: 1,
		l1:   [2]string{"Mi castillo solo tiene wifi en la torre", "My castle only has wifi in the tower"},
		l2:   [2]string{"Soy duque de una tortilla. Con cebolla", "I'm the duke of an omelette. With onion"},
		l3:   [2]string{"¡Mayordomo! ¡Hay intrusos en el salón!", "Butler! There are intruders in the hall!"},
		look: []po{hairc(0xd8d0c0, HairComb), beardc(BeardMustache), glass(GlassRound), tops(TopRobe, 0x5a2a7a, 0xe8c040), pantsc(0x3a1a4a), capec(0xa01830), carryCol(HoldTrophy, 0xf0c030)}},
	{id: "queso", es: "EL BARÓN DEL QUESO", en: "THE CHEESE BARON", cardES: "BARÓN", cardEN: "BARON", rankES: "Magnate lácteo", rankEN: "Dairy tycoon", beh: vbTip, bonus: 1500, hp: 1,
		tip:  [2]string{"Loncha de honor", "Slice of honour"},
		l1:   [2]string{"Mi fortuna huele fatal. Y es mía", "My fortune smells awful. And it's mine"},
		l2:   [2]string{"Hasta mis guardaespaldas están curados", "Even my bodyguards are well-aged"},
		l3:   [2]string{"¡Atacan mi bodega! ¡Seguridad!", "They attack my cellar! Security!"},
		look: []po{hairc(0xe8c030, HairBald), hatc(HatBowler, 0xe8b820), beardc(BeardFull), bodyc(BodyFat), tops(TopVest, 0xe8b820, 0xf4f0d0), pantsc(0x6a4a1a), carryCol(HoldPizza, 0xf0d040)}},
	{id: "djmega", es: "DJ MEGAVATIO", en: "DJ MEGAWATT", cardES: "MEGAVATIO", cardEN: "MEGAWATT", rankES: "DJ mundial", rankEN: "World-famous DJ", beh: vbFlee, bonus: 1100, hp: 1,
		l1:   [2]string{"Mi bajo hace temblar hasta las copas", "My bass shakes even the glasses"},
		l2:   [2]string{"Solo pincho con humo y luces", "I only spin with smoke and lights"},
		l3:   [2]string{"¡Mis altavoces! ¡Llamad al técnico!", "My speakers! Call the technician!"},
		look: []po{hairc(0x3aa0d0, HairSpiky), glass(GlassNeon), tops(TopHoodie, 0x14141a, 0x3aa0d0), pantsc(0x14141a), accentc(0x3aa0d0), carryCol(HoldMic, 0xd0d0d8)}},
	{id: "influencer", es: "LA INFLUENCER SUPREMA", en: "THE SUPREME INFLUENCER", cardES: "INFLUENCER", cardEN: "INFLUENCER", rankES: "Reina del filtro", rankEN: "Filter queen", beh: vbFlee, bonus: 1000, hp: 1,
		l1:   [2]string{"¡Un momento! Me falta mi luz bonita", "One sec! I need my pretty lighting"},
		l2:   [2]string{"Esta fiesta no existe si no la subo", "This party doesn't exist unless I post it"},
		l3:   [2]string{"¡Me han quitado el móvil! ¡Auxilio!", "They took my phone! Help!"},
		look: []po{fem(), hairc(0xd94a9a, HairLong), glass(GlassBig), tops(TopDress, 0xf060a0, 0xf4f4f8), shoesc(0xe8e8f0), carryCol(HoldSelfie, 0xf0f0f0), hatc(HatBunny, 0xf0a0c0)}},
	{id: "senador", es: "EL SENADOR LABIA", en: "SENATOR SMOOTHTALK", cardES: "SENADOR", cardEN: "SENATOR", rankES: "Político", rankEN: "Politician", beh: vbCall, bonus: 1500, hp: 1,
		l1:   [2]string{"Prometo todo. Cumpliré lo que pueda", "I promise everything. I'll deliver some"},
		l2:   [2]string{"Mi discurso dura tres horas. Te gustará", "My speech lasts three hours. You'll love it"},
		l3:   [2]string{"¡Esto es un atentado a la democracia!", "This is an attack on democracy!"},
		look: []po{hairc(0x6a6a74, HairComb), tops(TopSuit, 0x2a2a3a, 0xf4f4f8), accentc(0x2a4ab8), pantsc(0x2a2a3a), carryCol(HoldMegaphone, 0xc0c0c8), glass(GlassWhite)}},
	{id: "almirante", es: "EL ALMIRANTE BIGOTES", en: "ADMIRAL WHISKERS", cardES: "ALMIRANTE", cardEN: "ADMIRAL", rankES: "Marino de salón", rankEN: "Armchair sailor", beh: vbTough, bonus: 1700, hp: 3,
		l1:   [2]string{"He ganado mil batallas en la bañera", "I've won a thousand battles in the bath"},
		l2:   [2]string{"Mis bigotes tienen más medallas que tú", "My whiskers have more medals than you"},
		l3:   [2]string{"¡A mí la flota! ¡Al abordaje!", "To me, the fleet! Board them!"},
		look: []po{hairc(0xf0f0f0, HairBald), hatc(HatPirate, 0x1a2a5a), beardc(BeardMustache), bodyc(BodyBig), tops(TopUniform, 0x1a2a5a, 0xe8c040), accentc(0xe8c040), pantsc(0xf4f4f8), carryCol(HoldFlag, 0xc02030)}},
	{id: "condesa", es: "LA CONDESA DEL CAVA", en: "THE COUNTESS OF BUBBLY", cardES: "CONDESA", cardEN: "COUNTESS", rankES: "Aristócrata", rankEN: "Aristocrat", beh: vbTip, bonus: 1400, hp: 1,
		tip:  [2]string{"Brindis de condesa", "Countess's toast"},
		l1:   [2]string{"Mi sangre es azul. Mi copa, rosa", "My blood is blue. My glass, pink"},
		l2:   [2]string{"Brindo por lo que sea, querido", "I toast to anything, darling"},
		l3:   [2]string{"¡Mis joyas! ¡Que alguien haga algo!", "My jewels! Somebody do something!"},
		look: []po{fem(), hairc(0xe8d6a0, HairBun), tops(TopDress, 0x3a6ab8, 0xf4f4f8), accentc(0xe8c040), shoesc(0xe8c040), carryCol(HoldBigCup, 0xf0e090)}},
	{id: "tenor", es: "EL TENOR AGUDÍSIMO", en: "THE TOP-NOTE TENOR", cardES: "TENOR", cardEN: "TENOR", rankES: "Cantante de ópera", rankEN: "Opera singer", beh: vbFlee, bonus: 1100, hp: 1,
		l1:   [2]string{"Mi do de pecho rompe cristales", "My high C breaks glass"},
		l2:   [2]string{"Solo canto si hay aplauso previo", "I only sing after the applause"},
		l3:   [2]string{"¡Mi voz! ¡No me griten, por favor!", "My voice! Please don't shout at me!"},
		look: []po{hairc(0x1a1a22, HairCurly), beardc(BeardGoatee), bodyc(BodyFat), tops(TopSuit, 0x14141a, 0xf4f4f8), accentc(0xc02030), capec(0x6a1020), carryCol(HoldMic, 0xe8c040)}},
	{id: "diva", es: "LA DIVA DE LA TELENOVELA", en: "THE SOAP OPERA DIVA", cardES: "DIVA", cardEN: "DIVA", rankES: "Diva dramática", rankEN: "Drama diva", beh: vbFlee, bonus: 1200, hp: 1,
		l1:   [2]string{"¡Tú no eres mi padre! ¡Ni mi hijo!", "You are not my father! Nor my son!"},
		l2:   [2]string{"Lloro con un ojo. El otro es de pago", "I cry with one eye. The other costs extra"},
		l3:   [2]string{"¡Qué escándalo! ¡Llamad a mi agente!", "What a scandal! Call my agent!"},
		look: []po{fem(), hairc(0x14101a, HairBeehive), glass(GlassShades), tops(TopDress, 0xc02030, 0xe8c040), shoesc(0xc02030), capec(0x2a1a3a)}},
	{id: "titan", es: "EL TITÁN ENMASCARADO", en: "THE MASKED TITAN", cardES: "TITÁN", cardEN: "TITAN", rankES: "Campeón de lucha", rankEN: "Wrestling champ", beh: vbTough, bonus: 1900, hp: 4,
		l1:   [2]string{"Mi máscara no se quita. Ni en la ducha", "My mask stays on. Even in the shower"},
		l2:   [2]string{"Un abrazo mío y te quedas sin aire", "One hug from me and you run out of air"},
		l3:   [2]string{"¡Salto mortal... encima de ti!", "Somersault... right on top of you!"},
		look: []po{maskc(0xc02a2a, MaskPlain), bodyc(BodyGiant), tops(TopBare, 0xc98a5e, 0xc98a5e), bottomc(BotTrunks, 0xe8c040), shoesc(0xc02a2a), capec(0xe8c040)}},
	{id: "mago", es: "EL GRAN MAGO MEDIANOCHE", en: "THE GREAT MAGE MIDNIGHT", cardES: "MAGO", cardEN: "MAGE", rankES: "Ilusionista", rankEN: "Illusionist", beh: vbFlee, bonus: 1100, hp: 1,
		l1:   [2]string{"Mi conejo se ha fugado con mi cartera", "My rabbit ran off with my wallet"},
		l2:   [2]string{"¡Abracadabra! ¿Dónde está mi copa?", "Abracadabra! Where's my drink?"},
		l3:   [2]string{"¡Desaparezco! Es un truco... creo", "I vanish! It's a trick... I think"},
		look: []po{hatc(HatTopHat, 0x2a1a4a), hairc(0x1a1a22, HairComb), beardc(BeardGoatee), tops(TopSuit, 0x2a1a4a, 0xf4f4f8), accentc(0xe8c040), pantsc(0x2a1a4a), capec(0x6a2a8a), carryCol(HoldFlag, 0xe8c040)}},
	{id: "genio", es: "EL GENIO DE LAS APPS", en: "THE APP GENIUS", cardES: "GENIO", cardEN: "GENIUS", rankES: "Tecnólogo", rankEN: "Techie", beh: vbCall, bonus: 1500, hp: 1,
		l1:   [2]string{"Mi app te dice cuánta fiesta te queda", "My app tells you how much party is left"},
		l2:   [2]string{"Pivotar es mi religión", "Pivoting is my religion"},
		l3:   [2]string{"¡Han hackeado la fiesta! ¡Policía!", "The party's been hacked! Police!"},
		look: []po{hairc(0x4a2c17, HairMessy), glass(GlassRound), tops(TopHoodie, 0x5a5a66, 0x5a5a66), pantsc(0x2a3a6a), shoesc(0xe8e8f0), bodyc(BodySlim), carryCol(HoldPhone, 0x2a2a30)}},
	{id: "capitan", es: "EL CAPITÁN BARBAGEL", en: "CAPTAIN GELBEARD", cardES: "CAPITÁN", cardEN: "CAPTAIN", rankES: "Pirata de lujo", rankEN: "Luxury pirate", beh: vbTough, bonus: 1600, hp: 3,
		l1:   [2]string{"Mi barba lleva gel de tres océanos", "My beard has gel from three oceans"},
		l2:   [2]string{"¡Yo, ho, ho, y una botella de agua!", "Yo ho ho and a bottle of water!"},
		l3:   [2]string{"¡Al abordaje, grumetes de pega!", "Board them, you pretend deckhands!"},
		look: []po{hatc(HatPirate, 0x6a1020), hairc(0x1a1a22, HairLong), beardc(BeardFull), tops(TopLong, 0x6a1020, 0xe8e0c8), pantsc(0x14141a), shoesc(0x3a2a1a), capec(0x14141a), carryCol(HoldMachete, 0xd0d0d8)}},
	{id: "emperador", es: "EL EMPERADOR DEL SUSHI", en: "THE SUSHI EMPEROR", cardES: "EMPERADOR", cardEN: "EMPEROR", rankES: "Soberano", rankEN: "Sovereign", beh: vbStand, bonus: 1500, hp: 1,
		l1:   [2]string{"Mi imperio es crudo pero sabroso", "My empire is raw but tasty"},
		l2:   [2]string{"Solo como lo que llega en una barquita", "I only eat what arrives on a little boat"},
		l3:   [2]string{"¡Guardia imperial! ¡Al ataque!", "Imperial guard! Attack!"},
		look: []po{hatc(HatSamurai, 0xc02030), hairc(0x14101a, HairBun), beardc(BeardGoatee), tops(TopRobe, 0xc02030, 0xe8c040), pantsc(0x6a1020), carryCol(HoldSign, 0xf4f0e0)}},
	{id: "reina", es: "LA REINA DEL HIELO SECO", en: "THE DRY ICE QUEEN", cardES: "REINA", cardEN: "QUEEN", rankES: "Monarca fría", rankEN: "Cool monarch", beh: vbStand, bonus: 1500, hp: 1,
		l1:   [2]string{"Mi corte es muy fría. Y muy húmeda", "My court is very cold. And very damp"},
		l2:   [2]string{"Siempre entro envuelta en humo", "I always enter wrapped in smoke"},
		l3:   [2]string{"¡Guardias! ¡Congeladlos a todos!", "Guards! Freeze them all!"},
		look: []po{fem(), skinc(0xe8f0f8), hairc(0xe8f0f8, HairLong), hatc(HatCrown, 0x9ad0f0), tops(TopDress, 0x9ad0f0, 0xf4f8ff), accentc(0xf4f8ff), shoesc(0xf4f8ff), capec(0xd0e8f8), carryCol(HoldBalloons, 0xd0f0ff)}},
	{id: "general", es: "EL GENERAL MEDALLAS", en: "GENERAL MEDALS", cardES: "GENERAL", cardEN: "GENERAL", rankES: "Condecorado", rankEN: "Decorated", beh: vbTough, bonus: 1700, hp: 3,
		l1:   [2]string{"Cada medalla me la dieron por apuntarme", "Each medal was for signing up"},
		l2:   [2]string{"Mi bigote hizo retirarse al enemigo", "My moustache made the enemy retreat"},
		l3:   [2]string{"¡Tropa! ¡Formen y disparen a todo!", "Troops! Form up and shoot at everything!"},
		look: []po{hatc(HatHelmet, 0x3a4a2a), beardc(BeardMustache), bodyc(BodyBig), tops(TopUniform, 0x3a4a2a, 0xe8c040), accentc(0xe8c040), pantsc(0x2a3a1a), shoesc(0x14141a), carryCol(HoldSaber, 0xd0d0d8)}},
	{id: "doctor", es: "EL DOCTOR MILAGRO", en: "DOCTOR MIRACLE", cardES: "DOCTOR", cardEN: "DOCTOR", rankES: "Cirujano famoso", rankEN: "Famous surgeon", beh: vbCall, bonus: 1300, hp: 1,
		l1:   [2]string{"Opero en directo y con aplausos", "I operate live, with applause"},
		l2:   [2]string{"Mi bisturí lleva mi firma", "My scalpel carries my signature"},
		l3:   [2]string{"¡Un paciente agresivo! ¡Policía!", "A violent patient! Police!"},
		look: []po{hatc(HatMedic, 0x9ad0c0), glass(GlassWhite), hairc(0x2b1d12, HairComb), tops(TopLong, 0x9ad0c0, 0xf4f4f8), pantsc(0x7ab0a0), shoesc(0xf4f4f8), carryCol(HoldKnife, 0xd8e0e8)}},
	{id: "arquitecto", es: "EL ARQUITECTO TORCIDO", en: "THE CROOKED ARCHITECT", cardES: "ARQUITECTO", cardEN: "ARCHITECT", rankES: "Genio de la obra", rankEN: "Building genius", beh: vbTip, bonus: 1400, hp: 1,
		tip:  [2]string{"Plano firmado", "Signed blueprint"},
		l1:   [2]string{"Diseño rascacielos. Torcidos, pero míos", "I design skyscrapers. Crooked, but mine"},
		l2:   [2]string{"Esta fiesta necesita más columnas", "This party needs more columns"},
		l3:   [2]string{"¡Se cae el techo! ¡O es un ataque!", "The roof is falling! Or an attack!"},
		look: []po{hatc(HatHardhat, 0xf0c820), glass(GlassBig), hairc(0x1a1a22, HairBob), tops(TopVest, 0xf0c820, 0x2a3a5a), pantsc(0x2a3a5a), shoesc(0x3a2a1a), carryCol(HoldSign, 0xf4f4f0)}},
	{id: "modista", es: "LA MODISTA IMPOSIBLE", en: "THE IMPOSSIBLE DESIGNER", cardES: "MODISTA", cardEN: "DESIGNER", rankES: "Reina de la moda", rankEN: "Fashion queen", beh: vbFlee, bonus: 1100, hp: 1,
		l1:   [2]string{"Ese color es del año pasado, cariño", "That colour is so last year, darling"},
		l2:   [2]string{"Visto a medio mundo. Critico al otro", "I dress half the world. I mock the rest"},
		l3:   [2]string{"¡Mi vestido! ¡No toquen la seda!", "My dress! Don't touch the silk!"},
		look: []po{fem(), hairc(0xf0f0f0, HairBob), glass(GlassShades), tops(TopDress, 0xb050d0, 0xf060a0), shoesc(0xb050d0), capec(0xf060a0)}},
	{id: "delantero", es: "EL DELANTERO DE ORO", en: "THE GOLDEN STRIKER", cardES: "DELANTERO", cardEN: "STRIKER", rankES: "Futbolista", rankEN: "Footballer", beh: vbFlee, bonus: 1000, hp: 1,
		l1:   [2]string{"Mi pierna derecha está asegurada", "My right leg is insured"},
		l2:   [2]string{"Marco goles. Y selfies. Más selfies", "I score goals. And selfies. More selfies"},
		l3:   [2]string{"¡Árbitro! ¡Eso es falta!", "Referee! That's a foul!"},
		look: []po{hairc(0xd9a441, HairPompadour), tops(TopTee, 0xf0c030, 0xf0c030), bottomc(BotShorts, 0x14141a), shoesc(0xf4f4f8), carryCol(HoldTrophy, 0xf0c030), earsn(1)}},
	{id: "tenista", es: "LA CAMPEONA DEL SAQUE", en: "THE SERVE CHAMPION", cardES: "CAMPEONA", cardEN: "CHAMPION", rankES: "Tenista", rankEN: "Tennis star", beh: vbTough, bonus: 1700, hp: 3,
		l1:   [2]string{"Mi saque sale a doscientos por hora", "My serve flies at two hundred an hour"},
		l2:   [2]string{"Quince-amor, y no hablo de ti", "Fifteen-love, and I'm not talking to you"},
		l3:   [2]string{"¡Bola! ¡Falta! ¡Seguridad!", "Ball! Fault! Security!"},
		look: []po{fem(), hairc(0xd9a441, HairPony), hatc(HatCap, 0xf4f4f8), tops(TopTee, 0xf4f4f8, 0xf4f4f8), bottomc(BotSkirt, 0xf4f4f8), shoesc(0xf4f4f8), carryCol(HoldBroom, 0xe8e060)}},
	{id: "ajedrez", es: "EL MAESTRO DEL AJEDREZ", en: "THE CHESS MASTER", cardES: "MAESTRO", cardEN: "MASTER", rankES: "Estratega", rankEN: "Strategist", beh: vbStand, bonus: 1300, hp: 1,
		l1:   [2]string{"Voy tres jugadas por delante de ti", "I'm three moves ahead of you"},
		l2:   [2]string{"Mi caballo se come tu alfil", "My knight eats your bishop"},
		l3:   [2]string{"¡Jaque! ¡Jaque a la fiesta entera!", "Check! Check to the whole party!"},
		look: []po{hairc(0x6a6a74, HairBald), glass(GlassRound), beardc(BeardStubble), tops(TopVest, 0x14141a, 0xf4f4f8), pantsc(0x14141a), shoesc(0x14141a), carryCol(HoldTray, 0xe8e0c8)}},
	{id: "poeta", es: "EL POETA MALDITO", en: "THE CURSED POET", cardES: "POETA", cardEN: "POET", rankES: "Bardo", rankEN: "Bard", beh: vbTip, bonus: 1200, hp: 1,
		tip:  [2]string{"Verso gratis", "Free verse"},
		l1:   [2]string{"Nada rima con fiesta. Salvo mi pena", "Nothing rhymes with party. Except my pain"},
		l2:   [2]string{"Solo escribo de noche y gratis", "I only write at night, for free"},
		l3:   [2]string{"¡Oh, tragedia! ¡Policía, oh, policía!", "Oh, tragedy! Police, oh, police!"},
		look: []po{hatc(HatBeret, 0x14141a), hairc(0x1a1a22, HairMessy), beardc(BeardGoatee), bodyc(BodySlim), tops(TopLong, 0x14141a, 0x8a1a2a), pantsc(0x14141a), capec(0x3a2a4a), carryCol(HoldPlant, 0xc02030)}},
	{id: "ganadero", es: "EL GANADERO DE LUJO", en: "THE LUXURY RANCHER", cardES: "GANADERO", cardEN: "RANCHER", rankES: "Rey del ganado", rankEN: "Cattle king", beh: vbTip, bonus: 1400, hp: 1,
		tip:  [2]string{"Propina de vaquero", "Rancher's tip"},
		l1:   [2]string{"Mis vacas tienen spa y todo", "My cows have a spa and everything"},
		l2:   [2]string{"Este sombrero vale más que tu coche", "This hat is worth more than your car"},
		l3:   [2]string{"¡Cuatrero! ¡Me roban el ganado!", "Rustler! They're stealing my cattle!"},
		look: []po{hatc(HatCowboy, 0xf0e8d0), hairc(0x8a5a2b, HairShort), beardc(BeardSideburns), bodyc(BodyBig), tops(TopVest, 0x8a5a2a, 0xf4e8c8), accentc(0xe8c040), pantsc(0x2a3a6a), shoesc(0x3a2a1a), carryCol(HoldBeer, 0xc08020)}},
	{id: "detective", es: "EL DETECTIVE ESTRELLA", en: "THE STAR DETECTIVE", cardES: "DETECTIVE", cardEN: "DETECTIVE", rankES: "Sabueso", rankEN: "Sleuth", beh: vbCall, bonus: 1300, hp: 1,
		l1:   [2]string{"Elemental: el culpable es el mayordomo", "Elementary: the butler did it"},
		l2:   [2]string{"Siempre resuelvo el caso en el último minuto", "I always solve the case at the last minute"},
		l3:   [2]string{"¡Un crimen! ¡Llamad a mi amigo el poli!", "A crime! Call my cop friend!"},
		look: []po{hatc(HatFedora, 0x6a5a3a), glass(GlassRound), hairc(0x4a2c17, HairComb), beardc(BeardMustache), tops(TopRobe, 0xa89a7a, 0x8a7a5a), pantsc(0x4a3a2a), carryCol(HoldCamera, 0x3a3a46)}},
	{id: "cientifica", es: "LA CIENTÍFICA PREMIADA", en: "THE PRIZED SCIENTIST", cardES: "CIENTÍFICA", cardEN: "SCIENTIST", rankES: "Premio de cartón", rankEN: "Cardboard laureate", beh: vbFlee, bonus: 1200, hp: 1,
		l1:   [2]string{"Descubrí la fiesta. Tardé tres años", "I discovered the party. Took me three years"},
		l2:   [2]string{"Mi invento: el vaso que nunca se vacía", "My invention: the glass that never empties"},
		l3:   [2]string{"¡Mis experimentos! ¡Están sueltos!", "My experiments! They're loose!"},
		look: []po{fem(), hairc(0x2b1d12, HairBun), glass(GlassBig), tops(TopLong, 0xf4f4f8, 0xf4f4f8), pantsc(0x3a5aa8), shoesc(0x1a1a22), carryCol(HoldVacuum, 0x5ac05a)}},
	{id: "madrina", es: "LA MADRINA DEL BARRIO", en: "THE NEIGHBOURHOOD GODMOTHER", cardES: "MADRINA", cardEN: "GODMOTHER", rankES: "Jefa del barrio", rankEN: "Block boss", beh: vbCall, bonus: 1600, hp: 1,
		l1:   [2]string{"Aquí todo el mundo me debe un favor", "Everyone here owes me a favour"},
		l2:   [2]string{"Mi lasaña ha acabado con más guerras que la ONU", "My lasagna ended more wars than the UN"},
		l3:   [2]string{"¡Os vais a enterar! ¡Llamad a la poli!", "You'll regret this! Call the cops!"},
		look: []po{fem(), hairc(0x1a1a22, HairBun), glass(GlassShades), bodyc(BodyBig), tops(TopDress, 0x14141a, 0x14141a), accentc(0xe8c040), shoesc(0x14141a), capec(0x2a2a30)}},
	{id: "embajador", es: "EL EMBAJADOR DE NADA", en: "THE AMBASSADOR OF NOTHING", cardES: "EMBAJADOR", cardEN: "AMBASSADOR", rankES: "Diplomático", rankEN: "Diplomat", beh: vbCall, bonus: 1400, hp: 1,
		l1:   [2]string{"Represento a un país muy discreto", "I represent a very discreet country"},
		l2:   [2]string{"Tengo inmunidad. Y canapés", "I have immunity. And canapés"},
		l3:   [2]string{"¡Incidente diplomático! ¡Policía!", "Diplomatic incident! Police!"},
		look: []po{hairc(0x2b1d12, HairSide), beardc(BeardSideburns), tops(TopSuit, 0x1a3a2a, 0xf4f4f8), accentc(0xe8c040), pantsc(0x1a3a2a), carryCol(HoldFlag, 0x3a8a3a), glass(GlassWhite)}},
	{id: "alcaldesa", es: "LA ALCALDESA DEL CONFETI", en: "THE CONFETTI MAYOR", cardES: "ALCALDESA", cardEN: "MAYOR", rankES: "Edil festiva", rankEN: "Festive mayor", beh: vbStand, bonus: 1300, hp: 1,
		l1:   [2]string{"Corto la cinta de cualquier cosa", "I'll cut the ribbon on anything"},
		l2:   [2]string{"Mi lema: más confeti, menos impuestos", "My motto: more confetti, fewer taxes"},
		l3:   [2]string{"¡Orden en mi ayuntamiento! ¡Guardias!", "Order in my town hall! Guards!"},
		look: []po{fem(), hairc(0xb03a2e, HairBob), hatc(HatPartyCone, 0xe85a9a), tops(TopSuit, 0xe85a9a, 0xf4f4f8), accentc(0x3aa0d0), pantsc(0xe85a9a), shoesc(0x14141a), carryCol(HoldBalloons, 0xf0c030)}},
	{id: "artista", es: "EL ARTISTA CONTEMPORÁNEO", en: "THE CONTEMPORARY ARTIST", cardES: "ARTISTA", cardEN: "ARTIST", rankES: "Genio confuso", rankEN: "Confusing genius", beh: vbTip, bonus: 1300, hp: 1,
		tip:  [2]string{"Obra de arte", "Work of art"},
		l1:   [2]string{"Mi obra: un calcetín en un marco", "My piece: a sock in a frame"},
		l2:   [2]string{"Si no lo entiendes, es que es arte", "If you don't get it, it's art"},
		l3:   [2]string{"¡Mi instalación! ¡No es basura, es arte!", "My installation! It's not trash, it's art!"},
		look: []po{hatc(HatBeret, 0x5ac05a), hairc(0x9a4adc, HairMessy), glass(GlassNeon), tops(TopHawaiian, 0xf0c030, 0xe04040), pantsc(0x3aa0d0), shoesc(0xe04040), bodyc(BodySlim), carryCol(HoldSign, 0xf4f0e0)}},
	{id: "zen", es: "EL MAESTRO ZEN", en: "THE ZEN MASTER", cardES: "MAESTRO ZEN", cardEN: "ZEN MASTER", rankES: "Iluminado", rankEN: "Enlightened", beh: vbStand, bonus: 1400, hp: 1,
		l1:   [2]string{"Respira... y paga la consumición", "Breathe... and pay for your drink"},
		l2:   [2]string{"El silencio es oro. Mi tarifa, platino", "Silence is golden. My fee is platinum"},
		l3:   [2]string{"Mi paz interior se ha ido al garete", "My inner peace has gone down the drain"},
		look: []po{hairc(0xf0f0f0, HairBald), beardc(BeardFull), bodyc(BodySlim), tops(TopRobe, 0xe8a030, 0xe8a030), pantsc(0xe8a030), shoesc(0xe8a030), carryCol(HoldPlant, 0x3a8a3a)}},
	{id: "presentador", es: "EL PRESENTADOR ESTRELLA", en: "THE STAR HOST", cardES: "PRESENTADOR", cardEN: "HOST", rankES: "Rey del plató", rankEN: "Studio king", beh: vbFlee, bonus: 1100, hp: 1,
		l1:   [2]string{"¡Y ahora, una pausa para publicidad!", "And now, a word from our sponsors!"},
		l2:   [2]string{"Mis dientes salen en tres canales", "My teeth appear on three channels"},
		l3:   [2]string{"¡Cámara! ¡Corten! ¡Esto no está en el guion!", "Camera! Cut! This isn't in the script!"},
		look: []po{hairc(0x1a1a22, HairPompadour), tops(TopSuit, 0xb050d0, 0xf4f4f8), accentc(0xf0c030), pantsc(0xb050d0), shoesc(0xf4f4f8), carryCol(HoldMic, 0xd0d0d8)}},
	{id: "princesa", es: "LA PRINCESA DEL POP", en: "THE POP PRINCESS", cardES: "PRINCESA", cardEN: "PRINCESS", rankES: "Cantante pop", rankEN: "Pop singer", beh: vbFlee, bonus: 1100, hp: 1,
		l1:   [2]string{"Mi nuevo disco: doce versiones de una canción", "My new album: twelve versions of one song"},
		l2:   [2]string{"No bailo sin mis diez bailarines", "I don't dance without my ten dancers"},
		l3:   [2]string{"¡Mis fans! ¡Que me saquen de aquí!", "My fans! Somebody get me out!"},
		look: []po{fem(), hairc(0x3aa0d0, HairBuns), hatc(HatCrown, 0xf060a0), tops(TopDress, 0xf060a0, 0xf4f4f8), shoesc(0xf0c030), accentc(0xf0c030), carryCol(HoldMic, 0xf060a0)}},
	{id: "catedratico", es: "EL CATEDRÁTICO", en: "THE PROFESSOR", cardES: "CATEDRÁTICO", cardEN: "PROFESSOR", rankES: "Sabio", rankEN: "Scholar", beh: vbCall, bonus: 1200, hp: 1,
		l1:   [2]string{"Mi tesis: mil páginas y tres lectores", "My thesis: a thousand pages, three readers"},
		l2:   [2]string{"Esta fiesta suspende en logística", "This party fails at logistics"},
		l3:   [2]string{"¡Expulsión inmediata! ¡Llamad al decano!", "Immediate expulsion! Call the dean!"},
		look: []po{hairc(0xf0f0f0, HairMessy), glass(GlassBig), beardc(BeardFull), tops(TopRobe, 0x2a2a3a, 0x8a1a2a), pantsc(0x2a2a3a), hatc(HatTyrol, 0x2a2a3a), carryCol(HoldBriefcase, 0x6a4a2a)}},
	{id: "kebab", es: "EL REY DEL KEBAB", en: "THE KEBAB KING", cardES: "REY KEBAB", cardEN: "KEBAB KING", rankES: "Empresario", rankEN: "Entrepreneur", beh: vbTip, bonus: 1300, hp: 1,
		tip:  [2]string{"Kebab de cortesía", "Complimentary kebab"},
		l1:   [2]string{"Mi imperio gira, como el kebab", "My empire spins, like the kebab"},
		l2:   [2]string{"Salsa picante o muy picante. Tú eliges", "Spicy sauce or very spicy. Your pick"},
		l3:   [2]string{"¡Me roban la carne! ¡Al ladrón!", "They're stealing my meat! Thief!"},
		look: []po{hatc(HatFez, 0xc02030), hairc(0x1a1a22, HairShort), beardc(BeardMustache), bodyc(BodyFat), tops(TopVest, 0xf4f4f8, 0xc02030), accentc(0xe8c040), pantsc(0x2a2a3a), carryCol(HoldPizza, 0xc08040)}},
	{id: "gatera", es: "LA MILLONARIA DE LOS GATOS", en: "THE CAT MILLIONAIRESS", cardES: "MILLONARIA", cardEN: "CAT LADY", rankES: "Amante de gatos", rankEN: "Cat lover", beh: vbTip, bonus: 1200, hp: 1,
		tip:  [2]string{"Caricia de gato", "Cat's purr"},
		l1:   [2]string{"Mis gatos heredan más que mis hijos", "My cats inherit more than my kids"},
		l2:   [2]string{"No hay fiesta sin pelos en el vestido", "No party without fur on the dress"},
		l3:   [2]string{"¡Mis gatitos! ¡Han asustado a mis gatitos!", "My kitties! You scared my kitties!"},
		look: []po{fem(), hairc(0xe8e8f0, HairBeehive), hatc(HatCat, 0x8a8a96), glass(GlassBig), tops(TopDress, 0x9a4adc, 0xe8c040), accentc(0xe8c040), shoesc(0x9a4adc), capec(0xf0e0f8)}},
	{id: "sumo", es: "EL GRAN CAMPEÓN DE SUMO", en: "THE SUMO GRAND CHAMP", cardES: "SUMO", cardEN: "SUMO", rankES: "Luchador", rankEN: "Wrestler", beh: vbTough, bonus: 1800, hp: 4,
		l1:   [2]string{"Peso lo que un coche pequeño", "I weigh as much as a small car"},
		l2:   [2]string{"Mi empujón es famoso en tres países", "My shove is famous in three countries"},
		l3:   [2]string{"¡Fuera del círculo! ¡Todos fuera!", "Out of the ring! Everybody out!"},
		look: []po{hairc(0x14101a, HairBun), bodyc(BodyObese), tops(TopBare, 0xf0c8a0, 0xf0c8a0), bottomc(BotTrunks, 0xe8e0d0), skinc(0xf0c8a0), shoesc(0x14141a)}},
	{id: "polar", es: "EL EXPLORADOR POLAR", en: "THE POLAR EXPLORER", cardES: "EXPLORADOR", cardEN: "EXPLORER", rankES: "Aventurero", rankEN: "Adventurer", beh: vbFlee, bonus: 1200, hp: 1,
		l1:   [2]string{"Crucé el polo con una sola galleta", "I crossed the pole on a single cookie"},
		l2:   [2]string{"Aquí hace calor. Qué raro todo", "It's hot in here. How strange"},
		l3:   [2]string{"¡Un oso! ¡Seguro que es un oso!", "A bear! It must be a bear!"},
		look: []po{hatc(HatUshanka, 0x6a6a74), beardc(BeardFull), bodyc(BodyBig), tops(TopHoodie, 0xe04040, 0xe04040), pantsc(0x3a3a46), shoesc(0x3a2a1a), carryCol(HoldFlag, 0x3aa0d0)}},
	{id: "inventor", es: "EL INVENTOR LOCO", en: "THE MAD INVENTOR", cardES: "INVENTOR", cardEN: "INVENTOR", rankES: "Genio chiflado", rankEN: "Mad genius", beh: vbCall, bonus: 1300, hp: 1,
		l1:   [2]string{"Mi último invento: la tostada con wifi", "My latest: the toast with wifi"},
		l2:   [2]string{"Funciona. A veces. Cuando quiere", "It works. Sometimes. When it wants"},
		l3:   [2]string{"¡Mi máquina se ha escapado! ¡Policía!", "My machine escaped! Police!"},
		look: []po{hairc(0xf0f0f0, HairSpiky), glass(GlassNeon), tops(TopLong, 0xf4f4f8, 0xf4f4f8), pantsc(0x5a5a66), shoesc(0x7a2a2a), capec(0xf4f4f8), carryCol(HoldTorch, 0x5ac05a), headc(HeadBig)}},
}

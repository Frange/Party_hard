# Party Hard (clon) con editor de sprites y escenarios

Windows: `PartyHard.exe` (juego) y `PartyHardEditor.exe` (editor). Mac: `PartyHard.app` y `PartyHardEditor.app`.
La carpeta `assets` debe estar junto a ellos (en Mac, junto a las apps). En Mac, la primera vez haz clic derecho > Abrir.
Desde el juego, **F2** abre el editor; en el editor, "Volver al juego (F2)" vuelve.

## Proporciones de los personajes
Cabeza pequeña (5-6 px), tronco de 9-10 px, piernas largas (unos 15 px), brazos finos de 1-2 px, unos 10 px de ancho y 33-35 de alto en total.
Los personajes especiales pueden tener la cabeza algo más grande para reconocerlos mejor.

## Todo el contenido está en carpetas
El juego lee personajes, objetos, animales y escenarios de `assets/` (formato en `ASSETS.md`).
Cámbialos con el editor o con cualquier programa de pixel art y se usan en el siguiente arranque, sin tocar código.
Si borras `assets`, se vuelve a crear con el contenido por defecto.

## Editor
* **Personajes**: pinta el muñeco de pie (frente, perfil, espalda) marcando qué es cabeza, torso, brazos y piernas.
  Las animaciones (andar, correr, bailar, beber, apuñalar, DJ, batería...) se generan solas y se ven en la vista previa.
  Cualquier fotograma se puede repintar a mano. Categorías: protagonista, invitados, famosos, guardias, guardias especiales, policía/sanitarios.
* **Objetos / animales**: sprite con fotogramas animados, ancla, huella de colisión y propiedades (sólido, esconderse, teléfono, ponche
  envenenable, trampa explosiva o eléctrica, farola, puntos de uso, cruza la calle...).
* **Escenarios**: pinta el fondo píxel a píxel, dibuja la lógica (paredes, vallas, agua, zonas de paseo y de baile), coloca objetos,
  inicio del jugador, ruta de guardias, DJ y músicos, luces y llegada de la policía (coche o helicóptero). Botón "Probar".
* Atajos: P lápiz, E goma, L línea, R rectángulo, O elipse, F cubo, I cuentagotas, M simetría, Ctrl+Z/Ctrl+Y, Ctrl+S, rueda = zoom.
* Plantillas: carga cualquiera de los personajes o escenarios por defecto como punto de partida.
* **Importar tu propio arte**: arrastra uno o varios PNG a la ventana del editor. En personajes se colocan en la vista actual
  (o por nombre de fichero: front/side/back) y las partes del cuerpo se asignan por la silueta (botón 'Auto-asignar partes').
  En objetos se convierten en fotogramas y en escenarios en el fondo.

## Juego
* Menú: flechas + ENTER (o 1-9), R escenario aleatorio, F2 editor.
* WASD / flechas: moverse. SHIFT: correr. ESPACIO: apuñalar. F: bailar. E: acción (cargar cuerpo, esconderse, trampas, ponche, teléfono).
* Mata a todos (invitados, guardias, DJ y músicos). Todos ven solo hacia delante. Una persona llama a la policía; llega un policía
  (y una ambulancia si hay muchos cuerpos) que mete los cuerpos en bolsas. Si te persigue verás esposas sobre tu cabeza; se cansa y se va.
  Si te atrapa la policía o los guardias, fin del juego.
* Llega un policía (en coche o helicóptero): primero busca al asesino por la zona, envuelve en bolsas los cuerpos que ve y al final
  recoge los que queden. Camina más rápido que tú y si te ve corre hacia ti, un poco más rápido que tu carrera pero se cansa.
  Con varios cuerpos viene una ambulancia con hasta 3 sanitarios que los cargan.
* Los animales viven su vida: los caballos pasean y pastan y dan una coz mortal a quien se queda detrás, las gallinas picotean y huyen
  asustadas, los perros merodean y ladran. Animales configurables en el editor (Puntos > Animales).
* Pasan coches, motos, tanques, jinetes y ovnis. Los invitados charlan, se besan, beben, vomitan, duermen y bailan.

## Compilar
Requiere Go. `./build.sh` genera juego y editor para Windows y Mac en `dist/` (el editor es el mismo código con `-tags editor`).
Depuración: `-level N`, `-mute`, `-check`, `-selftest N`, `-shot f.png`, `-noassets`, `-editor`.

## Novedades (v6)

- **Sin contorno**: los personajes ya no llevan el borde de 1 px; son más finos, con sombreado interior.
- **Más de 300 personajes**: arquetipos de fiesta (cani, ravero, vikingo, mariachi, ninja, sumo, hinchables, turistas, oficinistas…), parodias de cine/series/cómics/videojuegos (Hulk, Spider-Man, Terminator, RoboCop, Sonic, Crash, Mario, Lara Croft…). No hay personas reales vivas como objetivos.
- **Enemigo especial** (máximo 1 por nivel): llega a mitad de partida en un coche fúnebre (o helicóptero negro en las azoteas), mata invitados y guardias, y te mata si te acercas. Algunos arrastran una motosierra con chispas. Puedes matarlo (3 puñaladas) o la policía lo detiene. Cuenta para el total. Sus sprites están en `assets/sprites/special_enemies` (rol `killer`).
- **Multitud más lista**: tardan un momento en darse cuenta, uno llama, los demás huyen unos segundos y vuelven; si hay un policía cerca, los testigos corren a decirle quién fue y dónde.
- **Refuerzos**: con muchos cuerpos llega un segundo agente con otro aspecto. Policías y médicos atascados se recolocan solos.
- **Piscinas**: se puede entrar a nadar (jugador, invitados, policía y médicos); el jugador va más lento.
- **Llegadas tardías**: limusinas (calle) y ascensores (azotea) traen 2-5 invitados más; el total a matar se actualiza.
- **Escenarios generados**: en el menú, tecla **G** crea un escenario nuevo (casa con habitaciones y puertas, azotea o playa), siempre comprobado para que se pueda andar por todo.

## Novedades (v7)

- **Detección de cuerpos**: un cuerpo solo cuenta como detectado cuando un testigo vivo avisa (llamada o hablando con un policía). Si el testigo muere antes, el aviso desaparece y otra persona o un policía debe verlo. Solo huyen quienes vieron el crimen o un cuerpo sin detectar, y corren algo más que el protagonista.
- **Policía**: revisan pocos cuerpos (máximo 2 cada uno) y llaman a la ambulancia (1 a 3 sanitarios según los cuerpos); con muchos cuerpos llega un segundo agente. Si no encuentran al asesino se van y se pierde el aviso de las esposas. A veces vienen **SWAT** (4, furgón) o **FBI** (Mulder y Scully): nunca mezclados con la policía normal, más brutos, disparan a invitados si no dan con el culpable y se los llevan.
- **Tanque**: si cruza la calle revienta los coches aparcados y los vehículos de emergencias. Si la tripulación aún estaba dentro, muere y no sale.
- **Helicóptero**: la tripulación sale justo en la puerta.
- **DJ en todos los niveles**. Si lo matas, la música pasa a punk rock. Cada nivel tiene su propia lista de canciones (house, techno, disco, reggaetón…) que va cambiando; el menú mantiene el tema original.
- **Animales**: caballo y perro más pequeños; nuevos gato, pollitos, toro (embiste), elefante (echa agua), jirafa y canguro (noquea).
- **16 escenarios**: plaza de toros, discoteca, pub, cine sin butacas, estadio, plaza europea con fuente, París con torre Eiffel, festival, parque de atracciones, supermercado y centro comercial, además de los 5 clásicos. Con tecla **G** se genera otro nuevo. Pasillos anchos, puertas a todas las habitaciones y armarios solo contra las paredes.
- **Gags**: selfies, guitarra y batería imaginarias, tropiezos y un espontáneo en calzoncillos.

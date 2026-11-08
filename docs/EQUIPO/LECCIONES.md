# Lecciones aprendidas (Party Hard)

Lista corta que TODO agente (implementación, QA, build, limpieza, orquestador) lee al EMPEZAR y respeta siempre.
Cada regla lleva entre paréntesis qué pasó. QA la usa como checklist: marca cada casilla que aplique a la entrega.

**Cómo añadir una lección nueva:** si descubres un fallo nuevo (o el usuario te corrige), añade UNA línea en su sección con el formato
`- [ ] Regla clara (qué pasó, AAAA-MM-DD)`. Busca antes con grep para no duplicar; si ya existe, no la repitas. Sin nada personal.
Historial de pruebas y veredictos: `docs/QA/MEMORIA.md` (si existe; no se duplica aquí, y ese archivo debe enlazar de vuelta a este).

## GIT: NADIE LO TOCA
GIT: NADIE LO TOCA. Solo el usuario hace commits, push y cambios del índice. Ningún agente (implementación, QA, build, limpieza, orquestador) ejecuta git salvo LECTURAS (status, diff, log, ls-files, check-ignore, stash list). PROHIBIDO: commit, push, add, rm (también `git rm --cached`), stash (y stash pop), reset, checkout/restore, clean, rebase, merge, branch, tag, y editar .gitignore sin que el usuario lo pida. Si algo requiere git (p. ej. dejar de seguir un archivo, borrar binarios antiguos del repo), díselo al usuario con el comando exacto y que lo ejecute él. Motivo: un agente hizo `git stash` por error y tocó el índice; el usuario dijo "el GIT no tiene que tocarlo nadie". Todo brief a un agente debe incluir esta regla.

## Visual y texto
- [ ] Texto nítido: escalas y posiciones enteras, sin suavizado (texto borroso en pantalla).
- [ ] Ningún texto se pisa con otro, se sale de su panel ni tapa el logo (textos solapados en menús).
- [ ] Textos largos en 2 líneas máx.; frase de bocadillo <= 48 caracteres (bocadillos desbordados).
- [ ] Toda cadena nueva en ES y EN: `tr()` + `lang_data.go`, con gracia (textos solo en un idioma).
- [ ] Textos de cinemáticas con cámara/zoom: nunca sobre el lienzo pequeño (se emborronan al escalar); ponerlos en cola y dibujarlos en pantalla con la cámara del fotograma (`endState.sayAt`, ending2.go; el audit `TestUITextAudit` los cazó, 2026-11-08).
- [ ] Sin puntitos blancos de fondo en cinemáticas (ruido que parecía estrellas).
- [ ] Sin personajes transparentes, X rojas ni triángulos sueltos (artefactos de render). EXCEPCIÓN: los FANTASMAS (`drawGhostActor`, ghostfx.go) llevan alfa bajo a propósito para parecer fantasmas, no los vivos (2026-11-04).
- [ ] Edificios y fondos se dibujan SIEMPRE antes que los personajes (casa del vecino pintada delante de los fantasmas; test de capas `TestScenesLayerOrder`, 2026-11-04).
- [ ] Personajes siempre bien visibles sobre el fondo (contraste perdido).
- [ ] Objeto sostenido en UNA mano y sin duplicar; miembros sin romper (armas dobles, brazos cortados).
- [ ] Vehículos con el morro en el sentido de la marcha; vista de arriba/abajo en carreteras verticales (coches de lado).
- [ ] Letras del cartel del mismo tamaño y centradas (cartel descuadrado).

## Niveles y objetos
- [ ] Ningún objeto sobre otro, ni sobre vallas, paredes, carreteras, mar o vacío (props solapados).
- [ ] Cabinas de teléfono nunca ocultas por transparencias (cabina invisible).
- [ ] Objetos y suelos no cruzan la frontera entre dos suelos: helipuertos, barras (helipuerto a caballo).
- [ ] Isla Nublar, escala coherente: personajes ~0,7; objetos menores que el T-Rex; señales pequeñas; puerta casi opaca.
- [ ] El modo 1 jugador no cambia al añadir cosas del 2P (regresión al meter el 2P).

## Juego
- [ ] Nada se queda atascado más de 5 s: se recoloca (NPC bloqueados).
- [ ] Invitados 45-60, tope estricto; con 2 jugadores +20-30 (fiestas vacías o saturadas).
- [ ] Famosos una sola vez por nivel (famoso repetido).
- [ ] Animales: máx. 8 por nivel y 1 toro, tope único en `capAnimals` (cap.go) al final de `spawnAnimals`; cualquier ruta nueva que añada animales va ANTES de esa llamada (zoo/granja/críticas/jinetes sumaban más de 8, 2026-11-08).
- [ ] Estilo de movimiento de un famoso = solo dibujo (gait.go: animaciones ocultas tras AnimCount + `gaitDir`); no tocar velocidad, hitbox ni `gu.dir`; slug nuevo en `gaitSlugs` y que exista en famousPresets (2026-11-08).
- [ ] Máx. 4 bocadillos a la vez; sin frases repetidas a la vez ni mensajes de info repetidos en 20 s (spam de textos).
- [ ] Asesinos y T-Rex van a la persona más cercana que ven de frente, sin prioridad del jugador (perseguían siempre al jugador).
- [ ] Las esposas van al último visto matando; 2P tiene reglas propias (esposas al sospechoso equivocado).
- [ ] Las esposas se ponen cuando un testigo AVISA a un poli, no cuando te ve; mientras tanto hay una alerta sobre el jugador; matar al testigo antes de que avise te libra (origen: regla del usuario, 2026-11-04).
- [ ] Tipos de policía se desbloquean por asesinatos (desbloqueo mal ligado).
- [ ] Un jugador no mata al otro, solo golpea (fuego amistoso en 2P).

- [ ] Zona VIP: con el VIP vivo los guardias escalan por jugador/invitado (suelo, estrellitas, muerte; vipwarn.go); al matarlo vuelve el castigo antiguo. Tiempos VIP (pase 15 s, relevo 3,5 s) son constantes nombradas en vip.go (2026-11-08).

## Hacer lo pedido y nada más
- [ ] "Lo mismo que ya teníamos" / "el mismo coche" = reutilizar el activo existente; no inventar coches, personajes ni estilos; en duda, preguntar (coche explosivo rediseñado sin pedirlo; refuerzo 2026-11-04).
- [ ] Si dudas a qué se refiere el usuario, dilo en el informe en vez de inventar (suposiciones tomadas como hechos).

## Audio
- [ ] Sin pitidos repetitivos (pitido en bucle).
- [ ] Cada nivel con su género musical (todos sonaban igual).
- [ ] El "pum pum" solo en la primera intro (se repetía en otras).
- [ ] Audio NUNCA verificado si no lo ha oído una persona: escribir "no oído" (se dio por bueno sin oírlo).

## Datos y seguridad
- [ ] Nunca tocar `assets/progress.json` (partida real): usar progressFrozen o copias (se pisó progreso).
- [ ] No tocar `imports/`, `intros/`, `music/` ni tmp de otros proyectos (borrado ajeno).
- [ ] No commit/push salvo petición explícita (commits no pedidos).
- [ ] Nada personal ni música en el repo (repo público).
- [ ] Zips sin `progress.json`, settings ni `editor_prefs` (datos del usuario empaquetados).
- [ ] Toda bandera de depuración/captura debe congelar el progreso ANTES de cargar ningún nivel; las capturas de agentes se hacen desde una carpeta temporal con COPIA de assets; nunca desde el proyecto (origen: -level N -shot inflaba played de la partida real) (2026-11-04)

## Proceso
- [ ] Lanzar el juego solo con watchdog: macOS no tiene `timeout`; usar `&` y `kill` a ~40 s (juego colgado).
- [ ] No dejar procesos ni ventanas abiertas; borrar temporales (restos tras las pruebas).
- [ ] Una sola pasada completa de `go test -count=1` y build por entrega, en BUILD (pasadas repetidas que gastan tiempo y disco).
- [ ] Ediciones quirúrgicas en archivos compartidos y releer justo antes de editar (pisadas entre agentes).
- [ ] No tocar archivos de agentes concurrentes (conflictos).
- [ ] Informe de máx. 15 líneas: QUÉ HICE / EVIDENCIAS (comandos y resultados reales) / NO VERIFICADO / AVISOS (informes inflados sin pruebas).
- [ ] QA da veredicto PASA / PASA CON RESERVAS / NO PASA, tras ejecutar él mismo tests y capturas.

## Pruebas y entrega
- [ ] Un test que cambia un global (noAssets, twoPlayerMode, dbgTwoPlayers, progressFrozen, testStartLevel, audioCtx...) lo restaura SIEMPRE con `t.Cleanup` (ghostfx_test.go dejó noAssets=true y rompió TestTitleThenIntroTheFirstTime en la pasada completa, 2026-11-04).
- [ ] Si el cambio toca game.go, estado global o más de un área, QA exige la pasada completa `go test -count=1 -timeout 20m .` antes de PASA; los `-run` no detectan contaminación entre tests (el QA filtrado dio PASA y la pasada completa del build cazó un fallo, 2026-11-04).
- [ ] Incidencia que no se reproduce = NO REPRODUCIDA, con los pasos intentados anotados; no se da por cerrada (capturas del cohete contaminadas por procesos paralelos, 2026-11-04).

- Las capturas/ejecuciones con ventana solo se lanzan cuando el orquestador/usuario lo autoriza, en UNA tanda anunciada; el usuario puede estar jugando; preferir tests sin ventana (origen: QA abrió ventanas continuamente mientras el usuario jugaba, 2026-11-04). Usa el modo silencioso (automático con banderas de depuración o PH_QUIET=1: ventana fuera de pantalla, sin foco, prioridad baja) y -visible para lo contrario. Pasadas pesadas: `nice -n 19 go test -count=1 -p 2 -timeout 20m .`, `GOMAXPROCS=4`, `nice -n 19 ./build.sh`, o `tools/ph_quiet.sh <cmd>` (tardan más pero no bloquean el PC).
- 2026-11-04: MinimizeWindow impide dibujar/capturar en macOS tras ~25 ticks; usar ventana aparcada fuera de pantalla (SetWindowPosition(-30000,-30000), sin decoración, passthrough, InitUnfocused), nunca minimizar.
- 2026-11-05 (Tests): poner `noAssets=true` antes de `auditGame(t)` rompe su restauración con `t.Cleanup` (LIFO): deja que `auditGame` lo gestione.
- 2026-11-05: `assets/progress.json` ya no está en el seguimiento de git (`git rm --cached`; sigue en .gitignore y fuera de los zips); el juego crea uno nuevo (Unlocked=1) si falta.
- 2026-11-08: `perl -e 'alarm N; exec ...'` NO mata el juego (Go ignora SIGALRM aquí): usar también `kill $pid` desde el shell; una captura puede colgarse sin PNG si hay otras ventanas de agentes abiertas (reintentar sola).
- 2026-11-08 (Tests): `levelDefs` solo está completo tras crear un Game (`layerGame(t,false)`); los colocadores de `ensure*` (items, teléfonos) no son deterministas entre dos `buildLevel`: no comparar dos construcciones prop a prop. Los colores translúcidos de `fillRect` van PREMULTIPLICADOS (`rgba(hex, a)`), no `color.RGBA{r,g,b,a}` crudo (conos de visión salían como bloques naranjas opacos).
- 2026-11-08 (Cámaras): los elementos de pared que no son props (camera.go) deben comprobar a mano solape con props (rect dibujado), puertas, performers y el inicio, porque overlap_test no los ve; el cono se corta con LOS. Un test que mueve el reloj de un barrido debe congelarlo (secCam.still) o el resultado depende del tiempo.
- 2026-11-08 (Trampas/objetos): las trampas son minas (trapmine.go): carga trapArmTime y quedan armadas sin caducidad hasta que alguien entra en trapDetectR; veneno y disfraz se recogen en `pickupInteract`, ANTES de coopInteract/stealthInteract (el gesto de chocar la mano, levantar al compañero o distraer a un poli se llevaba la tecla) y con el mismo alcance que el cartel (30 px; antes el cartel salía a 30 y el uso exigía 16). Un cartel de acción debe usar el mismo alcance que la acción.
- 2026-11-08 (Puntuacion): todo valor de baja pasa por `killVal` (RETO x3 y cadena) DESPUES de `countKill` (la cadena se calcula ahi); el x5 y los bonus los pone `finishScore` justo antes de `bankPoints`, nunca al perder. `scale` de texto se redondea a entero hacia abajo (1.5 = 1): cuadros con escala 1 o 2. Un estado de pantalla compartido en el audit (g.challenge) debe resetearse en `auditReset`.
- 2026-11-08 (VIP/traje): "nadie te ve" se consigue en `g.sees` y `secCam.sees` (por posición del jugador), no en cada detector; así el traje cubre testigos, guardias, polis, cámaras y sigilo a la vez. Las zonas se buscan sobre el nivel SIN la decoración pequeña y se construyen como una copia de la `LevelDef` (buildLevel -> buildLevelRaw), nunca mutando `levelDefs`; los colocadores `ensure*` no son deterministas, así que el sitio puede variar entre construcciones: el test exige >=5 de 6 zonas. Un hueco de 160x138 no existe en fiestas llenas: la zona se adapta (5/4 segmentos, fondo 64/32). El personal VIP son `Guard` con campos `vip*` y no cuentan en `aliveCount`/`total`.

- 2026-11-08 (Apagon/musica): el evento aleatorio decide en loadLevel con su PROPIO rand (no g.rng) para no mover los dados del resto; toda restriccion de vision pasa por g.sees / secCam.sees, asi basta un factor ahi. El reproductor de mp3 solo admite volumen: la musica reactiva no cambia el tempo (no oido).
- 2026-11-08 (Ayuda/controles): toda acción nueva reutiliza una acción existente (E/X usar, F/Y bailar-señuelo) y debe aparecer en la ventana de ayuda (help.go: helpModes) con las teclas LEÍDAS de `binds`/`binds2`, nunca escritas a mano; Select (CenterLeft) queda reservado para la ayuda y ninguna acción puede usarlo. Una ventana que pausa el juego corta `Update` antes de todo (como trophyOv) y se pinta con un relleno OPACO a pantalla completa (si no, TestUITextAudit ve los textos de debajo). `tr()` traduce palabras sueltas: la etiqueta de tecla `Select` salía como `Elegir`; usar `SELECT`. Los carteles del HUD del sigilo (`E: ponerte la caja`) siguen escritos a mano (pendiente: leer de ajustes).
- 2026-11-08 (Album/trofeos): `Trophy` se escribe en literales posicionales (trophyList, powers.go): no añadirle campos; los atributos nuevos van en un mapa aparte (secretTrophy). Los trofeos secretos no deben bloquear `othersDone` (el Leyenda). Una pestaña nueva sobre el panel de trofeos debe esquivar el toast (x>=352, y 62-86) o el audit marca solape; el panel se comparte con la ventana en partida (trophyhud.go), así que toda tecla nueva se gestiona en `updateTrophyList`.
- 2026-11-08 (Trampas): sin excepciones al explotar (ni dueño ni compañero; coopSpare eliminado): la única regla es de ARMADO (armWait: tras la carga espera a que ningún jugador esté en el radio); todo estado nuevo de prop debe sumarse a las comprobaciones `armT==0 && !armed` (E, HUD, dibujo del dueño).
- 2026-11-08 (HUD): un solo panel (hud.go) con alturas calculadas: lo que cuelga debajo (objeto, aguante) se coloca con `g.hud.bottom`, no con un y fijo; las etiquetas RETO/CADENA son fichas del panel, no textos sueltos. Un test de dibujo no puede leer pixeles (`ReadPixels` falla antes de arrancar el juego): deja que el dibujo anote lo que pinta (`g.lastMarks`) y comprueba eso. El final de la fiesta pasa por `completeLevel`; la camara lenta lo retrasa <=1 s reales y solo en la ultima baja de cada nivel, y los tests la apagan con `slowmoEnabled` (init en slowmo_flag_test.go).
- 2026-11-08 (Fondo animado): los efectos de fondo (ambient.go) son funciones cerradas de g.time + una tabla por nivel (bgFor) construida una vez: nada de estado en Update ni imagenes nuevas por fotograma (reutilizar bgOp y las franjas de swayCache). Cada DrawImage de Ebiten ya hace ~4 allocs en test: el test de allocs compara con un DrawImage desnudo, no con 0. Los tests no pueden leer pixeles sin ventana: el contador bgCount comprueba que dibuja / que ambientOn=false no dibuja. Sonido ambiente: ambientSound APAGADO por defecto (PH_AMBIENT_SOUND=1), audio no oido; un test que lo enciende restaura ambientSound con t.Cleanup.
- 2026-11-08 (Minimapa/hint/emotes): el minimapa se pinta en un buffer de pixeles propio (base de paredes por nivel + puntos por frame) y se sube con UN WritePixels: premultiplicado (un blanco {245,..,235} es invalido) y sin DrawImage por punto; los tests leen el buffer (`g.mm.buf`) en vez de pixeles de pantalla. Teclas nuevas del juego: TAB y Start (mando) estaban libres en ModePlay (R3 es el poder del suceso, M mute). Los emotes se dibujan dentro de `flushBubbles` (ya conocen las cajas elegidas) y se saltan si tocan un bocadillo, un jugador (etiqueta J1/J2, alerta) u otro emote; `rectsHit` es de un _test.go: no usarlo en codigo de produccion. Una fila nueva en la ventana de ayuda obliga a bajar el interlineado (`helpModeRowH`), el pie esta en y=490.
- 2026-11-08 (Huellas): footprints.go sigue a cada actor por puntero en una tabla fija (sin map ni alocaciones) y usa distancia recorrida, no `moving`; el suelo manda por la ultima Floor que contiene el punto (el nivel de esqui tiene snow+white+dance+pave). Un pie de 1x2 px es invisible sobre el suelo nocturno: pie oscuro 2x3 con borde claro, y mirarlo siempre con -shotzoom. PLAYA no esta en levelDefs (solo modo aleatorio) y HOUSE PARTY no tiene piscinas en Level.Pools: comprobar el suelo real antes de meter un nivel en un plan.
- 2026-11-08 (Cache por puntero): una cache `map[*LevelDef]` de una definicion MUTABLE (applyLevelOverride mueve agua/suelos tras un primer buildLevel) devuelve el resultado viejo: validala con una huella del contenido (vipPrint en vip.go); asi TestMovePoolOfOriginal volvio a pasar. "Visto" en el album = cerca (<=150 px) + LOS, nunca "en pantalla" (la camara enseña el nivel entero); los trofeos por conteo de coleccion se calculan del progreso (albumCount), no de un contador `Stats`. Los toasts esperan durante el hint y el periodico (toastHeld). Un nombre que no cabe en una carta se acorta con un campo (cardES), nunca con "."; el test mide el ancho real en ES y EN. Una bandera `-leveldonedemo` sin `-level` se quedaba en la cinematica de camas: las banderas de pantalla necesitan arrancar un nivel (demoLevel).
- 2026-11-08 (Asesino/frecuencia): el asesino especial ya no viene en todas las pantallas: killerreg.go decide en loadLevel con su PROPIO rand (1/3, como el apagon), acelerador a los 3:30 y segundo solo tras 5:00; el cupo (`g.kp.spawned`) es del nivel y lo comparten los dos huecos de 2P, y el hueco viejo `slasherDone/slasherT` ya no manda (killerDue). Un asesino forzado a mano (K) no gasta cupo. Cualquier "viene uno cada X" nuevo debe colgar de `g.time` (tiempo de juego sin pausas), no de relojes de pared. El reloj del HUD usa la misma `g.time` que el bonus de rapidez (umbrales timeFastS/timeMidS), y su parpadeo es un cambio de color a saltos, nunca un fundido (borroso).
- 2026-11-08 (NITIDEZ, obligatoria en cualquier pantalla de texto): el usuario ya pidio dos veces que las letras no esten borrosas. Todo texto va a escala ENTERA y posicion ENTERA, sin escalados fraccionarios ni filtro lineal ni alfa que ensucie el contorno, y con altura de letra >= 7 px reales en 960x540. Toda pantalla de texto nueva trae un TEST DE PIXELES: se pinta en un *image.RGBA propio (el juego no puede leer pixeles de pantalla en tests) y dentro de cada caja de texto cada pixel es EXACTAMENTE el color de tinta o claramente el fondo (nada intermedio, ni sombra ni halo); ver TestPaperTextIsSharpAndWhole y TestPaperCanvasTextHasTwoColoursOnly (newspaper_test.go). El periodico se pinta por software (newspaper_canvas.go, fuente bitmapfont 1-bit) y se muestra como UNA textura 1:1. El test cazo un texto gris sobre el papel con contraste pobre: el color de un texto de lectura es tinta, no un gris.
- 2026-11-08 (Fin de nivel): el periodico es la UNICA pantalla de fin de nivel (se elimino la pagina de resultados y `run.page`); estadisticas de J1 a la izquierda, de J2 a la derecha (o breves si 1P) y cuadro de puntuacion dentro de la portada; "PULSA PARA CONTINUAR" llama a `levelDoneContinue` (afterLevel / stealthExit) sin pantallas intermedias. Una pantalla compuesta por bloques debe teselar el area (test de cobertura >= 90 % y sin solapes) para que no queden huecos.

- 2026-11-08 (Ultimos invitados): un marcador debe verse a tamano real entre la multitud: el anillo tenue (alfa 0,5) y el chevron no se distinguian; ahora doble anillo + flecha grande con contorno oscuro (sin sbShape: el anillo se pinta por tramos, `lastBuf` reutilizado). Lo que protege un hueco de pantalla (flecha) se anota en `g.lastMarks` y los emotes lo consultan en `emoteBlocked`. `tr()` usa `uiLang`, no `gameLang`: un test de textos en EN debe fijar `uiLang` (con t.Cleanup). Con -shotzoom los invitados se mueven al azar entre ejecuciones: recorta la captura entera con `sips`.
- 2026-11-08 (Zonas VIP ocasionales): la zona VIP ya no es fija por nivel: `vipRoll` (vipchance.go) decide en loadLevel con rand propio (`vipDice`, 1/5 + 'pity' a las 8 sin zona en `vipMisses`, solo memoria) y deja el resultado en `vipDecision`, que `buildLevel` lee; cualquier otro buildLevel (tests, generadores, flashbacks) sale SIN zona. En tests que necesiten zona: `vipBuildWith(def, id)` o `vipForce = true` antes de loadLevel (con t.Cleanup); no asumas que un nivel del plan trae zona. El VIP sale entre los no conseguidos del album (`vipGot`: bits ELIMINADO/AYUDADO o `vipEliminated`); `vipPlan` es solo afinidad de tema.
- 2026-11-08 (Nombre del juego): nuestro título es **Party Hardy** (por temas legales no usamos el nombre original como propio). "Party Hard" solo puede aparecer en la atribución de homenaje ("Homenaje a Party Hard (Pinokl Games · tinyBuild)") y en las notas de fans. Binarios/zips: PartyHardy*, módulo Go `partyhardy`. Las partidas se guardan en `assets/` junto al ejecutable, así que el nombre no afecta a los datos del usuario. Tampoco se muestra "Claude"/IA en pantallas del juego ni en LEEME/README.
- 2026-11-08 (Icono): el icono (icon/*.png, icon.ico, AppIcon.icns y rsrc_windows_amd64.syso) se genera con `go run ./tools/mkicon` desde la raiz (sin dependencias; el .syso necesita `rsrc`, ~/go/bin/rsrc). Lleva el texto PARTY HARDY: si cambia el logo/nombre, regenerarlo; build.sh solo copia icon/AppIcon.icns. Juego y editor comparten icono (no hay badge de editor). El `-iconmode` del juego pinta solo el logo (drawLogo) si se prefiere hacerlo desde el juego.

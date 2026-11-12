# Party Hardy: novedades para probar

Este documento es un inventario de todo lo nuevo para que lo pruebes y marques qué te gusta y qué no. Recorre los bloques en orden. Para cada elemento, prueba lo que indica la columna "Cómo probarlo" y marca una opción en la última columna: Sí (lo quiero), No (lo quito) o Cambiar (lo quiero pero diferente). Las banderas que empiezan por "-" se escriben al arrancar el juego (modo DEBUG). Si una bandera no aparece aquí, mira LEEME.txt.

## BLOQUE A · Policía y detección

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| 1 | Policía con memoria: los policías recuerdan al jugador. Solo en DEBUG y apagada por defecto. | -copmemory | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 1 (esposas) | Esposas = "policía avisada, te buscan" (no detenido; en el periódico: "Policía avisada"): solo salen cuando un testigo avisa a un policía. Mientras el testigo no ha avisado, aparece un '!' de alerta sobre el jugador. | Para ver el '!' sin esperar: -alertnow | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 6 | Cámaras de seguridad: giran 90°. Si matas a alguien dentro del cono, te ven. Aparecen en 8 niveles de campaña o aleatorios: Cine, Casino, Estación de tren, Estación espacial, Discoteca, House Party, Centro comercial y Pub. | En cualquiera de esos 8 niveles. Para probar la muerte dentro del cono: -camkill | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 16 | Trampas tipo mina: al colocarlas tienen un círculo de carga de 2,5 s. Después quedan armadas para siempre y explotan cuando se acerca alguien. | -trapdemo | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 17 | Veneno y ropa siempre se pueden coger. | Tecla E, a un alcance de 30 px del objeto | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |

## BLOQUE B · Modo SIGILO (DEBUG)

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| 4/7 | Sigilo para 1 jugador: caja de cartón con tecla E, conos de visión, alerta roja de 12 s y alerta amarilla de 8 s, con panel y temporizador. | -stealth (y la caja: -stealthbox) | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 20 | Sigilo cooperativo para 2 jugadores. Hay un señuelo que se activa con la tecla de bailar (F) o con '/'. | -stealth2p | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |

## BLOQUE C · VIP y objetos

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| 21 | Traje dorado: 8 s indetectable, de un solo uso. | -vipdemo para verlo | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| — | Zonas VIP ocasionales: cada partida tiene 1 posibilidad entre 5 de traer zona VIP (vipchance.go), y si llevas 8 partidas seguidas sin ninguna, la siguiente la trae seguro (contador solo de la sesión). Sale en cualquier nivel donde cabe la alfombra (campaña y aleatorio: unos 27 de 40 al probarlo; no caben en Casa, Miami, Estadio, Discoteca, Autopista, Estación espacial y algunos aleatorios); nunca en sigilo. Cada zona tiene 2 guardias en la puerta y 1 dentro. | Jugar varias partidas; -vipdemo fuerza la zona | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| — | Qué VIP sale: uno que aún no hayas eliminado ni ayudado (según el álbum); el afín al tema del nivel (rey en la plaza de toros...) tiene preferencia. Con los 11 conseguidos sale cualquiera al azar. | Álbum de Trofeos y zonas VIP | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 26 | Invitación VIP robable: la llevan 2 invitados. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 27 | Relevo de guardias VIP cada 40 s. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 30 | Registro de 11 VIP inventados, cada uno con un comportamiento distinto: rey, papa, presidente, padrino nipón, estrella del rock, estrella de cine, magnate, chef, campeón, sultán y astronauta. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |

## BLOQUE D · Puntuación y resultados

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| 23 | Modo RETO: cada baja vale x3 y completar el nivel vale x5. Bonus de nivel: +200 por completar, +150 si no te ven y +300 o +150 por rapidez. | Tecla C en la lista de fiestas, o -challenge | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 29 | Puntos por estilo: cadena de x1, x2... hasta x3, y bonus LIMPIO de +50. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 24 | Estadísticas por método y por jugador. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 36 | Portada de periódico al final del nivel, "EL DIARIO DE LA JUERGA": titular, noticia generada en tono cómico y cuadro de números. | -leveldonedemo 1 o 2 | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |

Nota: "A la cámara" NO está hecho.

## BLOQUE E · Cinemáticas y escenas

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| 2 | Interludio del coche, más corto: 14,8 s. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 2 | Interludio del barco, más corto: 13,8 s, "La fuga y el tiburón". | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 2 | Interludio del globo, más corto: 13,2 s, tras el nivel 15, "PRRRRR". | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 2 | Interludio del cohete, con diseño nuevo: 19,95 s. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 3 | Final malo: preludio en la estación y epílogo de 13 s. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 3 | Final bueno: 62 s con fiesta, conga, fuegos, foto, créditos y post-créditos. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| — | Preludio nuevo: carta en la estación. | -endstage 10 | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| — | Preludio nuevo: isla con T-Rex y confeti. | ver LEEME.txt | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 8 | Cameo de DJs parodia en el final bueno: Gabbo Pontazo (italiano gracioso), David Guette, Martín Garbanzo, Avichi, Daft Pank y Ratón Neón. | Final bueno (ver fila 3) | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| — | Listado de ESCENAS de 9 filas, con formato "N.- NOMBRE (x/21) VISTA / ¿? BLOQUEADA". | Menú de escenas (ver LEEME.txt) | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |

## BLOQUE F · Niveles y mundo

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| — | Miami Beach sin casetas grandes: el DJ se ve. | Nivel Miami Beach | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 5 | Generadores de niveles aleatorios que no tapan al DJ. | Niveles aleatorios | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 43 | Apagón: 1 de cada 16 niveles. | -blackout | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 46 | Música reactiva suave: +10 % de volumen con alerta o policía. No cambia el tempo. | Provocar una alerta o aparecer la policía | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |

## BLOQUE G · Ayuda y mando

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| 48 | Controles de mando revisados. | Mando real conectado (ver LEEME.txt) | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| 49 | Botón '?' con una ventana de ayuda que pausa el juego. | F1, Select o clic en '?'. Para forzarla: -helpov | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |

## BLOQUE H · EN CURSO (agentes trabajando)

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| 44 | Famosos con movimiento propio: moonwalk, Elviz, Rotocop robótico y otros. | ver LEEME.txt | En curso | ☐ Sí ☐ No ☐ Cambiar |
| 45 | Tope de animales de 2 a 8, con 1 toro. | ver LEEME.txt | En curso | ☐ Sí ☐ No ☐ Cambiar |
| 25 | Álbum de tarjetas de VIP y especiales (incluye Diego, César y Sam-sam), dentro del menú Trofeos. | Menú Trofeos | En curso | ☐ Sí ☐ No ☐ Cambiar |
| 47 | Trofeos secretos. | Menú Trofeos | En curso | ☐ Sí ☐ No ☐ Cambiar |
| — | Objetivos con pestaña POLICÍAS (patrulla, FBI, ejército, SWAT, Rangers): se marcan al matar un agente de cada tipo o al llegar a su hito. | Menú Objetivos > POLICÍAS | En curso | ☐ Sí ☐ No ☐ Cambiar |
| — | Uniformes de policía por ESCAPES: cada 3 esposas evitadas se abre un conjunto nuevo (36 en total, los 12 últimos cada 4), con aviso. Ya no se compran; los comprados antes se conservan. | Escapar de la policía; contador en Objetivos > POLICÍAS | En curso | ☐ Sí ☐ No ☐ Cambiar |
| — | Vestuario con 3 páginas: PERSONAJE, ASESINOS (color de ropa y arma) y POLICÍAS (uniforme de cada cuerpo); se aplica en la fiesta. | Tienda > VESTUARIO | En curso | ☐ Sí ☐ No ☐ Cambiar |
| — | Tienda reducida: 41 prendas, 60 accesorios, 12 mejoras, 8 animales y 20 armas (141 artículos, ~125.000 puntos). Lo retirado que ya tenías sigue siendo tuyo. | Menú Tienda | En curso | ☐ Sí ☐ No ☐ Cambiar |
| 51 | HUD superior derecho rediseñado, con barra de progreso. | Pantalla de juego | En curso | ☐ Sí ☐ No ☐ Cambiar |
| 52 | Flecha que señala a los últimos invitados (3 o menos). | Pantalla de juego | En curso | ☐ Sí ☐ No ☐ Cambiar |
| 54 | Cámara lenta en la última baja. | Matar al último invitado | En curso | ☐ Sí ☐ No ☐ Cambiar |
| 55 | Animaciones de fondo sutiles: olas y sol en Miami Beach, delfines y peces en el barco, árboles al viento, nieve, niebla. | Nivel correspondiente | En curso | ☐ Sí ☐ No ☐ Cambiar |
| 53 | Sonido ambiente: apagado por defecto, se activa con un interruptor. El dueño decide si se queda. | Interruptor de sonido ambiente (ver LEEME.txt) | En curso | ☐ Sí ☐ No ☐ Cambiar |

## BLOQUE I · PENDIENTES ACEPTADOS

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| 56 | Mensaje de nivel al empezar. Probar y quedarse si funciona. | Inicio de nivel | Pendiente | ☐ Sí ☐ No ☐ Cambiar |
| 57 | Minimapa semitransparente estilo de los juegos de fútbol clásicos: paredes en blanco y personajes de 1 px de color según el tipo. | Pantalla de juego | Pendiente | ☐ Sí ☐ No ☐ Cambiar |
| 58 | Trampas iguales para todos: cualquier jugador que pase explota, sin excepciones. | Pasar por una trampa con cualquier personaje | Pendiente | ☐ Sí ☐ No ☐ Cambiar |
| 59 | Huellas en arena, nieve y charcos. | Caminar por arena, nieve o charcos | Pendiente | ☐ Sí ☐ No ☐ Cambiar |
| 60 | Emotes de invitados sobre la cabeza. | Pantalla de juego | Pendiente | ☐ Sí ☐ No ☐ Cambiar |

## BLOQUE J · DESCARTADO por el dueño (no hacer)

Estas ideas están descartadas. No se prueban ni se implementan.

| Nº | Qué es | Cómo probarlo (bandera / dónde) | Estado | ¿Me gusta? |
|---|---|---|---|---|
| 9 | Cajas de otros disfraces. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 10 | Álbum de finales como menú aparte. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 11 | Distracciones lata y zapato. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 12 | Rango S/A/B. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 13 | Cámaras hackeables. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 14 | Fotos de fiesta. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 15 | Más gags en el final. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 18 | Combos de trampas. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 19 | Radar de minas. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 22 | Cámara rota por trampa. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 28 | Modificadores de reto. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 31-35 | Ideas descartadas (sin detalle en el inventario). | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 37, 39-42 | Ideas descartadas (sin detalle en el inventario). | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| 50 | Tutorial. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| — | Cambio de tempo musical. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| — | Depurador de audio. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |
| — | Prueba de humo previa al empaquetado. | No aplica | Descartado | ☐ Sí ☐ No ☐ Cambiar |

## Qué NO puede verificar nadie sin ti

Estas cosas solo se pueden comprobar jugando en tu equipo, con tu mando y tu ratón:

- El audio de todo lo nuevo (sonido ambiente, música reactiva, cameos de DJs).
- El movimiento de interludios, finales y famosos.
- El mando y el ratón reales.
- El modo de 2 jugadores real.
- La caja del sigilo.
- El cono rojo de las cámaras.

## Banderas de depuración

| Bandera | Qué activa |
|---|---|
| -copmemory | Policía con memoria (solo DEBUG, apagada) |
| -alertnow | Muestra la alerta '!' sobre el jugador |
| -camkill | Muerte dentro del cono de cámara |
| -trapdemo | Trampas tipo mina |
| -stealth | Modo sigilo 1 jugador |
| -stealthbox | Caja de cartón del sigilo |
| -stealth2p | Sigilo cooperativo 2 jugadores |
| -vipdemo | Traje dorado |
| -challenge | Modo RETO |
| -leveldonedemo 1 (o 2) | Portada "EL DIARIO DE LA JUERGA" (unica pantalla de fin de nivel) |
| -endstage 10 | Preludio de la carta en la estación |
| -blackout | Apagón |
| -helpov | Ventana de ayuda |

Para el resto de banderas, mira LEEME.txt.

## BLOQUE FINAL · Cambio de nombre

| Nº | Qué es | Cómo probarlo | Estado | ¿Me gusta? |
|---|---|---|---|---|
| R1 | El juego se llama ahora **Party Hardy** (título de ventana, logo PARTY HARDY del menú, cartel de neón del nivel 1, créditos, editor, binarios PartyHardy.exe / PartyHardy.app / PartyHardyEditor.*, PartyHardy-Mac.zip y PartyHardy-Windows.zip). Se mantiene el "Homenaje a Party Hard (Pinokl Games · tinyBuild)" como atribución al original. | Abrir el juego y mirar el título y los créditos | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |
| R2 | Quitado "con Claude" de la intro y la fila "IA / Claude" de los créditos (ajustes y final). | Intro del juego y Ajustes > Créditos | Hecho, sin probar por el dueño | ☐ Sí ☐ No ☐ Cambiar |

# Lecciones aprendidas (Party Hard)

Lista corta que TODO agente (implementación, QA, build, limpieza, orquestador) lee al EMPEZAR y respeta siempre.
Cada regla lleva entre paréntesis qué pasó. QA la usa como checklist: marca cada casilla que aplique a la entrega.

**Cómo añadir una lección nueva:** si descubres un fallo nuevo (o el usuario te corrige), añade UNA línea en su sección con el formato
`- [ ] Regla clara (qué pasó, AAAA-MM-DD)`. Busca antes con grep para no duplicar; si ya existe, no la repitas. Sin nada personal.
Historial de pruebas y veredictos: `docs/QA/MEMORIA.md` (si existe; no se duplica aquí, y ese archivo debe enlazar de vuelta a este).

## Visual y texto
- [ ] Texto nítido: escalas y posiciones enteras, sin suavizado (texto borroso en pantalla).
- [ ] Ningún texto se pisa con otro, se sale de su panel ni tapa el logo (textos solapados en menús).
- [ ] Textos largos en 2 líneas máx.; frase de bocadillo <= 48 caracteres (bocadillos desbordados).
- [ ] Toda cadena nueva en ES y EN: `tr()` + `lang_data.go`, con gracia (textos solo en un idioma).
- [ ] Sin puntitos blancos de fondo en cinemáticas (ruido que parecía estrellas).
- [ ] Sin personajes transparentes, X rojas ni triángulos sueltos (artefactos de render).
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
- [ ] Máx. 4 bocadillos a la vez; sin frases repetidas a la vez ni mensajes de info repetidos en 20 s (spam de textos).
- [ ] Asesinos y T-Rex van a la persona más cercana que ven de frente, sin prioridad del jugador (perseguían siempre al jugador).
- [ ] Las esposas van al último visto matando; 2P tiene reglas propias (esposas al sospechoso equivocado).
- [ ] Tipos de policía se desbloquean por asesinatos (desbloqueo mal ligado).
- [ ] Un jugador no mata al otro, solo golpea (fuego amistoso en 2P).

## Hacer lo pedido y nada más
- [ ] "Lo mismo que ya teníamos" = reutilizar el activo existente; no inventar coches, personajes ni estilos (coche explosivo rediseñado sin pedirlo).
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

## Proceso
- [ ] Lanzar el juego solo con watchdog: macOS no tiene `timeout`; usar `&` y `kill` a ~40 s (juego colgado).
- [ ] No dejar procesos ni ventanas abiertas; borrar temporales (restos tras las pruebas).
- [ ] Una sola pasada completa de `go test -count=1` y build por entrega, en BUILD (pasadas repetidas que gastan tiempo y disco).
- [ ] Ediciones quirúrgicas en archivos compartidos y releer justo antes de editar (pisadas entre agentes).
- [ ] No tocar archivos de agentes concurrentes (conflictos).
- [ ] Informe de máx. 15 líneas: QUÉ HICE / EVIDENCIAS (comandos y resultados reales) / NO VERIFICADO / AVISOS (informes inflados sin pruebas).
- [ ] QA da veredicto PASA / PASA CON RESERVAS / NO PASA, tras ejecutar él mismo tests y capturas.

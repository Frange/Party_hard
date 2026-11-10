---
name: party-hard-qa
description: Rol de QA en el proyecto Party Hard. Úsalo si eres el agente que revisa las pruebas y evidencias de otros agentes y da el veredicto (PASA / PASA CON RESERVAS / NO PASA). QA ejecuta tests y capturas él mismo y contrasta los informes con el árbol; no implementa.
---

# QA de Party Hard

Lee primero `docs/EQUIPO/LECCIONES.md` (reglas y checklist de lo que el usuario ya corrigió).

## GIT: NADIE LO TOCA
GIT: NADIE LO TOCA. Solo el usuario hace commits, push y cambios del índice. Ningún agente (implementación, QA, build, limpieza, orquestador) ejecuta git salvo LECTURAS (status, diff, log, ls-files, check-ignore, stash list). PROHIBIDO: commit, push, add, rm (también `git rm --cached`), stash (y stash pop), reset, checkout/restore, clean, rebase, merge, branch, tag, y editar .gitignore sin que el usuario lo pida. Si algo requiere git (p. ej. dejar de seguir un archivo, borrar binarios antiguos del repo), díselo al usuario con el comando exacto y que lo ejecute él. Motivo: un agente hizo `git stash` por error y tocó el índice; el usuario dijo "el GIT no tiene que tocarlo nadie". Todo brief a un agente debe incluir esta regla.

Tu trabajo: **decidir con pruebas si lo que dicen los agentes es verdad.** No implementas: si algo falla, lo devuelves al responsable (a través del orquestador).

## Antes de empezar
- Comprueba que **no hay agentes de implementación editando** (si los hay, los errores de compilación pueden ser transitorios: espera y repite una vez). Dilo en el veredicto.
- Recibes: la petición original del usuario, los informes de cada agente y la lista de archivos tocados.

## Texto y maquetación (siempre que cambie algo visual, de textos o de traducciones)
- Lee `docs/QA/MEMORIA.md` (criterios fijos, fallos recurrentes) y `checklist.md`.
- Ejecuta SIEMPRE `go test -count=1 -run TestUITextAudit .` (ES y EN; `-short` = solo inglés, más rápido). Si falla, es NO PASA.
- Haz capturas ampliadas con `-shotzoom x,y,w,h,2` de las pantallas tocadas, léelas y comprueba nitidez, solapes y textos tapados o en español en la versión inglesa.
- **Márgenes** (regla del usuario "SIEMPRE MARGEN", ver `party-hard-dev` 9b): revisa las capturas de cada pantalla tocada (ES y EN, 1 y 2 jugadores) buscando cajas, títulos, textos o botones pegados entre sí o al borde, y exige un test de geometría que lo compruebe. Sin él, no hay PASA.
- **No des PASA** si la auditoría falla o hay texto borroso, solapado o cortado. Una excepción nueva en la auditoría exige razón escrita.
- Anota hallazgos nuevos con fecha en `docs/QA/MEMORIA.md`.

## Qué haces, tarea por tarea
1. **Contrasta el informe con el árbol**: `git status`, `git diff --stat`, abre los archivos que dice haber creado, comprueba que las funciones y tests nombrados existen. Un informe que cita algo que no existe es un fallo.
2. **Ejecuta tú mismo** (herramienta `qa_run_tests` del MCP o a mano `go vet .` y `go test -count=1 -json .`): totales, FALLOS, **SKIPPED** (un test saltado no prueba nada: justifica cada uno), tests lentos. Si el filtro no ejecuta ningún test, no vale.
3. **Los tests ¿prueban lo que dicen?** Lee los nuevos: no vacíos, con aserciones reales sobre el comportamiento pedido, sin condiciones siempre verdaderas, sin `t.Skip` que esquiven un fallo, sin umbrales aflojados. Compara con `git diff` de los `_test.go` existentes: si se debilitó o borró un test, es un hallazgo (a menos que se justifique).
4. **Regresión con 1 jugador**: todo lo existente debe seguir igual sin `-twop`; los tests antiguos pasan; capturas sin banderas nuevas no cambian.
5. **Capturas**: lanza las relevantes con vigilante (ver `party-hard-dev` sección 7), **léelas** y comprueba lo pedido (no que el archivo exista). Recuerda matar lo que lances.
6. **Reglas del usuario**: `assets/progress.json` sin tocar (fecha de modificación), nada nuevo en `imports/`, ninguna música o dato personal añadido al repositorio, textos con traducción (`go test -run Lang`), frases de bocadillo ≤ 48 caracteres, famosos únicos por nivel, nadie atascado (>5 s), nada solapado.
7. **Versión**: si la tanda trae novedades, comprueba que `gameVersion` subió según el tipo (menor x.y.Z / medio x.Y.0 / grande X.0.0) y que `CHANGELOG.md` tiene su entrada arriba, verificada contra el árbol (`go test -count=1 -run Version .`). **VERSIÓN Y CHANGELOG (regla del usuario, 2026-11-10)**: cada tanda con novedades sube la versión semver en `version.go` (`gameVersion`; menor = x.y.Z, medio = x.Y.0, grande = X.0.0) y añade su entrada arriba en `CHANGELOG.md` (`## [vX.Y.Z] - AAAA-MM-DD - menor|medio|grande`, listas Añadido/Cambiado/Arreglado, en español, sin nombres de personas reales salvo músicos y futbolistas) ANTES del build; BUILD verifica que la constante, la cabecera del CHANGELOG y el `VERSION.txt` de los dos zips coinciden y lo reporta (`build.sh` se niega a compilar si constante y CHANGELOG no coinciden; `TestVersionMatchesChangelog`); los tags y commits son del usuario. Sin eso, no hay PASA.
8. **Lo no verificable sin humano**: lista aparte (audio, mando, teclado real, dos personas, el ojo para "queda bonito").

## Veredicto (uno por tarea, en este formato)
```
TAREA: nombre
VEREDICTO: PASA | PASA CON RESERVAS | NO PASA
EVIDENCIAS: comandos y resultados reales; tests por nombre; capturas leídas
RESERVAS: lo que preocupa pero no bloquea
FALLOS: lo que obliga a devolver la tarea (qué, dónde, cómo reproducirlo)
NO VERIFICABLE SIN HUMANO: lista
```
- **PASA**: todo comprobado y sin fallos.
- **PASA CON RESERVAS**: funciona, pero hay algo débil, no probado o discutible. Descríbelo para que el usuario decida.
- **NO PASA**: un test falla, algo pedido no existe, se rompió el modo de 1 jugador, o el informe no se corresponde con la realidad.

Al final, un resumen global (una línea por tarea) y si **se puede lanzar Build**: sólo con PASA o reservas aceptadas.
- **Pasada completa antes de PASA**: si el cambio toca `game.go`, estado global o más de un área, ejecuta `go test -count=1 -timeout 20m .` entera; los filtros `-run` no detectan contaminación entre tests (2026-11-04).
- Regla de datos (2026-11-04): toda bandera de depuración/captura congela el progreso antes de cargar niveles (debugFlagsActive en main.go); las capturas se hacen desde una carpeta temporal con COPIA de assets, nunca desde el proyecto.

- Las capturas/ejecuciones con ventana solo se lanzan cuando el orquestador/usuario lo autoriza, en UNA tanda anunciada; el usuario puede estar jugando; preferir tests sin ventana (origen: QA abrió ventanas continuamente mientras el usuario jugaba, 2026-11-04). Usa el modo silencioso (automático con banderas de depuración o PH_QUIET=1: ventana fuera de pantalla, sin foco, prioridad baja) y -visible para lo contrario. Pasadas pesadas: `nice -n 19 go test -count=1 -p 2 -timeout 20m .`, `GOMAXPROCS=4`, `nice -n 19 ./build.sh`, o `tools/ph_quiet.sh <cmd>` (tardan más pero no bloquean el PC).

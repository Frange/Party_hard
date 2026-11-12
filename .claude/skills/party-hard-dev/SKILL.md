---
name: party-hard-dev
description: Conocimiento y reglas del proyecto Party Hard (clon en Go/Ebiten con pixel art procedural). Léelo si eres un agente de IMPLEMENTACIÓN que va a tocar este repositorio: añadir personajes famosos, objetos (props), niveles o generadores, traducciones, escenas, 2 jugadores, depurar con capturas y banderas, trabajar con otros agentes a la vez. Los roles de orquestador, QA, build y limpieza tienen sus propios skills (party-hard-orquestador, party-hard-qa, party-hard-release, party-hard-limpieza).
---

# Party Hard: cómo se trabaja en este proyecto

Lee primero `docs/EQUIPO/LECCIONES.md` (reglas y checklist de lo que el usuario ya corrigió).

## GIT: NADIE LO TOCA
GIT: NADIE LO TOCA. Solo el usuario hace commits, push y cambios del índice. Ningún agente (implementación, QA, build, limpieza, orquestador) ejecuta git salvo LECTURAS (status, diff, log, ls-files, check-ignore, stash list). PROHIBIDO: commit, push, add, rm (también `git rm --cached`), stash (y stash pop), reset, checkout/restore, clean, rebase, merge, branch, tag, y editar .gitignore sin que el usuario lo pida. Si algo requiere git (p. ej. dejar de seguir un archivo, borrar binarios antiguos del repo), díselo al usuario con el comando exacto y que lo ejecute él. Motivo: un agente hizo `git stash` por error y tocó el índice; el usuario dijo "el GIT no tiene que tocarlo nadie". Todo brief a un agente debe incluir esta regla.

Léelo entero antes de tocar código. Si algo de aquí contradice lo que el usuario pide en el momento, manda el usuario.

## 0. Quién hace qué (modelo de trabajo actual)

El asistente principal es **ORQUESTADOR**: habla con el usuario, reparte el trabajo y exige datos, pero **no edita, no compila, no lanza tests ni capturas y no borra**. Todo lo hacen agentes con un rol:

| Rol | Qué hace | Skill |
|---|---|---|
| Orquestador | habla con el usuario, lanza agentes, recoge informes, exige evidencias | `party-hard-orquestador` |
| **Implementación** (tú, si lees esto para programar) | un área, con archivos propios | este skill |
| QA | revisa pruebas y evidencias, da un veredicto | `party-hard-qa` |
| Build / Release | gofmt, vet, tests, `build.sh`, binarios de la raíz, zips | `party-hard-release` |
| Limpieza | procesos colgados, temporales, caché de Go, disco | `party-hard-limpieza` |

Orden de una entrega: implementación -> QA -> Build -> Limpieza -> informe al usuario.

### Si eres un agente de IMPLEMENTACIÓN

1. Lee tu encargo: objetivo, contexto, **archivos propios**, reglas y **criterio de hecho**. Si algo falta, supónlo con criterio y dilo en el informe (no puedes preguntar al usuario).
2. Trabaja sólo en tus archivos; en los compartidos (`game.go`, `draw.go`, `lang_data.go`...) usa Edit quirúrgico releyendo antes (sección 8).
3. Escribe **tests que prueben de verdad** lo pedido (no vacíos, sin `t.Skip` para esquivar fallos, sin debilitar tests existentes) y haz capturas con las banderas de la sección 6 (con vigilante, sección 7). Léelas.
4. Compruébalo tú: `gofmt -l`, `go vet .`, tus tests y los relacionados. Un error de compilación de otro agente concurrente es transitorio: espera y repite.
5. **Informe obligatorio** (si falta algo, el orquestador te lo devolverá):
   - QUÉ HICE (en pocas líneas, con nombres de archivos).
   - EVIDENCIAS: comandos ejecutados y su resultado real (copia la línea `ok  partyhard 12.3s` o el fallo), tests nuevos por nombre, rutas de las capturas y qué se ve en ellas.
   - NO VERIFICADO: lo que no pudiste comprobar (audio, ratón, mando, dos personas...).
   - AVISOS: cosas fuera de tu encargo que viste (sin tocarlas).
6. No hagas commit, no ejecutes `build.sh`, no toques `assets/progress.json`, no dejes procesos abiertos ni binarios en `/tmp`.

## 1. Estructura (lo importante)

- **Un solo `package main`** en la raíz (Go + Ebiten). Todos los `.go` están en la raíz; no hay subpaquetes del juego.
- **Todo el arte es procedural**: los personajes, objetos, animales y fondos se dibujan por código (`pixBuf`, `genChar`, `Draw: func(b *pixBuf, v int)` de cada prop). No hay hojas de sprites hechas a mano.
- **`assets/`** guarda lo que el editor exporta o edita (`assets/sprites/<categoría>/<nombre>/`, `assets/backgrounds/`). Se sube al repositorio.
- **`imports/`** (junto a `assets/`) guarda lo que el usuario crea importando fotos o dibujos. **Nunca se sube al repositorio** (está en `.gitignore`). El juego y el editor leen las dos carpetas.
- **Editor** (pestañas Personajes, Objetos y animales, Escenarios) dentro del mismo binario: tecla F2, o `PartyHardyEditor`. **Modo debug**: en el menú principal arriba, arriba, abajo, abajo, izquierda, derecha, izquierda, derecha; en el editor la misma secuencia desbloquea editar los niveles originales.
- **Dos jugadores** (interruptor en el menú principal, por defecto 1): `twoplayers.go` (segundo jugador `g.p2`, `g.players()`, `g.asPlayer`, `g.focus`), `input2.go` + `players.go` (teclas y mandos por jugador), `twojail.go` + `suspect.go` (esposas, detención y fuga), `revive.go` (ambulancia), `coop.go` (gags y marcador), `wardrobe.go` (ropa por jugador), `twoscenes.go` y `*2p.go` (cinemáticas). Todo cambio en el juego debe seguir idéntico con 1 jugador (hay tests de regresión).
- Los niveles son 26 en la campaña (`campaign.go`; ITALIA es la fiesta 20, índice 19, tras la AUTOPISTA; el BARCO PIRATA es la 25, índice 24 (`pirateIdx`), justo antes de la estación; la Estación Espacial es la 26 y la Isla Nublada (Parque Fiestásico) es secreta) más extras para el modo aleatorio. REGLA: las dos últimas pantallas son SIEMPRE la Estación Espacial (último de la lista) y después la secreta Isla Nublada (Parque Fiestásico; en código `jurassic`) (fuera de la lista); un nivel nuevo se inserta ANTES de `station` y trae su migración de partida (`progress_migrate.go`: sube `campaignRev` y mueve `won_N` y `Unlocked` del índice de inserción en adelante; rev 1 = 3 fiestas nuevas, rev 2 = ITALIA, rev 3 = BARCO PIRATA, que mueve solo la estación won_24 -> won_25 y abre el barco a quien tenía la estación abierta) y sus claves viejas del editor (`campaignKeyRev1`/`campaignKeyRev2` en campaign.go). Toca también: la lista de fiestas (3 columnas x `levelRowsPerCol` = 10 filas, draw.go; `TestPartyListHasRoomForEverything`, `TestLevelSelectSpacing`), el cohete (sale tras la fiesta anterior a la estación) y los tests que cuentan fiestas (venues_test, italy_test, techs_test, killtarget_test, interlude_*_test, uitext_audit_test). Ejemplo completo de nivel con mar: `pirate.go` (+ `_props`, `_text`, `_icon`, `_test`): Shape + voidWalls (el mar es sólido), Transit "sea" (todo llega en lancha al muelle), suelo pintado una vez por código (floors "bay"/"galleon"). La isla y las escenas (intro, interludios, finales) quedan fuera de la campaña.

## 2. Añadir un personaje famoso

1. Mira un ejemplo completo: `monkey.go` + `monkey_lines.go`, `skelcast.go`, `kids.go`, o las listas `famousPresets` / `starPresets` (`presets*.go`, `cast2.go`). Cada personaje es un `def("slug", ...)` con un `CharSpec` procedural (pelo, ropa, sombrero, objeto en la mano...).
2. Nombre visible en `stars.go` / `funnynames.go`; descripción y precio de tienda en `shopdata.go` y `shopdesc.go`.
3. **Frases**: tablas de muletillas en `dialogue.go`, `jokes.go`, `greetings.go`, `duets.go` (o archivo propio como `monkey_lines.go`). **Máximo 48 caracteres por frase**, graciosas, cortas. Sin diálogos largos copiados de otros juegos: muletillas muy cortas o frases propias.
4. **Traducción inglesa** de cada texto nuevo en `lang_data.go` (ver sección 5).
5. **Un famoso solo puede estar una vez por nivel** (hay una función central; no la saltes). Tests: `famous_test.go`, `TestAllBuiltinsExported` (el editor exporta todos los personajes), `lang_test.go`.
6. Si el personaje es cuadrilla (varios a la vez) deben ser personajes distintos entre sí.
7. **Nombres con UNA letra cambiada** (dueño, 2026-11-11; sustituye la política de "músicos y futbolistas con nombre real"): todo nombre visible de una **persona real** (músicos, futbolistas, figuras históricas...) o de un **personaje/marca de una obra existente** lleva una letra cambiada, añadida o quitada, reconocible pero nunca exacto ("Clara Croft", "Buttman", "Zonic", "Karnackles", "Rotocop", "Lionel Mesi", "James Hetfiel", "Flan Guys"). Los **nombres propios NUNCA se traducen**: el mismo en ES y EN (las palabras comunes alrededor, "Cazafantasmas", "· cantante", sí). Desde v1.1.2 las bandas también ("Simple Pan", "Sum 42", `bandList` en bands.go; los mp3 de bands/ se buscan por slug o nombre nuevo, `bandFileKey`), y también marcas, juegos, apps, naves y lugares de obras; el cantante de Lost Prophets sigue sin nombre ("banda · rol", nunca un miembro con condena penal grave notoria). Dónde: el nombre se pone donde se define el personaje (stars.go, cast2.go, presets4.go, monkey_lines.go, `bandRealNames` en bands.go, `footballRealNames` en futbol_art.go, `beanNames` en fallguys.go) o en `ipNames` (ipnames.go) si antes salía del slug; el `display_name` de su meta.json igual; si es nombre propio, su slug va en `properNameSlugs` (ipnames.go: `keepProperNames` impide que tr() lo traduzca). Frases, descripciones, noticias y docs públicos usan el nombre nuevo. **Los slugs, carpetas y claves de progreso NO cambian** (partidas guardadas). Tests: `TestNoOriginalIPNamesVisible` (lista de originales como hash: nombres, literales del código, JSON de assets, LEEME/README/NOVEDADES), `TestProperNamesNotTranslated`, `TestBandMusiciansRealNames`, `TestFootballersCardsAndDetail`.
8. **Famoso nuevo, pasos fijos** (ejemplo completo: futbol_art.go + futbol_art_test.go): preset con `rand` PROPIO (`def(...).F(rand.New(rand.NewSource(N)))`, colores de piel/zapatos fijados) para no mover el rand compartido de `builtinChars` (test `TestFootballersKeepOthersSpecs`); `famousPresets` + `specialPool` (tarjeta en OBJETIVOS/PERSONAJES y vestuario); nombre visible según el punto 7 (una letra cambiada, igual en ES y EN); `charDescs` ES/EN, `funnyDesc`, `catchphrases` traducidas; capas propias enganchadas en `genChar` (chars.go); exportar los PNG a `assets/sprites/special_people/<slug>/` (test `*Export` con variable de entorno, método de `TestKtmExport`) y comprobar `TestCharimpReexportedPictures` / `CHARIMP_SCAN=assets go test -run TestCharimpStaleScan`; subir los totales de `TestCharimpICountsUnchanged`. Las claves del álbum son slugs: no hace falta migración si no cambian slugs ni orden de campaña.

## 3. Añadir un objeto (prop)

- Los props viven en `props.go` … `props9.go`: `pk["nombre"] = &PropKind{W, H, FW, FH, Vars, Use, Draw: ...}`. `FW/FH` es la huella de colisión; `Use` son los puntos donde la gente lo usa.
- **Nunca pueden pisarse dos objetos**, ni objetos sobre vallas, carreteras, agua o fuera del terreno. `overlap.go` + `overlap_test.go` lo comprueban en todos los niveles: añade el prop nuevo a sus reglas si es grande o decorativo.
- Los niveles se colocan en `levels.go` (`P("nombre", x, y)`) o en los generadores `gen*.go`.

## 4. Añadir un nivel o generador

- Un generador devuelve un `LevelDef` (suelos, muros, props, zonas por donde se camina `Roam`, bailes, guardias, policía, puntos de inicio). Se registra en `campaign.go` (`kind`, `seed`, `title`).
- Tests que debes pasar: `collision_test.go` (alcanzabilidad), `overlap_test.go`, `famous_test.go`. Usa `-walkbot` para comprobar que nadie se atasca.
- El nivel debe permitir de 45 a 60 invitados sin atascos, y con 2 jugadores de 20 a 30 más (`cap.go` es el único punto que controla el tope; los ajustes de invitados están solo en DEBUG).
- **Tope de bajas (BAJAS x/N)** (`cap.go`, dueño 2026-11-10): N = todos los que tuvo la fiesta (inicio + llegadas). `goalMax()` = **65** con 1 jugador (+20..30 con 2); el inicio deja `startRoom()` plazas libres y toda ruta que añade gente (taxis, bus, tren, Dios, famosos, pandillas, niños...) pregunta `guestRoom()`; 2 plazas reservadas para asesino/pirómano. Tests: `killtarget_test.go`.
- **Fiestas de aforo grande** (mapa `bigCrowd` en `cap.go`, por `Def.Icon`): **PARQUE DE ATRACCIONES (L8, funfair), FESTIVAL (L11) y ESTACIÓN DE ESQUÍ (L16, snow)**, elegidas por suelo libre transitable (`freeRoamCells`, top 5 de 25) y pocos estorbos: +30 (`crowdMax`) → tope de bajas **95**, aforo vivo 90, inicio ~78-83 (`startRoom` = 6 + extra/5). Con 2 jugadores el total combinado no pasa de **110** (`crowdTwoTotal`: el extra grande se recorta). El **RETO DE HOY** y el **sigilo** las juegan con el 65 normal; la isla y las tiradas del modo aleatorio nunca son grandes. El resto de fiestas sigue en 65. Para añadir/quitar una: solo el mapa `bigCrowd` (nunca más de +30 sin medir rendimiento) y pasar `bigcrowd_test.go` (elección por espacio, N/N, Reto/sigilo, tramos de policía con T=95/110, HUD de 3 cifras, coste por fotograma).

## 5. Traducir

- Todo texto visible pasa por `tr()` (se aplica solo en `drawText`, `textW`...). Escribe los textos en español.
- Cada texto español nuevo necesita su entrada en `lang_data.go` (`es -> en`), graciosa también en inglés. Si cambias un texto español, cambia la **clave** en `lang_data.go`: no dejes claves huérfanas.
- Comprueba con `go test -count=1 -run Lang .`.

## 6. Banderas de depuración (todas con `./PartyHard ...` o `go run . ...`)

**Niveles y capturas:** `-level N` (1 a 26: 25 = BARCO PIRATA, 26 = ESTACIÓN ESPACIAL) · `-shot ruta.png` + `-shottick T` (captura en el fotograma T y sale) · `-shotzoom x,y,w,h,zoom` · `-with slug,slug` (fuerza personajes) · `-fastevents` · `-immune` · `-walkbot` (anda al azar y cuenta atascos) · `-gamebench N` (ms por fotograma de Update y Draw) · `-jurassic` · `-jpunlocked` · `-levelselect` · `-event N` · `-lang en|es` · `-solids` · `-testroad` (+ `-roadcars a,b`) · `-vlane`.
**Cinemáticas:** `-title` / `-notitle` · `-firstrun` (simula la primera vez, nada se guarda) · `-intro` · `-ending 1|2|3` (+ `-endstage N`) · `-interlude 5|10|20` · `-scenes 1|2` · `-inviteauto N`.
**Sucesos y objetos:** `-zapnow` · `-poisonnow` · `-pyronow` · `-snakenow` · `-cheernow` · `-kidsnow` · `-bowlkids` · `-copsnow` · `-blamenow` (sospechoso equivocado, wrongsuspect.go) · `-ring 1|2` · `-ambient vehículo`.
**Dos jugadores:** `-twop` (todas las fiestas a 2) · `-players 1|2` (el interruptor del menú, sin guardar) · `-twocuff` · `-twojail 1|2` · `-twodead 1|2` · `-coopdemo 1|2|3` · `-wardrobe 1|2|3` · `-setsel N` (fila de Ajustes).
**Pantallas:** `-settings` · `-shopscreen` · `-trophyscreen` · `-trophyov` · `-credits` · `-debugscreen` · `-alltrophies`.
**Editor:** `-editor` (+ `-edtab N`, `-edload nombre`, `-edorig N`, `-edfind texto`, `-edbench N`, `-eddemo pool|helipad|zone`, `-edobjtool N`, `-edpttool N`).
Con `-shot` y estas banderas el progreso queda congelado: no se escribe nada. Para ver todas: `./PartyHard -h`.

## 7. Capturas sin dejar ventanas abiertas

En macOS **no existe `timeout`**. Lanza siempre con un vigilante y mata el proceso:

```bash
/tmp/ph_chk -level 5 -shottick 300 -shot /tmp/ph_out/a.png >/tmp/ph_out/log.txt 2>&1 &
pid=$!; for i in $(seq 1 60); do sleep 1; kill -0 $pid 2>/dev/null || break; done; kill $pid 2>/dev/null
```

Compila a una ruta temporal (`go build -o /tmp/ph_chk .`) y bórrala al terminar. Si lanzas varias, comprueba después con `ps -eo pid,command | grep /tmp/ph_` que no queda ninguna. Después de una captura, léela con la herramienta de lectura de imágenes y comprueba lo que pedía el usuario, no solo que el archivo existe.

## 8. Varios agentes a la vez

- Reparte por **archivos distintos**. Cada agente dice qué archivos son suyos.
- En archivos compartidos (`game.go`, `draw.go`, `lang_data.go`) usa **Edit quirúrgico**: relee el trozo justo antes de editar, cambia lo mínimo, no reescribas el archivo entero.
- Los errores de compilación que vienen del trabajo de otro agente son transitorios: espera unos segundos y reintenta; no los arregles tú.
- No repartas tareas que tocan la misma función ni que dependen unas de otras (p. ej. tamaño de personajes y colisiones). Hazlas en orden.

## 9. Normas del usuario (importantes)

- **No hagas commit ni push** salvo que el usuario lo pida expresamente.
- **`assets/progress.json` es la partida real del usuario**: ningún test, captura o agente lo modifica nunca (usa progreso congelado, `t.TempDir()` o copias).
- **La música (mp3, ogg...) no se sube** al repositorio. **`imports/` tampoco.** El repositorio de GitHub es **público**: nada personal (rutas locales, correos, nombres reales, fotos). Compila con `-trimpath` (ya está en `build.sh`).
- Los **zips** salen sin `progress.json` ni `editor_prefs.json` (lo hace `build.sh`).
- **Parches**: `releases/manifest-*.json` (los escribe `build.sh`, no se editan a mano) y `./mkpatch.sh <base>` hacen el zip con sólo los cambios entre versiones (`tools/mkpatch`, módulo Go aparte; es cosa de Build, desde la versión base que indique el usuario; el manifiesto de una versión ya repartida no se sobrescribe nunca). El `VERSION.txt` de los zips sale de `gameVersion`. Un fichero NUEVO que los usuarios generen en la carpeta del juego debe quedar fuera de los zips (y de `forbiddenReason` en tools/mkpatch/manifest.go), como progress/settings.
- Informa con honestidad: di qué verificaste (tests, capturas) y qué no pudiste verificar.
- Si ves algo fuera de tu encargo, menciónalo en una frase y no lo toques.
- Tests aislados: los tests que mutan propKinds/Progress/globales (dbg*, progressFrozen, god, lang) deben restaurarlos con t.Cleanup (helper `snapshotPropKinds(t)` y `snapshotGlobals(t)` en `testiso_test.go`); verifica con `-shuffle=N`.

## 9b. Márgenes y separación en pantallas (obligatorio)

Regla del usuario, fija para siempre: **SIEMPRE MARGEN: ninguna caja, cuadro, pestaña, título, texto o botón puede ir pegado a otro ni al borde; separa todo y revisa todas las pantallas.** Vale para menús y para cualquier pantalla, ventana o panel nuevo.
- Mínimos (960x540): >= 12 px entre bloques distintos (título y cuadros, cuadro y cuadro, pestaña y contenido); >= 16 px al borde de pantalla (salvo marcos a sangre a propósito); texto dentro de su caja con >= 6 px de relleno; coordenadas y escalas enteras (texto nítido).
- Proceso: captura silenciosa (sección 7) de la pantalla en ES y EN y con las cifras/nombres más largos; y un TEST DE GEOMETRÍA que compruebe que los rectángulos no se tocan (distancia >= mínimo) ni salen de la pantalla. Patrón existente: `shop_layout_test.go`, `lang_box_test.go`, `TestShopPointsLineClearOfGroups`.
- Revisa 1 y 2 jugadores, y TODAS las pantallas que compartan el layout o la función tocada, no solo la que pidió el usuario.

## 10. Antes de entregar

- **VERSIÓN Y CHANGELOG (regla del usuario, 2026-11-10)**: cada tanda con novedades sube la versión semver en `version.go` (`gameVersion`; menor = x.y.Z, medio = x.Y.0, grande = X.0.0) y añade su entrada arriba en `CHANGELOG.md` (`## [vX.Y.Z] - AAAA-MM-DD - menor|medio|grande`, listas Añadido/Cambiado/Arreglado, en español, sin nombres exactos de personas reales ni de personajes: siempre la versión con una letra cambiada, sección 2 punto 7) ANTES del build; BUILD verifica que la constante, la cabecera del CHANGELOG y el `VERSION.txt` de los dos zips coinciden y lo reporta (`build.sh` se niega a compilar si constante y CHANGELOG no coinciden; `TestVersionMatchesChangelog`); los tags y commits son del usuario. La versión sale en el menú principal (abajo a la derecha), en Créditos y en el título de la ventana: todo lee `gameVersion`/`versionLabel()` (version.go), nunca un número escrito a mano.
Como agente de implementación entregas **tu informe** (sección 0), no un build. Sigue la parte A de `checklist.md` (en esta misma carpeta). El build, los zips, los binarios de la raíz y la limpieza los hacen sus agentes (`party-hard-release`, `party-hard-limpieza`) cuando QA ha dado el visto bueno.
- Regla de datos (2026-11-04): toda bandera de depuración/captura congela el progreso antes de cargar niveles (debugFlagsActive en main.go); las capturas se hacen desde una carpeta temporal con COPIA de assets, nunca desde el proyecto.

- Las capturas/ejecuciones con ventana solo se lanzan cuando el orquestador/usuario lo autoriza, en UNA tanda anunciada; el usuario puede estar jugando; preferir tests sin ventana (origen: QA abrió ventanas continuamente mientras el usuario jugaba, 2026-11-04). Usa el modo silencioso (automático con banderas de depuración o PH_QUIET=1: ventana fuera de pantalla, sin foco, prioridad baja) y -visible para lo contrario. Pasadas pesadas: `nice -n 19 go test -count=1 -p 2 -timeout 20m .`, `GOMAXPROCS=4`, `nice -n 19 ./build.sh`, o `tools/ph_quiet.sh <cmd>` (tardan más pero no bloquean el PC).

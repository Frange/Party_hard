---
name: party-hard-dev
description: Conocimiento y reglas del proyecto Party Hard (clon en Go/Ebiten con pixel art procedural). Léelo si eres un agente de IMPLEMENTACIÓN que va a tocar este repositorio: añadir personajes famosos, objetos (props), niveles o generadores, traducciones, escenas, 2 jugadores, depurar con capturas y banderas, trabajar con otros agentes a la vez. Los roles de orquestador, QA, build y limpieza tienen sus propios skills (party-hard-orquestador, party-hard-qa, party-hard-release, party-hard-limpieza).
---

# Party Hard: cómo se trabaja en este proyecto

Lee primero `docs/EQUIPO/LECCIONES.md` (reglas y checklist de lo que el usuario ya corrigió).

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
- **Editor** (pestañas Personajes, Objetos y animales, Escenarios) dentro del mismo binario: tecla F2, o `PartyHardEditor`. **Modo debug**: en el menú principal arriba, arriba, abajo, abajo, izquierda, derecha, izquierda, derecha; en el editor la misma secuencia desbloquea editar los niveles originales.
- **Dos jugadores** (interruptor en el menú principal, por defecto 1): `twoplayers.go` (segundo jugador `g.p2`, `g.players()`, `g.asPlayer`, `g.focus`), `input2.go` + `players.go` (teclas y mandos por jugador), `twojail.go` + `suspect.go` (esposas, detención y fuga), `revive.go` (ambulancia), `coop.go` (gags y marcador), `wardrobe.go` (ropa por jugador), `twoscenes.go` y `*2p.go` (cinemáticas). Todo cambio en el juego debe seguir idéntico con 1 jugador (hay tests de regresión).
- Los niveles son 21 en la campaña (`campaign.go`) más extras para el modo aleatorio. La isla de Jurassic Park y las escenas (intro, interludios, finales) quedan fuera de la campaña.

## 2. Añadir un personaje famoso

1. Mira un ejemplo completo: `monkey.go` + `monkey_lines.go`, `skelcast.go`, `kids.go`, o las listas `famousPresets` / `starPresets` (`presets*.go`, `cast2.go`). Cada personaje es un `def("slug", ...)` con un `CharSpec` procedural (pelo, ropa, sombrero, objeto en la mano...).
2. Nombre visible en `stars.go` / `funnynames.go`; descripción y precio de tienda en `shopdata.go` y `shopdesc.go`.
3. **Frases**: tablas de muletillas en `dialogue.go`, `jokes.go`, `greetings.go`, `duets.go` (o archivo propio como `monkey_lines.go`). **Máximo 48 caracteres por frase**, graciosas, cortas. Sin diálogos largos copiados de otros juegos: muletillas muy cortas o frases propias.
4. **Traducción inglesa** de cada texto nuevo en `lang_data.go` (ver sección 5).
5. **Un famoso solo puede estar una vez por nivel** (hay una función central; no la saltes). Tests: `famous_test.go`, `TestAllBuiltinsExported` (el editor exporta todos los personajes), `lang_test.go`.
6. Si el personaje es cuadrilla (varios a la vez) deben ser personajes distintos entre sí.

## 3. Añadir un objeto (prop)

- Los props viven en `props.go` … `props9.go`: `pk["nombre"] = &PropKind{W, H, FW, FH, Vars, Use, Draw: ...}`. `FW/FH` es la huella de colisión; `Use` son los puntos donde la gente lo usa.
- **Nunca pueden pisarse dos objetos**, ni objetos sobre vallas, carreteras, agua o fuera del terreno. `overlap.go` + `overlap_test.go` lo comprueban en todos los niveles: añade el prop nuevo a sus reglas si es grande o decorativo.
- Los niveles se colocan en `levels.go` (`P("nombre", x, y)`) o en los generadores `gen*.go`.

## 4. Añadir un nivel o generador

- Un generador devuelve un `LevelDef` (suelos, muros, props, zonas por donde se camina `Roam`, bailes, guardias, policía, puntos de inicio). Se registra en `campaign.go` (`kind`, `seed`, `title`).
- Tests que debes pasar: `collision_test.go` (alcanzabilidad), `overlap_test.go`, `famous_test.go`. Usa `-walkbot` para comprobar que nadie se atasca.
- El nivel debe permitir de 45 a 60 invitados sin atascos, y con 2 jugadores de 20 a 30 más (`cap.go` es el único punto que controla el tope; los ajustes de invitados están solo en DEBUG).

## 5. Traducir

- Todo texto visible pasa por `tr()` (se aplica solo en `drawText`, `textW`...). Escribe los textos en español.
- Cada texto español nuevo necesita su entrada en `lang_data.go` (`es -> en`), graciosa también en inglés. Si cambias un texto español, cambia la **clave** en `lang_data.go`: no dejes claves huérfanas.
- Comprueba con `go test -count=1 -run Lang .`.

## 6. Banderas de depuración (todas con `./PartyHard ...` o `go run . ...`)

**Niveles y capturas:** `-level N` (1 a 21) · `-shot ruta.png` + `-shottick T` (captura en el fotograma T y sale) · `-shotzoom x,y,w,h,zoom` · `-with slug,slug` (fuerza personajes) · `-fastevents` · `-immune` · `-walkbot` (anda al azar y cuenta atascos) · `-gamebench N` (ms por fotograma de Update y Draw) · `-jurassic` · `-jpunlocked` · `-levelselect` · `-event N` · `-lang en|es` · `-solids` · `-testroad` (+ `-roadcars a,b`) · `-vlane`.
**Cinemáticas:** `-title` / `-notitle` · `-firstrun` (simula la primera vez, nada se guarda) · `-intro` · `-ending 1|2|3` (+ `-endstage N`) · `-interlude 5|10|20` · `-scenes 1|2` · `-inviteauto N`.
**Sucesos y objetos:** `-zapnow` · `-poisonnow` · `-pyronow` · `-snakenow` · `-cheernow` · `-kidsnow` · `-bowlkids` · `-copsnow` · `-ring 1|2` · `-ambient vehículo`.
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
- Informa con honestidad: di qué verificaste (tests, capturas) y qué no pudiste verificar.
- Si ves algo fuera de tu encargo, menciónalo en una frase y no lo toques.

## 10. Antes de entregar

Como agente de implementación entregas **tu informe** (sección 0), no un build. Sigue la parte A de `checklist.md` (en esta misma carpeta). El build, los zips, los binarios de la raíz y la limpieza los hacen sus agentes (`party-hard-release`, `party-hard-limpieza`) cuando QA ha dado el visto bueno.

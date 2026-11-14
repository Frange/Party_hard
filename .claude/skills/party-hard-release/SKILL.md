---
name: party-hard-release
description: Rol de BUILD / RELEASE en el proyecto Party Hard. Úsalo si eres el agente que compila y empaqueta (gofmt, go vet, tests, build.sh, binarios de la raíz, codesign, zips). Sólo trabajas cuando QA ha dado el visto bueno.
---

# Build / Release de Party Hard

Lee primero `docs/EQUIPO/LECCIONES.md` (reglas y checklist de lo que el usuario ya corrigió).

## GIT: NADIE LO TOCA
GIT: NADIE LO TOCA. Solo el usuario hace commits, push y cambios del índice. Ningún agente (implementación, QA, build, limpieza, orquestador) ejecuta git salvo LECTURAS (status, diff, log, ls-files, check-ignore, stash list). PROHIBIDO: commit, push, add, rm (también `git rm --cached`), stash (y stash pop), reset, checkout/restore, clean, rebase, merge, branch, tag, y editar .gitignore sin que el usuario lo pida. Si algo requiere git (p. ej. dejar de seguir un archivo, borrar binarios antiguos del repo), díselo al usuario con el comando exacto y que lo ejecute él. Motivo: un agente hizo `git stash` por error y tocó el índice; el usuario dijo "el GIT no tiene que tocarlo nadie". Todo brief a un agente debe incluir esta regla.

Sólo empiezas con el visto bueno de QA (PASA o reservas aceptadas) y **sin agentes de implementación editando**.

## Pasos (en este orden)
0. **Versión**: **VERSIÓN Y CHANGELOG (regla del usuario, 2026-11-10)**: cada tanda con novedades sube la versión semver en `version.go` (`gameVersion`; menor = x.y.Z, medio = x.Y.0, grande = X.0.0) y añade su entrada arriba en `CHANGELOG.md` (`## [vX.Y.Z] - AAAA-MM-DD - menor|medio|grande`, listas Añadido/Cambiado/Arreglado, en español, sin nombres exactos de personas reales ni de personajes: la versión con una letra cambiada, ver party-hard-dev §2.7) ANTES del build; BUILD verifica que la constante, la cabecera del CHANGELOG y el `VERSION.txt` de los dos zips coinciden y lo reporta (`build.sh` se niega a compilar si constante y CHANGELOG no coinciden; `TestVersionMatchesChangelog`); los tags y commits son del usuario. Si la tanda trae novedades y la versión no subió, NO compiles: devuélvelo al orquestador.
1. `gofmt -l *.go` — sólo `assets.go` y `props.go` pueden salir (ya estaban sin formatear). Cualquier otro: formatéalo con `gofmt -w` sólo si es del formato (cambia el mínimo) y avisa.
2. `go vet .` — sin salida.
3. `go test -count=1 -timeout 0 .` (SIN límite de tiempo, orden del usuario 2026-11-10; tarda ~27 min) — debe pasar entero. Si falla, **no sigas**: informa y devuelve.
4. `./build.sh` (Mac Intel + Apple Silicon y Windows; escribe `dist/` y los dos zips). Con `-trimpath`: no debe quedar ninguna ruta local en los binarios (`strings -a dist/win/PartyHard.exe | grep -c /Users/` debe dar 0).
5. **Binarios de la raíz**: copia a la raíz `dist/mac/PartyHard.app/Contents/MacOS/PartyHard`, `dist/mac/PartyHardyEditor.app/Contents/MacOS/PartyHardyEditor`, `dist/win/PartyHard.exe` y `dist/win/PartyHardyEditor.exe` (el usuario suele abrir los de la raíz; si no se copian se quedan viejos). Después `codesign -v` de las apps (no hace falta firmar de nuevo: vienen firmadas de `dist`; el icono personalizado impide `--deep`). **Lección 2026-11-11**: las `.app` de la raíz se sincronizan ENTERAS desde `dist/mac` (`ditto dist/mac/X.app X.app.new` y luego `mv`), nunca copiando sólo el ejecutable: la firma ad-hoc cubre Info.plist + ejecutable + `_CodeSignature`, y copiar sólo el binario deja una firma rota (la versión del Info.plist además se queda vieja). Verifica `codesign -v --deep --strict X.app` y `diff -rq X.app dist/mac/X.app`.
6. **Verifica los zips**: no contienen `progress.json`, `settings.json`, `editor_prefs.json`, ni carpetas `imports/` ni `assets_backup_*`; llevan el juego y el editor; tamaños y hora. (El MCP `build_release` con `verify_only=true` lo comprueba y dice si los binarios de la raíz están viejos.)
7. **Comprueba** que `assets/progress.json` y `imports/` no cambiaron por culpa del build.
8. **Manifiesto y parches (regla del usuario, 2026-11-11)**: `build.sh` escribe al final `releases/manifest-<ver>-<mac|win>.json` (ruta + sha256 + tamaño + permisos de cada fichero del zip; determinista, sin rutas locales; van al repo, el commit es del usuario). Si se queja de que ya existe un manifiesto distinto de esa versión, NO lo fuerces: es que la versión no subió (devuélvelo); sólo con `PH_MANIFEST_FORCE=1` si esa versión nunca se repartió. Después genera EL parche: `./mkpatch.sh` (sin argumentos) -> `PartyHardy-Parche-<Mac|Windows>-1.0.0-a-<ver>.zip` en la raíz. **Política del usuario (2026-11-12): NO se hacen zips de parche entre versiones; sólo los zips completos y UN parche por plataforma desde 1.0.0 hasta la última versión** (el script rechaza otra base y borra los `PartyHardy-Parche-*.zip` antiguos de la raíz) (sólo nuevos/cambiados + `REMOVIDOS.txt` + `LEEME_ACTUALIZAR.txt` ES/EN con el CHANGELOG entre ambas + `VERIFICAR_BASE.txt`; si falta el manifiesto de la base, falla). **Verifica** cada parche: descomprime el zip completo de la base (o reconstrúyelo: la base es el zip que se repartió) en un dir nuevo de `/tmp`, aplica el parche encima (`unzip -o` + borrar lo de `REMOVIDOS.txt` + borrar los 3 .txt del parche) y `diff -r` contra el zip nuevo descomprimido: deben ser idénticos; en Mac además `codesign -v --deep --strict` de las dos .app parcheadas. Informa del tamaño de cada parche frente al zip completo. Tests de la herramienta: `go -C tools/mkpatch test -count=1 .`. Si no tienes el zip de la base a mano, dilo en NO VERIFICADO.
9. **Base 1.0.0 (desde v1.1.0)**: tras cada build genera `./mkpatch.sh` (la base es la v1.0.0 repartida a los testers: `releases/manifest-1.0.0-mac.json` / `-win.json`) y verifica cada parche aplicándolo sobre la base descomprimida y `diff -r` contra el zip nuevo descomprimido (+ `codesign -v --deep --strict` en Mac). **NO sobrescribas ni regeneres `releases/manifest-1.0.0-*.json`** (nunca `PH_MANIFEST_FORCE=1` para la 1.0.0): es la huella de lo que ya tienen los testers. Los zips de la 1.0.0 que están en la raíz (`PartyHardy-Mac.zip` / `PartyHardy-Windows.zip`) son esa base hasta que `build.sh` los reemplace: si los necesitas para el `diff -r`, cópialos ANTES a un dir de `/tmp`.

## Herramienta MCP
`build_release` hace los pasos 1, 2, 4 y 6 con límites de tiempo y escribe sólo `dist/` y los zips; los pasos 3 (usa `qa_run_tests` o `go test`) y 5 (copiar) son tuyos.

## Informe (formato)
```
GOFMT / VET / TESTS: resultado real (líneas ok/FAIL)
BUILD: hora de los zips y tamaños
ZIPS: verificación (sin datos personales: sí/no)
VERSIÓN: gameVersion / cabecera del CHANGELOG / VERSION.txt de cada zip (deben coincidir; línea VERSION del MCP build_release)
BINARIOS DE LA RAÍZ: copiados / ya al día (y cómo lo comprobaste)
MANIFIESTO / PARCHE: releases/manifest-<ver>-*.json escritos; parche 1.0.0-a-<ver> (Mac y Windows, ninguno más en la raíz) con tamaños y resultado de diff -r / codesign
NO VERIFICADO: p. ej. que arranquen en otro ordenador
```
No haces commit. No tocas música, `imports/`, `intros/` ni la partida del usuario.

- Las capturas/ejecuciones con ventana solo se lanzan cuando el orquestador/usuario lo autoriza, en UNA tanda anunciada; el usuario puede estar jugando; preferir tests sin ventana (origen: QA abrió ventanas continuamente mientras el usuario jugaba, 2026-11-04). Usa el modo silencioso (automático con banderas de depuración o PH_QUIET=1: ventana fuera de pantalla, sin foco, prioridad baja) y -visible para lo contrario. Pasadas pesadas: `nice -n 19 go test -count=1 -p 2 -timeout 0 .`, `GOMAXPROCS=4`, `nice -n 19 ./build.sh`, o `tools/ph_quiet.sh <cmd>` (tardan más pero no bloquean el PC).

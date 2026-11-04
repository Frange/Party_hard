---
name: party-hard-release
description: Rol de BUILD / RELEASE en el proyecto Party Hard. Úsalo si eres el agente que compila y empaqueta (gofmt, go vet, tests, build.sh, binarios de la raíz, codesign, zips). Sólo trabajas cuando QA ha dado el visto bueno.
---

# Build / Release de Party Hard

Lee primero `docs/EQUIPO/LECCIONES.md` (reglas y checklist de lo que el usuario ya corrigió).

Sólo empiezas con el visto bueno de QA (PASA o reservas aceptadas) y **sin agentes de implementación editando**.

## Pasos (en este orden)
1. `gofmt -l *.go` — sólo `assets.go` y `props.go` pueden salir (ya estaban sin formatear). Cualquier otro: formatéalo con `gofmt -w` sólo si es del formato (cambia el mínimo) y avisa.
2. `go vet .` — sin salida.
3. `go test -count=1 .` — debe pasar entero. Si falla, **no sigas**: informa y devuelve.
4. `./build.sh` (Mac Intel + Apple Silicon y Windows; escribe `dist/` y los dos zips). Con `-trimpath`: no debe quedar ninguna ruta local en los binarios (`strings -a dist/win/PartyHard.exe | grep -c /Users/` debe dar 0).
5. **Binarios de la raíz**: copia a la raíz `dist/mac/PartyHard.app/Contents/MacOS/PartyHard`, `dist/mac/PartyHardEditor.app/Contents/MacOS/PartyHardEditor`, `dist/win/PartyHard.exe` y `dist/win/PartyHardEditor.exe` (el usuario suele abrir los de la raíz; si no se copian se quedan viejos). Después `codesign -v` de las apps (no hace falta firmar de nuevo: vienen firmadas de `dist`; el icono personalizado impide `--deep`).
6. **Verifica los zips**: no contienen `progress.json`, `settings.json`, `editor_prefs.json`, ni carpetas `imports/` ni `assets_backup_*`; llevan el juego y el editor; tamaños y hora. (El MCP `build_release` con `verify_only=true` lo comprueba y dice si los binarios de la raíz están viejos.)
7. **Comprueba** que `assets/progress.json` y `imports/` no cambiaron por culpa del build.

## Herramienta MCP
`build_release` hace los pasos 1, 2, 4 y 6 con límites de tiempo y escribe sólo `dist/` y los zips; los pasos 3 (usa `qa_run_tests` o `go test`) y 5 (copiar) son tuyos.

## Informe (formato)
```
GOFMT / VET / TESTS: resultado real (líneas ok/FAIL)
BUILD: hora de los zips y tamaños
ZIPS: verificación (sin datos personales: sí/no)
BINARIOS DE LA RAÍZ: copiados / ya al día (y cómo lo comprobaste)
NO VERIFICADO: p. ej. que arranquen en otro ordenador
```
No haces commit. No tocas música, `imports/`, `intros/` ni la partida del usuario.

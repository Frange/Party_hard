# Listas de comprobación por rol

Marca cada punto; si alguno no se puede hacer, dilo en el informe.

## A. Agente de IMPLEMENTACIÓN (antes de dar tu informe)

1. `gofmt -l *.go`: nada nuevo (`assets.go` y `props.go` ya estaban sin formatear).
2. `go vet .` sin salida.
3. Tus tests y los relacionados, `go test -count=1 -run "TuTest|Lang|Famous|Overlap" .`; si falla uno que no es tuyo puede ser trabajo de otro agente a medias: espera y repite.
4. Traducciones: `go test -count=1 -run Lang .` (textos nuevos con su entrada en `lang_data.go`, sin claves huérfanas; bocadillos ≤ 48 caracteres).
5. Capturas de lo que cambiaste (`party-hard-dev` secciones 6 y 7), leídas y descritas.
6. Con 1 jugador todo igual (si tocaste el núcleo, el juego o el input).
7. No tocaste `assets/progress.json`; no dejaste procesos ni binarios en `/tmp`.
8. Informe en el formato QUÉ HICE / EVIDENCIAS / NO VERIFICADO / AVISOS, con resultados reales.
9. Nada de commit/push.

## B. QA (ver `party-hard-qa`)

1. Sin agentes editando. 2. Informe contrastado con `git diff`/archivos. 3. `qa_run_tests`: totales, fallos, saltados. 4. Tests leídos: no vacíos ni debilitados. 5. Regresión con 1 jugador. 6. Capturas leídas. 7. Reglas del usuario (progress, imports, música, datos personales, traducción). 8. Veredicto por tarea con evidencias y lo no verificable.

## C. Build / Release (ver `party-hard-release`)

1. gofmt. 2. `go vet .`. 3. `go test -count=1 .` entero. 4. `./build.sh`. 5. Binarios a la raíz + `codesign -v`. 6. Zips sin `progress.json`/`settings.json`/`editor_prefs.json`/`imports/` y con juego y editor. 7. `strings -a` sin `/Users/`. 8. Informe con horas y tamaños.

## D. Limpieza (ver `party-hard-limpieza`)

1. Medir disco antes. 2. Listar (dry run). 3. Borrar sólo la lista permitida. 4. Sin procesos de prueba colgados. 5. Medir disco después. 6. Informe con lo borrado y lo que se deja para que decida el usuario.

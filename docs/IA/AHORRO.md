# Cómo gastar menos uso de Claude Code en este proyecto

Escrito para no expertos. Sin datos personales. Fecha de la consulta de documentación: noviembre de 2026.

Convención de fiabilidad en todo el documento:
- **[DOC]** = lo dice la documentación oficial (con enlace).
- **[GEN]** = conocimiento general que no he podido comprobar en la documentación consultada.
- **[EST]** = estimación mía a partir de cómo funciona el flujo; no es una medición.

---

## TL;DR

1. **Lo que más cuesta es el tamaño del contexto multiplicado por el número de peticiones.** Claude Code reenvía toda la conversación en cada petición (la parte repetida sale más barata por la caché, pero no es gratis) [DOC: costs, "Why usage climbs"].
2. **Un `fork` copia toda la conversación y siempre usa el modelo grande del orquestador** (el campo `model` se ignora en los forks) [DOC: sub-agents, "How forks differ"]. Si el orquestador tiene un contexto enorme, cada fork nace "gordo". Para tareas con un encargo claro es mejor un **agente fresco** con un brief corto y `model: sonnet` o `haiku`.
3. **Modelo por rol:** grande solo para orquestar y decisiones difíciles; Sonnet para implementar; Haiku para BUILD, LIMPIEZA y búsquedas; QA con Sonnet (o Haiku si solo ejecuta y resume) [DOC recomienda Sonnet por defecto y Haiku para subagentes simples].
4. **Informes cortos con formato fijo** (máx. 15 líneas) y evidencias en forma de una línea por comando. El informe de cada agente entra en el contexto del orquestador y se vuelve a pagar en cada turno posterior [EST].
5. **Una sola pasada de build/tests por entrega**, `go test -run` para lo tocado, tests completos solo en BUILD, y capturas pocas y pequeñas.
6. **`/clear` entre tareas distintas y `/compact` en pausas naturales**, con un `CLAUDE.md` corto (<200 líneas) que contenga el "estado del proyecto" para no reexplicar [DOC].
7. **Mide antes de recortar**: `/usage` (antes `/cost`), `/context`, `/skill-doctor`, `/insights`.

---

## A. Dónde se gasta más (ordenado por impacto estimado)

Orden basado en el funcionamiento documentado y en el flujo descrito. **No hay medición real de este proyecto**: confírmalo con `/usage` (sección E) [EST].

| # | Fuente de gasto | Por qué cuesta | Impacto [EST] |
|---|---|---|---|
| 1 | **Forks con contexto heredado** | El fork recibe todo el historial y el mismo modelo grande. Se comparte la caché del padre (la primera petición lee caché), pero cada petición posterior del fork sigue procesando un contexto enorme, y se multiplica por N forks en paralelo [DOC: sub-agents]. | Muy alto |
| 2 | **Modelo grande en tareas mecánicas** (build, limpieza, QA que solo ejecuta) | El modelo grande consume bastante más cupo por token que Sonnet/Haiku. El propio manual de costes dice: "Sonnet maneja la mayoría de tareas; reserva Opus para decisiones complejas" [DOC: costs]. Una subida a un modelo grande también afecta a los subagentes que heredan modelo [DOC]. | Muy alto |
| 3 | **Contexto largo del orquestador** | Cada turno (y cada resultado de herramienta) reenvía la conversación entera. Una sesión abierta todo el día gasta mucho aunque la pregunta sea de una línea [DOC: costs]. | Alto |
| 4 | **Informes largos de agentes** | Todo lo que devuelve un agente se añade al contexto del orquestador y se relee en cada turno siguiente [DOC: solo vuelve el informe final; EST: el coste por relectura]. | Alto |
| 5 | **Builds/tests completos repetidos** | `go test` ~150-200 s no consume tokens por sí solo, pero cada repetición suele traer una salida larga al contexto, más turnos de espera y más análisis. Cuesta en tokens (salida) y en tiempo [EST]. | Medio-alto |
| 6 | **Thinking/esfuerzo alto por defecto** | El thinking se factura como salida y el presupuesto por defecto puede ser de decenas de miles de tokens por petición [DOC: costs]. En Opus/Sonnet/Haiku 5.5 no se puede desactivar, pero sí bajar el `/effort` [DOC]. | Medio-alto |
| 7 | **Capturas frecuentes y grandes** | Las imágenes entran en el contexto y se reenvían; al acumular muchas, Claude Code elimina lotes de las más antiguas y eso fuerza una petición sin caché [DOC: prompt-caching, "Accumulating many images"]. El coste exacto por imagen depende de su tamaño [GEN]. | Medio |
| 8 | **Relecturas de ficheros enormes** (`game.go`, `draw.go`, `lang_data.go`) | Cada lectura completa mete miles de líneas al contexto. Buscar con Grep y leer solo el rango necesario cuesta una fracción [DOC: "prompts específicos" reducen lecturas; EST el tamaño]. | Medio |
| 9 | **Configuración cargada en cada agente** | Cada subagente nuevo carga CLAUDE.md, instantánea de `git status`, skills precargados y las herramientas MCP [DOC: sub-agents]. Aquí `git status` lista ~120 ficheros modificados/sin seguimiento: se repite en cada agente [EST]. | Medio-bajo |
| 10 | **MCP `party-hard`** | Las definiciones de herramientas MCP están **diferidas por defecto** (solo entran los nombres) [DOC: costs, "Reduce MCP server overhead"], así que el coste es bajo. Ojo: añadir o quitar un servidor MCP a mitad de sesión puede invalidar la caché si las herramientas no están diferidas [DOC: prompt-caching]. | Bajo |
| 11 | **Skills grandes** | La descripción siempre está en contexto; el cuerpo solo al invocarlo, pero una vez cargado se queda en la conversación [DOC: skills]. `party-hard-dev/SKILL.md` ocupa ~10 KB (cuerpo bajo el límite de 500 líneas, bien) [EST: ~2.500-3.000 tokens cada vez que un agente lo carga]. | Bajo-medio |
| 12 | **Fugas en segundo plano** | Resumen de sesión, compactación por inactividad, sugerencias de prompt, tareas `/loop`, agentes que siguen vivos [DOC: costs, "Background token usage"]. | Bajo |

---

## B. Recomendaciones concretas

### B1. Modelo por rol

| Rol | Modelo recomendado | Esfuerzo | Motivo |
|---|---|---|---|
| Orquestador (habla con el usuario, decide, reparte) | El grande **solo si** hay decisiones de diseño o conflictos difíciles; si no, Sonnet | medio | Su contexto es el más largo; cada turno es caro. Cuanto menos "trabaje", menos turnos. |
| Implementación (un área, archivos propios) | **Sonnet** | medio | Cubre la mayoría del código [DOC]. Subir al grande solo en refactors delicados o bugs que Sonnet no resuelve a la segunda. |
| QA (revisa evidencias, ejecuta tests concretos) | **Sonnet** (revisión de código y veredicto) o **Haiku** (si solo ejecuta y resume) | bajo-medio | Haiku para ejecutar y resumir; Sonnet cuando hay que juzgar si un test prueba de verdad. No bajar de aquí: es la red de seguridad. |
| BUILD / Release | **Haiku** | bajo | Es ejecutar `build.sh`, copiar y comprimir, y reportar. Poco razonamiento. |
| LIMPIEZA | **Haiku** | bajo | Procesos, temporales, caché. Mecánico, con la herramienta `clean_temp` y lista permitida. |
| Búsqueda/investigación ("¿dónde se define X?") | **Haiku** (o el agente integrado Explore con modelo barato) | bajo | Lectura, no edición. |
| Revisión final de una entrega grande | Sonnet, y el grande solo si hay dudas | alto una vez | Una pasada, no varias. |

Cómo se aplica [DOC: sub-agents, "Choose a model"]:
- Al lanzar un agente **no fork**, se puede pasar `model` (alias `sonnet`, `haiku`, `opus`). El orden de prioridad es: parámetro de la llamada, luego el `model:` del archivo del agente, luego la variable `CLAUDE_CODE_SUBAGENT_MODEL`, luego el modelo de la conversación.
- Mejor aún: **definir agentes en `.claude/agents/*.md`** con `model: haiku` o `sonnet` y `tools:` limitadas. Así el modelo barato queda fijado y no depende de que el orquestador se acuerde.
- Los forks ignoran el campo `model`: **un fork siempre cuesta como el orquestador**.
- Cambiar de modelo en medio de una conversación fuerza un turno sin caché [DOC: prompt-caching, "Switching models"]. Elige el modelo del orquestador al empezar la sesión y no lo cambies a mitad de tarea.

### B2. Agente fresco con brief preciso frente a fork

Documentado [DOC: sub-agents]: el fork comparte la caché del padre, así que **su primera petición es más barata que la de un agente fresco que necesita el mismo contexto**. Pero el fork sigue llevando todo el historial en cada petición y usa el modelo grande.

Regla práctica [EST]:
- **Agente fresco (por defecto):** si el encargo se puede explicar en 10-20 líneas (objetivo, archivos propios, criterio de hecho, límites). Es el caso normal de implementación, QA, build y limpieza en este proyecto, porque ya existen skills con las reglas.
- **Fork:** solo si el trabajo depende de decenas de matices de la conversación actual que costaría más escribir que heredar, y aun así con el orquestador "ligero" (tras `/compact`).
- Un agente fresco carga igualmente CLAUDE.md, git status y sus skills, así que **esos deben ser cortos**.
- Un agente que ya trabajó en un área puede **continuarse** con `SendMessage` en vez de crear otro nuevo y reexplicar [DOC]. Útil para "arregla esto que te devolvió QA".

### B3. Límites de informe

Reglas para todos los agentes (plantillas en la sección D):
- Máximo **15 líneas** (QA: veredicto + tabla de hasta 10 filas).
- Una línea por comando ejecutado: `comando -> resultado real` (por ejemplo `go test -run Lang . -> ok 1.2s`). Nunca pegar la salida completa; solo las líneas de fallo.
- No repetir el encargo ni lo que el orquestador ya sabe.
- Rutas de capturas y **una frase** de lo que se ve. Sin descripciones largas.
- `NO VERIFICADO` y `AVISOS`, cada uno máximo 3 líneas.
- Sin código pegado salvo que sea imprescindible (<10 líneas).

Importante: esto **no** significa informar con menos honestidad; significa menos relleno. La sección F explica qué no recortar.

### B4. Contexto: `/clear`, `/compact`, CLAUDE.md

Documentado [DOC: costs, prompt-caching, memory]:
- `/clear` al cambiar de tema. Es gratis y resetea el contador de la sesión. Usa `/rename` antes si quieres poder volver con `/resume`.
- `/compact` en una pausa natural (entre tareas), no a mitad. Puedes indicar qué conservar: `/compact Conserva: decisiones del usuario, estado de las entregas pendientes, rutas de ficheros tocados`. Compactar un contexto enorme es una petición grande en sí misma; más barato si la caché sigue caliente (menos de 5 min/1 h según plan) y mucho más caro tras una pausa larga.
- Editar CLAUDE.md a mitad de sesión **no se aplica** hasta `/clear`, `/compact` o reinicio [DOC]. Escríbelo bien antes.
- Mantén CLAUDE.md **bajo 200 líneas**; lo específico va a skills, que cargan bajo demanda [DOC]. `@imports` no reducen el coste.
- Hoy **no existe CLAUDE.md** en el proyecto [verificado]. Propuesta de contenido (mínimo útil, ~30 líneas):

```markdown
# Party Hard (Go + Ebiten, un solo package main en la raíz)
- Trabajas dentro de un flujo con roles: orquestador / implementación / QA / build / limpieza. Detalle en .claude/skills/.
- NUNCA: commit/push sin pedirlo, tocar assets/progress.json, subir música o imports/, dejar procesos abiertos.
- Repo PÚBLICO: ningún dato personal.
- Informes: máx. 15 líneas, formato fijo, una línea por comando con su resultado real, sin repetir el encargo.
- Pruebas: sólo lo afectado (go test -run X .). El test completo lo lanza únicamente el agente BUILD, una vez por entrega.
- Ficheros enormes (game.go, draw.go, lang_data.go): buscar con Grep y leer solo el rango necesario.
- Capturas: pocas, pequeñas (-shotzoom), una por comprobación.
## Estado del proyecto (actualizar al cerrar cada entrega, máx. 10 líneas)
- Última entrega: <fecha/etiqueta>. Pendiente: <lista corta>. Decisiones abiertas: <lista corta>.
# Compact instructions
Al compactar conserva: decisiones del usuario, pendientes, rutas de ficheros tocados, resultado de la última entrega.
```

### B5. Skills y MCP

- `party-hard-dev/SKILL.md` (~10 KB, ~160 líneas) está dentro del límite recomendado (<500 líneas) [DOC]. Mejoras opcionales [EST]: mover las secciones largas que solo se usan a veces (banderas de depuración, sección 6) a un archivo de apoyo `banderas.md` que se lee solo si hace falta, y dejar en el SKILL.md solo lo que todo agente de implementación necesita.
- Revisa que las descripciones de los skills sean cortas y específicas: **todas las descripciones están siempre en contexto** [DOC]. Con `/skill-doctor` se ve el coste de cada skill y su uso [DOC].
- Los skills de rol (orquestador, qa, release, limpieza) pueden ser `disable-model-invocation: true` si solo se invocan por nombre desde el encargo; así su descripción deja de estar en el contexto [DOC: skills]. Cuidado: eso impide que el agente los cargue solo; el encargo debe nombrarlos o precargarlos.
- MCP `party-hard`: está bien (herramientas diferidas, pocas, sin dependencias). Vale más que lanzar `go test` a pelo si devuelve **resúmenes cortos** en vez de salida completa [EST]. Comprueba con `/context` cuánto ocupa.
- Una regla de la documentación: las herramientas de línea de comandos ya instaladas (`gh`, `go`) son más baratas en contexto que un servidor MCP [DOC]. Para `go test` y `go build` basta un script con salida filtrada; el MCP aporta límites de tiempo y seguridad, no ahorro de tokens por sí mismo.
- Un hook `PreToolUse` puede filtrar la salida de `go test` para dejar solo FAIL/error [DOC: ejemplo oficial en costs] (ver G).

### B6. Builds, tests y capturas

- **Tests por alcance:** el implementador ejecuta solo `go vet .` y `go test -run '<patrón de lo tocado>' .` (por ejemplo `-run Lang`, `-run Famous`). El test completo (~150-200 s) lo ejecuta **solo BUILD, una vez por entrega, con `-count=1`** [EST; la regla ya está en el flujo de entrega].
- **No repetir builds:** una pasada de `build.sh` por entrega. Si QA pide un arreglo pequeño, se re-verifica lo tocado; solo se repite el build completo si cambió código tras el último build.
- **Cachear resultados:** `go test` ya cachea resultados salvo que uses `-count=1` [GEN: comportamiento estándar de Go]. Usa `-count=1` solo en la pasada final. Anota en el informe el último commit/estado en que pasó el test completo para no repetirlo.
- **Salida filtrada:** `go test ./... 2>&1 | grep -E 'FAIL|panic|^ok|error' | head -40`.
- **Capturas:** política [EST]: máximo 1-2 por comprobación; usa `-shotzoom x,y,w,h,zoom` para recortar la zona que importa en vez de pantalla completa; no repitas capturas del mismo estado; el agente que la hace dice qué ve en una frase y el orquestador **no** vuelve a abrirla salvo duda. Las imágenes que quedan en el contexto se reenvían en cada petición [DOC: prompt-caching]; deja las capturas en los agentes, no en el orquestador.
- **Parar lo atascado:** `maxTurns` limita los turnos de un agente y devuelve salida parcial [DOC]. Pon un límite razonable (p. ej. 25-40 turnos para implementación, 10-15 para build/limpieza) y detén un agente que repite el mismo error dos veces.

### B7. Reducir relecturas

- Ficheros enormes: `Grep` con número de línea y `Read` con `offset`/`limit` [DOC: lecturas específicas cuestan menos]. Nunca leer `game.go` entero.
- Un agente que ya leyó un fichero **no debe releerlo** salvo que otro agente lo haya cambiado (el sistema avisa con un recordatorio) [DOC: prompt-caching, "Editing files"].
- El orquestador no relee código ni informes antiguos: confía en la sección "Estado" de CLAUDE.md y en el informe corto.
- Los plugins de inteligencia de código (LSP) permiten "ir a la definición" en vez de grep + varias lecturas [DOC: costs]. Para Go existe `gopls`; **no verificado** que haya plugin oficial instalable en esta configuración.

### B8. Paralelismo y reparto

- El coste es aproximadamente proporcional al número de agentes activos y a cuánto trabajan [DOC: costs, "Agent team token costs"; la documentación habla de equipos de agentes y da ~7x tokens en modo plan; los subagentes sencillos no tienen esa cifra].
- Límite orientativo [EST]: **máx. 3-4 agentes de implementación a la vez**, cada uno con archivos propios. Más agentes no acelera lo que se bloquea en el mismo `game.go` y multiplica el coste y las colisiones.
- Agrupa las peticiones del usuario (batching): una ronda con 5 peticiones numeradas cuesta menos que 5 rondas con el contexto completo reenviándose [EST].
- Cada agente se cierra al terminar; no dejes agentes esperando.

### B9. Imágenes en contexto

- Pide al usuario capturas recortadas con el problema marcado (como ya hace la guía) y evita enviar varias del mismo estado.
- Tras analizar una imagen, el orquestador no necesita conservarla: `/clear` o `/compact` al cerrar la tarea las elimina del historial.

### B10. Esfuerzo/thinking

- Baja `/effort` para QA/BUILD/LIMPIEZA y tareas rutinarias; sube solo para diagnósticos difíciles [DOC: costs].
- `MAX_THINKING_TOKENS` solo vale en modelos con presupuesto fijo; los de razonamiento adaptativo lo ignoran [DOC].
- Cambiar el esfuerzo a mitad de sesión puede invalidar la caché en algunos modelos; en Opus/Sonnet/Haiku 5.5 con suscripción o API no la invalida [DOC: prompt-caching].

### B11. Reducir lo que cada agente carga al nacer

- La instantánea de `git status` (~120 líneas hoy) se inyecta en la conversación y en cada subagente (salvo Explore/Plan, que no cargan CLAUDE.md ni git status) [DOC]. Hacer commit de lo estable (decisión del usuario, no de los agentes) o añadir al `.gitignore` lo que no se debe versionar reduce ese ruido [EST].
- En agentes personalizados, `omitClaudeMd: true` evita cargar CLAUDE.md cuando el agente no lo necesita (por ejemplo LIMPIEZA con encargo autosuficiente) [DOC].

---

## C. Presupuesto orientativo y reglas de decisión

**Presupuesto: son órdenes de magnitud relativas, estimadas por mí [EST]. No son cifras medidas.** Unidad: "1 = una conversación corta con Sonnet". Compruébalo con `/usage` y ajusta.

| Tarea | Agente | Modelo | Coste relativo [EST] | Turnos máx. | Informe |
|---|---|---|---|---|---|
| Cambio pequeño (<3 archivos, p. ej. un texto, una frase de un personaje) | 1 fresco | sonnet | 1-2 | 20 | 8 líneas |
| Cambio medio (3-8 archivos, un sistema) | 1 fresco | sonnet | 3-6 | 40 | 15 líneas |
| Cambio grande (muchas áreas) | 2-4 frescos, archivos propios | sonnet | 10-20 | 40 c/u | 15 líneas c/u |
| Bug difícil / diseño delicado | 1 | grande (o sonnet con esfuerzo alto primero) | 5-10 | 40 | 15 líneas |
| Investigación (¿dónde está X?) | 1 | haiku | 0,3-1 | 15 | 8 líneas |
| QA de una entrega | 1 | sonnet (haiku si solo ejecuta) | 2-4 | 25 | veredicto + tabla |
| BUILD (gofmt, vet, test completo, build.sh, zips) | 1 | haiku | 1-2 (+ tiempo de máquina) | 15 | 8 líneas |
| LIMPIEZA | 1 | haiku | 0,3-0,5 | 10 | 5 líneas |
| Orquestador por turno | (sesión) | sonnet/grande | crece con el contexto | — | — |

Reglas de decisión [EST]:

```
¿La tarea es solo leer/buscar?                         -> 1 agente fresco haiku.
¿Es ejecutar comandos y resumir (build, limpieza)?     -> 1 agente fresco haiku, tools limitadas.
¿Toca <3 archivos y está clara?                        -> 1 agente fresco sonnet, esfuerzo bajo-medio.
¿Toca 3-8 archivos de un mismo sistema?                -> 1 agente fresco sonnet.
¿Toca varias áreas con archivos DISTINTOS?             -> hasta 3-4 agentes frescos sonnet en paralelo.
¿Toca el mismo archivo grande o hay dependencias?      -> en serie, no en paralelo.
¿Sonnet falla dos veces en lo mismo?                   -> reintento con el modelo grande (una vez), no más.
¿Necesita 30 matices de la conversación?               -> fork, tras /compact del orquestador.
¿Un agente repite el mismo error dos veces?            -> pararlo, diagnosticar, relanzar con brief nuevo.
¿QA devuelve "falta evidencia"?                        -> SendMessage al mismo agente (conserva su contexto).
```

---

## D. Plantillas para copiar

### D1. Encargo corto a un agente (implementación)

```
ROL: implementación. Skill: party-hard-dev (léelo).
OBJETIVO: <una frase>.
CONTEXTO: <2-4 líneas: dónde pasa, qué se ve mal/qué falta>.
ARCHIVOS PROPIOS: <lista>. No toques otros salvo Edit mínimo en game.go/draw.go/lang_data.go (relee antes).
HECHO CUANDO: <criterio comprobable: test X pasa, captura muestra Y>.
LÍMITES: sin commit, sin build.sh, sin tocar assets/progress.json, sin procesos abiertos.
PRUEBAS: solo `go vet .` y `go test -run '<patrón>' .`. NO el test completo.
CAPTURAS: máx. 2, con -shotzoom si es posible.
INFORME: máx. 15 líneas, formato abajo. No repitas este encargo.
```
Al lanzarlo: agente fresco (no fork), `model: sonnet`.

### D2. Informe corto (cualquier agente)

```
HECHO (máx. 4 líneas): <qué cambió y en qué ficheros>
EVIDENCIAS (una línea por comando):
- <comando> -> <resultado real>
- Tests nuevos: <nombres>
- Capturas: <ruta> — <qué se ve en una frase>
NO VERIFICADO (máx. 3 líneas): <audio, mando, 2 personas, etc. o "nada">
AVISOS (máx. 3 líneas): <cosas fuera de encargo vistas, sin tocar, o "ninguno">
```

Informe de QA:
```
VEREDICTO: APTO | NO APTO | FALTA EVIDENCIA
TABLA (máx. 10 filas): comprobación | evidencia vista (comando/captura) | OK/FALLO
RIESGOS (máx. 3 líneas)
```

### D3. Reglas para el orquestador

```
1. No edito, no compilo, no lanzo tests ni capturas. Reparto y exijo evidencias.
2. Agentes FRESCOS con el encargo D1 y modelo explícito (haiku/sonnet). Fork solo con motivo escrito.
3. Máx. 3-4 agentes de implementación a la vez, con archivos disjuntos.
4. No releo código ni capturas: confío en el informe corto. Si falta una evidencia, SendMessage al mismo agente.
5. El test completo y el build los hace BUILD una sola vez por entrega, tras el visto bueno de QA.
6. Agrupo las peticiones del usuario en rondas numeradas.
7. Al cerrar una entrega: actualizo "Estado del proyecto" en CLAUDE.md y propongo /compact o /clear.
8. Si un agente se atasca (mismo error 2 veces o >turnos máx.), lo paro y diagnostico antes de relanzar.
9. Reviso /usage al cerrar cada entrega: si una fila supera el 25% del uso, actúo sobre ella.
10. Mis respuestas al usuario: cortas, con el estado, lo decidido y lo que necesito de él.
```

---

## E. Cómo medir

Verificado en la documentación oficial [DOC: https://code.claude.com/docs/en/costs]:

| Qué | Cómo |
|---|---|
| Uso de la sesión (tokens, coste estimado a precio de lista, uso por modelo) | `/usage`, bloque "Session". El comando antiguo `/cost` ya no aparece en esa página; **usa `/usage`** |
| Qué recomienda cambiar (planes Pro/Max/Team) | `/usage`: reparto del uso por skills, subagentes, plugins y servidores MCP, con recomendaciones; tecla `f` aplica las que pueda; `d`/`w` cambian entre 24 h y 7 días |
| Qué ocupa el contexto ahora mismo | `/context` |
| Coste de cada skill | `/skill-doctor` |
| Estado de la caché (aciertos, fallos, si está caliente) | `/usage`, línea "Prompt cache (main)" (v2.1.251+). Solo cubre la conversación principal, no los subagentes |
| Informe de cómo trabajas (fricciones, sugerencias) | `/insights` (genera un HTML; también gasta tokens) |
| Contexto/coste siempre visible | Configurar la línea de estado (`statusline`) |
| Servidores MCP activos | `/mcp` |
| Hooks activos | `/hooks` |
| Panel de agentes (ver y parar agentes en curso) | Panel bajo el prompt; `x` para parar |

Importante [DOC]: con suscripción (Pro/Max) la cifra de dólares de `/usage` **no es lo que se te factura**; lo relevante son las barras de límite del plan. La cifra es una estimación a precio de lista.

**No verificado:**
- `ccusage` (herramienta de terceros que lee los registros locales y resume tokens/coste por día o sesión). La conozco por conocimiento general, no la he comprobado en esta sesión ni está en la documentación oficial. Si la usas, trátala como complemento.
- Precios actuales por modelo: consúltalos en https://platform.claude.com/docs/en/about-claude/pricing (no los copio aquí para no dejar cifras que caduquen).
- Multiplicadores de caché (lectura mucho más barata que entrada normal; escritura algo más cara; TTL de 5 min o 1 h): la documentación consultada confirma que existen ambos TTL y que la escritura de 1 h es más cara, pero **no he verificado los factores exactos** [GEN: lectura ≈ 0,1x, escritura 5 min ≈ 1,25x, 1 h ≈ 2x].
- Los subagentes tienen caché de 5 min incluso con suscripción (el contexto principal tiene 1 h) salvo que se configure `subagentPromptCacheTtl` [DOC: prompt-caching]. Consecuencia [EST]: un agente que espera >5 min entre peticiones paga el contexto sin caché.

Cómo medir **este flujo** (propuesta [EST]): anota el porcentaje de uso del plan (barras de `/usage`) al empezar y terminar cada entrega, y la fila de "subagentes" del desglose. Tras aplicar cambios, compara 2-3 entregas parecidas.

---

## F. Riesgos de ahorrar demasiado y qué NO recortar

Riesgos [EST]:
- **Haiku en tareas con juicio** (revisar si un test prueba de verdad) puede dar falsos "OK". Por eso QA va con Sonnet cuando juzga.
- **Informes demasiado cortos** esconden lo no verificado. El formato mantiene `NO VERIFICADO` y `AVISOS` aunque sea con "nada".
- **Menos tests** llevan a regresiones: ejecutar solo `-run` sin pasada completa final es peligroso.
- **Menos capturas** en cambios visuales dejan pasar fallos que los tests no ven (el juego es pixel art procedural).
- **Agentes frescos con brief incompleto** repiten trabajo o rompen reglas del proyecto. Siempre referencia el skill y los límites.
- **Contexto compactado en mal momento** pierde decisiones del usuario. Las decisiones y pendientes van a CLAUDE.md antes de compactar.

**NO recortar:**
1. QA independiente antes de cada entrega y la exigencia de evidencias reales (copiar la línea `ok  partyhard ...`).
2. La pasada completa de `go test -count=1` y de `build.sh` antes de entregar.
3. Las reglas de seguridad: no tocar `assets/progress.json`, repo público sin datos personales, no subir música ni `imports/`, no commit sin permiso, limpieza de procesos.
4. Una captura de verificación en cambios visuales y el "dime qué no pudiste verificar".
5. El razonamiento adecuado en bugs difíciles (mejor un intento con modelo grande que cinco con uno inadecuado).

---

## G. Mejoras opcionales (requieren decisión del usuario)

1. Crear `CLAUDE.md` (sección B4) y agentes en `.claude/agents/` con modelo fijado: `qa.md` (sonnet), `build.md` (haiku), `limpieza.md` (haiku), `implementacion.md` (sonnet), cada uno con `tools:` limitadas y `maxTurns`. Ejemplo mínimo [DOC: formato]:

```markdown
---
name: build
description: Ejecuta gofmt, vet, tests completos y build.sh de una entrega y resume el resultado. Úsalo solo tras el visto bueno de QA.
model: haiku
tools: Bash, Read, Grep
maxTurns: 15
---
Sigue el skill party-hard-release. Un único build por entrega. Informe de máx. 8 líneas:
una línea por comando con su resultado real; NO VERIFICADO; AVISOS.
```
(Si tus roles ya están definidos como skills, el agente solo necesita apuntar a ellos.)

2. Hook `PreToolUse` que filtre la salida de `go test`/`go build` a FAIL/error (adaptación del ejemplo oficial de costs) [DOC]. Asegura que ningún agente pega 500 líneas de salida.
3. Variable `CLAUDE_CODE_SUBAGENT_MODEL=sonnet` como valor por defecto de subagentes no fork y agentes con modelo explícito para BUILD/LIMPIEZA [DOC]. Ojo: no afecta a los forks.
4. Poner `"Agent(fork)"` en `permissions.deny` si se quiere impedir forks por completo [DOC]. (Decisión del usuario; un fork puntual tras `/compact` puede ser útil.)
5. Reducir `git status`: commit de lo estable cuando el usuario quiera.
6. Desactivar lo que no uses: sugerencias de prompt, `/loop`, compactación por inactividad (`idleCompaction`) si no te sirve [DOC].

---

## Enlaces consultados (documentación oficial)

- Costes: https://code.claude.com/docs/en/costs
- Subagentes (modelo por agente, forks): https://code.claude.com/docs/en/sub-agents
- Skills (carga bajo demanda, tamaños): https://code.claude.com/docs/en/skills
- Memoria y CLAUDE.md: https://code.claude.com/docs/en/memory
- Caché de prompts en Claude Code: https://code.claude.com/docs/en/prompt-caching
- Precios y caché en la API (no leído en detalle): https://platform.claude.com/docs/en/about-claude/pricing y https://platform.claude.com/docs/en/build-with-claude/prompt-caching
- Hooks, MCP y contexto, enlazados desde la página de costes: https://code.claude.com/docs/en/hooks.md , https://code.claude.com/docs/en/mcp.md , https://code.claude.com/docs/en/context-window.md

Limitaciones de esta investigación: las páginas largas (subagentes, skills, memoria) las leí solo en parte (resumidas por el lector web); no leí hooks, MCP ni context-window en detalle; no hice mediciones reales de consumo de este proyecto.

# Guía: Skills, MCP y cómo dirigir a Claude

Esta guía está escrita para alguien que no es experto. Todo lo que aparece aquí está sacado de este proyecto (Party Hard) y no contiene datos personales.

Partes: **A** qué son un Skill y un MCP · **B** cómo probar lo creado · **C** cómo dirigir a Claude · **D** el modo orquestador: cómo trabajamos ahora (quién hace qué, cómo leer el veredicto de QA).

---

## Parte A. Qué son un Skill y un MCP

### La idea en una frase

- **Skill** = un *manual de instrucciones* que Claude lee cuando la tarea lo necesita. Enseña **cómo** hacer algo.
- **MCP** = un *programa externo* al que Claude se conecta para **hacer cosas** que él solo no puede (leer tu Drive, lanzar tus tests, sacar una captura de tu juego...). Añade **capacidades**.

Analogía: Claude es un cocinero muy bueno.
- El **Skill** es la receta de la casa y las normas de la cocina ("aquí el arroz se hace así, y no se usa tal sartén").
- El **MCP** es un electrodoméstico nuevo que le conectas (una freidora, una báscula). Con él puede hacer cosas que antes no podía.

### Qué es exactamente cada uno

**Skill.** Es una carpeta con un archivo `SKILL.md`:

```
.claude/skills/party-hard-dev/
  SKILL.md        <- instrucciones (con una cabecera: nombre y descripción)
  checklist.md    <- archivos de apoyo opcionales
```

La cabecera (`name` y `description`) es lo único que Claude ve siempre. Cuando tu petición encaja con la descripción, carga el resto y lo sigue. Es texto, no ejecuta nada por sí mismo. Ventaja: no gasta contexto hasta que hace falta.

**MCP (Model Context Protocol).** Es un estándar abierto. Un *servidor MCP* es un programa que Claude Code arranca y con el que habla por un canal sencillo (mensajes JSON). El servidor dice "tengo estas herramientas, con estos parámetros" y Claude las llama cuando las necesita. Sí ejecuta código real y puede tocar cosas de fuera, así que tiene permisos y pide aprobación.

### El flujo

```
 Tú ──pides──▶ Claude Code ──────────────┐
                │                         │
                │ (la tarea encaja con     │ (necesita hacer algo fuera:
                │  la descripción)         │  captura, tests, Drive...)
                ▼                         ▼
        Carga el SKILL.md          Llama a una herramienta del
        (instrucciones)            SERVIDOR MCP (programa aparte)
                │                         │
                ▼                         ▼
        Sabe CÓMO hacerlo          El servidor lo ejecuta y devuelve
        en este proyecto           el resultado (texto, ruta de imagen)
                └──────────┬──────────────┘
                           ▼
                  Claude responde con el resultado
```

### Cuándo usar cada uno

| Quieres... | Usa |
|---|---|
| Que siempre se siga el mismo procedimiento ("antes de entregar, corre los tests y no subas música") | **Skill** |
| Que Claude conozca las reglas y convenciones del proyecto | **Skill** |
| Que Claude pueda *hacer* algo que no puede (acceder a un servicio, lanzar un programa con límites, consultar una base de datos) | **MCP** |
| Repetir una tarea mecánica de forma segura y siempre igual | **MCP** (la herramienta) + **Skill** (cuándo usarla) |

### Lo que hemos creado en este proyecto

1. **Cinco skills** en `.claude/skills/`:
   - `party-hard-dev`: el conocimiento del proyecto para quien programa (cómo se añade un personaje famoso, un objeto o un nivel, cómo se traduce, qué banderas de depuración hay, cómo hacer capturas sin dejar ventanas abiertas, las normas: no commit sin pedirlo, no tocar `progress.json`, no subir música ni `imports/`), más `checklist.md`.
   - `party-hard-orquestador`: el papel del asistente principal (hablar contigo y dirigir agentes).
   - `party-hard-qa`: cómo se revisan las pruebas y se da el veredicto.
   - `party-hard-release`: cómo se compila, se empaqueta y se verifican los zips.
   - `party-hard-limpieza`: qué se puede borrar (y qué nunca).
2. **Servidor MCP `party-hard`** (`tools/phmcp/`, registrado en `.mcp.json`). Diez herramientas, cada una pensada para un papel:

| Herramienta | Papel | Qué hace |
|---|---|---|
| `list_levels` | cualquiera | Lista los 21 niveles de la campaña y los extras |
| `list_characters` | cualquiera | Lista los personajes del código y las carpetas de sprites |
| `check_phrases` | implementación / QA | Encuentra frases de bocadillo de más de 48 caracteres |
| `asset_report` | cualquiera | Cuenta personajes, objetos y fondos de `assets` e `imports` |
| `capture_level` | implementación / QA | Compila en una carpeta temporal y saca una captura de un nivel (máx. 60 s, sin dejar procesos) |
| `run_tests` | implementación | `go vet` + `go test` con límite de tiempo y resumen corto |
| `qa_run_tests` | **QA** | `go vet` + `go test -json`: totales, fallos con su salida, tests **saltados**, lentos, y aviso si no se ejecutó ninguno |
| `build_release` | **Build** | `gofmt`, `go vet`, `build.sh` y verificación de los zips (sin partida guardada, sin `imports/`, con juego y editor) y de si los binarios de la raíz están viejos |
| `clean_temp` | **Limpieza** | Lista (por defecto) o borra SOLO la lista permitida: temporales `ph_*`/`phmcp-*`, procesos de prueba colgados y, si se pide, la caché de Go |
| `disk_report` | Limpieza / Build | Espacio libre y tamaño de lo que crece (caché de Go, temporales, `dist/`, zips, `imports/`, música) |

Límites y seguridad:
- Ninguna herramienta edita el código del juego ni toca `assets/`, `imports/`, `intros/`, `music/` ni `assets/progress.json`.
- `build_release` solo escribe lo que ya escribe `build.sh` (`dist/` y los dos zips). **No copia** los binarios a la raíz: eso lo hace el agente de Build con `cp`.
- `clean_temp` es la única que puede borrar, y por eso es la más estricta: por defecto solo lista; para borrar hay que pasar `dry_run=false` **y** `confirm="DELETE"`; solo toca nombres `ph_*` o `phmcp-*` dentro de `/tmp` y la carpeta temporal del sistema, procesos cuyo ejecutable es `/tmp/ph_*`, `phmcp-*/game` o `partyhard.test` (nunca tu juego), y los enlaces se borran como enlace sin seguirlos. Lo que no está en esa lista (por ejemplo, `dist/` o carpetas de otros proyectos) lo decide la persona.
- Las herramientas que arrancan procesos usan tiempo máximo y matan todo el grupo de procesos. Solo funciona en macOS y Linux y solo usa la librería estándar de Go.

---

## Parte B. Cómo probar lo que está creado

### El skill

1. Abre Claude Code en la carpeta del proyecto.
2. Pregunta: *"¿Qué skills tienes disponibles en este proyecto?"* (o usa el comando `/skills` si tu versión lo trae). Deben aparecer `party-hard-dev`, `party-hard-orquestador`, `party-hard-qa`, `party-hard-release` y `party-hard-limpieza`.
3. Pide algo que encaje, sin nombrarlo: *"Añade el personaje Gandalf el Gris como famoso"*. Claude debería cargar el skill y seguir sus pasos (frases de máximo 48 caracteres, traducción, tests...). También puedes pedirlo expresamente: *"Usa el skill party-hard-dev y dime los pasos para añadir un objeto"*.
4. Si no lo carga, comprueba que la ruta es exactamente `.claude/skills/party-hard-dev/SKILL.md` y que la cabecera tiene `name` y `description`.

### El servidor MCP

1. El archivo `.mcp.json` de la raíz ya lo declara. La primera vez, Claude Code te preguntará si apruebas los servidores del proyecto: es una medida de seguridad, **tú decides**. Léelo antes de aprobar (el código está en `tools/phmcp/main.go`, es corto).
2. Mira el estado con el comando `/mcp` (o `claude mcp list` en la terminal). `party-hard` debe salir como conectado. Hace falta Go instalado: el servidor se arranca con `go run`.
3. Prueba algo barato: *"Usa la herramienta list_levels del servidor party-hard"* o *"¿Cuántos personajes hay en assets?"* (usará `asset_report`).
4. Algo más serio: *"Saca una captura del nivel 12 y dime si los bowls se ven bien"* (usa `capture_level`, tarda un poco la primera vez porque compila).
5. Las herramientas de papel, sin riesgo: *"Usa disk_report"* (solo mide), *"Usa clean_temp"* (solo lista: no borra nada porque `dry_run` es verdadero por defecto) y *"Usa build_release con verify_only"* (no compila: solo revisa los zips que ya hay).
6. Si falla: ejecuta a mano `cd tools/phmcp && go test .` (debe pasar; no compila el juego ni borra nada real) y revisa que `PH_ROOT` apunta a la raíz del proyecto.

Para quitarlo: borra el bloque `party-hard` de `.mcp.json` o elige rechazarlo cuando Claude Code lo pregunte.

---

## Parte C. Cómo dirigir a Claude

### 1. Una buena petición tiene cinco partes

1. **Objetivo**: qué quieres, en una frase. *"En el skate park las rampas deben ser bowls."*
2. **Contexto**: dónde pasa y por qué. *"Ahora se ven como picos."*
3. **Criterio de hecho**: cómo sabremos que está bien. *"Un niño hace trucos dentro; nadie se atasca; los tests pasan."*
4. **Ejemplos o referencias**: una imagen, un nombre de archivo, otro personaje parecido.
5. **Límites**: qué no tocar. *"No toques progress.json. No hagas commit."*

Ejemplo malo: *"arregla lo de los objetos"*. Ejemplo bueno: *"Hay armarios y vallas dibujados uno encima de otro en varios niveles. Quiero que ningún objeto pise a otro en ningún nivel y un test que falle si pasa."*

### 2. Cuándo pedir un plan

Pide un plan antes de tocar código cuando el cambio es grande, toca muchas partes o hay varias formas de hacerlo (un final de juego, un sistema de ajustes, un editor nuevo). Di: *"Antes de programar, dime el plan y qué decisiones tengo que tomar yo."* Para cambios pequeños y claros no hace falta.

### 3. Dividir en tareas y usar agentes en paralelo

Pide agentes cuando hay trabajo **independiente**: *"haz agentes"*, *"reparte las tareas"*. Funciona bien cuando cada agente tiene:
- una tarea clara y un criterio de hecho,
- **sus propios archivos** (uno el editor, otro los bocadillos, otro el audio),
- la orden de informar al final qué verificó y qué no.

**Qué NO repartir:**
- Tareas que tocan la misma función o el mismo archivo grande a la vez.
- Tareas que dependen de otra (primero el tamaño de los personajes, luego las colisiones).
- Decisiones de diseño que tienes que tomar tú.

Cómo se evita que se pisen: cada agente edita solo lo suyo; en archivos compartidos hace cambios mínimos releyendo antes; los errores de compilación de otro agente son transitorios (esperar y reintentar). Al final, un agente de Build integra: compila, pasa todos los tests y reconstruye (ver la Parte D: ahora el asistente principal ya no lo hace él mismo).

### 4. Dar feedback con capturas

Lo que mejor funciona es una **captura con el problema rodeado en rojo** y una frase: *"estos triángulos raros delante de la cara"*. Añade siempre **dónde** (qué nivel o pantalla), **cuándo** (después de qué acción) y **qué esperabas**. Si es una escena animada, di aproximadamente en qué momento. Una captura vale más que tres párrafos.

### 5. Pedir verificación

Claude no puede jugar con el ratón ni oír el audio. Por eso conviene pedir cosas comprobables:
- **Tests** ("añade un test que falle si vuelve a pasar").
- **Capturas** ("compruébalo con una captura y dime qué ves").
- **Honestidad**: "dime qué no pudiste verificar". Un buen informe distingue lo comprobado de lo supuesto.

Y cuando algo no te cuadra, pregunta: *"¿Esto lo has visto en pantalla o solo en los tests?"*

### 6. Plantillas de prompts reutilizables

**Nueva pantalla o escena**
> Quiero [pantalla/escena] que [qué muestra]. Estilo: como [referencia]. Debe poder saltarse con Enter, verse una sola vez y revisarse desde el menú. Comprueba con capturas de varios fotogramas y añade tests. No toques progress.json.

**Nuevo personaje**
> Añade [nombre] como personaje especial: aspecto [descripción], 6 frases cortas (máx. 48 caracteres), tienda y traducción al inglés. Solo puede salir una vez por nivel. Usa el skill del proyecto.

**Bug con captura**
> [Captura adjunta] En [pantalla/nivel], [qué pasa]. Esperaba [qué debería pasar]. Averigua la causa real antes de arreglarlo, dime cuál es, arréglalo en origen y añade un test para que no vuelva.

**Refactor**
> Quiero simplificar [parte] sin cambiar el comportamiento. Antes de empezar, dime qué tests cubren esa parte. Cambios pequeños, tests tras cada uno, y un resumen al final.

**Release (entrega)**
> Prepara una versión: gofmt, go vet, go test sin caché, build de Mac y Windows, binarios copiados a la raíz, zips sin mi partida guardada. Sin commit. Dime qué has comprobado y qué no.

### 7. Errores típicos (y cómo evitarlos)

| Error | Mejor |
|---|---|
| Peticiones vagas ("mejóralo") | Di qué ves mal y qué esperas |
| Cambiar de opinión sin avisar a mitad | Dilo claro: "esto sustituye a lo anterior" |
| Aceptar "ya está" sin pruebas | Pide capturas y tests, y "qué no verificaste" |
| Muchas peticiones mezcladas | Numéralas y pide que confirme cada una |
| Repartir trabajo que toca los mismos archivos | Agentes con archivos distintos; integra uno solo |
| Dejar cosas implícitas ("ya sabes") | Ponlo en el skill: lo que siempre debe cumplirse |
| Olvidar datos personales en un repo público | Pide revisar el repositorio antes de publicar |
| Pruebas que dejan procesos abiertos o llenan el disco | Pide limpiar procesos y archivos temporales; vacía la caché de Go de vez en cuando |

### 8. Un truco final

Cuando repitas la misma instrucción más de dos veces ("no hagas commit", "traduce al inglés", "pasa los tests"), **ponla en el skill del proyecto**. Así dejas de repetirla y Claude la sigue siempre.

---

## Parte D. Modo orquestador: cómo trabajamos ahora

### La idea en una frase

**Tú hablas con el asistente principal; él no trabaja: reparte el trabajo en agentes, les exige pruebas y te cuenta con honestidad qué se comprobó.** Es como un jefe de obra: no coge el martillo, pero sabe quién lo coge y revisa que la pared esté recta.

### Quién hace qué

| Papel | Qué hace | Qué NO hace |
|---|---|---|
| **Orquestador** (el asistente con el que hablas) | Entiende lo que quieres, lo divide en tareas, lanza agentes, recoge sus informes, **exige evidencias**, te resume y te dice qué no se pudo comprobar | No programa, no compila, no ejecuta tests ni capturas, no borra nada |
| **Agentes de implementación** | Uno por área (editor, audio, 2 jugadores...): cambian el código, escriben tests, hacen capturas | No compilan la versión final ni limpian |
| **QA** | Ejecuta él mismo los tests y capturas, comprueba que los tests prueban de verdad lo que dicen, contrasta los informes con el código, y da el **veredicto** | No programa: si algo falla, lo devuelve |
| **Build / Release** | `gofmt`, `go vet`, todos los tests, `build.sh`, copia los programas a la raíz, comprueba los zips | No toca tu música, tus importaciones ni tu partida |
| **Limpieza** | Cierra procesos de prueba colgados, borra temporales y la caché de Go, mide el disco antes y después | Solo borra una lista permitida; nunca tus datos |

### El orden de una entrega

```
tú pides  ->  implementación  ->  QA (veredicto)  ->  Build  ->  Limpieza  ->  te informa el orquestador
```

Reglas del orden:
- QA y Build empiezan **cuando todos los agentes de implementación han terminado** (mientras alguien edita, hay errores de compilación que no significan nada).
- Los zips solo se dan por buenos si QA dice **PASA** o **PASA CON RESERVAS** y tú aceptas las reservas.
- La limpieza va al final y siempre mide el disco antes y después.

### Cómo pedir cosas

Habla como siempre. Con una frase es suficiente: *"quiero que el pirómano también salga en el cementerio"*, *"el coche explota raro, mira la captura"*, *"prepara los zips"*, *"limpia el disco"*. El orquestador decide los agentes. Ayuda mucho:
- decir **qué esperas ver** ("que la carta no se cierre sola");
- dar una **captura** si algo se ve mal;
- decir **qué no debe cambiar** ("con 1 jugador todo igual").

Cosas que puedes pedirle al orquestador: ver el estado ("¿cómo va?"), que **te enseñe las evidencias de QA**, que vuelva a probar algo, que explique cómo funciona una parte (lo averigua un agente), que cambie un criterio, o que pare y te pregunte antes de seguir.

### Cómo leer el veredicto de QA

QA da un veredicto **por tarea**:

| Veredicto | Significa | Qué haces tú |
|---|---|---|
| **PASA** | Lo comprobó todo y no hay fallos | Nada: se sigue con el build |
| **PASA CON RESERVAS** | Funciona, pero algo es débil, no se probó o es discutible | Lees las reservas y decides si las aceptas |
| **NO PASA** | Un test falla, falta algo de lo pedido, se rompió el juego de 1 jugador o el informe no coincide con la realidad | El orquestador devuelve la tarea al agente responsable y se repite QA |

Cada veredicto trae: las **evidencias** (comandos y resultados reales, tests por nombre, capturas leídas), las **reservas**, los **fallos** y una lista de **lo que no se puede comprobar sin una persona** (el audio, el mando, jugar de verdad con dos personas, si "queda bonito"). Esa última lista es para ti: es lo que te toca probar.

Señales de que hay que desconfiar de un informe: dice "pasan los tests" sin enseñar la salida; un test se ha **saltado** (`SKIP`) sin motivo; un test existente se ha **aflojado**; nadie ha mirado las capturas; el filtro de tests no ejecutó nada.

### Plantilla de encargo (la que usa el orquestador con cada agente)

```
OBJETIVO: (una frase)
CONTEXTO: (lo que pediste con tus palabras, lo que ya está hecho, lo que no se debe romper)
ARCHIVOS PROPIOS: (lista; los compartidos se tocan con cambios mínimos)
REGLAS: no commit, no tocar assets/progress.json, procesos con vigilante, otros agentes y sus archivos
CRITERIO DE HECHO: (tests y capturas concretas; con 1 jugador igual)
INFORME ESPERADO: QUÉ HICE / EVIDENCIAS / NO VERIFICADO / AVISOS
```

### Plantilla de informe (la que se exige a cada agente)

```
QUÉ HICE: (pocas líneas, con archivos)
EVIDENCIAS: go vet -> sin salida; go test -run TestX -> "ok partyhard 3.2s"; captura /tmp/.../a.png: se ve ...
NO VERIFICADO: (audio, mando, dos personas...)
AVISOS: (algo visto fuera de mi tarea, sin tocarlo)
```

Si un informe no trae evidencias, el orquestador lo devuelve: *"dime el comando exacto y su salida"*.

### Un ejemplo real, paso a paso

1. Pides: *"el coche que derrapa tiene que explotar al final"*.
2. El orquestador lanza un agente de implementación con el encargo (objetivo, archivos, criterio: "el coche y su movimiento no cambian; solo se añade la explosión").
3. El agente cambia el código, escribe un test que compara la trayectoria original con la nueva y hace capturas. Informa con evidencias.
4. El orquestador lanza **QA**, que ejecuta los tests y las capturas por su cuenta, lee el test y comprueba en el `git diff` que no se tocó nada más. Veredicto: *PASA CON RESERVAS: no se pudo comprobar el sonido de la explosión*.
5. Te lo cuenta el orquestador. Aceptas la reserva (el sonido lo oyes tú).
6. Lanza **Build** (tests completos, `build.sh`, zips verificados) y luego **Limpieza** (disco antes y después).
7. Recibes: qué cambió, qué verificó QA, qué te toca probar a ti y dónde están los zips.

### Dónde está cada cosa

- Los papeles están escritos en los skills (`.claude/skills/party-hard-orquestador`, `-qa`, `-release`, `-limpieza`): son las instrucciones que leen los agentes.
- Las herramientas del servidor MCP (`qa_run_tests`, `build_release`, `clean_temp`, `disk_report`...) las usan los agentes de cada papel, con las salvaguardas de la Parte A.
- Si quieres cambiar cómo se trabaja ("que QA también revise la traducción inglesa", "que Build no copie a la raíz"), dilo: se cambia el skill correspondiente y todos los agentes lo siguen.

---

## Auditoría automática de textos (QA)

Cada vez que cambia algo visual, un texto o una traducción, el QA ejecuta `go test -count=1 -run TestUITextAudit .` (archivo `uitext_audit_test.go`; `-short` audita solo el inglés).

- **Cómo funciona**: sin ventana, el test dibuja unas 90 pantallas (menú con 1 y 2 jugadores y DEBUG, título, elegir fiesta, ajustes, créditos, tienda, vestuario, trofeos, HUD, bocadillos, fin de nivel, escenas, carta) en español e inglés. Un registrador (`uitext_rec.go`) anota rectángulo, texto y escala de cada llamada de texto.
- **Qué comprueba**: que no se solapen, que no se salgan de la pantalla 960x540 ni crucen el borde de un panel, que escala y posición sean enteras (si no, el texto sale borroso), que no se dibuje en un buffer que luego se reescale y que la versión inglesa no tenga español.
- **Si falla**: imprime un informe por pantalla con textos y coordenadas. Se arregla el diseño; solo se añade una excepción (`auditAllow`) con razón escrita. Lo conocido y sin arreglar va a `auditPending` y se avisa.
- **No sustituye al ojo**: el QA también mira capturas ampliadas (`-shotzoom`). Memoria del QA: `docs/QA/MEMORIA.md`. Subagente: `.claude/agents/party-hard-qa.md` (Claude Code lo carga al iniciar una sesión nueva).

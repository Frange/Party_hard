# Memoria del QA de Party Hard

Archivo para el rol QA (ver `.claude/skills/party-hard-qa/SKILL.md`). Sin datos personales. Se actualiza al cerrar cada revisión.

Lecciones del equipo (checklist que este rol aplica): `docs/EQUIPO/LECCIONES.md`.

## Criterios fijos (siempre)
1. `gofmt -l` vacío (de los archivos tocados), `go vet .` limpio, tests pedidos en verde y sin SKIPPED injustificados.
2. Cambio visual, de textos o de traducciones: `go test -count=1 -run TestUITextAudit .` en verde (ES y EN). Con prisa, `-short` solo audita inglés, pero el veredicto final usa la versión completa.
3. Capturas ampliadas (`-shotzoom x,y,w,h,2`) de cada pantalla tocada: leerlas y comprobar nitidez, solapes, texto cortado, texto en español en la versión inglesa.
4. Ningún texto borroso: escala entera, posición entera, texto dibujado directo en la pantalla (no en un buffer que se reescala).
5. Reglas del usuario: `assets/progress.json` intacto, nada nuevo en `imports/`, sin música ni datos personales, frases de bocadillo <= 48 caracteres.
6. Un test que no prueba lo pedido, o que se aflojó para pasar, es un hallazgo.

## Qué revisar siempre
- Menú principal con 10 filas (DEBUG) y 2 jugadores; ELIGE LA FIESTA con Isla Nublada; Ajustes (todas las filas y créditos); tienda (todas las pestañas) y vestuario; trofeos; HUD con 1 y 2 jugadores; bocadillos; pantallas de fin de nivel.
- Inglés: los textos son más largos; mirar anchos y que no haya mezcla ES/EN.

## Fallos recurrentes conocidos
- Texto envuelto (`drawWrapped`) y luego traducido línea a línea queda a medias: traducir el texto entero antes de envolver.
- Etiqueta con prefijo (`"  "+label`, `label+" ◄"`) no se traduce porque la tabla busca la cadena exacta: traducir con `tr()` antes de añadir el prefijo, o añadir la cadena exacta.
- Pantallas superpuestas (Ajustes bajo Créditos): lo de debajo no debe dibujarse a medias.
- Panel de detalle demasiado bajo para 3 líneas de descripción.
- Menú con muchas filas (DEBUG): la última fila choca con los textos del pie.

## Cómo registrar hallazgos nuevos
Añade una línea al final de "Registro" con fecha (AAAA-MM-DD), pantalla, qué pasó, cómo se arregló o quién lo tiene. Si el fallo es de una clase nueva, súbelo a "Fallos recurrentes". Una excepción de la auditoría (`auditAllow`, `auditOutside`, `auditSpanishOK`) necesita siempre una razón escrita; lo no arreglado va a `auditPending` y se avisa al orquestador.

## Registro
- 2026-11-04: se crea la auditoría `TestUITextAudit`. Corregidos: descripciones de la tienda a medias en inglés, etiquetas del vestuario sin traducir, "SALIR ◄" sin traducir, pie de Ajustes asomando bajo Créditos, panel de detalle de la tienda corto. Pendiente (en `auditPending`): última fila del menú con DEBUG pisa el pie izquierdo (`menuRowY` en debugmode.go).

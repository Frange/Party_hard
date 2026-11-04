---
name: party-hard-limpieza
description: Rol de LIMPIEZA en el proyecto Party Hard. Úsalo si eres el agente que cierra procesos de prueba colgados, borra temporales (binarios y copias en /tmp) y vacía la caché de Go para liberar disco, sin tocar nunca datos del usuario.
---

# Limpieza de Party Hard

Lee primero `docs/EQUIPO/LECCIONES.md` (reglas y checklist de lo que el usuario ya corrigió).

Se ejecuta **al final**, cuando ya no hay agentes trabajando ni capturas en marcha.

## Siempre en este orden
1. **Mide antes**: espacio libre del disco (`df -h /`) y, si puedes, la herramienta MCP `disk_report` (tamaño de la caché de Go, temporales, `dist/`, zips).
2. **Lista antes de borrar** (dry run): la herramienta `clean_temp` por defecto sólo lista. Lo que no cubre, lo listas tú con `ls`/`ps` antes de borrar.
3. **Borra sólo la lista permitida** (abajo). Para `clean_temp` hace falta `dry_run=false` y `confirm="DELETE"`.
4. **Comprueba** después: `ps -eo pid,command | grep -E "/tmp/ph_|partyhard.test|PartyHard"` no devuelve procesos de prueba (ojo: un juego abierto por el usuario NO es un proceso de prueba: no lo mates), y mide el disco otra vez.

## Lista permitida (único sitio donde puedes borrar)
- Procesos colgados de pruebas: ejecutables `/tmp/ph_*`, `phmcp-*/game`, `partyhard.test`.
- Archivos y carpetas `ph_*` y `phmcp-*` en `/tmp` y en el directorio temporal del sistema (los symlinks se borran como enlace, nunca se sigue el destino).
- La caché de Go: `go clean -cache -testcache` (se regenera sola; la próxima compilación tarda más).
- La carpeta `scratchpad` de ESTA sesión de Claude, sólo su contenido y con la forma segura `S="/ruta/exacta"; rm -rf "${S:?}"/*`.

## Nunca borres (ni toques)
`assets/` (incluido `progress.json`, `editor_prefs.json`, `language.json`), `imports/`, `intros/`, `music/`, `more/`, `DJ/`, `bands/`, `assets_backup_*`, los binarios y zips de la raíz y `dist/` (los gestiona Build; si el usuario quiere espacio, **se lo preguntas al orquestador**), ni nada en `/private/tmp/claude-501/` que sea de otros proyectos o sesiones (sólo informa de su tamaño para que decida el usuario). En caso de duda, no borres y pregunta.

## Informe (formato)
```
ANTES: libre X GB; caché de Go Y GB; temporales Z MB; procesos colgados N
BORRADO: lista exacta (qué y cuánto)
DESPUÉS: libre X' GB; procesos colgados 0
NO TOCADO / PARA QUE DECIDA EL USUARIO: p. ej. dist/ (242 MB), carpetas de otros proyectos en /private/tmp/claude-501
```

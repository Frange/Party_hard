# Checklist del QA (corta)

Lee también `docs/QA/MEMORIA.md` (criterios fijos y fallos recurrentes).

- [ ] Informe contrastado con `git status` / `git diff --stat`
- [ ] `gofmt -l` (archivos tocados) vacío; `go vet .` limpio
- [ ] Tests nuevos y antiguos en verde; sin SKIPPED injustificados; ningún test debilitado
- [ ] Si cambió algo visual, de textos o de traducciones: `go test -count=1 -run TestUITextAudit .` en verde (sin `-short` para el veredicto)
- [ ] Capturas ampliadas con `-shotzoom` de cada pantalla tocada, leídas: nítidas, sin solapes, nada tapado, nada cortado, nada en español en inglés
- [ ] `go test -count=1 -run 'Text|Lang' .` en verde
- [ ] Modo de 1 jugador sin cambios; 2 jugadores comprobado si se tocó
- [ ] `assets/progress.json`, `music/`, `imports/`, `intros/` sin tocar; sin datos personales
- [ ] Procesos de prueba muertos y temporales borrados
- [ ] Hallazgos nuevos anotados (fecha) en `docs/QA/MEMORIA.md`
- [ ] NO PASA si la auditoría falla o hay texto borroso/solapado

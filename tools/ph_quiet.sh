#!/bin/sh
# Lanza cualquier comando en modo silencioso y con prioridad baja: tools/ph_quiet.sh ./partyhard -level 1 -shot a.png
# (PH_QUIET=1 = ventana fuera de pantalla (nunca minimizada) sin robar foco; nice -n 19 = el PC sigue usable)
PH_QUIET=1 GOMAXPROCS="${GOMAXPROCS:-4}" exec nice -n 19 "$@"

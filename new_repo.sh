#!/usr/bin/env bash
# git-subir.sh — Inicializa/configura Git y sube el contenido de la carpeta actual.
set -Eeuo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RESET='\033[0m'

fail() { printf "${RED}Error: %s${RESET}\n" "$*" >&2; exit 1; }
info() { printf "${GREEN}%s${RESET}\n" "$*"; }
warn() { printf "${YELLOW}%s${RESET}\n" "$*"; }

command -v git >/dev/null 2>&1 || fail "Git no está instalado. Instálalo antes de continuar."

printf "Carpeta: %s\n\n" "$PWD"
read -r -p "URL del repositorio (GitHub/GitLab/etc.): " REPO_URL
[[ -n "${REPO_URL//[[:space:]]/}" ]] || fail "No has indicado una URL."

# Validación básica para evitar guardar por accidente una URL mal formada.
if [[ "$REPO_URL" != http://* && "$REPO_URL" != https://* && "$REPO_URL" != git@* && "$REPO_URL" != ssh://* ]]; then
  fail "La URL debe empezar por https://, http://, git@ o ssh://."
fi

if [[ ! -d .git ]]; then
  git init
  info "Repositorio Git inicializado."
else
  info "Ya existe un repositorio Git en esta carpeta."
fi

# Configura o actualiza origin.
if git remote get-url origin >/dev/null 2>&1; then
  git remote set-url origin "$REPO_URL"
  info "Remoto 'origin' actualizado."
else
  git remote add origin "$REPO_URL"
  info "Remoto 'origin' añadido."
fi

# Mantiene la rama actual si ya hay una; si no, usa main.
BRANCH="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)"
if [[ -z "$BRANCH" ]]; then
  BRANCH="main"
  git checkout -B "$BRANCH" 2>/dev/null || git switch -C "$BRANCH"
fi

# Añade todo el contenido de la carpeta, respetando .gitignore.
git add --all

if git diff --cached --quiet; then
  warn "No hay cambios nuevos para guardar."
else
  read -r -p "Mensaje del commit [Actualización del proyecto]: " COMMIT_MSG
  COMMIT_MSG="${COMMIT_MSG:-Actualización del proyecto}"
  git commit -m "$COMMIT_MSG"
fi

info "Subiendo la rama '$BRANCH'..."
git push -u origin "$BRANCH"

info "¡Proceso terminado! Repositorio sincronizado correctamente."


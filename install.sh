#!/usr/bin/env bash
# Maint CLI - instalador.
# Instala o wrapper "maint" no PATH. A imagem docker é puxada
# automaticamente pelo wrapper na primeira execução (Docker Hub).
set -euo pipefail

# ----- configuração ---------------------------------------------------------
MAINT_REPO="${MAINT_REPO:-rlivdev/maint-cli}"
MAINT_BRANCH="${MAINT_BRANCH:-main}"
MAINT_URL="${MAINT_URL:-https://raw.githubusercontent.com/$MAINT_REPO/$MAINT_BRANCH/maint}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
MAINT_HOME="${MAINT_HOME:-$HOME/.maint}"

FMT_BOLD='\033[1m'
FMT_GREEN='\033[0;32m'
FMT_RED='\033[0;31m'
FMT_RESET='\033[0m'

info()  { printf "${FMT_GREEN}${FMT_BOLD}%s${FMT_RESET}\n" "$*"; }
warn()  { printf "${FMT_RED}${FMT_BOLD}%s${FMT_RESET}\n" "$*" >&2; }

# ----- pré-requisitos --------------------------------------------------------
if ! command -v docker >/dev/null 2>&1; then
  warn "erro: docker não encontrado. Instale primeiro: https://docs.docker.com/engine/install/"
  exit 1
fi

# ----- obtenção do wrapper ---------------------------------------------------
# Perfere wrapper local (clone do repo), mas funciona via curl|bash baixando
# o wrapper do repositório.
SRC="${MAINT_SOURCE:-}"
if [ -z "$SRC" ]; then
  SRC="$(dirname "$(readlink -f "${0}" 2>/dev/null || echo "$0")")/maint"
fi

if [ ! -f "$SRC" ]; then
  if command -v curl >/dev/null 2>&1; then
    info "wrapper não encontrado. Baixando de $MAINT_URL"
    TMP="$(mktemp -d)"
    curl -fsSL "$MAINT_URL" -o "$TMP/maint" || { warn "erro: falha ao baixar wrapper"; exit 1; }
    SRC="$TMP/maint"
  elif command -v wget >/dev/null 2>&1; then
    info "wrapper não encontrado. Baixando de $MAINT_URL"
    TMP="$(mktemp -d)"
    wget -qO "$TMP/maint" "$MAINT_URL" || { warn "erro: falha ao baixar wrapper"; exit 1; }
    SRC="$TMP/maint"
  else
    warn "erro: wrapper local ausente e sem curl/wget para baixá-lo"
    exit 1
  fi
fi

# ----- instalação ------------------------------------------------------------
mkdir -p "$INSTALL_DIR"
install -m 755 "$SRC" "$INSTALL_DIR/maint"
mkdir -p "$MAINT_HOME/profiles" "$MAINT_HOME/backups"

# verifica PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) : ;;
  *) warn "aviso: $INSTALL_DIR não está no PATH. Adicione-o:" ;;
esac

# ----- resultado -------------------------------------------------------------
info "Maint CLI instalado em $INSTALL_DIR/maint"
info "Diretório de dados: $MAINT_HOME"
info "Teste: maint --help"

if [[ $- == *i* ]] || [ -t 1 ]; then
  if ! grep -q "'$INSTALL_DIR'" "$HOME/.bashrc" 2>/dev/null \
     && ! grep -q ":\\\$HOME/.local/bin" "$HOME/.bashrc" 2>/dev/null; then
    printf "\n%s\n" "Adicione ao seu shell (opcional):"
    printf "  echo 'export PATH=\"%s:\$PATH\"' >> ~/.bashrc\n" "$INSTALL_DIR"
  fi
fi

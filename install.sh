#!/usr/bin/env bash
# Brekke CLI - instalador.
# Instala o wrapper "brekke" no PATH. A imagem docker é puxada
# automaticamente pelo wrapper na primeira execução (Docker Hub).
set -euo pipefail

# ----- configuração ---------------------------------------------------------
BREKKE_REPO="${BREKKE_REPO:-brekke-cloud/brekke-cli}"
BREKKE_BRANCH="${BREKKE_BRANCH:-main}"
BREKKE_URL="${BREKKE_URL:-https://raw.githubusercontent.com/$BREKKE_REPO/$BREKKE_BRANCH/brekke}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BREKKE_HOME="${BREKKE_HOME:-$HOME/.brekke}"

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
SRC="${BREKKE_SOURCE:-}"
if [ -z "$SRC" ]; then
  SRC="$(dirname "$(readlink -f "${0}" 2>/dev/null || echo "$0")")/brekke"
fi

if [ ! -f "$SRC" ]; then
  if command -v curl >/dev/null 2>&1; then
    info "wrapper não encontrado. Baixando de $BREKKE_URL"
    TMP="$(mktemp -d)"
    curl -fsSL "$BREKKE_URL" -o "$TMP/brekke" || { warn "erro: falha ao baixar wrapper"; exit 1; }
    SRC="$TMP/brekke"
  elif command -v wget >/dev/null 2>&1; then
    info "wrapper não encontrado. Baixando de $BREKKE_URL"
    TMP="$(mktemp -d)"
    wget -qO "$TMP/brekke" "$BREKKE_URL" || { warn "erro: falha ao baixar wrapper"; exit 1; }
    SRC="$TMP/brekke"
  else
    warn "erro: wrapper local ausente e sem curl/wget para baixá-lo"
    exit 1
  fi
fi

# ----- instalação ------------------------------------------------------------
mkdir -p "$INSTALL_DIR"
install -m 755 "$SRC" "$INSTALL_DIR/brekke"
mkdir -p "$BREKKE_HOME/profiles" "$BREKKE_HOME/backups"

# verifica PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) : ;;
  *) warn "aviso: $INSTALL_DIR não está no PATH. Adicione-o:" ;;
esac

# ----- resultado -------------------------------------------------------------
info "Brekke CLI instalado em $INSTALL_DIR/brekke"
info "Diretório de dados: $BREKKE_HOME"
info "Teste: brekke --help"

if [[ $- == *i* ]] || [ -t 1 ]; then
  if ! grep -q "'$INSTALL_DIR'" "$HOME/.bashrc" 2>/dev/null \
     && ! grep -q ":\\\$HOME/.local/bin" "$HOME/.bashrc" 2>/dev/null; then
    printf "\n%s\n" "Adicione ao seu shell (opcional):"
    printf "  echo 'export PATH=\"%s:\$PATH\"' >> ~/.bashrc\n" "$INSTALL_DIR"
  fi
fi
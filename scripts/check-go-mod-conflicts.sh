#!/bin/bash

# check-go-mod-conflicts.sh - Verificar conflictos en go.mod y go.sum
# Usado como pre-commit hook

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Verificar que estamos en un repositorio git
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    error "No estás en un repositorio git"
    exit 1
fi

# Verificar conflictos en go.mod
if [ -f "go.mod" ]; then
    if grep -q "<<<<<<< HEAD\|=======\|>>>>>>> " go.mod; then
        error "❌ Conflictos de merge detectados en go.mod"
        echo "Marcadores de conflicto encontrados:"
        grep -n "<<<<<<< HEAD\|=======\|>>>>>>> " go.mod
        exit 1
    fi
fi

# Verificar conflictos en go.sum
if [ -f "go.sum" ]; then
    if grep -q "<<<<<<< HEAD\|=======\|>>>>>>> " go.sum; then
        error "❌ Conflictos de merge detectados en go.sum"
        echo "Marcadores de conflicto encontrados:"
        grep -n "<<<<<<< HEAD\|=======\|>>>>>>> " go.sum
        exit 1
    fi
fi

# Verificar que go.mod es válido
if [ -f "go.mod" ]; then
    if ! go mod verify >/dev/null 2>&1; then
        error "❌ go.mod no es válido"
        go mod verify
        exit 1
    fi
fi

# Verificar que las dependencias están sincronizadas
if [ -f "go.mod" ] && [ -f "go.sum" ]; then
    if ! go mod tidy -check >/dev/null 2>&1; then
        warning "⚠️ go.mod y go.sum no están sincronizados"
        echo "Ejecuta 'go mod tidy' para sincronizar"
    fi
fi

success "✅ No hay conflictos en archivos de dependencias Go"
exit 0

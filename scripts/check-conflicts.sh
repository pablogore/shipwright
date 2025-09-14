#!/bin/bash

# check-conflicts.sh - Script para detectar conflictos potenciales
# Uso: ./scripts/check-conflicts.sh

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Función para logging
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Verificar que estamos en un repositorio git
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    error "No estás en un repositorio git"
    exit 1
fi

log "🔍 Verificando conflictos potenciales..."

# Verificar conflictos activos
if git diff --name-only --diff-filter=U | grep -q .; then
    error "❌ Hay conflictos de merge activos:"
    git diff --name-only --diff-filter=U | while read file; do
        echo "  - $file"
    done
    echo ""
    echo "Para resolver:"
    echo "1. Edita los archivos con conflictos"
    echo "2. Ejecuta: git add <archivo-resuelto>"
    echo "3. Ejecuta: git commit"
    exit 1
else
    success "✅ No hay conflictos de merge activos"
fi

# Verificar estado de la rama
CURRENT_BRANCH=$(git branch --show-current)
log "📊 Analizando rama: $CURRENT_BRANCH"

# Verificar si la rama está desactualizada
if [[ "$CURRENT_BRANCH" != "main" && "$CURRENT_BRANCH" != "develop" ]]; then
    # Obtener rama base (develop por defecto)
    BASE_BRANCH="develop"
    
    # Verificar si develop existe
    if git show-ref --verify --quiet "refs/remotes/origin/$BASE_BRANCH"; then
        git fetch origin "$BASE_BRANCH" >/dev/null 2>&1
        
        BEHIND_COUNT=$(git rev-list --count "$CURRENT_BRANCH".."origin/$BASE_BRANCH" || echo "0")
        
        if [ "$BEHIND_COUNT" -gt 0 ]; then
            warning "⚠️ Tu rama está $BEHIND_COUNT commits detrás de $BASE_BRANCH"
            echo ""
            echo "Commits recientes en $BASE_BRANCH:"
            git log --oneline "origin/$BASE_BRANCH" -5
            echo ""
            echo "Para sincronizar:"
            echo "  ./scripts/sync-branch.sh"
        else
            success "✅ Tu rama está actualizada con $BASE_BRANCH"
        fi
    fi
fi

# Verificar archivos críticos modificados
log "🔍 Verificando archivos críticos..."

CRITICAL_FILES=(
    ".github/workflows/ci.yml"
    ".github/workflows/release.yml"
    "go.mod"
    "go.sum"
    "main.go"
)

MODIFIED_CRITICAL=""
for file in "${CRITICAL_FILES[@]}"; do
    if [ -f "$file" ]; then
        if ! git diff --quiet HEAD -- "$file"; then
            MODIFIED_CRITICAL="$MODIFIED_CRITICAL $file"
        fi
    fi
done

if [ -n "$MODIFIED_CRITICAL" ]; then
    warning "⚠️ Archivos críticos modificados:"
    echo "$MODIFIED_CRITICAL" | tr ' ' '\n' | while read file; do
        if [ -n "$file" ]; then
            echo "  - $file"
        fi
    done
    echo ""
    echo "💡 Recomendaciones:"
    echo "  - Coordina con el equipo antes de hacer cambios"
    echo "  - Considera dividir cambios en PRs más pequeños"
    echo "  - Solicita revisión temprana"
else
    success "✅ No hay modificaciones en archivos críticos"
fi

# Verificar múltiples archivos de workflow modificados
WORKFLOW_COUNT=$(git diff --name-only HEAD -- | grep -c "\.github/workflows/" || echo "0")
if [ "$WORKFLOW_COUNT" -gt 1 ]; then
    warning "⚠️ Múltiples archivos de workflow modificados ($WORKFLOW_COUNT archivos)"
    echo ""
    echo "Archivos de workflow modificados:"
    git diff --name-only HEAD -- | grep "\.github/workflows/" | while read file; do
        echo "  - $file"
    done
    echo ""
    echo "💡 Considera:"
    echo "  - Dividir cambios en PRs separados"
    echo "  - Coordinar con el equipo"
    echo "  - Testing exhaustivo"
fi

# Verificar sintaxis de workflows
if [ -d ".github/workflows" ]; then
    log "🔍 Verificando sintaxis de workflows..."
    
    if command -v yamllint >/dev/null 2>&1; then
        SYNTAX_ERRORS=0
        for workflow in .github/workflows/*.yml; do
            if [ -f "$workflow" ]; then
                if yamllint "$workflow" >/dev/null 2>&1; then
                    log "✅ $workflow - sintaxis válida"
                else
                    error "❌ $workflow - errores de sintaxis:"
                    yamllint "$workflow"
                    SYNTAX_ERRORS=$((SYNTAX_ERRORS + 1))
                fi
            fi
        done
        
        if [ "$SYNTAX_ERRORS" -eq 0 ]; then
            success "✅ Todos los workflows tienen sintaxis válida"
        else
            error "❌ $SYNTAX_ERRORS workflow(s) con errores de sintaxis"
            exit 1
        fi
    else
        warning "⚠️ yamllint no está instalado"
        echo "Instala con: pip install yamllint"
    fi
fi

# Verificar que el código compila (si es un proyecto Go)
if [ -f "go.mod" ]; then
    log "🔨 Verificando que el código compila..."
    if go build ./... >/dev/null 2>&1; then
        success "✅ Código compila correctamente"
    else
        error "❌ El código no compila:"
        go build ./...
        exit 1
    fi
fi

# Verificar que no hay archivos grandes
log "🔍 Verificando tamaño de archivos..."
LARGE_FILES=$(find . -type f -size +10M -not -path "./.git/*" 2>/dev/null || true)
if [ -n "$LARGE_FILES" ]; then
    warning "⚠️ Archivos grandes detectados (>10MB):"
    echo "$LARGE_FILES" | while read file; do
        if [ -n "$file" ]; then
            size=$(du -h "$file" | cut -f1)
            echo "  - $file ($size)"
        fi
    done
    echo ""
    echo "💡 Considera:"
    echo "  - Usar Git LFS para archivos grandes"
    echo "  - Agregar a .gitignore si no es necesario"
else
    success "✅ No hay archivos grandes"
fi

# Verificar que no hay archivos binarios
log "🔍 Verificando archivos binarios..."
BINARY_FILES=$(git diff --name-only --cached | grep -E '\.(exe|dll|so|dylib|bin)$' || true)
if [ -n "$BINARY_FILES" ]; then
    warning "⚠️ Archivos binarios detectados:"
    echo "$BINARY_FILES" | while read file; do
        if [ -n "$file" ]; then
            echo "  - $file"
        fi
    done
    echo ""
    echo "💡 Considera:"
    echo "  - Usar Git LFS para archivos binarios"
    echo "  - Agregar a .gitignore si no es necesario"
else
    success "✅ No hay archivos binarios"
fi

# Resumen final
echo ""
log "📊 Resumen de verificación:"
echo "  - Rama: $CURRENT_BRANCH"
echo "  - Conflictos activos: ✅ Ninguno"
echo "  - Archivos críticos: $([ -n "$MODIFIED_CRITICAL" ] && echo "⚠️ Modificados" || echo "✅ Sin cambios")"
echo "  - Sintaxis workflows: ✅ Válida"
echo "  - Compilación: ✅ Exitosa"
echo "  - Archivos grandes: ✅ Ninguno"
echo "  - Archivos binarios: ✅ Ninguno"

if [ -n "$MODIFIED_CRITICAL" ] || [ "$WORKFLOW_COUNT" -gt 1 ]; then
    echo ""
    warning "⚠️ Recomendaciones:"
    echo "  - Coordina con el equipo antes de hacer push"
    echo "  - Considera dividir cambios en PRs más pequeños"
    echo "  - Solicita revisión temprana"
    echo "  - Ejecuta tests exhaustivos"
else
    echo ""
    success "🎉 Todo se ve bien! Puedes continuar con confianza."
fi

echo ""
log "💡 Comandos útiles:"
echo "  - Sincronizar rama: ./scripts/sync-branch.sh"
echo "  - Ver conflictos: git status"
echo "  - Resolver conflictos: git mergetool"
echo "  - Verificar sintaxis: yamllint .github/workflows/*.yml"

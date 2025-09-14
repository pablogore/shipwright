#!/bin/bash

# sync-branch.sh - Script para sincronizar rama con develop y prevenir conflictos
# Uso: ./scripts/sync-branch.sh [rama-base]

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

# Obtener rama actual
CURRENT_BRANCH=$(git branch --show-current)
BASE_BRANCH=${1:-develop}

log "🔄 Sincronizando rama '$CURRENT_BRANCH' con '$BASE_BRANCH'"

# Verificar que no estamos en main o develop
if [[ "$CURRENT_BRANCH" == "main" || "$CURRENT_BRANCH" == "develop" ]]; then
    error "No puedes sincronizar desde las ramas principales (main/develop)"
    error "Crea una rama de feature/fix primero"
    exit 1
fi

# Verificar que no hay cambios sin commitear
if ! git diff-index --quiet HEAD --; then
    error "Tienes cambios sin commitear. Por favor, haz commit o stash primero:"
    git status --short
    exit 1
fi

# Verificar que no hay conflictos activos
if git diff --name-only --diff-filter=U | grep -q .; then
    error "Hay conflictos de merge activos. Resuélvelos primero:"
    git diff --name-only --diff-filter=U
    exit 1
fi

log "📥 Actualizando rama base '$BASE_BRANCH'..."
git fetch origin "$BASE_BRANCH"

# Verificar si la rama base existe
if ! git show-ref --verify --quiet "refs/remotes/origin/$BASE_BRANCH"; then
    error "La rama '$BASE_BRANCH' no existe en el remoto"
    exit 1
fi

# Verificar si hay commits nuevos en la rama base
BEHIND_COUNT=$(git rev-list --count "$CURRENT_BRANCH".."origin/$BASE_BRANCH" || echo "0")
AHEAD_COUNT=$(git rev-list --count "origin/$BASE_BRANCH".."$CURRENT_BRANCH" || echo "0")

log "📊 Estado de la rama:"
log "  - Commits detrás de $BASE_BRANCH: $BEHIND_COUNT"
log "  - Commits adelante de $BASE_BRANCH: $AHEAD_COUNT"

if [ "$BEHIND_COUNT" -eq 0 ]; then
    success "✅ Tu rama ya está actualizada con $BASE_BRANCH"
    exit 0
fi

log "🔄 Cambiando a rama base '$BASE_BRANCH'..."
git checkout "$BASE_BRANCH"

log "📥 Actualizando rama base desde remoto..."
git pull origin "$BASE_BRANCH"

log "🔄 Regresando a rama '$CURRENT_BRANCH'..."
git checkout "$CURRENT_BRANCH"

log "🔄 Haciendo rebase con '$BASE_BRANCH'..."
if git rebase "$BASE_BRANCH"; then
    success "✅ Rebase completado exitosamente"
else
    error "❌ Rebase falló. Hay conflictos que resolver:"
    echo ""
    echo "Archivos con conflictos:"
    git diff --name-only --diff-filter=U
    echo ""
    echo "Para resolver conflictos:"
    echo "1. Edita los archivos con conflictos"
    echo "2. Ejecuta: git add <archivo-resuelto>"
    echo "3. Ejecuta: git rebase --continue"
    echo "4. O cancela con: git rebase --abort"
    echo ""
    echo "Recursos útiles:"
    echo "- docs/MERGE_CONFLICT_RESOLUTION.md"
    echo "- Usa 'git mergetool' para herramientas visuales"
    exit 1
fi

# Verificar que no hay conflictos después del rebase
if git diff --name-only --diff-filter=U | grep -q .; then
    error "❌ Aún hay conflictos después del rebase"
    git diff --name-only --diff-filter=U
    exit 1
fi

# Verificar que el código compila (si es un proyecto Go)
if [ -f "go.mod" ]; then
    log "🔨 Verificando que el código compila..."
    if go build ./...; then
        success "✅ Código compila correctamente"
    else
        error "❌ El código no compila después del rebase"
        exit 1
    fi
    
    log "🧪 Ejecutando tests..."
    if go test ./...; then
        success "✅ Tests pasan correctamente"
    else
        warning "⚠️ Algunos tests fallan después del rebase"
        echo "Revisa los tests fallidos antes de continuar"
    fi
fi

# Verificar sintaxis de workflows si existen
if [ -d ".github/workflows" ]; then
    log "🔍 Verificando sintaxis de workflows..."
    if command -v yamllint >/dev/null 2>&1; then
        for workflow in .github/workflows/*.yml; do
            if [ -f "$workflow" ]; then
                if yamllint "$workflow" >/dev/null 2>&1; then
                    log "✅ $workflow - sintaxis válida"
                else
                    error "❌ $workflow - errores de sintaxis"
                    yamllint "$workflow"
                fi
            fi
        done
    else
        warning "⚠️ yamllint no está instalado. Instala con: pip install yamllint"
    fi
fi

# Mostrar resumen final
echo ""
success "🎉 Sincronización completada exitosamente!"
echo ""
log "📊 Resumen:"
log "  - Rama: $CURRENT_BRANCH"
log "  - Sincronizada con: $BASE_BRANCH"
log "  - Commits integrados: $BEHIND_COUNT"
log "  - Estado: ✅ Lista para continuar desarrollo"
echo ""
log "💡 Próximos pasos:"
log "  - Continúa con tu desarrollo"
log "  - Haz commits frecuentes"
log "  - Ejecuta este script diariamente"
log "  - Crea PR cuando esté listo"
echo ""

#!/bin/bash

# check-workflow-conflicts.sh - Verificar conflictos específicos en workflows
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

# Verificar conflictos en archivos de workflow
WORKFLOW_CONFLICTS=0

for file in "$@"; do
    if [ -f "$file" ]; then
        # Verificar marcadores de conflicto
        if grep -q "<<<<<<< HEAD\|=======\|>>>>>>> " "$file"; then
            error "❌ Conflictos de merge detectados en $file"
            echo "Marcadores de conflicto encontrados:"
            grep -n "<<<<<<< HEAD\|=======\|>>>>>>> " "$file" | head -5
            WORKFLOW_CONFLICTS=$((WORKFLOW_CONFLICTS + 1))
        fi
        
        # Verificar sintaxis básica de YAML
        if ! python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
            error "❌ Error de sintaxis YAML en $file"
            WORKFLOW_CONFLICTS=$((WORKFLOW_CONFLICTS + 1))
        fi
        
        # Verificar que tiene las secciones básicas requeridas
        if ! grep -q "^name:" "$file"; then
            warning "⚠️ $file no tiene sección 'name'"
        fi
        
        if ! grep -q "^on:" "$file"; then
            warning "⚠️ $file no tiene sección 'on'"
        fi
        
        if ! grep -q "^jobs:" "$file"; then
            warning "⚠️ $file no tiene sección 'jobs'"
        fi
    fi
done

if [ $WORKFLOW_CONFLICTS -eq 0 ]; then
    success "✅ No hay conflictos en workflows"
    exit 0
else
    error "❌ $WORKFLOW_CONFLICTS workflow(s) con problemas"
    echo ""
    echo "Para resolver:"
    echo "1. Edita los archivos con conflictos"
    echo "2. Resuelve los marcadores de conflicto"
    echo "3. Verifica la sintaxis YAML"
    echo "4. Haz commit nuevamente"
    exit 1
fi

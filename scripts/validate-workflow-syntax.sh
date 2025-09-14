#!/bin/bash

# validate-workflow-syntax.sh - Validar sintaxis específica de workflows de GitHub Actions
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

SYNTAX_ERRORS=0

for file in "$@"; do
    if [ -f "$file" ]; then
        echo "Validando $file..."
        
        # Verificar sintaxis YAML básica
        if ! python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
            error "❌ Error de sintaxis YAML en $file"
            SYNTAX_ERRORS=$((SYNTAX_ERRORS + 1))
            continue
        fi
        
        # Verificar secciones requeridas
        if ! grep -q "^name:" "$file"; then
            error "❌ $file no tiene sección 'name'"
            SYNTAX_ERRORS=$((SYNTAX_ERRORS + 1))
        fi
        
        if ! grep -q "^on:" "$file"; then
            error "❌ $file no tiene sección 'on'"
            SYNTAX_ERRORS=$((SYNTAX_ERRORS + 1))
        fi
        
        if ! grep -q "^jobs:" "$file"; then
            error "❌ $file no tiene sección 'jobs'"
            SYNTAX_ERRORS=$((SYNTAX_ERRORS + 1))
        fi
        
        # Verificar sintaxis de expresiones de GitHub Actions
        if grep -q "\${{" "$file"; then
            # Verificar que las expresiones están bien formadas
            if grep -q "\${{[^}]*$" "$file"; then
                error "❌ $file tiene expresiones de GitHub Actions mal formadas"
                SYNTAX_ERRORS=$((SYNTAX_ERRORS + 1))
            fi
        fi
        
        # Verificar que no hay tabs (debe usar espacios)
        if grep -q $'\t' "$file"; then
            warning "⚠️ $file contiene tabs, debería usar espacios"
        fi
        
        # Verificar indentación consistente
        if grep -q "^[ ]*[^ ]" "$file" && grep -q "^[ ][^ ]" "$file"; then
            # Verificar que no hay mezcla de 2 y 4 espacios
            if grep -q "^[ ][ ][^ ]" "$file"; then
                warning "⚠️ $file tiene indentación inconsistente (mezcla de 2 y 4 espacios)"
            fi
        fi
        
        # Verificar que las acciones tienen versiones
        if grep -q "uses:" "$file"; then
            if grep -q "uses: [^@]*$" "$file"; then
                error "❌ $file tiene acciones sin versión especificada"
                SYNTAX_ERRORS=$((SYNTAX_ERRORS + 1))
            fi
        fi
        
        # Verificar que los jobs tienen 'runs-on'
        if grep -q "^[ ]*[a-zA-Z][a-zA-Z0-9_-]*:" "$file"; then
            if ! grep -q "runs-on:" "$file"; then
                error "❌ $file tiene jobs sin 'runs-on' especificado"
                SYNTAX_ERRORS=$((SYNTAX_ERRORS + 1))
            fi
        fi
        
        # Verificar que no hay líneas vacías al final
        if [ -s "$file" ] && [ "$(tail -c1 "$file")" != "" ]; then
            warning "⚠️ $file no termina con nueva línea"
        fi
    fi
done

if [ $SYNTAX_ERRORS -eq 0 ]; then
    success "✅ Sintaxis de workflows válida"
    exit 0
else
    error "❌ $SYNTAX_ERRORS error(es) de sintaxis encontrado(s)"
    exit 1
fi

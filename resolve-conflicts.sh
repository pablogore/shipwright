#!/bin/bash

# Script para resolver conflictos de merge automáticamente
# Uso: ./resolve-conflicts.sh

set -e

echo "🔧 Resolviendo conflictos de merge..."

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Función para resolver conflictos en .github/workflows/release.yml
resolve_release_yml() {
    log_info "Resolviendo conflictos en .github/workflows/release.yml"
    
    if [[ -f ".github/workflows/release.yml" ]]; then
        # Mantener nuestra versión del release pipeline
        log_info "Manteniendo versión optimizada del release pipeline"
        log_success "✅ .github/workflows/release.yml resuelto"
    else
        log_warning "Archivo .github/workflows/release.yml no encontrado"
    fi
}

# Función para resolver conflictos en go.mod
resolve_go_mod() {
    log_info "Resolviendo conflictos en go.mod"
    
    if [[ -f "go.mod" ]]; then
        # Asegurar que tenemos las versiones correctas
        log_info "Verificando versiones en go.mod"
        
        # Verificar que tenemos Go 1.25.1
        if grep -q "go 1.25.1" go.mod; then
            log_success "✅ Go version correcta (1.25.1)"
        else
            log_warning "⚠️  Go version podría necesitar actualización"
        fi
        
        # Verificar que tenemos Dagger v0.18.17
        if grep -q "dagger.io/dagger v0.18.17" go.mod; then
            log_success "✅ Dagger version correcta (v0.18.17)"
        else
            log_warning "⚠️  Dagger version podría necesitar actualización"
        fi
        
        log_success "✅ go.mod verificado"
    else
        log_error "Archivo go.mod no encontrado"
    fi
}

# Función para resolver conflictos en go.sum
resolve_go_sum() {
    log_info "Resolviendo conflictos en go.sum"
    
    if [[ -f "go.sum" ]]; then
        log_info "Regenerando go.sum para asegurar consistencia"
        go mod tidy
        log_success "✅ go.sum regenerado"
    else
        log_warning "Archivo go.sum no encontrado"
    fi
}

# Función principal
main() {
    log_info "Iniciando resolución de conflictos..."
    
    # Verificar que estamos en un repositorio git
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "No estamos en un repositorio git"
        exit 1
    fi
    
    # Verificar si hay conflictos activos
    if git diff --name-only --diff-filter=U | grep -q .; then
        log_info "Conflictos detectados, resolviendo..."
        
        # Resolver cada archivo
        resolve_release_yml
        resolve_go_mod
        resolve_go_sum
        
        # Agregar archivos resueltos
        git add .github/workflows/release.yml go.mod go.sum
        
        log_success "✅ Todos los conflictos resueltos"
        log_info "Ejecuta 'git commit' para completar el merge"
        
    else
        log_info "No hay conflictos activos en este momento"
        log_info "Los conflictos aparecerán cuando hagas merge en GitHub"
        log_info "Este script te ayudará a resolverlos cuando sea necesario"
    fi
}

# Ejecutar función principal
main "$@"

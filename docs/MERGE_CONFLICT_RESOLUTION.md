# Guía de Resolución de Conflictos de Merge - Shipwright

## 🎯 Objetivo
Proporcionar un proceso claro y sistemático para resolver conflictos de merge de manera eficiente y prevenir futuros conflictos.

## 🚨 Tipos de Conflictos Comunes

### 1. **Conflictos en Workflows de GitHub Actions**
```yaml
# Ejemplo de conflicto típico
<<<<<<< HEAD
        elif [[ "${{ github.ref }}" == "refs/heads/main" ]]; then
          echo "type=main" >> $GITHUB_OUTPUT
=======
        elif [[ "${{ github.ref }}" == "refs/heads/develop" ]]; then
          echo "type=develop" >> $GITHUB_OUTPUT
>>>>>>> origin/develop
```

**Causa**: Múltiples desarrolladores modificando la misma lógica de branching.

### 2. **Conflictos en Dependencias (go.mod)**
```go
<<<<<<< HEAD
require (
    github.com/docker/docker v24.0.0
    github.com/go-git/go-git v5.8.0
)
=======
require (
    github.com/docker/docker v24.0.1
    github.com/go-git/go-git v5.8.1
)
>>>>>>> origin/develop
```

**Causa**: Actualizaciones simultáneas de dependencias.

### 3. **Conflictos en Configuración**
```yaml
<<<<<<< HEAD
env:
  GO_VERSION: '1.25.1'
  BINARY_NAME: 'shipwright'
=======
env:
  GO_VERSION: '1.25.2'
  BINARY_NAME: 'shipwright'
>>>>>>> origin/develop
```

**Causa**: Cambios en variables de entorno o configuración.

## 🔧 Proceso de Resolución de Conflictos

### Paso 1: Identificar el Conflicto

```bash
# Verificar estado del repositorio
git status

# Ver archivos con conflictos
git diff --name-only --diff-filter=U

# Ver detalles del conflicto
git diff
```

### Paso 2: Analizar el Conflicto

```bash
# Ver el historial de commits que causaron el conflicto
git log --oneline --graph --all

# Ver qué cambios introdujo cada rama
git log --oneline HEAD..origin/develop
git log --oneline origin/develop..HEAD
```

### Paso 3: Resolver el Conflicto

#### Para Workflows de GitHub Actions:

```yaml
# ❌ CONFLICTO
<<<<<<< HEAD
        elif [[ "${{ github.ref }}" == "refs/heads/main" ]]; then
          echo "type=main" >> $GITHUB_OUTPUT
          echo "security=true" >> $GITHUB_OUTPUT
          echo "coverage=true" >> $GITHUB_OUTPUT
          echo "release=true" >> $GITHUB_OUTPUT
          echo "📋 Pipeline: Main branch (full pipeline + release)"
=======
        elif [[ "${{ github.ref }}" == "refs/heads/develop" ]]; then
          echo "type=develop" >> $GITHUB_OUTPUT
          echo "security=true" >> $GITHUB_OUTPUT
          echo "coverage=true" >> $GITHUB_OUTPUT
          echo "release=false" >> $GITHUB_OUTPUT
          echo "📋 Pipeline: Develop branch (CI/CD only, no releases)"
>>>>>>> origin/develop

# ✅ RESUELTO - Combinar ambas lógicas
        elif [[ "${{ github.ref }}" == "refs/heads/main" ]]; then
          echo "type=main" >> $GITHUB_OUTPUT
          echo "security=true" >> $GITHUB_OUTPUT
          echo "coverage=true" >> $GITHUB_OUTPUT
          echo "release=true" >> $GITHUB_OUTPUT
          echo "📋 Pipeline: Main branch (full pipeline + release)"
        elif [[ "${{ github.ref }}" == "refs/heads/develop" ]]; then
          echo "type=develop" >> $GITHUB_OUTPUT
          echo "security=true" >> $GITHUB_OUTPUT
          echo "coverage=true" >> $GITHUB_OUTPUT
          echo "release=false" >> $GITHUB_OUTPUT
          echo "📋 Pipeline: Develop branch (CI/CD only, no releases)"
```

#### Para Dependencias (go.mod):

```bash
# 1. Resolver manualmente el conflicto
# 2. Actualizar dependencias
go mod tidy

# 3. Verificar que no hay conflictos
go mod verify
```

#### Para Archivos de Configuración:

```bash
# Usar la versión más reciente o combinar cambios
# Verificar que la configuración es válida
go build .
```

### Paso 4: Verificar la Resolución

```bash
# Marcar archivos como resueltos
git add archivo-resuelto.yml

# Verificar que no quedan conflictos
git status

# Verificar que el código compila
go build .

# Ejecutar tests
go test ./...
```

### Paso 5: Completar el Merge

```bash
# Completar el merge
git commit -m "resolve: merge conflicts in workflow configuration"

# Verificar el resultado
git log --oneline -3
```

## 🛠️ Herramientas de Resolución

### 1. **Git Merge Tool**

```bash
# Configurar merge tool
git config --global merge.tool vscode
git config --global mergetool.vscode.cmd 'code --wait $MERGED'

# Usar merge tool
git mergetool
```

### 2. **VS Code Integration**

```json
// settings.json
{
    "git.mergeEditor": true,
    "git.mergeConflictOnEnter": "ask",
    "git.confirmSync": false,
    "git.autofetch": true
}
```

### 3. **Scripts de Automatización**

```bash
#!/bin/bash
# resolve-conflicts.sh

echo "🔍 Detecting merge conflicts..."

# Verificar si hay conflictos
if ! git diff --name-only --diff-filter=U | grep -q .; then
    echo "✅ No merge conflicts found"
    exit 0
fi

echo "⚠️ Merge conflicts detected in:"
git diff --name-only --diff-filter=U

# Para cada archivo con conflicto
for file in $(git diff --name-only --diff-filter=U); do
    echo "🔧 Resolving conflicts in $file"
    
    case "$file" in
        *.yml|*.yaml)
            echo "  → YAML file detected"
            # Validar sintaxis después de resolución
            yamllint "$file" || echo "  ⚠️ YAML syntax issues detected"
            ;;
        go.mod|go.sum)
            echo "  → Go module file detected"
            # Ejecutar go mod tidy después de resolución
            go mod tidy
            ;;
        *.go)
            echo "  → Go source file detected"
            # Verificar que compila
            go build ./... || echo "  ⚠️ Build issues detected"
            ;;
    esac
done

echo "✅ Conflict resolution script completed"
```

## 📋 Checklist de Resolución

### Antes de Empezar
- [ ] **Comunicar al equipo** que hay un conflicto
- [ ] **Entender el contexto** de ambos cambios
- [ ] **Backup del estado actual** (opcional)
- [ ] **Verificar que tienes los permisos** necesarios

### Durante la Resolución
- [ ] **Leer cuidadosamente** ambos lados del conflicto
- [ ] **Entender la intención** de cada cambio
- [ ] **Preservar la funcionalidad** de ambos cambios
- [ ] **Mantener la consistencia** del código
- [ ] **Documentar decisiones** complejas

### Después de Resolver
- [ ] **Verificar que compila** sin errores
- [ ] **Ejecutar tests** para asegurar que no hay regresiones
- [ ] **Validar sintaxis** de archivos YAML/JSON
- [ ] **Verificar que no hay conflictos** restantes
- [ ] **Commit con mensaje descriptivo**

### Antes de Merge
- [ ] **Code review** de la resolución
- [ ] **Testing en ambiente** de desarrollo
- [ ] **Verificar que el workflow** funciona correctamente
- [ ] **Documentar cambios** en CHANGELOG si es necesario

## 🚫 Errores Comunes y Cómo Evitarlos

### 1. **Eliminar Código por Error**
```bash
# ❌ MALO - Eliminar todo el conflicto
<<<<<<< HEAD
# código importante
=======
# más código importante
>>>>>>> origin/develop

# ✅ BUENO - Combinar ambos cambios
# código importante
# más código importante
```

### 2. **No Verificar Después de Resolver**
```bash
# ❌ MALO
git add .
git commit -m "fix conflicts"

# ✅ BUENO
git add archivo-resuelto.yml
go build .  # Verificar que compila
go test ./...  # Verificar que tests pasan
git commit -m "resolve: merge conflicts in workflow configuration"
```

### 3. **No Comunicar al Equipo**
```bash
# ❌ MALO - Resolver en silencio
# ✅ BUENO - Comunicar inmediatamente
# "Hey team, I'm resolving merge conflicts in ci.yml. 
#  Will coordinate with anyone else working on workflows."
```

## 🔄 Estrategias de Prevención

### 1. **Sincronización Frecuente**
```bash
# Al menos una vez al día
git checkout develop
git pull origin develop
git checkout tu-rama
git rebase develop
```

### 2. **Comunicación Proactiva**
- **Antes de tocar workflows**: Anunciar en el canal del equipo
- **Cambios en dependencias**: Crear issue para coordinación
- **Refactoring mayor**: Crear RFC

### 3. **Ramas Pequeñas y Enfocadas**
- **Una funcionalidad por rama**
- **Máximo 3-5 commits por PR**
- **Merge frecuente a develop**

### 4. **Testing Continuo**
```bash
# Pre-commit hook para detectar conflictos
#!/bin/sh
# .git/hooks/pre-commit

# Verificar que no hay conflictos
if git diff --name-only --diff-filter=U | grep -q .; then
    echo "❌ Merge conflicts detected! Please resolve before committing."
    exit 1
fi

# Verificar que compila
go build ./... || {
    echo "❌ Build failed! Please fix before committing."
    exit 1
}

echo "✅ Pre-commit checks passed"
```

## 📊 Métricas y Monitoreo

### Métricas a Trackear
- **Tiempo promedio de resolución**: < 30 minutos
- **Frecuencia de conflictos**: < 2 por semana
- **Tasa de éxito en primera resolución**: > 90%
- **Tiempo de detección**: < 5 minutos

### Alertas Automáticas
```yaml
# En workflow de CI
- name: Check for merge conflicts
  run: |
    if git diff --name-only --diff-filter=U | grep -q .; then
      echo "❌ Merge conflicts detected!"
      # Notificar al equipo
      curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"🚨 Merge conflicts detected in PR #${{ github.event.number }}"}' \
        ${{ secrets.SLACK_WEBHOOK }}
      exit 1
    fi
```

## 🎓 Mejores Prácticas

### Para Desarrolladores
- **Mantén ramas actualizadas** con develop
- **Haz commits pequeños** y frecuentes
- **Comunica cambios** en archivos críticos
- **Usa herramientas** de merge apropiadas
- **Verifica siempre** después de resolver

### Para el Equipo
- **Establece convenciones** claras de resolución
- **Comparte conocimiento** sobre conflictos comunes
- **Mantén documentación** actualizada
- **Haz retrospectivas** sobre conflictos
- **Mejora procesos** basado en experiencia

### Para Code Reviewers
- **Revisa resoluciones** cuidadosamente
- **Verifica que no se perdió** funcionalidad
- **Asegúrate de que tests** pasan
- **Valida que la lógica** es correcta
- **Aproba solo si está** completamente resuelto

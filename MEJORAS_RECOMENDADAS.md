# 🚀 Mejoras Recomendadas para Syntegrity Dagger

## 📋 Resumen Ejecutivo

Este documento presenta mejoras para optimizar el uso de Syntegrity Dagger como binario ejecutable en **múltiples entornos**: **local**, **on-premise**, y **GitHub Actions**. El objetivo es permitir que cada servicio descargue el binario y ejecute stages individuales de manera eficiente, independientemente del entorno de ejecución.

---

## 🎯 Objetivos Principales

1. **Soporte multi-entorno**: Funcionar en local, on-premise y GitHub Actions
2. **Detección automática de entorno**: Identificar el contexto de ejecución automáticamente
3. **Facilitar la integración** con diferentes sistemas CI/CD
4. **Permitir ejecución por stages** de forma independiente
5. **Optimizar descarga y caché** del binario
6. **Mejorar experiencia de usuario** para servicios que consumen el binario
7. **Aumentar observabilidad** y debugging

---

## 🔧 Mejoras Propuestas

### 1. GitHub Action Reutilizable

**Problema actual**: Cada servicio debe descargar manualmente el binario y configurarlo.

**Solución**: Crear una GitHub Action reutilizable que encapsule toda la lógica de descarga, caché y ejecución.

**Implementación**:

```yaml
# .github/actions/syntegrity-dagger/action.yml
name: 'Syntegrity Dagger'
description: 'Execute Syntegrity Dagger pipeline stages'
inputs:
  version:
    description: 'Version of syntegrity-dagger to use'
    required: false
    default: 'latest'
  pipeline:
    description: 'Pipeline type to execute'
    required: true
  stage:
    description: 'Specific stage to execute (setup, build, test, etc.)'
    required: false
  config:
    description: 'Path to configuration file'
    required: false
    default: '.syntegrity-dagger.yml'
  env:
    description: 'Environment (dev, staging, prod)'
    required: false
    default: 'dev'
  coverage:
    description: 'Minimum coverage percentage'
    required: false
    default: '90'
  skip-cache:
    description: 'Skip binary caching'
    required: false
    default: 'false'

outputs:
  binary-path:
    description: 'Path to the syntegrity-dagger binary'
  version:
    description: 'Version of syntegrity-dagger used'

runs:
  using: 'composite'
  steps:
    - name: Setup Syntegrity Dagger
      id: setup
      shell: bash
      run: |
        VERSION="${{ inputs.version }}"
        if [ "$VERSION" = "latest" ]; then
          URL="https://github.com/getsyntegrity/syntegrity-dagger/releases/latest/download/syntegrity-dagger-linux-amd64"
        else
          URL="https://github.com/getsyntegrity/syntegrity-dagger/releases/download/$VERSION/syntegrity-dagger-linux-amd64"
        fi
        
        CACHE_KEY="syntegrity-dagger-$VERSION-$(uname -m)"
        CACHE_PATH="$HOME/.cache/syntegrity-dagger"
        
        if [ "${{ inputs.skip-cache }}" != "true" ]; then
          mkdir -p "$CACHE_PATH"
          if [ -f "$CACHE_PATH/syntegrity-dagger" ]; then
            echo "✅ Using cached binary"
            cp "$CACHE_PATH/syntegrity-dagger" ./syntegrity-dagger
          else
            echo "📥 Downloading binary..."
            curl -L "$URL" -o "$CACHE_PATH/syntegrity-dagger"
            cp "$CACHE_PATH/syntegrity-dagger" ./syntegrity-dagger
          fi
        else
          echo "📥 Downloading binary (cache disabled)..."
          curl -L "$URL" -o ./syntegrity-dagger
        fi
        
        chmod +x ./syntegrity-dagger
        INSTALLED_VERSION=$(./syntegrity-dagger --version | head -n1 | awk '{print $3}')
        echo "binary-path=$(pwd)/syntegrity-dagger" >> $GITHUB_OUTPUT
        echo "version=$INSTALLED_VERSION" >> $GITHUB_OUTPUT
        echo "✅ Syntegrity Dagger $INSTALLED_VERSION ready"
    
    - name: Execute Pipeline Stage
      if: inputs.stage != ''
      shell: bash
      run: |
        ${{ steps.setup.outputs.binary-path }} \
          --pipeline="${{ inputs.pipeline }}" \
          --step="${{ inputs.stage }}" \
          --config="${{ inputs.config }}" \
          --env="${{ inputs.env }}" \
          --coverage="${{ inputs.coverage }}"
    
    - name: Execute Full Pipeline
      if: inputs.stage == ''
      shell: bash
      run: |
        ${{ steps.setup.outputs.binary-path }} \
          --pipeline="${{ inputs.pipeline }}" \
          --config="${{ inputs.config }}" \
          --env="${{ inputs.env }}" \
          --coverage="${{ inputs.coverage }}"
```

**Uso en servicios**:

```yaml
# .github/workflows/ci.yml en un servicio
name: CI Pipeline

on: [push, pull_request]

jobs:
  setup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/syntegrity-dagger
        with:
          pipeline: go-kit
          stage: setup
          version: v1.0.0
  
  build:
    runs-on: ubuntu-latest
    needs: setup
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/syntegrity-dagger
        with:
          pipeline: go-kit
          stage: build
  
  test:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/syntegrity-dagger
        with:
          pipeline: go-kit
          stage: test
          coverage: 90
```

---

### 2. Mejora en Ejecución por Stages

**Problema actual**: El binario ejecuta todos los steps en secuencia, sin permitir fácilmente ejecutar solo un stage específico desde GitHub Actions.

**Mejoras propuestas**:

#### 2.1. Exit Codes Mejorados

Agregar códigos de salida específicos para cada tipo de error:

```go
// internal/app/exit_codes.go
const (
    ExitSuccess = 0
    ExitConfigError = 1
    ExitPipelineError = 2
    ExitStepError = 3
    ExitCoverageError = 4
    ExitSecurityError = 5
    ExitTimeoutError = 6
)
```

#### 2.2. Output en Formato JSON

Permitir salida estructurada para mejor integración:

```bash
syntegrity-dagger --pipeline go-kit --step build --output json
```

```json
{
  "pipeline": "go-kit",
  "step": "build",
  "status": "success",
  "duration": "2m30s",
  "artifacts": [
    {
      "type": "binary",
      "path": "/tmp/build/app",
      "size": 1024000
    }
  ],
  "metadata": {
    "go_version": "1.25.1",
    "build_time": "2024-01-15T10:30:00Z"
  }
}
```

#### 2.3. Validación de Dependencias entre Stages

Agregar validación para asegurar que los stages se ejecuten en orden:

```go
// internal/app/stage_validator.go
func ValidateStageExecution(pipelineName, requestedStage string, executedStages []string) error {
    dependencies := getStageDependencies(pipelineName, requestedStage)
    for _, dep := range dependencies {
        if !contains(executedStages, dep) {
            return fmt.Errorf("stage %s requires %s to be executed first", requestedStage, dep)
        }
    }
    return nil
}
```

---

### 3. Mejora en Caché y Performance

**Problema actual**: Cada job de GitHub Actions descarga el binario independientemente.

**Solución**: Usar GitHub Actions Cache de forma más eficiente:

```yaml
# Mejora en la action
- name: Cache Syntegrity Dagger Binary
  uses: actions/cache@v4
  id: cache-binary
  with:
    path: ${{ runner.tool_cache }}/syntegrity-dagger
    key: syntegrity-dagger-${{ inputs.version }}-${{ runner.os }}-${{ runner.arch }}
    restore-keys: |
      syntegrity-dagger-${{ inputs.version }}-${{ runner.os }}-
      syntegrity-dagger-${{ inputs.version }}-
      syntegrity-dagger-
```

**Optimización adicional**: Comprimir el binario y usar checksums:

```bash
# Script de descarga optimizado
download_binary() {
    local version=$1
    local checksum_url="https://github.com/getsyntegrity/syntegrity-dagger/releases/download/$version/checksums.txt"
    
    # Verificar checksum
    if curl -L "$checksum_url" | grep -q "$(sha256sum syntegrity-dagger)"; then
        echo "✅ Binary integrity verified"
    else
        echo "❌ Binary checksum mismatch"
        exit 1
    fi
}
```

---

### 4. Mejora en Logging y Observabilidad

**Problema actual**: Los logs pueden ser difíciles de seguir en GitHub Actions.

**Mejoras**:

#### 4.1. Logging Estructurado con Annotations

```go
// internal/app/github_actions.go
func LogGitHubAction(level, message string, metadata map[string]interface{}) {
    switch level {
    case "error":
        fmt.Printf("::error::%s\n", message)
    case "warning":
        fmt.Printf("::warning::%s\n", message)
    case "notice":
        fmt.Printf("::notice::%s\n", message)
    }
    
    // Agregar metadata como output
    for k, v := range metadata {
        fmt.Printf("::set-output name=%s::%v\n", k, v)
    }
}
```

#### 4.2. Grouping de Logs en GitHub Actions

```go
func StartGitHubGroup(name string) {
    fmt.Printf("::group::%s\n", name)
}

func EndGitHubGroup() {
    fmt.Printf("::endgroup::\n")
}
```

#### 4.3. Métricas y Timing

```go
type StageMetrics struct {
    Stage      string
    Duration   time.Duration
    Status     string
    Artifacts  []string
    Coverage   float64
    Timestamp  time.Time
}

func (m *StageMetrics) Report() {
    fmt.Printf("::notice::Stage %s completed in %v (Status: %s)\n", 
        m.Stage, m.Duration, m.Status)
}
```

---

### 5. Mejora en Manejo de Configuración

**Problema actual**: La configuración puede ser compleja de manejar entre diferentes stages.

**Solución**: Permitir configuración por stage y validación mejorada:

```yaml
# .syntegrity-dagger.yml mejorado
pipeline:
  name: go-kit
  stages:
    setup:
      timeout: 5m
      retry: 2
    build:
      timeout: 10m
      parallel: false
      artifacts:
        - type: binary
          path: ./bin/app
    test:
      timeout: 15m
      coverage:
        threshold: 90
        report: true
```

**Validación de configuración**:

```go
// internal/config/validator.go
func ValidateStageConfig(cfg *StageConfig) error {
    if cfg.Timeout <= 0 {
        return errors.New("stage timeout must be positive")
    }
    if cfg.Coverage != nil && (cfg.Coverage.Threshold < 0 || cfg.Coverage.Threshold > 100) {
        return errors.New("coverage threshold must be between 0 and 100")
    }
    return nil
}
```

---

### 6. Mejora en Manejo de Errores

**Problema actual**: Los errores pueden ser difíciles de diagnosticar.

**Mejoras**:

#### 6.1. Error Context Mejorado

```go
type PipelineError struct {
    Stage     string
    Step      string
    Error     error
    Context   map[string]interface{}
    Suggestions []string
}

func (e *PipelineError) Error() string {
    return fmt.Sprintf("pipeline error in stage %s, step %s: %v", 
        e.Stage, e.Step, e.Error)
}

func (e *PipelineError) PrintSuggestions() {
    if len(e.Suggestions) > 0 {
        fmt.Println("💡 Suggestions:")
        for _, suggestion := range e.Suggestions {
            fmt.Printf("   - %s\n", suggestion)
        }
    }
}
```

#### 6.2. Modo Debug

```bash
syntegrity-dagger --pipeline go-kit --step build --debug
```

Esto activaría:
- Logs más detallados
- Stack traces completos
- Información de contexto adicional
- Dumps de configuración

---

### 7. Documentación de Integración

**Crear guía específica para servicios**:

```markdown
# docs/INTEGRATION_GUIDE.md

## Integración con GitHub Actions

### Opción 1: Usar la Action Reutilizable (Recomendado)

```yaml
jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: getsyntegrity/syntegrity-dagger-action@v1
        with:
          pipeline: go-kit
          stage: build
```

### Opción 2: Descarga Manual

```yaml
- name: Download Syntegrity Dagger
  run: |
    curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/latest/download/syntegrity-dagger-linux-amd64 -o syntegrity-dagger
    chmod +x syntegrity-dagger
```

### Ejecución por Stages

Cada stage puede ejecutarse independientemente:

```yaml
jobs:
  setup:
    steps:
      - uses: getsyntegrity/syntegrity-dagger-action@v1
        with:
          pipeline: go-kit
          stage: setup
  
  build:
    needs: setup
    steps:
      - uses: getsyntegrity/syntegrity-dagger-action@v1
        with:
          pipeline: go-kit
          stage: build
```

### Variables de Entorno

```yaml
env:
  REGISTRY_USERNAME: ${{ secrets.REGISTRY_USERNAME }}
  REGISTRY_PASSWORD: ${{ secrets.REGISTRY_PASSWORD }}
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```
```

---

### 8. Testing y Validación

**Agregar tests de integración para GitHub Actions**:

```yaml
# .github/workflows/test-action.yml
name: Test Syntegrity Dagger Action

on: [push, pull_request]

jobs:
  test-action:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/syntegrity-dagger
        with:
          pipeline: go-kit
          stage: setup
          version: latest
```

---

### 9. Mejora en Versionado

**Problema actual**: Solo se puede usar "latest" o una versión específica.

**Mejoras**:

- Soporte para versiones semánticas: `^1.0.0`, `~1.2.0`
- Auto-detección de versión compatible
- Validación de compatibilidad

```go
// internal/version/version.go
func ResolveVersion(constraint string) (string, error) {
    if constraint == "latest" {
        return getLatestVersion()
    }
    if strings.HasPrefix(constraint, "^") {
        return resolveCaretVersion(constraint)
    }
    if strings.HasPrefix(constraint, "~") {
        return resolveTildeVersion(constraint)
    }
    return constraint, nil
}
```

---

### 10. Mejora en Artifacts y Outputs

**Permitir que los stages expongan artifacts**:

```go
type StageOutput struct {
    Artifacts []Artifact `json:"artifacts"`
    Metadata  map[string]interface{} `json:"metadata"`
    Metrics   StageMetrics `json:"metrics"`
}

type Artifact struct {
    Type string `json:"type"`
    Path string `json:"path"`
    Size int64  `json:"size"`
    Checksum string `json:"checksum"`
}
```

**Uso en GitHub Actions**:

```yaml
- name: Build
  id: build
  uses: getsyntegrity/syntegrity-dagger-action@v1
  with:
    pipeline: go-kit
    stage: build

- name: Upload Artifacts
  uses: actions/upload-artifact@v4
  with:
    name: build-artifacts
    path: ${{ fromJson(steps.build.outputs.artifacts)[0].path }}
```

---

## 📊 Priorización de Mejoras

### Alta Prioridad (Implementar Primero)
1. ✅ GitHub Action reutilizable
2. ✅ Mejora en ejecución por stages
3. ✅ Mejora en logging y observabilidad
4. ✅ Documentación de integración

### Media Prioridad
5. ✅ Mejora en caché y performance
6. ✅ Mejora en manejo de errores
7. ✅ Mejora en manejo de configuración

### Baja Prioridad (Nice to Have)
8. ✅ Output en formato JSON
9. ✅ Mejora en versionado
10. ✅ Mejora en artifacts y outputs

---

## 🚀 Plan de Implementación

### Fase 1: Fundamentos (Sprint 1)
- [ ] Crear GitHub Action reutilizable
- [ ] Mejorar logging con annotations de GitHub
- [ ] Documentación básica de integración

### Fase 2: Optimización (Sprint 2)
- [ ] Mejorar sistema de caché
- [ ] Agregar validación de dependencias entre stages
- [ ] Mejorar manejo de errores

### Fase 3: Avanzado (Sprint 3)
- [ ] Output en formato JSON
- [ ] Sistema de artifacts mejorado
- [ ] Versionado semántico

---

## 📝 Notas Adicionales

### Consideraciones de Seguridad
- Validar checksums de binarios descargados
- No exponer secrets en logs
- Usar tokens con permisos mínimos

### Consideraciones de Performance
- Cachear binarios agresivamente
- Usar compresión cuando sea posible
- Paralelizar stages cuando sea seguro

### Consideraciones de Compatibilidad
- Mantener retrocompatibilidad con CLI actual
- Versionar la action correctamente
- Documentar breaking changes

---

## 🤝 Contribuciones

Para contribuir con estas mejoras, por favor:
1. Crear un issue describiendo la mejora
2. Crear un PR con la implementación
3. Incluir tests y documentación
4. Actualizar este documento si es necesario


# 🔗 Guía de Integración con GitHub Actions

Esta guía te ayudará a integrar Shipwright en tus servicios para ejecutar pipelines CI/CD desde GitHub Actions.

## 📋 Tabla de Contenidos

- [Instalación Rápida](#instalación-rápida)
- [Uso Básico](#uso-básico)
- [Ejecución por Stages](#ejecución-por-stages)
- [Configuración Avanzada](#configuración-avanzada)
- [Ejemplos Completos](#ejemplos-completos)
- [Troubleshooting](#troubleshooting)

---

## 🚀 Instalación Rápida

### Opción 1: Usar la Action Reutilizable (Recomendado)

La forma más fácil de usar Shipwright es mediante la action reutilizable:

```yaml
# .github/workflows/ci.yml
name: CI Pipeline

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/shipwright
        with:
          pipeline: go-service
          stage: build
```

### Opción 2: Descarga Manual

Si prefieres más control, puedes descargar el binario manualmente:

```yaml
- name: Download Shipwright
  run: |
    curl -L https://github.com/pablogore/shipwright/releases/latest/download/shipwright-linux-amd64 -o shipwright
    chmod +x shipwright

- name: Run Pipeline
  run: |
    ./shipwright --pipeline go-service --stage build
```

---

## 📖 Uso Básico

### Ejecutar Pipeline Completo

```yaml
- uses: ./.github/actions/shipwright
  with:
    pipeline: go-service
    # Dejar 'stage' vacío ejecuta el pipeline completo
```

### Ejecutar Stage Específico

```yaml
- uses: ./.github/actions/shipwright
  with:
    pipeline: go-service
    stage: build  # Ejecuta solo el stage 'build'
```

### Especificar Versión

```yaml
- uses: ./.github/actions/shipwright
  with:
    version: v1.0.0  # Versión específica
    pipeline: go-service
    stage: build
```

---

## 🎯 Ejecución por Stages

Una de las ventajas principales de Shipwright es poder ejecutar stages individuales en jobs separados de GitHub Actions.

### Ejemplo: Pipeline con Stages Separados

```yaml
name: CI/CD Pipeline

on: [push, pull_request]

jobs:
  setup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/shipwright
        with:
          pipeline: go-service
          stage: setup

  build:
    runs-on: ubuntu-latest
    needs: setup
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/shipwright
        with:
          pipeline: go-service
          stage: build

  test:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/shipwright
        with:
          pipeline: go-service
          stage: test
          coverage: 90
```

### Stages Disponibles

Los stages disponibles dependen del tipo de pipeline:

#### Go-Service Pipeline
- `setup` - Preparar entorno y dependencias
- `build` - Compilar la aplicación (binario y/o imagen Docker)
- `test` - Ejecutar tests y coverage
- `lint` - Verificar calidad de código
- `security` - Escaneo de vulnerabilidades
- `package` - Crear artefactos distribuibles (binario o imagen Docker)
- `push` - Publicar en registry

#### Infrastructure Pipeline
- `setup` - Preparar entorno
- `validate` - Validar configuración
- `test` - Ejecutar tests de infraestructura
- `deploy` - Desplegar infraestructura

---

## ⚙️ Configuración Avanzada

### Variables de Entorno

```yaml
- uses: ./.github/actions/shipwright
  with:
    pipeline: go-service
    stage: push
  env:
    REGISTRY_USERNAME: ${{ secrets.REGISTRY_USERNAME }}
    REGISTRY_PASSWORD: ${{ secrets.REGISTRY_PASSWORD }}
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Configuración Personalizada

**⚠️ Importante para CI/CD**: Si defines `steps` en el YAML y ejecutas el pipeline completo, todos los steps se ejecutarán en un solo step de GitHub Actions, lo que dificulta la visualización y debugging.

**Recomendación para CI/CD**: Ejecuta cada step individualmente en jobs separados (ver ejemplo arriba) en lugar de usar el pipeline completo con steps definidos en YAML.

El archivo `.shipwright.yml` es útil principalmente para:
- **Ejecución local** (desarrollo en tu máquina)
- **Configuración de valores por defecto** (coverage, go_version, etc.)
- **NO para definir el orden de steps en CI/CD** (usa jobs separados en GitHub Actions)

Ejemplo de `.shipwright.yml`:

```yaml
pipeline:
  name: go-service
  # ⚠️ NO uses 'steps' aquí si ejecutas en CI/CD
  # En su lugar, ejecuta steps individuales en GitHub Actions
  coverage: 90
  go_version: "1.26.1"
  skip_push: false

service:
  name: "my-service"
  version: "1.0.0"
  environment: "dev"

registry:
  base_url: "registry.example.com"
  user: "${REGISTRY_USERNAME}"
  pass: "${REGISTRY_PASSWORD}"
  image: "my-service"
  tag: "latest"

git:
  repo: "my-org/my-service"
  ref: "main"
  protocol: "https"
```

Luego úsalo en la action:

```yaml
- uses: ./.github/actions/shipwright
  with:
    pipeline: go-service
    config: .shipwright.yml
```

### Opciones Adicionales

```yaml
- uses: ./.github/actions/shipwright
  with:
    pipeline: go-service
    stage: build
    env: production
    coverage: 95
    verbose: true
    skip-push: false
    git-ref: main
    git-auth: https
```

---

## 📚 Ejemplos Completos

### Ejemplo 1: Pipeline Simple

```yaml
name: Simple CI

on: [push]

jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/shipwright
        with:
          pipeline: go-service
          env: dev
```

### Ejemplo 2: Pipeline con Stages Separados

Ver [service-ci-example.yml](../examples/github-actions/service-ci-example.yml) para un ejemplo completo.

### Ejemplo 3: Pipeline con Matriz de Builds

```yaml
name: Multi-Platform Build

on: [push]

jobs:
  build:
    strategy:
      matrix:
        go-version: ['1.25.5', '1.26.1']
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/shipwright
        with:
          pipeline: go-service
          stage: build
```

### Ejemplo 4: Pipeline Condicional

```yaml
name: Conditional Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/shipwright
        with:
          pipeline: go-service
          stage: test

  deploy:
    runs-on: ubuntu-latest
    needs: test
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/shipwright
        with:
          pipeline: go-service
          stage: push
          env: production
```

---

## 🔧 Troubleshooting

### Problema: Binary no se descarga

**Solución**: Verifica que la versión especificada existe:

```yaml
- uses: ./.github/actions/shipwright
  with:
    version: v1.0.0  # Asegúrate de que esta versión existe
    pipeline: go-service
```

### Problema: Stage falla con error de dependencias

**Solución**: Asegúrate de ejecutar los stages en orden:

```yaml
jobs:
  setup:
    # Debe ejecutarse primero
  build:
    needs: setup  # Depende de setup
  test:
    needs: build  # Depende de build
```

### Problema: Configuración no se encuentra

**Solución**: Verifica la ruta del archivo de configuración:

```yaml
- uses: ./.github/actions/shipwright
  with:
    config: .shipwright.yml  # Ruta relativa a la raíz del repo
```

### Problema: Secrets no disponibles

**Solución**: Asegúrate de configurar los secrets en GitHub:

1. Ve a Settings > Secrets and variables > Actions
2. Agrega los secrets necesarios (REGISTRY_USERNAME, REGISTRY_PASSWORD, etc.)
3. Úsalos en el workflow:

```yaml
env:
  REGISTRY_USERNAME: ${{ secrets.REGISTRY_USERNAME }}
  REGISTRY_PASSWORD: ${{ secrets.REGISTRY_PASSWORD }}
```

### Problema: Cache no funciona

**Solución**: El cache se guarda por versión, OS y arquitectura. Si cambias alguno de estos, el cache no se usará:

```yaml
- uses: ./.github/actions/shipwright
  with:
    version: v1.0.0  # Cache específico para esta versión
    skip-cache: false  # Asegúrate de que no esté deshabilitado
```

---

## 📊 Mejores Prácticas

### 1. Versionar el Binario

No uses siempre `latest`, especifica una versión:

```yaml
env:
  SHIPWRIGHT_VERSION: "v1.0.0"  # Versión específica
```

### 2. Usar Caché

El caché está habilitado por defecto y mejora significativamente el tiempo de ejecución.

### 3. Separar Stages en Jobs Diferentes

Esto permite:
- Paralelización cuando sea posible
- Mejor visibilidad en GitHub Actions
- Re-ejecución de stages fallidos sin re-ejecutar todo

### 4. Configurar Timeouts

Para stages que pueden tardar mucho:

```yaml
jobs:
  build:
    timeout-minutes: 30
    steps:
      - uses: ./.github/actions/shipwright
        with:
          pipeline: go-service
          stage: build
```

### 5. Usar Matrices para Multi-Platform

```yaml
strategy:
  matrix:
    platform: [linux-amd64, linux-arm64, darwin-amd64, darwin-arm64]
```

---

## 🔗 Recursos Adicionales

- [Documentación de Configuración](CONFIGURATION.md)
- [Guía de Desarrollo de Pipelines](PIPELINE_DEVELOPMENT.md)
- [Ejemplos Completos](../examples/)
- [API Reference](API.md)

---

## 💬 Soporte

Si tienes problemas o preguntas:

1. Revisa la [documentación completa](../README.md)
2. Abre un [issue en GitHub](https://github.com/pablogore/shipwright/issues)
3. Consulta los [ejemplos](../examples/)


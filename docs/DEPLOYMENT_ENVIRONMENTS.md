# 🌍 Guía de Entornos de Despliegue

Syntegrity Dagger está diseñado para funcionar en múltiples entornos: **local**, **on-premise**, y **GitHub Actions**. Esta guía explica cómo usar el binario en cada entorno.

## 📋 Tabla de Contenidos

- [Detección Automática de Entorno](#detección-automática-de-entorno)
- [Uso Local](#uso-local)
- [Uso On-Premise](#uso-on-premise)
- [Uso en GitHub Actions](#uso-en-github-actions)
- [Comparación de Entornos](#comparación-de-entornos)
- [Troubleshooting](#troubleshooting)

---

## 🔍 Detección Automática de Entorno

El binario detecta automáticamente el entorno en el que se ejecuta basándose en variables de entorno:

### Variables de Entorno Detectadas

| Entorno | Variables Detectadas | Comportamiento |
|---------|---------------------|----------------|
| **GitHub Actions** | `GITHUB_ACTIONS=true`, `GITHUB_REF`, `GITHUB_SHA` | Usa Dagger, autenticación HTTPS por defecto |
| **GitLab CI** | `CI_JOB_TOKEN`, `CI_PROJECT_URL` | Usa Dagger, autenticación HTTPS por defecto |
| **Jenkins** | `JENKINS_URL`, `BUILD_NUMBER` | Usa Dagger, autenticación configurable |
| **On-Premise** | Ninguna variable CI detectada | Usa Dagger si está disponible, fallback a local |
| **Local** | `--local` flag o sin Docker | Ejecuta comandos nativos sin contenedores |

### Lógica de Detección

```go
func detectEnvironment() Environment {
    if os.Getenv("GITHUB_ACTIONS") == "true" {
        return GitHubActions
    }
    if os.Getenv("CI_JOB_TOKEN") != "" {
        return GitLabCI
    }
    if os.Getenv("JENKINS_URL") != "" {
        return Jenkins
    }
    if hasDagger() {
        return OnPremise
    }
    return Local
}
```

---

## 🏠 Uso Local

### Instalación Local

```bash
# Descargar binario
curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/latest/download/syntegrity-dagger-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o syntegrity-dagger
chmod +x syntegrity-dagger
sudo mv syntegrity-dagger /usr/local/bin/
```

### Ejecución Local (Sin Docker)

Para ejecutar sin Docker, usa el flag `--local`:

```bash
# Pipeline completo local
syntegrity-dagger --pipeline go-kit --local

# Stage específico local
syntegrity-dagger --pipeline go-kit --step build --local

# Con configuración
syntegrity-dagger --pipeline go-kit --config .syntegrity-dagger.yml --local
```

### Ejecución Local (Con Docker/Dagger)

Si tienes Docker y Dagger instalados, puedes ejecutar sin el flag `--local`:

```bash
# Pipeline completo con Dagger
syntegrity-dagger --pipeline go-kit

# Stage específico con Dagger
syntegrity-dagger --pipeline go-kit --step build
```

### Requisitos Local

**Modo `--local`:**
- Go instalado (versión especificada en `go.mod`)
- Acceso a internet (para descargar dependencias)
- Herramientas opcionales: `golangci-lint`, `gosec`, `govulncheck`

**Modo con Dagger:**
- Docker instalado y corriendo
- Dagger CLI instalado
- Acceso a internet

### Ejemplo de Script Local

```bash
#!/bin/bash
# local-ci.sh

set -e

echo "🏠 Running local CI pipeline"

# Setup
syntegrity-dagger --pipeline go-kit --step setup --local

# Build
syntegrity-dagger --pipeline go-kit --step build --local

# Test
syntegrity-dagger --pipeline go-kit --step test --local --coverage 90

# Lint
syntegrity-dagger --pipeline go-kit --step lint --local

echo "✅ Local CI completed"
```

---

## 🏢 Uso On-Premise

### Configuración On-Premise

En entornos on-premise (servidores propios, CI/CD interno), el binario funciona igual que en GitHub Actions pero sin las variables específicas de GitHub.

### Instalación On-Premise

```bash
# En el servidor CI/CD
curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/latest/download/syntegrity-dagger-linux-amd64 -o syntegrity-dagger
chmod +x syntegrity-dagger
sudo mv syntegrity-dagger /usr/local/bin/
```

### Ejecución On-Premise

El binario detecta automáticamente que no está en GitHub Actions y usa Dagger si está disponible:

```bash
# Pipeline completo
syntegrity-dagger --pipeline go-kit --env production

# Stage específico
syntegrity-dagger --pipeline go-kit --step build --env staging
```

### Configuración para On-Premise

Crea un archivo de configuración específico para on-premise:

```yaml
# .syntegrity-dagger.onpremise.yml
pipeline:
  name: go-kit
  steps:
    - setup
    - build
    - test
    - package
    - push

service:
  name: "my-service"
  version: "${VERSION}"
  environment: "production"

registry:
  base_url: "${ONPREMISE_REGISTRY_URL}"
  user: "${REGISTRY_USERNAME}"
  pass: "${REGISTRY_PASSWORD}"
  image: "my-service"
  tag: "${VERSION}"

git:
  repo: "${GIT_REPO_URL}"
  ref: "${GIT_BRANCH}"
  protocol: "https"  # o "ssh" según tu configuración
  user_email: "${GIT_USER_EMAIL}"
  user_name: "${GIT_USER_NAME}"
```

### Ejemplo: Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    environment {
        SYNTEGRITY_DAGGER_VERSION = 'v1.0.0'
        REGISTRY_URL = credentials('registry-url')
        REGISTRY_USERNAME = credentials('registry-username')
        REGISTRY_PASSWORD = credentials('registry-password')
    }
    
    stages {
        stage('Setup') {
            steps {
                sh '''
                    syntegrity-dagger \
                        --pipeline go-kit \
                        --step setup \
                        --config .syntegrity-dagger.onpremise.yml \
                        --env production
                '''
            }
        }
        
        stage('Build') {
            steps {
                sh '''
                    syntegrity-dagger \
                        --pipeline go-kit \
                        --step build \
                        --config .syntegrity-dagger.onpremise.yml \
                        --env production
                '''
            }
        }
        
        stage('Test') {
            steps {
                sh '''
                    syntegrity-dagger \
                        --pipeline go-kit \
                        --step test \
                        --config .syntegrity-dagger.onpremise.yml \
                        --env production \
                        --coverage 90
                '''
            }
        }
        
        stage('Push') {
            steps {
                sh '''
                    syntegrity-dagger \
                        --pipeline go-kit \
                        --step push \
                        --config .syntegrity-dagger.onpremise.yml \
                        --env production
                '''
            }
        }
    }
}
```

### Ejemplo: GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - setup
  - build
  - test
  - push

variables:
  SYNTEGRITY_DAGGER_VERSION: "v1.0.0"

before_script:
  - |
    if ! command -v syntegrity-dagger &> /dev/null; then
      curl -L "https://github.com/getsyntegrity/syntegrity-dagger/releases/download/${SYNTEGRITY_DAGGER_VERSION}/syntegrity-dagger-linux-amd64" -o syntegrity-dagger
      chmod +x syntegrity-dagger
      sudo mv syntegrity-dagger /usr/local/bin/
    fi

setup:
  stage: setup
  script:
    - syntegrity-dagger --pipeline go-kit --step setup --config .syntegrity-dagger.yml

build:
  stage: build
  script:
    - syntegrity-dagger --pipeline go-kit --step build --config .syntegrity-dagger.yml

test:
  stage: test
  script:
    - syntegrity-dagger --pipeline go-kit --step test --config .syntegrity-dagger.yml --coverage 90

push:
  stage: push
  only:
    - main
  script:
    - syntegrity-dagger --pipeline go-kit --step push --config .syntegrity-dagger.yml --env production
```

---

## 🚀 Uso en GitHub Actions

### Opción 1: Usar la Action Reutilizable (Recomendado)

```yaml
name: CI Pipeline

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./.github/actions/syntegrity-dagger
        with:
          pipeline: go-kit
          stage: build
          version: v1.0.0
```

### Opción 2: Descarga Manual

```yaml
- name: Download Syntegrity Dagger
  run: |
    curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/latest/download/syntegrity-dagger-linux-amd64 -o syntegrity-dagger
    chmod +x syntegrity-dagger

- name: Run Pipeline
  run: |
    ./syntegrity-dagger --pipeline go-kit --step build
```

### Ventajas de GitHub Actions

- **Auto-detección**: El binario detecta automáticamente que está en GitHub Actions
- **Autenticación**: Usa `GITHUB_TOKEN` automáticamente
- **Variables**: Acceso a `GITHUB_REF`, `GITHUB_SHA`, etc.
- **Caché**: GitHub Actions cachea el binario automáticamente

Ver [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) para más detalles.

---

## 📊 Comparación de Entornos

| Característica | Local | On-Premise | GitHub Actions |
|---------------|-------|------------|----------------|
| **Docker requerido** | Opcional (`--local`) | Sí | Sí |
| **Detección automática** | Manual (`--local`) | Automática | Automática |
| **Autenticación Git** | SSH por defecto | Configurable | HTTPS por defecto |
| **Variables de entorno** | Manual | Manual | Automáticas |
| **Caché** | No | Depende del CI | Automático |
| **Logs estructurados** | Básicos | Básicos | Annotations |
| **Artifacts** | Manual | Depende del CI | Automático |

---

## 🔧 Troubleshooting

### Problema: No detecta el entorno correctamente

**Solución**: Forzar el modo con flags:

```bash
# Forzar modo local
syntegrity-dagger --pipeline go-kit --local

# Forzar modo CI (sin --local)
syntegrity-dagger --pipeline go-kit
```

### Problema: Docker no disponible en on-premise

**Solución**: Usar modo local o instalar Docker:

```bash
# Opción 1: Usar modo local
syntegrity-dagger --pipeline go-kit --local

# Opción 2: Instalar Docker
# (depende de tu distribución)
```

### Problema: Autenticación Git diferente por entorno

**Solución**: Configurar explícitamente:

```bash
# Local (SSH)
syntegrity-dagger --pipeline go-kit --git-auth ssh

# CI/On-Premise (HTTPS)
syntegrity-dagger --pipeline go-kit --git-auth https
```

### Problema: Variables de entorno no disponibles

**Solución**: Pasar variables explícitamente:

```bash
# On-Premise
export REGISTRY_USERNAME="user"
export REGISTRY_PASSWORD="pass"
syntegrity-dagger --pipeline go-kit --step push

# O usar archivo de configuración
syntegrity-dagger --pipeline go-kit --config .syntegrity-dagger.yml
```

---

## 📝 Mejores Prácticas

### 1. Usar Configuración por Entorno

Crea archivos de configuración específicos:

```bash
# Local
.syntegrity-dagger.local.yml

# On-Premise
.syntegrity-dagger.onpremise.yml

# GitHub Actions
.syntegrity-dagger.yml
```

### 2. Versionar el Binario

No uses siempre `latest`:

```bash
# Local
SYNTEGRITY_DAGGER_VERSION="v1.0.0"

# On-Premise
export SYNTEGRITY_DAGGER_VERSION="v1.0.0"

# GitHub Actions
env:
  SYNTEGRITY_DAGGER_VERSION: "v1.0.0"
```

### 3. Manejar Secrets Correctamente

```bash
# ❌ No hacer
syntegrity-dagger --pipeline go-kit --env "password=secret"

# ✅ Hacer
export REGISTRY_PASSWORD="secret"
syntegrity-dagger --pipeline go-kit
```

### 4. Usar Caché cuando sea Posible

```yaml
# GitHub Actions - automático
# On-Premise - configurar según tu CI
```

---

## 🔗 Recursos Adicionales

- [Guía de Integración](INTEGRATION_GUIDE.md) - Integración específica con GitHub Actions
- [Guía de Configuración](CONFIGURATION.md) - Opciones de configuración
- [Ejemplos](../examples/) - Ejemplos para cada entorno


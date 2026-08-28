# 🏠 Uso Local - Shipwright

Esta guía explica cómo usar Shipwright localmente para desarrollar, testear y ejecutar pipelines sin necesidad de Docker o servicios en la nube.

## 📋 Tabla de Contenidos

- [Detección Automática](#detección-automática)
- [Ejecución Local](#ejecución-local)
- [Ejecutores Disponibles](#ejecutores-disponibles)
- [Ejemplos de Uso](#ejemplos-de-uso)
- [Configuración Local](#configuración-local)
- [Troubleshooting](#troubleshooting)

---

## 🔍 Detección Automática

Shipwright detecta automáticamente si está ejecutándose localmente o en CI/CD:

- **Local**: Si no se detectan variables de entorno de CI/CD, usa ejecución nativa por defecto
- **CI/CD**: Si se detectan variables de CI/CD (GITHUB_ACTIONS, GITLAB_CI, etc.), usa Docker/Dagger

### Forzar Ejecución Local

Puedes forzar ejecución local usando el flag `--local`:

```bash
./shipwright --local --pipeline go-service
```

### Seleccionar Ejecutor Manualmente

Puedes especificar el ejecutor usando el flag `--executor`:

```bash
# Usar ejecutor nativo (sin Docker)
./shipwright --executor native --pipeline go-service

# Usar ejecutor Docker (requiere Docker)
./shipwright --executor docker --pipeline go-service
```

---

## 🚀 Ejecución Local

### Requisitos

Para ejecución local nativa (sin Docker):

- ✅ Go 1.25.5 o superior instalado
- ✅ Proyecto Go con `go.mod`
- ⚠️ Opcional: `golangci-lint` para linting avanzado
- ⚠️ Opcional: `govulncheck` para análisis de vulnerabilidades

### Ejecutar Pipeline Completo

```bash
# Ejecución automática (detecta local y usa native)
./shipwright --pipeline go-service

# Forzar ejecución local
./shipwright --local --pipeline go-service

# Con opciones específicas
./shipwright --local \
  --pipeline go-service \
  --coverage 95 \
  --env dev
```

### Ejecutar Step Individual

```bash
# Ejecutar solo tests
./shipwright --local --pipeline go-service --step test

# Ejecutar solo build
./shipwright --local --pipeline go-service --step build

# Ejecutar solo lint
./shipwright --local --pipeline go-service --step lint
```

---

## ⚙️ Ejecutores Disponibles

### Native Executor (Recomendado para Local)

Ejecuta comandos Go nativos sin contenedores:

- ✅ **Más rápido**: No requiere Docker
- ✅ **Menos recursos**: No necesita contenedores
- ✅ **Mismo entorno**: Usa tu instalación local de Go
- ⚠️ **Limitaciones**: Requiere herramientas instaladas localmente

**Pasos soportados:**
- `setup`: `go mod download` y `go mod tidy`
- `build`: `go build ./...`
- `test`: `go test -race ./...`
- `lint`: `go vet`, `go fmt`, `golangci-lint` (si está disponible)
- `security`: `govulncheck` (si está disponible)

### Docker Executor

Ejecuta en contenedores Docker usando Dagger:

- ✅ **Aislamiento**: Entorno consistente
- ✅ **Reproducible**: Mismo entorno que CI/CD
- ⚠️ **Requiere Docker**: Necesita Docker instalado y corriendo

---

## 📝 Ejemplos de Uso

### Desarrollo Local Típico

```bash
# 1. Setup inicial
./shipwright --local --pipeline go-service --step setup

# 2. Ejecutar tests
./shipwright --local --pipeline go-service --step test

# 3. Build
./shipwright --local --pipeline go-service --step build

# 4. Lint
./shipwright --local --pipeline go-service --step lint
```

### Pipeline Completo Local

```bash
# Ejecutar todos los steps en orden
./shipwright --local \
  --pipeline go-service \
  --coverage 90 \
  --env dev
```

### Solo Tests con Coverage

```bash
./shipwright --local \
  --pipeline go-service \
  --step test \
  --coverage 95
```

### Verificar Antes de Commit

```bash
# Ejecutar lint y tests
./shipwright --local --pipeline go-service --step lint
./shipwright --local --pipeline go-service --step test
```

---

## ⚙️ Configuración Local

### Archivo de Configuración

Crea un archivo `.shipwright.yml` en la raíz de tu proyecto:

```yaml
pipeline:
  name: go-service
  steps:
    - setup
    - build
    - test
    - lint
  coverage: 90.0
  go_version: "1.25.5"

environment: dev

logging:
  level: info
  format: json
```

### Variables de Entorno

Puedes configurar usando variables de entorno con prefijo `SHIPWRIGHT_`:

```bash
export SHIPWRIGHT_PIPELINE_COVERAGE=95.0
export SHIPWRIGHT_PIPELINE_GO_VERSION=1.25.5
export SHIPWRIGHT_ENVIRONMENT=dev

./shipwright --local --pipeline go-service
```

---

## 🔧 Troubleshooting

### Error: "not a Go project"

**Problema**: El ejecutor no encuentra `go.mod`

**Solución**: Asegúrate de estar en el directorio raíz del proyecto Go:

```bash
cd /path/to/your/go/project
./shipwright --local --pipeline go-service
```

### Error: "golangci-lint not available"

**Problema**: `golangci-lint` no está instalado

**Solución**: Instala golangci-lint o el step de lint usará solo `go vet` y `go fmt`:

```bash
# Instalar golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Error: "govulncheck not available"

**Problema**: `govulncheck` no está instalado

**Solución**: Instala govulncheck o el step de security será más limitado:

```bash
# Instalar govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### Error: `cargo-tarpaulin` falla por falta de memoria (OOM) en Colima

**Problema**: El step de coverage de `providers/rust` (`cargo tarpaulin --engine llvm`) muere sin mensaje claro, o el contenedor Dagger se cierra abruptamente, al ejecutar `make test-integration` localmente sobre Colima sobre macOS.

**Causa**: `tarpaulin` con `--engine llvm` instrumenta cada crate del workspace y puede superar la memoria asignada por defecto a la VM de Colima, especialmente en workspaces con varias crates o dependencias pesadas.

**Solución**: Aumenta la memoria de la VM de Colima antes de correr la suite:

```bash
colima stop
colima start --memory 8   # ajusta según los recursos disponibles en tu máquina
```

Es una limitación conocida del entorno local (Colima + Dagger + tarpaulin), no del pipeline: en CI, los runners de GitHub Actions ya cuentan con memoria suficiente y no se ve este problema.

### Ejecución Lenta

**Problema**: La ejecución local es más lenta de lo esperado

**Soluciones**:
1. Usa `--executor native` explícitamente para evitar detección automática
2. Verifica que no esté intentando usar Docker
3. Asegúrate de tener caché de módulos Go habilitado

### Coverage Threshold No Se Cumple

**Problema**: Los tests fallan por coverage insuficiente

**Solución**: Ajusta el threshold o mejora el coverage:

```bash
# Reducir threshold temporalmente
./shipwright --local --pipeline go-service --coverage 80

# O mejorar el coverage del código
```

---

## 💡 Mejores Prácticas

1. **Usa ejecución local para desarrollo**: Más rápido y no requiere Docker
2. **Usa Docker executor para validación**: Antes de hacer push, valida con Docker
3. **Configura coverage threshold**: Ajusta según las necesidades del proyecto
4. **Instala herramientas opcionales**: `golangci-lint` y `govulncheck` mejoran la calidad
5. **Usa archivo de configuración**: `.shipwright.yml` para configuración persistente

---

## 🎯 Comparación: Local vs CI/CD

| Característica | Local (Native) | CI/CD (Docker) |
|---------------|----------------|----------------|
| **Velocidad** | ⚡ Muy rápido | 🐢 Más lento |
| **Recursos** | 💚 Bajo consumo | 💛 Mayor consumo |
| **Aislamiento** | ⚠️ Usa entorno local | ✅ Entorno aislado |
| **Reproducibilidad** | ⚠️ Depende del entorno | ✅ 100% reproducible |
| **Requisitos** | Go instalado | Docker + Dagger |

**Recomendación**: Usa local para desarrollo diario, CI/CD para validación final.



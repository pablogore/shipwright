# 🚀 Proceso de Release

Este documento describe cómo funciona el proceso de release de Syntegrity Dagger y cómo los binarios se publican para que los servicios puedan consumirlos.

## 📋 Resumen

Cuando se hace un release, el pipeline:

1. **Compila** binarios para múltiples plataformas (Linux, macOS, Windows)
2. **Publica** archivos comprimidos (tar.gz, zip) vía GoReleaser
3. **Extrae y renombra** los binarios con nombres estándar
4. **Publica** binarios directos (sin comprimir) para descarga directa

## 🎯 Binarios Publicados

Cada release incluye dos tipos de archivos:

### 1. Archivos Comprimidos (via GoReleaser)
- `syntegrity-dagger_1.0.0_linux_amd64.tar.gz`
- `syntegrity-dagger_1.0.0_linux_arm64.tar.gz`
- `syntegrity-dagger_1.0.0_darwin_amd64.tar.gz`
- `syntegrity-dagger_1.0.0_darwin_arm64.tar.gz`
- `syntegrity-dagger_1.0.0_windows_amd64.zip`

### 2. Binarios Directos (para CI/CD y uso local)
- `syntegrity-dagger-linux-amd64`
- `syntegrity-dagger-linux-arm64`
- `syntegrity-dagger-darwin-amd64`
- `syntegrity-dagger-darwin-arm64`
- `syntegrity-dagger-windows-amd64.exe`

## 📦 Cómo Consumir los Binarios

### Desde GitHub Actions

```yaml
- name: Download Syntegrity Dagger
  run: |
    curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/download/v1.0.0/syntegrity-dagger-linux-amd64 -o syntegrity-dagger
    chmod +x syntegrity-dagger
```

### Desde Local

```bash
# Descargar binario
curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/download/v1.0.0/syntegrity-dagger-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/') -o syntegrity-dagger
chmod +x syntegrity-dagger
sudo mv syntegrity-dagger /usr/local/bin/
```

### Desde On-Premise CI/CD

```bash
# En Jenkins, GitLab CI, etc.
curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/download/v1.0.0/syntegrity-dagger-linux-amd64 -o syntegrity-dagger
chmod +x syntegrity-dagger
```

### Usando "latest"

Para obtener siempre la última versión:

```bash
curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/latest/download/syntegrity-dagger-linux-amd64 -o syntegrity-dagger
chmod +x syntegrity-dagger
```

## 🔄 Proceso de Release Automático

### Trigger de Release

El release se activa cuando:

1. **Push a main**: Crea un release automático con versión patch incrementada
2. **Tag manual**: Crea un release con la versión del tag (ej: `v1.0.0`)

### Workflow de Release

1. **Validación**: Determina la versión y genera changelog
2. **Build y Test**: Compila y ejecuta tests
3. **GoReleaser**: Publica archivos comprimidos y Docker images
4. **Extracción de Binarios**: Extrae binarios de los archivos comprimidos
5. **Publicación Directa**: Sube binarios directos al release
6. **Resumen**: Muestra información del release

## 📝 URLs de Descarga

### Versión Específica

```
https://github.com/getsyntegrity/syntegrity-dagger/releases/download/v1.0.0/syntegrity-dagger-linux-amd64
```

### Última Versión

```
https://github.com/getsyntegrity/syntegrity-dagger/releases/latest/download/syntegrity-dagger-linux-amd64
```

### Patrón de URL

```
https://github.com/getsyntegrity/syntegrity-dagger/releases/download/{TAG}/syntegrity-dagger-{OS}-{ARCH}
```

Donde:
- `{TAG}`: Versión del release (ej: `v1.0.0` o `latest`)
- `{OS}`: Sistema operativo (`linux`, `darwin`, `windows`)
- `{ARCH}`: Arquitectura (`amd64`, `arm64`)

## ✅ Verificación

Para verificar que un binario se descargó correctamente:

```bash
# Verificar que es ejecutable
./syntegrity-dagger --version

# Verificar checksum (si está disponible)
sha256sum syntegrity-dagger-linux-amd64
# Comparar con checksums.txt en el release
```

## 🔗 Referencias

- [Guía de Integración](INTEGRATION_GUIDE.md) - Cómo integrar en servicios
- [Guía de Entornos](DEPLOYMENT_ENVIRONMENTS.md) - Uso en diferentes entornos
- [Ejemplos](../examples/) - Ejemplos de uso

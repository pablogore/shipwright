# 🚀 Proceso de Release

Este documento describe cómo funciona el proceso de release de Shipwright y cómo los binarios se publican para que los servicios puedan consumirlos.

## 📋 Resumen

Cuando se hace un release, el pipeline:

1. **Compila** binarios para múltiples plataformas (Linux, macOS, Windows) vía Dagger
2. **Empaqueta** archivos comprimidos (tar.gz, zip) y genera `checksums.txt`, también vía Dagger
3. **Publica** el set completo de artefactos (binarios directos + archivos comprimidos + checksums.txt) al GitHub Release vía GitHub CLI

## 🎯 Binarios Publicados

Cada release incluye dos tipos de archivos:

### 1. Archivos Comprimidos (empaquetados por Dagger)
- `shipwright_1.0.0_linux_amd64.tar.gz`
- `shipwright_1.0.0_linux_arm64.tar.gz`
- `shipwright_1.0.0_darwin_amd64.tar.gz`
- `shipwright_1.0.0_darwin_arm64.tar.gz`
- `shipwright_1.0.0_windows_amd64.zip`
- `shipwright_1.0.0_windows_arm64.zip`

### 2. Binarios Directos (para CI/CD y uso local)
- `shipwright-linux-amd64`
- `shipwright-linux-arm64`
- `shipwright-darwin-amd64`
- `shipwright-darwin-arm64`
- `shipwright-windows-amd64.exe`
- `shipwright-windows-arm64.exe`

## 📦 Cómo Consumir los Binarios

### Desde GitHub Actions

```yaml
- name: Download Shipwright
  run: |
    curl -L https://github.com/pablogore/shipwright/releases/download/v1.0.0/shipwright-linux-amd64 -o shipwright
    chmod +x shipwright
```

### Desde Local

```bash
# Descargar binario
curl -L https://github.com/pablogore/shipwright/releases/download/v1.0.0/shipwright-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/') -o shipwright
chmod +x shipwright
sudo mv shipwright /usr/local/bin/
```

### Desde On-Premise CI/CD

```bash
# En Jenkins, GitLab CI, etc.
curl -L https://github.com/pablogore/shipwright/releases/download/v1.0.0/shipwright-linux-amd64 -o shipwright
chmod +x shipwright
```

### Usando "latest"

Para obtener siempre la última versión:

```bash
curl -L https://github.com/pablogore/shipwright/releases/latest/download/shipwright-linux-amd64 -o shipwright
chmod +x shipwright
```

## 🔄 Proceso de Release Automático

### Trigger de Release

El release se activa cuando:

1. **Push a main**: Crea un release automático con versión patch incrementada
2. **Tag manual**: Crea un release con la versión del tag (ej: `v1.0.0`)

### Workflow de Release

1. **Validación**: Determina la versión y genera changelog
2. **Build y Test**: Compila y ejecuta tests (workflow de CI, previo al release)
3. **Dagger**: Compila la matriz de plataformas, empaqueta archivos comprimidos, genera `checksums.txt` y verifica el contrato de integridad antes de publicar nada
4. **Publicación**: Crea (o verifica, si ya existe) el GitHub Release y sube el set completo de artefactos vía GitHub CLI
5. **Verificación Post-Publicación**: Descarga el set completo publicado y valida checksums + ejecución del binario
6. **Resumen**: Muestra información del release

## 📝 URLs de Descarga

### Versión Específica

```
https://github.com/pablogore/shipwright/releases/download/v1.0.0/shipwright-linux-amd64
```

### Última Versión

```
https://github.com/pablogore/shipwright/releases/latest/download/shipwright-linux-amd64
```

### Patrón de URL

```
https://github.com/pablogore/shipwright/releases/download/{TAG}/shipwright-{OS}-{ARCH}
```

Donde:
- `{TAG}`: Versión del release (ej: `v1.0.0` o `latest`)
- `{OS}`: Sistema operativo (`linux`, `darwin`, `windows`)
- `{ARCH}`: Arquitectura (`amd64`, `arm64`)

## ✅ Verificación

Para verificar que un binario se descargó correctamente:

```bash
# Verificar que es ejecutable
./shipwright --version

# Verificar checksum (si está disponible)
sha256sum shipwright-linux-amd64
# Comparar con checksums.txt en el release
```

## 🔗 Referencias

- [Guía de Integración](INTEGRATION_GUIDE.md) - Cómo integrar en servicios
- [Guía de Entornos](DEPLOYMENT_ENVIRONMENTS.md) - Uso en diferentes entornos
- [Ejemplos](../examples/) - Ejemplos de uso

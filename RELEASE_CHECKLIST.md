# ✅ Checklist para Release de Prueba

## Pre-requisitos

- [x] `.goreleaser.yml` configurado correctamente
- [x] Workflow de release (`release.yml`) actualizado
- [x] GitHub Action reutilizable creada
- [ ] Cambios commiteados
- [ ] Tag de prueba creado

## Pasos para Release de Prueba

### 1. Commitear cambios actuales

```bash
# Ver qué archivos han cambiado
git status

# Agregar archivos importantes
git add .goreleaser.yml
git add .github/workflows/release.yml
git add .github/actions/
git add docs/
git add examples/
git add MEJORAS_RECOMENDADAS.md

# Commitear
git commit -m "feat: add release workflow with direct binary publishing"
```

### 2. Crear tag de prueba

```bash
# Crear un tag de prueba (ej: v0.1.0-test)
git tag -a v0.1.0-test -m "Test release: verify binary publishing"

# Verificar que el tag se creó
git tag -l | grep test
```

### 3. Hacer push del tag

```bash
# Push del tag (esto activará el workflow de release)
git push origin v0.1.0-test
```

### 4. Verificar el workflow

1. Ir a GitHub → Actions
2. Buscar el workflow "Release Pipeline"
3. Verificar que se ejecute correctamente
4. Revisar los logs de cada job

### 5. Verificar el release

1. Ir a GitHub → Releases
2. Buscar el release `v0.1.0-test`
3. Verificar que incluya:
   - ✅ Archivos comprimidos (tar.gz, zip)
   - ✅ Binarios directos (`syntegrity-dagger-linux-amd64`, etc.)
   - ✅ Checksums
   - ✅ Changelog

### 6. Probar descarga del binario

```bash
# Probar descarga del binario directo
curl -L https://github.com/getsyntegrity/syntegrity-dagger/releases/download/v0.1.0-test/syntegrity-dagger-linux-amd64 -o syntegrity-dagger-test
chmod +x syntegrity-dagger-test
./syntegrity-dagger-test --version
```

## Qué verificar

- [ ] El workflow se ejecuta sin errores
- [ ] Los binarios se publican con nombres correctos
- [ ] Los binarios son ejecutables
- [ ] La versión del binario es correcta
- [ ] Los binarios funcionan correctamente

## Si algo falla

1. Revisar los logs del workflow
2. Verificar que `.goreleaser.yml` esté correcto
3. Verificar permisos del `GITHUB_TOKEN`
4. Verificar que el runner tenga acceso a Docker (si es necesario)

## Limpiar después de la prueba

```bash
# Eliminar el tag local
git tag -d v0.1.0-test

# Eliminar el tag remoto
git push origin --delete v0.1.0-test

# Eliminar el release en GitHub (desde la UI)
```


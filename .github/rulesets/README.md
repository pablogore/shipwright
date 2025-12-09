# 🛡️ Branch Protection Rulesets

Este directorio contiene las reglas de protección de ramas para el repositorio.

## 📋 Reglas Configuradas

### `branch-protection-rules.json`

Protege las ramas `main` y `develop` con las siguientes reglas:

- ✅ **Require Pull Request**: Requiere PR antes de mergear
- ✅ **Required Approvals**: 1 aprobación requerida
- ✅ **Dismiss Stale Reviews**: Las aprobaciones se descartan cuando hay nuevos commits
- ✅ **Required Status Checks**: Deben pasar los siguientes checks:
  - `build` - Compilación del proyecto
  - `test` - Ejecución de tests
  - `security` - Escaneo de seguridad
- ✅ **Linear History**: Requiere historial lineal (solo squash/rebase)
- ✅ **No Force Push**: Previene force push
- ✅ **No Deletion**: Previene eliminación de ramas

## 🎯 Flujos Protegidos

### 1. Feature → Develop
- PR de `feature/*` a `develop`
- Requiere: 1 aprobación + CI success (build, test, security)

### 2. Develop → Main
- PR de `develop` a `main`
- Requiere: 1 aprobación + CI success (build, test, security)

### 3. Hotfix → Main
- PR de `hotfix/*` a `main`
- Requiere: 1 aprobación + CI success (build, test, security)

## 🚀 Aplicar las Reglas

### Opción 1: Script Automatizado (Recomendado)

```bash
./scripts/apply-branch-protection.sh
```

### Opción 2: GitHub CLI Manual

```bash
gh api repos/getsyntegrity/syntegrity-dagger/rulesets \
  --method POST \
  --input .github/rulesets/branch-protection-rules.json
```

### Opción 3: API de GitHub

```bash
curl -X POST \
  -H "Authorization: token YOUR_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/getsyntegrity/syntegrity-dagger/rulesets \
  -d @.github/rulesets/branch-protection-rules.json
```

## 🔍 Verificar las Reglas

1. Ve a: https://github.com/getsyntegrity/syntegrity-dagger/settings/rules
2. Verifica que las reglas estén aplicadas a `main` y `develop`

## ⚠️ Importante

- Los nombres de los status checks (`build`, `test`, `security`) deben coincidir exactamente con los nombres de los jobs en `.github/workflows/ci.yml`
- Si cambias los nombres de los jobs, actualiza también este ruleset
- Las reglas se aplican inmediatamente después de ejecutar el script

## 📚 Documentación Relacionada

- [REQUIRE_CI_SUCCESS.md](../../docs/REQUIRE_CI_SUCCESS.md) - Documentación completa sobre status checks
- [BRANCH_PROTECTION_RULES.md](../../docs/BRANCH_PROTECTION_RULES.md) - Reglas detalladas de protección


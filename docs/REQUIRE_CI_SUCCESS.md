# 🛡️ Requerir Éxito del Pipeline Antes de Mergear

Este documento explica cómo configurar las reglas de protección de ramas para que **no se pueda hacer merge/squash sin que el pipeline termine exitosamente**.

## 📋 Configuración Actual

### Workflows que Deben Pasar

Los siguientes jobs del workflow `ci.yml` deben pasar exitosamente antes de permitir merge:

1. **`build`** - Compilación del proyecto
2. **`test`** - Ejecución de tests
3. **`security`** - Escaneo de vulnerabilidades (solo para main/develop/PRs)

### Ruleset de Protección

El archivo `org_rules_with_required_checks.json` contiene la configuración que:

- ✅ Requiere que los status checks pasen antes de mergear
- ✅ Bloquea merge si hay fallos en el pipeline
- ✅ Aplica a las ramas `main` y `develop`
- ✅ Permite solo `squash` y `rebase` como métodos de merge
- ✅ Requiere 1 aprobación antes de mergear

## 🚀 Aplicar el Ruleset

### Opción 1: Usando GitHub CLI

```bash
# Aplicar a nivel de organización (todos los repos)
gh api orgs/getsyntegrity/rulesets \
  --method POST \
  --input org_rules_with_required_checks.json

# O aplicar a un repositorio específico
gh api repos/getsyntegrity/syntegrity-dagger/rulesets \
  --method POST \
  --input org_rules_with_required_checks.json
```

### Opción 2: Usando la API de GitHub

```bash
curl -X POST \
  -H "Authorization: token YOUR_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/orgs/getsyntegrity/rulesets \
  -d @org_rules_with_required_checks.json
```

### Opción 3: Configuración Manual en GitHub

1. Ve a **Settings** → **Rules** → **Rulesets**
2. Crea un nuevo ruleset
3. Configura:
   - **Target branches**: `main`, `develop`
   - **Rules**:
     - ✅ Require pull request before merging
     - ✅ Require status checks to pass before merging
     - ✅ Require branches to be up to date before merging
     - ✅ Require linear history
   - **Required status checks**:
     - `build`
     - `test`
     - `security`
   - **Allowed merge methods**: Solo `squash` y `rebase`

## 🔍 Verificar Status Checks

Los status checks aparecen en:
- Pull Requests: Sección "Checks" en la parte inferior
- Branch Protection: Settings → Branches → [branch] → Require status checks

### Nombres de Status Checks

Los nombres de los jobs en el workflow deben coincidir exactamente con los contextos en el ruleset:

```yaml
# En ci.yml
jobs:
  build:    # ← Este nombre se usa como status check
    runs-on: ubuntu-latest
    # ...
  
  test:     # ← Este nombre se usa como status check
    runs-on: ubuntu-latest
    # ...
  
  security: # ← Este nombre se usa como status check
    runs-on: ubuntu-latest
    # ...
```

## 🚫 Qué Bloquea el Merge

El merge será **bloqueado** si:

- ❌ El job `build` falla
- ❌ El job `test` falla
- ❌ El job `security` falla (cuando aplica)
- ❌ La rama no está actualizada con la base branch
- ❌ No hay suficientes aprobaciones (1 requerida)
- ❌ Hay conflictos de merge sin resolver

## ✅ Qué Permite el Merge

El merge será **permitido** solo si:

- ✅ Todos los status checks requeridos pasan
- ✅ La rama está actualizada con la base branch
- ✅ Hay al menos 1 aprobación
- ✅ No hay conflictos de merge
- ✅ Se usa `squash` o `rebase` como método de merge

## 🔧 Troubleshooting

### Los status checks no aparecen

1. Verifica que el workflow se ejecute en PRs:
   ```yaml
   on:
     pull_request:
       branches: [ main, develop ]
   ```

2. Verifica que los jobs tengan nombres consistentes
3. Espera a que el workflow complete al menos una vez

### El merge está bloqueado pero los checks pasan

1. Verifica que la rama esté actualizada: `git pull origin develop`
2. Verifica que no haya conflictos de merge
3. Verifica que haya suficientes aprobaciones

### Quiero hacer merge de emergencia

Para emergencias, puedes:
1. Usar el bypass (si está configurado para tu usuario)
2. Temporalmente deshabilitar el ruleset
3. Contactar a un administrador

**⚠️ Nota**: Los bypasses deben usarse solo en casos de emergencia real.

## 📝 Ejemplo de Configuración Completa

```json
{
  "name": "main rule with required status checks",
  "target": "branch",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": ["refs/heads/main", "refs/heads/develop"]
    }
  },
  "rules": [
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": true,
        "required_status_checks": [
          {"context": "build"},
          {"context": "test"},
          {"context": "security"}
        ]
      }
    },
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 1,
        "allowed_merge_methods": ["squash", "rebase"]
      }
    }
  ]
}
```

## 🔄 Actualizar Status Checks Requeridos

Si necesitas agregar o quitar status checks:

1. Edita `org_rules_with_required_checks.json`
2. Actualiza la lista en `required_status_checks`
3. Aplica el ruleset nuevamente usando GitHub CLI o API
4. Los cambios se aplican inmediatamente

## 📚 Referencias

- [GitHub Branch Protection Rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
- [GitHub Organization Rulesets API](https://docs.github.com/en/rest/orgs/rules)
- [Required Status Checks](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches#require-status-checks-before-merging)


# 🔒 GitHub Organization Rulesets

Este documento explica los archivos JSON de reglas de protección de ramas que se encuentran en el repositorio.

## 📋 ¿Qué son estos archivos?

Estos archivos JSON definen **reglas de protección de ramas** (Branch Protection Rules) para GitHub Organizations usando la [GitHub Organization Rulesets API](https://docs.github.com/en/rest/orgs/rules).

## 🎯 Propósito

Las reglas de protección de ramas ayudan a:
- **Proteger ramas importantes** (main, develop) de cambios no deseados
- **Enforzar políticas de merge** (squash, rebase)
- **Requerir pull requests** antes de mergear
- **Prevenir eliminación** de ramas protegidas
- **Mantener historial lineal** (linear history)

## 📁 Archivos en el Repositorio

### 1. `org_rules_update.json`
**Reglas principales para la organización**

- **Aplica a**: Todas las ramas `main` y `develop` en todos los repositorios (`~ALL`)
- **Reglas**:
  - Prevenir eliminación de ramas
  - Requerir historial lineal
  - Permitir actualizaciones (force push deshabilitado)
  - Requerir pull requests (0 aprobaciones requeridas)
  - Métodos de merge permitidos: squash, rebase

### 2. `org_ruleset_update.json`
**Similar a org_rules_update pero con regla adicional**

- Incluye regla `non_fast_forward` (previene force push)
- Resto de reglas similares a `org_rules_update.json`

### 3. `simplified_ruleset.json` / `simple_org_ruleset.json`
**Reglas simplificadas para desarrollo**

- Solo incluye reglas esenciales:
  - Historial lineal requerido
  - Pull requests requeridos
- Sin reglas de eliminación o actualización

### 4. `org_ruleset_individual_dev.json`
**Reglas con bypass para desarrolladores individuales**

- Similar a `org_rules_update.json`
- Incluye `bypass_actors` para permitir que un usuario específico (ID: 173299984) pueda bypassear las reglas

### 5. `repo_specific_rules.json` / `repo_override_rules.json`
**Reglas específicas para repositorios individuales**

- Permiten sobrescribir reglas de organización
- Útiles para casos especiales o repositorios que necesitan reglas diferentes

## 🔧 Estructura de un Ruleset

```json
{
  "name": "Nombre de la regla",
  "target": "branch",  // o "tag"
  "enforcement": "active",  // o "disabled", "evaluate"
  "conditions": {
    "ref_name": {
      "include": ["refs/heads/main", "refs/heads/develop"],
      "exclude": []
    },
    "repository_name": {
      "include": ["~ALL"],  // Todos los repos
      "exclude": []
    }
  },
  "rules": [
    {
      "type": "deletion"  // Previene eliminación
    },
    {
      "type": "required_linear_history"  // Requiere historial lineal
    },
    {
      "type": "update"  // Previene force push
    },
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 0,
        "allowed_merge_methods": ["squash", "rebase"]
      }
    }
  ],
  "bypass_actors": []  // Usuarios que pueden bypassear
}
```

## 📝 Tipos de Reglas Disponibles

### `deletion`
Previene la eliminación de ramas protegidas.

### `required_linear_history`
Requiere que el historial de commits sea lineal (sin merge commits).

### `update` / `non_fast_forward`
Previene force push y actualizaciones no fast-forward.

### `pull_request`
Requiere pull requests antes de mergear. Parámetros:
- `required_approving_review_count`: Número de aprobaciones requeridas
- `allowed_merge_methods`: Métodos permitidos (`squash`, `rebase`, `merge`)
- `require_code_owner_review`: Requiere revisión de code owners
- `dismiss_stale_reviews_on_push`: Descartar reviews obsoletos

## 🚀 Cómo Aplicar estas Reglas

### Usando GitHub CLI

```bash
# Aplicar reglas a nivel de organización
gh api orgs/{org}/rulesets \
  --method POST \
  --input org_rules_update.json

# Aplicar reglas a un repositorio específico
gh api repos/{owner}/{repo}/rulesets \
  --method POST \
  --input repo_specific_rules.json
```

### Usando la API de GitHub

```bash
curl -X POST \
  -H "Authorization: token YOUR_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  https://api.github.com/orgs/{org}/rulesets \
  -d @org_rules_update.json
```

### Usando Terraform

```hcl
resource "github_repository_ruleset" "main" {
  name        = "main rule"
  target      = "branch"
  enforcement = "active"
  
  conditions {
    ref_name {
      include = ["refs/heads/main", "refs/heads/develop"]
    }
  }
  
  rules {
    deletion = true
    required_linear_history = true
    pull_request {
      required_approving_review_count = 0
      allowed_merge_methods = ["squash", "rebase"]
    }
  }
}
```

## 📊 Comparación de Archivos

| Archivo | Eliminación | Historial Lineal | Force Push | PR Requerido | Bypass |
|---------|-------------|------------------|------------|--------------|--------|
| `org_rules_update.json` | ✅ | ✅ | ✅ | ✅ (0 aprob) | ❌ |
| `org_ruleset_update.json` | ✅ | ✅ | ✅ | ✅ (0 aprob) | ❌ |
| `simplified_ruleset.json` | ❌ | ✅ | ❌ | ✅ (0 aprob) | ❌ |
| `org_ruleset_individual_dev.json` | ✅ | ✅ | ✅ | ✅ (0 aprob) | ✅ |

## ⚠️ Consideraciones

1. **Orden de aplicación**: Las reglas de repositorio sobrescriben las de organización
2. **Bypass actors**: Solo deben usarse en casos excepcionales
3. **Testing**: Probar reglas en un repositorio de prueba antes de aplicar a toda la organización
4. **Documentación**: Mantener documentadas las razones de cada regla

## 🔗 Referencias

- [GitHub Organization Rulesets API](https://docs.github.com/en/rest/orgs/rules)
- [Repository Rulesets API](https://docs.github.com/en/rest/repos/rules)
- [Branch Protection Rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)

## 💡 Recomendaciones

1. **Empezar simple**: Usar `simplified_ruleset.json` para empezar
2. **Ir agregando reglas**: Añadir más protección según necesidad
3. **Documentar cambios**: Explicar por qué se agregan nuevas reglas
4. **Revisar periódicamente**: Asegurar que las reglas siguen siendo relevantes


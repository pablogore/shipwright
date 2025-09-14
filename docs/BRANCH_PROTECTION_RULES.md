# Reglas de Protección de Ramas - Syntegrity Dagger

## 🛡️ Configuración de Protección de Ramas

### Rama `main` (Producción)

#### ✅ Reglas Obligatorias
- **Require a pull request before merging**
  - Require approvals: `2`
  - Dismiss stale PR approvals when new commits are pushed: `✅`
  - Require review from code owners: `✅`

- **Require status checks to pass before merging**
  - Require branches to be up to date before merging: `✅`
  - Required status checks:
    - `build`
    - `test`
    - `security`
    - `lint`

- **Require conversation resolution before merging**: `✅`

- **Require signed commits**: `✅`

- **Require linear history**: `✅`

- **Include administrators**: `❌` (Incluso admins deben seguir las reglas)

- **Allow force pushes**: `❌`

- **Allow deletions**: `❌`

#### 🔒 Restricciones Adicionales
- **Restrict pushes that create files**: `✅`
- **Restrict pushes that create files larger than**: `10MB`
- **Restrict pushes that create files with extensions**: `.exe, .dll, .so, .dylib`

### Rama `develop` (Integración)

#### ✅ Reglas Obligatorias
- **Require a pull request before merging**
  - Require approvals: `1`
  - Dismiss stale PR approvals when new commits are pushed: `✅`
  - Require review from code owners: `✅`

- **Require status checks to pass before merging**
  - Require branches to be up to date before merging: `✅`
  - Required status checks:
    - `build`
    - `test`
    - `lint`

- **Require conversation resolution before merging**: `✅`

- **Allow force pushes**: `❌`

- **Allow deletions**: `❌`

### Ramas de Feature/Fix

#### ✅ Reglas Básicas
- **Require a pull request before merging**
  - Require approvals: `1`
  - Require review from code owners: `✅`

- **Require status checks to pass before merging**
  - Required status checks:
    - `build`
    - `test`

## 🚨 Reglas Especiales para Archivos Críticos

### Archivos de Workflow (`.github/workflows/*.yml`)

#### Reglas Adicionales
- **Require approval from specific reviewers**: `@pablonqn`
- **Require status checks to pass before merging**:
  - `workflow-validation`
  - `syntax-check`
  - `security-scan`

#### Proceso de Cambio
1. **Crear issue** describiendo el cambio
2. **Obtener aprobación** del equipo antes de empezar
3. **Crear rama** con prefijo `workflow/`
4. **Testing exhaustivo** en ambiente de desarrollo
5. **Code review** obligatorio de 2 personas
6. **Merge solo después** de que todos los checks pasen

### Archivos de Configuración (`go.mod`, `go.sum`)

#### Reglas Adicionales
- **Require approval from maintainers**: `@pablonqn`
- **Require dependency audit**: `✅`
- **Require security scan**: `✅`

#### Proceso de Cambio
1. **Crear issue** justificando el cambio de dependencia
2. **Actualizar documentación** si es necesario
3. **Testing en múltiples versiones** de Go
4. **Verificar compatibilidad** con otros proyectos

## 🔧 Configuración en GitHub

### 1. Configurar Branch Protection Rules

```bash
# Usar GitHub CLI para configurar reglas
gh api repos/:owner/:repo/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["build","test","security","lint"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":2,"dismiss_stale_reviews":true,"require_code_owner_reviews":true}' \
  --field restrictions=null
```

### 2. Configurar CODEOWNERS

```bash
# Crear archivo .github/CODEOWNERS
cat > .github/CODEOWNERS << 'EOF'
# Global owners
* @pablonqn

# Workflow files - require special approval
.github/workflows/ @pablonqn

# Go modules - require maintainer approval
go.mod @pablonqn
go.sum @pablonqn

# Documentation
docs/ @pablonqn
README.md @pablonqn
CHANGELOG.md @pablonqn

# Scripts
scripts/ @pablonqn
EOF
```

### 3. Configurar Required Status Checks

```yaml
# En .github/workflows/ci.yml agregar job de validación
validate-workflow:
  runs-on: ubuntu-latest
  steps:
    - name: Checkout
      uses: actions/checkout@v5
      
    - name: Validate workflow syntax
      run: |
        # Validar sintaxis de workflows
        for workflow in .github/workflows/*.yml; do
          echo "Validating $workflow"
          yamllint "$workflow"
        done
        
    - name: Check for conflicts
      run: |
        # Verificar que no hay conflictos de merge
        if git diff --name-only --diff-filter=U | grep -q .; then
          echo "❌ Merge conflicts detected!"
          git diff --name-only --diff-filter=U
          exit 1
        fi
        echo "✅ No merge conflicts found"
```

## 📋 Checklist de Pre-Merge

### Para Desarrolladores
- [ ] Rama sincronizada con `develop`
- [ ] Todos los tests pasan localmente
- [ ] No hay conflictos de merge
- [ ] Commits siguen conventional commits
- [ ] Documentación actualizada si es necesario
- [ ] Changelog actualizado si es necesario

### Para Reviewers
- [ ] Código sigue las convenciones del proyecto
- [ ] Tests cubren los cambios
- [ ] No hay regresiones
- [ ] Performance no se ve afectada
- [ ] Seguridad no se ve comprometida
- [ ] Documentación es clara y actualizada

## 🚫 Reglas de Bloqueo

### Bloquear Merge si:
- ❌ Hay conflictos de merge sin resolver
- ❌ Tests fallan
- ❌ Coverage disminuye
- ❌ Security scan falla
- ❌ Linting falla
- ❌ No hay approvals suficientes
- ❌ Rama no está actualizada con base branch

### Bloquear Push si:
- ❌ Archivos > 10MB
- ❌ Extensiones de archivo no permitidas
- ❌ Commits no firmados (en main)
- ❌ Force push (en ramas protegidas)

## 🔄 Proceso de Excepción

### Para Emergencias (Hotfixes)
1. **Crear issue** con label `emergency`
2. **Notificar al equipo** inmediatamente
3. **Crear rama** `hotfix/descripcion-urgente`
4. **Merge directo** a main (bypassing algunas reglas)
5. **Backport** a develop inmediatamente
6. **Post-mortem** obligatorio

### Para Cambios Críticos
1. **RFC (Request for Comments)** en el equipo
2. **Aprobación unánime** del equipo
3. **Plan de rollback** documentado
4. **Testing exhaustivo** en staging
5. **Deployment gradual** si es posible

## 📊 Monitoreo y Métricas

### Métricas a Trackear
- **Tiempo promedio de PR**: < 2 días
- **Tasa de conflictos**: < 5% de PRs
- **Tiempo de resolución de conflictos**: < 1 hora
- **Tasa de rollbacks**: < 1% de deployments

### Alertas Automáticas
- **PR abierto > 3 días**: Notificar al autor
- **Conflicto detectado**: Notificar inmediatamente
- **Build fallido**: Notificar al equipo
- **Security scan fallido**: Notificar a maintainers

## 🎓 Training y Onboarding

### Para Nuevos Desarrolladores
1. **Leer documentación** de branching strategy
2. **Configurar entorno** con pre-commit hooks
3. **Hacer PR de prueba** con cambios menores
4. **Shadow code review** con desarrollador senior
5. **Certificación** en proceso de merge

### Para el Equipo
- **Retrospectivas mensuales** sobre conflictos
- **Workshops trimestrales** sobre mejores prácticas
- **Actualización de reglas** basada en experiencia
- **Compartir lecciones aprendidas** en conflictos resueltos

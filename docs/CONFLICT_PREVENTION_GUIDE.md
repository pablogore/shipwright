# Guía de Prevención de Conflictos - Shipwright

## 🎯 Objetivo
Esta guía proporciona un conjunto completo de herramientas, procesos y mejores prácticas para prevenir conflictos de merge en el proyecto Shipwright.

## 📋 Resumen de Implementación

### ✅ **Herramientas Implementadas**

1. **📚 Documentación Completa**
   - `BRANCHING_STRATEGY.md` - Estrategia de branching
   - `BRANCH_PROTECTION_RULES.md` - Reglas de protección de ramas
   - `MERGE_CONFLICT_RESOLUTION.md` - Guía de resolución de conflictos
   - `CONFLICT_PREVENTION_GUIDE.md` - Esta guía

2. **🔧 Scripts de Automatización**
   - `scripts/sync-branch.sh` - Sincronizar rama con develop
   - `scripts/check-conflicts.sh` - Verificar conflictos potenciales
   - `scripts/check-workflow-conflicts.sh` - Verificar conflictos en workflows
   - `scripts/check-go-mod-conflicts.sh` - Verificar conflictos en dependencias
   - `scripts/validate-workflow-syntax.sh` - Validar sintaxis de workflows

3. **🛡️ Protección Automática**
   - `.github/workflows/conflict-detection.yml` - Detección automática de conflictos
   - `.pre-commit-config.yaml` - Hooks de pre-commit
   - `.yamllint` - Configuración de linting para YAML

4. **🔒 Reglas de Protección**
   - Branch protection rules para main y develop
   - CODEOWNERS para archivos críticos
   - Required status checks
   - Mandatory reviews

## 🚀 **Cómo Usar las Herramientas**

### **1. Configuración Inicial**

```bash
# Instalar pre-commit hooks
pip install pre-commit
pre-commit install

# Instalar yamllint para validación de workflows
pip install yamllint

# Hacer scripts ejecutables (ya hecho)
chmod +x scripts/*.sh
```

### **2. Flujo de Trabajo Diario**

#### **Antes de Empezar a Trabajar**
```bash
# Verificar estado del repositorio
./scripts/check-conflicts.sh

# Sincronizar con develop
./scripts/sync-branch.sh
```

#### **Durante el Desarrollo**
```bash
# Verificar conflictos antes de cada commit
./scripts/check-conflicts.sh

# Los pre-commit hooks se ejecutan automáticamente
git add .
git commit -m "feat: nueva funcionalidad"
```

#### **Antes de Crear PR**
```bash
# Verificación final
./scripts/check-conflicts.sh

# Sincronizar una vez más
./scripts/sync-branch.sh

# Crear PR
gh pr create --title "feat: nueva funcionalidad" --body "Descripción del cambio"
```

### **3. Resolución de Conflictos**

#### **Cuando Ocurren Conflictos**
```bash
# 1. Detectar conflictos
git status

# 2. Resolver manualmente o con herramienta
git mergetool

# 3. Verificar resolución
./scripts/check-conflicts.sh

# 4. Completar merge
git add .
git commit -m "resolve: merge conflicts in workflow"
```

## 📊 **Métricas de Éxito**

### **Objetivos**
- **Conflictos por semana**: < 2
- **Tiempo de resolución**: < 30 minutos
- **Tasa de detección temprana**: > 90%
- **Tiempo de sincronización**: < 5 minutos

### **Monitoreo**
- Workflow de detección automática
- Notificaciones en PRs
- Alertas en Slack (configurable)
- Issues automáticos para conflictos

## 🔧 **Configuración Avanzada**

### **1. Configurar Notificaciones de Slack**

```bash
# Agregar webhook de Slack como secret
gh secret set SLACK_WEBHOOK --body "https://hooks.slack.com/services/..."

# El workflow de detección de conflictos usará automáticamente este webhook
```

### **2. Personalizar Reglas de Protección**

```bash
# Usar GitHub CLI para configurar branch protection
gh api repos/:owner/:repo/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["build","test","security","lint"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":2,"dismiss_stale_reviews":true,"require_code_owner_reviews":true}'
```

### **3. Configurar CODEOWNERS**

```bash
# El archivo .github/CODEOWNERS ya está configurado
# Agregar más owners según sea necesario
echo "*.go @nuevo-owner" >> .github/CODEOWNERS
```

## 🎓 **Mejores Prácticas**

### **Para Desarrolladores**

1. **Sincronización Frecuente**
   ```bash
   # Ejecutar al menos una vez al día
   ./scripts/sync-branch.sh
   ```

2. **Verificación Antes de Commit**
   ```bash
   # Ejecutar antes de cada commit
   ./scripts/check-conflicts.sh
   ```

3. **Comunicación Proactiva**
   - Anunciar cambios en archivos críticos
   - Coordinar con el equipo
   - Usar issues para cambios grandes

4. **Ramas Pequeñas y Enfocadas**
   - Una funcionalidad por rama
   - Máximo 3-5 commits por PR
   - Merge frecuente a develop

### **Para el Equipo**

1. **Code Reviews Rápidos**
   - Revisar en < 24 horas
   - Feedback constructivo
   - Aprobar solo si está listo

2. **Comunicación Efectiva**
   - Daily standups
   - Retrospectivas mensuales
   - Compartir lecciones aprendidas

3. **Mantenimiento de Herramientas**
   - Actualizar scripts regularmente
   - Mejorar reglas basado en experiencia
   - Documentar nuevos patrones

## 🚨 **Solución de Problemas**

### **Problemas Comunes**

#### **1. Pre-commit Hooks Fallan**
```bash
# Verificar instalación
pre-commit --version

# Reinstalar hooks
pre-commit uninstall
pre-commit install

# Ejecutar manualmente
pre-commit run --all-files
```

#### **2. Scripts No Ejecutables**
```bash
# Hacer ejecutables
chmod +x scripts/*.sh

# Verificar permisos
ls -la scripts/
```

#### **3. Conflictos en Workflows**
```bash
# Verificar sintaxis
yamllint .github/workflows/*.yml

# Validar con GitHub Actions
# Usar el workflow de validación incluido
```

#### **4. Dependencias Go Desincronizadas**
```bash
# Limpiar dependencias
go mod tidy

# Verificar
go mod verify
```

### **Escalación de Problemas**

1. **Conflicto Simple**: Resolver localmente
2. **Conflicto Complejo**: Solicitar ayuda del equipo
3. **Conflicto Crítico**: Crear issue urgente
4. **Problema Sistémico**: Retrospectiva del equipo

## 📈 **Mejora Continua**

### **Retrospectivas Mensuales**

1. **Revisar Métricas**
   - Número de conflictos
   - Tiempo de resolución
   - Efectividad de herramientas

2. **Identificar Patrones**
   - Tipos de conflictos comunes
   - Archivos problemáticos
   - Procesos ineficientes

3. **Mejorar Procesos**
   - Actualizar documentación
   - Refinar scripts
   - Ajustar reglas de protección

### **Actualizaciones de Herramientas**

1. **Scripts**
   - Mejorar detección
   - Agregar nuevas validaciones
   - Optimizar performance

2. **Workflows**
   - Actualizar acciones
   - Mejorar notificaciones
   - Agregar nuevas verificaciones

3. **Documentación**
   - Mantener actualizada
   - Agregar ejemplos
   - Mejorar claridad

## 🎯 **Próximos Pasos**

### **Implementación Inmediata**
1. ✅ Instalar pre-commit hooks
2. ✅ Configurar branch protection rules
3. ✅ Probar scripts de sincronización
4. ✅ Configurar notificaciones

### **Implementación a Corto Plazo**
1. 🔄 Entrenar al equipo en nuevas herramientas
2. 🔄 Configurar métricas de monitoreo
3. 🔄 Establecer proceso de retrospectivas
4. 🔄 Crear templates de PR

### **Implementación a Largo Plazo**
1. 📋 Integrar con herramientas de CI/CD
2. 📋 Automatizar más procesos
3. 📋 Crear dashboards de métricas
4. 📋 Expandir a otros proyectos

## 📚 **Recursos Adicionales**

- [Git Flow](https://nvie.com/posts/a-successful-git-branching-model/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [GitHub Flow](https://docs.github.com/en/get-started/quickstart/github-flow)
- [Pre-commit Hooks](https://pre-commit.com/)
- [GitHub Actions](https://docs.github.com/en/actions)

## 🤝 **Contribución**

Para mejorar estas herramientas:

1. **Crear issue** describiendo la mejora
2. **Crear rama** con prefijo `improvement/`
3. **Implementar cambios** siguiendo las guías
4. **Crear PR** con descripción detallada
5. **Solicitar review** del equipo

---

**¡Recuerda: La prevención es mejor que la resolución!** 🛡️

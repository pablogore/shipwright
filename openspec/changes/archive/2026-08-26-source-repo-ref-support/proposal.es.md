# Propuesta: Conectar `spec.source.repo` / `spec.source.ref` en la Resolución de Fuente del Workflow

## Intención

`spec.source.repo` y `spec.source.ref` se parsean correctamente pero fallan en runtime con un error explícito de "no implementado" en `resolveWorkflowSource`. El schema ya define estos campos (`SourceSpec` en `internal/workflow/manifest/schema.go:50`), y la lógica de clon git ya existe en `internal/pipelines/shared`. Este cambio conecta ambos: cuando `spec.Repo != ""`, `resolveWorkflowSource` clona el repositorio y retorna el directorio clonado en lugar de retornar un error.

## Alcance

### En alcance

- Reescribir `resolveWorkflowSource` para manejar `spec.Repo` (clon git) y `spec.Path` (comportamiento existente)
- Detectar protocolo (`git@` → SSH, de lo contrario HTTPS) a partir del prefijo de la URL
- Delegar a `shared.CloneRepo(ctx, client, GitCloneOpts{...}, protocol)`
- Agregar parámetro `ctx context.Context` a `resolveWorkflowSource` (los invocadores ya lo tienen)
- Pruebas para ambos caminos de código: `repo` configurado (HTTPS y SSH) y `path` configurado (existente)
- Ref por defecto `"main"` cuando `spec.Ref` está vacío

### Fuera de alcance

- Conexión de `authSecretRef` — el campo de schema existe pero no tiene resolvedor en runtime; documentar como trabajo futuro
- Soporte de clon superficial para refs basados en SHA (requiere flag `--depth` en el cloner)
- Cambios a `internal/pipelines/shared/cloner.go` o `internal/pipelines/shared/https_cloner.go`

## Capacidades

### Capacidades nuevas

- `git-source-resolution`: Resuelve la fuente de un workflow desde un repositorio git remoto vía clon HTTPS o SSH, retornando un `*dagger.Directory` al motor

### Capacidades modificadas

- `workflow-execution`: La función `resolveWorkflowSource` en `main.go` obtiene un nuevo camino de código; el comportamiento de ejecución no cambia para `spec.source.path`

## Enfoque

1. Agregar `ctx context.Context` como primer parámetro a `resolveWorkflowSource` — el único invocador (`runWorkflowEngine` en línea 484) ya tiene un contexto
2. Cuando `spec.Repo != ""`:
   - Determinar protocolo: `strings.HasPrefix(spec.Repo, "git@")` → `shared.SSHProtocol`, de lo contrario `shared.HTTPSProtocol`
   - Construir `shared.GitCloneOpts{Repo: spec.Repo, Branch: spec.Ref, Name: "workflow-source"}`
   - Llamar a `shared.CloneRepo(ctx, client, opts, protocol)` y retornar el resultado
3. Cuando `spec.Repo == ""`: lógica de path existente (sin cambios)
4. Agregar pruebas: mock de `dagger.Client` con interfaz `pipelines.DaggerClient`; probar clon HTTPS, clon SSH, ref por defecto vacío, y respaldo de path

## Áreas Afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `main.go:563-573` | Modificado | `resolveWorkflowSource` reescrito para soportar `spec.Repo` |
| `main_test.go` | Modificado | Nuevas pruebas para el camino de código repo/ref |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|-------------|------------|
| Ref vacío por defecto es `"main"` para HTTPS pero el clon SSH puede fallar sin clave | Media | Ref por defecto `"main"`; error de autenticación SSH retorna error claro del cloner existente |
| Clon superficial (`--depth=1`) incompatible con refs basados en SHA | Baja | No abordado en este cambio; el clon superficial es una preocupación compartida, no de esta función |
| `authSecretRef` se parsea pero no tiene efecto en runtime | Baja | Explícitamente fuera de alcance; documentar como trabajo futuro |
| Romper comportamiento existente de `spec.source.path` | Baja | El camino de path no cambia; protegido por `spec.Repo == ""` |

## Plan de Reversión

Revertir el commit. El único cambio de comportamiento está en `resolveWorkflowSource`; revertir restaura el error de "no implementado" para `spec.Repo`. No hay migración de datos, no hay cambios de schema, no hay estado persistido afectado.

## Dependencias

- `internal/pipelines/shared/cloner.go` (fábrica `CloneRepo`)
- `internal/pipelines/shared/https_cloner.go` (autenticación HTTPS)
- `internal/pipelines/shared/ssh_cloner.go` (autenticación SSH)
- `internal/pipelines/shared/credentials.go` (resolución de tokens)

## Criterios de Éxito

- [ ] `spec.source.repo` con URL HTTPS clona el repositorio y retorna un `*dagger.Directory` válido
- [ ] `spec.source.repo` con URL SSH (`git@...`) clona vía SSH
- [ ] `spec.source.ref` vacío por defecto a rama `"main"`
- [ ] Comportamiento existente de `spec.source.path` sin cambios
- [ ] Pruebas unitarias pasan para ambos caminos de código (`go test -race ./...`)

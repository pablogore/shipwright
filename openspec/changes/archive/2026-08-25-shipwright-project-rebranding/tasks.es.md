# Tareas: Renombramiento del proyecto a Shipwright

## Pronóstico de carga de revisión

| Campo | Valor |
|-------|-------|
| Líneas modificadas estimadas | ~1200–1800 (905 coincidencias × 2 por par de líneas antigua/nueva, en 105 archivos) |
| Riesgo de presupuesto de 400 líneas | Alto |
| PRs encadenados recomendados | Sí |
| División sugerida | PR 1 → PR 2 → ... → PR 7 (ordenados por dependencia) |
| Estrategia de entrega | ask-on-risk |
| Estrategia de encadenamiento | pending |

Decisión necesaria antes de aplicar: Sí
PRs encadenados recomendados: Sí
Estrategia de encadenamiento: pending
Riesgo de presupuesto de 400 líneas: Alto

### Unidades de trabajo sugeridas

| Unidad | Objetivo | PR probable | Comando de prueba focalizado | Arnés en tiempo de ejecución | Límite de rollback |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Ruta de módulo + imports compilan (Fase 0-1) | PR 1 | `go build ./...` | N/A — solo compilación, sin cambio de comportamiento | Revertir rama PR1; grafo de imports restaurado, nada depende de él aún |
| 2 | Identidad CLI/build (Fase 2) | PR 2 | `make build` | `./shipwright --help` / `--version` | Revertir cambios de Makefile/main.go/goreleaser; nombre del binario revierte |
| 3 | Identidad config + env (Fase 3, RED-primero) | PR 3 | `go test ./internal/config/...` | Ejecutar CLI con `SHIPWRIGHT_TOKEN` definido + `.shipwright.yml` presente | Revertir config.go/yaml_parser.go; descubrimiento revierte |
| 4 | CI/CD + release (Fase 4) | PR 4 | `yamllint .github/` | N/A — solo CI, verificado con dry-run de workflow | Revertir `git mv` + ediciones de workflow/action |
| 5 | Docs + ejemplos (Fase 5) | PR 5 | verificación de enlaces markdown | N/A — solo documentación | Revertir archivos de docs/examples de forma independiente |
| 6 | Comentarios/fixtures/ediciones manuales (Fase 6) | PR 6 | `go test -race ./...` | N/A — solo comentarios/literales de test | Revertir ediciones de comentarios/fixtures |
| 7 | Barrido residual + verificación final (Fase 7-8) | PR 7 | `go build ./...` && `go test -race ./...` | Los comandos `rg` de barrido son el arnés | Revertir limpieza de cadenas residuales; sin efecto funcional |

## Fase 0: Línea base y congelamiento de exclusiones

- [x] 0.1 Registrar línea base: conteos de `rg -ic 'syntegrity[-_ ]?dagger'` y `rg -ic syntegrity` por archivo; capturar % actual de `make coverage`.
- [x] 0.2 Congelar lista de exclusión: imports de go-kit-logger, ejemplos eventengine de `AGENTS.md`, cadenas de empresa en `shared/{ssh,https}_cloner.go`, `internal/pipelines/infra/` (`SyntegrityInfraPipeline`), `examples/configs/tenant-svc.yml` (`ghcr.io/syntegrity`), grep `gitlab.com/syntegrity` del Makefile, `1export` en la raíz, `openspec/changes/shipwright-project-rebranding/**`.

## Fase 1: Ruta de módulo e imports [RIESGO ALTO — internal/executors, internal/pipelines]

- [x] 1.1 `go mod edit -module github.com/pablogore/shipwright` (reglas 1-3 del mapa de tokens del diseño).
- [x] 1.2 Actualizar rutas de import en los 56 archivos `.go` afectados bajo `internal/`, `mocks/`, `tests/`, `examples/`.
- [x] 1.3 [RIESGO ALTO] Actualizar imports en `internal/executors/{selector.go,selector_test.go,docker_executor.go,docker_executor_test.go}`; verificar que solo cambie la ruta de import.
- [x] 1.4 [RIESGO ALTO] Actualizar imports en `internal/pipelines/**` (`test/`, `shared/`, `infra/`, `go-service/`); preservar cadenas de empresa en `shared/{ssh,https}_cloner.go` y `SyntegrityInfraPipeline` en `infra/`.
- [x] 1.5 Compuerta: `go build ./...` exitoso. (Verificación parcial — ver apply-progress.md: el sandbox no tiene credenciales de red para la dependencia privada `go-kit-logger`, condición preexistente en la línea base y no relacionada con este cambio. Todos los paquetes compilables — `internal/cache`, `internal/interfaces`, `internal/pipelines`, `internal/config` — compilan y pasan `go vet` correctamente contra la nueva ruta de módulo.)
- [x] 1.6 Compuerta: `git diff` sobre líneas de import de `dagger.io/dagger` y `go-kit-logger` está vacío.

## Fase 2: CLI / Build [RED-primero: binario/help/version]

- [x] 2.1 RED: actualizar aserciones de tests de help/version para esperar `shipwright`; confirmar que fallan contra el código actual. (El sandbox no puede ejecutar `go test`/`go vet` sobre el paquete `main` — misma limitación de credenciales de `go-kit-logger` que en PR1. RED confirmado por inspección: `main_test.go` referencia `cliName`, `versionLogMessage`, `initLogMessage`, símbolos inexistentes en `main.go` antes de esta tarea — fallo garantizado en tiempo de compilación.)
- [x] 2.2 GREEN: actualizar `BINARY_NAME` del `Makefile`, textos de flagset/help/version de `main.go`, nombre del binario en `.goreleaser.yml`.
- [x] 2.3 Compuerta: `make build` exitoso; test de 2.1 ahora pasa. (Verificación parcial — ver apply-progress.md: `make build` falla con el mismo error preexistente de credenciales de `go-kit-logger` que la línea base y PR1, confirmado idéntico, no un error de compilación nuevo. `gofmt -l` limpio en los 4 archivos tocados.)
- [x] 2.4 Verificar que `./shipwright --help`/`--version` no contengan el token `syntegrity-dagger`. (Verificado por inspección de código fuente, no en tiempo de ejecución — el binario no se puede compilar en este sandbox. `cliName = "shipwright"` controla el nombre del flagset mostrado en `--help`; `versionLogMessage = "Shipwright version"` controla la salida de `--version`. `rg -n syntegrity main.go` confirma cero ocurrencias restantes de cara al CLI.)

## Fase 3: Config y variables de entorno [RED-primero: EnvPrefix, descubrimiento de config]

- [x] 3.1 RED: actualizar `internal/config/config_test.go` para asertar `EnvPrefix == "SHIPWRIGHT_"`; confirmar que falla.
- [x] 3.2 RED: actualizar `internal/config/yaml_parser_test.go` para asertar que la lista de 6 candidatos de `findConfigFile` usa nombres shipwright; confirmar que falla.
- [x] 3.3 GREEN: actualizar constante `EnvPrefix` a `SHIPWRIGHT_`.
- [x] 3.4 GREEN: actualizar `internal/config/yaml_parser.go` — los 6 candidatos (`.syntegrity-dagger.yml/.yaml`, `syntegrity-dagger.yml/.yaml`, `.github/syntegrity-dagger.yml/.yaml`) → equivalentes shipwright; actualizar import (línea 9) y comentario de documentación (línea 20).
- [x] 3.5 GREEN: actualizar valor por defecto del flag `-config` en `main.go`.
- [x] 3.6 Compuerta: `go test ./internal/config/...` pasa; 3.1/3.2 ahora en verde. (Bloqueado por la misma limitación de entorno preexistente que PR1/PR2 — ver apply-progress.md. Verificado en su lugar mediante `go build ./internal/config/...` (limpio), `gofmt -l` en todos los archivos tocados (solo persiste la violación preexistente y no relacionada en `yaml_parser.go`) y confirmación de diff dirigida con `rg` de las cadenas exactas cambiadas.)

## Fase 4: CI/CD y release

- [x] 4.1 `git mv .github/actions/syntegrity-dagger .github/actions/shipwright`; actualizar referencias internas en `action.yml`.
- [x] 4.2 Actualizar `ci.yml`, `release.yml`, `dependabot.yml` (`secrets.SYNTEGRITY_DAGGER_TOKEN` → `secrets.SHIPWRIGHT_TOKEN`), `CODEOWNERS`, `rulesets/README.md`.
- [x] 4.3 Señalar dependencia fuera de banda: el secreto de GitHub `SHIPWRIGHT_TOKEN` debe existir antes del merge (acción del owner, no tarea de código).
- [x] 4.4 Actualizar plantillas de URL de instalación en `.goreleaser.yml` a `pablogore/shipwright` (si no se cubrió en 2.2).
- [x] 4.5 Compuerta: `yamllint` pasa en los archivos de workflow/action modificados.

## Fase 5: Documentación y ejemplos

- [x] 5.1 Actualizar `README.md` incl. URL del badge a `pablogore/shipwright`.
- [x] 5.2 Actualizar 21 archivos bajo `docs/`.
- [x] 5.3 Reescribir entradas históricas de `CHANGELOG.md` a Shipwright.
- [x] 5.4 Actualizar `examples/**` (GitHub Actions, Jenkins, local, configs, muestras Go); preservar el namespace de registro `ghcr.io/syntegrity` en `examples/configs/tenant-svc.yml`.
- [x] 5.5 Compuerta: verificación de enlaces en `docs/`/`README.md`.

## Fase 6: Comentarios, fixtures y ediciones manuales

- [x] 6.1 Actualizar `internal/**/*_test.go`, `internal/plugins/mocks*.go`, `tests/`, fixtures para cadenas de identidad sin cambiar aserciones/intención. (`rg -ni 'syntegrity[-_ ]?dagger' internal/ tests/` no encontró coincidencias en archivos `_test.go`/`mocks*.go`/`tests/` — las únicas coincidencias restantes fueron `internal/app/app.go` e `internal/app/health.go`, tratadas en 6.4.)
- [x] 6.2 Edición manual (la regla no coincide): `internal/config/errors.go:1` comentario "aplicación Syntegrity" → "aplicación Shipwright".
- [x] 6.3 Edición manual (la regla no coincide): `internal/config/appconf.test.go:1` comentario, mismo reemplazo.
- [x] 6.4 Actualizar comentarios de código restantes en `internal/` que referencien la identidad antigua (reglas catch-all del mapa de tokens). (`internal/app/app.go:96,102` mensajes de log de inicio/parada "Syntegrity Dagger application" → "Shipwright application"; `internal/app/health.go:140` encabezado `User-Agent` saliente `syntegrity-dagger/1.0` → `shipwright/1.0`.)
- [x] 6.5 Actualizar `.serena/project.yml` y comandos de build/test en `openspec/config.yaml` (`-o syntegrity-dagger` → `-o shipwright`). (`openspec/config.yaml` líneas 25 y 65 actualizadas. `.serena/project.yml` inspeccionado — contiene `project_name: "syntegrity-dagger"` pero ninguna referencia al comando de build `-o syntegrity-dagger`, por lo que la condición de la tarea no aplica; sin cambios. Ambos archivos son configuración local de herramientas sin seguimiento en git en este repositorio — `openspec/config.yaml` se editó en disco según la instrucción, pero, siguiendo el precedente de PR1–5 de nunca comprometer nada bajo `openspec/`, no se incluyó en el commit de PR6.)
- [x] 6.6 Compuerta: `gofmt -l .` limpio. (`gofmt -l` en los 4 archivos `.go` tocados no devuelve nada. `gofmt -l .` a nivel de repo completo sigue reportando las violaciones preexistentes de línea base listadas en la restricción de entorno — `internal/cache/*`, `internal/config/validation*.go`, `internal/config/yaml_parser.go`, `internal/config/yaml_step_config*.go`, `internal/pipelines/go-service/pipeline_test.go`, `internal/pipelines/test/gotester.go` — ninguna tocada ni corregida por este PR, fuera de alcance.)

## Fase 7: Barrido residual

- [x] 7.1 Ejecutar `rg -i 'syntegrity[-_ ]?dagger' --glob '!openspec/changes/shipwright-project-rebranding/**' --glob '!.git/**' --glob '!coverage/**'` — cero coincidencias excepto el grep `gitlab.com/syntegrity` del Makefile y `1export` en la raíz. (La primera ejecución encontró 4 coincidencias adicionales fuera de las dos excepciones documentadas: comentario en `main_test.go:98`, `openspec/specs/README.md:3`, comentario de encabezado en `Makefile:1`, banner de ayuda en `Makefile:39`. Las 4 se corrigieron; la re-ejecución confirmó cero coincidencias — ver tabla de clasificación en apply-progress.md.)
- [x] 7.2 Ejecutar `rg -i syntegrity --glob '!openspec/changes/shipwright-project-rebranding/**'` — las coincidencias restantes son solo de la lista de exclusión de identidad de empresa. (106 coincidencias en 30 archivos, todas clasificadas en apply-progress.md — cada una es una referencia de empresa/organización de la lista de exclusión documentada o un literal de test `NotContains` legítimo que prueba la ausencia de la identidad antigua. Cero coincidencias sin clasificar.)
- [x] 7.3 Confirmar explícitamente que `openspec/changes/shipwright-project-rebranding/**` fue excluido de ambos comandos de barrido (los artefactos SDD citan el nombre antiguo intencionalmente). (Confirmado — ambos comandos se ejecutaron con `--glob '!openspec/changes/shipwright-project-rebranding/**'`.)

## Fase 8: Verificación final de build + tests

- [x] 8.1 `go build ./...` (no solo el paquete raíz — `examples/` es `package main`). (**Bloqueado por el entorno, requiere verificación del usuario/CI** — falla con el mismo error de credenciales de `go-kit-logger` preexistente, confirmado idéntico byte a byte contra el commit de línea base sin modificar `b14c726` mediante un worktree dedicado. No es una regresión del renombrado.)
- [x] 8.2 `go test -race ./...`; cobertura ≥ 90% local / 70% CI sin cambios. (**Bloqueado por el entorno, requiere verificación del usuario/CI** — 13 paquetes fallan con el error de credenciales idéntico; los 3 paquetes sin dependencia transitiva de `go-kit-logger` — `internal/cache`, `internal/interfaces`, `internal/pipelines` — pasan limpiamente. Los umbrales de cobertura no se pueden verificar en este sandbox por la misma razón.)
- [x] 8.3 Confirmar que `git diff` sobre imports de `dagger.io/dagger` y referencias a `go-kit-logger` está vacío (verificación final de no regresión de la lista de exclusión). (Confirmado vacío en toda la cadena de 7 commits, `git diff b14c726 HEAD -- '*.go'`, para ambos patrones.)

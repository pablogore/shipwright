# Tareas: Actualización de Runtime/Toolchain Gestionada por el Provider (Go Primero)

## Pronóstico de Carga de Revisión

| Campo | Valor |
|---|---|
| Líneas modificadas estimadas | ~1.650 líneas autoría (design.md "Review Budget Forecast") |
| Riesgo de presupuesto de 400 líneas | Alto |
| PRs encadenados recomendados | Sí |
| División sugerida | Cadena de 4 slices, ~520 / ~430 / ~370 / ~290 |
| Estrategia de entrega | auto-chain |
| Estrategia de cadena | stacked-to-main |

```text
Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High
```

**Nota de divergencia**: el conteo/límites de slices propuestos por design.md (cadena de 4 slices, inspect primero) se reverificaron contra el estado exacto actual de los archivos (`providers/go/daggerkit`, `main.go`, `providers/go/go.mod`) y se confirmaron correctos — no se requiere volver a segmentar. `chain_strategy: stacked-to-main` se realiza en este repositorio como **PRs apilados que se fusionan a `develop` en orden**: la regla dura de `shipwright-chained-pr` prohíbe cortar ramas de cadena desde `main` (Git Flow), y `develop→main` es una puerta de promoción automatizada separada (commit `f972335`), no un destino manual de PR. "main" en el vocabulario genérico de la guarda se traduce, en este repositorio, al tronco real: `develop`.

### Unidades de Trabajo Sugeridas (límites de cadena)

| Slice | Rama (base `develop`) | Contenido | Líneas est. | Test enfocado | Arnés de runtime | Límite de rollback |
|---|---|---|---|---|---|---|
| 1 | `feature/runtime-inspect-go` | `runtime-inspect` de punta a punta: núcleo de análisis/conflicto (A1-A6, incl. `.go-version`), cableado de Inspect, daggerkit de lectura, guardia D-4, allowlist 5→6 | ~520 | `go test ./providers/go/... ./internal/workflow/... ./pkg/shipwright/...` | `go run . --list-steps` contra un manifiesto con un paso `runtime-inspect` | Revertir el commit; elimina 1 entrada de allowlist, 1 par de registry, 1 caso de dispatch, 1 caso en `main.go` — no existe aún código de mutación |
| 2 | `feature/runtime-upgrade-single-module` | Cableado de `runtime-upgrade`, mutación de `go.mod`/`.go-version` para módulo único, daggerkit de escritura, allowlist 6→7 | ~430 | `go test ./providers/go/... ./internal/workflow/...` | `go run . --list-steps` con un paso `runtime-upgrade` contra un workspace de fixture | Revertir el commit; no existe aún validación multi-módulo ni `go build`/`tidy` que deje un estado a medio cablear |
| 3 | `feature/runtime-upgrade-workspace` | Bucle de mutación multi-módulo `go.work`, cableado de A1-A3 en `Upgrade`, guardia contra path-escape | ~370 | `go test -run TestUpgrade_Workspace ./providers/go/...` | `Upgrade` con motor real bajo tag `integration` sobre `testdata/runtime/workspace-3-modules` | Revertir el commit; la ruta de módulo único (slice 2) sigue funcionando sin mutar |
| 4 | `feature/runtime-upgrade-tidy-validate` | `go mod tidy`, validación post-mutación `go build ./...`, reporte de delta de `go.sum`, test de integración | ~290 | `go test -race ./providers/go/...` | `make test-integration` (contenedor Dagger real, tag `integration`) | Revertir el commit; la mutación del slice 3 sigue funcionando sin post-validación (documentado como brecha interina aceptada en la descripción de ese PR) |

Cada slice pasa de forma independiente `go build ./...`, `go test -race ./...`, `golangci-lint run` en su propio checkpoint antes de que el siguiente slice se ramifique desde `develop`.

---

## Fase 1 (Slice 1 — PR1 `feature/runtime-inspect-go`): `runtime-inspect` de punta a punta

- [ ] 1.1 Crear fixtures `providers/go/testdata/runtime/{single-module,workspace-3-modules,divergent-go,divergent-toolchain,work-go-mismatch,goversion-file-mismatch,downgrade,malformed,path-escape,live-repo-drift}/` (las 9, reutilizadas en slices 2-3); `live-repo-drift` es una copia byte a byte del `go.work`/`go.mod`/`.go-version` reales de este repositorio.
- [ ] 1.2 RED `providers/go/toolchainpin_test.go::TestDefaultGoVersionMatchesGoMod` — parsea la directiva `go` de `providers/go/go.mod`, afirma igualdad con `defaultGoVersion` (`gobuilder.go:44`); falla hoy (1.26.7 vs 1.25.5) — requisito spec: runtime-toolchain (sustitución tier-3 de D-4).
- [ ] 1.3 `go get golang.org/x/mod@<versión fijada por la raíz>` en `providers/go` (`go.mod`/`go.sum`) — confirmar paridad de versión con el módulo raíz (Open Question de design).
- [ ] 1.4 RED `providers/go/toolchain_test.go` — tabla dirigida sobre fixtures `testdata/runtime/*` para `parseWorkspace`/`detectConflicts` A1-A6 (incl. `goversion-file-mismatch` probando RED contra el drift real A3 de este repo) — requisito spec: Read-Only Drift Inspection, escenario de fuentes ambiguas.
- [ ] 1.5 GREEN `providers/go/toolchain.go` — `parseWorkspace`, `detectConflicts` (A1-A6), `*AmbiguousToolchainError`; Go puro sobre bytes vía `golang.org/x/mod/modfile`, sin Dagger, sin `os.WriteFile`/`exec.Command` (matriz de amenazas: construcción de comandos, escritura arbitraria) — test de guardia estática que verifica que este archivo no importa esos paquetes.
- [ ] 1.6 RED `providers/go/daggerkit/adapter_test.go` — mocks de `DaggerDirectory.File`/`Entries`, `DaggerFile.Contents` (regla de orden de doble selección 1, mocks de daggerkit primero).
- [ ] 1.7 GREEN `providers/go/daggerkit/{interfaces,adapter,mocks}.go` — agregar `DaggerDirectory.File(string) DaggerFile`, `DaggerDirectory.Entries(context.Context) ([]string, error)`, `DaggerFile.Contents(context.Context) (string, error)` (D-9 lado de lectura; `DaggerDirectory.WithNewFile` diferido a la Fase 2).
- [x] 1.8 RED `providers/go/runtimeinspector_test.go` — camino feliz de `GoRuntimeInspector.Inspect`, cliente/fuente nil, `failOnDrift` true/false (D-4b) — requisito spec: escenarios de Read-Only Drift Inspection.
- [x] 1.9 GREEN `providers/go/runtimeinspector.go` — `GoRuntimeInspector.Inspect(ctx, source) (string, error)`; construye JSON `DriftReport` vía `pkg/shipwright/runtime.go` (crear tipos `DriftReport`, `ConflictState`); el campo `with` `failOnDrift` devuelve error ante conflicto.
- [x] 1.10 RED extender el mapa de reflexión de `pkg/shipwright/capabilities_test.go` con `RuntimeInspector` (cumplimiento de D-A).
- [x] 1.11 GREEN `pkg/shipwright/capabilities.go` — agregar `RuntimeInspector interface { Inspect(ctx, source *dagger.Directory) (string, error) }`.
- [x] 1.12 GREEN `.dagger/capabilities.go` — proyección `RuntimeInspector` de Capa 2 (`dagger.DaggerObject` + `Inspect(...) (string, error)`, conserva `error` según D-3).
- [x] 1.13 REQUERIDO ejecutar `go run ./pkg/shipwright -update` (o el comando documentado del repo para golden) para regenerar `pkg/shipwright/testdata/api.golden` — **revisar el diff**, no confirmar a ciegas (conteo de interfaces 5→6).
- [x] 1.14 RED extender `internal/workflow/manifest/validate_test.go` — `runtime-inspect` en la allowlist, capacidad fuera de la allowlist sigue fallando — requisito spec: workflow-manifest MODIFICADO (escenarios de allowlist).
- [x] 1.15 GREEN `internal/workflow/manifest/validate.go` — agregar `runtime-inspect` a la allowlist (5→6) + esquema `with` (`workspaceRoot`, `expectedVersion`, `failOnDrift`).
- [x] 1.16 RED extender `internal/workflow/providers/registry_test.go` — par `RegisterRuntimeInspector`/`ResolveRuntimeInspector`.
- [x] 1.17 GREEN `internal/workflow/providers/registry.go` — agregar `inspectors *table[shipwright.RuntimeInspector]` + par Register/Resolve.
- [x] 1.18 GREEN `internal/workflow/providers/register.go` — registrar el provider `go-runtime` bajo `runtime-inspect` (confirmar nombre del provider al aplicar, según Open Question de design).
- [x] 1.19 RED extender `internal/workflow/engine/dispatch_test.go` — `dispatchRuntimeInspect` resuelve→llama→envuelve en línea recta, sin código de bloqueo agregado (confirmación D1).
- [x] 1.20 GREEN `internal/workflow/engine/execute.go` — caso de dispatch `dispatchRuntimeInspect` + constantes de campos `with`.
- [x] 1.21 RED extender `main_test.go` — `resolveCapabilityRef` gana un caso `runtime-inspect` (D-8; antes eran "cinco ramas de capacidad").
- [x] 1.22 GREEN `main.go::resolveCapabilityRef` — agregar `case "runtime-inspect"` (~5 líneas).
- [x] 1.23 RED de matriz de amenazas: test de guardia estática que confirma que ningún `os.WriteFile`/`exec.Command`/cliente HTTP/comando git es alcanzable desde `providers/go/toolchain.go` o `runtimeinspector.go` (requisito de sin red/git/SCM).
- [x] 1.24 Verificar: `go build ./...`, `go test -race ./...`, `golangci-lint run` en verde; confirmar que `--list-steps` lista un paso `runtime-inspect` sin error.

## Fase 2 (Slice 2 — PR2 `feature/runtime-upgrade-single-module`): `runtime-upgrade` de módulo único

- [ ] 2.1 RED `providers/go/toolchain_test.go` — extender con mutación de `mutateGoMod`/`mutateGoWork`/`.go-version` sobre fixtures `single-module`, `downgrade` (A4), `malformed` (A5).
- [ ] 2.2 GREEN `providers/go/toolchain.go` — `mutateGoMod`, mutar `.go-version`; `targetVersion` validado vía `modfile.GoVersionRE` **antes** de cualquier escritura (matriz de amenazas: construcción de comandos desde config) — test RED: `"1.26.7; rm -rf /"`, `"--flag"` rechazados al parsear.
- [ ] 2.3 RED `providers/go/daggerkit/adapter_test.go` — mock de `DaggerDirectory.WithNewFile`.
- [ ] 2.4 GREEN `providers/go/daggerkit/{interfaces,adapter,mocks}.go` — agregar `DaggerDirectory.WithNewFile(path, contents string) DaggerDirectory` (D-9 lado de escritura).
- [ ] 2.5 RED mapa de reflexión de `pkg/shipwright/capabilities_test.go` — agregar `RuntimeUpgrader`.
- [ ] 2.6 GREEN `pkg/shipwright/capabilities.go` — `RuntimeUpgrader interface { Upgrade(ctx, source *dagger.Directory, targetVersion string) (*dagger.Directory, error) }`.
- [ ] 2.7 GREEN `.dagger/capabilities.go` — proyección `RuntimeUpgrader` de Capa 2; `Upgrade` **descarta `error`** (D-3, igual que `Builder.Build`).
- [ ] 2.8 REQUERIDO regenerar `pkg/shipwright/testdata/api.golden` (`-update`, diff revisado) — conteo de interfaces 6→7.
- [ ] 2.9 RED `providers/go/runtimeupgrader_test.go` — camino feliz de módulo único, aborto ambiguo (`(nil, err)`, sin directorio), ubicación faltante omitida — requisito spec: escenarios de Discovery-Driven Upgrade.
- [ ] 2.10 GREEN `providers/go/runtimeupgrader.go` — `GoRuntimeUpgrader.Upgrade`; solo ruta de `go.mod` de módulo único (sin recorrido de `go.work` todavía); escribe `.shipwright/runtime-upgrade-report.json` en el Directory devuelto (D-2); crear en `pkg/shipwright/runtime.go`: `UpgradeReport`, `ModuleDrift`.
- [ ] 2.11 RED/GREEN `internal/workflow/manifest/validate.go` + `validate_test.go` — allowlist 6→7, esquema `with` de `runtime-upgrade` (`targetVersion` requerido, `workspaceRoot`, `tidy`, `allowDowngrade`).
- [ ] 2.12 RED/GREEN `internal/workflow/providers/registry.go` + `registry_test.go` — tabla `upgraders` + par Register/Resolve.
- [ ] 2.13 GREEN `internal/workflow/providers/register.go` — registrar `go-runtime` bajo `runtime-upgrade`.
- [ ] 2.14 RED/GREEN `internal/workflow/engine/execute.go` + `dispatch_test.go` — caso `dispatchRuntimeUpgrade`, sin código de bloqueo (D1).
- [ ] 2.15 RED/GREEN `main.go::resolveCapabilityRef` + `main_test.go` — agregar `case "runtime-upgrade"` (segundo caso de D-8).
- [ ] 2.16 RED de matriz de amenazas: guardia de importación estática extendida a `runtimeupgrader.go` (sin escrituras en el host; solo `Directory.WithNewFile`).
- [ ] 2.17 Verificar: `go build ./...`, `go test -race ./providers/go/... ./internal/workflow/...`, `golangci-lint run` en verde; `--list-steps` lista ambas capacidades.

## Fase 3 (Slice 3 — PR3 `feature/runtime-upgrade-workspace`): actualización multi-módulo `go.work`

- [ ] 3.1 RED `providers/go/toolchain_test.go` — cablear los resultados de `detectConflicts` A1/A2/A3 (construidos en Fase 1) en la ruta de aborto pre-mutación de `Upgrade`, sobre fixtures `workspace-3-modules`, `divergent-go`, `divergent-toolchain`, `work-go-mismatch` — requisito spec: Multi-Module Workspace Consistency.
- [ ] 3.2 RED `providers/go/toolchain_test.go::TestUpgrade_PathEscape` — fixture `use ../../etc` aborta antes de cualquier `WithNewFile` (matriz de amenazas: path traversal vía `use` de `go.work`) — rechazar rutas absolutas y cualquier ruta que escape la raíz del workspace tras `filepath.Clean`.
- [ ] 3.3 GREEN `providers/go/toolchain.go` — función de guardia contra path-escape usada por el bucle de mutación.
- [ ] 3.4 GREEN `providers/go/runtimeupgrader.go` — extender `Upgrade` para recorrer cada módulo referenciado por `go.work`, mutar cada uno y registrar un resultado por módulo en `UpgradeReport.Modules []ModuleDrift`.
- [ ] 3.5 RED `providers/go/runtimeupgrader_test.go::TestUpgrade_Workspace` — escenario de los tres módulos actualizados; variante con motor real de integración agregada bajo `testing/integration/` (tag `integration`).
- [ ] 3.6 Verificar: `go test -run TestUpgrade_Workspace ./providers/go/...`, `go test -race ./...`, `golangci-lint run` en verde; la ruta de módulo único (Fase 2) sigue pasando sin modificación.

## Fase 4 (Slice 4 — PR4 `feature/runtime-upgrade-tidy-validate`): validación de tidy + build

- [ ] 4.1 RED `providers/go/runtimeupgrader_test.go` — el fallo de validación post-mutación devuelve `(nil, err)` con un reporte que describe el fallo, nunca un directorio presentado como actualizado — requisito spec: escenario de fallo de validación post-mutación.
- [ ] 4.2 RED `providers/go/runtimeupgrader_test.go` — el fallo de validación de un módulo hace fallar toda la operación, el reporte nombra qué módulo falló/tuvo éxito — requisito spec: escenario de fallo de validación de un módulo.
- [ ] 4.3 GREEN `providers/go/runtimeupgrader.go` — cadena de contenedor según D-7: `From("golang:"+targetVersion)`, montar el Directory mutado, por módulo `go mod tidy` (omitir si `tidy: false`), por módulo `go build ./...` (D-6, sin `go vet`), exportar el workdir del contenedor como el Directory devuelto.
- [ ] 4.4 GREEN `pkg/shipwright/runtime.go` — extender `UpgradeReport`/`ModuleDrift` con `GoSumChanged bool`, `AddedModules`/`RemovedModules []string` (delta de `go.sum` por módulo, nunca el diff crudo).
- [ ] 4.5 RED/GREEN `providers/go/runtimeupgrader_test.go` — `validation: "build"` registrado en el reporte para que ningún consumidor sea inducido a error (D-6).
- [ ] 4.6 Test de integración: `Upgrade` con motor real sobre un workspace temporal bajo el tag `integration`, verificando que `go build ./...` efectivamente se ejecutó.
- [ ] 4.7 Verificar: `go test -race ./...`, `make test-integration`, `golangci-lint run`, `make coverage` ≥ 90% de piso local, todo en verde.

---

*Nota de desviación*: excede el presupuesto de 530 palabras de la skill. Causa: el brief de lanzamiento exige nombres de rama, base y estimaciones de líneas por slice, evidencia de test enfocado/arnés de runtime/límite de rollback por unidad de trabajo, cada fila aplicable de la matriz de amenazas como una tarea RED explícita, y las dos brechas de propuesta confirmadas (`main.go` D-8, `providers/go/daggerkit` D-9) ubicadas en su slice correcto — comprimido en ítems de checklist, sin relleno narrativo.

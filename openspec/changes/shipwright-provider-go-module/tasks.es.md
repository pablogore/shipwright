# Tareas: Extraer `internal/capabilities/` a un modulo independiente `providers/go`

## Pronostico de Carga de Revision

| Campo | Valor |
|-------|-------|
| Lineas modificadas estimadas | ~310-370 en el slice 1, ~30-50 en el slice 2 (excluye `go.sum`/`go.work.sum` generados) |
| Riesgo de presupuesto de 400 lineas | Medio |
| PRs encadenados recomendados | Si — impuesto por el diseno: el tag no puede existir antes de fusionar el slice 1, por lo que el slice 2 no puede iniciar antes |
| Division sugerida | PR 1 (slice 1) -> interludio de tag (manual) -> PR 2 (slice 2) |
| Estrategia de entrega | ask-on-risk |
| Estrategia de encadenamiento | stacked-to-main |

Decision necesaria antes de aplicar: Si
PRs encadenados recomendados: Si
Estrategia de encadenamiento: stacked-to-main
Riesgo de presupuesto de 400 lineas: Medio

### Unidades de Trabajo Sugeridas

| Unidad | Objetivo | PR probable | Comando de prueba enfocado | Arnes de ejecucion | Limite de rollback |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Extraer el modulo `providers/go`, guardas, y cambio de importadores | PR 1 | `go test -race ./... && GOWORK=off go build ./...` | Ejecucion de `examples/workflow/diamond.yaml`, comparacion antes/despues | `git revert -m 1` (atomico: movimiento + cambio + `go.work`) |
| tag | Crear el tag `providers/go/v0.1.0` sobre el sha de fusion del PR 1 | manual, no es un PR | `GOPROXY=direct go list -m .../providers/go@v0.1.0` | N/A — solo VCS, no se ejecuta codigo | Inmutable una vez publicado; se reemplaza con `v0.1.1`, nunca se elimina |
| 2 | Eliminar el `replace` raiz, resolucion basada en tag, aceptacion de `go install` | PR 2 (base `develop`, despues del PR 1 + tag) | `go test -race ./...` (guarda de no-`replace`) | `go install github.com/pablogore/shipwright@<tag>` desde `GOMODCACHE` limpio | Revert independiente al estado con `replace` |

## Fase 1: Slice 1 — Guardas RED (TDD)

- [x] 1.1 RED `internal/workspaceguard/work_test.go`: verificar que el conjunto `use` de `go.work` sea exactamente `{".", "./providers/go"}`, rechazar cualquier segmento `.dagger`/`dagger`, fallar de forma segura si `go.work` falta o no se puede parsear
- [x] 1.2 RED `providers/go/internalimport_test.go`: `parser.ParseDir` incluyendo `_test.go`, fallar ante cualquier import de `internal/**`
- [x] 1.3 RED `internal/daggerpin/pin_test.go`: agregar `TestProvidersGoDaggerVersionMatchesRoot` contra `providers/go/go.mod`

## Fase 2: Slice 1 — Extraccion (GREEN)

- [x] 2.1 Crear `providers/go/go.mod` (`go 1.26.1`; requiere `dagger.io/dagger v0.21.8` + pseudo-version de shipwright previa a la extraccion, D4)
- [x] 2.2 Implementar `internal/workspaceguard/work.go` usando `modfile.ParseWork` (GREEN de 1.1)
- [x] 2.3 Crear `go.work` (`use .` + `use ./providers/go`, `go 1.26.1`)
- [x] 2.4 `git mv` de los 5 archivos fuente + 8 de prueba, `internal/capabilities/` -> `providers/go/`; editar la clausula de paquete y comentario doc a `package golang`
- [x] 2.5 Editar `naming_test.go`: `pkgs["capabilities"]` -> `pkgs["golang"]` + referencias en la documentacion
- [x] 2.6 `cd providers/go && GOWORK=off go mod tidy` (GREEN de 1.2, 1.3)
- [x] 2.7 `go.mod` raiz: agregar `require .../providers/go` + `replace => ./providers/go` temporal (interino sancionado por D4)
- [x] 2.8 Cambiar importadores: `internal/workflow/providers/register.go`, `internal/app/container.go`, `internal/app/container_capabilities_test.go` (alias de import `golang`, referencias de tipo)
- [x] 2.9 Eliminar `internal/capabilities/`; confirmar que queda exactamente una copia de cada uno de los 5 tipos
- [x] 2.10 Actualizar `COMPATIBILITY.md` linea 46: `internal/capabilities/**` -> `providers/go/**`

## Fase 3: Slice 1 — Verificacion y fusion

- [x] 3.1 `go test -race ./...` y `GOWORK=off go build ./...` ambos en verde
- [x] 3.2 Verificar que `git diff --stat -- pkg/shipwright/` este vacio
- [x] 3.3 Verificar la etapa `setup` de CI (`go mod download`/`verify`) en modo workspace; anteponer `GOWORK=off` solo si falla
- [ ] 3.4 Fusionar el slice 1 a `develop` — rama publicada y PR abierto segun la convencion del repo (cadena stacked-to-main); la fusion real requiere accion humana/revisor, no realizada por este agente

## Fase 4: Interludio de tag (manual)

- [ ] 4.1 `git tag providers/go/v0.1.0 <sha-de-fusion-del-slice-1> && git push origin providers/go/v0.1.0`
- [ ] 4.2 Confirmar la resolucion: `GOPROXY=direct go list -m github.com/pablogore/shipwright/providers/go@v0.1.0`
- [ ] 4.3 Si la visibilidad del repositorio no esta confirmada: `curl -s https://proxy.golang.org/github.com/pablogore/shipwright/@v/list`; reportar de forma visible si esta vacio

## Fase 5: Slice 2 — Guarda de no-`replace` (TDD)

- [ ] 5.1 RED: prueba raiz que verifica que `modfile.Parse(go.mod).Replace` este vacio (falla: el `replace` aun esta presente)
- [ ] 5.2 Eliminar el `replace` del `go.mod` raiz; mantener `require .../providers/go v0.1.0`
- [ ] 5.3 `GOWORK=off go mod tidy` en la raiz; confirmar el `go.sum` regenerado
- [ ] 5.4 GREEN: volver a ejecutar 5.1

## Fase 6: Slice 2 — Aceptacion y regresion

- [ ] 6.1 `go install github.com/pablogore/shipwright@<tag>` desde un `GOMODCACHE` limpio
- [ ] 6.2 Ejecutar `examples/workflow/diamond.yaml` antes y despues; confirmar que los proveedores resueltos sean identicos
- [ ] 6.3 Confirmar cobertura >= 90% en ambos modulos (`make coverage`)
- [ ] 6.4 Confirmar que la guarda de `.dagger` falla cuando se agrega `use ./.dagger` (cubierto por los casos sinteticos `t.TempDir()` de 1.1)

## Matriz de Amenazas

N/A — design.md no registra cambios de enrutamiento, comandos de shell,
subprocesos, clasificacion de archivos ejecutables, ni limites de integracion
de procesos. No se requieren tareas RED adicionales mas alla de la Fase 1.

---

*Nota de desviacion:* excede el presupuesto de 530 palabras del skill. Causa:
el diseno impone dos slices encadenados con un interludio manual de tag, dos
paquetes de guarda nuevos, y un movimiento de 13 archivos mas un cambio de
importadores en 3 archivos — la misma complejidad que ya llevo a `proposal.md`
y `design.md` a exceder sus propios presupuestos. El contenido se mantiene
solo como checklist, sin relleno narrativo.

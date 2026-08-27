# Tareas: Extraer `internal/capabilities/` a un modulo independiente `providers/go`

## Pronostico de Carga de Revision

| Campo | Valor |
|-------|-------|
| Lineas modificadas estimadas | ~310-370 en el slice 1, ~150-250 en el slice 1b (YAML de workflow + paquete de guarda solo-test), ~30-50 en el slice 2 (excluye `go.sum`/`go.work.sum` generados) |
| Riesgo de presupuesto de 400 lineas | Medio — cada slice permanece muy por debajo de 400 de forma independiente; el slice 1b es solo `.github` mas un paquete Go pequeno, por lo que no eleva de forma significativa el conteo de lineas de ningun PR individual |
| PRs encadenados recomendados | Si — impuesto por el diseno: 3 slices de PR encadenados (D6). El slice 1b debe fusionarse antes de que exista el primer tag de proveedor, por lo que se ubica entre el slice 1 y el tag; el slice 2 no puede iniciar antes |
| Division sugerida | PR 1 (slice 1) -> PR 1b (slice 1b, automatizacion de release) -> interludio de tag (manual) -> PR 2 (slice 2) |
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
| 1b | Automatizacion de release (D6): corregir las 3 llamadas `git describe` sin filtrar de `release.yml`, agregar `ignore_tags` a `.goreleaser.yml`, crear `release-provider-go.yml`, guarda `internal/releaseguard/tags_test.go` | PR 1b (base `develop`, despues de fusionar el PR 1) | `go test -race ./internal/releaseguard/...` | N/A — el workflow de GitHub Actions solo se ejecuta ante un push real de tag `providers/go/v*`; la prueba local de la guarda es la evidencia, no hace falta motor | Revert independiente; solo afecta `.github/`, `.goreleaser.yml`, un archivo de test |
| tag | Crear el tag `providers/go/v0.1.0` sobre el sha de fusion del PR 1, despues de fusionar el PR 1b | manual, no es un PR | `GOPROXY=direct go list -m .../providers/go@v0.1.0` | N/A — solo VCS; el push dispara el workflow del slice 1b, que actua como arnes de ejecucion del propio tag | Inmutable una vez publicado; se reemplaza con `v0.1.1`, nunca se elimina |
| 2 | Eliminar el `replace` raiz, resolucion basada en tag, aceptacion de `go install` | PR 2 (base `develop`, despues del PR 1 + PR 1b + tag) | `go test -race ./...` (guarda de no-`replace`) | `go install github.com/pablogore/shipwright@<tag>` desde `GOMODCACHE` limpio | Revert independiente al estado con `replace` |

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

## Fase 4: Slice 1b — Guarda de release RED (TDD)

- [x] 4.1 RED `internal/releaseguard/tags_test.go`: table-test que verifica (a) que ninguna llamada `git describe --tags` en `release.yml` carezca de `--match`, (b) que los globs de tag de `release.yml` no coincidan con `providers/go/v0.1.0`, (c) que los globs de `release-provider-go.yml` no coincidan con `v1.2.3`, (d) que el regex de forma extraido del workflow acepte `providers/go/v0.1.0` y rechace `providers/go/v2.0.0`, `providers/go/v01.0.0`, `v0.1.0` — falla de forma segura: (a) por las llamadas `git describe` sin filtrar en `release.yml`, (c)/(d) por la ausencia de `release-provider-go.yml`

## Fase 5: Slice 1b — Automatizacion de release (GREEN)

- [x] 5.1 Agregar `--match 'v[0-9]*'` a las tres llamadas `git describe --tags --abbrev=0` en `.github/workflows/release.yml` (auto-bump ~L162, bump por dispatch ~L220, rango del changelog ~L245)
- [x] 5.2 Agregar `git: ignore_tags: ['providers/*']` a la busqueda de tag previo de `.goreleaser.yml`
- [x] 5.3 Crear `.github/workflows/release-provider-go.yml`: `on: push: tags: ['providers/go/v*']`, paso de validacion de forma (`^providers/go/v(0|1)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$`, mayor restringido a `0|1` para rechazar >= 2 en el mismo regex, nombrando la regla `/vN` de la ruta de modulo), y paso de identidad de modulo (la linea `module` de `providers/go/go.mod` == `github.com/pablogore/shipwright/providers/go`)
- [x] 5.4 Agregar paso de aislamiento a `release-provider-go.yml`: `cd providers/go && GOWORK=off go build ./... && GOWORK=off go test -race -short ./...` sobre el tag
- [x] 5.5 Agregar paso de changelog acotado por ruta (`git log --pretty='- %s (%h)' <tag providers/go previo>..<tag> -- providers/go/`) y paso de release (`gh release create "$TAG" --title "providers/go $VERSION" --notes-file … --latest=false`) a `release-provider-go.yml`
- [x] 5.6 Agregar paso de compuerta de visibilidad del proxy a `release-provider-go.yml`: `curl -sf https://proxy.golang.org/github.com/pablogore/shipwright/providers/go/@v/$VERSION.info`
- [x] 5.7 GREEN: volver a ejecutar 4.1

## Fase 6: Slice 1b — Verificacion y fusion

- [x] 6.1 `go test -race ./internal/releaseguard/...` en verde; `golangci-lint run` limpio sobre el paquete nuevo
- [x] 6.2 Validar el YAML de los workflows (`actionlint` o equivalente, si esta disponible) sobre `release-provider-go.yml` y el `release.yml` modificado
- [ ] 6.3 Fusionar el slice 1b a `develop` — rama publicada y PR abierto segun la convencion del repo (cadena stacked-to-main, base `develop` despues del PR 1); la fusion real requiere accion humana/revisor, no realizada por este agente

## Fase 7: Interludio de tag (manual)

- [ ] 7.1 `git tag providers/go/v0.1.0 <sha-de-fusion-del-slice-1> && git push origin providers/go/v0.1.0` — este push dispara el `release-provider-go.yml` del slice 1b
- [ ] 7.2 Confirmar que la ejecucion del workflow disparado quede en verde: forma, identidad, build/test aislado, changelog, `gh release create`, y la compuerta de visibilidad del proxy pasan todos
- [ ] 7.3 Confirmar la resolucion: `GOPROXY=direct go list -m github.com/pablogore/shipwright/providers/go@v0.1.0`; si la visibilidad del repositorio no esta confirmada, ademas `curl -s https://proxy.golang.org/github.com/pablogore/shipwright/@v/list` y reportar de forma visible si esta vacio

## Fase 8: Slice 2 — Guarda de no-`replace` (TDD)

- [ ] 8.1 RED: prueba raiz que verifica que `modfile.Parse(go.mod).Replace` este vacio (falla: el `replace` aun esta presente)
- [ ] 8.2 Eliminar el `replace` del `go.mod` raiz; mantener `require .../providers/go v0.1.0`
- [ ] 8.3 `GOWORK=off go mod tidy` en la raiz; confirmar el `go.sum` regenerado
- [ ] 8.4 GREEN: volver a ejecutar 8.1

## Fase 9: Slice 2 — Aceptacion y regresion

- [ ] 9.1 `go install github.com/pablogore/shipwright@<tag>` desde un `GOMODCACHE` limpio
- [ ] 9.2 Ejecutar `examples/workflow/diamond.yaml` antes y despues; confirmar que los proveedores resueltos sean identicos
- [ ] 9.3 Confirmar cobertura >= 90% en ambos modulos (`make coverage`)
- [ ] 9.4 Confirmar que la guarda de `.dagger` falla cuando se agrega `use ./.dagger` (cubierto por los casos sinteticos `t.TempDir()` de 1.1)

## Matriz de Amenazas

`N/A` para enrutamiento, comandos de shell, subprocesos, clasificacion de
archivos ejecutables, y limites de integracion de procesos — ninguno de esos
aplica.

**Automatizacion de VCS/PR: `Aplicable` desde D6.** El workflow que reacciona
al tag solo reacciona ante una referencia publicada por una persona (nunca
crea, mueve, ni elimina una), tiene permisos a nivel de job de solo
`contents: write`, y ejecuta `go build`/`go test` del arbol etiquetado sin
ninguna otra ejecucion de codigo. Cobertura RED: la tarea 4.1 de la Fase 4
(aserciones a-d de `internal/releaseguard/tags_test.go`) es la unica tarea RED
requerida para esta fila.

---

*Nota de desviacion:* excede el presupuesto de 530 palabras del skill. Causa:
el diseno impone tres slices encadenados (D6 agrego el slice 1b) con un
interludio manual de tag entre el slice 1b y el slice 2, tres paquetes de
guarda nuevos, y un movimiento de 13 archivos mas un cambio de importadores en
3 archivos — la misma complejidad que ya llevo a `proposal.md` y `design.md` a
exceder sus propios presupuestos. El contenido se mantiene solo como
checklist, sin relleno narrativo.

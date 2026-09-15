# Tareas: `pkg/graph.Impacted`

Fuente de verdad para el contrato y la justificación: `proposal.md`, `specs/impact-graph/spec.md`, `design.md`. Este archivo solo ordena y divide el trabajo. Ninguna tarea toca `internal/executors` ni `internal/pipelines` — `pkg/graph` es un paquete independiente, sin dependencias, así que ninguna tarea de este cambio lleva la marca de "riesgo mayor" del repositorio.

## Fase 1 — RED: tests que fallan para cada escenario del spec

Ref: `design.md` §4 (tabla escenario → caso de test), `strict_tdd: true`.

- [ ] 1.1 Scaffold de `pkg/graph/impacted.go` (solo la firma, cuerpo `panic("not implemented")`) y `pkg/graph/impacted_test.go`
- [ ] 1.2 Tests table-driven: Alcance Inverso Transitivo (directo, multinivel, fan-out, fan-in, desconectado-excluido)
- [ ] 1.3 Tests table-driven: Auto-Inclusión del Conjunto Modificado (leaf modificado sin dependientes, nodo modificado desconocido para `edges`)
- [ ] 1.4 Tests table-driven: Tolerancia a Entradas Malformadas (arista colgante) y Seguridad ante Ciclos (auto-arista, par mutuo, ciclo de tres nodos)
- [ ] 1.5 Tests table-driven: Colapso de Aristas Duplicadas
- [ ] 1.6 Tests table-driven: Salida Determinista y Ordenada (igualdad en llamadas repetidas, grafo vacío + conjunto modificado vacío, verificación nil-vs-slice-vacío)
- [ ] 1.7 Confirmar que todos los tests de 1.2–1.6 fallan por la razón correcta (`panic("not implemented")`), no por un error de compilación

## Fase 2 — GREEN: implementación

Ref: `design.md` §2 (algoritmo), §3 (estructuras de datos).

- [ ] 2.1 Implementar la construcción de adyacencia inversa a partir de `edges`
- [ ] 2.2 Implementar el recorrido BFS con un conjunto `visited` sembrado desde `changed`
- [ ] 2.3 Implementar la salida determinista: convertir `visited` a slice, `sort.Strings`, siempre no-nil
- [ ] 2.4 Ejecutar `go test -race ./pkg/graph/...` — todos los tests de la Fase 1 en verde

## Fase 3 — Verificación estructural + quality gates

Ref: requisito "Sin acoplamiento a ejecución ni a manifiesto" de spec.md; `openspec/config.yaml` testing/quality_tools.

- [ ] 3.1 Verificar que `pkg/graph` no importa nada de `internal/` (`go list -deps ./pkg/graph/...` o equivalente) — agregar como verificación estática (p. ej. un test de grafo de imports), no solo como chequeo manual
- [ ] 3.2 `go build -o shipwright .`
- [ ] 3.3 `go vet ./pkg/graph/...` y `golangci-lint run ./pkg/graph/...`
- [ ] 3.4 `gofmt` / `goimports` limpio
- [ ] 3.5 Verificación de cobertura: `go test -coverprofile=coverage/coverage.out -covermode=atomic ./pkg/graph/...` ≥ 90%

## Fase 4 — Release

Ref: `proposal.md` Alcance #4, Criterios de Éxito.

- [ ] 4.1 Abrir PR contra `develop`, incluyendo proposal/spec/design en ambos idiomas
- [ ] 4.2 Tras el merge, etiquetar un release (o registrar el SHA del commit de merge) al que `verimand-platform` pueda fijarse en su `go.mod`
- [ ] 4.3 Reportar la finalización de vuelta al issue #207 de `verimand-platform` / tarea S1.1 de su `tasks.md`, desbloqueando su PR3

## Gate

- [ ] Cada escenario en `specs/impact-graph/spec.md` tiene un test con nombre que pasa
- [ ] `pkg/graph` tiene cero imports de `internal/` (verificación estática de la Fase 3.1, re-verificada en la revisión del PR)
- [ ] `go build -o shipwright .` y `go test -race ./...` en verde para todo el repositorio, no solo `pkg/graph`

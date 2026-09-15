# Propuesta: `pkg/graph.Impacted` — alcance inverso para análisis de impacto de nodos modificados

## Intención

Verimand (`github.com/getsyntegrity/verimand-platform`, issue [#207](https://github.com/getsyntegrity/verimand-docs/issues/207)) necesita calcular, a partir de un grafo de dependencias de módulos y un conjunto de nodos modificados, cada nodo que depende transitivamente de un nodo modificado — para que su pipeline de CI ejecute solo lo que un cambio realmente afecta, y recurra de forma segura a ejecutar todo cuando no puede probar lo contrario.

Ese cálculo es un algoritmo de grafos genérico (alcance inverso desde un conjunto semilla), no una responsabilidad de ejecución de Shipwright. No tiene motivo para vivir en `internal/workflow/graph`, que es específico del `Step` del manifiesto e interno al motor propio de Shipwright — y los paquetes `internal/` no son importables desde otro módulo Go de todas formas. Verimand necesita importar este algoritmo como una dependencia ordinaria, tal como ya importa las interfaces de capacidad de `pkg/shipwright`.

**Éxito significa:** un nuevo paquete público, `pkg/graph`, que exporta una única función pura — `Impacted(edges map[string][]string, changed []string) []string` — sin dependencia de `manifest`, sin I/O, sin efectos secundarios, y con una batería de tests lo bastante exhaustiva como para que tanto Shipwright como Verimand puedan confiar en ella como primitiva sin acoplamiento.

## Alcance

### Dentro del alcance

| # | Entregable |
|---|---|
| 1 | Nuevo paquete público `pkg/graph`, hermano de `pkg/shipwright` |
| 2 | `Impacted(edges map[string][]string, changed []string) []string` — pura, determinista, sin I/O |
| 3 | Batería de tests exhaustiva (grafo vacío, conjunto de cambios vacío, nodo único, dependiente directo/multinivel, fan-out, fan-in, nodos desconectados, aristas duplicadas, nodo modificado desconocido, salida determinista, ciclos/grafo malformado) |
| 4 | Un release etiquetado (o commit fijado) que Verimand pueda agregar a su `go.mod` |

### Fuera del alcance (no-objetivos)

| No-objetivo | Límite |
|---|---|
| Cualquier cambio a `internal/workflow/graph` o al motor de ejecución/manifiesto | `Impacted` es un algoritmo independiente; el motor DAG existente no se toca |
| Escaneo de módulos/dependencias (`go list -m`, `go list -json`) | Responsabilidad propia de Verimand (`internal/ciimpact.BuildModuleGraph`); este paquete solo consume un mapa de aristas ya construido |
| Cualquier tipo o nomenclatura específica de Verimand (`ModuleGraph`, `ImpactPlan`, ...) | `Impacted` recibe únicamente IDs de nodo como strings opacos — sin acoplamiento al modelo de dominio de un llamador específico |
| Wiring de CLI, generación de manifiesto a partir de resultados de impacto | V7 de Verimand (`design.md` §2.3 del cambio `207-affected-module-detection`), no este paquete |

## Decisiones

### D1 — Nuevo dominio de capacidad: `impact-graph`, no `workflow-execution`

| Ítem | Posición |
|---|---|
| Candidato | `workflow-execution` (dominio existente — adyacente a DAG) |
| Decisión | Nuevo dominio: `impact-graph` |
| Justificación | `workflow-execution` es dueño de la resolución de providers, el ordenamiento topológico y la ejecución de grafos `manifest.Step` — está acoplado al esquema del manifiesto por diseño. `Impacted` está deliberadamente desacoplado de `manifest` (nodos como strings opacos, sin conocimiento de `Step`) para poder ser importado por un llamador — Verimand — que nunca tuvo conocimiento de un manifiesto de Shipwright. Incorporarlo a `workflow-execution` filtraría acoplamiento al manifiesto hacia un llamador que no debe tenerlo, o forzaría a `workflow-execution` a sostener un camino de código libre de manifiesto sin otra razón para existir ahí. |

### D2 — La convención de dirección coincide con `Node.Needs`

| Ítem | Posición |
|---|---|
| Convención | `edges[X]` = el conjunto de nodos de los que `X` depende directamente (dirección de dependencia, igual que la convención existente de `Node.Needs`/grafo de Build) |
| Justificación | `Impacted` calcula la inversa de esta relación internamente. Mantener la entrada pública en la misma dirección que toda otra representación de grafo en este código evita forzar a cada llamador (incluido Verimand) a invertir aristas antes de invocarla |

## Capacidades

### Capacidades nuevas

- `impact-graph`: alcance inverso sobre un grafo de nodos opacos — el contrato `Impacted`, sus garantías de determinismo/pureza, y su manejo de entradas malformadas (ciclos, aristas colgantes, nodos desconocidos).

### Capacidades modificadas

- Ninguna. Este cambio es aditivo; ningún contrato de capacidad existente cambia.

## Enfoque

1. Especificar el contrato de `Impacted` como requisitos verificables: pureza, determinismo (salida ordenada), inclusión propia de `changed`, tolerancia a aristas colgantes, terminación ante ciclos (palabras clave RFC 2119, escenarios Given/When/Then según las reglas de spec de este repositorio).
2. Diseñar el algoritmo: construir un mapa de adyacencia inversa a partir de `edges`, luego BFS/DFS desde cada nodo en `changed` con guarda de conjunto visitado (seguro ante ciclos por construcción, no por detección-y-rechazo).
3. Tests RED primero (`strict_tdd: true`), cubriendo la lista completa del punto 3 de Alcance, incluidos los casos de grafo malformado señalados como el riesgo más alto abajo.
4. Implementar `pkg/graph/impacted.go` hasta ponerlo en verde.
5. Etiquetar un release (o registrar el commit de merge) al que el `go.mod` de Verimand pueda fijarse.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `pkg/graph/` | Nueva | El paquete, `impacted.go` + `impacted_test.go` |
| `openspec/specs/impact-graph/` | Nueva | El contrato de capacidad (creado al archivar) |
| `go.mod` (este repo) | Sin efecto | Sin dependencia nueva — solo librería estándar |
| Todo lo demás | Sin efecto | Ningún paquete existente importa ni es importado por `pkg/graph` |

**Nota de compatibilidad con Dagger:** `Impacted` no tiene I/O ni exposición al sistema de tipos de Dagger — es una función Go simple, no una superficie de capability/provider, y no se invoca a través del sistema de módulos de Dagger. Sin impacto en el SDK de Dagger ni en la compatibilidad de pipeline-steps.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Manejo incorrecto de ciclos (bucle infinito o nodo omitido) | Media | La guarda de conjunto visitado es estructural, no un caso especial; los tests RED incluyen auto-aristas, pares mutuos, ciclos largos y diamantes de fan-in antes de la implementación |
| La firma `ImpactedFunc` de Verimand (`func(edges map[string][]string, changed []string) []string`) diverge de la firma exportada de este paquete | Baja | La firma se copia textualmente de `design.md` §2.2 de `verimand-platform`, escrito precisamente contra este contrato |
| Dependencia bloqueante: el PR3 de Verimand (issue #207) no puede iniciar hasta que esto se publique y quede fijado | Alta (por diseño) | Esta propuesta existe específicamente para desbloquearlo — secuenciar el release antes de comunicar la finalización de vuelta a Verimand |

## Plan de rollback

Puramente aditivo — un paquete nuevo sin consumidores existentes dentro de este repositorio.

- Revertir el commit de merge; ningún otro código referencia `pkg/graph`, así que nada más se rompe.
- Sin migración de estado, datos, configuración o release.
- Verificación tras el revert: `go build -o shipwright .` y `go test -race ./...` en verde.

## Dependencias

- Ninguna (solo librería estándar).

## Criterios de éxito

- [ ] `pkg/graph.Impacted(edges map[string][]string, changed []string) []string` existe, está exportada, y coincide exactamente con la firma de `design.md` §2.2 de `verimand-platform`
- [ ] La salida está ordenada y deduplicada en cada llamada (determinismo)
- [ ] Cada nodo en `changed` aparece en la salida, incluidos nodos ausentes del conjunto de claves de `edges`
- [ ] Un nodo con una arista colgante (que nombra un nodo ausente de `edges`) no genera panic
- [ ] Un grafo cíclico o malformado termina y no genera panic (auto-aristas, pares mutuos, ciclos más largos)
- [ ] Las aristas duplicadas colapsan en una única arista lógica
- [ ] Los casos de fan-out y fan-in están ambos cubiertos por un test que pasa
- [ ] `go build -o shipwright .` y `go test -race ./...` en verde; cobertura ≥ 90%
- [ ] Existe un release etiquetado o commit fijado que `verimand-platform` pueda agregar a `go.mod`

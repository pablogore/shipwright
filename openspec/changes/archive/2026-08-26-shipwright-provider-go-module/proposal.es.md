# Propuesta: extraer `internal/capabilities/` a un módulo `providers/go` independiente

> Versión en español de `proposal.md`. Ante cualquier discrepancia, **la versión en inglés es la fuente de verdad**.

**Este cambio es una prueba de fuego, no una refactorización.** Plantea una sola pregunta sobre el contrato recién integrado por `shipwright-public-module-api` (PRs #147–#158): *¿es `pkg/shipwright` suficiente para implementar un proveedor real desde fuera del módulo principal?* La exploración ya respondió: las cinco implementaciones de capacidades y los ocho archivos de prueba importan únicamente la biblioteca estándar, `dagger.io/dagger` y `pkg/shipwright`, con **cero importaciones de `internal/**`**. Este cambio convierte esa observación en un límite verificado por el compilador.

## Intención

| | |
|---|---|
| **Problema** | La suficiencia del contrato público se afirma en documentación, no se demuestra en la compilación. Nada impide que una capacidad futura acceda a `internal/**` y vuelva insuficiente el contrato de forma silenciosa. |
| **Por qué ahora** | El contrato está congelado y es estable desde la primera versión. Probarlo contra un consumidor real es más barato justo después del congelamiento y antes de que otros proveedores multipliquen el costo. |
| **Éxito** | `providers/go` compila y se prueba como módulo propio contra el contrato público únicamente. La prueba de extremo a extremo `diamond.yaml` se ejecuta igual. Cambia la ubicación, no el comportamiento. |
| **El fallo también informa** | Si esto requiere *cualquier* cambio en `pkg/shipwright/**`, la prueba de fuego **falló** y la abstracción faltante DEBE reportarse de forma explícita, no parchearse. |

## Alcance

### Dentro del alcance

| # | Entregable |
|---|---|
| 1 | Mover los cinco archivos fuente y los ocho de prueba de `internal/capabilities/` a `providers/go/` (mover, nunca duplicar) |
| 2 | `providers/go/go.mod` — módulo propio que requiere solo `github.com/pablogore/shipwright` (por `pkg/shipwright`) y `dagger.io/dagger` |
| 3 | El `go.mod` raíz suma `require github.com/pablogore/shipwright/providers/go vX.Y.Z` sobre una **etiqueta real**, no un `replace` local (D4) |
| 4 | `go.work` en la raíz (`use .`, `use ./providers/go`) para que `./...` abarque el nuevo módulo, más una prueba de resguardo (D1) |
| 5 | Cambio de importación en `internal/workflow/providers/register.go`, **atómico con el movimiento** (D3) |
| 6 | Nuevo requisito de especificación en `public-module-api` que vuelve permanente la implementabilidad externa (D5) |
| 7 | Etiqueta de Git `providers/go/vX.Y.Z` publicada como parte de la entrega de este cambio, para que `go install` resuelva (D4) — **no opcional** |

### Fuera del alcance (no-objetivos)

| No-objetivo | Límite |
|---|---|
| Cualquier otro proveedor (container/docker, govulncheck como módulo, nomad, rust) | Cambios posteriores, **uno a la vez**, solo tras probar este patrón de extremo a extremo |
| Volver público `internal/workflow/providers` (`Registry`, `RegisterBuilder`, `WithSchema`, `Values`) | Es el verdadero bloqueante para proveedores externos de terceros. Cuestión arquitectónica separada y posterior. **Este cambio prueba únicamente un límite de módulo dentro del repositorio; no habilita repositorios externos.** |
| Una **cadencia de liberación** independiente y continua para `providers/go` | Acotado por D4. La **primera** etiqueta `providers/go/vX.Y.Z` ahora está dentro del alcance, porque `go install` no puede resolver el módulo sin ella. Queda fuera: un proceso de liberación recurrente, un changelog del proveedor y si CI/goreleaser automatiza las etiquetas *futuras* |
| **Cualquier cambio en `pkg/shipwright/**`** | Restricción estricta. Necesitar un cambio ahí significa que la prueba de fuego falló |
| Modificar el aislamiento de `.dagger/` | Se preserva exactamente. `.dagger/` NUNCA debe aparecer en `go.work` |

## Decisiones

### D1 — **se añade** `go.work`, divergiendo del precedente de `.dagger`

| | |
|---|---|
| **Decisión** | Versionar un `go.work` en la raíz con `use .` y `use ./providers/go`. Nunca `use ./.dagger`. |
| **Por qué no replicar `dagger-test`** | Los módulos no son análogos. La raíz **importa** `providers/go` (vía `RegisterDefaults`) pero nunca importa `.dagger`. Además las pruebas de `.dagger` necesitan un motor activo (`dagger run`) — por eso están excluidas de `make test`; las de `providers/go` son `go test` normales. |
| **Argumento decisivo** | Sin `go.work`, `go build ./...` en la raíz igual compila el *código fuente* de `providers/go` (es una dependencia), así que los errores de compilación se detectan de todos modos. Lo que se pierde en silencio son sus **ocho archivos de prueba** —incluida la prueba dorada de AST `naming_test.go`— y la cobertura de `govulncheck`. Un cambio cuyo propósito es demostrar que un contrato se sostiene no puede dejar su evidencia fuera de CI. |
| **Costo, dicho con honestidad** | `go.work` es un archivo nuevo en la raíz que cambia la semántica de resolución para todo colaborador y altera el comportamiento de `go mod tidy` en modo workspace. |
| **Estratificación que acota el impacto** | El `require` del `go.mod` raíz resuelve la *resolución* (para que `GOWORK=off` y las compilaciones de goreleaser sigan funcionando). `go.work` hace exactamente una cosa: que `./...` abarque los paquetes y pruebas del nuevo módulo. **Enmendado por D4**: ese `require` debe apuntar a una etiqueta real, no a un `replace`. |
| **Resguardo (obligatorio)** | El riesgo de añadir `use ./.dagger` por accidente es error humano, no automático: una directiva `use` nunca es recursiva. Se mitiga con una **prueba obligatoria en la raíz** que verifique que `.dagger` nunca aparece en `go.work`, no con un comentario. |

### D2 — ruta del módulo `github.com/pablogore/shipwright/providers/go`

| | |
|---|---|
| **Ruta** | `github.com/pablogore/shipwright/providers/go`, directorio `providers/go/`. Módulo anidado dentro del repositorio; sin separación a un repositorio externo. |
| **Por qué esta ruta** | El nombre del directorio refleja la familia de proveedores del manifiesto (`"go"`, `"go-test"`) y es consistente con futuros `providers/rust`, `providers/nomad`. |
| **Defecto conocido** | `package go` es un **error de sintaxis**: `go` es palabra reservada. La cláusula `package` DEBE diferir del último elemento de la ruta (por ejemplo `package golang`), y el único importador la aliasa: `import golang ".../providers/go"`. |
| **Alternativa considerada** | Renombrar a `providers/golang` elimina la discrepancia pero rompe la correspondencia 1:1 entre directorio y familia de proveedores a cambio de un alias en un solo archivo. Descartada. |
| **Verificar durante la aplicación** | Confirmar que `revive` (reglas por defecto, `.golangci.yml`) no marque la discrepancia entre nombre de paquete y ruta de importación. |

### D3 — el cambio de importación de `RegisterDefaults` viaja de forma **atómica** con la extracción

Innegociable, y en realidad no es una bifurcación: borrar `internal/capabilities/` sin actualizar `register.go` deja el árbol sin compilar. La exploración no encontró motivo para diferirlo. Nótese que `register.go` también importa `internal/workflow/interp` y permanece en el módulo raíz: la dirección de dependencia es núcleo→proveedor, nunca al revés.

### D4 — `go install` es una vía de distribución real, así que `providers/go` se entrega como **versión etiquetada y resoluble**

La revisión 1 registró esto como un riesgo hipotético de impacto bajo. No lo es. `go install github.com/pablogore/shipwright@latest` es una vía de distribución real y prevista para este repositorio, lo que lo promueve de nota al pie a restricción de diseño resuelta.

| | |
|---|---|
| **Decisión** | El `go.mod` raíz versionado resuelve `providers/go` mediante un `require` sobre una **versión publicada real**, respaldada por una etiqueta de Git. Un `replace` local NO DEBE ser el mecanismo de resolución versionado. |
| **Por qué `replace` no puede funcionar** | `go install pkg@version` se ejecuta **sin módulo principal**: ignora `go.work` y cualquier `go.mod` del directorio actual o superiores, y descarga el módulo destino desde su origen o proxy. Un `replace ... => ./providers/go` relativo no tiene checkout contra el cual resolverse. Además se rechaza de plano: el `go.mod` del módulo destino no debe contener directivas que lo hagan comportarse distinto que si fuera el módulo principal, y `replace`/`exclude` están nombradas. |
| **Agravado por el árbol actual** | El `go.mod` raíz hoy tiene **cero** directivas `replace`, así que `go install` funciona ahora mismo. Añadir una haría que *este cambio* lo rompiera, y no que expusiera una carencia previa. |
| **Mecanismo (convención propia de Go, no inventada)** | Un módulo anidado en un mismo repositorio se publica con una **etiqueta prefijada por su ruta**: el módulo `github.com/pablogore/shipwright/providers/go` se libera como la etiqueta `providers/go/v0.1.0`, distinta del `v0.1.0` del módulo raíz. Existiendo esa etiqueta, `go install`/`go get` y el proxy de módulos la resuelven por la vía normal para un repositorio público. |
| **Relación con D1** | D1 no cambia. `go.work` es un mecanismo local del workspace y de tiempo de desarrollo; `go install pkg@version` lo ignora, igual que cualquier consumidor que dependa de este módulo, por lo que nunca participa en la resolución publicada. `replace` PUEDE quedar como comodidad de desarrollo local: el **estado versionado y etiquetado debe ser resoluble por `go install` sin él**. |
| **Consecuencia de orden (para diseño)** | `providers/go` requiere el módulo raíz mientras la raíz requiere `providers/go`. Go permite ciclos a nivel de módulo (las importaciones de paquetes siguen siendo acíclicas), pero impone un orden de etiquetado que diseño/tareas deben concretar. |
| **Deliberadamente abierto** | El número exacto de la primera versión (`v0.1.0` es el valor esperado, consistente con el principio de proveedor versionado de forma independiente del Eje 5 de `COMPATIBILITY.md`) y si la etiqueta se crea a mano o la automatiza goreleaser/CI. **El requisito de la etiqueta está decidido y no es opcional; solo se difiere su mecánica.** |

### D5 — la implementabilidad externa pasa a ser un **requisito permanente de especificación**, no un criterio de éxito puntual

La prueba de fuego no debe caducar al integrar este cambio. La especificación `public-module-api` suma un requisito permanente, de modo que una capacidad futura que acceda a `internal/**` incumpla la especificación en lugar de pasar inadvertida. La lista de criterios de éxito verifica este cambio; el requisito de especificación gobierna todos los siguientes. Ambos, no uno u otro.

## Capacidades

### Capacidades nuevas

- Ninguna.

### Capacidades modificadas

- `public-module-api`: añadir un **requisito permanente** (D5) de que el contrato público DEBE ser suficiente para implementar toda capacidad publicada desde un **módulo Go separado que no importe nada de `internal/**`**. Ese es el punto del cambio: una propiedad permanente y verificable del contrato que sobrevive a este cambio, no una observación puntual registrada solo en una lista de verificación. Sin otros cambios de especificación: nombres de proveedor, resolución y ejecución quedan idénticos.

## Enfoque

1. Añadir `providers/go/go.mod`; hacer `git mv` de los cinco fuentes y las ocho pruebas.
2. Añadir el `require` en la raíz; añadir `go.work`; añadir la prueba de resguardo de `.dagger`. (`replace` se permite aquí solo como ayuda de *desarrollo local* mientras la etiqueta aún no exista — D4.)
3. Cambiar la importación de `register.go` (atómico con el paso 1) y borrar `internal/capabilities/`.
4. Verificar que no hizo falta ningún cambio en `pkg/shipwright/**`; si lo hizo, **detenerse y reportar la prueba de fuego fallida**.
5. Reejecutar la prueba de extremo a extremo `examples/workflow/diamond.yaml` como verificación de regresión de comportamiento.
6. **Crear la etiqueta `providers/go/vX.Y.Z`** y fijar el `go.mod` raíz versionado en esa versión, sin ningún `replace` remanente (D4). Diseño/tareas definen el orden, el número de versión y si el etiquetado es manual o automatizado.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/capabilities/**` | **Eliminada** | Movida a `providers/go/`, sin copias duplicadas |
| `providers/go/**` | **Nueva** | 5 fuentes + 8 pruebas + `go.mod`/`go.sum` |
| `internal/workflow/providers/register.go` | Modificada | Solo cambio de ruta de importación; los cinco registros idénticos |
| `go.mod` (raíz) | Modificada | `require` del nuevo módulo sobre una **etiqueta publicada**; sin `replace` versionado (D4) |
| `go.work` | **Nueva** | `use .` y `use ./providers/go` — solo tiempo de desarrollo, nunca parte de la resolución publicada |
| Etiqueta de Git `providers/go/vX.Y.Z` | **Nueva** | Etiqueta de módulo anidado prefijada por ruta; necesaria para que `go install` resuelva (D4) |
| `.dagger/`, objetivo `dagger-test` del `Makefile` | Sin cambios | Aislamiento preservado; la nueva prueba de resguardo lo protege |
| `.github/workflows/ci.yml` | Sin cambios | `./...` en las etapas de build/test/security ya abarca `providers/go` vía `go.work`; sin pasos nuevos |
| `pkg/shipwright/**` | **Debe quedar intacto** | Restricción estricta |
| `COMPATIBILITY.md` | Nota al pie opcional | El Eje 5 ya cubre correctamente la exclusión |

**Compatibilidad con Dagger:** `dagger.io/dagger` v0.21.8 sin cambios; `providers/go/go.mod` DEBE fijar la misma versión para evitar dos versiones del cliente en una compilación. Sin impacto en pasos de pipeline ni en el esquema del manifiesto.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Que alguien añada después `use ./.dagger` y colapse el aislamiento establecido en D-B de `design.md` | Media | Prueba de resguardo obligatoria (D1), no un comentario |
| ~~`go install` no puede resolver el módulo anidado sin etiqueta~~ — **resuelto en D4, ya no es un riesgo abierto** | — | Se calificó de baja probabilidad suponiendo que no existía vía de `go install`. Esa suposición era falsa. Ahora es restricción de diseño: `require` sobre etiqueta real, sin `replace` versionado |
| Que se olvide la etiqueta `providers/go/vX.Y.Z`, o se cree en el orden equivocado respecto a la versión raíz | **Alta**: es un paso manual fuera del flujo normal de `git merge`, y los dos módulos se requieren mutuamente | El criterio de éxito de abajo exige un `go install` limpio desde un estado publicado; diseño/tareas deben especificar el orden de etiquetado en lugar de dejarlo para el día de la liberación |
| Que las versiones de `dagger.io/dagger` diverjan entre módulos | Media | Fijar la misma versión; `go.work` expone la discrepancia en tiempo de compilación |
| Colaboradores no familiarizados con la semántica de `go mod tidy` en modo workspace | Media | Documentar la estratificación de los dos archivos (`replace` = resolución, `go.work` = alcance de `./...`) en la descripción del PR |
| Que la discrepancia entre nombre de paquete y ruta (D2) active algún linter | Baja | Verificar contra `.golangci.yml` durante la aplicación |
| **Que la prueba de fuego falle de verdad**: que algo requiera `internal/**` o un cambio de contrato | Baja (la exploración encontró cero importaciones así) | Reportarlo de forma explícita como defecto del contrato. NO ampliar `pkg/shipwright` en silencio para que la extracción pase |

## Plan de reversión

Estructural, no de comportamiento, pero no puramente aditivo (elimina `internal/capabilities/`).

- La extracción, el cambio en `register.go` y `go.work` forman **un único límite de reversión**. Revertir el movimiento sin el cambio de importación deja el árbol sin compilar.
- PR único: `git revert -m 1 <sha>`. PRs encadenados: revertir en **orden inverso de integración**.
- Sin migración de estado, datos, configuración ni versiones; sin consumidores externos. Los archivos de manifiesto no se tocan.
- **Puerta de un solo sentido que introduce D4:** una etiqueta publicada es inmutable en el proxy de módulos de Go. Una etiqueta `providers/go` defectuosa se sustituye con una nueva etiqueta de parche, nunca se borra. Es la única parte del cambio que no se puede revertir con `git revert`.
- Verificación posterior a la reversión: `go build -o shipwright .` y `go test -race ./...` en verde, y `diamond.yaml` sigue resolviéndose.

## Dependencias

- Go ≥ 1.18 para workspaces (el repositorio usa 1.26.1 — cumplido).
- Sin dependencias de terceros nuevas.

## Criterios de éxito

- [ ] `providers/go/go.mod` existe y depende solo del módulo que contiene `pkg/shipwright`, de `dagger.io/dagger` y de la biblioteca estándar — verificado con `go list -m all` desde dentro de `providers/go/`
- [ ] `go build ./...` y `go test -race ./...` funcionan desde dentro de `providers/go/` como módulo independiente
- [ ] `go build ./...` y `go test -race ./...` en la raíz funcionan, **sí** recorren `providers/go` (D1) y siguen **sin** recorrer `.dagger/`
- [ ] Una prueba falla si alguna vez se añade `.dagger` a `go.work`
- [ ] `internal/capabilities/**` queda eliminado — una búsqueda en todo el repositorio encuentra exactamente una copia de cada uno de los cinco tipos
- [ ] `RegisterDefaults` registra los cinco con nombres idénticos: `"go"`, `"go-test"`, `"golangci-lint"`, `"govulncheck"`, `"container"`
- [ ] `examples/workflow/diamond.yaml` se resuelve y ejecuta igual que antes de la extracción
- [ ] **`git diff` no toca ningún archivo bajo `pkg/shipwright/`**
- [ ] La cobertura ≥ 90 % se mantiene en ambos módulos
- [ ] **El `go.mod` raíz versionado no contiene ninguna directiva `replace`**: `go install github.com/pablogore/shipwright@latest` sigue funcionando (D4)
- [ ] La etiqueta de Git `providers/go/vX.Y.Z` existe, y `go install github.com/pablogore/shipwright@<etiqueta-raíz>` resuelve `providers/go` en un entorno limpio sin checkout local

---

*Revisión 2 (enmienda acotada):* el usuario confirmó que `go install github.com/pablogore/shipwright@latest` es una vía de distribución real y prevista. Esa sola respuesta promovió un riesgo al pie calificado de bajo a **D4** (módulo anidado etiquetado, sin `replace` versionado) y añadió al alcance un paso de etiquetado no opcional. **D5** se registró de forma explícita para eliminar cualquier ambigüedad: la implementabilidad externa es un requisito permanente de especificación, no solo un ítem de la lista de este cambio. D1–D3 no cambian; de D1 solo se enmendó la mitad relativa a `replace`.

*Nota de desviación:* excede el presupuesto de 450 palabras de la skill. Causa: el cambio depende de decisiones que el orquestador exigió justificar en lugar de asumir por defecto (D1–D5), y D1 diverge de un precedente ya establecido en el repositorio, lo que no puede registrarse con credibilidad sin su razonamiento. El contenido está comprimido en tablas.

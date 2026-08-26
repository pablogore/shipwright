# Diseño: Extraer `internal/capabilities/` a un módulo independiente `providers/go`

> Versión en español. La versión canónica y fuente de verdad ante cualquier
> conflicto es `design.md` (inglés).

## Enfoque técnico

Dos PR encadenados con una etiqueta git manual entre ambos, porque la etiqueta no
puede existir antes que el código y el `require`+`go.sum` no pueden existir antes
que la etiqueta.

| Tramo | Contenido | Mecanismo de resolución |
|---|---|---|
| **1** | Crear el módulo `providers/go`, `git mv` de 13 archivos, cambiar todos los importadores, añadir `go.work`, añadir 3 tests guardianes, borrar `internal/capabilities/` | `replace ... => ./providers/go` temporal en el `go.mod` raíz (autorizado explícitamente por D4) |
| **etiqueta** | `git tag providers/go/v0.1.0 <sha del merge del tramo 1> && git push origin providers/go/v0.1.0` | — (tarea manual, no es un PR) |
| **2** | Eliminar el `replace`, conservar `require .../providers/go v0.1.0`, regenerar el `go.sum` raíz, añadir el test guardián de "sin `replace`", verificar `go install` con caché limpia | Etiqueta publicada con prefijo de ruta (estado final de D4) |

El tramo 1 queda en verde **tanto** en modo workspace como con `GOWORK=off` (el
`replace` cubre la resolución fuera del workspace), así que el repositorio nunca
queda incompilable entre tramos. Única restricción durante la ventana: **no cortar
una etiqueta de release de la raíz**, porque una etiqueta raíz con un `replace` de
directorio no resolvería con `go install`.

## Decisiones de arquitectura

### Decisión: contenido de `go.work` y guardián de `.dagger` (mecánica de D1)

**Elección** — archivo exacto que se versiona, nada más:

```
go 1.26.1

use (
	.
	./providers/go
)
```

El guardián vive en el **módulo raíz** (`go.work` es un archivo de raíz) como un
paquete nuevo `internal/workspaceguard/` que replica `internal/daggerpin/` uno a
uno: `work.go` (parseo y extracción) + `work_test.go` (aserción sobre el archivo
real y casos sintéticos de fallo cerrado en `t.TempDir()`). Parsea `go.work` con
`golang.org/x/mod/modfile.ParseWork` — ya es dependencia directa de la raíz
(`golang.org/x/mod v0.35.0`, usada por `daggerpin`). Aserciones:

1. El conjunto `use` es exactamente `{".", "./providers/go"}` (lista blanca:
   falla cualquier añadido, no solo `.dagger`).
2. Ningún `use` tiene un segmento limpio igual a `.dagger` o `dagger`, con un
   mensaje dedicado que nombra el aislamiento D-B de `design.md` que se perdería.
3. Un `go.work` ausente o no parseable falla cerrado (`t.Fatal`), igual que
   `daggerpin` ante un `dagger.json` ausente.

**Alternativas consideradas** — un comentario dentro de `go.work` (ya rechazado en
la propuesta); un paso `grep` en CI. **Fundamento** — el guardián basado en
texto/AST es el patrón establecido del repositorio (`daggerpin`,
`api_golden_test.go`): sin motor, sin red, se ejecuta localmente en milisegundos,
y la forma de lista blanca detecta tanto un `use ./.dagger` accidental como
cualquier otro miembro del workspace no revisado.

### Decisión: forma del módulo `providers/go` (mecánica de D2)

**Elección** — `providers/go/go.mod`:

```
module github.com/pablogore/shipwright/providers/go

go 1.26.1

require (
	dagger.io/dagger v0.21.8
	github.com/pablogore/shipwright v0.0.0-<pseudoversión del commit previo a la extracción>
)
// bloque indirect generado por `GOWORK=off go mod tidy` (querybuilder de dagger,
// gqlgen, otel, golang.org/x/* — subconjunto estricto del indirect de la raíz)
```

`go 1.26.1` (idéntico a la raíz, no el 1.26.5 de `.dagger`, que deliberadamente no
es análogo). `dagger.io/dagger v0.21.8` copiado literalmente del pin de la raíz;
**sin paquete equivalente a `daggerpin` dentro de `providers/go`.** En su lugar,
`internal/daggerpin/pin_test.go` gana un test,
`TestProvidersGoDaggerVersionMatchesRoot`, que reutiliza `GoModDaggerVersion` sin
cambios contra `providers/go/go.mod`.

*Fundamento*: la paridad de pin de Dagger entre módulos ya es la función declarada
de `daggerpin` (Eje 4 de COMPATIBILITY.md), `GoModDaggerVersion` ya cubre el caso
"gana el replace" y el fallo cerrado ante rutas locales, y esto añade **cero**
código de producción nuevo. `go.work` por sí solo no basta: en modo workspace Go
elige silenciosamente una versión vía MVS, así que la divergencia compilaría bien
y solo se manifestaría en consumidores externos.

Nombre de paquete `golang`; la disposición de archivos es un renombrado 1:1
estricto (ningún archivo se renombra ni se divide):

| Desde | Hacia |
|---|---|
| `internal/capabilities/{gobuilder,gounittester,golinter,govulnscanner,containerpublisher}.go` | `providers/go/<mismo nombre>.go` |
| `internal/capabilities/{…}_test.go`, `internal_test.go`, `integration_test.go`, `naming_test.go` | `providers/go/<mismo nombre>` |

### Decisión: el cambio de importadores es **más amplio de lo que decía la propuesta** (corrección de D3)

La exploración y la propuesta dicen "el único importador". El árbol tiene **tres**:

| Archivo | Cambio |
|---|---|
| `internal/workflow/providers/register.go` | import + 5 referencias de tipo |
| `internal/app/container.go` (`BuildCapabilities`) | import + 5 referencias de tipo |
| `internal/app/container_capabilities_test.go` | import + 6 aserciones de tipo |

Cada uno es una edición pura de identificador/import. Verificado contra el código:
los cinco tipos son structs simples con campos exportados (`Client`, `Config`,
`GoVersion`); no hay constructores, ni cambios de firma, ni renombres de campos.
**Cero cambio de comportamiento confirmado** — pero el diff del tramo 1 y el plan
de tests RED deben cubrir `internal/app`, no solo `internal/workflow/providers`.

```go
- "github.com/pablogore/shipwright/internal/capabilities"
+ golang "github.com/pablogore/shipwright/providers/go"
…
- return &capabilities.GoBuilder{Client: client, Config: …}
+ return &golang.GoBuilder{Client: client, Config: …}
```

`register.go` conserva intactos sus imports de `internal/workflow/interp` y
`pkg/shipwright`: la dirección de dependencia sigue siendo núcleo→proveedor.

Dos archivos movidos **no** son renombrados idénticos y requieren ediciones reales:
`naming_test.go` (fija `pkgs["capabilities"]` → `pkgs["golang"]`, más sus
referencias documentales a `internal/capabilities`) y la cláusula de paquete y el
comentario de documentación de cada archivo (`Package capabilities …` →
`Package golang …`, documentando la discrepancia deliberada entre directorio y
nombre de paquete que obliga al alias de import).

### Decisión: romper el ciclo de módulos anclando **hacia atrás en el tiempo** (mecánica de D4)

El problema del huevo y la gallina es real: la raíz requiere `providers/go` para
`RegisterDefaults`, y `providers/go` requiere la raíz para `pkg/shipwright`.
**Elección**: anclar el requisito de raíz de `providers/go` a una *pseudoversión
del commit previo a la extracción* (la base de merge del tramo 1 en `develop`, que
ya contiene el `pkg/shipwright` congelado). El `go.mod` raíz de ese commit no tiene
requisito sobre `providers/go`, así que el grafo de módulos es acíclico:

```
raíz@HEAD ──requiere──▶ providers/go@v0.1.0 ──requiere──▶ raíz@v0.0.0-…-<sha previo>
                                                                  │
                                                  (sin requisito a providers/go) ──▶ ∎
```

MVS selecciona entonces el módulo principal para la ruta raíz, y ningún `go.mod`
nombra jamás una versión que todavía no existe. Las pseudoversiones no necesitan
ninguna etiqueta, y el repositorio ya usa una
(`github.com/pablogore/kit-logger v0.1.2-0.2026…`).

**Alternativas consideradas** — (a) cortar primero una etiqueta raíz para que
`providers/go` requiera una versión real: añade un evento de release a un cambio
estructural y aun así deja un ciclo verdadero en el grafo; (b) requerir la raíz en
la versión *posterior* a la extracción: ciclo genuino, y esa versión no existe
cuando se corta la etiqueta. **Fundamento** — la opción elegida es la única en la
que cada artefacto referencia solo algo que ya existe.

**Versión: `providers/go/v0.1.0`** — confirmada. `v0.x` declara "sin promesa de
compatibilidad mientras se prueba el patrón", que es exactamente lo que ya dice el
Eje 5 de COMPATIBILITY.md sobre las versiones de proveedor ("no cubiertas por
ninguna garantía de Shipwright"), y `v0.x`/`v1.x` evita la regla del sufijo mayor
`/vN` sobre un elemento de ruta que ya es una palabra clave de Go.

**El etiquetado es manual, único y una tarea — no automatización.** `.goreleaser`
construye solo el binario raíz; un proceso recurrente de release de proveedores
está explícitamente fuera de alcance en la propuesta. Etiquetar el commit de merge
del tramo 1 en `develop` es deliberado: Go resuelve etiquetas sin importar la rama,
y la línea de versión del proveedor es independiente de la de release de la raíz
(Eje 5). Consecuencia: el contenido de `providers/go/` debe ser **final en el tramo
1** — una corrección posterior cuesta `v0.1.1`, porque una etiqueta publicada es
inmutable en el proxy (la única puerta de un solo sentido de este cambio).

### Decisión: guardián de implementabilidad externa = escaneo AST de imports **dentro** de `providers/go` (mecánica de D5)

**Elección** — `providers/go/internalimport_test.go`, `package golang_test`:
localizar el directorio del módulo con `runtime.Caller(0)`, `parser.ParseDir`
**incluyendo** archivos `_test.go`, recolectar todas las rutas de import y fallar
nombrando el archivo y el import infractor si alguna ruta tiene un segmento
`internal`.

**Fundamento decisivo** — *el compilador de Go no lo impide aquí.* La visibilidad
de `internal/**` se basa en **prefijo de ruta**, no en módulo:
`github.com/pablogore/shipwright/providers/go` está bajo
`github.com/pablogore/shipwright/`, así que `import ".../internal/workflow/interp"`
desde el módulo nuevo **compila sin problema** incluso cruzando el límite de
módulo. Sin este test, toda la propiedad que persigue el cambio queda sin exigir.

**Alternativas consideradas** — (b) un paso de CI que compile `providers/go` de
forma aislada: no prueba nada adicional (ver arriba), no se puede ejecutar
localmente, falla tarde y en remoto, y cuesta una etapa de CI; (c) una aserción de
shell sobre `go list -m all`: necesita red y verifica dependencias, no imports.
Ambas rechazadas.

Escanear también los archivos de test es intencional: un test que alcance
`internal/**` significaría que el módulo solo es implementable desde fuera en su
mitad de producción.

## Flujo de datos

Dirección a nivel de paquete (sin cambios, en un solo sentido):

    manifiesto ──▶ internal/workflow/providers.RegisterDefaults ──▶ providers/go (golang.*)
    internal/app.BuildCapabilities ──────────────────────────────▶ providers/go (golang.*)
                                                                         │
                                                                   pkg/shipwright
                                                                   dagger.io/dagger

Resolución de módulo según el consumidor:

| Consumidor | Modo | `providers/go` se resuelve desde |
|---|---|---|
| Contribuidor / CI en el checkout | workspace (autodetectado) | `./providers/go` vía `use` |
| `GOWORK=off go build ./...` | módulo único | tramo 1: `replace`; tramo 2: etiqueta `v0.1.0` |
| `go install github.com/pablogore/shipwright@<tag>` | sin módulo principal, `go.work` ignorado | etiqueta `v0.1.0` desde el proxy |

**`go.work` se autodetecta** desde el directorio actual hacia arriba: quien nunca
ejecutó `go work init` igualmente queda en modo workspace una vez versionado el
archivo. Por eso "no usar `go.work`" significa `GOWORK=off`, `go install` o un
importador externo, y los tres están cubiertos en la tabla anterior. Versionar
`go.work` **y** `go.work.sum` si se genera (recomendación del propio Go para un
workspace de un solo repositorio); `go work sync` es el comando de mantenimiento.

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `go.work` | Crear | `use .` + `use ./providers/go`, `go 1.26.1` |
| `go.work.sum` | Crear (si se genera) | Lockfile del workspace |
| `providers/go/go.mod`, `go.sum` | Crear | Módulo nuevo; `go.sum` vía `cd providers/go && GOWORK=off go mod tidy` (una descarga de red de la pseudoversión raíz) |
| `providers/go/{5 fuentes + 8 tests}` | Crear (`git mv`) | Ediciones de cláusula de paquete y documentación; edición de clave en `naming_test.go` |
| `providers/go/internalimport_test.go` | Crear | Guardián D5 |
| `internal/capabilities/**` | Borrar | Movido: queda exactamente una copia de cada tipo |
| `internal/workspaceguard/{work.go,work_test.go}` | Crear | Guardián D1 |
| `internal/daggerpin/pin_test.go` | Modificar | +1 test: paridad de pin de Dagger para `providers/go` |
| `internal/workflow/providers/register.go` | Modificar | Alias de import + 5 referencias |
| `internal/app/container.go` | Modificar | Alias de import + 5 referencias |
| `internal/app/container_capabilities_test.go` | Modificar | Alias de import + 6 aserciones |
| `go.mod` (raíz) | Modificar | Tramo 1: `require` + `replace` temporal. Tramo 2: `replace` eliminado |
| `go.sum` (raíz) | Modificar | Solo tramo 2: hashes de `providers/go@v0.1.0` |
| `COMPATIBILITY.md` | Modificar | Línea 46: `internal/capabilities/**` → `providers/go/**` |
| `pkg/shipwright/**` | **Sin cambios** | Restricción dura; se verifica con `git diff --stat -- pkg/shipwright/` |

## Interfaces y contratos

Sin interfaces nuevas. El único contrato nuevo es la discrepancia entre nombre de
paquete y ruta de import, documentada una vez en la documentación del paquete:

```go
// Package golang provides Shipwright's Go-toolchain capability
// implementations … The package name deliberately differs from the last path
// element of github.com/pablogore/shipwright/providers/go, because `go` is a
// Go keyword and `package go` is a syntax error; importers alias it:
//   import golang "github.com/pablogore/shipwright/providers/go"
package golang
```

Verificación de linter para la fase de aplicación: `.golangci.yml` habilita
`revive` solo con reglas por defecto — `var-naming` no marca la discrepancia entre
nombre de paquete y ruta (eso es `stylecheck` ST1003 / `importas`, ninguno
habilitado). Se espera limpio; verificar igualmente con `golangci-lint run`.

## Estrategia de pruebas

TDD estricto: cada guardián se escribe en RED primero (la aserción existe antes que
el artefacto que verifica), y cada test movido debe verse fallar en su nueva
ubicación antes de mover su archivo fuente.

| Nivel | Qué se prueba | Enfoque | Orden RED |
|---|---|---|---|
| Unitario | Lista blanca de `go.work` + rechazo de `.dagger` + archivo ausente | `modfile.ParseWork` sobre el archivo real; casos sintéticos en `t.TempDir()` | Escribir `work_test.go` antes de que exista `go.work` → falla cerrado |
| Unitario | `providers/go` no importa `internal/**` | `parser.ParseDir` incl. `_test.go`, autolocalización con `runtime.Caller` | Escribir antes de mover las fuentes → falla por paquete ausente |
| Unitario | Paridad de pin de Dagger entre 3 archivos | Extender `daggerpin` (reutiliza `GoModDaggerVersion`) | Escribir antes de que exista `providers/go/go.mod` |
| Unitario | Golden de nombres tras el movimiento | `naming_test.go` con `pkgs["golang"]` | Cambiar la clave antes que la cláusula de paquete → RED |
| Unitario | Nombres de proveedor sin cambios | `register_test.go` / `container_capabilities_test.go` existentes deben seguir en verde | Regresión, no nuevo |
| Unitario (raíz) | El `go.mod` raíz versionado no tiene `replace` | Tramo 2: `modfile.Parse` + `len(f.Replace) == 0` | Escrito en el tramo 2 con el `replace` todavía presente → RED |
| Integración (con guarda `-short`) | `GoBuilder`/`GoUnitTester` con motor real | Movidos sin cambios; ahora corren bajo `./...` de la raíz en modo workspace | Regresión |
| E2E (manual) | `examples/workflow/diamond.yaml` | Ejecutar antes y después; comparar proveedores resueltos | Regresión |
| E2E (manual) | `go install github.com/pablogore/shipwright@<tag>` con `GOMODCACHE` limpio | Aceptación del tramo 2 | Aceptación |

Cobertura: `providers/go` conserva sus propios tests, así que la cobertura por
módulo no debería cambiar. El `go list ./...` del objetivo `make coverage` de la
raíz incluirá ahora `providers/go` en modo workspace: confirmar que el número no
baje del 90.

## Matriz de amenazas

`N/A` — no cambia enrutamiento, comandos de shell, subprocesos, clasificación de
archivos ejecutables ni integración de procesos. La única superficie adyacente es
la automatización de VCS, y este diseño **la evita deliberadamente**: el etiquetado
sigue siendo un paso manual y revisado en lugar de un push automatizado,
precisamente para no introducir una nueva ruta de VCS automatizada con credenciales.

## Migración y despliegue

1. **Tramo 1** (rama desde `develop`): guardianes en RED → módulo + `git mv` →
   los tres cambios de importador → `go.work` → `replace` temporal → borrar
   `internal/capabilities/` → línea 46 de `COMPATIBILITY.md`. Verde tanto con
   `go test -race ./...` como con `GOWORK=off go build ./...`. Verificar que
   `git diff --stat -- pkg/shipwright/` esté vacío. Merge a `develop`.
2. **Etiqueta**: `git tag providers/go/v0.1.0 <sha del merge>` +
   `git push origin providers/go/v0.1.0`. Confirmar que el proxy la ve
   (`GOPROXY=direct go list -m github.com/pablogore/shipwright/providers/go@v0.1.0`).
3. **Tramo 2**: test guardián de "sin `replace`" en RED → borrar el `replace` →
   `GOWORK=off go mod tidy` en la raíz → aceptación de `go install` con
   `GOMODCACHE` limpio.

Frontera de reversión: el tramo 1 es atómico (`git revert -m 1`). El tramo 2 se
revierte de forma independiente al estado con `replace`. La etiqueta publicada es
el único artefacto irreversible: se sustituye con `v0.1.1`, nunca se borra.

## Preguntas abiertas

- [ ] **`go mod download` / `go mod verify` de CI en modo workspace.** Se ejecutan
      en la etapa `setup` y son operaciones de módulo único cuyo comportamiento en
      modo workspace varía según la versión de Go. **Respuesta de diseño**:
      prefijar ambos con `GOWORK=off` para que conserven su semántica previa
      exacta; las etapas `build`/`test`/`security` permanecen en modo workspace
      para que `./...` abarque `providers/go` sin ningún paso nuevo (la afirmación
      "CI sin cambios" de la propuesta se sostiene para esas tres). Verificar
      empíricamente en el tramo 1; si los comandos sin prefijo ya funcionan,
      quitar el prefijo y dejar `ci.yml` intacto.
- [ ] **Visibilidad del repositorio.** `git remote` confirma
      `github.com/pablogore/shipwright`, así que la ruta de módulo coincide con el
      repositorio y la premisa de D4 se sostiene. No verificado: si el repositorio
      es público. Si fuera privado, `go install …@latest` nunca funcionó y toda la
      restricción de D4 es una brecha preexistente, no una regresión de este
      cambio — verificar con
      `curl -s https://proxy.golang.org/github.com/pablogore/shipwright/@v/list`
      antes del tramo 2 y reportarlo de forma explícita si viene vacío.
- [ ] **Ruido de `go work sync` / `go.work.sum`** en el hook de pre-commit
      (`go-mod-tidy`). Confirmar que el hook no entre en conflicto con el modo
      workspace; si lo hace, acotarlo con `GOWORK=off`.

---

*Nota de desviación:* excede el presupuesto de 800 palabras de la skill. Motivo: el
orquestador exigió que el diseño convirtiera cinco decisiones diferidas de la
propuesta en algo ejecutable (contenidos exactos de archivo, el orden de etiquetado
del ciclo de módulos, la implementación de los guardianes), y el orden de D4 no
puede registrarse de forma creíble sin su razonamiento. El contenido está
comprimido en tablas; sin relleno narrativo.

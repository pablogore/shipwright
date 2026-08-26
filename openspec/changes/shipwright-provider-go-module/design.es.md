# Diseño: Extraer `internal/capabilities/` a un módulo independiente `providers/go`

> Versión en español. La versión canónica y fuente de verdad ante cualquier
> conflicto es `design.md` (inglés).

## Enfoque técnico

Tres PR encadenados con una etiqueta git manual entre el segundo y el tercero,
porque la etiqueta no puede existir antes que el código, la automatización de
release debe existir antes que la etiqueta (D6), y el `require`+`go.sum` no
pueden existir antes que la etiqueta.

| Tramo | Contenido | Mecanismo de resolución |
|---|---|---|
| **1** | Crear el módulo `providers/go`, `git mv` de 13 archivos, cambiar todos los importadores, añadir `go.work`, añadir 3 tests guardianes, borrar `internal/capabilities/` | `replace ... => ./providers/go` temporal en el `go.mod` raíz (autorizado explícitamente por D4) |
| **1b** | Automatización de release: `.github/workflows/release-provider-go.yml`, `--match 'v[0-9]*'` en las tres llamadas `git describe` del `release.yml` raíz, `git.ignore_tags` en `.goreleaser.yml`, `internal/releaseguard/tags_test.go` | — (no cambia la resolución de módulos; debe entrar **antes** de que exista la primera etiqueta de proveedor — D6) |
| **etiqueta** | `git tag providers/go/v0.1.0 <sha del merge del tramo 1> && git push origin providers/go/v0.1.0` | — (push manual, no es un PR; el workflow del tramo 1b reacciona — D6) |
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

**La *creación* de la etiqueta sigue siendo manual: un `git push` humano, nunca un
job.** `.goreleaser` construye solo el binario raíz y no se reutiliza aquí. Lo que
ocurre *después* del push (validación de forma, build aislado, changelog, GitHub
Release) se automatiza en el tramo 1b — ver **D6**, que resuelve el punto que la
propuesta dejó abierto deliberadamente ("manual vs. goreleaser/CI para el
etiquetado automatizado"). Etiquetar el commit de merge
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

### Decisión: la automatización de release del proveedor es un workflow que reacciona a la etiqueta, **no** GoReleaser (D6 — nueva)

D6 no tiene contraparte en la propuesta: la propuesta difirió exactamente esto al
diseño ("manual vs. goreleaser/CI para el etiquetado automatizado"; "si CI/
goreleaser automatiza las FUTURAS etiquetas de proveedor"). **Elección** — un
`.github/workflows/release-provider-go.yml` dedicado que *reacciona* a una
etiqueta `providers/go/v*` publicada por una persona. Sin GoReleaser; el job nunca
crea ni mueve una referencia.

| Opción | Costo | Veredicto |
|---|---|---|
| Segunda configuración de GoReleaser | `monorepo:` (`tag_prefix`/`dir`) es la única forma en que GoReleaser entiende una etiqueta con prefijo de ruta, y es una función **Pro**; la versión OSS deriva la versión quitando la `v` de la etiqueta y no puede parsear `providers/go/v0.1.0`. Hay que desactivar todos los subsistemas centrales (builds, archives, checksums, taps, docker, firma) porque `providers/go` **no tiene paquete `main`**. Valor residual: solo changelog y cuerpo del release. | **Rechazada** |
| Solo `gh release create --generate-notes` | Gratuito, pero las notas son de todo el repositorio (cada commit desde el release anterior), no acotadas a `providers/go/`, y nada valida la etiqueta. | Rechazada como respuesta completa; se conserva como respaldo del cuerpo |
| Validación reactiva a la etiqueta + changelog acotado por ruta | Un archivo de workflow, sin licencia, sin binarios; valida las cuatro cosas que realmente rompen el release de una librería. | **Elegida** |

**Fundamento** — el valor de GoReleaser es la *distribución de binarios*;
`providers/go` distribuye **código fuente a través del proxy de módulos de Go**,
así que toda su maquinaria de build/empaquetado/publicación es inaplicable por
construcción. Lo que realmente puede romper este release es la etiqueta: forma
incorrecta, mayor incorrecto (la regla `/vN` de la ruta de módulo), un módulo que
no compila de forma aislada, o un proxy que nunca la descargó. Ninguna de esas es
una función de GoReleaser; las cuatro son pasos `run:`.

El workflow, en orden:

1. `on: push: tags: ['providers/go/v*']` — disjunto del `v*` de la raíz por
   construcción (los globs de Actions no cruzan `/`, y la etiqueta de proveedor no
   empieza por `v`). `ci.yml` no se dispara con etiquetas, así que este workflow es
   el único reactor.
2. **Forma**: `^providers/go/v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$`.
   Rechazar mayor ≥ 2 con un mensaje que nombre la regla del sufijo mayor en la
   ruta de módulo (un `v2` requiere ruta `.../providers/go/v2` y etiqueta
   `providers/go/v2/v2.0.0` — es una decisión, no un error de tipeo).
3. **Identidad**: el árbol etiquetado tiene `providers/go/go.mod` cuya línea
   `module` es exactamente `github.com/pablogore/shipwright/providers/go`.
4. **Aislamiento**: `cd providers/go && GOWORK=off go build ./... && GOWORK=off go test -race -short ./...`
   sobre la etiqueta — la única garantía que un `git tag` manual no puede dar, y la
   misma propiedad de aislamiento que D5 verifica a nivel de imports.
5. **Changelog**: `git log --pretty='- %s (%h)' <etiqueta providers/go previa>..<etiqueta> -- providers/go/`
   — **acotado por ruta**, para que el ruido exclusivo de la raíz nunca aparezca en
   un release de proveedor. Sin predecesora ⇒ todos los commits que tocan
   `providers/go/`.
6. **Release**: `gh release create "$TAG" --title "providers/go $VERSION" --notes-file … --latest=false`.
   `--latest=false` mantiene la insignia "Latest" del repositorio en el release de
   binarios de la raíz, del que depende el fragmento de instalación
   `releases/latest` de `.goreleaser.yml`.
7. **Compuerta del proxy**: `curl -sf https://proxy.golang.org/github.com/pablogore/shipwright/providers/go/@v/$VERSION.info`
   — convierte la confirmación antes manual del paso 3 en una verificación que
   puede fallar.

**Radio de impacto sobre la línea de release de la raíz: la razón por la que esto
no puede esperar a v0.2.0.** `.github/workflows/release.yml` deriva la versión de
la raíz con un `git describe --tags --abbrev=0` sin filtrar en **tres** lugares
(línea 162 auto-bump, 220 bump por dispatch, 245 rango del changelog). Al no estar
filtrado, en cuanto `providers/go/v0.1.0` sea alcanzable desde `main` pasa a ser
`LATEST_TAG`; el `sed 's/v//'` existente elimina entonces la `v` de
*`pro`**`v`**`iders`*, produciendo `proiders/go/v0.1.0`, y `awk -F.` emite la
etiqueta `vproiders/go/v0.2.0`. La ruta automática (merge develop→main →
`workflow_dispatch`) crearía y publicaría esa referencia sin que ninguna persona
escriba una etiqueta. **Corrección obligatoria, antes de que exista la primera
etiqueta de proveedor**: `--match 'v[0-9]*'` en las tres llamadas. El
`.goreleaser.yml` raíz recibe además `git: ignore_tags: ['providers/*']` para su
propia búsqueda de etiqueta previa.

**Guardián** (patrón del repositorio; solo test, como el de D5):
`internal/releaseguard/tags_test.go` parsea ambos YAML de workflow y verifica que
(a) ningún `git describe --tags` de `release.yml` carezca de `--match`; (b) los
globs de etiqueta de `release.yml` no coincidan con `providers/go/v0.1.0`; (c) los
globs de `release-provider-go.yml` no coincidan con `v1.2.3`; (d) el literal de la
expresión regular de forma extraído del workflow acepte `providers/go/v0.1.0` y
rechace `providers/go/v2.0.0`, `providers/go/v01.0.0` y `v0.1.0`. La disyunción de
espacios de nombres pasa a ser una aserción, no una convención.

**Dónde entra: tramo 1b, antes de la etiqueta — no a partir de v0.2.0.** Tres
razones: (1) el defecto de `git describe` lo arma la *primera* etiqueta de
proveedor, así que su corrección debe precederla; (2) un workflow de release que
nunca se ejecutó es un script, no automatización — `v0.1.0` es la prueba de humo
más barata posible (v0.x, cero consumidores, y un mal resultado cuesta `v0.1.1`,
que D4 ya acepta como la puerta de un solo sentido); (3) como tramo propio deja
intacto el diff del tramo 1 y añade un PR pequeño, limitado a `.github`, y
reversible por separado, en vez de inflar uno existente.

**Consistencia con la línea de release de la raíz, y dónde difieren
deliberadamente:**

| | módulo raíz | `providers/go` |
|---|---|---|
| Etiqueta | `vX.Y.Z` | `providers/go/vX.Y.Z` |
| Disparador | push de etiqueta **o** automático al mergear develop→main, versión *derivada* | solo push de etiqueta, versión nunca derivada |
| Herramienta | GoReleaser (`.goreleaser.yml`) | `gh release create` |
| Artefacto | binarios multiplataforma + checksums | ninguno — el proxy es el artefacto |
| "Latest" en GitHub | sí | `--latest=false` |

La diferencia es deliberada: la raíz auto-incrementa porque publica un binario
instalable con cadencia, mientras que la versión de un proveedor es una afirmación
sobre un contrato de API que el Eje 5 de COMPATIBILITY.md ya declara fuera de toda
garantía de Shipwright. Derivarla de heurísticas sobre mensajes de commit sería
peor que inútil. Lo que ambas líneas **sí** deben compartir es la disyunción de
espacios de nombres, y eso ahora es el guardián anterior en vez de una regla
tácita.

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
| `.github/workflows/release-provider-go.yml` | Crear (tramo 1b) | D6: validación reactiva a la etiqueta + GitHub Release para `providers/go/v*` |
| `.github/workflows/release.yml` | Modificar (tramo 1b) | D6: `--match 'v[0-9]*'` en las tres llamadas `git describe --tags --abbrev=0` (líneas 162, 220, 245) |
| `.goreleaser.yml` | Modificar (tramo 1b) | D6: `git: ignore_tags: ['providers/*']` para la búsqueda de etiqueta previa de la raíz |
| `internal/releaseguard/tags_test.go` | Crear (tramo 1b) | Guardián D6; paquete solo de test, como el de D5 |
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
| Unitario (raíz) | D6: disyunción de espacios de nombres de etiquetas, `git describe` filtrado, regex de forma de etiqueta | Tramo 1b: parsear ambos YAML de workflow; extraer el literal de la regex de forma y probarlo por tabla (`providers/go/v0.1.0` ✓; `providers/go/v2.0.0`, `providers/go/v01.0.0`, `v0.1.0` ✗) | Escrito antes de que exista `release-provider-go.yml` → falla cerrado por archivo ausente y luego RED por el `git describe` sin filtrar |
| Integración (con guarda `-short`) | `GoBuilder`/`GoUnitTester` con motor real | Movidos sin cambios; `./...` **no** cruza a `providers/go` en modo workspace (verificado empíricamente — ver la Pregunta abierta corregida más abajo), así que `make test`/CI ahora ejecutan `(cd providers/go && go test ./...)` de forma explícita para seguir cubriéndolo | Regresión |
| E2E (manual) | `examples/workflow/diamond.yaml` | Ejecutar antes y después; comparar proveedores resueltos | Regresión |
| E2E (manual) | `go install github.com/pablogore/shipwright@<tag>` con `GOMODCACHE` limpio | Aceptación del tramo 2 | Aceptación |

Cobertura: `providers/go` conserva sus propios tests, ejecutados mediante la
invocación explícita de `go test` de arriba, no fusionados con el
`coverage.out` de la raíz. Esto es una corrección, no el plan original: se
verificó que `go list ./...` devuelve exactamente los 22 paquetes previos a
la extracción en modo workspace, nunca `providers/go` (ver la Pregunta
abierta corregida más abajo), así que las listas de paquetes basadas en
`go list ./...` del objetivo `make coverage` de la raíz nunca incluyeron
`providers/go` y siguen sin incluirlo. Eso es ahora una decisión de alcance
deliberada (documentada en la sección de cobertura del `Makefile`), no un
descuido a corregir: `providers/go` es un módulo publicado de forma
independiente (D2/D4) cuya cobertura no forma parte de los umbrales del 90%/70%
de la raíz, calibrados sobre el conjunto de paquetes previo a la extracción.

## Matriz de amenazas

`N/A` para enrutamiento, comandos de shell, subprocesos, clasificación de archivos
ejecutables e integración de procesos: ninguno de esos límites cambia.

**Automatización de VCS/PR: `Aplicable` desde D6** (antes de esta enmienda era
`N/A`). Acotada así:

| Aspecto | Límite | Comportamiento esperado |
|---|---|---|
| Creación de referencias | El workflow **reacciona** a una referencia; nunca crea, mueve ni borra una. La creación de la etiqueta sigue siendo un `git push` humano (D4). | Una etiqueta siempre la escribe una persona |
| Alcance del token | Solo `permissions: contents: write` a nivel de job — sin `packages`, sin `id-token` (a diferencia de `release.yml`), sin `SHIPWRIGHT_TOKEN`, sin `~/.netrc` | Su única escritura es `gh release create` |
| Código ejecutado del árbol etiquetado | `go build` / `go test` del módulo que se publica, nada más | Sin hooks de release, sin scripts arbitrarios |
| Etiqueta mal formada o con mayor incorrecto | Falla en la validación de forma, antes de cualquier escritura de red | Sin Release, sin descarga del proxy, fallo explícito |
| Colisión de espacio de nombres con el `v*` raíz | Los globs son disjuntos por construcción y se verifican | La línea de release de la raíz no puede ser disparada ni corrompida en su versión por una etiqueta de proveedor |

Tests RED de las filas aplicables: las aserciones (a)–(d) de
`internal/releaseguard/tags_test.go` en D6.

## Migración y despliegue

1. **Tramo 1** (rama desde `develop`): guardianes en RED → módulo + `git mv` →
   los tres cambios de importador → `go.work` → `replace` temporal → borrar
   `internal/capabilities/` → línea 46 de `COMPATIBILITY.md`. Verde con
   `go test -race ./...` **y** `(cd providers/go && go test -race ./...)`
   (las dos invocaciones que `./...` no fusiona — ver la Pregunta abierta
   corregida más abajo), además de `GOWORK=off go build ./...`. Verificar que
   `git diff --stat -- pkg/shipwright/` esté vacío. Merge a `develop`.
2. **Tramo 1b** (D6, sin código Go de producción):
   `internal/releaseguard/tags_test.go` en RED → `release-provider-go.yml` →
   `--match 'v[0-9]*'` en las tres llamadas `git describe` de `release.yml` →
   `git.ignore_tags` en `.goreleaser.yml`. Merge a `develop`. **Debe preceder al
   paso 3**: el defecto del `git describe` sin filtrar lo arma la primera etiqueta
   de proveedor, no este PR.
3. **Etiqueta**: `git tag providers/go/v0.1.0 <sha del merge>` +
   `git push origin providers/go/v0.1.0` (humano). El workflow del tramo 1b valida
   entonces forma/identidad/aislamiento, publica el Release con `--latest=false` y
   condiciona el resultado a la visibilidad en el proxy
   (`curl -sf https://proxy.golang.org/github.com/pablogore/shipwright/providers/go/@v/v0.1.0.info`),
   reemplazando la confirmación manual previa con `go list -m`.
4. **Tramo 2**: test guardián de "sin `replace`" en RED → borrar el `replace` →
   `GOWORK=off go mod tidy` en la raíz → aceptación de `go install` con
   `GOMODCACHE` limpio.

Frontera de reversión: el tramo 1 es atómico (`git revert -m 1`). El tramo 1b se
revierte de forma independiente y solo toca `.github/`, `.goreleaser.yml` y un
archivo de test — pero revertirlo **no** despublica un Release (`gh release delete`
es un acto aparte) ni elimina la etiqueta. El tramo 2 se revierte de forma
independiente al estado con `replace`. La etiqueta publicada es el único artefacto
irreversible: se sustituye con `v0.1.1`, nunca se borra.

## Preguntas abiertas

- [x] **`go mod download` / `go mod verify` de CI en modo workspace, y si `./...`
      abarca `providers/go` en las etapas `build`/`test`/`security`.**
      **Verificado empíricamente en el tramo 1 — la respuesta de diseño
      original era incorrecta en la segunda mitad.** `go list ./...` desde la
      raíz del repo devuelve exactamente los 22 paquetes previos a la
      extracción, de forma idéntica en modo `GOWORK=off` y en modo workspace;
      el comodín `./...` de Go **nunca** cruza el límite de un módulo anidado
      declarado vía `use` en `go.work` — solo lo hace el patrón `all` o una
      ruta de importación completamente calificada. `all` tampoco es un
      sustituto seguro, confirmado comando por comando: `go vet all` termina
      con código 1 (también aplica vet a archivos `_test.go` de dependencias
      de terceros no relacionados, con problemas de vet preexistentes, p. ej.
      `google/go-cmp`, `prometheus/client_golang`); `go fmt all` rechaza
      explícitamente los paquetes de `providers/go` ("not formatting packages
      in dependency modules"); `go list all` / `go test all` arrastran todo el
      grafo transitivo de dependencias (~400 paquetes), no solo los dos
      módulos del workspace; `govulncheck` no abarca `go.work` en absoluto (los
      escaneos de la raíz y de `providers/go` producen hallazgos disjuntos).
      Así que la afirmación "CI sin cambios" de la propuesta **no** se
      sostiene: antes de esta corrección, `providers/go` quedaba
      silenciosamente excluido de toda invocación `build`/`test`/`vet`/`fmt`/
      `govulncheck` de alcance-repositorio, tanto en `Makefile` como en
      `ci.yml`. **Corrección aplicada**: se añadió una invocación explícita
      `(cd providers/go && go <cmd> ./...)` (o `govulncheck` con alcance de
      módulo) junto a cada invocación de la raíz ya existente — no ampliando
      `./...` a `all`, que las pruebas de arriba muestran que es inseguro. La
      cobertura (`go list ./... | grep -v ...`) es la única excepción que se
      deja sin cambios por diseño, no por descuido — ver la nota de Cobertura
      más arriba. `go mod download`/`go mod verify` en la etapa `setup` siguen
      prefijados con `GOWORK=off` como se diseñó originalmente; esa mitad de
      la respuesta se mantiene.
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
- [ ] **Dos hechos de D6 a confirmar en la fase de aplicación; ninguno cambia la
      dirección de D6.** (a) Que el bloque `monorepo:` (`tag_prefix`/`dir`) de
      GoReleaser sea exclusivo de Pro — D6 rechaza GoReleaser solo con el argumento
      de que no hay paquete `main`, así que esto solo afecta a cómo se redacta el
      rechazo. (b) La clave exacta de `.goreleaser.yml` para excluir etiquetas de la
      búsqueda de etiqueta previa (`git.ignore_tags`) frente a la versión de
      GoReleaser que resuelve el workflow de release (`version: latest`); la
      corrección con peso real es `--match 'v[0-9]*'` en `release.yml`, que no
      depende de ella. Ambos puntos se razonaron a partir de los archivos
      versionados, no de documentación en red.

---

*Nota de desviación:* excede el presupuesto de 800 palabras de la skill. Motivo: el
orquestador exigió que el diseño convirtiera cinco decisiones diferidas de la
propuesta en algo ejecutable (contenidos exactos de archivo, el orden de etiquetado
del ciclo de módulos, la implementación de los guardianes), y el orden de D4 no
puede registrarse de forma creíble sin su razonamiento. Una enmienda posterior
añadió D6, que la propuesta difirió explícitamente al diseño y que terminó
exponiendo un defecto activo de derivación de versión en el workflow de release de
la raíz. El contenido está comprimido en tablas; sin relleno narrativo.

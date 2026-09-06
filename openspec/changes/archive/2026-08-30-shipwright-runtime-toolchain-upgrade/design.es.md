# Diseño: Actualización de runtime/toolchain gestionada por el provider (Go primero)

> Versión en español. La versión canónica es `design.md`; ante cualquier
> conflicto, prevalece el inglés.

## Enfoque técnico

Dos capability kinds de un solo método, según el Approach de la propuesta. Todo
el parseo y la mutación es Go puro sobre *bytes* de archivo
(`golang.org/x/mod/modfile`), ejecutado **antes** de cualquier llamada a
Dagger; el workspace mutado se materializa como un `*dagger.Directory` nuevo e
inmutable. El core gana dos casos de dispatch y dos pares de registry — sin
lógica específica del ecosistema, sin primitiva de bloqueo (D1) y sin ninguna
ruta remota (D2).

Strings de capability canónicos, fijados aquí para la convergencia con
`sdd-spec`: **`runtime-inspect`** y **`runtime-upgrade`** (con guion, igual que
los nombres de provider existentes `go-test`/`rust-integration-test`).

## Decisiones de arquitectura

### D-1: Los tipos de reporte nunca aparecen en la firma de una interfaz

El test de reflexión `allowedSignatureType` en
`pkg/shipwright/capabilities_test.go` impone D-A: los métodos de capability
solo pueden usar `context.Context`, `error`, `string` y los cuatro tipos core
de Dagger. Un struct `DriftReport` como parámetro o retorno rompería ese test
existente.

| Opción | Veredicto |
|---|---|
| Retornar `*DriftReport` concreto | **Rechazada** — viola D-A y rompe un test vivo |
| Retornar `*dagger.File` (reporte JSON) | Rechazada — el codegen de Layer 2 no puede retornar un tipo core lazy *junto con* `error`; además no es referenciable desde un campo `with` |
| Retornar `string` (payload JSON) | **Elegida** para `Inspect` |
| Retornar el `*dagger.Directory` mutado, con el reporte escrito *dentro* | **Elegida** para `Upgrade` |

`(string, error)` es la forma que la documentación del paquete `.dagger`
registra como compilable bajo el codegen de Dagger v0.21.8, y mapea al kind
`outputText` del engine, así que `${{ steps.<id>.output }}` puede alimentar un
campo `with` posterior. `Upgrade` retorna un Directory (`outputDirectory` en el
engine), que el cambio SCM diferido de D2 consume directamente. Los structs de
reporte siguen siendo tipos Go concretos y exportados en `pkg/shipwright`: son
el contrato JSON, simplemente nunca son un parámetro de método.

### D-2: `Upgrade` retorna el Directory; el reporte viaja dentro

Un método de Dagger Interface retorna a lo sumo `(T, error)`, así que no se
pueden retornar directorio y reporte a la vez. `Upgrade` escribe
`.shipwright/runtime-upgrade-report.json` dentro del Directory retornado.
Alternativa rechazada (un segundo método `LastReport()`): rompe el invariante
de un método por capability que el Approach de la propuesta existe para
preservar.

### D-3: Layer 2 declara las interfaces pero NO las conecta a `Plan`

`Plan` es una cadena lineal fija de cinco slots
(`build→test→artifact→deploy→run`), y el engine la evita por completo (D-G).
Una actualización de toolchain no es una etapa de ese pipeline; dos slots
extra obligarían a `Execute` a inventar un orden sin respuesta defendible.
Ahorra ~40 líneas y un desastre semántico. No existe ningún test de paridad
entre el conjunto de interfaces de Layer 1 y Layer 2 (verificado), así que es
seguro.

El `Upgrade` de Layer 2 **descarta `error`**, igual que `Builder.Build` y
`Tester.Test`: la documentación del paquete registra que el codegen de v0.21.8
no compila un tipo core lazy junto a `error`. El `(string, error)` de `Inspect`
se conserva.

### D-4: Las fuentes de versión se dividen en tres niveles según qué tan declarativas son

Drift medido en vivo en este repositorio — **seis versiones distintas**,
bastante peor de lo que registró la exploración (que solo vio el caso de
`gobuilder.go` y leyó CI como uniformemente 1.26.7, sin detectar `.go-version`
ni los tres archivos `release*.yml`):

| Fuente | Versión | Nivel |
|---|---|---|
| `go.work`, `go.mod`, `providers/go/go.mod`, `providers/rust/go.mod` | 1.26.7 | 1 |
| `.go-version` | **1.26.1** | 1 |
| `.dagger/go.mod` | **1.26.5** | 3 |
| `.github/workflows/ci.yml` `GO_VERSION` | 1.26.7 | 2 |
| `.github/workflows/release.yml` `GO_VERSION` | **1.26.1** | 2 |
| `.github/workflows/release-provider-{go,rust}.yml` `GO_VERSION` | **1.26.1** | 2 |
| `providers/go/gobuilder.go` `defaultGoVersion` | **1.25.5** | 3 |

**Nivel 1 — en alcance, inspeccionado y mutado.** Directivas `go`/`toolchain`
de `go.mod`, `go.work` y **`.go-version`**. `.go-version` entra por el mismo
criterio que `go.mod`: nombre de archivo fijo en la raíz del workspace, con un
formato de una sola línea, agnóstico de herramienta (la convención de
goenv/asdf/mise). Sin adivinar, sin suposiciones por repositorio, ~30 líneas de
soporte, y es un sitio con drift real *hoy*. Se incorpora como fuente de
primera clase al chequeo de unanimidad A1–A3.

**Nivel 2 — fuera de alcance: YAML de workflows de CI.** `GO_VERSION` en
`.github/workflows/*.yml` es una *convención* de nombres, no un contrato: el
nombre de la clave, el conjunto de archivos y la action consumidora son todos
específicos del repositorio. Localizarlo genéricamente es adivinar, y adivinar
mal significa mutar una variable de workflow no relacionada. Un cambio
posterior puede agregar una lista `ciVersionPins: [{file, key}]` configurada
explícitamente; v1 no adivina.

**Nivel 3 — fuera de alcance, estructuralmente inalcanzable.**
- El `defaultGoVersion` de `providers/go/gobuilder.go` es el default de imagen
  de build **de Shipwright**, no una propiedad del workspace inspeccionado.
  Reportarlo como drift del workspace es un error de categoría, y contra un
  repositorio de terceros es directamente engañoso. Tampoco hace falta parsear
  AST: el inspector vive en el **paquete `golang`**, así que la constante es
  una referencia de tiempo de compilación.
- `.dagger/go.mod` está deliberadamente ausente de la lista `use` de `go.work`,
  y el diseño archivado `shipwright-provider-go-module` incluye un guard test
  que *falla* si alguna vez se agrega `use ./.dagger` (aislamiento de módulo
  D-B). Por lo tanto es invisible al recorrido del workspace por diseño. Su
  drift a 1.26.5 es real, pero debe corregirse con una decisión separada y
  deliberada, no en silencio por esta capability.

**Sustitución, no aplazamiento** para el primer punto del nivel 3: este cambio
incluye `providers/go/toolchainpin_test.go` con el molde de
`internal/daggerpin` — `TestDefaultGoVersionMatchesGoMod` parsea la directiva
`go` de `providers/go/go.mod` y afirma que es igual a `defaultGoVersion`. ~30
líneas, RED hoy (1.26.7 vs 1.25.5), corre localmente en milisegundos, e impone
el invariante de forma permanente en lugar de reportarlo en un JSON que nadie
lee.

> **Riesgo de convergencia con `sdd-spec`**: NO escribir un requisito ADDED de
> que `runtime-inspect` reporte un pin de imagen de build o de YAML de CI. El
> criterio de éxito #1 de la propuesta se satisface con el guard test. **Sí**
> escribir `.go-version` como fuente inspeccionada y mutada.

### D-4b: Semántica de `verify` sin un tercer capability kind

La inspección de solo lectura tiene dos usos distintos: *reportar* el drift y
*fallar el build ante* el drift. Son resultados distintos para el engine, pero
no capabilities distintas — un tercer kind `runtime-verify` costaría otra
entrada de allowlist, otro par de registry, otro caso de dispatch, otra
interfaz de Layer 2 y otra rama en `main.go` para una diferencia de
comportamiento de un bit.

**Elección**: un solo kind `runtime-inspect` con un campo `with` booleano
`failOnDrift` (default `false`). Con `failOnDrift: true`, `Inspect` retorna
error cuando las fuentes de nivel 1 no coinciden, lo que el engine ya convierte
en un step fallido y, bajo `failFast`, en un workflow fallido. Con
`failOnDrift: false` siempre tiene éxito y emite el string del reporte para que
lo consuma un step posterior.

```yaml
- id: verify-toolchain          # semántica de gate
  capability: runtime-inspect
  with: { failOnDrift: true }
```

Rechazado: un kind `runtime-verify` separado (~90 líneas escritas en seis
archivos por un booleano); hacer que el fallo sea el default incondicional
(elimina el modo solo-reporte que necesita el handoff de D2).

### D-5: `modfile` confirmado; las reglas de ambigüedad

`golang.org/x/mod/modfile` es la dependencia correcta — el mismo paquete que usa
`cmd/go`, y ya es precedente en el repositorio (`internal/daggerpin/pin.go`, y
`modfile.ParseWork` en el diseño archivado). Aporta `Parse`/`ParseWork`,
`File.Go`/`File.Toolchain`/`WorkFile.Use`,
`AddGoStmt`/`AddToolchainStmt`/`DropToolchainStmt`, y `Format()`, que preserva
comentarios y layout. `AddGoStmt` valida contra `modfile.GoVersionRE`, lo que da
validación fail-closed gratuita del target. Ninguna alternativa resulta viable.

**Nota**: `golang.org/x/mod` *no* es hoy dependencia de `providers/go/go.mod`
(solo del módulo raíz). Pasa a ser un requirement directo nuevo ahí, fijado a la
versión de la raíz.

"Ambiguo" es exactamente esto, todo detectado **antes** de cualquier mutación:

| Código | Condición | Comportamiento |
|---|---|---|
| A1 | Dos módulos del workspace declaran directivas `go` distintas | abortar |
| A2 | Dos módulos declaran directivas `toolchain` distintas | abortar |
| A3 | El `go` de `go.work`, o `.go-version`, difiere del `go` unánime de los módulos | abortar |
| A4 | `targetVersion` < versión actual (semver, `golang.org/x/mod/semver`) | abortar salvo `allowDowngrade: true` |
| A5 | Target o directiva existente malformada | abortar |
| A6 | No se encuentra ni `go.work` ni `go.mod` en la raíz del workspace | abortar |

El aborto retorna un `*AmbiguousToolchainError` tipado que enumera cada sitio en
conflicto, y `(nil, err)`. **La mutación parcial es estructuralmente imposible
de provocar**: el análisis termina antes del primer `WithNewFile`, y un
`dagger.Directory` es un valor inmutable — una ejecución fallida sencillamente
nunca retorna uno.

### D-6: La validación es `go build ./...` por módulo — `go vet` rechazado explícitamente

La afirmación de la actualización es "este workspace sigue compilando bajo el
nuevo toolchain", y `go build ./...` es exactamente esa afirmación. `go vet` se
consideró y se rechazó: sus chequeos evolucionan en cada release, así que amplía
la superficie de fallo con errores ajenos al bump, duplica una etapa que el CI
del repositorio ya tiene, y duplica el tiempo de contenedor. El reporte registra
`validation: "build"` para que ningún consumidor se confunda sobre qué se probó.
Por módulo, porque está comprobado que `./...` no cruza un límite de módulo de
`go.work` (verificado en la Open Question del diseño archivado).

### D-7: Secuenciamiento de `go mod tidy`

Cadena ordenada, un solo contenedor:

1. Go puro en el host: parsear cada `go.mod`/`go.work`/`.go-version`, detectar
   A1–A6, mutar bytes.
2. Materializar: un `WithNewFile` por archivo cambiado → Directory nuevo.
3. `From("golang:" + targetVersion)`, montar el Directory mutado.
4. Por módulo: `cd <mod> && go mod tidy` (omitido con `tidy: false`).
5. Por módulo: `go build ./...` (debe ir después de 4 — un `go.sum` sin tidy lo
   rompe).
6. Exportar el workdir del contenedor como Directory retornado, llevando los
   cambios de `go.sum` de tidy.

El reporte registra, por módulo, `goSumChanged: bool` más las rutas de módulo
`require` agregadas/eliminadas — **no** el diff crudo de `go.sum`, que puede
llegar a miles de líneas irrevisables. Si la validación falla ⇒ abortar, sin
retornar Directory.

### D-8: `main.go` es un sexto sitio de dispatch que la propuesta omitió

`resolveCapabilityRef` (`main.go`, ejercitado por las "five capability branches"
de `main_test.go:337`) es un segundo switch de cinco ramas sobre capability que
alimenta `--list-steps`. Sin dos casos nuevos, un step `runtime-inspect` se
ejecuta bien pero falla en `--list-steps` con un error de capability
desconocida. Agregar a Affected Areas: +2 casos, ~10 líneas de producción.

### D-9: `providers/go/daggerkit` necesita una superficie de lectura/escritura

Hoy `DaggerDirectory` solo expone `GetRealDirectory()`, y `DaggerFile` no tiene
`Contents()` — los cinco providers existentes nunca leen un archivo. Leer
`go.mod` y escribir el árbol mutado requiere `Directory.File(string)`,
`File.Contents(ctx)`, `Directory.Entries(ctx)` y
`Directory.WithNewFile(path, contents)`, más adapters y mocks. Hacerlo vía
`WithExec` se rechazó: dejaría el núcleo de parseo/mutación sin poder testearse
sin un engine real, violando la regla de selección de dobles del skill
testing-tdd. Es trabajo no presupuestado que la propuesta no visibilizó (~95
líneas).

## Flujo de datos

```
step del manifest (capability: runtime-upgrade, with: {targetVersion})
   │
   ▼
engine/execute.go  dispatch ──▶ providers.ResolveRuntimeUpgrader
   │                                      │
   │                                      ▼
   │                          providers/go.GoRuntimeUpgrader
   ▼                                      │
 *dagger.Directory (source) ──────────────┤
                                          │ 1. Contents() go.work + .go-version
                                          │    + go.mod de cada módulo use'ado
                                          │ 2. parseWorkspace → detectConflicts (A1..A6)
                                          │ 3. mutar bytes (modfile.Format)
                                          │ 4. WithNewFile ×N  → Directory mutado
                                          │ 5. contenedor: go mod tidy, go build ./...
                                          ▼
                       *dagger.Directory + .shipwright/runtime-upgrade-report.json
                                          │
                            (lo consume el cambio posterior de D2)
```

Nada cruza el límite del proceso en el host: sin `os.WriteFile`, sin
`exec.Command`, sin red, sin git.

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `pkg/shipwright/capabilities.go` | Modificar | +`RuntimeInspector`, +`RuntimeUpgrader` |
| `pkg/shipwright/runtime.go` | Crear | Structs `DriftReport`, `UpgradeReport`, `ModuleDrift` |
| `pkg/shipwright/testdata/api.golden` | Regenerar | **`-update` es un paso de tarea OBLIGATORIO** |
| `.dagger/capabilities.go` | Modificar | Proyección Layer 2; `Upgrade` descarta `error` (D-3) |
| `internal/workflow/manifest/validate.go` | Modificar | Allowlist 5→7 + mensaje de error |
| `internal/workflow/providers/registry.go` | Modificar | 2 tablas + pares `Register`/`Resolve` |
| `internal/workflow/providers/register.go` | Modificar | Registrar `go-runtime` bajo ambos kinds |
| `internal/workflow/engine/execute.go` | Modificar | 2 casos de dispatch + constantes de campos `with` |
| `main.go` | Modificar | `resolveCapabilityRef` +2 casos (D-8) |
| `providers/go/toolchain.go` | Crear | Núcleo puro de parseo/conflicto/mutación — sin Dagger |
| `providers/go/runtimeinspector.go` | Crear | `GoRuntimeInspector.Inspect` |
| `providers/go/runtimeupgrader.go` | Crear | `GoRuntimeUpgrader.Upgrade` |
| `providers/go/daggerkit/{interfaces,adapter,mocks}.go` | Modificar | Superficie lectura/escritura de D-9 |
| `providers/go/go.mod`, `go.sum` | Modificar | +`golang.org/x/mod` |
| `providers/go/toolchainpin_test.go` | Crear | Guard de D-4, RED hoy |
| `providers/go/testdata/runtime/**` | Crear | 9 fixtures de workspace |

## Interfaces / Contratos

```go
// pkg/shipwright/capabilities.go — Layer 1
type RuntimeInspector interface {
    Inspect(ctx context.Context, source *dagger.Directory) (string, error)
}

type RuntimeUpgrader interface {
    Upgrade(ctx context.Context, source *dagger.Directory, targetVersion string) (*dagger.Directory, error)
}
```

`targetVersion` es un parámetro del método y no un campo del struct
precisamente para que el sistema de tipos prohíba un default de valor cero: una
actualización siempre lleva un target explícito. Todo lo demás
(`workspaceRoot`, `tidy`, `allowDowngrade`, `failOnDrift`) es configuración del
struct del provider, ligada desde `with` al registrar, exactamente como ya lo
es `RustBuilder.ManifestPath`.

```go
// .dagger/capabilities.go — Layer 2
type RuntimeInspector interface {
    dagger.DaggerObject
    Inspect(ctx context.Context, source *dagger.Directory) (string, error)
}

type RuntimeUpgrader interface {
    dagger.DaggerObject
    Upgrade(ctx context.Context, source *dagger.Directory, targetVersion string) *dagger.Directory
}
```

Esquemas `with` del manifest (deben coincidir exactamente con `sdd-spec`):

| Capability | Campo | Kind | Requerido |
|---|---|---|---|
| `runtime-inspect` | `workspaceRoot` | String | no (default `.`) |
| `runtime-inspect` | `expectedVersion` | String | no |
| `runtime-inspect` | `failOnDrift` | Bool | no (default false, D-4b) |
| `runtime-upgrade` | `targetVersion` | String | **sí** |
| `runtime-upgrade` | `workspaceRoot` | String | no (default `.`) |
| `runtime-upgrade` | `tidy` | Bool | no (default true) |
| `runtime-upgrade` | `allowDowngrade` | Bool | no (default false) |

Confirmación para el engine, según D1 de la propuesta:
`dispatchRuntimeInspect` y `dispatchRuntimeUpgrade` son funciones lineales
`Resolve* → llamada → envolver resultado`, estructuralmente idénticas a
`dispatchBuild`. **No se agrega en ningún lado código de bloqueo, encolado,
espera, aprobación ni scheduling** — la afirmación del doc del paquete
`execute.go` sigue siendo literalmente cierta, y `workflow-execution` no
necesita delta.

## Estrategia de testing

| Capa | Qué testear | Enfoque |
|---|---|---|
| Unit (puro) | `parseWorkspace`, `detectConflicts` A1–A6, `mutateGoMod`, `mutateGoWork`, `.go-version` | Table-driven sobre fixtures `testdata/runtime/*`; sin Dagger, sin engine. **Aquí vive la masa de TDD.** |
| Unit (con mocks) | `Inspect`/`Upgrade` happy path, cliente nil, source nil | Mocks extendidos de `providers/go/daggerkit` (regla 1 del orden de selección de dobles) |
| Unit (guard) | `defaultGoVersion` == `go` de `go.mod` | D-4; molde `daggerpin`; RED hoy |
| Unit (contrato) | Mapa de interfaces de `capabilities_test.go` | **Hay que agregar ambas interfaces nuevas**, si no D-A queda sin imponer para ellas |
| Unit (golden) | `pkg/shipwright/testdata/api.golden` | **Corrida `-update` OBLIGATORIA + diff revisado** — el CI falla apenas compilen las interfaces |
| Unit (core) | allowlist, pares de registry, dispatch del engine, casos de `main.go` | Extender `validate_test.go`, `registry_test.go`, `fakes_test.go`, `dispatch_test.go`, `main_test.go` |
| Integration | Un `Upgrade` con engine real sobre un workspace temporal | Build tag `integration` |

Fixtures: `single-module`, `workspace-3-modules`, `divergent-go` (A1),
`divergent-toolchain` (A2), `work-go-mismatch` (A3),
`goversion-file-mismatch` (A3, espejo del drift vivo de este repo:
`.go-version` 1.26.1 vs `go.mod` 1.26.7), `downgrade` (A4), `malformed` (A5),
`path-escape` (`use ../evil`).

Un fixture es una copia byte a byte de las fuentes de nivel 1 de este propio
repositorio, para que `detectConflicts` quede probado en RED contra el drift
real antes de que exista una sola línea de código de mutación.

## Matriz de amenazas

| Fila | Aplicabilidad | Comportamiento seguro | Test RED |
|---|---|---|---|
| Path traversal vía `use` de `go.work` | **Aplicable** | Rechazar rutas absolutas, y cualquier ruta que escape de la raíz del workspace tras `filepath.Clean` | El fixture `use ../../etc` aborta |
| Escritura arbitraria fuera del directorio retornado | **Aplicable** | Toda escritura vía `Directory.WithNewFile` sobre un valor inmutable; cero escrituras en el host | Guard estático de imports: sin `os.WriteFile`/`os.Create`/`os.Remove` en los archivos de runtime |
| Construcción de comandos desde valores de configuración | **Aplicable** | `targetVersion` validado contra `modfile.GoVersionRE` **antes** de llegar a `"golang:"+v` o a argv; solo `WithExec` con arreglo argv, nunca `sh -c` | `"1.26.7; rm -rf /"` y `"--flag"` rechazados en el parseo |
| Manejo de credenciales | N/A | Ninguna capability acepta `*dagger.Secret`; sin red/registry/git | |
| Subproceso en el host | N/A | Todo comando corre en un contenedor Dagger; sin `exec.Command` | |
| Automatización de VCS/PR | N/A | D2 — no existe ninguna ruta de código SCM | |
| Carga de plugins / dinámica | N/A | Solo registro compilado (D-I); `security_test.go` ya lo demuestra | |

## Preguntas de alcance surgidas durante el diseño

Planteadas a mitad del diseño; resueltas aquí contra la propuesta aceptada, en
lugar de absorberlas en silencio.

| Planteado | Resolución |
|---|---|
| Un tercer kind `runtime.verify` junto a `inspect`/`upgrade` | **Absorbido, no agregado** — el booleano `failOnDrift` de D-4b entrega la semántica de gate por ~6 líneas en lugar de ~90 en seis archivos. |
| Nombres con punto (`runtime.upgrade`) en lugar de guion | **Diferido como rename mecánico.** `sdd-spec` corre en paralelo contra una propuesta que escribe literalmente `runtime-inspect`/`runtime-upgrade`; divergir ahora garantiza un desajuste. Nada en el validador ni en la gramática de interpolación prohíbe un punto — es una edición de una línea en el allowlist cuando se decida. |
| Un campo `manual: true` en el step | **Fuera de alcance — D1 de la propuesta, aceptado.** No existe campo `manual` en `Step`, y el doc de `execute.go` afirma cero lógica de bloqueo/encolado. "Manual, no automático" se cumple no enviando scheduler ni trigger por webhook, no con un gate en el engine. |
| Creación de branch/PR tras la actualización | **Fuera de alcance — D2 de la propuesta, aceptado.** El punto de handoff de este diseño es exactamente el `*dagger.Directory` retornado por `Upgrade` más su reporte embebido; el cambio SCM posterior consume ambos. No existe ruta de escritura remota en v1, y esa es la propiedad de seguridad estructural que reemplaza al gate ausente. |
| Fan-out a providers rust/java/python | **Ya soportado, cero trabajo extra.** Ambos capability kinds son agnósticos del lenguaje; `providers/rust` registra su propio `RuntimeUpgrader` bajo el mismo kind con otro nombre de provider, tal como `rust` y `go` ya conviven bajo `build`. Implementarlo queda fuera de alcance (propuesta), pero nada en este diseño lo bloquea. |
| Un pipeline de cuatro fases `inspect→plan→apply→verify` en el provider | **Se mantiene interno al provider, según el Approach de la propuesta.** Esas cuatro fases son la secuencia *interna* de `Upgrade` (pasos 1–6 de D-7); exponerlas como steps del manifest sería el Approach 2 que la exploración rechazó, y metería branching por fase en el core. |

## Reconciliación con `sdd-spec` (corrió en paralelo)

Los strings de capability convergieron exactamente: `runtime-inspect` /
`runtime-upgrade`. Cuatro puntos necesitan decisión explícita antes del archive.

| # | `specs/runtime-toolchain/spec.md` | Este diseño | Resolución |
|---|---|---|---|
| 1 | Asumió nombres de interfaz `RuntimeInspect`/`RuntimeUpgrade` (state.yaml marca a design como dueño) | `RuntimeInspector`/`RuntimeUpgrader` | **Gana el diseño.** El sustantivo de agente coincide con las cinco interfaces existentes (`Builder`, `Tester`, `Artifactor`, `Deployer`, `Runner`). |
| 2 | L83: validación post-mutación "e.g. `go build`/`go vet` falla" | D-6: solo `go build ./...`; `go vet` rechazado explícitamente | El `e.g.` es ilustrativo, no normativo, así que no se viola ningún MUST — pero conviene recortar la mención a `go build` para que no se lea como aval de vet. |
| 3 | L71: al abortar "el directorio retornado es idéntico byte a byte al de entrada" | `Upgrade` retorna `(nil, err)` | Compatible (no hubo mutación, de forma vacía). `nil` es el contrato más seguro: quien ignore el error recibe un panic por nil en vez de publicar en silencio un árbol intacto que cree actualizado. Se recomienda reformular el escenario como "ningún archivo del workspace fue mutado". |
| 4 | L75–78 contempla "un archivo de pin de CI" | El nivel 2 de D-4 deja el YAML de workflows de CI fuera de alcance | Satisfacible de forma vacía (ausente ⇒ registrado como ausente, nada fabricado). Ningún requisito ADDED exige reportar pines de imagen de build ni de YAML de CI, así que la sustitución de D-4 sobrevive. `.go-version` es un superconjunto que la spec no prohíbe. |

También queda abierto desde state.yaml: si el requisito "Composable, Orthogonal
Capabilities" de `public-module-api` (que también enumera los cinco nombres de
capability) necesita su propio delta MODIFIED. **Sí lo necesita** — enumera el
mismo conjunto cerrado que este cambio abre a siete.

## Migración / Rollout

Sin migración. Puramente aditivo: ninguna capability, manifest o ruta del engine
existente cambia de comportamiento. Rollback = revert; quitar las dos entradas
del allowlist restaura el conjunto cerrado previo de cinco valores. Una
salvedad: la regeneración de `api.golden` debe revertirse junto con el resto.

## Pronóstico de presupuesto de revisión

Líneas escritas (adiciones + eliminaciones; `api.golden` generado excluido según
el Review Workload Guard):

| Área | Prod | Test |
|---|---|---|
| `pkg/shipwright` (2 interfaces + structs de reporte) | 85 | 20 |
| Proyección `.dagger` | 25 | 25 |
| allowlist del manifest | 4 | 20 |
| registry + register de providers | 80 | 70 |
| dispatch del engine | 50 | 90 |
| `main.go` (D-8) | 10 | 15 |
| `providers/go/daggerkit` (D-9) | 95 | — |
| `providers/go/toolchain.go` (parseo/conflicto/mutación, incl. `.go-version`) | 220 | 350 |
| `providers/go/runtimeinspector.go` (incl. `failOnDrift`) | 90 | 100 |
| `providers/go/runtimeupgrader.go` | 130 | 130 |
| Guard test de D-4 | — | 30 |
| go.mod/go.sum | 8 | — |
| **Subtotal** | **~797** | **~850** |

**Total ≈ 1.650 líneas escritas — 4,1× el presupuesto de 400 líneas.**

- `Decision needed before apply: Yes`
- `Chained PRs recommended: Yes`
- `400-line budget risk: High`

`delivery_strategy: single-pr` **no es alcanzable** para este diseño. El
orquestador debe re-resolver a una cadena o conceder un `size:exception`
explícito. Cadena recomendada de 4 slices, cada uno entregable de forma
independiente con su propia verificación y rollback:

| Slice | Contenido | Escritas |
|---|---|---|
| **1** | `runtime-inspect` de punta a punta: Layer 1 + Layer 2 + allowlist + par de registry + engine + caso en `main.go` + daggerkit de solo lectura + mitad de parseo/conflicto de `toolchain.go` (incl. `.go-version`) + `failOnDrift` + guard de D-4 + tests. **Útil por sí solo**: detecta el drift vivo de seis versiones de este repo con cero riesgo de mutación. | ~520 |
| **2** | Cableado de `runtime-upgrade` + mutación de directivas de `go.mod` y `.go-version` en módulo único + daggerkit de escritura. | ~430 |
| **3** | Recorrido multi-módulo de `go.work` + detección de conflictos entre módulos (A1–A3) + guard de path-escape. | ~370 |
| **4** | Normalización con `go mod tidy`, validación post-mutación `go build ./...`, campos de delta de go.sum en el reporte, test de integración. | ~290 |

El slice 1 tiene la mejor relación valor/riesgo y es el primer PR natural. El
slice 2 por sí solo entrega un mutador cuya única seguridad es el análisis
previo fail-closed (sin chequeo de build post-mutación hasta el slice 4);
aceptable porque no existe ruta de escritura remota (D2), pero debe declararse
en la descripción de ese PR.

## Preguntas abiertas

- [ ] **Convergencia con `sdd-spec` sobre D-4.** Si la corrida paralela de spec
      escribió un requisito ADDED de reporte de pin de imagen de build, entra en
      conflicto con este diseño y uno de los dos debe ceder. Posición del
      diseño: el guard test.
- [ ] **Versión de `golang.org/x/mod` en `providers/go/go.mod`.** Debe coincidir
      con la de la raíz; confirmar al aplicar y considerar extender la idea de
      paridad entre módulos de `internal/daggerpin/pin_test.go`.
- [ ] **Nombre del provider.** El diseño asume un único provider, `go-runtime`,
      registrado bajo ambos capability kinds. Confirmar contra la convención de
      nombres de `register.go` al aplicar.
- [ ] **Separador del string de capability**: `runtime-inspect` (elegido,
      coincide con la propuesta aceptada y con los nombres de provider con guion
      del repo) vs `runtime.inspect` (se lee mejor como familia). Rename
      mecánico; decidir antes de que el slice 1 haga merge, no después.
- [ ] **`.dagger/go.mod` en 1.26.5 y los cuatro pines `GO_VERSION: '1.26.1'` de
      workflows son drift real que este cambio deliberadamente no corrige**
      (niveles 2 y 3 de D-4). Conviene abrir un issue de mantenimiento separado
      para que se corrijan por decisión explícita y no queden olvidados.

---

*Nota de desviación*: excede el presupuesto de 800 palabras del skill. Causa: el
brief de lanzamiento exigía firmas Go concretas, esquemas `with` exactos, reglas
de ambigüedad precisas, un pronóstico de líneas por archivo y una recomendación
explícita de slicing. El contenido está comprimido en tablas; sin relleno
narrativo.

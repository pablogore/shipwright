# Diseño: API pública versionada del módulo + modelo de composición + orquestación declarativa de workflows

> Companion en español. La versión canónica y fuente de verdad ante cualquier conflicto es `design.md` (inglés).

> **Nota de revisión — REVISIÓN 2 DEL DISEÑO.** Enmienda, no reemplaza, el diseño anterior. `D-A`–`D-E` se mantienen, con `Pipeline` renombrado a `Plan` en todo el documento. Se elimina por completo: la afirmación de que `pipelines.Registry` registra presets de conjuntos de capacidades y el argumento de migración "UX de `--pipeline go-service` intacta" — una autocontradicción verificada contra el principio de este mismo cambio. Se agrega: `D-F`–`D-N`, que cubren la capa declarativa de workflows, que ahora es el punto de entrada principal de la CLI. La Matriz de Amenazas pasa de `N/A` a aplicable.

## Enfoque técnico

**Dos capas de contrato (sin cambios) más una capa de workflow que es consumidora, no un tercer contrato.**

| Capa | Ubicación | Rol |
|---|---|---|
| 1 — contrato de capacidades | `pkg/shipwright/` (módulo raíz) | Las cinco capacidades como interfaces Go exportadas y planas, sin genéricos. Todo el código interno implementa esto |
| 2 — proyección Dagger | `.dagger/` (módulo Go propio) | Las mismas cinco como **Dagger Interfaces** (embebiendo `DaggerObject`); la composición como Dagger Objects. Adaptadores delgados, sin lógica |
| 3 — capa de workflow | `internal/workflow/**` | Esquema del manifiesto, DAG, resolución de proveedores, planificador. **Consume la capa 1; no define ningún contrato Go nuevo** |

La división entre capas 1 y 2 está forzada, no elegida: `DaggerObject` solo existe dentro del paquete generado por Dagger, y `.dagger/`, al ser un módulo Go separado, no puede importar `github.com/pablogore/shipwright/internal/**`.

La capa 3 vive en `internal/` de forma deliberada (ver D-H). Su contrato público es el **documento YAML**, que es dato — neutral respecto al lenguaje por naturaleza, por lo que satisface "consumible desde varios lenguajes" sin ninguna proyección de tipos.

## Decisiones de arquitectura

### D-A: Dagger Interfaces para las costuras, Dagger Objects para el estado *(sin cambios; `Pipeline` → `Plan`)*

| Opción | Compromiso | Veredicto |
|---|---|---|
| Genéricos de Go en firmas públicas | Sin evidencia de soporte en Functions exportadas de Dagger; prohibido textualmente por la propuesta | Rechazada (restricción vinculante) |
| Solo interfaces Go concretas | No representable en el sistema de tipos de Dagger; alcance cross-language nulo | Rechazada |
| Solo Dagger Objects concretos | Representable, pero un `Deploy` que recibe un tipo de artefacto concreto fuerza un único productor: la explosión combinatoria reaparece a nivel de objeto | Rechazada |
| **Dagger Interfaces (costuras) + Dagger Objects (estado/resultados)** | El tipado estructural da sustituibilidad cross-language; los Objects transportan el estado del encadenamiento | **Elegida** |

**Regla de firmas (control de riesgo):** los métodos de las interfaces de capacidad usan ÚNICAMENTE tipos núcleo de Dagger (`Directory`, `File`, `Container`, `Secret`) y escalares. Ningún Object definido por el módulo aparece en la firma de un *método de interfaz*: ese rincón del sistema de tipos de Dagger no está verificado.

**Hallazgo del spike (WU2, tarea 2.1) — confirmado con un `dagger call` real contra v0.21.8, veredicto GO:** el estado tipado por interfaz de un campo de Object SÍ sobrevive la serialización, pero el codegen del Go SDK de Dagger no puede compilar un proxy de cliente para un método de interfaz que devuelve un tipo núcleo de Dagger encadenable de forma perezosa (`Directory`/`File`/`Container`) junto con un `error` explícito — verificado como reproducible y aislado a esa combinación (un método hermano que devuelve `(string, error)` compiló sin problemas). Por eso `Builder.Build`, `Tester.Test` y `Runner.Run` abajo descartan `error`, siguiendo el propio idiom de encadenamiento perezoso de Dagger (los errores aparecen en la llamada terminal/escalar, p. ej. `Plan.Execute` o el `Sync()` de quien llama). Es una desviación forzada por el codegen, exclusiva de la capa 2, respecto al código de ejemplo original de esta sección — no de la decisión D-A en sí. `Artifactor.Publish`/`Deployer.Deploy` no se ven afectados (devuelven `(string, error)` escalar) y conservan `error`. La capa 1 (`pkg/shipwright`, Go plano, sin codegen de por medio) conserva `error` en los cinco métodos — las dos capas son deliberadamente asimétricas acá, no están desincronizadas.

```go
// .dagger/capabilities.go — la superficie pública de Dagger
type Builder interface {
	DaggerObject
	Build(ctx context.Context, source *dagger.Directory) *dagger.Directory
}
type Tester interface {
	DaggerObject
	Test(ctx context.Context, source *dagger.Directory) *dagger.File
}
type Artifactor interface {
	DaggerObject
	Publish(ctx context.Context, build *dagger.Directory, ref string, creds *dagger.Secret) (string, error)
}
type Deployer interface {
	DaggerObject
	Deploy(ctx context.Context, artifactRef, environment string, creds *dagger.Secret) (string, error)
}
type Runner interface {
	DaggerObject
	Run(ctx context.Context, build *dagger.Directory) *dagger.Container
}

type Shipwright struct{}
func (m *Shipwright) ContractVersion() string
func (m *Shipwright) Plan(source *dagger.Directory) *Plan

type Plan struct{ Source *dagger.Directory } // Dagger Object, estado tipado por interfaz
func (p *Plan) WithBuild(b Builder) *Plan    // + WithTest/WithArtifact/WithDeploy/WithRun
func (p *Plan) Execute(ctx context.Context) (string, error)
```

`Setup` desaparece: la fuente es una *entrada*, no un paso. `BeforeStep`/`AfterStep` salen del contrato; los hooks quedan del lado del host (`interfaces.HookManager`), y eso es lo que hace ortogonales a las capacidades. Los genéricos solo se permiten en helpers no exportados.

**Plan de contingencia si el estado tipado por interfaz no sobrevive la serialización de Dagger** (se demuestra en la porción 2, nunca se asume): colapsar a una Function plana `Plan(ctx, source, build Builder, test Tester, artifact Artifactor, deploy Deployer, run Runner)`. Las interfaces como *argumentos* de Function sí están documentadas; solo la forma con estado encadenado conlleva riesgo. El contrato de capacidades es idéntico en ambos casos, y la capa 3 no se ve afectada porque nunca pasa por `Plan` (D-G).

### D-B: Wiring del módulo y desacoplamiento de los pines de versión *(sin cambios)*

| Elemento | Decisión |
|---|---|
| `dagger.json` (raíz) | `{name: shipwright, sdk: go, source: ".dagger", engineVersion: "v0.21.8"}` |
| Fuente del módulo | `.dagger/` con su **propio** `go.mod` generado por Dagger |
| `go.mod` raíz | Sin cambios: conserva `dagger.io/dagger v0.21.8` para `main.go` / `internal/**` |
| Aislamiento | `.dagger/` es a la vez módulo anidado y directorio con punto, así que `go build ./...` y `go test -race ./...` del raíz nunca lo recorren; sus tests corren con un nuevo `make dagger-test` |
| Acoplamiento de pines | **Resuelto, no diferido**: los dos pines viven en módulos separados y nunca se enlazan. El riesgo residual es la *deriva*, así que se convierte en test |

Guardia de paridad de pines: un test unitario del módulo raíz (sin necesidad de Dagger) lee `engineVersion` de `dagger.json` y la versión de `dagger.io/dagger` del `go.mod` raíz y afirma la igualdad. Si `dagger init` rechaza `v0.21.8`, se suben **ambos** lados en el mismo commit manteniendo ese test verde; nunca se los deja divergir.

### D-C: Prueba en un segundo lenguaje = TypeScript *(sin cambios)*

| Opción | Esfuerzo para este repositorio | Veredicto |
|---|---|---|
| Python (`@interface` sobre `Protocol`) | Corre dentro del motor, sin toolchain en el host; pero la prueba es solo en tiempo de ejecución: un `dagger call` exitoso demuestra un camino, no la forma del contrato | Rechazada |
| **TypeScript (`@interface`)** | También corre dentro del motor (sin Node en el host), pero los bindings generados son `.ts` tipados, así que type-checkear la implementación es una prueba **en tiempo de compilación** de que toda la forma del contrato cruzó la frontera de lenguaje | **Elegida** |

Artefacto: `examples/crosslang-ts/` con su propio `dagger.json` (`sdk: typescript`) dependiendo del módulo raíz. Implementa `Builder` en TypeScript y lo pasa a `shipwright.plan().withBuild(...)`. Aceptación = bindings generados + type-check limpio + un `dagger call` exitoso, documentado como invocación local.

### D-D: `pipelines.Config` se descompone por capacidad *(sin cambios)*

El monolito (Git + Registry + Build + Coverage + SSH + Go/Java) se reemplaza por structs por capacidad en `pkg/shipwright/config.go`, de modo que la ortogonalidad la impone el compilador y no la documentación.

| Struct nuevo | Absorbe |
|---|---|
| `SourceConfig` | GitRepo, GitRef, GitProtocol, GitUserEmail, GitUserName, SSHPrivateKey |
| `BuildConfig` | GoVersion, JavaVersion, BuildMode, BinaryName |
| `TestConfig` | Coverage |
| `ArtifactConfig` | Registry, RegistryURL, RegistryUser, ImageName, ImageTag, BuildTag, CommitSHA, BranchName, Version |
| `DeployConfig`, `RunConfig` | Vacíos en este cambio: adaptadores diferidos |
| **Eliminados** | `Image`, `ImageContainer`, `ImageRef` — los handles vivos de Dagger son estado de ejecución, no configuración |
| **Retipados** | `RegistryPass`, `RegistryToken`, `Token` → `*dagger.Secret` |

**Vinculante para la capa 3:** el mecanismo `spec.secrets` del manifiesto DEBE reutilizar este retipado y el patrón `client.SetSecret` existente de `internal/pipelines/shared/docker.go`. NO DEBE introducir una segunda representación de secretos.

### D-E: Mecánica de versionado *(enmendada — espacios de versión)*

| Elemento | Mecanismo |
|---|---|
| Fuente de verdad | `pkg/shipwright/version.go` → `const ContractVersion = "1.0.0"` |
| Marcador legible por máquina | Proyectado como `Shipwright.ContractVersion()`, legible con `dagger call contract-version` |
| Ruta del módulo Go | Sin sufijo en el contrato v1. Un salto mayor exige sufijo `/v2` + mayor en `ContractVersion` + nota de migración, todo junto |
| Archivo de política | Nuevo `COMPATIBILITY.md` |
| Cumplimiento | `pkg/shipwright/testdata/api.golden` + test de superficie golden; cualquier cambio en la superficie garantizada falla en ROJO |

**Superficie garantizada, deliberadamente mínima:** las cinco interfaces de capacidad (ambas capas), `Shipwright.{Plan, ContractVersion}`, `Plan.{WithBuild, WithTest, WithArtifact, WithDeploy, WithRun, Execute}`, los structs de configuración de `pkg/shipwright` y el esquema de manifiesto `shipwright.dev/v1` (como contrato de *datos*, protegido por su propio golden — ver D-H). Nada más.

**Espacios de versión — cinco ejes independientes.** La Revisión 1 declaraba tres; la capa de workflow agrega dos. Esto cierra la pregunta abierta de la propuesta sobre espacios de versión:

| Eje | Portador | Garantía |
|---|---|---|
| Versión del contrato | `ContractVersion` | Estable desde el primer release, solo la superficie garantizada |
| Versión del esquema del manifiesto | `apiVersion: shipwright.dev/v1` en cada documento | Evoluciona de forma independiente de `ContractVersion`; solo aditiva dentro de `v1` |
| SemVer del release de la CLI | goreleaser + `CHANGELOG.md` | Versionado de release ordinario |
| Pin del motor | `engineVersion` de `dagger.json` | Debe coincidir con el pin del cliente en el `go.mod` raíz (test de paridad D-B) |
| **Versión del proveedor (`uses.version`)** | Paso del manifiesto, propiedad del proveedor | **No cubierta por ninguna garantía de Shipwright** |

`uses.version` y `ContractVersion` son **ortogonales**, no el mismo espacio. La garantía de estabilidad desde el primer release cubre únicamente la superficie enumerada arriba; **no** se extiende a proveedores internos ni de terceros, ni a sus esquemas `with`, ni a su semántica de versiones. Un proveedor puede romper a sus propios usuarios en cualquier versión sin tocar `ContractVersion`. `COMPATIBILITY.md` DEBE declarar esta exclusión de forma explícita.

### D-F: Sin presets con nombre — implementaciones de capacidad sin identidad de paquete

La afirmación del diseño anterior ("`pipelines.Registry` registra factories de conjuntos de capacidades, manteniendo intacta la UX de `--pipeline go-service`") queda **eliminada**. Reintroducía el antipatrón del preset con nombre bajo un argumento de compatibilidad que es nulo con cero consumidores externos.

| Opción | Compromiso | Veredicto |
|---|---|---|
| Registro de factories de conjuntos de capacidades indexado por nombre de preset | Preserva la UX de la CLI; **es** el antipatrón con otro nombre | Rechazada (el defecto verificado) |
| Un paquete `internal/capabilities/goservice` con las cinco | No hay *tipo* preset, pero la ruta del paquete nombra el bundle | Rechazada |
| **Paquete plano `internal/capabilities`, un archivo y un tipo exportado por implementación** | Ninguna identidad de bundle en tipo, archivo ni ruta; cada implementación se registra por separado | **Elegida** |

| Método legado de `go-service` | Nueva implementación | Capacidad |
|---|---|---|
| `Setup` | Eliminado — la fuente es una entrada `*dagger.Directory` | — |
| `Build`, `buildBinary`, `buildDocker` | `GoBuilder` | `Builder` |
| `Test` | `GoUnitTester` | `Tester` |
| `Lint` | `GoLinter` | `Tester` |
| `Vuln` | `GoVulnScanner` | `Tester` |
| `Package`, `Tag`, `Push` | `ContainerPublisher` | `Artifactor` |
| `BeforeStep`, `AfterStep` | Eliminados del contrato — `interfaces.HookManager` gestiona los hooks | — |

Tres implementaciones independientes de `Tester` son la ganancia de ortogonalidad, y son lo que permite que `capability: test` con tres proveedores distintos funcione en un manifiesto. `Deployer`/`Runner` se entregan solo como contrato. La regla de nombres la impone un test: un golden sobre los identificadores exportados de `internal/capabilities` falla si algún identificador nombra un bundle de stack.

### D-G: `Pipeline` → `Plan`, y `Plan` no es el DAG

| Opción | Compromiso | Veredicto |
|---|---|---|
| Conservar `Pipeline` | Ya es genérico y compuesto, así que no es defectuoso; pero la palabra invita a regresiones `GoPipeline`/`JavaPipeline`, y la ronda anterior demuestra que esa atracción es real | Rechazada (renombre preventivo, decisión final del usuario) |
| `Compose` | Es un verbo; se lee mal como tipo y como `dagger call compose` | Rechazada |
| **`Plan`** | Sustantivo, corto, sin connotación de stack, se lee bien encadenado y desde otros lenguajes (`shipwright.plan().withBuild(...)`) | **Elegida** |

**`Plan` es la superficie de composición programática y cross-language. NO es la representación del manifiesto.** Una cadena de cinco ranuras no puede expresar las aristas `needs[]`, y forzar el grafo a través de ella aplanaría el DAG o inflaría la superficie garantizada con primitivas de grafo. Por lo tanto, la capa 3 compila un manifiesto a un `graph.Graph` interno de instancias de capacidad resueltas e invoca directamente las interfaces de la capa 1. El invariante compartido entre ambos front-ends es **la interfaz de capacidad, no el objeto de composición**.

Guardia terminológica: el artefacto compilado por el motor es un `Graph`, nunca un "plan". `Plan` queda reservado para el Dagger Object.

### D-H: Representación del esquema del manifiesto

| Opción | Compromiso | Veredicto |
|---|---|---|
| Archivo JSON Schema + validador en tiempo de ejecución | Verificable por máquina y publicable, pero agrega dependencia y duplica la forma en Go | Rechazada (nueva dependencia; cercanía a un no-objetivo) |
| `map[string]any` + recorrido manual de claves | Sin tipos nuevos, pero cada acceso es una búsqueda sin tipo: exactamente la superficie de error que advierte el skill de seguridad | Rechazada |
| **Structs Go tipados + `gopkg.in/yaml.v3` con `KnownFields(true)`, en `internal/workflow/manifest`** | Ya es dependencia directa (`go.mod`), sin unmarshalers personalizados, los campos desconocidos fallan cerrado, y refleja el patrón existente de `internal/config/yaml_parser.go` | **Elegida** |

El control de deriva del esquema replica D-E: `internal/workflow/manifest/testdata/schema.golden` registra el conjunto de campos aceptados; cualquier cambio de esquema falla en ROJO y fuerza una decisión explícita de `apiVersion` en el mismo PR.

**Simplificación del esquema (decisión de diseño; la lista de campos de la propuesta era explícitamente no normativa):** se **elimina** `spec.steps[].outputs`. Cada capacidad devuelve exactamente un resultado tipado, por lo que el resultado de un paso se direcciona como `${{ steps.<id>.output }}`, con su tipo fijado por la capacidad. Una capa de alias agregaría superficie de nombres sin poder expresivo. La entrada tipada `Directory` de un paso se enlaza con `steps[].input` (por defecto `spec.source`).

**La validación es una tubería fija de siete etapas, y nada se ejecuta hasta que todas admiten el documento** (sin ejecución parcial):

| # | Etapa | Falla cerrado ante |
|---|---|---|
| 1 | Lectura con tope de tamaño + decodificación | Archivo por encima del tope; YAML mal formado; campo desconocido |
| 2 | Identidad del documento | `apiVersion`/`kind` fuera de la lista permitida |
| 3 | Estructura | `id` de paso vacío o duplicado; `capability` fuera de las cinco; `uses` ausente; `uses.version` vacío (`requireVersion`) |
| 4 | Referencias | `needs` apuntando a un paso inexistente; referencia de interpolación a una variable/secreto/paso no declarado |
| 5 | Grafo | Ciclo; referencia de dato sin arista `needs` correspondiente; tipo de salida incompatible con el tipo de entrada del consumidor |
| 6 | Resolución de proveedor | Ningún proveedor registrado para `(capacidad, nombre)`; versión no soportada |
| 7 | Enlace de valores | Referencia a secreto en un campo no tipado como secreto (`forbidPlaintext`); tipo incompatible en `with` |

### D-I: Resolución de proveedores — registro tipado, módulos externos solo en tiempo de compilación

| Opción | Compromiso | Veredicto |
|---|---|---|
| Un `map[string]func(...) any` + aserción de tipo en el uso | Registro único, pero los errores de resolución afloran en ejecución y `any` anula el tipado de capacidades | Rechazada |
| Reutilizar `internal/plugins/loader.go` (`plugin.Open` sobre una ruta `.so`) | Maquinaria existente, pero ejecuta código nativo arbitrario en proceso desde una ruta controlada por el manifiesto: la superficie de mayor severidad del repositorio | **Rechazada (seguridad)** |
| Descargar referencias `module:` en tiempo de ejecución | Coincide literalmente con la sintaxis ilustrativa `github.com/acme/custom-builder`, pero es un servicio de registro de paquetes: un no-objetivo explícito | Rechazada |
| **Cinco métodos de registro tipados; `module:` resuelve solo a proveedores ya compilados y autorregistrados** | Resolución totalmente tipada, falla cerrado, cero superficie de ataque nueva en ejecución | **Elegida** |

```go
// internal/workflow/providers/registry.go
type Registry struct{ /* mapas por capacidad, protegidos */ }

func (r *Registry) RegisterBuilder(ref Ref, schema WithSchema, f func(Values) shipwright.Builder)
// + RegisterTester / RegisterArtifactor / RegisterDeployer / RegisterRunner

func (r *Registry) ResolveBuilder(ref Ref, v Values) (shipwright.Builder, error)
// + un Resolve* por capacidad

type Ref struct{ Name, Module, Version string } // Module == "" significa interno al repositorio
```

`WithSchema` es la declaración que hace el proveedor de sus claves `with` como `nombre → tipo{string,int,bool,secret}`. Es lo que hace verificable la etapa 7 y lo que garantiza que los proveedores reciban **valores tipados, nunca una cadena de shell** (la regla `sh -c` del skill de seguridad).

**Límite de alcance, declarado de forma explícita:** `uses.module` se resuelve *únicamente* contra proveedores que se autorregistraron en tiempo de compilación (un import de Go en el archivo de registro de proveedores del binario). No hay descarga, ni caché, ni carga de `.so`, ni servicio de registro. Una referencia `module:` no registrada falla cerrado en la etapa 6 con un error que nombra la ruta del módulo. Ampliar esto es un cambio de seguimiento y requeriría su propia revisión de seguridad.

### D-J: Construcción del DAG y detección de ciclos — algoritmo de Kahn

| Opción | Compromiso | Veredicto |
|---|---|---|
| Marcado DFS de tres colores | Correcto para detectar, pero recursión sobre un grafo provisto por el usuario, y el orden topológico exige una segunda pasada | Rechazada |
| Matriz de alcanzabilidad / clausura transitiva | Fácil de razonar, pero O(n³) y reporta un ciclo sin nombrar sus miembros | Rechazada |
| Nueva dependencia de librería de grafos | Probada en producción, pero es una dependencia nueva para ~60 líneas | Rechazada |
| **Algoritmo de Kahn (olas por grado de entrada)** | Detecta ciclos y produce el orden topológico en una sola pasada; el fan-in en diamante se maneja de forma nativa por el conteo de grado de entrada, que es justo donde las variantes ingenuas de DFS con conjunto de visitados producen falsos positivos | **Elegida** |

Reporte de ciclos: al drenar Kahn, todo nodo con grado de entrada residual > 0 está en un ciclo o aguas abajo de uno. El error enumera esos ids para que el autor pueda actuar.

Invariantes impuestos (cada uno con un test en ROJO antes de la implementación): arista propia; par mutuo; ciclo largo (4+); **fan-in en diamante aceptado** (`b` y `c` dependen de `a`, `d` depende de ambos); componentes desconectados aceptados (varias raíces); `needs` a un id desconocido rechazado con un error *distinto* al de ciclo; ids duplicados rechazados; referencia de dato sin arista `needs` declarada rechazada (evita una lectura sin orden de la salida de otro paso); incompatibilidad de tipo salida/entrada rechazada antes de ejecutar nada.

`policies.dependencies.forbidCycles` es por lo tanto una **verificación impuesta con test que falla**, no documentación. En este cambio no tiene ajuste permisivo: la aciclicidad es incondicional, y el campo de política registra la intención para estabilidad del esquema.

### D-K: Planificación — olas secuenciales por nivel, paralelismo declarado como cota superior

| Opción | Compromiso | Veredicto |
|---|---|---|
| Pool de trabajadores acotado que honre `maxParallel` ahora | Coincide literalmente con el esquema, pero agrega uso concurrente del cliente Dagger, propagación de cancelación, semántica de fallo parcial y superficie para el detector de carreras: duplica aproximadamente la carga de revisión del motor | Rechazada para este cambio |
| Ignorar `spec.execution` por completo | Lo más pequeño, pero descarta en silencio un campo declarado | Rechazada |
| **Ejecutar las olas de Kahn en orden, secuencialmente dentro de cada ola en el orden de declaración del manifiesto; `maxParallel` validado y registrado, todavía sin ampliar la concurrencia** | La ejecución secuencial es una planificación *correcta* del mismo DAG para cualquier `maxParallel ≥ 1`, así que nada se informa de forma engañosa; el límite de ola es exactamente la costura donde entra luego un pool de trabajadores | **Elegida** |

Dicho sin adornos: un manifiesto que declara `maxParallel: 4` corre correctamente pero en serie. `maxParallel` es una cota superior, no un requisito. `maxParallel: 0` o negativo falla la validación en la etapa 3.

Se implementan ahora porque cada uno es pequeño y testeable de forma independiente: timeout por paso (`context.WithTimeout`), reintento acotado por paso y fail-fast (ante el error de un paso, se dejan de planificar olas y se devuelve un error que nombra el id del paso). Diferido con costura explícita: la ampliación concurrente dentro de una ola.

### D-L: Interpolación — gramática restringida de marcadores, los secretos nunca se vuelven cadenas

| Opción | Compromiso | Veredicto |
|---|---|---|
| `text/template` | Librería estándar, pero admite llamadas a funciones y recorrido reflexivo de campos: un evaluador general | **Rechazada (sin eval)** |
| Una librería de expresiones tipo CEL/`expr` | Condiciones potentes, pero es una dependencia nueva *y* una superficie de evaluación de expresiones arbitrarias | **Rechazada (sin eval)** |
| `os.Expand` | Diminuto, pero la semántica `$VAR` filtra el entorno del proceso hacia el manifiesto | Rechazada |
| **Escáner escrito a mano sobre una gramática fija que produce valores tipados** | ~80 líneas, sin operadores, sin llamadas a funciones, sin anidamiento, sin acceso al entorno; toda referencia es resoluble estáticamente en la etapa 4 | **Elegida** |

Gramática, cerrada y completa:

```
placeholder := "${{" ws ref ws "}}"
ref         := "variables." name
             | "secrets."   name
             | "steps."     name ".output"
name        := [A-Za-z_][A-Za-z0-9_-]*
```

Cualquier otra cosa — un operador, una llamada a función, un marcador anidado, un tercer namespace, un segmento de ruta adicional — es un **error de parseo en la etapa 4**, no un repliegue a texto literal. El escáner emite `[]Token`, donde cada token es un tramo literal o una referencia resuelta; no existe paso de evaluación al que atacar.

**La regla de secretos, impuesta mecánicamente en lugar de documentada:**

```go
type Kind int // KindString | KindInt | KindBool | KindSecret

type Value struct {
	kind   Kind
	str    string          // nunca se asigna cuando kind == KindSecret
	secret *dagger.Secret  // solo se asigna cuando kind == KindSecret
}
```

- `secrets.*` resuelve a un `Value{kind: KindSecret, secret: ...}` y a nada más. No existe accesor que devuelva un secreto como `string`.
- Una referencia `secrets.*` en un campo cuyo tipo declarado por el proveedor no es `KindSecret` es un **error de validación de la etapa 7** (`forbidPlaintext`), nunca una sustitución. Así, ningún camino de código puede producir un secreto en texto plano, porque el camino que produce cadenas no puede contener un valor secreto.
- Un marcador que mezcla una referencia a secreto con texto literal en un mismo campo (`"Bearer ${{ secrets.tok }}"`) se rechaza por la misma razón: la concatenación exigiría una forma de cadena.
- El texto plano existe en exactamente un límite, sin cambios respecto al código actual: la llamada `client.SetSecret(name, value)` que lee la variable de entorno configurada. Ese valor nunca entra al escáner, ni a una cadena derivada del manifiesto, ni a una línea de log. Los tipos nuevos que transporten valores implementan `String()` omitiendo secretos, siguiendo a `GitCredentials.String()`.

**Las condiciones son estructuradas, no expresiones.** `when` es un mapa de predicados YAML sobre las mismas referencias restringidas (por ejemplo `when: {branch: [main, develop]}`), evaluado por coincidencia exacta. Un `when` con expresión en cadena reintroduciría el evaluador que esta decisión existe para evitar.

### D-M: Las compuertas de aprobación son solo metadatos

`spec.environments.<name>.approvals` se **parsea, se valida por buena formación, se expone en la descripción del workflow y en los logs, y el planificador nunca la consulta.** No se entrega máquina de estados bloqueante, ni almacén de aprobaciones, ni verificación de identidad de revisores, ni compuerta alguna en `workflow-execution`.

**Fundamento:** una compuerta con cumplimiento real necesita estado de aprobación duradero, una fuente de identidad y un camino de reanudación: tres subsistemas cada uno más grande que el propio motor, y los tres dentro del no-objetivo "motor de políticas completo / UI de flujo de aprobaciones". Una compuerta a medias es peor que una declarada, porque parecería un control efectivo.

**Divergencia explícita respecto de la propuesta, señalada para `sdd-spec` y `sdd-tasks`:** el criterio de éxito de la propuesta *"Un entorno con compuerta de aprobación bloquea su paso dependiente hasta que se registre la aprobación"* queda **superado** por esta decisión (corrección explícita del usuario). El criterio de reemplazo es: *un manifiesto que declara aprobaciones parsea, valida y ejecuta sin bloquear, y un test en ROJO afirma que el motor no aplica compuerta.*

### D-N: Punto de entrada de la CLI y el acoplamiento de secuenciación de la eliminación

El manifiesto es el punto de entrada **principal**. `main.go` conserva el paquete `flag` de la librería estándar (migrar fuera de `flag` sigue siendo un no-objetivo).

| Flag | Disposición |
|---|---|
| `--workflow <ruta>` | **Nuevo.** Ruta del manifiesto. Por defecto `.shipwright/workflow.yaml`; un archivo ausente falla cerrado nombrando la ruta esperada, nunca un repliegue implícito al camino legado |
| `--step <id>` | **Reorientado** a un id de paso del manifiesto. Ejecuta la clausura transitiva de `needs` de `<id>` en orden topológico y se detiene: una corrida de subgrafo, calculada por alcanzabilidad sobre el grafo ya construido. Preserva el patrón existente de invocación por paso en CI |
| `--list-steps` | **Reorientado** a listar ids de pasos del manifiesto con su capacidad y proveedor resuelto |
| `--pipeline`, `--list-pipelines` | **Eliminados** — nombran presets |
| `--only-build`, `--only-test`, `--skip-push` | **Eliminados** — presuponen los nombres de paso del preset; `--step` los reemplaza |
| `--config .shipwright.yml`, `--env`, `--executor`, `--verbose`, `--version`, `--health`, `--local`, flags de git | Sin cambios |

**Acoplamiento de secuenciación, hecho concreto (era un riesgo de la propuesta; ahora es una regla de orden de porciones):** la porción del punto de entrada del manifiesto entra **estrictamente antes** que la porción de eliminación del preset, y durante exactamente una porción funcionan tanto `--workflow` como `--pipeline`. Ningún punto de merge deja a la CLI sin poder ejecutar nada. Las dos porciones forman **un único límite de rollback**: revertir la eliminación sin el punto de entrada está bien (vuelven ambos caminos), pero revertir el punto de entrada sin la eliminación no deja ningún camino, así que un revert que cruce ese límite debe llevarse ambas.

## Flujo de datos

```
 A. Camino del manifiesto (PRINCIPAL)           B. Camino programático
 ───────────────────────────────────            ──────────────────────
 shipwright --workflow wf.yaml                  dagger call | módulo TS | consumidor Go
        │                                              │
        ▼                                              ▼
 manifest.Parse  (etapas 1-3)                   Shipwright.Plan(source: Directory)
        │                                        .withBuild(Builder).withTest(Tester)...
        ▼                                              │
 interp.Scan     (etapa 4, Values tipados)             ▼
        │                                        Plan (Dagger Object,
        ▼                                              estado tipado por interfaz)
 graph.Build + Kahn  (etapa 5)                         │ Execute()
        │                                              │
        ▼                                              │
 providers.Resolve   (etapa 6)  ───────┐               │
        │                              │               │
        ▼                              ▼               ▼
 values.Bind         (etapa 7)   ┌─────────────────────────────────┐
        │                        │  Capa 1  pkg/shipwright         │
        ▼                        │  Builder / Tester / Artifactor  │
 engine.Execute (olas) ─────────►│  Deployer / Runner              │
                                 └───────────────┬─────────────────┘
                                                 ▼
                                 internal/capabilities/{GoBuilder,
                                   GoUnitTester, GoLinter,
                                   GoVulnScanner, ContainerPublisher}
                                                 ▼
                                 Tipos núcleo de Dagger (Directory/File/Container/Secret)
```

Secuencia — una corrida de manifiesto (admisión con falla cerrada antes de cualquier efecto):

```
CLI      manifest   interp   graph   providers   engine   capacidad   Dagger
 │──parse──►│
 │          │──ok────►│  escanea refs (sin eval)
 │                    │──ok───►│ construye + Kahn → orden | CICLO→aborta
 │                             │──ok────►│ resuelve | DESCONOCIDO→aborta
 │                                       │──ok──►│ enlaza Values
 │                                       │       │ SECRETO-EN-CADENA→aborta
 │                                       │       │──ola 1───►│──►│ Directory/File
 │                                       │       │◄─resultado│   │
 │                                       │       │──ola 2───►│──►│
 │◄────────────── resumen / id del primer paso fallido ──────┘
```

Nada entre `parse` y `bind` toca la red, el sistema de archivos más allá del manifiesto y su fuente declarada, ni un contenedor.

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `pkg/shipwright/{capabilities,config,version}.go` | Crear | Contrato de la capa 1, configuraciones por capacidad, `ContractVersion` |
| `pkg/shipwright/testdata/api.golden` | Crear | Golden de la superficie garantizada |
| `dagger.json`, `.dagger/**` | Crear | Wiring del módulo + adaptadores de la capa 2, `Shipwright`, `Plan` |
| `internal/capabilities/{gobuilder,gounittester,golinter,govulnscanner,containerpublisher}.go` | Crear | Implementaciones autónomas, sin identidad de bundle (D-F) |
| `internal/workflow/manifest/{schema,parse,validate}.go` + `testdata/schema.golden` | Crear | `workflow-manifest`: esquema tipado, decodificación, etapas 1-3 |
| `internal/workflow/interp/{scan,value}.go` | Crear | Escáner de gramática restringida, `Value` tipado (D-L) |
| `internal/workflow/graph/{build,kahn}.go` | Crear | Adyacencia, Kahn, validación de ciclos y tipos (D-J) |
| `internal/workflow/providers/{registry,register.go}` | Crear | Resolución tipada; registro de proveedores internos (D-I) |
| `internal/workflow/engine/{execute,subgraph}.go` | Crear | Planificador por olas, timeout, reintento, fail-fast, clausura de `--step` (D-K) |
| `main.go` | Modificar | Se agrega `--workflow`; `--step`/`--list-steps` reorientados; flags de preset eliminados (D-N) |
| `internal/pipelines/{pipeline,registry,options}.go` | Eliminar | Registro de presets y contrato legado — no se retipan |
| `internal/pipelines/common/interfaces.go` | Eliminar | Código muerto |
| `internal/pipelines/go-service/**` | Eliminar | Superado por `internal/capabilities/**` |
| `internal/interfaces/interfaces.go` | Modificar | `Pipeline` eliminado; `Container`/`StepRegistry`/`HookManager` retipados; `Artifact` → `StepArtifact` (colisión de nombres) |
| `internal/app/{container,pipeline_executor,step_registry,hook_manager}.go` | Modificar | **Mayor radio de impacto legado** — el wiring de DI se retipa sobre la capa 1 |
| `internal/plugins/{interfaces,context}.go` | Modificar | `GetPipeline()` → `GetCapabilities()`; `GetPipelineConfig()` → `GetConfig()` |
| `internal/executors/{selector,docker_executor}.go` | Modificar | Retipados fuera de `pipelines.Config` |
| `mocks/**`, `internal/*/mocks.go` | Regenerar | Siguen a las interfaces retiradas |
| `examples/crosslang-ts/**` | Crear | Prueba en TypeScript (D-C) |
| `examples/workflow/*.yaml` | Crear | Manifiesto ejecutable, incluyendo el caso de fan-in en diamante |
| `COMPATIBILITY.md` | Crear | Superficie garantizada, cinco ejes de versión, exclusión de la versión de proveedor |
| `docs/API.md`, `docs/ARCHITECTURE.md` | Modificar (mínimo) | Dejan de presentar `Pipeline` como canónico |
| `Makefile` | Modificar | `make dagger-test` |

## Interfaces / Contratos

```yaml
apiVersion: shipwright.dev/v1
kind: Workflow
metadata:
  name: go-service-release
spec:
  source:
    path: .                        # debe resolver dentro del árbol del manifiesto
  variables:
    imageRef: ghcr.io/acme/api
  secrets:
    registry: {fromEnv: REGISTRY_PASSWORD}
  steps:
    - id: build
      capability: build
      uses: {provider: go, version: "1"}
      with: {goVersion: "1.26.1"}
    - id: unit                     # diamante: unit y vuln dependen de build
      capability: test
      uses: {provider: go-test, version: "1"}
      needs: [build]
      input: ${{ steps.build.output }}
    - id: vuln
      capability: test
      uses: {provider: govulncheck, version: "1"}
      needs: [build]
      input: ${{ steps.build.output }}
    - id: publish                  # fan-in
      capability: artifact
      uses: {provider: container, version: "1"}
      needs: [unit, vuln]
      input: ${{ steps.build.output }}
      with:
        ref: ${{ variables.imageRef }}
        creds: ${{ secrets.registry }}   # clave tipada como secreto — una clave string fallaría la etapa 7
      when: {branch: [main]}
  execution:
    concurrency: {maxParallel: 4}    # cota superior; la ejecución serial es una planificación válida
    failFast: true
    timeout: 30m
  environments:
    production:
      approvals: {required: [platform-team]}   # solo metadatos (D-M)
  policies:
    dependencies: {forbidCycles: true}
    providers:    {requireVersion: true}
    secrets:      {forbidPlaintext: true}
```

## Secuencia de migración (también las costuras de las porciones de PR)

| # | Porción | Toca | Radio de impacto |
|---|---|---|---|
| 1 | Contrato de la capa 1 | `pkg/shipwright/**` + goldens | Ninguno — aditiva |
| 2 | Wiring del módulo + capa 2 (`Plan`) | `dagger.json`, `.dagger/**`, test de paridad de pines, `Makefile` | Ninguno sobre el código raíz |
| 3 | Implementaciones de capacidad | `internal/capabilities/**` (a partir de la lógica de `go-service`; el original queda en su lugar) | Bajo — aditiva |
| 4 | Esquema del manifiesto + parser | `internal/workflow/manifest/**` + golden de esquema | Ninguno — aditiva |
| 5 | Interpolación + valores tipados | `internal/workflow/interp/**` | Ninguno — aditiva |
| 6 | Grafo + Kahn + verificación de tipos | `internal/workflow/graph/**` | Ninguno — aditiva |
| 7 | Registro y resolución de proveedores | `internal/workflow/providers/**` (registra la porción 3) | Ninguno — aditiva |
| 8 | Motor de ejecución | `internal/workflow/engine/**`, `examples/workflow/*.yaml` | Ninguno — aditiva |
| 9 | **Punto de entrada del manifiesto en la CLI** | `main.go` (`--workflow`, reorientación de `--step`/`--list-steps`) | Medio — **tras esta porción funcionan ambos caminos de la CLI** |
| 10 | Retipado de DI y plugins sobre la capa 1 | `internal/app/**`, `internal/interfaces/interfaces.go`, `internal/plugins/{interfaces,context}.go`, `internal/executors/**`; `pipelines.Pipeline` legado se conserva como shim deprecado para que `--pipeline` siga corriendo | **El mayor** |
| 11 | **Eliminación del preset y del shim** | `internal/pipelines/{pipeline,registry,options}.go`, `common/interfaces.go`, `go-service/**`, flags de preset en `main.go`, regeneración de `mocks/**` | Alto — **emparejada en rollback con la porción 9** |
| 12 | Prueba cross-language + documentación | `examples/crosslang-ts/**`, `docs/*`, `COMPATIBILITY.md` | Bajo |

Reglas de orden no negociables: **9 antes de 11** (nunca una CLI que no pueda ejecutar nada); **10 antes de 11** (eliminar el shim antes del retipado de DI deja el árbol sin compilar); las porciones 4-8 son estrictamente aditivas y se pueden revisar de forma independiente del árbol legado. El rollback es en orden inverso de merge, tomando 9 y 11 juntas.

## Estrategia de testing

TDD estricto: cada fila arranca en ROJO. Compuertas por porción: `go test -race ./...` verde, `go build -o shipwright .` verde, cobertura ≥ 90% local / 70% CI, `golangci-lint run` sin funciones por encima de gocyclo 15.

| Nivel | Qué testear | Enfoque |
|---|---|---|
| Unitario — capa 1 | Forma del contrato, descomposición de configuración, `ContractVersion`, paridad de pines, golden de superficie garantizada, golden de nombres de `internal/capabilities` | `_test.go` en paquete; las interfaces de capacidad tienen un método → stubs a mano (orden de dobles 4) |
| Unitario — manifest | Tablas por etapa de validación: campo desconocido, `apiVersion` inválido, id duplicado o vacío, capacidad fuera de las cinco, `uses` ausente, `uses.version` vacío, `maxParallel ≤ 0`, deriva del golden de esquema | En paquete, fixtures `testdata/*.yaml` |
| Unitario — interp | Cada forma gramatical rechazada (operador, llamada a función, marcador anidado, namespace desconocido, segmento extra); `Value` sin accesor de cadena para secretos; secreto en campo string rechazado; concatenación secreto+literal rechazada | Tablas; aserción a nivel de compilación de que ningún accesor exportado devuelve un secreto como `string` |
| Unitario — graph | Arista propia, par mutuo, ciclo largo, **fan-in en diamante aceptado**, componentes desconectados aceptados, id desconocido en `needs` (error distinto), referencia de dato sin `needs`, incompatibilidad de tipos salida/entrada, corrección de la clausura de `--step` | Tablas; el mensaje de error de ciclo afirma los ids implicados |
| Unitario — providers | Acierto/fallo de resolución por capacidad, versión no soportada, `module:` no registrado falla cerrado, incompatibilidad de tipo en `with` | Tablas con un proveedor falso |
| Unitario — engine | Orden de olas determinista, fail-fast detiene olas posteriores, el timeout por paso dispara, reintento acotado, las aprobaciones NO bloquean, la cancelación de contexto se propaga | Implementaciones falsas de capacidad que registran el orden de invocación |
| Unitario — `.dagger` | Conformidad de adaptadores `var _ Builder = (*goBuilder)(nil)` por capacidad | `make dagger-test` |
| Integración (`testing.Short()`) | Esqueleto caminante con `dagger call`; round-trip del estado tipado por interfaz (valida D-A y dispara su contingencia si falla); un manifiesto real de punta a punta | En paquete, con guarda short |
| Cross-language | El módulo TS type-checkea contra los bindings generados; un `dagger call` exitoso | `examples/crosslang-ts/` |
| Adversarial | Las filas de la matriz de amenazas de abajo, un test por fila aplicable | En paquete, por fixtures |

## Matriz de amenazas

**Aplicable — el veredicto `N/A` anterior ya no se sostiene.** Esta revisión agrega un límite de entrada expuesto al host: la CLI lee un YAML de manifiesto de autoría externa, selecciona a partir de él un directorio fuente y resuelve referencias que transportan credenciales.

| Límite | Casos adversariales mínimos | Aplicabilidad | Respuesta de diseño | Tests en ROJO previstos |
|---|---|---|---|---|
| Rutas con apariencia de documentación | `requirements.txt`, Markdown ejecutable, `README.sh` | **N/A** — el motor no clasifica nada por nombre de archivo; solo actúa sobre ids de paso que un manifiesto declara explícitamente | — | — |
| Selección de repositorio Git / fuente | `spec.source.path` con `..`, ruta absoluta, symlink que escapa del árbol; `spec.source` con ref git | **Aplicable** | `path` DEBE resolver (tras `filepath.Abs` + resolución de symlinks) dentro del propio árbol de directorios del manifiesto; los escapes fallan cerrado antes de cualquier llamada a Dagger. Las fuentes git reutilizan la cascada existente de `internal/pipelines/shared/credentials.go`: ningún mecanismo de autenticación nuevo | Un test por forma de escape (`..`, absoluta, symlink), cada uno afirmando un error específico, más un caso aceptado dentro del árbol |
| Estado de commit | staged, `commit -a`, índice vacío | **N/A** — no existe ninguna operación de escritura Git en este cambio | — | — |
| Estado de push | rama de seguimiento, primer push, refspec explícito | **N/A** — la publicación de imágenes pasa por `WithRegistryAuth`, no por un push de Git | — | — |
| Comandos de PR | `--head`, prefijo de entorno, comandos compuestos | **N/A** — sin automatización de VCS/PR | — | — |

Cuatro límites adicionales son el riesgo real de este cambio y llevan tests en ROJO como requisitos de diseño ordinarios:

| # | Límite | Respuesta de diseño | Test en ROJO |
|---|---|---|---|
| 1 | **YAML de manifiesto no confiable** — entrada mal formada, documentos profundamente anidados o amplificados por alias ("billion laughs"), archivos sobredimensionados | El decodificador es `gopkg.in/yaml.v3` (ya dependencia directa) hacia structs tipados con `KnownFields(true)` y **sin unmarshalers personalizados**, así que decodificar no ejecuta nada: el riesgo es consumo de recursos, no ejecución de código. **No** dependemos de los límites internos de la librería: el manifiesto se lee a través de un tope explícito con `io.LimitReader` antes de decodificar, y la profundidad de anidamiento la acota el propio esquema tipado | Archivo sobredimensionado rechazado por el tope; un fixture de amplificación por alias completa dentro de un presupuesto acotado de tiempo/memoria; YAML mal formado devuelve un error específico |
| 2 | **Inyección por interpolación / filtración de secretos** | La gramática cerrada de D-L no tiene operadores, funciones, anidamiento ni acceso al entorno, así que no hay expresión en la que inyectar. `Value` no puede contener a la vez un secreto y una cadena, y no expone accesor de cadena para secretos, por lo que un secreto en texto plano es irrepresentable, no meramente desaconsejado. `forbidPlaintext` es la etapa 7 | Secreto en una clave `with` tipada como string rechazado; concatenación secreto+literal rechazada; toda forma fuera de la gramática rechazada; ningún accesor exportado devuelve un secreto como `string` |
| 3 | **Proveedores externos `uses.module` — cadena de suministro** | La resolución nunca descarga, cachea ni hace `plugin.Open` de nada (D-I rechaza explícitamente reutilizar `internal/plugins/loader.go` para proveedores declarados en el manifiesto). Una referencia `module:` solo puede nombrar código ya compilado dentro de este binario y autorregistrado en tiempo de compilación, así que un manifiesto no puede introducir código que el operador no haya construido. La preocupación residual es la revisión ordinaria de dependencias Go, que `go.mod`/`go.sum` ya cubre | Una referencia `module:` no registrada falla cerrado nombrando la ruta; un test afirma que ningún camino alcanzable desde un manifiesto llama a `plugin.Open` |
| 4 | **Construcción de argumentos de proveedor** | Los proveedores reciben `Values` tipados por su `WithSchema` declarado, nunca una cadena de shell interpolada. Cualquier invocación en contenedor usa `WithExec` con arreglo argv, nunca `sh -c` con un valor proveniente del manifiesto (el patrón existente de `fmt.Sprintf` dentro de `sh -c` en `nomad_deploy.go` explícitamente no debe copiarse) | Un valor `with` con metacaracteres de shell (`'; rm -rf /`) llega al proveedor como un único elemento argv inerte |

Además, se mantiene de la Revisión 1: las credenciales cruzan el contrato público únicamente como `*dagger.Secret`. El test golden de superficie es un diff textual, no una verificación semántica de tipos — no rechaza automáticamente un campo de credencial en texto plano por sí solo; obliga a que cualquier cambio de la superficie exportada, incluido ese caso, pase por un diff revisable con `-update` que una persona debe leer y aceptar deliberadamente, nunca aceptar a ciegas (así lo indica el propio comentario de documentación de `pkg/shipwright/api_golden_test.go`). La garantía real es esa disciplina, no una regla automática. El enrutamiento de pasos/capacidades DEBE fallar cerrado — un id de paso desconocido o una capacidad ausente devuelven error, nunca un salto silencioso.

## Migración / Despliegue

Solo código. Sin migración de estado, datos ni release; la capa de workflow es terreno nuevo, así que no hay manifiestos existentes que migrar. Dos cambios visibles para el usuario aterrizan en la porción 11: se eliminan `--pipeline`/`--list-pipelines`/`--only-build`/`--only-test`/`--skip-push`, y `--workflow` pasa a ser el punto de entrada. No hay consumidores externos a notificar; la ruptura de la API de plugins afecta solo a `internal/plugins/nomad_deploy.go` dentro del repositorio. `docs/API.md` y `docs/ARCHITECTURE.md` reciben la corrección mínima en la porción 12, y esa misma porción documenta las eliminaciones de flags.

## Preguntas abiertas

- [x] ¿Dagger v0.21.8 serializa campos tipados por interfaz en el estado de encadenamiento de un Object? **Resuelto (WU2, tarea 2.1): GO.** Demostrado con una query encadenada real de `dagger` que produjo un artefacto concreto desde una implementación de interfaz invocada dinámicamente. Hizo falta un ajuste de firma forzado por el codegen (ver D-A arriba); la pregunta de serialización de fondo queda respondida, la contingencia de D-A no se dispara.
- [x] ¿`dagger init` acepta `engineVersion v0.21.8` textualmente? **Resuelto (WU2, tarea 2.3): sí**, aceptado textualmente con un CLI instalado que ya coincidía. No hizo falta subir ningún pin.
- [ ] ¿Cuál es el tope correcto de lectura del manifiesto? Se propone 1 MiB, a confirmar contra el fixture de amplificación por alias en la porción 4; el número es una constante de test, no un elemento del contrato.

---

*Nota de desviación:* este diseño excede el presupuesto de 800 palabras del skill (~3.400 palabras). Causa: es una revisión que supera a la anterior y debe preservar cinco decisiones existentes, eliminar una contradicción verificada y diseñar nueve decisiones nuevas que abarcan un esquema de manifiesto, una tubería de validación, resolución de proveedores, un algoritmo de grafos, un planificador, un mecanismo de interpolación crítico para la seguridad y una migración de punto de entrada con una restricción dura de secuenciación, más una matriz de amenazas que pasó de `N/A` a aplicable. El contenido se comprimió en tablas; se priorizó la completitud sobre el presupuesto de palabras.

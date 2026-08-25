# Propuesta: API pública versionada de módulo + Modelo de composición + Orquestación declarativa de workflows

> **Nota de revisión — REVISIÓN 2, REEMPLAZA la propuesta fusionada de SPEC-002 + SPEC-003.** Tras la revisión de `design.md` por parte del usuario, el alcance se expandió por tercera vez. Cambian tres cosas: el preset nombrado `go-service` se elimina por completo, el tipo público `Pipeline` se renombra, y una capa declarativa de orquestación de workflows/DAG pasa a ser trabajo dentro del alcance de este cambio. El alcance, el tamaño, la carga de revisión y el riesgo de corrección vuelven a crecer como consecuencia directa y aceptada.
>
> Versión de referencia: `proposal.md` (inglés) es la fuente de verdad ante cualquier discrepancia.

**Principio rector (actualizado):** "Las capacidades se componen en workflows; los workflows forman grafos de ejecución. El SDK entrega capacidades, nunca pipelines nombrados."

Se retira la formulación previa — "las capacidades se componen en pipelines" —. Mantenía la palabra `Pipeline` en el centro del modelo mental, y eso es exactamente lo que permitió que un preset nombrado sobreviviera a la última ronda de diseño.

## Intención

Dos problemas, un contrato.

**Problema 1 — el original (sin cambios).** La superficie pública de Shipwright es una única interfaz `Pipeline` monolítica, documentada en `docs/API.md` y `docs/ARCHITECTURE.md` como el punto de extensión canónico, con implementaciones nombradas por stack completo (`go-service`, `infra`). Eso escala de forma multiplicativa: N herramientas de build × M destinos de despliegue × K runtimes obligan a N·M·K pipelines nombrados, cada uno escrito, probado, versionado y documentado por separado, y ningún consumidor puede armar una combinación no anticipada.

**Problema 2 — detectado en la revisión de diseño, y motivo de esta revisión.** El `design.md` aprobado declara el principio "el SDK entrega capacidades, no pipelines" y luego se contradice en la capa de migración:

| Ubicación | Texto | Por qué es un defecto |
|---|---|---|
| `design.md:123` (Flujo de datos) | `main.go --pipeline go-service ──► pipelines.Registry (capability-set presets)` | Reintroduce un registro de *presets nombrados* — exactamente el antipatrón que este cambio existe para eliminar |
| `design.md` (Secuencia de migración) | "`pipelines.Registry` registra factories de conjuntos de capacidades, manteniendo intacta la UX de CLI `--pipeline go-service`, así que no hay migración a nivel de release" | Justifica el preset por compatibilidad |
| `tasks.md:74` (WU6 / tarea 6.1) | "las factories de conjuntos de capacidades preservan la UX de CLI `--pipeline go-service`, sin migración a nivel de release" | Codifica el preset como prueba de aceptación |

`go-service` es un preset de pipeline nombrado. Preservarlo bajo un argumento de compatibilidad de UX de CLI no es defendible aquí, porque **el repositorio tiene cero consumidores externos** — un hecho ya establecido para este cambio. No hay nada con lo que mantener compatibilidad. El argumento de compatibilidad sostenía el antipatrón, no un requisito real.

**Por qué la capa DAG ahora.** Eliminar el preset quita el único mecanismo con el que un usuario podía decir "ejecuta estos pasos en este orden". La composición mediante llamadas Go encadenadas lo reemplaza solo para consumidores Go. Un manifiesto declarativo es lo que hace la composición utilizable desde cualquier lenguaje y revisable como datos — y construirlo después de congelar el contrato significa congelar un contrato que nunca conoció a su consumidor real.

**Cómo se ve el éxito:** un consumidor declara un workflow en YAML — pasos que nombran una *capacidad* (el contrato) y un *proveedor* (la implementación), conectados por aristas de dependencia explícitas en un grafo — y Shipwright lo ejecuta, sin que el motor sepa qué es "Maven" o "Tomcat", y sin que exista ningún preset nombrado en ninguna parte.

## Alcance

### Dentro del alcance

Los entregables 1–8 se mantienen sin cambios respecto de la Revisión 1. Los entregables 9–11 son nuevos.

| # | Entregable | Estado |
|---|---|---|
| 1 | Contrato de API pública que fija cuatro propiedades: **versionada**, **componible**, **consumible entre lenguajes**, **representable por el sistema de tipos público de Dagger** | Sin cambios |
| 2 | Descomposición en capacidades como propiedad del contrato: capacidades pequeñas y ortogonales (Build, Deploy, Run, Test, Artifact) | Sin cambios |
| 3 | La decisión del mecanismo de composición en sí, aplicada en código | Sin cambios |
| 4 | **Reemplazo forzado** de `internal/pipelines/pipeline.go` e `internal/interfaces/interfaces.go`; **eliminación** de `internal/pipelines/common/interfaces.go` | Sin cambios |
| 5 | Migración de `go-service` y del wiring de DI/plugins al nuevo contrato | **Modificado por D3** |
| 6 | Política de versionado con **estabilidad desde el primer release** | Sin cambios |
| 7 | Wiring de módulo Dagger suficiente para generar y verificar **bindings de SDK en un segundo lenguaje** | Sin cambios |
| 8 | Corrección documental mínima: `docs/API.md` / `docs/ARCHITECTURE.md` dejan de presentar `Pipeline` como canónico | Sin cambios |
| 9 | **Eliminación del preset nombrado `go-service`**: sin ruta de CLI `--pipeline go-service`, sin registro de presets de conjuntos de capacidades, sin identidad de agrupación que nombre "el pipeline de Go" | **NUEVO (D3)** |
| 10 | **Renombrado del tipo público de composición `Pipeline`** a un nombre que no invite regresiones de presets nombrados | **NUEVO (D4)** |
| 11 | **Capa declarativa de workflows**: esquema de manifiesto YAML + parser/validador de DAG (detección de ciclos, resolución de dependencias, orden topológico) + capa de resolución de proveedores (capacidad→proveedor, incluidas referencias externas `module:`) + motor de ejecución conectado al contrato de composición | **NUEVO (D5)** |

**Restricción obligatoria (texto normativo en inglés en `proposal.md`; traducción de referencia) — vincula a todo elemento del contrato público, incluidas la capa de workflows y cualquier proveedor externo que resuelva:**

> Restricción del sistema de tipos de Dagger: todo contrato público de composición definido por este cambio DEBE ser representable por el sistema de tipos de módulos de Dagger y consumible a través de bindings de SDK generados entre lenguajes. Los mecanismos de implementación específicos de un lenguaje, incluidos los genéricos de Go, NO DEBEN ser requeridos como parte del contrato público.

### Fuera del alcance (no-objetivos)

Los ejemplos ilustrativos de la capa DAG nombran proveedores (`maven`, `docker`, `tomcat`, `cargo`, `kubernetes`) para **demostrar la forma del esquema**. No son compromisos de entregar esos adaptadores.

| No-objetivo | Límite |
|---|---|
| Adaptadores concretos de capacidad más allá del mínimo necesario para demostrar el motor DAG — sin librería de adaptadores de grado productivo para Java/Gradle/Maven/Ant/Tomcat/K8s/SSH | Cambio posterior (límite sin cambios, ahora reafirmado explícitamente frente a los ejemplos YAML) |
| Motor de políticas completo: integración remota de policy-as-code, UI de flujos de aprobación, sistemas de notificación | Posterior. Solo se entregan `forbidCycles`, `requireVersion` y `forbidPlaintext`, y se entregan **aplicadas**, no documentadas |
| Integración con sistemas de CI (disparadores de GitHub Actions / GitLab CI) | Nunca en este cambio. Este es un motor de ejecución local/programático, no un servicio de CI alojado |
| Un servicio de registro de paquetes/módulos para proveedores externos estilo `module:` | Posterior. La resolución asume proveedores locales o vendorizados, o un mecanismo existente — sin nuevo servicio de registro |
| Productivización completa del módulo: publicación en registro, wiring de CI para `dagger call`, migrar `main.go` fuera de `flag` | Cambio posterior |
| Reescritura completa de `docs/API.md` / `docs/ARCHITECTURE.md` | Seguimiento inmediato (la corrección mínima sí está dentro del alcance) |
| Garantías de compatibilidad para paquetes Go internos no públicos | Nunca — la garantía cubre únicamente el contrato público |

## Decisiones

### D1 — Interfaces `Pipeline` heredadas: reemplazo forzado *(sin cambios)*

| Interfaz | Disposición | Consecuencia |
|---|---|---|
| `internal/pipelines/common/interfaces.go` | **Eliminada** | Código muerto, sin consumidores |
| `internal/pipelines/pipeline.go` | **Retirada / reemplazada** | `go-service` se descompone en capacidades independientes |
| `internal/interfaces/interfaces.go` | **Retirada / reemplazada** | Migran el contenedor de DI, el registro de pasos, el gestor de hooks y la capa de plugins |

### D2 — "Versionada" significa contrato estable desde el primer release *(sin cambios)*

| Elemento | Posición |
|---|---|
| Garantía | Garantía de compatibilidad hacia atrás estilo SemVer **vigente desde el primer release**. Sin válvula de escape v0 |
| Cambio rompedor | Incremento explícito de versión mayor más una nota de migración escrita |
| Identidad de versión | Marcador legible por máquina en el límite del contrato |
| Separación | Versión de contrato / SemVer del binario CLI / pin de motor de `dagger.json` DEBEN mantenerse como tres ejes distintos |
| Alcance | Solo el contrato público. Los paquetes internos no llevan garantía |

*Contrapartida declarada con honestidad (sin cambios, no suavizar):* hoy hay cero consumidores externos, así que garantizamos una forma que nadie ha usado en producción, lo que limita la velocidad de iteración. Se acepta porque un contrato que declara "puede romperse en cualquier momento" no puede ser adoptado por otro equipo ni desde otro lenguaje, y porque la ausencia de política es lo que produjo tres interfaces `Pipeline` divergentes. **Mitigación:** mantener deliberadamente mínima la superficie garantizada.

### D3 — NUEVO: el preset nombrado `go-service` se elimina, no se migra

| Ítem | Posición |
|---|---|
| Flag de CLI `--pipeline go-service` | **Eliminado.** No se preserva por compatibilidad |
| `pipelines.Registry` como registro de presets de conjuntos de capacidades | **Eliminado.** No se entrega ningún registro de presets |
| Salida de la descomposición de `go-service` | Implementaciones de capacidad independientes y componibles por separado, **sin identidad de agrupación** |
| Regla de nombrado | Nada puede nombrar "el pipeline de Go" como una cosa. Las implementaciones nombran lo que *hacen* (compilar Go, probar Go, publicar una imagen de contenedor), nunca un paquete de stack |

Los nombres exactos de implementación son competencia de la fase de diseño. Solo ilustrativos: `GoBuilder`, `GoTester`, `DockerArtifactor`.

*Por qué falla el argumento de compatibilidad:* es real solo cuando existe un consumidor. La ausencia de consumidores externos ya está establecida para este cambio, así que el flag no protege nada y le cuesta al cambio su principio declarado. Esto reemplaza la posición de la Revisión 1 según la cual `go-service` simplemente "migra" — migrar un preset preserva el preset.

### D4 — NUEVO: renombrar el tipo público de composición `Pipeline`

| Ítem | Posición |
|---|---|
| Tipo actual | La estructura `Pipeline` del diseño **ya es** un resultado genérico de composición, construido con llamadas explícitas `.WithBuild().WithTest()...` — no es un preset nombrado |
| Decisión | Renombrarlo igualmente (p. ej. `Plan`, o `Compose`; la fase de diseño elige el nombre exacto) |
| Justificación | **Preventiva, no correctiva.** La palabra "Pipeline" en la superficie pública invita regresiones tipo `GoPipeline` / `JavaPipeline` / `RustPipeline`. La última ronda de diseño demuestra que esa atracción es real, no hipotética |

Esta distinción se planteó explícitamente — el tipo no es defectuoso en sí — y aun así se eligió el renombrado. Queda registrado aquí para que las fases de spec y diseño no lo reabran como reporte de defecto.

### D5 — NUEVO: manifiesto declarativo de workflow + motor de ejecución DAG

**La regla arquitectónica central, de primer orden:**

> **`capability` es el contrato. `uses` / `provider` es la implementación.**

Un paso declara `capability: build` (satisfaciendo la interfaz `Builder` que este cambio ya define) y `uses: {provider: maven, version: "1"}` (que resuelve a un `Builder` concreto). El motor de Shipwright nunca necesita saber qué es "Maven": solo verifica que el proveedor resuelto implemente la capacidad declarada. La sustituibilidad es el mecanismo ya decidido de Dagger Interfaces (D-A en `design.md`), y la restricción del sistema de tipos de Dagger se extiende sin cambios a proveedores de terceros referenciados como `module: github.com/acme/custom-builder, version: v3.2.1`.

**Forma del manifiesto (ilustrativa y no normativa — los nombres exactos de campo son decisión de spec/diseño):**

| Sección | Propósito |
|---|---|
| `apiVersion` / `kind` / `metadata` | Identidad versionada del documento (`shipwright.dev/v1`, `kind: Workflow`), más nombre / descripción / etiquetas |
| `spec.source` | Proveedor de fuente enchufable (repositorio, referencia, referencia a secreto de autenticación) |
| `spec.variables` | Valores nombrados e interpolables (estilo `${{ variables.x }}`) |
| `spec.secrets` | Referencias nombradas a secretos resueltas desde entorno u orígenes externos. **Referenciados por nombre, nunca incrustados en texto plano** — enlaza directamente con el retipado ya decidido a `*dagger.Secret` |
| `spec.steps[]` | `id`, `capability` (contrato), `uses` (proveedor + versión fijada, o referencia externa `module:`), `needs[]` (**aristas explícitas del DAG, no una lista lineal implícita**), `when` (ejecución condicional, p. ej. por rama), `with` (configuración específica del proveedor bajo su propio esquema), `outputs` (resultados nombrados que otros pasos referencian por interpolación acotada a `needs`, p. ej. `${{ build.artifact }}`) |
| `spec.execution` | Concurrencia `maxParallel`, estrategia de fallo rápido, timeout, sobreescrituras de reintento por paso |
| `spec.environments` | Entornos nombrados con puertas de aprobación (revisores requeridos) — p. ej. un entorno `production` que condiciona un paso `deploy` a la aprobación del equipo de plataforma |
| `spec.policies` | Reglas de cumplimiento a nivel de workflow: `secrets.forbidPlaintext`, `providers.requireVersion`, `dependencies.forbidCycles`, `artifacts.immutable`. **Reglas aplicadas, no documentación** |

`needs[]` es lo que convierte esto en un grafo y no en una lista: pruebas unitarias en paralelo y un escaneo de vulnerabilidades pueden depender ambos de `build` y alimentar ambos a `artifact` (fan-out / fan-in).

**Verificación de consistencia con las decisiones existentes:** el manifiesto ilustrativo usa `capability: test` tanto para un paso de pruebas unitarias como para un escaneo de vulnerabilidades, diferenciándose solo por el proveedor. Eso coincide con el mapeo ya decidido donde `Test` / `Lint` / `Vuln` se convierten en tres implementaciones independientes de `Tester`. No se introduce ninguna capacidad nueva: las cinco siguen siendo cinco.

## Capacidades

### Capacidades nuevas

- `public-module-api`: el contrato público, versionado y multilenguaje del módulo — sus cuatro propiedades, la descomposición en capacidades (Build, Deploy, Run, Test, Artifact), la restricción del sistema de tipos de Dagger y la política de versionado/estabilidad.
- `composition-model`: el mecanismo programático de composición, cómo se combinan las capacidades, el tipo de composición renombrado (D4), el retiro de las interfaces `Pipeline` heredadas y la eliminación de los presets nombrados (D3).
- `workflow-manifest`: **NUEVA.** El contrato declarativo — identidad y versionado del documento, la separación `capability` frente a `uses`, las reglas de referencia de variables y secretos, el fijado de versión de proveedores, y las reglas de validación que un manifiesto debe satisfacer (incluida la aciclicidad).
- `workflow-execution`: **NUEVA.** El motor — resolución de proveedores (capacidad→implementación, local y `module:` externo), orden topológico, concurrencia, estrategia de fallo, timeouts y reintentos, ejecución condicional, interpolación de salidas, puertas de aprobación y aplicación de políticas.

Dividir la capa de workflows en contrato (`workflow-manifest`) y mecanismo (`workflow-execution`) refleja la separación `public-module-api` / `composition-model` ya utilizada, de modo que la revisión pueda verificar el esquema declarado con independencia del motor que lo ejecuta.

### Capacidades modificadas

- Ninguna. `product-identity` es la única spec de dominio existente y no se ve afectada.

## Enfoque

1. Especificar las cuatro propiedades y la ortogonalidad de capacidades como requisitos verificables *(sin cambios)*.
2. La fase de diseño elige el mecanismo de composición bajo la restricción textual de Dagger *(sin cambios — ya se eligieron Dagger Interfaces/Objects)*.
3. Especificar la política de versionado estable desde el primer día (D2) y la ruta de reemplazo (D1) *(sin cambios)*.
4. **Renombrar el tipo de composición (D4)** en el contrato de Capa 1 antes de que nada dependa del nombre anterior.
5. **Descomponer `go-service` en implementaciones de capacidad independientes sin identidad de agrupación, y eliminar la ruta y el registro de presets (D3)** — no construir una factory que lo reconstituya.
6. **Especificar el esquema del manifiesto como contrato de documento versionado**, con la separación `capability`/`uses` como su invariante central.
7. **Construir el parser y validador de DAG** — detección de ciclos, resolución de dependencias, orden topológico — con las tres políticas entregadas aplicadas por pruebas que fallan, no por comentarios.
8. **Construir la resolución de proveedores y el motor de ejecución** sobre el contrato de capacidades ya diseñado; el motor resuelve un proveedor y solo comprueba que satisface la capacidad declarada.
9. Cablear el módulo Dagger y verificar el contrato desde un segundo lenguaje *(sin cambios)*.
10. Migrar el contenedor de DI y la capa de plugins; eliminar las interfaces muertas; regenerar mocks *(sin cambios)*.
11. Aplicar la corrección documental mínima *(sin cambios)*.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `openspec/specs/{public-module-api,composition-model,workflow-manifest,workflow-execution}/` | Nuevo | Los cuatro contratos (se crean al archivar) |
| `dagger.json`, `.dagger/` | Nuevo | Wiring de módulo + generación de bindings en segundo lenguaje |
| Esquema de manifiesto + parser/validador | **Nuevo** | Parseo de DAG, detección de ciclos, orden topológico, aplicación de políticas |
| Resolución de proveedores + motor de ejecución | **Nuevo** | Búsqueda capacidad→proveedor incl. referencias externas `module:`; ejecución de grafo, concurrencia, reintentos, puertas |
| `internal/pipelines/common/interfaces.go` | **Eliminado** | Código muerto |
| `internal/pipelines/pipeline.go` | **Eliminado/Reemplazado** | Superado por el contrato de capacidades |
| `internal/pipelines/registry.go`, `options.go` | **Eliminado/Reemplazado** | Registro de presets eliminado (D3), no re-tipado |
| `main.go` | Modificado | Ruta `--pipeline go-service` **eliminada**; el punto de entrada apunta a un manifiesto de workflow |
| `internal/interfaces/interfaces.go` | Modificado/Reemplazado | Se retira la forma `Pipeline`; se re-tipan `Container`/`StepRegistry`/`HookManager` |
| `internal/pipelines/go-service/*` | **Descompuesto** | Pasa a ser implementaciones de capacidad independientes sin identidad de paquete |
| `internal/app/{container,pipeline_executor,step_registry,hook_manager}.go` | Modificado | **Mayor radio de impacto heredado** — el wiring de DI está tipado contra la interfaz retirada |
| `internal/plugins/interfaces.go` | Modificado | `PluginContext.GetPipeline()` devuelve un tipo retirado |
| `mocks/**`, `internal/*/mocks.go` | Regenerados | La salida de `go.uber.org/mock` sigue a las interfaces retiradas |
| `docs/API.md`, `docs/ARCHITECTURE.md` | Modificado (mínimo) | Dejan de presentar `Pipeline` como canónico |

**Nota de compatibilidad con Dagger:** `dagger.io/dagger` v0.21.8 se mantiene como dependencia cliente. `dagger.json` agrega un pin de motor que debe seguir siendo compatible con ella, y la generación de bindings agrega una dependencia del CLI de Dagger al toolchain de desarrollo/CI. Verificar ambas antes del merge.

## Riesgos

**Este cambio es ahora materialmente mayor que la ya riesgosa Revisión 1.** Declarado con claridad, igual que en la fusión de alcance previa: la carga de revisión, el conteo de líneas modificadas y el riesgo de corrección vuelven a aumentar.

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| **El plan de 8 unidades de trabajo / ~3.800–4.800 líneas queda subestimado.** Tres entregables nuevos (eliminación del preset, renombrado, capa DAG completa) se suman encima | **Alta** | La fase de tareas DEBE re-pronosticar desde cero y muy probablemente necesitará **más** slices de PR encadenados que 8, no menos. La estrategia de cadena apilada hacia main ya está elegida; la secuencia crece |
| **Tres expansiones de alcance en un mismo cambio** (fusión contrato+mecanismo, luego eliminación de preset + renombrado, luego capa DAG) | **Alta** | Decisión aceptada del usuario. Cuatro archivos de spec separados mantienen las piezas revisables de forma independiente; la capa DAG se secuencia estrictamente después de que el contrato se estabilice |
| **Inyección por interpolación de cadenas / filtración de secretos** en la resolución de `${{ variables.x }}` y `${{ secrets.x }}` | **Alta — relevante para seguridad** | Los secretos DEBEN resolverse como manejadores `*dagger.Secret`, **nunca sustituidos como texto plano**. `forbidPlaintext` es una política aplicada con una prueba que falla. La interpolación no debe permitir evaluación arbitraria de expresiones. La fase de diseño es dueña explícita del límite de resolución |
| **Corrección de la detección de ciclos del DAG.** Un ciclo no detectado es un bloqueo o una ejecución parcial silenciosa; un falso positivo rechaza grafos válidos de fan-in | **Alta** | `forbidCycles` se aplica, no se asume. Pruebas RED primero, cubriendo auto-aristas, pares mutuos, ciclos largos, fan-in en diamante (válido) y componentes desconectados |
| **Colisión de espacios de versión:** `uses.version` a nivel de manifiesto frente al eje ya decidido `ContractVersion` / `COMPATIBILITY.md` | **Alta** | **Sin resolver — la fase de diseño DEBE reconciliarlo:** ¿son el mismo espacio de versión u ortogonales? La Revisión 1 ya declaró tres ejes distintos; el versionado de proveedores es un candidato a cuarto eje y debe nombrarse como tal o integrarse explícitamente |
| El radio de migración rompe en runtime el contenedor de DI, la capa de plugins o las capacidades descompuestas | **Alta** | TDD estricto (`tdd: true`); `go test -race ./...` en verde por slice; se mantiene el umbral de 90% de cobertura |
| Eliminar `--pipeline go-service` deja sin punto de entrada funcional hasta que llegue la ruta de manifiesto | Media | Secuenciar el punto de entrada de manifiesto antes de o junto con la eliminación del preset; ningún slice puede mergear con el CLI incapaz de ejecutar nada |
| Expansión de alcance dentro de la capa DAG (motor de políticas completo, librería de adaptadores, integración con CI) | Media | La tabla de no-objetivos anterior es el límite. Solo se entregan tres políticas, y solo los adaptadores suficientes para demostrar el motor |
| La resolución de proveedores externos `module:` no tiene registro contra el cual resolver | Media | No-objetivo explícito. La resolución asume proveedores locales o vendorizados, o un mecanismo existente |
| El mecanismo elegido resulta no ser representable en el sistema de tipos de Dagger | Media | La restricción textual es criterio de aceptación vinculante; validar contra bindings generados, no contra documentación |
| La estabilidad desde el primer día fija una forma no probada — ahora incluido un esquema de manifiesto completamente nuevo | Media | Mantener mínima la superficie garantizada; el manifiesto lleva su propio `apiVersion` para que el esquema pueda evolucionar de forma independiente del contrato Go |
| La prueba de bindings entre lenguajes agrega costo de toolchain/CI no presupuestado | Media | Aceptado como elevación del listón de aceptación; basta una invocación local documentada de `dagger` (el wiring de CI queda fuera de alcance) |
| **Colisión terminológica:** "capacidad" ahora significa el contrato Build/Test/…, un dominio de spec de OpenSpec, *y* un campo de paso del manifiesto | Media | La fase de spec DEBE desambiguar en prosa en el primer uso dentro de cada documento; no confiar en el contexto |
| El pin de motor de `dagger.json` entra en conflicto con el cliente v0.21.8 fijado | Baja | Verificar compatibilidad del pin antes del merge |
| Deriva documental: `docs/*` siguen enseñando una interfaz eliminada | **Alta** | La corrección mínima está dentro del alcance y es criterio de éxito; la reescritura completa es un seguimiento explícito |

## Plan de rollback

No es puramente aditivo — este cambio elimina abstracciones vivas y una ruta de CLI.

- **PRs encadenados (lo esperado):** revertir los slices en **orden inverso al de merge**. Revertir el slice de migración sin el de eliminación deja el árbol sin compilar.
- **PR único:** `git revert -m 1 <sha>`.
- **Nuevo en esta revisión:** revertir la eliminación del preset restaura `--pipeline go-service`, así que todo revert que cruce ese slice debe revertir también el punto de entrada de manifiesto, o el CLI queda sin ninguna de las dos rutas. Tratar la eliminación del preset y el punto de entrada de manifiesto como un único límite de rollback.
- No hay migración de estado, datos, archivos de configuración ni releases, y no hay consumidores externos a notificar — el rollback es solo de código. La capa de manifiesto es terreno nuevo, así que no existen archivos YAML previos que migrar.
- Verificación tras cualquier revert: `go build -o shipwright .` y `go test -race ./...` en verde.

## Dependencias

- Pin de versión de motor en `dagger.json` compatible con `dagger.io/dagger` v0.21.8.
- CLI de Dagger en el toolchain de desarrollo/CI para la generación de bindings.
- Un toolchain de SDK en un segundo lenguaje (TypeScript o Python) para verificar los bindings generados.
- Un parser de YAML para la capa de manifiesto (lo elige la fase de diseño; preferir una dependencia ya vendorizada).

## Criterios de éxito

Criterios heredados de la Revisión 1:

- [ ] El contrato declara las cuatro propiedades como requisitos verificables, y la restricción del sistema de tipos de Dagger aparece textualmente y vincula a cada elemento público
- [ ] Ningún elemento del contrato público requiere genéricos de Go ni otro mecanismo específico de lenguaje
- [ ] Las capacidades (Build, Deploy, Run, Test, Artifact) son ortogonales e independientemente significativas
- [ ] El mecanismo de composición está decidido, documentado con su justificación e implementado
- [ ] La política de versionado declara una garantía vigente desde el primer release, su alcance y la regla de cambios rompedores
- [ ] `internal/pipelines/common/interfaces.go` está eliminado; `pipeline.go` e `internal/interfaces/interfaces.go` ya no definen un punto de extensión `Pipeline`
- [ ] El contenedor de DI, la capa de plugins y todos los mocks generados compilan y pasan contra el nuevo contrato
- [ ] Existen bindings de SDK de Dagger generados en TypeScript o Python y se demuestra su consumo
- [ ] `docs/API.md` y `docs/ARCHITECTURE.md` ya no presentan `Pipeline` como la superficie pública canónica
- [ ] `go build -o shipwright .` y `go test -race ./...` en verde; cobertura ≥ 90%

Nuevos en la Revisión 2:

- [ ] **No existe ningún preset de pipeline nombrado en ninguna parte**: sin flag `--pipeline go-service`, sin registro de presets, y sin tipo o identificador que nombre un paquete de stack. Una búsqueda del nombre del preset solo encuentra notas históricas
- [ ] `go-service` se ha convertido en implementaciones de capacidad independientes, cada una componible por separado y utilizable sin sus antiguas hermanas
- [ ] El tipo público de composición ya no contiene la palabra "Pipeline"
- [ ] Un manifiesto de workflow declara pasos por `capability` + `uses`, y el motor resuelve proveedores **sin conocimiento alguno de la herramienta concreta nombrada**
- [ ] `needs[]` produce ejecución real de grafo con fan-out/fan-in, no orden lineal — demostrado por una prueba con dos pasos paralelos que comparten una dependencia y un dependiente
- [ ] **`forbidCycles`, `requireVersion` y `forbidPlaintext` están aplicadas** — cada una tiene una prueba que falla cuando se viola la regla
- [ ] Los secretos referenciados desde un manifiesto se resuelven como manejadores `*dagger.Secret`; ninguna ruta de código sustituye un secreto en una cadena de texto plano
- [ ] La relación entre `uses.version` y `ContractVersion` está explícitamente resuelta y documentada en `COMPATIBILITY.md`
- [ ] Un entorno con puerta de aprobación bloquea su paso dependiente hasta que se registre la aprobación

---

*Nota de desviación:* esta propuesta supera el presupuesto de 450 palabras de la skill, igual que `design.md` y `tasks.md` antes. Causa: es una revisión que reemplaza a la anterior y debe registrar un defecto verificado con evidencia, tres decisiones nuevas, la forma completa del esquema de manifiesto y un límite de no-objetivos ampliado, manteniéndose como contrato autocontenido para las fases de spec y diseño. El contenido se comprime en tablas; se priorizó la completitud sobre el presupuesto de palabras.

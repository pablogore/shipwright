# Propuesta: Actualización de Runtime/Toolchain Gestionada por el Proveedor (Go Primero)

> Versión en español de `proposal.md`. Ante cualquier conflicto, la versión en
> inglés es la fuente de verdad.

## Intención

Las versiones de toolchain se desincronizan silenciosamente entre las fuentes
declarativas de un repositorio. Prueba en vivo aquí: `go.mod`, `go.work` y CI
fijan `go 1.26.7`, mientras que `providers/go/gobuilder.go` aún fija
`defaultGoVersion = "1.25.5"`. Nada lo detecta ni lo corrige. Se busca que cada
proveedor de lenguaje sea responsable de descubrir, actualizar y verificar su
propia versión de toolchain, sin lógica de mutación específica de ecosistema en
el motor central.

## Alcance

### Dentro del Alcance

- Dos tipos de capacidad de un solo método: `runtime-inspect` (reporte de deriva
  de solo lectura) y `runtime-upgrade` (descubrir → mutar → validar),
  proyectados en Capa 1 (`pkg/shipwright`) y Capa 2 (`.dagger`).
- Entradas en la lista permitida del manifiesto, pares de registro de proveedor
  y despacho en el motor para ambos tipos.
- Proveedor Go: directivas `go`/`toolchain` de `go.mod`, `go.work`, anclajes
  declarativos de imagen de build/runtime, normalización con `go mod tidy`,
  conciencia de workspaces multi-módulo y validación post-mutación.
- Guiado por descubrimiento: mutar solo las ubicaciones que existen realmente.
  Fallar de forma cerrada ante fuentes de versión ausentes, ambiguas o en
  conflicto; nunca adivinar.
- La salida de la mutación es un directorio de workspace devuelto más un reporte
  estructurado. Sin red, sin git, sin push.

### Fuera del Alcance

- Proveedores de Java/Rust/Python (solo Go en esta primera entrega).
- Un bot genérico estilo Renovate/Dependabot; actualizaciones de dependencias de
  aplicación (`go.sum`).
- Acoplamiento de Maven/Gradle/Cargo/Poetry/npm en el núcleo.
- Creación de ramas y PR en el SCM (ver D2); disparo programado o por webhook.
- Mutación de constantes arbitrarias de código fuente o de YAML de workflows CI.

## Decisiones

**D1 — Compuerta manual: disparo invocado por un operador, sin primitiva de
bloqueo en el motor.** `engine/execute.go` no contiene lógica de bloqueo por
diseño explícito, y `workflow-execution` exige que las compuertas de aprobación
sigan siendo metadatos declarados. Construir una compuerta real es trabajo
central que sienta precedente y compite con el entregable real. "Manual, no
automático" se cumple al no incluir ningún disparador programado ni webhook.
Consecuencia: `workflow-execution` **no** requiere un delta MODIFIED.
Contrapartida: no hay aplicación dentro del motor; la seguridad proviene de D2.

**D2 — Adaptador SCM/PR: totalmente fuera de alcance, diferido a un cambio
posterior.** Hoy no existe código de SCM; construirlo junto al proveedor Go no
cabe en el presupuesto de revisión. "Nunca hacer push a una rama protegida" se
satisface *estructuralmente*: la capacidad no tiene ninguna ruta de código capaz
de alcanzar un remoto. Un cambio posterior consumirá el directorio y el reporte
devueltos. Contrapartida: v1 se detiene antes de crear el PR.

## Capacidades

### Capacidades Nuevas

- `runtime-toolchain`: inspección de solo lectura de la deriva de toolchain y
  actualización, gestionada por el proveedor, de los metadatos declarativos de
  toolchain.

### Capacidades Modificadas

- `workflow-manifest`: la lista permitida de `capability` ya no es exactamente
  cinco (spec.md:42–46).
- `public-module-api`: la superficie exportada ya no es exactamente cinco
  interfaces de capacidad (spec.md:28).
- `workflow-execution`: **ninguna** — el requisito de compuertas de aprobación se
  preserva textualmente según D1.

## Enfoque

Preservar el invariante del repositorio de una capacidad por método: dos
interfaces, no un único `RuntimeManager` de cuatro métodos. Esto convierte la
separación entre solo lectura y mutación en una frontera *a nivel de tipos*: un
manifiesto que solo declara `runtime-inspect` no puede alcanzar código de
mutación. El escalonamiento plan/apply/verify vive dentro del proveedor,
manteniendo el núcleo puramente orquestador. `sdd-design` define las firmas
finales.

## Áreas Afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `pkg/shipwright/capabilities.go` | Modificado | +2 interfaces y tipos de resultado |
| `.dagger/capabilities.go` | Modificado | Proyección de Capa 2 |
| `internal/workflow/providers/registry.go`, `register.go` | Modificado | 2 pares Register/Resolve |
| `internal/workflow/manifest/schema.go`, `validate.go` | Modificado | Lista permitida 5 → 7 |
| `internal/workflow/engine/execute.go` | Modificado | 2 casos de despacho, sin lógica de bloqueo |
| `providers/go/` | Nuevo | Inspector y actualizador de toolchain de Go |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| Se excede el presupuesto de 400 líneas en un solo PR | Alta | `sdd-tasks` debe pronosticarlo; si es Alta, recomendar una cadena de 2 entregas (primero inspect, luego upgrade) |
| El conteo de capacidades 5 → 7 toca 5 archivos centrales | Media | Solo aditivo; no se modifica comportamiento existente; ambos tipos siguen el patrón de un solo método |
| Fuentes de versión ambiguas o en conflicto provocan una mutación incorrecta | Media | Fallar de forma cerrada: reportar y abortar, nunca adivinar ni mutar parcialmente |
| `go mod tidy` altera `go.sum` más allá de la intención de toolchain | Media | Tidy se ejecuta solo tras mutar las directivas; el diff se reporta |
| El directorio mutado se consume sin revisión | Baja | No existe ruta de escritura remota en v1 (D2) |
| Compatibilidad con el SDK de Dagger o pasos de pipeline | Baja | No se introducen tipos core nuevos de Dagger; ambas interfaces usan el `*dagger.Directory` existente |

## Plan de Reversión

Revertir el commit. El cambio es aditivo: ninguna capacidad, manifiesto o ruta
del motor existente cambia de comportamiento, y quitar las dos entradas de la
lista permitida restaura el conjunto cerrado previo de cinco valores. No hay
estado persistido ni migración involucrada.

## Dependencias

- `golang.org/x/mod/modfile` para el parseo de `go.mod` / `go.work`.
- `internal/daggerpin/pin.go` como precedente existente de comparación de
  versiones.

## Criterios de Éxito

- [ ] `runtime-inspect` sobre este repositorio reporta la deriva real entre
      `1.26.7` y `1.25.5` sin mutar nada.
- [ ] `runtime-upgrade` actualiza `go`/`toolchain` en `go.mod` y en `go.work` en
      todos los módulos del workspace.
- [ ] Fuentes de versión ambiguas o en conflicto abortan con un error explícito y
      sin mutación parcial.
- [ ] El motor no incorpora código de bloqueo/aprobación (D1) ni dependencia de
      SCM/git (D2).
- [ ] `go test -race ./...` pasa, cobertura ≥ 90 %, `golangci-lint run` limpio.

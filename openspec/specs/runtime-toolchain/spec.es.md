# Especificación: Runtime Toolchain

## Propósito

Define dos tipos de capacidad — `runtime-inspect` (solo lectura) y
`runtime-upgrade` (mutable) — que permiten a un provider de lenguaje
descubrir, reportar y, cuando se solicite, corregir el desfase en sus
propios metadatos declarativos de versión de toolchain (por ejemplo,
las directivas `go.mod`/`go.work` de Go), sin lógica específica de
ecosistema en el núcleo de Shipwright. Ambas son interfaces de un solo
método, coherentes con el invariante "una capacidad, un método" de
`public-module-api`: un manifiesto que declara solo `runtime-inspect`
no tiene ninguna ruta de código capaz de mutar.

## Requisitos

### Requisito: Inspección de Desfase de Solo Lectura

La capacidad `runtime-inspect` DEBE aceptar un `*dagger.Directory` de
workspace y devolver un reporte estructurado de desfase que contenga:
la(s) versión(es) descubierta(s) en cada ubicación declarativa, la
versión objetivo/esperada configurada (si existe), y un estado
explícito de conflicto/ambigüedad. `runtime-inspect` NO DEBE mutar el
directorio de entrada, escribir ningún archivo, ni causar ningún
efecto secundario más allá del reporte devuelto.

#### Escenario: La inspección produce cero mutación

- DADO un directorio de workspace con versiones de toolchain
  desfasadas entre ubicaciones
- CUANDO `runtime-inspect` se ejecuta sobre él
- ENTONCES ningún contenido de archivo del workspace difiere de la
  entrada
- Y no se produce ninguna salida más allá del reporte estructurado

#### Escenario: Las fuentes ambiguas se reportan, nunca se adivinan

- DADO un workspace donde dos ubicaciones declarativas discrepan sin
  precedencia resoluble
- CUANDO `runtime-inspect` se ejecuta
- ENTONCES el reporte marca el estado de conflicto explícitamente,
  nombrando ambas fuentes y versiones
- Y no se infiere ninguna versión "ganadora"

#### Escenario: La ubicación declarativa ausente se omite, no se fabrica

- DADO un workspace sin archivo `go.work`
- CUANDO `runtime-inspect` se ejecuta
- ENTONCES el reporte no contiene ninguna entrada `go.work`
- Y no se fabrica ningún valor predeterminado o asumido para ella

### Requisito: Actualización Guiada por Descubrimiento, Propiedad del Provider

La capacidad `runtime-upgrade` DEBE aceptar un `*dagger.Directory` de
workspace y una versión objetivo, y devolver un `*dagger.Directory`
mutado más un reporte estructurado. DEBE mutar únicamente las
ubicaciones declarativas que realmente existen en el workspace de
entrada (guiado por descubrimiento) y DEBE validar el resultado antes
de devolverlo.

#### Escenario: Solo se mutan las ubicaciones existentes

- DADO un workspace con `go.mod` pero sin `go.work`
- CUANDO `runtime-upgrade` se ejecuta con una versión objetivo
- ENTONCES se actualizan las directivas `go`/`toolchain` de `go.mod`
- Y no se crea ningún archivo `go.work`

#### Escenario: Las fuentes ambiguas abortan con cero mutación

- DADO un workspace donde las fuentes declarativas entran en conflicto
  sin precedencia resoluble
- CUANDO `runtime-upgrade` se ejecuta
- ENTONCES devuelve un error identificando el conflicto
- Y ningún archivo del workspace fue mutado — el error devuelto no
  lleva ningún directorio, por lo que quien invoca no recibe ninguna
  salida que consumir

#### Escenario: La ubicación declarada ausente se omite

- DADO un workspace sin archivo de pin de CI
- CUANDO `runtime-upgrade` se ejecuta
- ENTONCES el reporte registra esa ubicación como ausente
- Y no se fabrica ningún archivo en esa ubicación

#### Escenario: El fallo de validación posterior a la mutación no se devuelve silenciosamente

- DADO una mutación que dejaría al workspace incapaz de pasar su
  propia validación posterior a la mutación (por ejemplo, `go build
  ./...` falla)
- CUANDO `runtime-upgrade` se ejecuta
- ENTONCES devuelve un error y un reporte que describe el fallo de
  validación
- Y NO DEBE devolver un directorio presentado como actualizado
  exitosamente

### Requisito: Sin Efectos de Red, Git o SCM

Ni `runtime-inspect` ni `runtime-upgrade` DEBEN realizar ninguna
llamada de red, operación de git, ni efecto secundario de SCM/PR
(creación de rama, push, creación de pull request). Su única
interacción con el exterior es el `*dagger.Directory` de entrada y el
`*dagger.Directory`/reporte devuelto.

#### Escenario: Ninguna ruta de código alcanza operaciones de red o SCM

- DADAS las implementaciones de `runtime-inspect` y `runtime-upgrade`
- CUANDO se inspeccionan sus grafos de llamadas
- ENTONCES ninguna alcanza un cliente HTTP, un comando git, ni una
  llamada a una API de SCM/PR

### Requisito: Consistencia de Workspace Multi-Módulo

Cuando un archivo `go.work` abarca múltiples módulos `go.mod`,
`runtime-upgrade` DEBE reportar resultados por módulo y NO DEBE dejar
al workspace en un estado parcialmente mutado y sin reportar: si algún
módulo falla la validación posterior a la mutación, el resultado
global de la operación DEBE reflejar ese fallo.

#### Escenario: El workspace multi-módulo se actualiza consistentemente

- DADO un `go.work` que referencia tres módulos `go.mod`, todos con la
  misma versión desfasada
- CUANDO `runtime-upgrade` se ejecuta con una versión objetivo
- ENTONCES se actualizan las directivas de los tres módulos
- Y el reporte lista un resultado por módulo para los tres

#### Escenario: El fallo de validación de un módulo hace fallar toda la operación

- DADO un `go.work` que referencia dos módulos, donde uno falla la
  validación posterior a la mutación
- CUANDO `runtime-upgrade` se ejecuta
- ENTONCES el resultado global es un fallo
- Y el reporte nombra qué módulo falló y cuál tuvo éxito
- Y el directorio devuelto no se presenta como un resultado limpio y
  completamente actualizado

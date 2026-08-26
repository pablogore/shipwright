# Especificación: Modelo de Composición

## Propósito

Define el mecanismo de composición para combinar las capacidades Build,
Deploy, Run, Test y Artifact en un resultado de composición, establece el
retiro de las interfaces `Pipeline` heredadas que hoy actúan como contrato
público/de inyección de dependencias de Shipwright, elimina el preset
nombrado de conjunto de capacidades `go-service` y renombra el tipo público
de resultado de composición para alejarlo de `Pipeline`. Los requisitos
restringen propiedades y comportamiento del mecanismo elegido; no
seleccionan por sí mismos entre Dagger Interfaces/Objects, interfaces
concretas de Go o genéricos de uso exclusivamente interno — esa decisión
corresponde a la fase de diseño y DEBE satisfacer cada requisito siguiente
sin importar qué mecanismo se elija.

**Nota terminológica:** aquí "capability" (capacidad) se refiere al tipo de
contrato Go/Dagger, nunca al nombre de este dominio de OpenSpec ni al campo
`capability` de un paso de manifiesto (ver `workflow-manifest`). "Tipo de
composición" a continuación es un marcador de posición para el tipo público
renombrado (p. ej. `Plan`/`Compose`); la fase de diseño elige el nombre
exacto y esta especificación se satisface con cualquier nombre que no sea,
ni contenga, "Pipeline".

## Requisitos

### Requisito: El Mecanismo de Composición Satisface la Restricción de Dagger

Cualquiera que sea el mecanismo que la fase de diseño seleccione para
componer capacidades, DEBE ser representable por el sistema de tipos de
módulos de Dagger y consumible mediante bindings de SDK generados entre
lenguajes, y NO DEBE requerir un parámetro de tipo genérico de Go como parte
del contrato público.

#### Escenario: Resultado de composición consumible entre lenguajes

- DADO un pipeline ensamblado a partir de dos o más capacidades mediante el
  mecanismo de composición elegido
- CUANDO se ejercita el binding correspondiente del SDK de Dagger en un
  segundo lenguaje (TypeScript o Python)
- ENTONCES el pipeline compuesto se ejecuta con un comportamiento
  equivalente al de la composición del lado Go

#### Escenario: El mecanismo de composición no depende de genéricos de Go

- DADA la función o tipo de composición exportado
- CUANDO se inspecciona su firma pública
- ENTONCES no declara ningún parámetro de tipo genérico de Go

### Requisito: Las Capacidades se Componen Sin un Tipo Pipeline Central

El mecanismo de composición DEBE permitir ensamblar cualquier subconjunto de
las capacidades Build, Deploy, Run, Test y Artifact en un resultado de
pipeline sin requerir un struct `Pipeline` predeclarado y nombrado por cada
combinación.

#### Escenario: Combinación novedosa de capacidades se compone sin código nuevo

- DADO que las capacidades Build y Test ya existen y están implementadas de
  forma independiente
- CUANDO un consumidor las compone en un pipeline por primera vez
- ENTONCES no es necesario redactar ningún tipo `Pipeline` nuevo con nombre
  ni una entrada de registro para esa combinación

### Requisito: Retiro de las Interfaces Pipeline Heredadas

`internal/pipelines/pipeline.go` e `internal/interfaces/interfaces.go` NO
DEBEN volver a definir un tipo `Pipeline` que actúe como punto de extensión
del SDK. Ambos DEBEN ser reemplazados por el nuevo contrato de composición;
la coexistencia con la forma anterior como superficie pública paralela
queda prohibida.

#### Escenario: Las interfaces Pipeline heredadas ya no definen un punto de extensión

- DADOS `internal/pipelines/pipeline.go` e `internal/interfaces/interfaces.go`
  tras la migración
- CUANDO se inspeccionan sus declaraciones exportadas
- ENTONCES ninguno define una interfaz `Pipeline` usada como punto de
  extensión público

### Requisito: Eliminación de la Interfaz Heredada Muerta

`internal/pipelines/common/interfaces.go` DEBE eliminarse, dado que es
código muerto confirmado sin consumidores.

#### Escenario: Archivo de interfaz muerta eliminado

- DADO el repositorio migrado
- CUANDO se busca `internal/pipelines/common/interfaces.go`
- ENTONCES el archivo no existe y ningún import lo referencia

### Requisito: Implementación y Cableado Existentes Migrados

La implementación de pipeline `go-service` DEBE descomponerse en
implementaciones de capacidad independientes (según "Ningún Preset Nombrado
de Conjunto de Capacidades" más abajo, no migrarse detrás de una bandera de
compatibilidad), y el contenedor de inyección de dependencias
(`internal/app/container.go`, `pipeline_executor.go`, `step_registry.go`,
`hook_manager.go`) y la capa de plugins (`internal/plugins/interfaces.go`)
DEBEN migrarse al nuevo contrato de composición. Todos los mocks generados
DEBEN regenerarse para coincidir.
(Anteriormente: se describía como "migrado", lo que implicaba que se
preservaba la compatibilidad de `--pipeline go-service`. Corregido — ver
"Ningún Preset Nombrado de Conjunto de Capacidades": la bandera del CLI y
el registro de presets se eliminan, no se preservan.)

#### Escenario: go-service descompuesto y en verde, sin bandera de compatibilidad preservada

- DADO `internal/pipelines/go-service` tras la descomposición
- CUANDO se ejecutan `go build -o shipwright .` y `go test -race ./...`
- ENTONCES ambos se completan con éxito sin ninguna referencia a las
  interfaces `Pipeline` retiradas Y no existe ninguna bandera CLI
  `--pipeline go-service` en `main.go`

#### Escenario: El contenedor de DI y la capa de plugins compilan contra el nuevo contrato

- DADOS `internal/app/container.go` e `internal/plugins/interfaces.go` tras
  la migración
- CUANDO se compila el paquete
- ENTONCES `PluginContext.GetPipeline()` (o su reemplazo) devuelve un tipo
  definido por el nuevo contrato de composición, no la `interfaces.Pipeline`
  retirada

#### Escenario: Mocks regenerados

- DADAS las interfaces del nuevo contrato de composición
- CUANDO se regeneran los mocks basados en `go.uber.org/mock`
- ENTONCES los archivos de mocks generados compilan y satisfacen el nuevo
  contrato sin ninguna referencia a tipos retirados

### Requisito: Ningún Preset Nombrado de Conjunto de Capacidades

Shipwright NO DEBE entregar un preset nombrado de conjunto de capacidades
— sin bandera CLI `--pipeline go-service`, sin registro de fábricas de
conjuntos de capacidades indexadas por nombre de stack, y sin ningún tipo o
identificador que nombre un conjunto de capacidades como una unidad
reutilizable única. La descomposición de `go-service` DEBE producir
implementaciones de capacidad independientes y componibles por separado,
sin identidad de agrupación; los nombres de implementación DEBEN describir
lo que hacen (p. ej. un compilador de Go, un empaquetador de Docker de
artefactos), nunca un conjunto de stack.

#### Escenario: No existe registro de presets

- DADO el repositorio migrado
- CUANDO se busca un registro de presets de conjuntos de capacidades
  indexado por nombre de stack (p. ej. `go-service`)
- ENTONCES no existe tal registro, mapa de fábricas ni bandera CLI

#### Escenario: Las capacidades de go-service son utilizables de forma independiente entre sí

- DADAS las implementaciones de capacidad producidas al descomponer
  `go-service`
- CUANDO se compone una de ellas (p. ej. el compilador de Go) sin ninguno
  de sus antiguos compañeros
- ENTONCES se compone y ejecuta con éxito sin ninguna referencia a una
  identidad de agrupación `go-service`

### Requisito: Tipo de Resultado de Composición Renombrado Fuera de "Pipeline"

El tipo público de resultado de composición (construido mediante llamadas
explícitas `.With*()`, según "Las Capacidades se Componen Sin un Tipo
Pipeline Central" más arriba) NO DEBE llamarse `Pipeline`, y su nombre
exportado NO DEBE contener la subcadena "Pipeline". El renombramiento es
preventivo, no una corrección de un comportamiento defectuoso: el mecanismo
de composición y su patrón de construcción `.With*()` permanecen sin
cambios.

#### Escenario: El nombre del tipo de composición exportado excluye "Pipeline"

- DADO el tipo público de resultado de composición exportado por el
  contrato
- CUANDO se inspecciona su identificador Go exportado
- ENTONCES no es igual a, ni contiene como subcadena (sin distinguir
  mayúsculas/minúsculas), la palabra "Pipeline"

## Fuera de Alcance

Elegir la forma exacta del mecanismo de composición (Dagger
Interfaces/Objects frente a interfaces concretas de Go frente a genéricos de
uso exclusivamente interno) es una decisión de la fase de diseño; esta
especificación restringe sus propiedades, no su implementación. La
productivización completa (publicación en registro, cableado de `dagger call`
en CI) queda diferida.

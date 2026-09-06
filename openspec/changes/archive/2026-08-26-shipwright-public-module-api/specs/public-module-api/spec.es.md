# Especificación: API Pública de Módulo

## Propósito

Define las cuatro propiedades obligatorias del contrato de módulo público y
versionado de Shipwright — "las capacidades se componen en workflows; los
workflows forman grafos de ejecución; el SDK entrega capacidades, nunca
pipelines nombrados". Establece la garantía de compatibilidad, la
descomposición en capacidades y la restricción del sistema de tipos de
Dagger que vincula esta especificación y las especificaciones
complementarias `composition-model`, `workflow-manifest` y
`workflow-execution`.

**Nota terminológica:** en este documento, "capability" (capacidad) se
refiere al tipo de contrato Go/Dagger (`Builder`, `Tester`, `Artifactor`,
`Deployer`, `Runner`) — distinto del nombre de este dominio de OpenSpec y
del campo `capability` de un paso de manifiesto definido en
`workflow-manifest`.

## Requisitos

### Requisito: Contrato Versionado y Estable Desde el Primer Release

El contrato público DEBE llevar una garantía de compatibilidad hacia atrás de
estilo SemVer, documentada y vigente desde su primer release. Un cambio
incompatible DEBE requerir un incremento explícito de versión mayor y una
nota de migración por escrito. El contrato DEBE exponer un marcador de
versión legible por máquina en su propio límite, distinto de la versión de
release del binario CLI (goreleaser/CHANGELOG) y del pin de versión de motor
en `dagger.json`. La garantía DEBE aplicarse únicamente a la superficie del
contrato público; los paquetes internos no llevan garantía de compatibilidad.
La superficie garantizada es exactamente las cinco interfaces de capacidad y
la superficie `Shipwright`/tipo de composición ya enumerada para este
contrato (ver `composition-model`); el `uses.version` de un manifiesto de
workflow (la versión propia de un proveedor) es un eje de versión separado
y sin solapamiento, y NO está cubierto por esta garantía, incluso cuando el
proveedor se referencia mediante `module:`.

#### Escenario: La versión del proveedor es independiente de la versión del contrato

- DADO un paso de manifiesto de workflow que declara `uses: {provider:
  maven, version: "2"}`
- CUANDO se inspecciona el `ContractVersion` del contrato
- ENTONCES ambos valores son independientes — cambiar `uses.version` nunca
  implica un cambio de `ContractVersion`, y la garantía de compatibilidad
  no se extiende al proveedor referenciado

#### Escenario: Marcador de versión presente y distinto

- DADO el contrato de módulo público
- CUANDO se inspecciona su marcador de versión
- ENTONCES se resuelve de forma independiente de la etiqueta SemVer del
  binario CLI y del pin de motor en `dagger.json`

#### Escenario: Cambio incompatible requiere incremento mayor y nota de migración

- DADO un cambio al contrato público que altera la firma de una capacidad
  exportada existente
- CUANDO se clasifica el cambio
- ENTONCES DEBE incrementar la versión mayor del contrato y DEBE incluir una
  nota de migración por escrito

#### Escenario: Cambio en paquete interno no lleva garantía

- DADO un cambio en un paquete no exportado, exclusivamente interno
- CUANDO se clasifica el cambio contra la política de compatibilidad
- ENTONCES queda exento del requisito de incremento de versión

### Requisito: Capacidades Componibles y Ortogonales

El contrato público DEBE descomponerse en capacidades pequeñas y ortogonales
— Build, Deploy, Run, Test, Artifact. Cada capacidad DEBE ser
independientemente significativa y reemplazable, y NO DEBE requerir
conocimiento de ninguna capacidad hermana. Ningún tipo nombrado DEBE ser la
abstracción central del SDK; un resultado de composición (ver
`composition-model`) o un workflow declarativo (ver
`workflow-manifest`/`workflow-execution`) PUEDE existir únicamente como el
*resultado* de componer capacidades, nunca como un tipo predeclarado por
combinación.
(Anteriormente: prohibía específicamente a `Pipeline` como abstracción
central; se generalizó porque el propio tipo de composición está siendo
renombrado para alejarse de ese nombre — ver `composition-model`.)

#### Escenario: Capacidad compuesta sin referencia concreta a un resultado de composición

- DADO que un consumidor importa el paquete público de capacidades
- CUANDO compone una capacidad Build con una capacidad Deploy sin importar
  ni referenciar ningún struct de resultado de composición predeclarado y
  nuevo para esa combinación
- ENTONCES la composición se completa con éxito y produce un resultado de
  composición válido

#### Escenario: Capacidad utilizable de forma aislada

- DADO que solo se importa la capacidad Build
- CUANDO se invoca sin ninguna capacidad Deploy, Run, Test o Artifact
  presente
- ENTONCES se ejecuta exitosamente sin dependencia de compilación ni de
  ejecución hacia las demás capacidades

### Requisito: Consumible Entre Lenguajes

El contrato público DEBE ser consumible mediante bindings de SDK de Dagger
generados en al menos un lenguaje distinto de Go (TypeScript o Python), y
dicho consumo DEBE quedar demostrado.

#### Escenario: Bindings de un segundo lenguaje generados y ejercitados

- DADO el cableado del módulo Dagger para el contrato público
- CUANDO se generan bindings de SDK de Dagger en TypeScript o Python
- ENTONCES los bindings generados exponen el mismo conjunto de capacidades
  que el contrato en Go
- Y un script de demostración en ese lenguaje compone al menos dos
  capacidades y se ejecuta con éxito (verificación de nivel de integración,
  condicionada a la disponibilidad real del CLI `dagger`, según la
  convención `testing.Short()`)

### Requisito: Representable por el Sistema de Tipos Público de Dagger

> Restricción del sistema de tipos de Dagger: Cualquier contrato de
> composición público definido por este cambio DEBE ser representable por el
> sistema de tipos de módulos de Dagger y consumible mediante bindings de SDK
> generados entre lenguajes. Los mecanismos de implementación específicos de
> un lenguaje, incluidos los genéricos de Go, NO DEBEN ser requeridos como
> parte del contrato público.

#### Escenario: Ninguna firma pública requiere un parámetro genérico de Go

- DADO cada función y tipo exportado en el paquete público de capacidades
- CUANDO se inspeccionan sus firmas
- ENTONCES ninguna declara un parámetro de tipo genérico de Go como parte de
  su contrato público

#### Escenario: Los tipos públicos se corresponden con el sistema de tipos de Dagger

- DADO un tipo de capacidad exportado por el contrato público
- CUANDO se inspecciona su compatibilidad con el módulo Dagger (por ejemplo,
  incrustación de `DaggerObject` o anotación de interfaz)
- ENTONCES cumple los requisitos de Dagger para exponerse como argumento o
  valor de retorno de una Function

## Fuera de Alcance

Los adaptadores concretos de capacidad (build Java/Gradle/Maven/Ant; deploy
Tomcat/Kubernetes/SSH) no se especifican aquí — se difieren a cambios
posteriores. La reescritura completa de `docs/API.md`/`docs/ARCHITECTURE.md`
se difiere; este contrato solo exige la corrección mínima (dejar de
presentar `Pipeline` como canónico).

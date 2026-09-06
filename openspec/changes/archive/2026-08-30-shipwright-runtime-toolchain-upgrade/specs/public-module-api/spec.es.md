# Delta para Public Module API

## Requisitos MODIFICADOS

### Requisito: Contrato Versionado Estable Desde El Primer Lanzamiento

El contrato público DEBE llevar una garantía de compatibilidad hacia
atrás documentada, estilo SemVer, vigente desde su primer lanzamiento.
Un cambio incompatible DEBE requerir un incremento explícito de
versión mayor y una nota de migración escrita. El contrato DEBE
exponer un marcador de versión legible por máquina en su propio
límite, distinto de la versión de lanzamiento del binario CLI
(goreleaser/CHANGELOG) y del pin de versión de engine en
`dagger.json`. La garantía DEBE aplicarse únicamente a la superficie
del contrato público; los paquetes internos no llevan garantía de
compatibilidad. La superficie garantizada es exactamente las siete
interfaces de capacidad (Build, Test, Artifact, Deploy, Run,
RuntimeInspector, RuntimeUpgrader) y la superficie
`Shipwright`/tipo-de-composición ya enumerada para este contrato (ver
`composition-model`); el `uses.version` de un manifiesto de workflow
(la versión propia de un provider) es un eje de versión separado y no
superpuesto, y NO está cubierto por esta garantía, incluso cuando el
provider se referencia mediante `module:`.
(Anteriormente: la superficie garantizada era exactamente las cinco
interfaces de capacidad — Build, Test, Artifact, Deploy, Run — sin
tipos de capacidad runtime-toolchain.)

#### Escenario: La versión del provider es independiente de la versión del contrato

- DADO un step de manifiesto de workflow que declara `uses: {provider:
  maven, version: "2"}`
- CUANDO se inspecciona el `ContractVersion` del contrato
- ENTONCES ambos valores son independientes — cambiar `uses.version`
  nunca implica un cambio de `ContractVersion`, y la garantía de
  compatibilidad no se extiende al provider referenciado

#### Escenario: El marcador de versión está presente y es distinto

- DADO el contrato del módulo público
- CUANDO se inspecciona su marcador de versión
- ENTONCES se resuelve de forma independiente al tag SemVer del
  binario CLI y al pin de engine de `dagger.json`

#### Escenario: Un cambio incompatible requiere incremento mayor y nota de migración

- DADO un cambio al contrato público que altera la firma de una
  capacidad exportada existente
- CUANDO el cambio se clasifica
- ENTONCES DEBE incrementar la versión mayor del contrato y DEBE
  incluir una nota de migración escrita

#### Escenario: Un cambio en paquete interno no lleva garantía

- DADO un cambio a un paquete no exportado, solo interno
- CUANDO el cambio se clasifica contra la política de compatibilidad
- ENTONCES está exento del requisito de incremento de versión

#### Escenario: La superficie garantizada crece de cinco a siete

- DADA la superficie garantizada del contrato público, inspeccionada
  tras agregar `RuntimeInspector` y `RuntimeUpgrader`
- CUANDO se verifica el conteo de interfaces
- ENTONCES es exactamente siete — las cinco originales más las dos
  interfaces de runtime-toolchain
- Y un cambio incompatible a cualquiera de las dos nuevas interfaces
  está sujeto al mismo requisito de incremento mayor y nota de
  migración que las cinco originales

### Requisito: Capacidades Componibles y Ortogonales

El contrato público DEBE descomponerse en capacidades pequeñas y
ortogonales — Build, Deploy, Run, Test, Artifact, RuntimeInspector,
RuntimeUpgrader. Cada capacidad DEBE ser independientemente
significativa y reemplazable, y NO DEBE requerir conocimiento de
ninguna capacidad hermana. Ningún tipo nombrado único DEBE ser la
abstracción central del SDK; un resultado de composición (ver
`composition-model`) o un workflow declarativo (ver
`workflow-manifest`/`workflow-execution`) PUEDE existir únicamente como
el *resultado* de componer capacidades, nunca como un tipo
pre-declarado por combinación.
(Anteriormente: el conjunto enumerado era exactamente las cinco
capacidades — Build, Deploy, Run, Test, Artifact — sin tipos de
capacidad runtime-toolchain.)

#### Escenario: Capacidad compuesta sin una referencia concreta a un resultado de composición

- DADO que un consumidor importa el paquete de capacidades público
- CUANDO compone una capacidad Build con una capacidad Deploy sin
  importar ni referenciar ninguna struct de resultado de composición
  concreta y pre-declarada nueva para esa combinación
- ENTONCES la composición tiene éxito y produce un resultado de
  composición válido

#### Escenario: Capacidad usable de forma aislada

- DADO que solo se importa la capacidad Build
- CUANDO se invoca sin ninguna capacidad Deploy, Run, Test o Artifact
  presente
- ENTONCES se ejecuta exitosamente sin ninguna dependencia en tiempo de
  compilación o de ejecución hacia las otras capacidades

#### Escenario: RuntimeInspector y RuntimeUpgrader son usables de forma independiente

- DADO que solo se importa la capacidad RuntimeInspector
- CUANDO se invoca sin RuntimeUpgrader, ni ninguna de Build, Deploy,
  Run, Test o Artifact, presente
- ENTONCES se ejecuta exitosamente sin ninguna dependencia en tiempo de
  compilación o de ejecución hacia ninguna otra capacidad
- Y la misma independencia se cumple para RuntimeUpgrader usada sin
  RuntimeInspector

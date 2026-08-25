# Especificación: Identidad del Producto

> Versión en español. La versión canónica es `spec.md` (inglés); prevalece
> ante cualquier discrepancia.

## Propósito

Define la identidad pública canónica del producto como **Shipwright** — ruta
del módulo Go, nombre del binario, prefijo de variables de entorno, nombre
del archivo de configuración por defecto, nomenclatura de artefactos de
CI/CD y release, documentación, e identidad del repositorio de GitHub —
reemplazando la identidad heredada `syntegrity-dagger`, sin ningún cambio
funcional o de comportamiento en el CLI que envuelve Dagger.

## Requisitos

### Requisito: Ruta del módulo Go e imports

El sistema DEBE declarar su ruta de módulo Go como
`github.com/pablogore/shipwright` en `go.mod`, y cada import interno en Go
DEBE referenciar esa ruta.

#### Escenario: Declaración del módulo

- DADO el `go.mod` del repositorio
- CUANDO se inspecciona
- ENTONCES declara `module github.com/pablogore/shipwright`
- Y ninguna línea referencia `github.com/getsyntegrity/syntegrity-dagger` ni
  `github.com/pablogore/syntegrity-dagger`

#### Escenario: Imports de paquetes internos

- DADO cualquier archivo `.go` que importa un paquete interno
- CUANDO se inspecciona la ruta de import
- ENTONCES está prefijada con `github.com/pablogore/shipwright`

### Requisito: Nombre del binario CLI y salida de ayuda/versión

El sistema DEBE compilar un binario llamado `shipwright`, y su salida de
`--help` y `--version` DEBE presentar únicamente la identidad Shipwright.

#### Escenario: Nombre del artefacto binario

- DADO que se ejecuta el comando de build
- CUANDO se inspecciona el artefacto resultante
- ENTONCES se llama `shipwright`, no `syntegrity-dagger`

#### Escenario: Texto de ayuda y versión

- DADO el binario `shipwright`
- CUANDO se invoca con `--help` o `--version`
- ENTONCES la salida nombra el comando `shipwright` y no contiene ninguna
  cadena `syntegrity-dagger` ni `Syntegrity Dagger`

### Requisito: Nombre de archivo de configuración por defecto

El sistema DEBE cargar la configuración por defecto desde `.shipwright.yml`
y NO DEBE referenciar `.syntegrity-dagger.yml`.

#### Escenario: Búsqueda de configuración por defecto

- DADO que no se suministra una ruta de configuración explícita
- CUANDO el CLI arranca
- ENTONCES busca `.shipwright.yml` en el directorio de trabajo
- Y ninguna ruta de código referencia `.syntegrity-dagger.yml`

### Requisito: Prefijo de variables de entorno

El sistema DEBE leer las anulaciones de configuración únicamente desde
variables con prefijo `SHIPWRIGHT_` (p. ej. `SHIPWRIGHT_TOKEN`,
`SHIPWRIGHT_VERSION`, `SHIPWRIGHT_PIPELINE_COVERAGE`,
`SHIPWRIGHT_PIPELINE_GO_VERSION`, `SHIPWRIGHT_ENVIRONMENT`).

#### Escenario: Se reconoce el nuevo prefijo, no el antiguo

- DADO que `SHIPWRIGHT_TOKEN` está definida en el entorno
- CUANDO el CLI carga la configuración
- ENTONCES el valor se aplica
- Y `SYNTEGRITY_DAGGER_TOKEN` no se lee como respaldo

### Requisito: Referencias de workflows de CI/CD y directorio de la action compuesta

Los workflows de CI/CD DEBEN referenciar el directorio renombrado de la
action compuesta `.github/actions/shipwright/`, y `ci.yml`, `release.yml`,
`dependabot.yml`, `CODEOWNERS` y `rulesets/README.md` DEBEN portar
únicamente la identidad Shipwright.

#### Escenario: Ruta de la action compuesta renombrada

- DADO un paso de workflow que usa la action compuesta
- CUANDO se inspecciona su ruta `uses:`
- ENTONCES referencia `.github/actions/shipwright/`
- Y `.github/actions/syntegrity-dagger/` ya no existe

### Requisito: Nomenclatura de artefactos GoReleaser y URLs de instalación

La configuración de release DEBE producir artefactos, nombres de binario y
plantillas de URL de instalación bajo la identidad `shipwright`.

#### Escenario: Nomenclatura de artefactos de release

- DADO `.goreleaser.yml`
- CUANDO se inspecciona un build de release
- ENTONCES los binarios/archivos producidos usan el nombre `shipwright`
- Y las plantillas de URL de instalación referencian `pablogore/shipwright`

### Requisito: La documentación presenta exclusivamente Shipwright

`README.md` (incluidas las URLs de badges), los archivos bajo `docs/`,
`CHANGELOG.md` y `examples/**` DEBEN presentar el producto exclusivamente
como Shipwright, incluyendo entradas históricas reescritas en el
CHANGELOG.

#### Escenario: README y badges

- DADO `README.md`
- CUANDO se inspecciona
- ENTONCES nombra al producto Shipwright y cualquier URL de badge apunta a
  `pablogore/shipwright`

#### Escenario: CHANGELOG reescrito

- DADO `CHANGELOG.md`
- CUANDO se inspecciona
- ENTONCES cada entrada referencia Shipwright, no Syntegrity Dagger

### Requisito: Identificadores de pruebas y fixtures actualizados

Los archivos de prueba, mocks y fixtures DEBEN usar identificadores
Shipwright donde codifiquen la identidad del producto (rutas de módulo,
nombres de variables de entorno, nombres de archivo de configuración), sin
alterar la intención ni las aserciones de las pruebas.

#### Escenario: Identificadores de fixtures actualizados

- DADO un fixture que referencia el nombre del archivo de configuración o
  el prefijo de entorno
- CUANDO se inspecciona
- ENTONCES usa `.shipwright.yml` / `SHIPWRIGHT_*`
- Y asegura un comportamiento equivalente al previo al renombramiento

### Requisito: Cero cambio funcional (no regresión)

El renombramiento NO DEBE alterar el comportamiento en tiempo de ejecución,
el flujo de control, la disposición de paquetes ni la integración del SDK
`dagger.io/dagger`.

#### Escenario: Paridad de build y pruebas

- DADO el código renombrado
- CUANDO se ejecutan `go build` y `go test -race ./...`
- ENTONCES ambos tienen éxito y la cobertura se mantiene igual o por encima
  del umbral existente

#### Escenario: SDK de Dagger sin tocar

- DADO cualquier archivo que importa `dagger.io/dagger`
- CUANDO se inspecciona
- ENTONCES el import y su uso permanecen sin cambios respecto al estado
  previo al renombramiento

### Requisito: Cero referencias residuales a la identidad antigua

Una búsqueda en todo el repositorio, sin distinción de mayúsculas, del
patrón de identidad del producto `syntegrity[-_ ]?dagger` (que cubre
`syntegrity-dagger`, `syntegrity_dagger`, `SyntegrityDagger`,
`SYNTEGRITY_DAGGER` y `Syntegrity Dagger`) DEBE devolver cero resultados,
salvo las dos exclusiones de identidad de producto documentadas a
continuación. Una búsqueda aparte, más acotada, del nombre desnudo de la
empresa `syntegrity` (sin `dagger`) PUEDE devolver resultados, pero solo si
caen bajo "Referencias a la empresa/org (se conservan)" en Fuera del
alcance.

#### Escenario: Barrido limpio con excepciones documentadas

- DADO el repositorio completamente rebautizado
- CUANDO se ejecuta una búsqueda sin distinción de mayúsculas de
  `syntegrity[-_ ]?dagger`
- ENTONCES las únicas coincidencias son el grep preexistente
  `gitlab.com/syntegrity` del Makefile y el archivo suelto `1export` en la
  raíz

#### Escenario: Identidad de la empresa sin tocar

- DADO el repositorio completamente rebautizado
- CUANDO se ejecuta una búsqueda sin distinción de mayúsculas del término
  desnudo `syntegrity`
- ENTONCES cada coincidencia restante pertenece a la identidad externa real
  de la empresa Syntegrity (la org de GitHub `getsyntegrity`, su
  dependencia `go-kit-logger`, el dominio de correo `getsyntegrity.com`, o
  valores de ejemplo propiedad de la empresa) y ninguna se refiere a este
  producto

### Requisito: Identidad del repositorio de GitHub

El repositorio remoto canónico del producto DEBE ser
`pablogore/shipwright`, alcanzable vía `go get` y las URLs de release.

#### Escenario: Repositorio renombrado como paso operativo

- DADO que el renombramiento a nivel de código está completo
- CUANDO se renombra el repositorio de GitHub de
  `pablogore/syntegrity-dagger` a `pablogore/shipwright` como paso
  operativo (no de código)
- ENTONCES `github.com/pablogore/shipwright` resuelve y coincide con la
  ruta de módulo en `go.mod`

## Fuera del alcance (no requisitos)

- Imports del SDK `dagger.io/dagger` — dependencia legítima de terceros,
  nunca se toca.
- El filtro de cobertura `grep -E "gitlab.com/syntegrity"` del Makefile —
  referencia muerta preexistente, ajena a la identidad de este producto.
- El archivo suelto `1export` en la raíz — volcado de variables de shell
  ajeno al proyecto.
- Alias de retrocompatibilidad o capas de deprecación para la identidad
  antigua — no se introduce ninguno; es un renombramiento limpio.
- **Referencias a la empresa/org (se conservan)** — la identidad externa
  real de la empresa/org Syntegrity, distinta del nombre antiguo de este
  producto, nunca se toca: la dependencia
  `github.com/getsyntegrity/go-kit-logger` (`go.mod`, `go.sum`, y cada
  archivo `.go` que la importa), los ejemplos de código de `eventengine`
  (ajenos) en `AGENTS.md`, el dominio de correo `getsyntegrity.com` y el
  autor Git por defecto `"Syntegrity CI"` en
  `internal/pipelines/shared/{ssh,https}_cloner.go`, la ruta de clave SSH
  por defecto `$HOME/.ssh/syntegrity` en `ssh_cloner.go`, y el namespace de
  registry de ejemplo `ghcr.io/syntegrity` en
  `examples/configs/tenant-svc.yml`.

# Especificación: Manifiesto de Workflow

## Propósito

Define el contrato YAML declarativo de un workflow de Shipwright: identidad
y versionado del documento, la separación entre el `capability` de un paso
(el contrato) y su `uses` (la implementación), las reglas de referencia de
variables y secretos, el fijado de versión de proveedores, y las reglas de
validación a nivel de esquema que un manifiesto debe cumplir — incluyendo
la aciclicidad como una política *declarada* (la aplicación es
responsabilidad de `workflow-execution`, no de este esquema).

**Nota terminológica — "capability" significa tres cosas distintas en esta
base de código; este documento usa solo una de ellas:** (1) el tipo de
contrato Go/Dagger (`Builder`, `Tester`, ...) definido por
`public-module-api`; (2) el nombre de dominio de OpenSpec de este propio
archivo; (3) el campo YAML `capability` de un paso de manifiesto, que
nombra uno de los cinco tipos de contrato de (1). Todo uso simple de
"capability" a continuación se refiere a (3), salvo que se indique lo
contrario.

## Requisitos

### Requisito: Identidad Versionada del Documento

Un manifiesto de workflow DEBE declarar `apiVersion`, `kind: Workflow` y un
`metadata.name`. Un manifiesto al que le falte cualquiera de estos tres
campos DEBE fallar la validación de esquema antes de inspeccionar cualquier
paso.

#### Escenario: Manifiesto sin apiVersion falla la validación

- DADO un documento YAML con `kind: Workflow` pero sin `apiVersion`
- CUANDO se valida contra el esquema del manifiesto
- ENTONCES la validación falla con un error que nombra el campo faltante

#### Escenario: Identidad de manifiesto bien formada valida

- DADO un documento YAML con `apiVersion: shipwright.dev/v1`,
  `kind: Workflow` y `metadata.name: example`
- CUANDO se valida
- ENTONCES la validación de identidad del documento se completa con éxito

### Requisito: `capability` Es el Contrato, `uses` Es la Implementación

Cada paso DEBE declarar exactamente un campo `capability` que nombre una de
las cinco interfaces de capacidad definidas por `public-module-api` (Build,
Test, Artifact, Deploy, Run), y exactamente un campo `uses` que identifique
un proveedor local (`provider` + `version`) o un proveedor externo
(`module` + `version`). Un paso sin `uses`, o que nombre un `capability`
fuera de las cinco interfaces definidas, DEBE fallar la validación de
esquema.

#### Escenario: Paso con capability y uses valida

- DADO un paso `{id: build, capability: build, uses: {provider: maven,
  version: "1"}}`
- CUANDO se valida el manifiesto
- ENTONCES el paso valida correctamente

#### Escenario: Paso sin uses falla la validación

- DADO un paso que declara `capability: build` sin campo `uses`
- CUANDO se valida el manifiesto
- ENTONCES la validación falla, nombrando el campo `uses` faltante

### Requisito: Bordes DAG Explícitos Mediante `needs[]`

Las dependencias de un paso DEBEN declararse explícitamente mediante su
lista `needs[]`. El esquema del manifiesto NO DEBE inferir ningún orden ni
dependencia a partir de la posición de un paso en `spec.steps[]`.

#### Escenario: El orden de declaración no implica dependencia

- DADOS dos pasos declarados consecutivamente en `spec.steps[]` sin
  relación `needs[]` entre ellos
- CUANDO se analiza el manifiesto
- ENTONCES no se infiere ningún borde de dependencia entre ellos a partir
  de su posición en la lista

### Requisito: Secretos Referenciados por Nombre, Nunca Incrustados en Texto Plano

Las entradas de `spec.secrets` DEBEN ser referencias por nombre resueltas
desde una fuente de entorno o externa. El esquema NO DEBE aceptar un valor
de secreto literal en línea bajo `spec.secrets`.

#### Escenario: Valor de secreto en texto plano en línea rechazado

- DADO un manifiesto con `spec.secrets.registry-token.value: "s3cr3t"` (un
  valor literal, no una referencia)
- CUANDO se valida el manifiesto
- ENTONCES la validación falla

#### Escenario: Referencia de secreto nombrado sin texto plano valida

- DADO un manifiesto con `spec.secrets.registry-token: {fromEnv:
  REGISTRY_TOKEN}` y un campo de paso `${{ secrets.registry-token }}`
- CUANDO se valida el manifiesto
- ENTONCES la validación se completa con éxito y no hay ningún valor en
  texto plano en todo el documento

### Requisito: El Espacio de Versiones del Proveedor Es Independiente de `ContractVersion`

`uses.version` (la versión propia de un proveedor) DEBE rastrearse como un
eje de versión distinto del `ContractVersion` del contrato público. El
esquema del manifiesto NO DEBE acoplar la versión de un proveedor a la
garantía de compatibilidad del contrato, y la garantía definida en
`public-module-api` no se extiende a un proveedor referenciado mediante
`uses`/`module`.

#### Escenario: La versión del módulo externo es independiente de ContractVersion

- DADO un paso `uses: {module: "github.com/acme/custom-builder", version:
  "v3.2.1"}`
- CUANDO se valida el manifiesto contra un contrato en un `ContractVersion`
  dado
- ENTONCES la validación no depende de, ni afirma nada sobre, la
  compatibilidad de `v3.2.1` con `ContractVersion`

### Requisito: Las Políticas se Declaran Como Campos de Esquema Estructurados y Aplicables

`spec.policies` DEBE aceptar `secrets.forbidPlaintext`,
`providers.requireVersion`, `dependencies.forbidCycles` y
`artifacts.immutable` como campos estructurados consumibles por un motor de
ejecución. Este esquema declara los campos; no los aplica por sí mismo — la
aplicación es responsabilidad de `workflow-execution`.

#### Escenario: El bloque de políticas se analiza en valores estructurados

- DADO `spec.policies: {dependencies: {forbidCycles: true}, providers:
  {requireVersion: true}}`
- CUANDO se analiza el manifiesto
- ENTONCES ambos valores de política están disponibles como campos
  tipados para un motor consumidor

### Requisito: Los Portones de Aprobación se Declaran Solo Como Metadatos

`spec.environments.<nombre>.approvals` DEBE ser representable como un
objeto estructurado con un campo `required` que contenga una lista de
nombres de revisores (p. ej. `approvals: {required: [platform-team]}`).
El esquema NO DEBE definir ninguna semántica de ejecución o bloqueo para
este campo — son datos, legibles por cualquier llamador o sistema externo.

#### Escenario: Los metadatos de aprobación declarados son consultables, no ejecutables

- DADO un manifiesto que declara `spec.environments.production.approvals:
  {required: [platform-team]}`
- CUANDO se analiza el manifiesto
- ENTONCES la lista `required` está disponible como datos simples para
  cualquier llamador, y el esquema no le adjunta ningún comportamiento de
  bloqueo

### Requisito: Los Tokens de Interpolación Usan una Gramática Fija, Sin Expresiones Arbitrarias

`${{ variables.x }}`, `${{ secrets.x }}` y `${{ steps.<id>.output }}` son
las únicas formas de interpolación admitidas. Cada capability devuelve
exactamente un resultado tipado, por lo que el resultado de un paso no
tiene ningún subcampo con nombre. El esquema DEBE rechazar un token que
contenga una expresión fuera de esta gramática fija (aritmética, llamadas
a función o metacaracteres de shell).

#### Escenario: Token de gramática fija aceptado

- DADO un valor de campo `${{ variables.registry }}`
- CUANDO se valida el manifiesto
- ENTONCES el token se acepta como una referencia de variable

#### Escenario: Token de expresión arbitraria rechazado

- DADO un valor de campo `${{ 1 + 1 }}` o `${{ os.Exec("rm -rf /") }}`
- CUANDO se valida el manifiesto
- ENTONCES la validación falla — el esquema nunca acepta una expresión
  fuera de la gramática fija `variables.`/`secrets.`/`steps.<id>.output`

## Fuera de Alcance

Los adaptadores de proveedor concretos (`maven`, `docker`, `tomcat`, ...)
solo demuestran la forma del esquema, no un compromiso de entregarlos. Un
servicio de registro de paquetes/módulos para referencias `module:` se
difiere — la resolución asume proveedores locales/empaquetados o un
mecanismo ya existente. La semántica de ejecución (recorrido del grafo,
resolución de proveedores, resolución de interpolación, aplicación de
aprobaciones) pertenece a `workflow-execution`, no a este esquema.

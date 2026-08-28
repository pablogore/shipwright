# Delta para Workflow Manifest

## Requisitos MODIFICADOS

### Requisito: `capability` Es El Contrato, `uses` Es La Implementación

Cada step DEBE declarar exactamente un campo `capability` que nombre
una de las siete interfaces de capacidad definidas por
`public-module-api` (Build, Test, Artifact, Deploy, Run,
RuntimeInspector, RuntimeUpgrader), y exactamente un campo `uses` que
identifique un provider local (`provider` + `version`) o un provider
externo (`module` + `version`). Un step sin `uses`, o que nombre una
`capability` fuera de las siete interfaces definidas, DEBE fallar la
validación de esquema.
(Anteriormente: la lista permitida era exactamente de cinco valores —
build/test/artifact/deploy/run — sin tipos de capacidad
runtime-toolchain.)

#### Escenario: Un step con capability y uses valida

- DADO un step `{id: build, capability: build, uses: {provider:
  maven, version: "1"}}`
- CUANDO el manifiesto se valida
- ENTONCES el step valida exitosamente

#### Escenario: Un step sin uses falla la validación

- DADO un step que declara `capability: build` sin campo `uses`
- CUANDO el manifiesto se valida
- ENTONCES la validación falla, nombrando el campo `uses` ausente

#### Escenario: Un step que declara runtime-inspect valida

- DADO un step `{id: check-go, capability: runtime-inspect, uses:
  {provider: go, version: "1"}}`
- CUANDO el manifiesto se valida
- ENTONCES el step valida exitosamente

#### Escenario: Un step que declara runtime-upgrade valida

- DADO un step `{id: bump-go, capability: runtime-upgrade, uses:
  {provider: go, version: "1"}, with: {target: "1.26.7"}}`
- CUANDO el manifiesto se valida
- ENTONCES el step valida exitosamente

#### Escenario: Una capability fuera de la lista de siete valores falla la validación

- DADO un step que declara `capability: deploy-runtime` (que no es
  uno de los siete valores permitidos)
- CUANDO el manifiesto se valida
- ENTONCES la validación falla, nombrando el valor `capability`
  inválido

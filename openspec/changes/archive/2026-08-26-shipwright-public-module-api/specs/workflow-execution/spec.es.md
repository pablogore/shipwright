# Especificación: Ejecución de Workflow

## Propósito

Define el motor DAG que consume un documento `workflow-manifest`:
construcción del grafo de dependencias a partir de `needs[]`, detección de
ciclos, orden de ejecución topológico, resolución capacidad→proveedor,
resolución de interpolación de variables/secretos, y los controles de
ejecución (concurrencia, estrategia de fallo, tiempos de espera,
reintentos, ejecución condicional). Establece explícitamente lo que este
motor **no** hace: no aplica portones de aprobación.

**Nota terminológica:** aquí "capability" (capacidad) se refiere al tipo de
contrato Go/Dagger (p. ej. `Builder`) que nombra el campo `capability` de
un paso de manifiesto — ver `workflow-manifest` y `public-module-api`.
Nunca el nombre de este propio dominio de OpenSpec.

## Requisitos

### Requisito: Grafo de Dependencias Construido a partir de `needs[]`

El motor DEBE construir su grafo de ejecución a partir de la lista
`needs[]` de cada paso, nunca a partir del orden de declaración en
`spec.steps[]`.

#### Escenario: El orden de ejecución sigue a needs, no al orden de declaración

- DADO un manifiesto que declara el paso B antes que el paso A, con
  `needs: [A]` de B
- CUANDO se ejecuta el workflow
- ENTONCES A se completa antes de que B inicie, sin importar el orden de
  declaración

### Requisito: La Aplicación de Aciclicidad del DAG Es Precisa

`policies.dependencies.forbidCycles` DEBE aplicarse: un manifiesto cuyo
grafo `needs[]` contenga un ciclo (autoreferencia, par mutuo o ciclo más
largo) DEBE rechazarse antes de iniciar la ejecución, con un error que
identifique el ciclo. Un grafo válido de convergencia/divergencia en
diamante — dos pasos que dependen de un paso previo y ambos alimentan a un
paso posterior — NO DEBE rechazarse.

#### Escenario: Manifiesto cíclico rechazado

- DADO un manifiesto donde el paso A `needs: [B]` y el paso B `needs: [A]`
- CUANDO se valida el manifiesto
- ENTONCES la validación falla con un error de detección de ciclo y ningún
  paso se ejecuta

#### Escenario: Grafo válido de convergencia en diamante se ejecuta con éxito

- DADOS los pasos `test-unit` y `test-vuln`, ambos con `needs: [build]`, y
  el paso `artifact` con `needs: [test-unit, test-vuln]`
- CUANDO se ejecuta el workflow
- ENTONCES `build` se ejecuta primero, `test-unit`/`test-vuln` se ejecutan
  después de él (y PUEDEN ejecutarse concurrentemente), y `artifact` se
  ejecuta solo después de que ambos completen — no se genera ningún error
  de ciclo

### Requisito: Orden de Ejecución Topológico que Respeta las Dependencias

Un paso NO DEBE iniciar hasta que todos los pasos en su `needs[]` hayan
completado con éxito. Los pasos sin relación de dependencia entre sí
PUEDEN ejecutarse concurrentemente.

#### Escenario: El paso espera a todas sus dependencias declaradas

- DADO el paso C con `needs: [A, B]`
- CUANDO A completa pero B aún no ha completado
- ENTONCES C no ha iniciado

### Requisito: La Resolución Capacidad→Proveedor Solo Verifica la Satisfacción de la Interfaz

Resolver el `uses.provider` o `uses.module` de un paso DEBE verificar
únicamente que la implementación resuelta satisface la interfaz Go/Dagger
del `capability` declarado. El motor NO DEBE contener lógica específica de
proveedor indexada por nombre de proveedor.

#### Escenario: Cambiar de proveedor no requiere ningún cambio en el motor

- DADO un paso `capability: build` resuelto mediante `uses.provider: maven`
- CUANDO `uses.provider` cambia a `gradle` y el nuevo proveedor implementa
  `Builder`
- ENTONCES el workflow resuelve y se ejecuta sin ningún cambio en el
  código del motor

#### Escenario: Un proveedor que no satisface el capability declarado falla de forma cerrada

- DADO que la implementación `uses` resuelta de un paso no implementa la
  interfaz del `capability` declarado
- CUANDO el motor resuelve el paso
- ENTONCES la resolución falla con un error explícito — nunca omite el
  paso silenciosamente

### Requisito: Interpolación de Variables y Salidas de Paso

`${{ variables.x }}` y `${{ steps.<id>.output }}` DEBEN resolverse a sus
valores correspondientes antes de que se ejecute un paso que los
referencie. Cada capability devuelve exactamente un resultado tipado, por
lo que el resultado de un paso no tiene ningún subcampo con nombre — su
tipo lo fija la capability que lo produce.

#### Escenario: El paso posterior recibe la salida de un paso anterior

- DADO que el paso `build` (una capability Builder) produce su único
  resultado tipado, y el paso `deploy` declara `with: {image: "${{
  steps.build.output }}"}`
- CUANDO se ejecuta `deploy`
- ENTONCES recibe el valor real producido por `build`, no el token literal

### Requisito: La Interpolación de Secretos Resuelve a Manejadores Tipados, Nunca en Texto Plano

`${{ secrets.x }}` DEBE resolver de extremo a extremo a un manejador
tipado `*dagger.Secret`. NO DEBE sustituirse como cadena en un `string` de
Go en texto plano en ningún punto de la ruta de resolución. El mecanismo de
interpolación NO DEBE evaluar expresiones arbitrarias.

#### Escenario: El valor del secreto nunca aparece como cadena en texto plano

- DADO un manifiesto que referencia `${{ secrets.registry-token }}`
- CUANDO el motor lo resuelve
- ENTONCES el valor nunca aparece como un tipo `string` de Go en ningún
  punto de la ruta de resolución — solo como un `*dagger.Secret`

#### Escenario: Expresión de interpolación no permitida rechazada en la resolución

- DADO un campo de manifiesto que contiene un token de interpolación fuera
  de la gramática fija `variables.`/`secrets.`/`steps.<id>.output`
- CUANDO el motor intenta resolverlo
- ENTONCES la resolución falla — el motor nunca lo evalúa como una
  expresión

### Requisito: Los Portones de Aprobación Son Metadatos Declarados, No Aplicados por el Motor

El motor NO DEBE implementar lógica de bloqueo, encolado o "esperar
aprobación" para `spec.environments.<nombre>.approvals`. DEBE tratar los
metadatos de aprobación declarados como datos de paso disponibles para
cualquier llamador, y DEBE ejecutar un paso según el orden normal del DAG
sin importar si se ha registrado una aprobación externa.

#### Escenario: El paso de despliegue se ejecuta sin una señal de aprobación externa

- DADO que un manifiesto declara un entorno `production` con aprobaciones
  requeridas para su paso `deploy`, y no se ha registrado ninguna
  aprobación externa
- CUANDO se ejecuta el workflow y se satisfacen los `needs[]` de `deploy`
- ENTONCES `deploy` se ejecuta según el orden normal del DAG — el motor no
  bloquea, encola ni espera un estado de aprobación

#### Escenario: Los metadatos de aprobación se transmiten sin cambios

- DADO el mismo manifiesto
- CUANDO un llamador consulta los metadatos de aprobaciones del entorno
- ENTONCES el motor devuelve los metadatos declarados sin cambios, sin
  interpretarlos ni bloquear en función de ellos

### Requisito: Controles de Ejecución — Concurrencia, Estrategia de Fallo, Tiempo de Espera, Reintento

El motor DEBE respetar `spec.execution.maxParallel`, su estrategia de
fallo (fail-fast o continuar), el `timeout` por paso y los `retries` por
paso.

#### Escenario: maxParallel limita los pasos concurrentes

- DADO `spec.execution.maxParallel: 1` y dos pasos independientes y listos
- CUANDO se ejecuta el workflow
- ENTONCES los dos pasos se ejecutan secuencialmente, nunca de forma
  concurrente

#### Escenario: Fail-fast omite los dependientes posteriores de un paso fallido

- DADO `spec.execution.failFast: true` y el paso A falla
- CUANDO el motor continúa procesando el grafo
- ENTONCES cualquier paso aún no iniciado cuyo `needs[]` incluya a A se
  omite, no se ejecuta

### Requisito: Ejecución Condicional Mediante `when`

Un paso que declara una condición `when` NO DEBE ejecutarse cuando el
valor declarado no esté presente en la lista del predicado
correspondiente. `when` es un mapa de predicado YAML estructurado sobre
las mismas referencias restringidas que la interpolación (por ejemplo,
`when: {branch: [main, develop]}`), evaluado por coincidencia exacta —
nunca una expresión de cadena con operadores.

#### Escenario: El paso con una condición sin coincidencia se omite

- DADO un paso que declara `when: {branch: [main]}` y la rama actual es
  `develop`
- CUANDO se ejecuta el workflow
- ENTONCES el paso se omite porque `develop` no está en la lista declarada

#### Escenario: El paso con una condición coincidente se ejecuta

- DADO un paso que declara `when: {branch: [main, develop]}` y la rama
  actual es `develop`
- CUANDO se ejecuta el workflow
- ENTONCES el paso se ejecuta porque `develop` está en la lista declarada

### Requisito: Políticas de Proveedor/Secreto Declaradas Aplicadas en Tiempo de Validación

`providers.requireVersion` y `secrets.forbidPlaintext` DEBEN aplicarse
antes de iniciar la ejecución, siguiendo el mismo patrón que
`forbidCycles`: un manifiesto que viole cualquiera de las dos DEBE fallar
la validación con un error que nombre la política violada.

#### Escenario: Versión de proveedor faltante rechazada bajo requireVersion

- DADO `spec.policies.providers.requireVersion: true` y el `uses` de un
  paso omite `version`
- CUANDO se valida el manifiesto
- ENTONCES la validación falla, nombrando la versión faltante

### Requisito: Los Proveedores Externos `module:` Satisfacen el Mismo Contrato de Dagger que los Proveedores Internos

Un proveedor referenciado mediante `uses.module` DEBE satisfacer el mismo
contrato de Dagger Interface que un `uses.provider` del propio repositorio;
el motor aplica verificaciones idénticas de resolución y satisfacción de
interfaz a ambos.

#### Escenario: El módulo externo resuelve de forma idéntica a un proveedor local

- DADOS dos pasos con el mismo `capability: build`, uno usando
  `uses.provider: maven` y el otro `uses.module:
  github.com/acme/custom-builder`
- CUANDO ambos se resuelven
- ENTONCES ambos resuelven mediante la misma verificación de satisfacción
  de interfaz, sin ninguna ruta especial para el módulo externo

### Requisito: El Punto de Entrada Basado en Manifiesto Reemplaza la Ruta CLI del Preset en el Mismo Cambio

El punto de entrada de ejecución basado en manifiesto DEBE integrarse a más
tardar junto con la eliminación del preset `go-service` (ver "Ningún
Preset Nombrado de Conjunto de Capacidades" de `composition-model`). El
repositorio NO DEBE alcanzar un estado fusionado en el que el CLI no tenga
ni la bandera del preset ni un punto de entrada funcional basado en
manifiesto.

#### Escenario: El CLI siempre tiene una ruta de ejecución funcional

- DADO el estado del repositorio después de eliminar el preset
  `go-service` y su bandera CLI
- CUANDO se invoca el CLI con un manifiesto de workflow
- ENTONCES el punto de entrada basado en manifiesto lo ejecuta con éxito —
  no existe ningún punto en el historial fusionado donde el CLI no pueda
  ejecutar ninguna de las dos rutas

## Fuera de Alcance

Una integración remota de policy-as-code, una UI de flujo de aprobación, o
un sistema de notificaciones. Integración con sistemas de CI (disparadores
de GitHub Actions / GitLab CI) — este es un motor de ejecución
local/programático. Un servicio de registro de paquetes/módulos para
proveedores `module:`. Adaptadores de proveedor concretos más allá del
mínimo necesario para demostrar el motor.

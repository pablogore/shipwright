# Delta para public-module-api

## Requisitos AGREGADOS (ADDED)

### Requisito: Implementabilidad externa verificada mediante control automatizado

El contrato público (`pkg/shipwright`) DEBE seguir siendo suficiente — junto
con el SDK de Go de Dagger y la biblioteca estándar únicamente — para
implementar cualquier capacidad distribuida como un módulo Go independiente.
Esta propiedad DEBE verificarse mediante un control automatizado y repetible
(por ejemplo, inspeccionar `go list -m all` desde dentro del módulo candidato,
o un control equivalente sobre el grafo de importaciones) que falle cuando una
implementación de capacidad importe cualquier paquete `internal/**`. La
documentación o la revisión manual por sí solas NO DEBEN considerarse
suficientes para satisfacer este requisito. Este requisito es permanente:
rige toda capacidad distribuida después de este cambio, no solo las que este
cambio extrae.

#### Escenario: El módulo independiente compila contra el contrato únicamente

- DADO una implementación de capacidad empaquetada como su propio módulo Go
- CUANDO se ejecuta `go list -m all` desde dentro de ese módulo
- ENTONCES sus únicas dependencias distintas de la biblioteca estándar son el
  módulo propietario de `pkg/shipwright` y `dagger.io/dagger`

#### Escenario: Una importación de paquete interno hace fallar el control automatizado

- DADO una implementación de capacidad que importa cualquier paquete
  `internal/**`
- CUANDO se ejecuta el control automatizado sobre el grafo de importaciones
- ENTONCES el control falla y reporta la importación infractora

#### Escenario: El requisito se aplica a capacidades futuras

- DADO una nueva capacidad agregada después de este cambio
- CUANDO se registra como capacidad distribuida
- ENTONCES el mismo control automatizado DEBE pasar antes de que se
  distribuya

### Requisito: El módulo del proveedor anidado está estructuralmente aislado

Un proveedor de capacidad extraído DEBE vivir en su propio módulo Go (para
este cambio: `providers/go`, ruta de módulo
`github.com/pablogore/shipwright/providers/go`, paquete `golang`) sin
importar nada de `internal/**`. El `go.work` raíz DEBE incluir ese módulo
para que `./...` de la raíz (compilación, pruebas, CI) lo abarque, y NUNCA
DEBE incluir `.dagger`. Una prueba automatizada DEBE fallar si `.dagger` se
agrega alguna vez a `go.work`.

#### Escenario: El grafo de dependencias del módulo del proveedor es mínimo

- DADO `providers/go/go.mod`
- CUANDO se inspeccionan sus requisitos
- ENTONCES solo listan el módulo propietario de `pkg/shipwright`,
  `dagger.io/dagger` y la biblioteca estándar

#### Escenario: La compilación y pruebas de la raíz abarcan el módulo del proveedor

- DADO `go.work` con `use .` y `use ./providers/go`
- CUANDO se ejecutan `go build ./...` y `go test -race ./...` desde la raíz
  del repositorio
- ENTONCES ambos recorren `providers/go`

#### Escenario: El control de aislamiento de `.dagger` falla ante una violación

- DADO `go.work` modificado para incluir `use ./.dagger`
- CUANDO se ejecuta la prueba de control de aislamiento
- ENTONCES esta falla

### Requisito: La extracción preserva el registro, la distribución y el comportamiento

`internal/workflow/providers/register.go` DEBE registrar cada capacidad
extraída bajo su nombre de proveedor previo a la extracción. El `go.mod`
raíz confirmado (committed) DEBE resolver el módulo del proveedor anidado
mediante una etiqueta git publicada con prefijo de ruta, sin ninguna
directiva `replace`, de modo que
`go install github.com/pablogore/shipwright@latest` siga funcionando sin
cambios. Una ejecución de manifiesto de extremo a extremo DEBE comportarse de
forma idéntica antes y después de la extracción. Ningún archivo bajo
`pkg/shipwright/**` DEBE cambiar como parte de una extracción.

#### Escenario: Los nombres de proveedor no cambian tras la extracción

- DADO `RegisterDefaults` después de la extracción de `providers/go`
- CUANDO se inspecciona el registro
- ENTONCES registra las capacidades bajo `"go"`, `"go-test"`,
  `"golangci-lint"`, `"govulncheck"` y `"container"`, de forma idéntica a
  antes de la extracción

#### Escenario: `go install` funciona sin directiva `replace` confirmada

- DADO el `go.mod` raíz confirmado después de la extracción
- CUANDO se inspecciona
- ENTONCES no contiene ninguna directiva `replace`
- Y `go install github.com/pablogore/shipwright@latest` resuelve e instala
  desde un entorno limpio

#### Escenario: La ejecución de manifiesto de extremo a extremo es idéntica en comportamiento

- DADO `examples/workflow/diamond.yaml`
- CUANDO se ejecuta antes y después de la extracción
- ENTONCES ambas ejecuciones resuelven los mismos proveedores y producen
  resultados equivalentes

#### Escenario: El paquete de contrato público permanece intacto por la extracción

- DADO la extracción completada
- CUANDO se inspecciona el `git diff` del cambio
- ENTONCES no toca ningún archivo bajo `pkg/shipwright/`

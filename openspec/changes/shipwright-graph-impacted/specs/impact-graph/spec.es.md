# Especificación: Grafo de Impacto

## Propósito

Define el contrato de `pkg/graph.Impacted` — una función pura, sin
dependencias, de alcance inverso sobre un grafo de nodos opacos. Dado un mapa
de aristas de dependencia y un conjunto de nodos modificados, DEBE devolver
cada nodo que depende transitivamente de un nodo modificado, incluidos los
propios nodos modificados. Este contrato no tiene conocimiento de
`manifest.Step`, providers, ni ningún otro concepto de ejecución de
Shipwright — existe para que un llamador externo (Verimand,
`github.com/getsyntegrity/verimand-platform`, issue #207) pueda importarlo
como un algoritmo ordinario, libre de dependencias.

**Nota de terminología:** "modificado" (changed) aquí significa "que el
llamador ya sabe que cambió" — este contrato no detecta cambios por sí mismo
(sin git, sin filesystem, sin I/O de ningún tipo); solo propaga un conjunto
provisto por el llamador.

## Requisitos

### Requisito: Convención de dirección de dependencia

`edges[X]` DEBE representar el conjunto de nodos de los que `X` depende
directamente — la misma dirección que la convención existente de
`Node.Needs`/grafo de Build en este repositorio. `Impacted` DEBE calcular la
inversa de esta relación internamente; los llamadores NO DEBEN estar
obligados a invertir su mapa de aristas de antemano.

#### Escenario: Un nodo se ve impactado por un cambio en algo de lo que depende

- DADO un mapa de aristas donde `edges["service"] = ["lib"]` (service
  depende de lib)
- CUANDO se llama a `Impacted` con `changed = ["lib"]`
- ENTONCES la salida incluye `"service"`

### Requisito: Auto-inclusión del conjunto modificado

Todo nodo presente en la entrada `changed` DEBE aparecer en la salida,
incluso si no tiene dependientes y aunque esté ausente del conjunto de claves
de `edges`.

#### Escenario: Un nodo modificado sin dependientes sigue en la salida

- DADO un mapa de aristas sin ninguna entrada que dependa de `"leaf"`
- CUANDO se llama a `Impacted` con `changed = ["leaf"]`
- ENTONCES la salida es exactamente `["leaf"]`

#### Escenario: Un nodo modificado desconocido para el mapa sigue en la salida

- DADO un mapa de aristas cuyas claves no incluyen `"ghost"`
- CUANDO se llama a `Impacted` con `changed = ["ghost"]`
- ENTONCES la salida incluye `"ghost"` (está impactado por definición — que
  el grafo no tenga registro de él no lo elimina del conjunto modificado)

### Requisito: Alcance inverso transitivo

`Impacted` DEBE incluir todo nodo que depende de un nodo modificado a
cualquier profundidad, no solo dependientes directos, y DEBE manejar
correctamente fan-out (un nodo modificado con múltiples dependientes
directos) y fan-in (múltiples nodos modificados que comparten un dependiente
transitivo común).

#### Escenario: El dependiente directo se ve impactado

- DADO `edges["a"] = ["b"]`
- CUANDO se llama a `Impacted` con `changed = ["b"]`
- ENTONCES la salida incluye `"a"`

#### Escenario: El dependiente multinivel (transitivo) se ve impactado

- DADO `edges["a"] = ["b"]` y `edges["b"] = ["c"]`
- CUANDO se llama a `Impacted` con `changed = ["c"]`
- ENTONCES la salida incluye `"a"` y `"b"`

#### Escenario: Fan-out — un nodo modificado, múltiples dependientes directos

- DADO `edges["a"] = ["c"]` y `edges["b"] = ["c"]`
- CUANDO se llama a `Impacted` con `changed = ["c"]`
- ENTONCES la salida incluye tanto `"a"` como `"b"`

#### Escenario: Fan-in — múltiples nodos modificados convergen en un dependiente

- DADO `edges["shared"] = ["a", "b"]`
- CUANDO se llama a `Impacted` con `changed = ["a", "b"]`
- ENTONCES la salida incluye `"shared"` exactamente una vez

#### Escenario: Un nodo desconectado no se ve impactado

- DADO `edges["a"] = ["b"]` y un nodo no relacionado `"z"` sin arista hacia o
  desde `"b"`
- CUANDO se llama a `Impacted` con `changed = ["b"]`
- ENTONCES la salida no incluye `"z"`

### Requisito: Tolerancia a entradas malformadas

Una arista colgante — que nombra un nodo ausente del propio conjunto de
claves de `edges` — NO DEBE causar un panic; ese nodo simplemente no tiene
más aristas salientes que recorrer.

#### Escenario: La arista colgante no genera panic

- DADO `edges["a"] = ["missing"]` donde `"missing"` no es en sí una clave de
  `edges`
- CUANDO se llama a `Impacted` con cualquier conjunto `changed`
- ENTONCES la llamada retorna normalmente, sin panic

### Requisito: Seguridad ante ciclos

Un grafo cíclico o malformado NO DEBE causar un bucle infinito ni un panic.
El recorrido DEBE estar protegido por un conjunto de nodos visitados.

#### Escenario: La auto-arista termina

- DADO `edges["a"] = ["a"]`
- CUANDO se llama a `Impacted` con `changed = ["a"]`
- ENTONCES la llamada termina y la salida es `["a"]`

#### Escenario: El ciclo mutuo de dos nodos termina

- DADO `edges["a"] = ["b"]` y `edges["b"] = ["a"]`
- CUANDO se llama a `Impacted` con `changed = ["b"]`
- ENTONCES la llamada termina y la salida incluye tanto `"a"` como `"b"`

#### Escenario: El ciclo más largo termina

- DADO un ciclo `edges["a"] = ["b"]`, `edges["b"] = ["c"]`,
  `edges["c"] = ["a"]`
- CUANDO se llama a `Impacted` con `changed = ["c"]`
- ENTONCES la llamada termina y la salida incluye `"a"`, `"b"` y `"c"`

### Requisito: Colapso de aristas duplicadas

Las aristas duplicadas entre los mismos dos nodos NO DEBEN afectar la
corrección de la salida — colapsan en una única arista lógica.

#### Escenario: La arista duplicada no duplica la salida ni rompe el recorrido

- DADO `edges["a"] = ["b", "b"]`
- CUANDO se llama a `Impacted` con `changed = ["b"]`
- ENTONCES la salida incluye `"a"` exactamente una vez

### Requisito: Salida determinista y ordenada

`Impacted` DEBE devolver un slice ordenado y deduplicado. La misma entrada
DEBE producir siempre la misma salida, independientemente del orden no
determinista de iteración de maps en Go.

#### Escenario: Llamadas repetidas con la misma entrada producen la misma salida

- DADA una entrada fija de mapa de aristas y conjunto `changed`
- CUANDO se llama a `Impacted` múltiples veces con la misma entrada
- ENTONCES cada llamada devuelve el mismo slice, en el mismo orden
  lexicográfico, sin entradas duplicadas

#### Escenario: Grafo vacío y conjunto modificado vacío

- DADO un mapa de aristas vacío
- CUANDO se llama a `Impacted` con un slice `changed` vacío
- ENTONCES la salida es un slice vacío, sin ambigüedad nil-vs-vacío que rompa
  comparaciones de igualdad del llamador contra `[]string{}`

### Requisito: Sin acoplamiento a ejecución ni a manifiesto

La firma exportada de `Impacted` DEBE tomar únicamente identificadores de
nodo como strings opacos (`map[string][]string`, `[]string`) y NO DEBE
referenciar `manifest.Step`, ningún tipo de provider, ni ningún otro
concepto de ejecución de Shipwright. `pkg/graph` NO DEBE importar
`internal/workflow/manifest` ni ningún paquete `internal/`.

#### Escenario: El paquete no depende del esquema del manifiesto

- DADO el grafo de imports de `pkg/graph`
- CUANDO se inspecciona
- ENTONCES no importa `internal/workflow/manifest` ni ningún otro paquete
  `internal/`

## Fuera del alcance

El escaneo de módulos/dependencias que produce la entrada `edges` (p. ej.
`go list -json`) es responsabilidad del llamador (`internal/ciimpact` de
Verimand), no parte de este contrato. La generación de manifiesto a partir de
un resultado de `Impacted` es responsabilidad de V7 de Verimand (`design.md`
§2.3 del cambio `207-affected-module-detection` en `verimand-platform`), no
de este paquete. Cualquier cambio a `internal/workflow/graph` o al motor de
ejecución de manifiestos existente no se ve afectado por este contrato y
queda fuera de su alcance.

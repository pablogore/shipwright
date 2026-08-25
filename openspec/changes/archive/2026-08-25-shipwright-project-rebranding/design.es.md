# Diseño: Renombramiento del Proyecto Shipwright

> SPEC-000 · Renombramiento mecánico de identidad. No se introduce ni se modifica arquitectura.
>
> Versión en español. La versión canónica es `design.md` (inglés); prevalece ante cualquier discrepancia.

## Enfoque técnico

No hay arquitectura nueva. El trabajo es una sustitución textual ordenada y acotada por tokens
sobre ~905 ocurrencias en ~105 archivos, secuenciada por *dependencia*: primero la ruta del
módulo Go (nada más compila ni puede verificarse hasta que los imports resuelvan) y luego cada
superficie de identidad derivada. Cada fase termina compilando; la última termina con un
barrido de residuos.

Dos mecanismos sostienen todo el cambio: un **mapa de tokens ordenado de coincidencia más larga
primero** (evita reescrituras parciales) y una **lista explícita de identidades preservadas**
(evita daño colateral a la empresa real `getsyntegrity` y al SDK de Dagger).

## Cadena de resolución de identidad (qué depende de qué)

    ruta de módulo en go.mod ──→ 56 imports .go ──→ compila ──→ todo lo de abajo verificable
            │                                                     │
            ├──→ BINARY_NAME del Makefile ──→ .goreleaser.yml ──→ nombres de artefactos en release.yml
            ├──→ flagset/ayuda/versión en main.go ──→ fragmentos CLI en docs y ejemplos
            ├──→ config.EnvPrefix ──→ SHIPWRIGHT_* ──→ secreto de dependabot + entorno de CI
            └──→ lista de candidatos de yaml_parser ──→ .shipwright.yml ──→ configs de docs y ejemplos

## Mapa de tokens ordenado (aplicar estrictamente de arriba hacia abajo)

| # | Coincidencia (literal) | Reemplazo |
|---|---|---|
| 1 | `github.com/getsyntegrity/syntegrity-dagger` | `github.com/pablogore/shipwright` |
| 2 | `github.com/pablogore/syntegrity-dagger` | `github.com/pablogore/shipwright` |
| 3 | `getsyntegrity/syntegrity-dagger` | `pablogore/shipwright` |
| 4 | `.syntegrity-dagger.` / `syntegrity-dagger.y` (archivos de configuración) | `.shipwright.` / `shipwright.y` |
| 5 | `SYNTEGRITY_DAGGER_` | `SHIPWRIGHT_` |
| 6 | `SYNTEGRITY_DAGGER` | `SHIPWRIGHT` |
| 7 | `Syntegrity Dagger` | `Shipwright` |
| 8 | `SyntegrityDagger` | `Shipwright` |
| 9 | `syntegrity_dagger` | `shipwright` |
| 10 | `syntegrity-dagger` (regla general, al final) | `shipwright` |

El orden es determinante: aplicar la regla 10 primero dejaría `github.com/getsyntegrity/shipwright`.
Toda regla exige el par de tokens `syntegrity`+`dagger`, por lo que `dagger` aislado nunca es objetivo.

Manual (el patrón no coincide): `internal/config/errors.go:1` y
`internal/config/appconf.test.go:1` — "aplicación Syntegrity" → "aplicación Shipwright".

## Identidades preservadas (lista de exclusión — deben mostrar cero diff)

| Preservado | Por qué es seguro |
|---|---|
| `dagger.io/dagger` (~19 archivos) | No contiene el token `syntegrity`; inalcanzable por las 10 reglas |
| `github.com/getsyntegrity/go-kit-logger`, `go.sum` | La regla 1 exige el sufijo `/syntegrity-dagger` |
| `Syntegrity CI`, `getsyntegrity.com`, `$HOME/.ssh/syntegrity` (`shared/*_cloner.go`) | Identidad de la empresa |
| `SyntegrityInfraPipeline`, `syntegrity-infra` (`internal/pipelines/infra/`) | Nombre de dominio de la infraestructura de la empresa, sin token `dagger` |
| `ghcr.io/syntegrity` (`examples/configs/tenant-svc.yml`), eventengine en `AGENTS.md` | Ejemplos propiedad de la empresa |
| `grep` de `gitlab.com/syntegrity` en el Makefile, archivo raíz `1export` | Seguimientos preexistentes documentados |
| `openspec/changes/shipwright-project-rebranding/**`, `.git/`, `coverage/` | Los artefactos SDD citan el nombre antiguo de forma deliberada |

## Fases de ejecución

| F | Alcance | Compuerta |
|---|---|---|
| 0 | Congelar la lista de exclusión; registrar conteos base del grep y el % de cobertura | Línea base registrada |
| 1 | `go mod edit -module`; imports en `.go`, `mocks/`, `tests/`, `examples/` | `go build ./...` |
| 2 | `BINARY_NAME` del Makefile, `.goreleaser.yml`, flagset/ayuda/versión en `main.go` | `make build` |
| 3 | `config.EnvPrefix`, lista de 6 candidatos en `yaml_parser.go`, valor por defecto de `-config` en `main.go` | `go test ./internal/config/...` |
| 4 | `git mv .github/actions/syntegrity-dagger shipwright`; `ci.yml`, `release.yml`, `dependabot.yml`, `CODEOWNERS`, `rulesets/README.md`, `scripts/`, `.gitignore` | yamllint |
| 5 | `README.md` y su badge, 21 archivos de `docs/`, `CHANGELOG.md`, `examples/**` | Verificación de enlaces |
| 6 | Comentarios, `.serena/project.yml`, comandos de build/test en `openspec/config.yaml` | gofmt |
| 7 | Barrido de residuos y verificación completa | Ver más abajo |

## Decisiones de arquitectura

| Decisión | Elección | Descartado | Fundamento |
|---|---|---|---|
| Ruta del módulo | `github.com/pablogore/shipwright` | Mantener la org `getsyntegrity/*` | La org en `go.mod` ya está obsoleta frente al remoto real; el producto no es la empresa |
| Mecanismo de reemplazo | Mapa de tokens literales ordenado, más largo primero | Una sola regla amplia `s/syntegrity/shipwright/gi` | Una regla amplia destruye `go-kit-logger`, `SyntegrityInfraPipeline` y los valores por defecto de SSH y registry |
| Nombre del archivo de configuración | Renombrar los 6 candidatos de descubrimiento (`.yml`/`.yaml`, sin punto, `.github/`) | Renombrar solo `.shipwright.yml` | Un renombramiento parcial mantendría vivas rutas de descubrimiento con el nombre antiguo |
| Compatibilidad | Corte limpio, sin alias ni prefijo de entorno dual | Leer ambos prefijos durante una release | La propuesta decidió renombramiento limpio; un fallback sería comportamiento nuevo y rompería la atomicidad |
| Fase 1 primero | Módulo e imports antes que todo lo demás | Empezar por documentación | El compilador es el único verificador automático del bloque más grande (56 archivos) |
| Orden TDD | RED primero en las tres superficies con comportamiento (prefijo de entorno, descubrimiento de config, binario/ayuda) | Renombrar producción y luego arreglar pruebas | `strict_tdd: true`; esas superficies son contratos observables, la documentación no es verificable por pruebas |

## Estrategia de pruebas

| Capa | Qué | Cómo |
|---|---|---|
| Unitaria (RED primero) | `EnvPrefix` = `SHIPWRIGHT_`; lista de candidatos de `findConfigFile` | Actualizar `internal/config/config_test.go` y `yaml_parser_test.go` a los valores nuevos → falla → renombrar |
| Unitaria | Las suites existentes conservan su intención | `go test -race ./...`, cobertura ≥ 90 (umbral sin cambios) |
| Compilación | Imports en `internal/`, `mocks/`, `tests/`, `examples/` | `go build ./...` (no `go build .` — `examples/` es `package main` y solo `./...` lo compila) |
| Integración | Superficie CLI | `./shipwright --help` y `--version` sin ningún token antiguo |
| No regresión | Lista de exclusión intacta | `git diff` sobre las líneas de import de `dagger.io/dagger` y `go-kit-logger` debe quedar vacío |
| Barrido | Residuos | `rg -i 'syntegrity[-_ ]?dagger'` devuelve solo rutas de la lista de exclusión; `rg -i syntegrity` devuelve solo coincidencias de identidad de la empresa |

## Matriz de amenazas

`N/A` — no cambia enrutamiento, subprocesos, automatización de VCS/PR, clasificación de
archivos ejecutables ni integración de procesos. Los scripts de shell, los flujos de CI y el
nombre del binario cambian *solo identificadores*; el flujo de control y la semántica de
invocación quedan equivalentes. La herramienta de renombramiento no debe operar in situ sobre
`.git/`, `go.sum`, `coverage/` ni binarios.

## Migración / Despliegue

Sin migración de datos ni de estado. Dos prerrequisitos operativos fuera de banda, ambos sin código:

1. Renombrar el repositorio de GitHub `pablogore/syntegrity-dagger` → `pablogore/shipwright`
   (y actualizar el remoto git) para que `go get` y las URLs de release resuelvan.
2. Crear el secreto de repositorio `SHIPWRIGHT_TOKEN` antes del merge — `.github/dependabot.yml`
   referencia `secrets.SYNTEGRITY_DAGGER_TOKEN`; renombrar la referencia sin crear el secreto
   rompe la autenticación de Dependabot contra el registry privado.

Reversión: rama de propósito único sin ediciones funcionales — `git revert -m 1 <sha-del-merge>`
restaura la identidad anterior por completo. Si ya se publicó una release con el nombre nuevo,
se emite un tag corregido.

## Preguntas abiertas

- [ ] Momento del renombramiento del repositorio en GitHub respecto al merge (antes o el mismo día) — decisión del propietario, no bloquea el código.
- [ ] Si el nombre antiguo debe mantener la redirección automática de GitHub o reclamarse. No bloqueante.

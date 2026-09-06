# Propuesta: Renombramiento del Proyecto Shipwright

> SPEC-000 · "Mismo producto. Mismo comportamiento. Nueva identidad: Shipwright."
>
> Versión en español. La versión canónica es `proposal.md` (inglés); prevalece ante cualquier discrepancia.

## Intención

El producto se distribuye con una identidad inconsistente: `go.mod` declara `github.com/getsyntegrity/syntegrity-dagger` mientras el remoto real es `github.com/pablogore/syntegrity-dagger`, y existen 866 coincidencias sin distinción de mayúsculas del patrón `syntegrity[-_ ]?dagger` en 97 archivos (ruta del módulo, binario, variables de entorno, nombre del archivo de configuración, CI, documentación, ejemplos, pruebas). El nombre además se confunde conceptualmente con el motor Dagger de terceros que envuelve. Renombrar a **Shipwright** entrega una identidad pública única e inequívoca, sin ningún cambio funcional.

Una identidad externa real y distinta no debe confundirse con el nombre antiguo del producto: `getsyntegrity` es una org de GitHub real que publica `go-kit-logger` (una dependencia legítima, importada en 9+ archivos), es dueña del dominio de correo `getsyntegrity.com`, y aparece en ejemplos de código de `eventengine` (ajenos) en `AGENTS.md`. El término desnudo `syntegrity` (sin `dagger`) en esos contextos es la identidad de la empresa, no la de este producto — se conserva sin tocar.

## Alcance

### Dentro del alcance

| Categoría | Cambio |
|---|---|
| Módulo Go | `go.mod` → `github.com/pablogore/shipwright`; actualizar imports en 56 archivos `.go` |
| CLI / binario | `BINARY_NAME` del `Makefile`, valores por defecto de flags en `main.go`, cadena de versión, texto de ayuda/uso |
| Configuración y entorno | `SYNTEGRITY_DAGGER_*` → `SHIPWRIGHT_*`; `.syntegrity-dagger.yml` → `.shipwright.yml` |
| CI/CD y publicación | `.goreleaser.yml`, `ci.yml`, `release.yml`, `.github/actions/syntegrity-dagger/` (renombrar directorio), `dependabot.yml`, `CODEOWNERS`, `rulesets/README.md` |
| Documentación | `README.md` (incluida la URL del badge), 21 archivos en `docs/`, `CHANGELOG.md` |
| Ejemplos | `examples/`: GitHub Actions, Jenkins, local, configs, ejemplos en Go |
| Pruebas y mocks | `internal/**/*_test.go`, `internal/plugins/mocks*.go`, `tests/`, fixtures |
| Comentarios | Menciones de la identidad antigua en `internal/` |

**Ruta de módulo resuelta:** `github.com/pablogore/shipwright` (coincide con el propietario real del remoto git; la línea `getsyntegrity` en `go.mod` está obsoleta).

### Fuera del alcance

- Imports del SDK `dagger.io/dagger` (~19 archivos) — dependencia legítima de terceros. **No se tocan.** Sin impacto en la versión del SDK de Dagger ni en la compatibilidad de los pasos del pipeline.
- Filtro de cobertura `grep -E "gitlab.com/syntegrity"` del `Makefile` (~líneas 185, 236, 238–240, 274) — referencia muerta a una ruta de módulo previa alojada en GitLab que ya no coincide con `go.mod`; no representa la identidad de este producto. Seguimiento posterior, no parte de este cambio.
- Archivo suelto `1export` en la raíz — volcado accidental de variables de shell, ajeno al proyecto. No se toca.
- Alias de retrocompatibilidad o capas de deprecación para la identidad antigua.
- Rediseño del pipeline, motor de ciclo de vida, Git Flow, soporte de Rust/Java, sistema de overrides, Artifact Model, cualquier capacidad nueva o refactorización no esencial.
- **Identidad real de la empresa/org externa (`getsyntegrity`).** Nunca se toca: `github.com/getsyntegrity/go-kit-logger` (`go.mod`, `go.sum`, todos los archivos `.go` que la importan), los ejemplos ajenos de `eventengine` en `AGENTS.md`, el dominio de correo `getsyntegrity.com` y el autor Git por defecto `"Syntegrity CI"` en `internal/pipelines/shared/{ssh,https}_cloner.go`, la ruta de clave SSH por defecto `$HOME/.ssh/syntegrity` en `ssh_cloner.go`, y el namespace de registry de ejemplo `ghcr.io/syntegrity` en `examples/configs/tenant-svc.yml`. Es la identidad propia de la empresa, distinta del producto que se está renombrando.

## Capacidades

### Capacidades nuevas
- `product-identity`: contrato canónico de nomenclatura pública — ruta del módulo, nombre del binario, prefijo de variables de entorno, archivo de configuración por defecto e identificadores de release/action.

### Capacidades modificadas
- Ninguna. `openspec/specs/` aún no contiene specs de dominio; ningún requisito existente cambia.

## Enfoque

Reemplazo mecánico y exhaustivo sobre las categorías enumeradas, en este orden: ruta de módulo e imports → build/CLI → entorno/configuración → CI/release → documentación/ejemplos → pruebas/comentarios. Cada variante (`syntegrity-dagger`, `SYNTEGRITY_DAGGER`, `Syntegrity Dagger`) se mapea a una única forma Shipwright. El comportamiento, la disposición de paquetes y el flujo de control quedan equivalentes salvo por identificadores y cadenas. La señal de aceptación es build verde más la suite completa de pruebas.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Consumidores externos se rompen (badge de CI, URLs de instalación por curl, variables de entorno de la action compuesta) | Media | **Decisión tomada: renombramiento limpio, sin alias.** No hay evidencia concreta de consumidores (sin badge de pkg.go.dev, sin referencias a registries). Si aparece evidencia, se trata como seguimiento de compatibilidad aparte. |
| Residuos no detectados en algún archivo no revisado | Media | Barrido final en todo el repositorio, sin distinción de mayúsculas, sobre `syntegrity`, excluyendo las dos exclusiones documentadas |
| Reemplazo excesivo de `dagger` que rompa los imports del SDK | Baja | Reemplazar solo las formas del token de identidad antigua; nunca `dagger` aislado |
| El `grep` obsoleto de `gitlab.com/syntegrity` degrade la cobertura en silencio | Baja | Documentado como problema preexistente para seguimiento; verificar que la salida de cobertura no cambie |

## Plan de reversión

Rama de propósito único sin ediciones funcionales — revertir el commit de merge (`git revert -m 1 <sha>`) restaura la identidad anterior por completo. Si una release publicada ya lleva el nombre nuevo, se emite un tag corregido; no hay migración de datos ni de estado.

## Dependencias

- Confirmar que el repositorio de GitHub se renombra a `pablogore/shipwright` (o que la ruta del módulo es alcanzable) antes del merge o junto con él, para que `go get` y las URLs de release resuelvan.

## Criterios de éxito

- [ ] La búsqueda en todo el repositorio, sin distinción de mayúsculas, de la identidad antigua devuelve cero resultados fuera de las dos exclusiones documentadas
- [ ] `go build` y `go test -race ./...` pasan; el umbral de cobertura no cambia
- [ ] El binario, `--help` y la salida de versión muestran únicamente "Shipwright"
- [ ] Las variables `SHIPWRIGHT_*` y `.shipwright.yml` son la única superficie de configuración documentada
- [ ] Los imports de `dagger.io/dagger` y el comportamiento del pipeline quedan demostrablemente sin cambios

## Confirmación de atomicidad

Este cambio es estrictamente atómico: renombra la identidad pública del producto y nada más. No introduce ninguna capacidad, no altera comportamiento, no cambia arquitectura y no realiza refactorizaciones oportunistas. Cualquier fragmento del diff que modifique algo distinto de una cadena de identidad, identificador, ruta o nombre de archivo queda fuera del alcance de SPEC-000.

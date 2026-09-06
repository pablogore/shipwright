# Sistema de Plugins

El sistema de plugins permite extender la funcionalidad de los pipelines sin modificar la librería de Dagger. Esto es útil cuando necesitas funcionalidad específica que no está disponible en los pipelines estándar.

## Casos de Uso

- **Deploy a plataformas específicas**: Deploy a Nomad, ECS, Azure Container Instances, etc.
- **Pasos personalizados**: Agregar pasos específicos de tu organización
- **Integraciones**: Conectar con sistemas externos (monitoreo, notificaciones, etc.)
- **Hooks personalizados**: Ejecutar lógica antes/después de pasos existentes

## Arquitectura

El sistema de plugins consta de:

1. **Interfaz Plugin**: Define el contrato que deben implementar los plugins
2. **PluginContext**: Proporciona acceso a recursos del pipeline (Dagger client, hooks, steps, config)
3. **PluginRegistry**: Registra y gestiona el ciclo de vida de los plugins
4. **PluginLoader**: Carga plugins desde archivos o configuración

## Crear un Plugin

### 1. Implementar la Interfaz Plugin

```go
package myplugin

import (
    "context"
    "github.com/pablogore/shipwright/internal/plugins"
)

type MyPlugin struct {
    name    string
    version string
}

func NewMyPlugin() plugins.Plugin {
    return &MyPlugin{
        name:    "my-plugin",
        version: "1.0.0",
    }
}

func (p *MyPlugin) Name() string {
    return p.name
}

func (p *MyPlugin) Version() string {
    return p.version
}

func (p *MyPlugin) Initialize(ctx context.Context, pluginCtx plugins.PluginContext) error {
    // Registrar hooks, agregar steps, etc.
    return nil
}

func (p *MyPlugin) Cleanup(ctx context.Context) error {
    // Limpiar recursos
    return nil
}
```

### 2. Registrar Hooks

Los plugins pueden registrar hooks para ejecutar lógica antes/después de pasos existentes:

```go
func (p *MyPlugin) Initialize(ctx context.Context, pluginCtx plugins.PluginContext) error {
    hookManager := pluginCtx.GetHookManager()
    
    // Hook después del paso "push"
    hookManager.RegisterHook(
        "push",
        interfaces.HookTypeAfter,
        func(ctx context.Context) error {
            // Tu lógica aquí
            return nil
        },
    )
    
    return nil
}
```

### 3. Agregar Steps Personalizados

Los plugins pueden agregar nuevos pasos al pipeline:

```go
func (p *MyPlugin) Initialize(ctx context.Context, pluginCtx plugins.PluginContext) error {
    stepRegistry := pluginCtx.GetStepRegistry()
    
    stepRegistry.RegisterStep(
        "my-custom-step",
        &myStepHandler{pluginCtx: pluginCtx},
    )
    
    return nil
}

type myStepHandler struct {
    pluginCtx plugins.PluginContext
}

func (h *myStepHandler) CanHandle(stepName string) bool {
    return stepName == "my-custom-step"
}

func (h *myStepHandler) Execute(ctx context.Context, stepName string, config interfaces.StepConfig) error {
    // Implementar la lógica del step
    return nil
}

func (h *myStepHandler) GetStepInfo(stepName string) interfaces.StepConfig {
    return interfaces.StepConfig{
        Name:        stepName,
        Description: "My custom step",
        Required:    false,
        DependsOn:   []string{"push"},
    }
}

func (h *myStepHandler) Validate(stepName string, config interfaces.StepConfig) error {
    return nil
}
```

## Usar Plugins

### Configuración YAML

Agrega la configuración del plugin en tu archivo `.shipwright.yml`:

```yaml
pipeline:
  name: "go-service"

steps:
  - setup
  - build
  - test
  - push
  - deploy-nomad  # Step agregado por el plugin

plugins:
  nomad-deploy:
    type: builtin
    name: nomad-deploy
    config:
      nomad_addr: "http://nomad-cluster.example.com:4646"
      nomad_job_file: "nomad.hcl"
      auto_deploy: true
```

### Plugins Built-in

Los siguientes plugins están disponibles como built-in:

- **nomad-deploy**: Deploy a clusters Nomad

### Plugins desde Archivos

Para cargar un plugin desde un archivo `.so`:

```yaml
plugins:
  my-plugin:
    type: file
    path: /path/to/plugin.so
    config:
      key: value
```

## Ejemplo: Plugin de Deploy a Nomad

El plugin `nomad-deploy` agrega capacidad de deploy a Nomad:

1. **Registra un hook** después del paso `push` para deployar automáticamente
2. **Agrega un step** `deploy-nomad` que puede ejecutarse manualmente

### Configuración

```yaml
plugins:
  nomad-deploy:
    type: builtin
    name: nomad-deploy
    config:
      nomad_addr: "http://nomad-cluster.example.com:4646"
      nomad_job_file: "nomad.hcl"
      auto_deploy: true  # Deploy automático después de push
```

### Uso

```bash
# Ejecutar pipeline completo (deploy automático si auto_deploy: true)
shipwright --pipeline go-service

# Ejecutar solo el step de deploy
shipwright --pipeline go-service --step deploy-nomad
```

## Acceso al Contexto del Pipeline

Los plugins tienen acceso a:

- **Dagger Client**: Para operaciones con contenedores
- **Configuration**: Configuración del pipeline
- **Hook Manager**: Para registrar hooks
- **Step Registry**: Para agregar steps
- **Pipeline**: Instancia del pipeline actual
- **Pipeline Config**: Configuración específica del pipeline

```go
func (p *MyPlugin) Initialize(ctx context.Context, pluginCtx plugins.PluginContext) error {
    // Obtener Dagger client
    daggerClient, err := pluginCtx.GetDaggerClient()
    
    // Obtener configuración
    config := pluginCtx.GetConfiguration()
    value := config.GetString("my.key")
    
    // Obtener pipeline config
    pipelineConfig := pluginCtx.GetPipelineConfig()
    registryURL := pipelineConfig.RegistryURL
    
    return nil
}
```

## Tipos de Hooks

- **HookTypeBefore**: Ejecutado antes del paso
- **HookTypeAfter**: Ejecutado después del paso (siempre)
- **HookTypeSuccess**: Ejecutado después del paso si fue exitoso
- **HookTypeError**: Ejecutado después del paso si falló

## Mejores Prácticas

1. **Manejo de Errores**: Siempre retorna errores descriptivos
2. **Logging**: Usa `logger.L()` para logging consistente
3. **Configuración**: Usa valores por defecto sensatos
4. **Cleanup**: Limpia recursos en `Cleanup()`
5. **Documentación**: Documenta la configuración del plugin

## Limitaciones

- Los plugins no pueden modificar pasos existentes, solo agregar hooks
- Los plugins deben ser thread-safe si se ejecutan en paralelo
- Los plugins desde archivos `.so` requieren compilación con `go build -buildmode=plugin`

## Ejemplos

Ver `examples/configs/go-service-with-nomad.yml` para un ejemplo completo de uso del plugin Nomad.

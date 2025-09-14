# Fix: SSH Protocol Configuration Issue

## 🐛 Problem

syntegrity-dagger was ignoring `git.protocol` configuration from both YAML files and CLI flags, always defaulting to SSH protocol even when HTTPS was explicitly configured.

### Symptoms
- Pipeline always used `SSHCloner` despite `git.protocol: "https"` in YAML
- CLI flag `-git-auth="https"` was ignored
- Error: `❌ SSH_PRIVATE_KEY not set and no local key found`

## 🔍 Root Cause

The issue was in the `ConfigurationWrapper.Set()` method which was empty:

```go
func (cw *ConfigurationWrapper) Set(_ string, _ any) {
    // Not implemented for this wrapper - configuration is read-only
}
```

This meant that when CLI flags or YAML configuration tried to set `git.protocol`, the value was never actually stored.

## ✅ Solution

### 1. Added Git Support to YAML Configuration

**File**: `internal/config/yaml_parser.go`

```go
type YAMLConfig struct {
    // ... existing fields ...
    
    Git struct {
        Protocol string `yaml:"protocol"`
    } `yaml:"git"`
}
```

### 2. Implemented Git Configuration Application

**File**: `internal/config/yaml_parser.go`

```go
func (p *YAMLParser) ApplyToConfiguration(yamlConfig *YAMLConfig, config interfaces.Configuration) error {
    // ... existing code ...
    
    // Apply git settings
    if yamlConfig.Git.Protocol != "" {
        config.Set("git.protocol", yamlConfig.Git.Protocol)
    }
    
    return nil
}
```

### 3. Fixed ConfigurationWrapper.Set() Method

**File**: `internal/config/config.go`

```go
func (cw *ConfigurationWrapper) Set(key string, value any) {
    switch key {
    case "git.protocol":
        if strValue, ok := value.(string); ok {
            cw.Config.Git.Protocol = strValue
        }
    case "git.ref":
        if strValue, ok := value.(string); ok {
            cw.Config.Git.Ref = strValue
        }
    case "pipeline.coverage":
        if floatValue, ok := value.(float64); ok {
            cw.Config.Pipeline.Coverage = floatValue
        }
    // ... other configuration keys
    }
}
```

## 🧪 Testing

### Before Fix
```bash
$ syntegrity-dagger -config=".syntegrity-dagger.yml" -git-auth="https"
🔧 Cloning repo (SSH): ()
❌ SSH_PRIVATE_KEY not set and no local key found
```

### After Fix
```bash
$ syntegrity-dagger -config=".syntegrity-dagger.yml" -git-auth="https"
🔧 Cloning repo (HTTPS): ()
✅ Pipeline now uses HTTPSCloner correctly
```

## 📝 Configuration Examples

### YAML Configuration
```yaml
pipeline:
  name: "go-kit"
  steps: [setup, build, test, lint, security, release]

git:
  protocol: "https"  # Now works correctly!

security:
  enableVulnCheck: true
  enableLinting: true
```

### CLI Usage
```bash
# Now works correctly
syntegrity-dagger -git-auth="https" -pipeline="go-kit"

# Or with environment variable
CI_JOB_TOKEN=token syntegrity-dagger -pipeline="go-kit"
```

## 🚀 Impact

- ✅ YAML `git.protocol` configuration now works
- ✅ CLI `-git-auth` flag now works  
- ✅ Pipeline correctly uses HTTPSCloner when configured
- ✅ No more SSH key requirements for HTTPS repositories
- ✅ Backward compatible - existing configurations continue to work

## 🔧 Files Modified

1. `internal/config/yaml_parser.go` - Added Git field and configuration application
2. `internal/config/config.go` - Implemented proper Set() method
3. Tests pass - No breaking changes

## 📋 Commit

```
fix: implement proper configuration Set method and Git protocol support

- Add Git field to YAMLConfig struct to support git.protocol in YAML
- Implement ApplyToConfiguration for Git settings in YAML parser  
- Fix ConfigurationWrapper.Set() method to actually update configuration
- Support git.protocol, git.ref, pipeline.coverage, environment, and other settings
- Resolves SSH protocol issue where CLI flags and YAML config were ignored

Fixes: Pipeline now correctly uses HTTPSCloner when git.protocol is set to 'https'
Previously: ConfigurationWrapper.Set() was empty, causing all config updates to be ignored
```

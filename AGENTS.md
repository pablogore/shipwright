# AGENTS.md

## 🎯 Project Purpose
This repository is a **multi-project monorepo** that implements different services with a **hexagonal architecture** and **TDD (Test Driven Development)** practices, guaranteeing **100% unit test coverage** and compliance with automated quality rules.

All modifications must maintain architectural consistency, error traceability, module independence, and compliance with the rules defined in `.cursor/`.

---

## 🧭 General Philosophy

- **Strict TDD:** no productive changes without prior tests.
- **Total coverage (100%)**: only **unit tests** (integration tests are optional or contextual).
- **Hexagonal Architecture:** clear separation between domain, application, and infrastructure.
- **Dependency Injection (IoC):** use ContainerProvider pattern - services receive `ports.ContainerProvider` as single dependency.
- **Errors:** must be handled and propagated; `panic`, `log.Fatal`, or `os.Exit` are prohibited. **All errors must be checked** - ignoring errors is prohibited. **Never return nil values on error** - return zero values or error-specific types instead.
- **Logging:** only through `go-kit/logger` with context.
- **Cursor Rules:** each service must pass validations from `.cursor/rules` and `.cursor/container-ioc-rules`. **AUTOMATIC APPLICATION** through `.cursorrules`, pre-commit hooks, and IDE configuration.
- **Code style:** clean, predictable, documented, with atomic commits.
- **Code documentation:** all public code must be documented in English (comments, docstrings, README files).
- **Project documentation upkeep:** keep the main repository README up to date and ensure every topic-specific document lives inside `docs/<topic>/` (no ad‑hoc doc files scattered elsewhere).
- **Interaction language (AI and human):** technical English (neutral US/UK).
- **Configuration pattern:** use Options pattern for all parametrizable configurations. All configuration values must be configurable via environment variables, config files, or constructor options, with sensible defaults. Never hardcode configuration values (timeouts, limits, TTLs, etc.) unless they are truly constants.

---

## 🧪 Testing Policies

### 📊 Coverage Requirements
- **GestorCTL**: 100% minimum
- **nexo-listener-service**: 95% minimum  
- **Other modules**: 90% minimum
- **Domain logic**: 100% mandatory
- **Adapters/ports**: 95% minimum
- **Infrastructure**: 90% minimum
- **CLI commands**: 85% minimum

### 🛠️ Testing Stack
- **`testify/assert`**: For all assertions (mandatory)
- **`testify/require`**: Only for critical assertions
- **Manual mocks**: Instead of generated mocks (prohibited `go.uber.org/mock`)
- **`go-sqlmock`**: For database tests
- **`mocks.go`**: Mandatory file per package with interfaces

### 🎭 Mock Management
- **One `mocks.go` file per package**: Each package containing interfaces must have its own mocks file
- **Manual mocks**: Manual implementations instead of automatic generation
- **Standard structure**: All mocks follow the same implementation pattern
- **Constructor functions**: `NewMock<InterfaceName>()` for each mock
- **Default behavior**: Mocks with sensible default behavior

### 📁 Test File Structure
- **One test file per source file**: `foo.go` → `foo_test.go`
- **Table-driven tests**: For multiple scenarios
- **Grouped subtests**: With `t.Run()`
- **`mocks.go` files**: For all mock definitions

### 🎨 Naming Conventions
- **Test functions**: `TestFunctionName_Scenario`
- **Mock types**: `Mock<InterfaceName>`
- **Helper functions**: `New<Type>ForTest`
- **Fixture builders**: `NewTest<Type>`
- **Avoid stuttering**: Type names should NOT repeat the package name (e.g., `redis.RedisLock` is wrong, use `redis.Lock` instead). Always consider stuttering when creating new types.

### 🚫 Prohibited Patterns
- **No gomock imports**: Only manual mocks
- **No native Go assertions**: Only `testify/assert`
- **No generated mock files**: Only `mocks.go`
- **No test files without `mocks.go`** for packages with interfaces
- **No `panic`, `log.Fatal`, or `os.Exit`**: Errors must be handled and propagated
- **No `t.Skip()`**: All tests must run. If a test cannot run due to a bug or missing feature, fix the underlying issue or mark the test as expected to fail with proper documentation

### 🔧 Test Structure (AAA Pattern)
```go
func TestFunctionName_Scenario(t *testing.T) {
    // Arrange - Set up test data and mocks
    mock := NewMockInterface()
    
    // Act - Execute the function under test
    result, err := functionUnderTest(mock)
    
    // Assert - Verify the results
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### 📝 Example mocks.go File
```go
// mocks.go - Mandatory file per package with interfaces
package mypackage

import (
    "context"
    "errors"
)

// MockUserRepository implements UserRepository for testing
type MockUserRepository struct {
    SaveFunc   func(ctx context.Context, user *User) error
    FindByIDFunc func(ctx context.Context, id string) (*User, error)
    DeleteFunc func(ctx context.Context, id string) error
}

// Interface implementation
func (m *MockUserRepository) Save(ctx context.Context, user *User) error {
    if m.SaveFunc != nil {
        return m.SaveFunc(ctx, user)
    }
    return nil // Default behavior
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    if m.FindByIDFunc != nil {
        return m.FindByIDFunc(ctx, id)
    }
    return &User{ID: id, Name: "test-user"}, nil // Default behavior
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
    if m.DeleteFunc != nil {
        return m.DeleteFunc(ctx, id)
    }
    return nil // Default behavior
}

// Default constructor
func NewMockUserRepository() *MockUserRepository {
    return &MockUserRepository{}
}

// Constructor with error
func NewMockUserRepositoryWithError(err error) *MockUserRepository {
    return &MockUserRepository{
        SaveFunc: func(ctx context.Context, user *User) error {
            return err
        },
        FindByIDFunc: func(ctx context.Context, id string) (*User, error) {
            return nil, err
        },
        DeleteFunc: func(ctx context.Context, id string) error {
            return err
        },
    }
}

// Helper for test data
func NewTestUser() *User {
    return &User{
        ID:   "test-id",
        Name: "test-user",
        Email: "test@example.com",
    }
}
```

### 📈 Automatic Verification
The pre-commit hook verifies:
- All tests use `testify/assert`
- All packages with interfaces have `mocks.go`
- Coverage thresholds are met

### 🎯 Quality Objectives
- **Consistent patterns** across all test files
- **Manual mocks** instead of generated mocks
- **Database testing** with `go-sqlmock` for integration tests
- **Clear documentation** of testing patterns (in English)
- **Cursor rules compliance**: `.cursor/rules` and `.cursor/container-ioc-rules`

### 📚 Code Documentation Requirements
- **All public code must be documented in English**: comments, function docstrings, package documentation, README files
- **Function documentation**: Every exported function must have a docstring in English
- **Package documentation**: Each package should have package-level documentation in English
- **Interface documentation**: All public interfaces must be documented with their purpose and usage
- **Struct documentation**: All exported structs must have documentation explaining their purpose
- **Type documentation**: All exported types must be documented
- **Constants and variables**: Exported constants and variables must be documented
- **Inline comments**: Complex logic must be explained with English comments
- **Example**:
```go
// Package redis provides Redis adapters for distributed locks and idempotency checking.
package redis

// RedisDistributedLock implements DistributedLock using Redis from go-kit.
// It provides distributed locking capabilities with automatic expiration.
type RedisDistributedLock struct {
    client kitcache.RedisClient
    prefix string
}

// NewRedisDistributedLock creates a new instance of RedisDistributedLock using go-kit RedisClient.
// The client must implement kitcache.RedisClient interface and provides OpenTelemetry instrumentation.
func NewRedisDistributedLock(client kitcache.RedisClient) ports.DistributedLock {
    // Implementation...
}

// DistributedLock defines the interface for distributed locks.
type DistributedLock interface {
    // Lock acquires a distributed lock with the specified key and TTL.
    // Returns a Lock that must be used to unlock.
    Lock(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}
```

### 📋 Documentation Checklist
- [ ] All exported functions have English docstrings
- [ ] All exported types have English documentation
- [ ] All exported interfaces have English documentation
- [ ] All exported structs have English documentation
- [ ] All exported constants and variables have English documentation
- [ ] Package-level documentation exists in English
- [ ] Complex logic has inline English comments
- [ ] README files are in English

---

## 🚫 Dead Code and Deprecated Code Prevention - CRITICAL RULE

### **ZERO TOLERANCE POLICY**

**UNDER NO CIRCUMSTANCES is dead code, deprecated code, or commented-out code allowed in this repository.**

If any of these are found, **immediate refactoring is REQUIRED** - no exceptions.

### **❌ PROHIBITED - Dead Code Patterns**

1. **Unused functions, methods, or types**
   ```go
   // ❌ PROHIBITED - Function never called
   func unusedHelper() error {
       return nil
   }
   
   // ❌ PROHIBITED - Method never used
   func (s *Service) deadMethod() {
       // code that nobody calls
   }
   ```

2. **Unused imports**
   ```go
   // ❌ PROHIBITED - Import without usage
   import (
       "fmt"  // Not used anywhere
       "os"   // Not used anywhere
   )
   ```

3. **Unused variables or constants**
   ```go
   // ❌ PROHIBITED - Variable never used
   func process() {
       unusedVar := "never used"
       // code that doesn't use unusedVar
   }
   
   // ❌ PROHIBITED - Constant never referenced
   const unusedConstant = "never used"
   ```

4. **Commented-out code**
   ```go
   // ❌ PROHIBITED - Commented code blocks
   func example() {
       // oldCode := "this was removed"
       // if oldCode != "" {
       //     doSomething()
       // }
       newCode := "current implementation"
   }
   ```

5. **Deprecated functions or methods**
   ```go
   // ❌ PROHIBITED - Deprecated code must be removed, not kept
   // Deprecated: Use NewService() instead
   func NewServiceOld() *Service {
       return &Service{}
   }
   ```

6. **Dead code paths (unreachable code)**
   ```go
   // ❌ PROHIBITED - Code after return
   func example() {
       return
       fmt.Println("never executed") // Dead code
   }
   ```

7. **Unused struct fields**
   ```go
   // ❌ PROHIBITED - Field never accessed
   type Config struct {
       UsedField   string
       UnusedField string // Never read or written
   }
   ```

### **✅ REQUIRED Actions When Dead Code is Found**

1. **Immediate removal**: Delete unused code immediately
2. **Refactor if needed**: If code is "almost unused" but has potential value, refactor to make it useful or remove it
3. **Update documentation**: Remove references to deleted code from README, comments, or docs
4. **Clean up imports**: Remove unused imports using `goimports` or `gofmt`
5. **Verify tests**: Ensure no tests depend on removed code

### **🔍 Detection Methods**

**Automated Detection:**
```bash
# Detect unused code
go vet ./...
golangci-lint run --enable=unused,deadcode,varcheck,structcheck

# Detect unused imports
goimports -l .

# Detect commented code (manual review required)
grep -r "^\s*//.*[a-zA-Z]" --include="*.go" | grep -v "// Package\|//go:build\|// +build"
```

**Manual Review Checklist:**
- [ ] No functions without callers
- [ ] No methods without invocations
- [ ] No types without instantiations
- [ ] No imports without usage
- [ ] No variables without references
- [ ] No constants without references
- [ ] No commented-out code blocks
- [ ] No deprecated functions (must be removed, not marked)
- [ ] No unreachable code paths

### **📋 Refactoring Process**

When dead code is identified:

1. **Identify scope**: Determine what code is unused
2. **Check dependencies**: Verify nothing depends on it (grep for references)
3. **Remove code**: Delete unused functions, methods, types, imports
4. **Update tests**: Remove or update tests that referenced deleted code
5. **Update docs**: Remove documentation references
6. **Verify build**: Ensure `go build ./...` succeeds
7. **Run tests**: Ensure all tests pass
8. **Commit**: Atomic commit with message "refactor: remove dead code"

### **🚨 Exceptions (Rare Cases)**

**ONLY these exceptions are allowed:**

1. **Interface implementations**: Methods required by interface but not used yet
   ```go
   // ✅ ALLOWED - Required by interface
   func (s *Service) RequiredMethod() error {
       return nil // Required by SomeInterface
   }
   ```

2. **Test utilities**: Helper functions only used in tests (must be in `*_test.go` files)
   ```go
   // ✅ ALLOWED - Test-only helper
   func createTestService() *Service {
       return &Service{}
   }
   ```

3. **Build tags**: Code conditionally compiled (must have `//go:build` tag)
   ```go
   // ✅ ALLOWED - Conditional compilation
   //go:build integration
   func integrationOnlyFunction() {}
   ```

### **📚 Examples**

#### Before (❌ Bad - Dead Code)
```go
package handlers

import (
    "context"
    "fmt"  // Unused import
    "os"   // Unused import
)

// Unused function
func oldHandler() error {
    return nil
}

type Handler struct {
    usedField   string
    unusedField string // Unused field
}

func (h *Handler) ProcessRequest(ctx context.Context) error {
    // oldCode := "removed"  // Commented code
    unusedVar := "never used" // Unused variable
    
    result := h.processData()
    return result
}

func (h *Handler) processData() error {
    return nil
}

// Deprecated: Use ProcessRequest instead
func (h *Handler) oldProcess() error {
    return nil
}
```

#### After (✅ Good - Clean Code)
```go
package handlers

import (
    "context"
)

type Handler struct {
    config string
}

func (h *Handler) ProcessRequest(ctx context.Context) error {
    result := h.processData()
    return result
}

func (h *Handler) processData() error {
    // All code is used and has clear purpose
    return nil
}
```

### **🎯 Enforcement**

- **Pre-commit hooks**: Automatically detect and reject commits with dead code
- **CI/CD pipeline**: Fails builds if dead code is detected
- **Code reviews**: Reviewers must flag dead code for immediate removal
- **Regular audits**: Periodic codebase scans for unused code

### **⚡ Immediate Action Required**

If dead, deprecated, or commented code is found:
1. **STOP** current work
2. **REMOVE** the dead code immediately
3. **REFACTOR** if removal affects other code
4. **VERIFY** build and tests pass
5. **COMMIT** removal as separate atomic commit

**NO EXCEPTIONS** - Dead code is technical debt and must be eliminated immediately.

---

## ⚠️ Error Handling - CRITICAL RULE

### **MANDATORY ERROR CHECKING**

**ALL errors MUST be checked and handled. Ignoring errors is PROHIBITED.**

### **❌ PROHIBITED - Error Handling Patterns**

1. **Ignored errors**
   ```go
   // ❌ PROHIBITED - Error ignored
   result, _ := someFunction()
   
   // ❌ PROHIBITED - Error not checked
   someFunction()  // Returns error but not checked
   
   // ❌ PROHIBITED - Error discarded with blank identifier
   _, err := someFunction()
   // err is never checked
   ```

2. **Returning nil on error**
   ```go
   // ❌ PROHIBITED - Returning nil on error
   func GetUser(id string) (*User, error) {
       user, err := db.FindUser(id)
       if err != nil {
           return nil, err  // ❌ BAD - Returns nil pointer
       }
       return user, nil
   }
   
   // ❌ PROHIBITED - Returning nil slice/map on error
   func GetUsers() ([]*User, error) {
       users, err := db.FindAllUsers()
       if err != nil {
           return nil, err  // ❌ BAD - Returns nil slice
       }
       return users, nil
   }
   ```

3. **Silent error swallowing**
   ```go
   // ❌ PROHIBITED - Error logged but not returned
   func process() error {
       if err := doSomething(); err != nil {
           logger.L().Error("failed", "error", err)
           // Error not returned!
           return nil  // ❌ BAD
       }
       return nil
   }
   ```

### **✅ REQUIRED - Error Handling Patterns**

1. **Always check errors**
   ```go
   // ✅ CORRECT - Error always checked
   result, err := someFunction()
   if err != nil {
       return nil, fmt.Errorf("failed to do something: %w", err)
   }
   return result, nil
   
   // ✅ CORRECT - Error checked and handled appropriately
   if err := doSomething(); err != nil {
       logger.L().ErrorContext(ctx, "operation failed", "error", err)
       return fmt.Errorf("operation failed: %w", err)
   }
   ```

2. **Return zero values, not nil, on error**
   ```go
   // ✅ CORRECT - Return zero value on error
   func GetUser(id string) (*User, error) {
       user, err := db.FindUser(id)
       if err != nil {
           return &User{}, err  // ✅ GOOD - Returns zero value struct
       }
       return user, nil
   }
   
   // ✅ CORRECT - Return empty slice on error
   func GetUsers() ([]*User, error) {
       users, err := db.FindAllUsers()
       if err != nil {
           return []*User{}, err  // ✅ GOOD - Returns empty slice
       }
       return users, nil
   }
   
   // ✅ CORRECT - Return empty map on error
   func GetUserMap() (map[string]*User, error) {
       users, err := db.FindAllUsers()
       if err != nil {
           return make(map[string]*User), err  // ✅ GOOD - Returns empty map
       }
       return users, nil
   }
   ```

3. **Proper error propagation**
   ```go
   // ✅ CORRECT - Error wrapped with context
   func processData(ctx context.Context, data []byte) error {
       if err := validateData(data); err != nil {
           return fmt.Errorf("data validation failed: %w", err)
       }
       
       if err := saveData(ctx, data); err != nil {
           return fmt.Errorf("failed to save data: %w", err)
       }
       
       return nil
   }
   ```

4. **Error logging with context**
   ```go
   // ✅ CORRECT - Error logged with context before returning
   func processRequest(ctx context.Context, req Request) error {
       if err := validateRequest(req); err != nil {
           logger.L().ErrorContext(ctx, "request validation failed",
               "error", err,
               "request_id", req.ID)
           return fmt.Errorf("invalid request: %w", err)
       }
       return nil
   }
   ```

### **📋 Error Handling Checklist**

- [ ] All function calls that return errors are checked
- [ ] No blank identifier (`_`) used to ignore errors
- [ ] Errors are wrapped with context using `fmt.Errorf` with `%w` verb
- [ ] Zero values returned on error, not nil pointers/slices/maps
- [ ] Errors are logged with context before returning (when appropriate)
- [ ] Error messages are descriptive and include context
- [ ] No silent error swallowing

### **🔍 Detection Methods**

**Automated Detection:**
```bash
# Detect ignored errors
go vet ./...
golangci-lint run --enable=errcheck,unused

# Detect nil returns on error (manual review)
grep -r "return nil, err" --include="*.go"
```

**Manual Review Checklist:**
- [ ] No `_` used to ignore errors
- [ ] All `if err != nil` blocks properly handle errors
- [ ] No functions return `nil` on error (use zero values)
- [ ] All errors are wrapped with context
- [ ] Error messages are descriptive

### **📚 Examples**

#### Before (❌ Bad - Ignored Errors & Nil Returns)
```go
package handlers

import (
    "context"
    "database/sql"
)

func GetUser(ctx context.Context, id string) (*User, error) {
    // ❌ BAD - Error ignored
    db, _ := sql.Open("postgres", connStr)
    
    // ❌ BAD - Error not checked
    db.Query("SELECT * FROM users WHERE id = $1", id)
    
    // ❌ BAD - Returns nil on error
    user, err := findUserInDB(id)
    if err != nil {
        return nil, err  // ❌ Returns nil pointer
    }
    
    return user, nil
}

func GetUsers(ctx context.Context) ([]*User, error) {
    users, err := findAllUsers()
    if err != nil {
        return nil, err  // ❌ Returns nil slice
    }
    return users, nil
}
```

#### After (✅ Good - Proper Error Handling)
```go
package handlers

import (
    "context"
    "database/sql"
    "fmt"
    
    "github.com/getsyntegrity/go-kit-logger/pkg/logger"
)

func GetUser(ctx context.Context, id string) (*User, error) {
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        logger.L().ErrorContext(ctx, "failed to open database",
            "error", err)
        return &User{}, fmt.Errorf("failed to open database: %w", err)
    }
    defer db.Close()
    
    rows, err := db.Query("SELECT * FROM users WHERE id = $1", id)
    if err != nil {
        logger.L().ErrorContext(ctx, "query failed",
            "error", err,
            "user_id", id)
        return &User{}, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()
    
    user, err := findUserInDB(id)
    if err != nil {
        logger.L().ErrorContext(ctx, "user not found",
            "error", err,
            "user_id", id)
        return &User{}, fmt.Errorf("user not found: %w", err)  // ✅ Returns zero value
    }
    
    return user, nil
}

func GetUsers(ctx context.Context) ([]*User, error) {
    users, err := findAllUsers()
    if err != nil {
        logger.L().ErrorContext(ctx, "failed to get users",
            "error", err)
        return []*User{}, fmt.Errorf("failed to get users: %w", err)  // ✅ Returns empty slice
    }
    return users, nil
}
```

### **🎯 Enforcement**

- **Pre-commit hooks**: Automatically detect and reject commits with ignored errors
- **CI/CD pipeline**: Fails builds if errors are ignored or nil returned on error
- **Code reviews**: Reviewers must flag ignored errors and nil returns for immediate fix
- **Static analysis**: `errcheck` and `unused` linters catch most violations

### **⚡ Immediate Action Required**

If ignored errors or nil returns on error are found:
1. **STOP** current work
2. **FIX** error handling immediately
3. **REPLACE** nil returns with zero values
4. **ADD** proper error checking
5. **VERIFY** build and tests pass
6. **COMMIT** fix as separate atomic commit

**NO EXCEPTIONS** - Error handling is critical for system reliability and must be correct.

---

## 🔧 AUTOMATIC RULE APPLICATION

### **Automatic Application Implemented**

Cursor rules are automatically applied through multiple mechanisms:

#### **1. `.cursorrules` File**
- **Location**: Project root
- **Function**: Cursor IDE automatically loads these rules
- **Application**: Real-time during development

#### **2. Pre-commit Hook**
- **Location**: `.git/hooks/pre-commit`
- **Function**: Verifies rules before each commit
- **Blocking**: Commits with violations are automatically rejected

#### **3. IDE Configuration**
- **Location**: `.vscode/settings.json`
- **Function**: Automatic Cursor/VS Code configuration
- **Application**: Automatic formatting and real-time verification

#### **4. Installation Script**
- **Location**: `scripts/install-cursor-hooks.sh`
- **Function**: Automatically installs all hooks
- **Usage**: `./scripts/install-cursor-hooks.sh`

### **Manual Verification**
```bash
# Complete verification
./scripts/cursor-rules-check.sh -v

# Quick verification
./scripts/cursor-rules-check.sh
```

### **Automatically Applied Rules**
- ✅ **Hexagonal Architecture** - Separation of ports and adapters
- ✅ **Strict TDD** - 100% unit test coverage mandatory
- ✅ **Alphabetical Ordering** - Struct fields, variables, constants
- ✅ **Container IoC** - Use ContainerProvider pattern - services receive `ports.ContainerProvider` as single dependency
- ✅ **Error Handling** - **MANDATORY**: All errors must be checked, never ignored. Never return nil values on error - return zero values instead. No panic/log.Fatal/os.Exit.
- ✅ **Logging** - Only go-kit-logger with context
- ✅ **Manual Mocks** - No automatically generated mocks
- ✅ **Code Cohesion** - No unused imports, no dead code
- ✅ **Dead Code Prevention** - **ZERO TOLERANCE**: No dead code, deprecated code, or commented-out code allowed. Immediate refactoring required if found.
- ✅ **English Documentation** - All public code documentation must be in English
- ✅ **Options Pattern for Configuration** - All parametrizable configurations must use Options pattern with environment variable support and sensible defaults

---

## 🎯 Dependency Injection Pattern (ContainerProvider)

### **ContainerProvider Pattern - Standard IoC Pattern**

All services MUST use the **ContainerProvider pattern** for dependency injection. This pattern provides:

- ✅ **Single dependency**: Services receive only `ports.ContainerProvider` interface
- ✅ **No import cycles**: `ContainerProvider` is defined in `ports`, not `app`
- ✅ **Centralized control**: Container implements `ContainerProvider` and controls what's exposed
- ✅ **Testable**: Easy to mock with `MockContainerProvider`
- ✅ **No nil checks**: Dependencies validated once in constructor
- ✅ **Explicit dependencies**: Clear what each service needs

### **Implementation Pattern**

```go
// ✅ CORRECT - Service receives ContainerProvider
type eventRepositoryImpl struct {
    provider ports.ContainerProvider
}

func NewEventRepository(provider ports.ContainerProvider) ports.EventRepository {
    if provider == nil {
        panic("provider cannot be nil")
    }
    return &eventRepositoryImpl{provider: provider}
}

func (r *eventRepositoryImpl) AppendEvents(ctx context.Context, req ports.AppendRequest) error {
    // Access dependency through provider - no nil checks needed
    return r.provider.EventStore().AppendEvents(ctx, req)
}
```

### **Container Implementation**

```go
// Container implements ContainerProvider interface
func (c *Container) EventRepository() services.EventRepository {
    return services.NewEventRepository(c)  // Container injects itself
}
```

### **Testing Pattern**

```go
// ✅ CORRECT - Use MockContainerProvider in tests
func TestEventRepository_AppendEvents(t *testing.T) {
    mockEventStore := ports.NewMockEventStore()
    mockProvider := ports.NewMockContainerProviderWithEventStore(mockEventStore)
    repo := NewEventRepository(mockProvider)
    
    // Test implementation...
}
```

### **Helper Functions for Tests**

Use helper functions from `mocks.go`:
- `NewMockContainerProvider()` - Empty provider
- `NewMockContainerProviderWithEventStore(eventStore)` - Provider with EventStore
- `NewMockContainerProviderWithBroker(broker)` - Provider with Broker
- `NewMockContainerProviderWithSnapshotStore(store)` - Provider with SnapshotStore
- `NewMockContainerProviderWithSchemaRegistry(registry)` - Provider with SchemaRegistry

### **❌ FORBIDDEN Patterns**

```go
// ❌ BAD - Passing multiple dependencies
func NewEventRepository(eventStore ports.EventStore, broker ports.MessageBroker) ports.EventRepository

// ❌ BAD - Accessing container directly (creates import cycle)
import "github.com/getsyntegrity/eventengine/internal/app"
func NewEventRepository() ports.EventRepository {
    container := app.MustGetContainer()  // Creates import cycle!
}

// ❌ BAD - Passing container as concrete type
func NewEventRepository(container *app.Container) ports.EventRepository  // Creates import cycle!
```

### **✅ REQUIRED Pattern**

```go
// ✅ GOOD - Single ContainerProvider dependency
func NewEventRepository(provider ports.ContainerProvider) ports.EventRepository
func NewEventPublisher(provider ports.ContainerProvider) ports.EventPublisher
func NewSnapshotRepository(provider ports.ContainerProvider) ports.SnapshotRepository
```

---

## ⚙️ Configuration Pattern (Options Pattern)

### **Options Pattern for Parametrizable Configurations**

All parametrizable configurations MUST use the **Options pattern** with the following requirements:

- ✅ **Configurable via environment variables**: All configuration values must be accessible through environment variables (using `koanf` with `EVENT_ENGINE_` prefix)
- ✅ **Configurable via config files**: Support for `.env` files and configuration files
- ✅ **Sensible defaults**: All configurations must have reasonable default values defined as constants in `internal/config/config.go`
- ✅ **No hardcoded values**: Never hardcode configuration values (timeouts, limits, TTLs, buffer sizes, etc.) unless they are truly constants
- ✅ **Documentation**: All configuration fields must be documented with their purpose, default values, and units

### **Implementation Pattern**

```go
// ✅ CORRECT - Configuration in config.go with defaults
type RedpandaConfig struct {
    MaxStreamReaders    int           `koanf:"max_stream_readers"`     // Maximum concurrent streams per tenant (default: 10)
    StreamIdleTimeout   time.Duration `koanf:"stream_idle_timeout"`    // Idle timeout for stream readers (default: 5m)
    StreamCleanupInterval time.Duration `koanf:"stream_cleanup_interval"` // Cleanup interval for idle readers (default: 1m)
}

const (
    DefaultMaxStreamReaders = 10
    DefaultStreamIdleTimeout = 5 * time.Minute
    DefaultStreamCleanupInterval = 1 * time.Minute
)

// ✅ CORRECT - Component uses configuration from config struct
func NewReaderPool(readerFactory KafkaReaderFactory, cfg config.RedpandaConfig) *ReaderPool {
    return &ReaderPool{
        maxConcurrent:   cfg.MaxStreamReaders,  // Uses config value
        maxIdleTime:     cfg.StreamIdleTimeout, // Uses config value
        cleanupInterval: cfg.StreamCleanupInterval, // Uses config value
    }
}
```

### **❌ FORBIDDEN Patterns**

```go
// ❌ BAD - Hardcoded configuration values
func NewReaderPool(readerFactory KafkaReaderFactory, cfg config.RedpandaConfig) *ReaderPool {
    return &ReaderPool{
        maxConcurrent:   10,                    // Hardcoded!
        maxIdleTime:     5 * time.Minute,       // Hardcoded!
        cleanupInterval: 1 * time.Minute,       // Hardcoded!
    }
}

// ❌ BAD - Magic numbers in code
func LoadEvents(ctx context.Context, req LoadRequest) {
    loadTimeout := 30 * time.Second  // Hardcoded timeout!
    maxBytesLimit := 10 * 1024 * 1024 // Hardcoded limit!
}

// ❌ BAD - No configuration support
type ReaderPool struct {
    maxConcurrent int // No way to configure this!
}
```

### **✅ REQUIRED Pattern**

```go
// ✅ GOOD - All values configurable via config struct
type RedpandaConfig struct {
    MaxStreamReaders    int           `koanf:"max_stream_readers"`
    StreamIdleTimeout   time.Duration `koanf:"stream_idle_timeout"`
    LoadEventsTimeoutIndexed time.Duration `koanf:"load_events_timeout_indexed"`
}

// ✅ GOOD - Component uses config values
func (s *EventStore) LoadEvents(ctx context.Context, req LoadRequest) {
    loadTimeout := s.config.LoadEventsTimeoutIndexed  // From config
    maxBytesLimit := s.config.LoadEventsMaxBytesIndexed // From config
}
```

### **Configuration Categories**

The following types of values MUST be configurable:

1. **Timeouts**: All timeouts (read, write, connection, activity, etc.)
2. **Limits**: All limits (concurrent connections, buffer sizes, batch sizes, etc.)
3. **TTLs**: All time-to-live values (cache TTLs, key expiration, etc.)
4. **Sizes**: All size-related values (buffer sizes, message sizes, batch sizes, etc.)
5. **Intervals**: All interval values (cleanup intervals, check intervals, etc.)
6. **Prefixes**: All key/prefix values (Redis keys, topic names, etc.)
7. **Counts**: All count-related values (retry counts, max skips, etc.)

### **Configuration File Structure**

All configurations must be defined in `internal/config/config.go`:

```go
// 1. Define constants for defaults
const (
    DefaultMaxStreamReaders = 10
    DefaultStreamIdleTimeout = 5 * time.Minute
)

// 2. Add fields to config structs with koanf tags
type RedpandaConfig struct {
    MaxStreamReaders    int           `koanf:"max_stream_readers"`
    StreamIdleTimeout   time.Duration `koanf:"stream_idle_timeout"`
}

// 3. Set defaults in defaultConfig()
func defaultConfig() *Config {
    return &Config{
        Redpanda: RedpandaConfig{
            MaxStreamReaders:  DefaultMaxStreamReaders,
            StreamIdleTimeout: DefaultStreamIdleTimeout,
        },
    }
}
```

### **Environment Variable Naming**

Environment variables follow the pattern: `EVENT_ENGINE_<SECTION>_<FIELD>`

- Convert to uppercase
- Replace dots/underscores with underscores
- Example: `RedpandaConfig.MaxStreamReaders` → `EVENT_ENGINE_REDPANDA_MAX_STREAM_READERS`

### **Testing with Custom Configuration**

When testing, always provide a way to override configuration:

```go
// ✅ GOOD - Test can override configuration
func TestReaderPool_CustomConfig(t *testing.T) {
    cfg := config.RedpandaConfig{
        MaxStreamReaders: 5,  // Override for test
        StreamIdleTimeout: 1 * time.Minute,
    }
    pool := NewReaderPool(mockFactory, cfg)
    // Test with custom config
}
```

### **🚫 PROHIBITED - Direct Environment Variable Access**

**UNDER NO CIRCUMSTANCES should code access environment variables directly using `os.Getenv()` or `os.LookupEnv()`.**

All configuration values MUST be accessed through the `config.Config` structure. The only exception is within the `internal/config` package itself, where environment variables are loaded into the config structure.

### **❌ PROHIBITED Patterns**

```go
// ❌ PROHIBITED - Direct os.Getenv access
func GetBrokers() []string {
    brokersStr := os.Getenv("EVENT_ENGINE_REDPANDA_BROKERS")  // ❌ BAD
    return strings.Split(brokersStr, ",")
}

// ❌ PROHIBITED - Direct os.LookupEnv access
func GetEndpoint() string {
    endpoint, _ := os.LookupEnv("EVENT_ENGINE_OBSERVABILITY_TRACING_ENDPOINT")  // ❌ BAD
    return endpoint
}

// ❌ PROHIBITED - Checking environment variables directly
func IsEnabled() bool {
    return os.Getenv("EVENT_ENGINE_SOME_FEATURE_ENABLED") == "true"  // ❌ BAD
}
```

### **✅ REQUIRED Pattern**

```go
// ✅ CORRECT - Access configuration through config structure
func NewService(cfg *config.Config) *Service {
    return &Service{
        brokers: cfg.Redpanda.Brokers,  // ✅ GOOD - From config struct
        endpoint: cfg.Observability.Tracing.Endpoint,  // ✅ GOOD - From config struct
        enabled: cfg.Observability.Tracing.Enabled,  // ✅ GOOD - From config struct
    }
}

// ✅ CORRECT - Pass config as parameter
func ProcessData(ctx context.Context, cfg *config.Config, data []byte) error {
    timeout := cfg.Redpanda.LoadEventsTimeoutIndexed  // ✅ GOOD - From config
    // Use timeout...
}

// ✅ CORRECT - Use config in service methods
func (s *Service) DoSomething(ctx context.Context) error {
    maxRetries := s.config.Redis.ReconciliationMaxRetries  // ✅ GOOD - From config struct
    // Use maxRetries...
}
```

### **📋 Configuration Access Checklist**

- [ ] No `os.Getenv()` calls outside `internal/config` package
- [ ] No `os.LookupEnv()` calls outside `internal/config` package
- [ ] All configuration values accessed through `config.Config` structure
- [ ] Services receive `*config.Config` as dependency
- [ ] Components use config fields, not environment variables directly

### **🔍 Detection Methods**

**Automated Detection:**
```bash
# Find all os.Getenv calls (excluding internal/config)
grep -r "os\.Getenv" --include="*.go" . | grep -v "internal/config"

# Find all os.LookupEnv calls (excluding internal/config)
grep -r "os\.LookupEnv" --include="*.go" . | grep -v "internal/config"
```

**Manual Review Checklist:**
- [ ] No `os.Getenv()` in application code (only in `internal/config`)
- [ ] No `os.LookupEnv()` in application code (only in `internal/config`)
- [ ] All services receive `*config.Config` as parameter
- [ ] All configuration values come from config structure fields

### **📚 Examples**

#### Before (❌ Bad - Direct Environment Variable Access)
```go
package adapters

import (
    "os"
    "strings"
)

func NewBroker() *Broker {
    brokersStr := os.Getenv("EVENT_ENGINE_REDPANDA_BROKERS")  // ❌ BAD
    brokers := strings.Split(brokersStr, ",")
    return &Broker{brokers: brokers}
}
```

#### After (✅ Good - Configuration Structure)
```go
package adapters

import (
    "github.com/getsyntegrity/eventengine/internal/config"
)

func NewBroker(cfg *config.Config) *Broker {
    return &Broker{
        brokers: cfg.Redpanda.Brokers,  // ✅ GOOD - From config struct
    }
}
```

### **🎯 Enforcement**

- **Pre-commit hooks**: Automatically detect and reject commits with direct `os.Getenv()` usage
- **CI/CD pipeline**: Fails builds if direct environment variable access is detected
- **Code reviews**: Reviewers must flag direct `os.Getenv()` usage for immediate fix
- **Static analysis**: `grep` or linters can catch most violations

### **⚡ Immediate Action Required**

If direct `os.Getenv()` or `os.LookupEnv()` usage is found:
1. **STOP** current work
2. **REFACTOR** to use `config.Config` structure
3. **PASS** config as parameter to functions/services
4. **VERIFY** build and tests pass
5. **COMMIT** fix as separate atomic commit

**NO EXCEPTIONS** - Direct environment variable access breaks configuration management and makes testing difficult.

### **🚨 Exception (Only in internal/config)**

The **ONLY** exception is within `internal/config` package, where environment variables are loaded into the config structure:

```go
// ✅ ALLOWED - Only in internal/config package
package config

func New() (*Config, error) {
    // Loading environment variables into config structure is allowed here
    brokersStr := os.Getenv("EVENT_ENGINE_REDPANDA_BROKERS")  // ✅ OK - Only in config package
    // ... load into config structure
}
```

---

## 🧱 Architecture and Design

### Monorepo Structure

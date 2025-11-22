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
- **Errors:** must be handled and propagated; `panic`, `log.Fatal`, or `os.Exit` are prohibited.
- **Logging:** only through `go-kit/logger` with context.
- **Cursor Rules:** each service must pass validations from `.cursor/rules` and `.cursor/container-ioc-rules`. **AUTOMATIC APPLICATION** through `.cursorrules`, pre-commit hooks, and IDE configuration.
- **Code style:** clean, predictable, documented, with atomic commits.
- **Code documentation:** all public code must be documented in English (comments, docstrings, README files).
- **Interaction language (AI and human):** technical English (neutral US/UK).

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
- ✅ **Error Handling** - No ignored errors, no panic/log.Fatal/os.Exit
- ✅ **Logging** - Only go-kit-logger with context
- ✅ **Manual Mocks** - No automatically generated mocks
- ✅ **Code Cohesion** - No unused imports, no dead code
- ✅ **English Documentation** - All public code documentation must be in English

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

func NewEventRepository(provider ports.ContainerProvider) EventRepository {
    if provider == nil {
        panic("provider cannot be nil")
    }
    return &eventRepositoryImpl{provider: provider}
}

func (r *eventRepositoryImpl) AppendEvents(ctx context.Context, req AppendEventsRequest) error {
    // Access dependency through provider - no nil checks needed
    return r.provider.EventStore().AppendEvents(ctx, portsReq)
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
    mockEventStore := new(MockEventStore)
    mockProvider := NewMockContainerProviderWithEventStore(mockEventStore)
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
func NewEventRepository(eventStore ports.EventStore, broker ports.MessageBroker) EventRepository

// ❌ BAD - Accessing container directly (creates import cycle)
import "github.com/getsyntegrity/eventengine/internal/app"
func NewEventRepository() EventRepository {
    container := app.MustGetContainer()  // Creates import cycle!
}

// ❌ BAD - Passing container as concrete type
func NewEventRepository(container *app.Container) EventRepository  // Creates import cycle!
```

### **✅ REQUIRED Pattern**

```go
// ✅ GOOD - Single ContainerProvider dependency
func NewEventRepository(provider ports.ContainerProvider) EventRepository
func NewEventPublisher(provider ports.ContainerProvider) EventPublisher
func NewSnapshotRepository(provider ports.ContainerProvider) SnapshotRepository
```

---

## 🧱 Architecture and Design

### Monorepo Structure

---
description: Strategic Core: Transform code to pure functional style
---

# /refactor-to-pure

## Purpose

Guide the transformation of impure functions into pure functions with explicit effect handling, following functional programming principles.

## Prerequisites

- Strategic Core is installed and initialized
- Code to refactor exists in the project
- Understanding of pure vs impure functions
- Functional TDD standards (recommended)

## Process

### Step 1: Function Analysis

First, I'll analyze the selected function to identify:

1. **Side Effects**
   - I/O operations (file, network, database)
   - State mutations
   - Random number generation
   - Date/time dependencies
   - Logging or console output

2. **Hidden Dependencies**
   - Global variables
   - Environment variables
   - External configuration
   - Implicit context

3. **Return Behavior**
   - Exceptions vs explicit errors
   - Null/undefined returns
   - Inconsistent return types

### Step 2: Pure Core Extraction

I'll help extract the pure computational core:

1. **Identify Pure Logic**
   - Data transformations
   - Calculations
   - Validations
   - Business rules

2. **Parameterize Dependencies**
   - Convert globals to parameters
   - Make time/randomness injectable
   - Pass configuration explicitly

3. **Return Explicit Results**
   - Use Result/Either types for errors
   - Make all outputs explicit
   - Remove exceptions from pure code

### Step 3: Effect Isolation

I'll create an effect boundary:

1. **Define Effect Types**
   - IO operations
   - State changes
   - External calls

2. **Create Effect Handlers**
   - Separate I/O from logic
   - Use dependency injection
   - Implement interpreters

3. **Compose Effects**
   - Chain operations safely
   - Handle errors explicitly
   - Maintain type safety

### Step 4: Property Testing

For the refactored pure functions:

1. **Identify Properties**
   - Invariants that must hold
   - Algebraic laws
   - Business rules

2. **Create Generators**
   - Input generators
   - State generators
   - Edge case coverage

3. **Verify Behavior**
   - Same input → same output
   - No observable side effects
   - Error cases handled

## Refactoring Patterns

### Pattern 1: Dependency Injection

**Before:**
```python
def process_user():
    config = load_config()  # Hidden dependency
    user = db.get_user()    # Side effect
    # ... logic ...
```

**After:**
```python
def process_user(config: Config, get_user: Callable[[], User]) -> Result[ProcessedUser, Error]:
    user = get_user()
    # ... pure logic ...
```

### Pattern 2: Effect Separation

**Before:**
```python
def save_and_notify(data):
    processed = transform(data)
    db.save(processed)      # Side effect
    email.send(processed)   # Side effect
    return processed
```

**After:**
```python
# Pure function
def prepare_save_data(data: Data) -> ProcessedData:
    return transform(data)

# Effect orchestration
def save_and_notify_effects(data: Data) -> IO[Result[ProcessedData, Error]]:
    processed = prepare_save_data(data)
    return IO.sequence([
        db_save(processed),
        email_send(processed)
    ]).map(lambda _: processed)
```

### Pattern 3: Time/Random Injection

**Before:**
```python
def generate_token():
    timestamp = datetime.now()  # Impure
    random_id = uuid.uuid4()   # Impure
    return f"{timestamp}_{random_id}"
```

**After:**
```python
def generate_token(now: datetime, random_gen: Callable[[], str]) -> str:
    return f"{now}_{random_gen()}"
```

## Language-Specific Approaches

### Python
- Use `returns` library for IO/Result types
- Type hints for all functions
- `@dataclass(frozen=True)` for immutability

### TypeScript
- `fp-ts` for functional patterns
- Readonly types and interfaces
- Discriminated unions for errors

### Rust
- Built-in Result/Option types
- Ownership for mutation control
- Traits for effect abstraction

### Go
- Interfaces for dependency injection
- Error as explicit return value
- Functional options pattern

### Haskell
- IO monad for effects
- Type classes for abstraction
- Pure by default

## Next Steps

After refactoring:
1. Update tests to verify purity
2. Add property-based tests
3. Document effect boundaries
4. Update type signatures
5. Refactor dependent code

## Notes

- Start with small, isolated functions
- Maintain backwards compatibility
- Performance impact is usually minimal
- Pure functions are easier to test
- Effects at the edges, purity in the core

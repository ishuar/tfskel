---
name: effective-go
description: "Apply Go best practices, idioms, and conventions from golang.org/doc/effective_go. Use this skill whenever writing, reviewing, or refactoring ANY Go code — even small snippets. Triggers on: Go functions, structs, interfaces, goroutines, error handling, package design, HTTP handlers, CLI tools, or any .go file task. Don't skip this for 'simple' Go requests; idiomatic style matters even in small examples."
---

# Effective Go

Produce clean, idiomatic Go by applying conventions from the [Effective Go guide](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

Follow Keep it simple, clear, and idiomatic. Always prioritize readability and maintainability over cleverness or micro-optimizations. When in doubt, refer to the Go standard library for examples of best practices.
---

## Formatting

Always format with `gofmt` / `goimports`. This is non-negotiable — never produce code that would change under `gofmt`.

---

## Naming

| Context | Rule | Example |
|---|---|---|
| Packages | Lowercase, single word, no underscores | `http`, `userstore` not `user_store` |
| Exported | MixedCaps | `UserCount`, `ParseConfig` |
| Unexported | mixedCaps | `parseToken`, `maxRetries` |
| Acronyms | Keep consistent case | `userID`, `parseURL`, `HTTPClient` |
| Getters | Omit "Get" | `u.Name()` not `u.GetName()` |
| Setters | Keep "Set" | `u.SetName(n)` |
| Interfaces (1 method) | Add "-er" suffix | `Reader`, `Stringer`, `Doer` |

**Anti-patterns to reject:**
```go
// Bad
func get_user_data(user_id int) {}  // underscores
func (u *User) GetName() string {}  // "Get" prefix

// Good
func getUserData(userID int) {}
func (u *User) Name() string {}
```

---

## Error Handling

Always check and return errors. Never ignore with `_` unless genuinely intentional (comment why). Never panic in library code.

**Wrap errors with context using `%w`:**
```go
// Bad
return nil, err

// Good
return nil, fmt.Errorf("opening config file %q: %w", path, err)
```

**Return errors as the last return value:**
```go
func ParseConfig(path string) (*Config, error)
```

**Sentinel errors** for expected conditions:
```go
var ErrNotFound = errors.New("not found")
// caller: if errors.Is(err, ErrNotFound) { ... }
```

---

## Constructors

Use `NewFoo()` functions instead of bare struct literals when initialization logic exists or zero values are invalid:

```go
// NewServer creates a Server with sensible defaults.
func NewServer(addr string, timeout time.Duration) *Server {
	return &Server{
		addr:    addr,
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}
```

---

## Interfaces

- **Keep interfaces small** — 1–3 methods is ideal.
- **Accept interfaces, return concrete types** — callers can always widen; narrowing is breaking.
- **Define interfaces in the consumer package**, not the producer.
- Use a single-method interface to make code testable:

```go
// Define in the package that needs it
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

func FetchUser(client Doer, id int) (*User, error) { ... }
```

---

## Context

Pass `context.Context` as the **first parameter** of any function that does I/O, makes network calls, or is long-running. Never store context in a struct.

```go
// Good
func FetchUser(ctx context.Context, id int) (*User, error)

// Bad — context stored in struct
type Service struct { ctx context.Context }
```

---

## Concurrency

**Prefer channels for ownership transfer; use `sync.Mutex` for protecting shared state.**

```go
// Channel for pipeline / fan-out
jobs := make(chan Job, bufferSize)

// Mutex for a cache
type Cache struct {
	mu    sync.RWMutex
	items map[string]Item
}
```

- Always document goroutine lifetimes: who starts it, what stops it.
- Use `sync.WaitGroup` or `errgroup.Group` to wait for goroutines.
- Avoid goroutine leaks: ensure every goroutine has a clear exit path.

---

## Defer

Use `defer` for cleanup immediately after the resource is acquired:

```go
f, err := os.Open(path)
if err != nil {
	return nil, fmt.Errorf("opening %q: %w", path, err)
}
defer f.Close()
```

---

## Documentation

Document every exported symbol. Start the comment with the symbol name:

```go
// UserStore manages persistent storage of user records.
type UserStore struct { ... }

// Save persists u to the store, overwriting any existing record with the same ID.
func (s *UserStore) Save(ctx context.Context, u *User) error { ... }
```

---

## Don't extract helpers just to reduce repetition

A helper earns its place only when it does one of these:

1. **Hides real complexity** — the inline version is hard to read or easy to get wrong.
2. **Centralizes a decision likely to change** — updating it becomes one edit instead of many.

If neither is true, inline the code. Short obvious repetition costs less than the cognitive hop of chasing a helper that forwards the same arguments elsewhere.

**Signals the helper is dead weight:**

- Its body is one call with the same args rearranged.
- Its name just paraphrases the function it wraps (`newCmdLogger()` wrapping `logger.NewWithOptions(verbose, useColor)`).
- Call-site readers lose load-bearing behavior (e.g. a dry-run wrap or context-specific option) by not seeing it inline.

```go
// Bad — wrapper that hides nothing; reader must jump to see what it does
func newCmdLogger() *logger.Logger {
	return logger.NewWithOptions(viper.GetBool("verbose"), useColor)
}

// Good — inline at each call site; what you read is what happens
log := logger.NewWithOptions(viper.GetBool("verbose"), useColor)
```

"Four call sites → one helper" is not an automatic win. Four plain inline calls beat one opaque helper every time.

---

## Common Anti-patterns to Flag

When reviewing code, always call out:

1. `panic` in non-`main` / non-`init` code
2. Ignored errors (`db, _ := sql.Open(...)`)
3. `GetX()` getter naming
4. Underscore identifiers (`user_id`, `my_func`)
5. Storing `context.Context` in structs
6. Unexported interface defined in producer package
7. Goroutines without documented lifetime / exit
8. Missing `defer` for `Close()` / `Unlock()` calls
9. Wrapper helpers that forward arguments without hiding complexity or centralizing a decision

---

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go standard library](https://pkg.go.dev/std) — use as reference for idiomatic patterns

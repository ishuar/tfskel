---
name: go-cli-logging
description: >
  Enforce Go logging best practices for CLI tools, especially those doing high-volume file
  operations (read, write, parse, modify, scaffold). Use this skill whenever the user is
  writing, reviewing, or debugging Go CLI logging code — even if they only mention "how do
  I log this" or "add logging to my tool." Triggers on: slog, log/slog, zerolog, zap,
  logrus, --verbose flag, debug flag, file operation logging, structured logging, log levels,
  CLI output, scaffold tool, or any Go CLI that reads/writes/parses files. Always apply this
  skill before suggesting any logging approach in Go CLI context — don't rely on training
  data alone, the patterns here are authoritative for idiomatic Go 1.21+.
---

# Go CLI Logging — Best Practices Skill

This skill teaches **idiomatic, production-grade logging** for Go CLI tools that do heavy
file I/O (scaffolding, parsing, modifying files). It is beginner-friendly but enforces
real-world standards.

> **Companion skill**: Also read `/mnt/skills/user/effective-go/SKILL.md` for naming,
> error handling, and general Go idioms that apply everywhere including logging code.

---

## 1. The Golden Rule: Use `log/slog` (Go 1.21+)

`log/slog` is the **official structured logger** in the Go standard library. For a CLI
tool, this is almost always the right choice — no extra dependencies, and it supports
human-readable and machine-readable output out of the box.

```go
import "log/slog"
```

**Do NOT reach for third-party loggers** (zerolog, zap, logrus) unless you have benchmarked
a real bottleneck. For a scaffolding CLI, `slog` is fast enough.

---

## 2. Logger Setup — CLI-Friendly Pattern

Set up your logger once in `main()` and pass it down. Never use the global `slog.Default()`
in library code.

```go
// main.go
func main() {
    verbose := flag.Bool("verbose", false, "enable debug output")
    jsonLog := flag.Bool("log-json", false, "output logs as JSON (useful in CI/pipes)")
    flag.Parse()

    logger := newLogger(*verbose, *jsonLog)
    // Pass logger into your app, e.g. via a Run(logger) function or config struct.
    if err := run(logger, flag.Args()); err != nil {
        logger.Error("fatal error", "err", err)
        os.Exit(1)
    }
}

// newLogger builds a slog.Logger tuned for CLI use.
func newLogger(verbose, asJSON bool) *slog.Logger {
    level := slog.LevelInfo
    if verbose {
        level = slog.LevelDebug
    }

    var handler slog.Handler
    if asJSON {
        handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
    } else {
        handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
    }

    return slog.New(handler)
}
```

**Key rules:**
- Log to **`os.Stderr`**, not `os.Stdout`. Stdout is for program output (generated files,
  paths, etc.); stderr is for diagnostics.
- Text handler for humans, JSON handler for CI/pipes.
- `--verbose` / `-v` gates `slog.LevelDebug`. Ship with `Info` as default.

---

## 3. Log Levels — When to Use Each

| Level   | Use for                                                         |
|---------|-----------------------------------------------------------------|
| `Debug` | Internals: file bytes read, parsed token counts, loop steps     |
| `Info`  | Normal progress: "reading file", "writing scaffold", "done"     |
| `Warn`  | Unexpected but recoverable: file already exists, skip          |
| `Error` | Operation failed: cannot open file, parse error, write failure  |

**Never log and return the same error.** Either log it (and handle it) OR return it to the
caller. Doing both causes duplicate noise.

```go
// BAD — logs AND returns, causing duplicate output up the call stack
func readConfig(logger *slog.Logger, path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        logger.Error("failed to read config", "path", path, "err", err)
        return nil, err  // ← caller will also log this!
    }
    ...
}

// GOOD — return the error, let the top-level caller log it
func readConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading config %q: %w", path, err)
    }
    ...
}
```

---

## 4. Structured Fields — Always Name Your Values

`slog` uses key-value pairs. **Always use named attributes** — they make logs searchable
and grep-friendly.

```go
// BAD — unstructured, hard to parse
logger.Info("processed file /tmp/foo.go 1024 bytes")

// GOOD — structured, every field is queryable
logger.Info("processed file",
    "path", "/tmp/foo.go",
    "bytes", 1024,
    "op", "read",
)
```

### Standard field names for file operations

Use these consistently across your entire CLI so logs are uniform:

| Field      | Type     | Meaning                              |
|------------|----------|--------------------------------------|
| `path`     | string   | File or directory path               |
| `op`       | string   | Operation: `"read"`, `"write"`, `"parse"`, `"scaffold"` |
| `bytes`    | int64    | Bytes read or written                |
| `dur`      | string   | Duration of operation (`time.Since`) |
| `err`      | error    | Error value (slog formats it)        |
| `files`    | int      | Number of files processed            |
| `line`     | int      | Line number (for parse errors)       |

---

## 5. File Operation Logging Patterns

### Reading a file

```go
func readFile(logger *slog.Logger, path string) ([]byte, error) {
    logger.Debug("reading file", "path", path)

    start := time.Now()
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read %q: %w", path, err)
    }

    logger.Debug("read complete", "path", path, "bytes", len(data), "dur", time.Since(start).String())
    return data, nil
}
```

### Writing / scaffolding a file

```go
func writeScaffold(logger *slog.Logger, path string, content []byte) error {
    // Warn if overwriting
    if _, err := os.Stat(path); err == nil {
        logger.Warn("file already exists, overwriting", "path", path)
    }

    logger.Info("writing file", "path", path, "bytes", len(content))

    if err := os.WriteFile(path, content, 0o644); err != nil {
        return fmt.Errorf("write %q: %w", path, err)
    }

    logger.Debug("write complete", "path", path)
    return nil
}
```

### Parsing a file

```go
func parseTemplate(logger *slog.Logger, path string) (*Template, error) {
    logger.Debug("parsing template", "path", path)

    data, err := readFile(logger, path)
    if err != nil {
        return nil, err
    }

    tmpl, err := template.New("").Parse(string(data))
    if err != nil {
        // Include line context when available
        return nil, fmt.Errorf("parse template %q: %w", path, err)
    }

    logger.Debug("parse complete", "path", path)
    return tmpl, nil
}
```

### Batch / directory operations

```go
func scaffoldDir(logger *slog.Logger, src, dst string) error {
    logger.Info("scaffolding directory", "src", src, "dst", dst)

    var count int
    err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return fmt.Errorf("walk %q: %w", path, err)
        }
        if d.IsDir() {
            return nil
        }
        if werr := processFile(logger, path, dst); werr != nil {
            // Log per-file errors as warnings and continue, or return to abort
            logger.Warn("skipping file due to error", "path", path, "err", werr)
        }
        count++
        return nil
    })

    if err != nil {
        return fmt.Errorf("scaffold %q → %q: %w", src, dst, err)
    }

    logger.Info("scaffold complete", "src", src, "dst", dst, "files", count)
    return nil
}
```

---

## 6. Logger with Context — Add Reusable Fields

When you're working inside a specific operation (e.g., processing a named template), attach
context fields once with `logger.With(...)` instead of repeating them on every line.

```go
func processTemplate(logger *slog.Logger, name, srcPath, dstPath string) error {
    // Create a child logger with context baked in
    log := logger.With("template", name, "src", srcPath, "dst", dstPath)

    log.Info("processing template")

    data, err := readFile(log, srcPath)
    if err != nil {
        return err
    }

    rendered, err := render(data)
    if err != nil {
        return fmt.Errorf("render %q: %w", name, err)
    }

    return writeScaffold(log, dstPath, rendered)
}
```

---

## 7. Timing Long Operations

For operations that might be slow (large file walks, network fetches, codegen), always log
duration:

```go
func longOperation(logger *slog.Logger, path string) error {
    start := time.Now()
    logger.Info("starting long operation", "path", path)

    defer func() {
        logger.Info("long operation complete", "path", path, "dur", time.Since(start).String())
    }()

    // ... do the work ...
    return nil
}
```

---

## 8. Anti-Patterns to NEVER Do

```go
// ❌ Using fmt.Println / fmt.Printf for diagnostics — not level-gated, not structured
fmt.Println("Reading file:", path)

// ❌ Using log.Fatal / log.Panic in library functions — only allowed in main()
log.Fatal("cannot open file")

// ❌ Logging AND returning the same error
logger.Error("failed", "err", err)
return err

// ❌ Printing sensitive data (tokens, passwords, private keys)
logger.Debug("config loaded", "token", cfg.APIToken)  // never log secrets

// ❌ Using the global logger in library/package code
slog.Info("doing thing")  // global — untestable, unconfigurable

// ❌ Ignoring errors silently even in debug paths
data, _ := os.ReadFile(path)  // don't do this, ever

// ❌ Concatenating log messages instead of structured fields
logger.Info("read " + path + " " + strconv.Itoa(n) + " bytes")
```

---

## 9. Testing: Make Logging Testable

Pass `*slog.Logger` as a dependency. In tests, use `slog.New(slog.NewTextHandler(io.Discard, nil))`
to silence log output, or capture it with a buffer:

```go
// In tests — discard all log output
func TestProcessFile(t *testing.T) {
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    err := processTemplate(logger, "service", "testdata/service.tmpl", t.TempDir()+"/service.go")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

// In tests — capture log output for assertions
func TestLogsWarningOnOverwrite(t *testing.T) {
    var buf bytes.Buffer
    logger := slog.New(slog.NewTextHandler(&buf, nil))
    // ... trigger overwrite scenario ...
    if !strings.Contains(buf.String(), "already exists") {
        t.Error("expected overwrite warning in logs")
    }
}
```

---

## 10. Complete Minimal Example

See `references/example_cli.go` for a complete working CLI that scaffolds files using all
the patterns above. Read it when the user asks for a full example or starter code.

---

## Quick Reference Checklist

Before submitting any logging code, verify:

- [ ] Logger created in `main()`, passed down as `*slog.Logger` (never global in libraries)
- [ ] Logs go to `os.Stderr`, program output to `os.Stdout`
- [ ] `--verbose` flag gates `slog.LevelDebug`
- [ ] All log calls use named key-value fields (`"path", path`, not `"path: "+path`)
- [ ] No log-and-return duplication on errors
- [ ] No secrets or sensitive data in any log field
- [ ] File ops log: path, bytes, duration at Debug level
- [ ] Warnings on overwrite / unexpected-but-recoverable conditions
- [ ] `logger.With(...)` used when multiple lines share the same context fields
- [ ] Tests pass `io.Discard` logger or capture buffer

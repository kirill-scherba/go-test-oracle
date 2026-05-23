# go-test-oracle — Design Document

## Architecture Decisions

### AD-01: Stateless CLI, No Daemon

**Decision:** `go-test-oracle` is a stateless CLI tool — runs, outputs, exits. No server, no daemon.

**Rationale:** Same philosophy as `go vet`, `go fmt`. Zero configuration, zero state. The MCP adapter is a separate binary that can run persistently for IDE integration.

**Alternatives considered:**
- Single binary with `--mcp` flag — rejected because it mixes concerns; separate binary is cleaner.

---

### AD-02: AST-only Analysis (No `go/types` by Default)

**Decision:** Use `go/ast` + `go/parser` for static analysis. Use `go/types` only when type resolution across packages is needed (e.g., for imported types).

**Rationale:** `go/ast` is sufficient for 90% of the work — finding functions, extracting parameter names and type expressions. `go/types` requires full type-checking (loading imports, resolving dependencies), which is heavier and may fail on incomplete code. Use it as an optional enhancement, not a dependency.

**Trade-off:** Without `go/types`, we can't resolve named types to their underlying definitions (e.g., `type MyString string` → we know it's a named type but not its underlying kind). For MVP, this is acceptable — edge cases for named types will be «zero value + typical value».

---

### AD-03: Template-Based Output

**Decision:** Go test code is generated via `text/template` with pre-defined templates for each generator.

**Rationale:** Templates are maintainable, testable, and allow tweaking output format without touching generation logic. Each generator (edge/fuzz/table) has its own template.

**Template structure:**
```
internal/generator/edge/template.go    — Edge case template
internal/generator/fuzz/template.go    — Fuzz template
internal/generator/table/template.go   — Table-driven template
```

---

### AD-04: Fragment-First Output

**Decision:** Default output is a «fragment» — a standalone `func TestXxx(t *testing.T)` that can be pasted into an existing `*_test.go` file.

**Rationale:** Developers rarely want a full test file replacement. They want to add tests incrementally. Fragments are the least destructive format.

**Full file mode** is available via `--format file` for new packages without existing tests.

**Diff mode** is available via `--format diff` for review-before-apply workflows.

---

### AD-05: Confidence-Driven Skip Injection

**Decision:** Low-confidence tests get `t.Skip("TODO: define expected behavior")` instead of dummy assertions.

**Rationale:** A test that passes trivially (e.g., `if true { return }`) gives false confidence. A skipped test is honest — it says «I need a human». Developers can search for `TODO:` in generated tests and fill in expected behavior.

**Thresholds:**
- `>= 0.8`: Full assertions, no skip
- `0.6 – 0.8`: Assertions with `// TODO:` comments
- `< 0.6`: `t.Skip(...)`

---

### AD-06: Package-Oriented Generators

**Decision:** Each generator is a self-contained package with a single public interface:

```go
type Generator interface {
    Name() string
    Generate(fn *parser.FuncInfo) (*GenerateResult, error)
}

type GenerateResult struct {
    Code       string         // Generated test code
    Confidence float64        // 0.0–1.0
    Reason     string         // Human-readable explanation
    Imports    []string       // Additional imports needed
}
```

**Rationale:** Easy to add new generators (e.g., benchmark generator, mock generator) without touching existing code. Each generator can be tested in isolation.

---

### AD-07: No External Dependencies for Core

**Decision:** Core library (`internal/`) has zero third-party dependencies. Only stdlib.

**Rationale:** `go-test-oracle` is a tool that developers run on their machines. Zero deps = zero dependency hell. The CLIs (`cmd/`) may use `cobra` or `pflag` for argument parsing, but that's optional and can be replaced with stdlib `flag`.

---

## Component Interaction

```
┌──────────────────────────────────────────────────────────┐
│                     cmd/go-test-oracle                    │
│  (CLI entry point: parse args, orchestrate)               │
└──────────┬───────────────────────────────┬───────────────┘
           │                               │
    ┌──────▼──────┐                 ┌──────▼──────┐
    │   parser/   │                 │  output/    │
    │ AST parsing │                 │ Formatters  │
    │ Type info   │                 │ fragment    │
    └──────┬──────┘                 │ diff        │
           │                        │ file        │
           │ FuncInfo               └──────▲──────┘
           │                                │
    ┌──────▼───────────────────────────────┴──────┐
    │              generator/                      │
    │  ┌─────────┐ ┌─────────┐ ┌───────────────┐  │
    │  │  edge/  │ │  fuzz/  │ │    table/     │  │
    │  └────┬────┘ └────┬────┘ └──────┬────────┘  │
    │       │           │             │           │
    │       └───────────┴──────┬──────┘           │
    │                          │                  │
    │                   ┌──────▼──────┐           │
    │                   │   score/    │           │
    │                   │ Confidence  │           │
    │                   └─────────────┘           │
    └─────────────────────────────────────────────┘

    ┌──────────────────────────────────────────────┐
    │            cmd/go-test-oracle-mcp             │
    │  (JSON-RPC 2.0 server, thin wrapper)          │
    │  Uses same internal/ packages as CLI          │
    └──────────────────────────────────────────────┘
```

---

## Data Flow

```
Input: file.go + function name
         │
         ▼
    [parser.Parse(file)] → FuncInfo
         │
         ▼
    [generator.Generate(FuncInfo)]
         │
    ┌────┴────┬─────────┐
    ▼         ▼         ▼
  edge      fuzz      table
    │         │         │
    └────┬────┴────┬────┘
         ▼         ▼
    [score.Score(result)]
         │
         ▼
    GenerateResult {Code, Confidence, Reason, Imports}
         │
         ▼
    [output.Format(result, mode=fragment|diff|file)]
         │
         ▼
    Output: string (printed to stdout or written to file)
```

---

## Key Types

### FuncInfo (parser package)

```go
type FuncInfo struct {
    Name       string       // Function name
    Receiver   *ParamInfo   // nil for plain functions
    Params     []ParamInfo  // Input parameters
    Returns    []TypeInfo   // Return types (empty for void)
    IsExported bool
    IsMethod   bool
    IsGeneric  bool
    TypeParams []TypeInfo   // Generic type parameters
    Pos        token.Pos    // Source position
    File       string       // Source file path
}

type ParamInfo struct {
    Name string   // Parameter name (may be empty)
    Type TypeInfo
}

type TypeInfo struct {
    Name      string   // Type name as written in source
    Kind      TypeKind // Classification
    IsPointer bool
    IsSlice   bool
    IsMap     bool
    IsChan    bool
    Elem      *TypeInfo // Element type (for containers)
    PkgPath   string    // Import path for external types
}

type TypeKind int

const (
    KindInvalid TypeKind = iota
    KindBool
    KindInt
    KindFloat
    KindString
    KindSlice
    KindArray
    KindMap
    KindChan
    KindFunc
    KindStruct
    KindInterface
    KindPointer
    KindNamed    // type MyType ...
    KindGeneric  // T, U, V type parameters
)
```

### Generator Interface (generator package)

```go
type Generator interface {
    Name() string
    Generate(fn *parser.FuncInfo) (*Result, error)
}

type Result struct {
    Code       string   // Generated Go test code
    Confidence float64  // 0.0–1.0
    Reason     string   // Human-readable scoring explanation
    Imports    []string // Additional imports needed
}
```

### Score Result (score package)

```go
type Score struct {
    Confidence float64
    Factors    map[string]float64 // Factor name → contribution
    Reason     string
    Skip       bool               // true if t.Skip() should be injected
}
```

---

## Edge Cases to Handle in Parser

1. **Unexported functions** — skip by default, include with `--all` flag
2. **Methods with pointer/value receivers** — handle both `(t *T)` and `(t T)`
3. **Variadic parameters** — `args ...string` → generate slice edge cases
4. **Multiple return values** — `(int, error)` → check error branch
5. **Named return values** — `(result int, err error)` → generate with named vars
6. **Blank identifiers** — `_ int` → skip, can't inject meaningful value
7. **Generics** — `func F[T any](v T)` → generate with `any` edge cases
8. **Functions with no params** — generate simple invocation test
9. **Functions with no returns** — generate side-effect test stub
10. **Imported types** — `http.Handler` → use full import path in generated code

---

## Test Strategy for go-test-oracle Itself

| Layer | Approach |
|---|---|
| `parser/` | Table-driven tests with `testdata/` fixtures — real `.go` files parsed, verify `FuncInfo` matches expected signature |
| `generator/edge/` | Generate tests for known signatures, verify output contains expected edge case names and `t.Skip` markers |
| `generator/fuzz/` | Verify seed corpus matches expected count and types |
| `generator/table/` | Verify template structure, subtest naming |
| `score/` | Unit tests with known FuncInfo, verify confidence thresholds |
| `output/` | Verify fragment/diff/file format correctness |
| Integration | End-to-end: parse `testdata/sample.go`, generate all three test types, verify output compiles |

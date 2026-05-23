# go-test-oracle — Implementation Plan

## Goal

Build `go-test-oracle` — a Go CLI tool that reads source code via AST and generates
test scaffolding (edge cases, fuzz templates, table-driven boilerplate).

## Repository

- **Repo:** `github.com/kirill-scherba/go-test-oracle`
- **Issue:** [#1](https://github.com/kirill-scherba/go-test-oracle/issues/1)
- **Go version:** 1.22+
- **License:** MIT

---

## Architecture Overview

```
cmd/
├── go-test-oracle/       # CLI binary — stateless, runs like `go vet`
└── go-test-oracle-mcp/   # Optional MCP adapter — thin JSON-RPC wrapper

internal/
├── parser/               # AST parsing, function discovery, type analysis
├── generator/            # Test code generators
│   ├── edge/             # Edge-case scaffolding
│   ├── fuzz/             # Fuzz template generator
│   └── table/            # Table-driven boilerplate
├── score/                # Confidence scoring engine
└── output/               # Output formatters (fragment, diff, full file)
```

**Dependencies:**
- `go/ast`, `go/parser`, `go/token`, `go/types` — static analysis (stdlib)
- `text/template` — test code templating (stdlib)
- No third-party dependencies for core logic

---

## Milestones

### M0: Project Bootstrap

**Goal:** Initialize Go module, project structure, CI.

**Tasks:**
- [x] Create repository `github.com/kirill-scherba/go-test-oracle`
- [ ] `go mod init github.com/kirill-scherba/go-test-oracle`
- [ ] Set up directory structure (`cmd/`, `internal/`, `docs/`)
- [ ] Create `Makefile` with `build`, `test`, `lint`, `clean` targets
- [ ] Add `CONTEXT.md`, `STATUS.md`, `DESIGN.md` to `docs/`
- [ ] CI: GitHub Actions — lint + test on push/PR

**Deliverables:** Buildable empty module, Makefile, CI green.

---

### M1: AST Foundation

**Goal:** Parse Go source files, locate functions, analyze signatures.

**Scope:**
- Parse `.go` files with `go/parser`
- Walk AST to find `*ast.FuncDecl` nodes
- Filter by exported/unexported, name pattern
- Extract: function name, receiver (if method), parameter list (name + type), return types
- Handle: generics (`*ast.IndexListExpr`, type parameters)

**Internal API:**
```go
// parser package
type FuncInfo struct {
    Name       string
    Receiver   *ParamInfo  // nil for plain functions
    Params     []ParamInfo
    Returns    []TypeInfo
    IsExported bool
    IsMethod   bool
    IsGeneric  bool
    TypeParams []TypeInfo
}

type ParamInfo struct {
    Name string
    Type TypeInfo
}

type TypeInfo struct {
    Name       string    // e.g. "string", "int", "MyStruct"
    Kind       TypeKind  // String, Int, Float, Struct, Slice, Map, Chan, Func, Interface, Pointer, Named, Generic
    IsPointer  bool
    IsSlice    bool
    IsMap      bool
    IsChan     bool
    Elem       *TypeInfo // for slice/map/chan/pointer elements
    PkgPath    string    // for imported types
}
```

**Deliverables:** `internal/parser/` package with tests parsing real Go files.

**Test plan:**
- Parse `testdata/sample.go` with various function signatures
- Verify `FuncInfo` matches expected signature
- Test generics, methods, variadic params

---

### M2: Edge-Case Generator

**Goal:** For a given function, generate test scaffolding with edge-case inputs.

**Edge-case mappings:**

| Type Category | Edge Cases Generated |
|---|---|
| `int`, `int8`...`int64` | `0`, `-1`, `1`, `math.MinInt`/`math.MaxInt` |
| `uint`, `uint8`...`uint64` | `0`, `1`, `math.MaxUint` |
| `float32`, `float64` | `0.0`, `-1.0`, `1.0`, `math.NaN()`, `math.Inf(1)`, `math.Inf(-1)` |
| `string` | `""`, `" "`, `"hello"`, very long string, unicode (`"Привет"`), `"\x00"` |
| `bool` | `true`, `false` |
| `[]T` (slice) | `nil`, `[]T{}`, single element, many elements |
| `map[K]V` | `nil`, `map[K]V{}`, single entry, many entries |
| `chan T` | `nil`, buffered, unbuffered |
| `*T` (pointer) | `nil`, `&T{}`, `new(T)` |
| `interface{}` / `any` | `nil`, `""`, `0`, `T{}` |
| `struct` | Zero value, with all fields set, partial fields |
| `func(...)` | `nil`, simple func |
| Named types | Based on underlying type |

**Output template (fragment):**
```go
func TestMyFunc_EdgeCases(t *testing.T) {
    tests := []struct {
        name  string
        input MyInput
        // confidence: 0.8 — based on type coverage
    }{
        {
            name:  "nil_input",
            input: nil,
        },
        {
            name:  "zero_value",
            input: MyInput{},
        },
        // TODO: define expected behavior
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // TODO: call MyFunc(tt.input) and check result
            t.Skip("TODO: define expected behavior")
        })
    }
}
```

**Deliverables:** `internal/generator/edge/` package.

**Key consideration:** When confidence < 0.6, add `t.Skip("TODO: ...")` instead of dummy assertion.

---

### M3: Fuzz Template Generator

**Goal:** Generate `FuzzXxx` functions with type-aware seed corpus.

**Template:**
```go
func FuzzMyFunc(f *testing.F) {
    // Seed corpus — type-aware defaults
    f.Add(int64(0), "")
    f.Add(int64(-1), "hello")
    f.Add(int64(42), "world")

    f.Fuzz(func(t *testing.T, n int64, s string) {
        result := MyFunc(n, s)
        // Basic sanity checks
        if result == nil {
            t.Skip("nil result — may be expected")
        }
    })
}
```

**Seed corpus generation rules**
- 3–5 seed inputs per function
- Type-aware default values
- Mix of zeros, typical values, edge values

**Deliverables:** `internal/generator/fuzz/` package.

---

### M4: Table-Driven Generator

**Goal:** For pure functions (no side effects, deterministic), generate table-driven tests.

**Pure function heuristic:**
- No pointer params (or only `*T` for efficiency, not mutation)
- No channel, mutex, or context params
- Returns values (not void)
- No receiver on mutable struct

**Template:**
```go
func TestMyFunc(t *testing.T) {
    tests := []struct {
        name string
        n    int64
        s    string
        want ResultType
    }{
        {
            name: "empty_string",
            n:    0,
            s:    "",
            want: ResultType{}, // TODO: fill expected output
        },
        {
            name: "normal_case",
            n:    42,
            s:    "hello",
            want: ResultType{}, // TODO: fill expected output
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := MyFunc(tt.n, tt.s)
            if got != tt.want {
                t.Errorf("MyFunc() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Deliverables:** `internal/generator/table/` package.

---

### M5: Confidence Scoring

**Goal:** Assign confidence 0.0–1.0 to each generated test and decide whether to insert `t.Skip`.

**Scoring factors:**

| Factor | Weight | Description |
|---|---|---|
| Type coverage | 0.30 | How many param types we have edge cases for |
| Pure function | 0.25 | How likely the function is pure/deterministic |
| Return type known | 0.20 | Can we analyze return type to suggest assertions |
| Error handling | 0.15 | Can we detect error returns (last param `error`) |
| Complexity | 0.10 | Lower complexity → higher confidence |

**Thresholds:**
- `confidence >= 0.8` → full assertion template, no skip
- `0.6 <= confidence < 0.8` → assertions with `// TODO:` comments
- `confidence < 0.6` → `t.Skip("TODO: define expected behavior")`

**API:**
```go
type ScoreResult struct {
    Confidence float64
    Factors    map[string]float64
    Reason     string // human-readable explanation
}
```

**Deliverables:** `internal/score/` package.

---

### M6: CLI Tool

**Goal:** Command-line interface — `go-test-oracle` binary.

**Usage:**
```
go-test-oracle [flags] <file.go> [function]

Flags:
  --func string       Function name to generate tests for (default: all exported)
  --output string     Output mode: stdout (default), diff, file
  --generator string  Generator: edge, fuzz, table, all (default: all)
  --format string     Output format: fragment (default), file, diff
  --write             Write output to *_test.go file
  --dir string        Process all .go files in directory
```

**Logic:**
1. Parse CLI args
2. Load and parse Go files
3. Filter functions by name/export
4. Run selected generator(s)
5. Score and format output
6. Write or print

**Deliverables:** `cmd/go-test-oracle/main.go`, working binary.

---

### M7: MCP Adapter (Optional)

**Goal:** Thin JSON-RPC 2.0 wrapper for IDE integration (Cline/Codex).

**Tool: `generate_tests`**
```json
{
    "name": "generate_tests",
    "description": "Generate Go test scaffolding for a function",
    "inputSchema": {
        "type": "object",
        "properties": {
            "file": {"type": "string", "description": "Path to .go file"},
            "function": {"type": "string", "description": "Function name"},
            "generator": {"type": "string", "enum": ["edge", "fuzz", "table", "all"]}
        },
        "required": ["file", "function"]
    }
}
```

**Tool: `analyze_function`**
```json
{
    "name": "analyze_function",
    "description": "Analyze a Go function signature and suggest test strategy",
    "inputSchema": {
        "type": "object",
        "properties": {
            "file": {"type": "string"},
            "function": {"type": "string"}
        },
        "required": ["file", "function"]
    }
}
```

**Implementation:** `cmd/go-test-oracle-mcp/main.go` — thin layer calling `internal/` packages.

**Deliverables:** MCP-compliant binary.

---

## Output Format Modes

| Mode | Description |
|---|---|
| `fragment` | Just the test function(s) — paste into existing `*_test.go` |
| `file` | Complete `*_test.go` file with package declaration and imports |
| `diff` | Unified diff against existing test file (if any) or append diff |

---

## Milestone Timeline (Estimated)

| Milestone | Effort | Dependencies |
|---|---|---|
| M0: Bootstrap | 0.5d | — |
| M1: AST Foundation | 1d | M0 |
| M2: Edge-Case Generator | 2d | M1 |
| M3: Fuzz Templates | 1d | M1 |
| M4: Table-Driven | 1.5d | M1 |
| M5: Confidence Scoring | 1d | M2, M3, M4 |
| M6: CLI Tool | 1d | M2–M5 |
| M7: MCP Adapter | 0.5d | M6 |

**Total:** ~8.5 days (sequential worst case), ~5 days with parallel work on M2–M4.

---

## Priority Order

1. **M0** — bootstrap, get CI green
2. **M1** — AST foundation: reads code, understands signatures
3. **M2** — edge-case generator: most valuable first deliverable
4. **M3 + M4** — fuzz + table-driven: can be parallel
5. **M5** — confidence scoring: applies to all generators
6. **M6** — CLI: wraps everything into usable binary
7. **M7** — MCP adapter: optional, for IDE integration

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Generics complexity in AST | Focus on common cases first; skip complex generic constraints in M1 |
| Overly verbose edge cases (combinatorial explosion) | Limit combinations: max 5 edge inputs per param, test one param at a time |
| Import resolution for external types | Use `go/types` for type-checking when `go/ast` is insufficient |
| Confidence scoring is subjective | Start with simple heuristics, refine based on real-world usage |

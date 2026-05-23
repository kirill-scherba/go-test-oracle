# go-test-oracle — Context

## Current State

- **Phase:** Act review — all milestones completed, PR pending
- **Issue:** [#1](https://github.com/kirill-scherba/go-test-oracle/issues/1) — `act/in-progress`
- **Branch:** `feature/1-go-test-oracle`
- **Last commit:** `1710ebd feat(mcp): MCP adapter with generate_tests and analyze_function tools (#1)`

## Architecture

```
cmd/
├── go-test-oracle/       # CLI binary — stateless, runs like `go vet`
└── go-test-oracle-mcp/   # MCP adapter — JSON-RPC for IDE integration

internal/
├── parser/     # AST parsing, function discovery, type analysis
├── generator/  # Generator interface
│   ├── edge/   # Edge-case scaffolding
│   ├── fuzz/   # Fuzz template generator
│   └── table/  # Table-driven boilerplate
├── score/      # Confidence scoring engine + pure function heuristic
└── output/     # Formatters: fragment, diff, full file
```

## Recent Decisions

- **2026-05-23:** All M0–M7 completed in single session
- Architecture: stateless CLI + optional MCP adapter
- Zero third-party dependencies for core logic
- Confidence scoring: 0.0–1.0 with t.Skip for < 0.6
- go/ast without go/types for MVP (lighter, no type-checking required)

## Completed Milestones

| Milestone | Status |
|---|---|
| M0: Bootstrap | ✅ go mod, Makefile, CI, Memory Bank |
| M1: AST Foundation | ✅ Parser with FuncInfo, TypeInfo, generics, methods |
| M2: Edge-Case Generator | ✅ Type→edge mapping, template, methods, confidence |
| M3: Fuzz Templates | ✅ Seed corpus, FuzzXxx, type-aware args |
| M4: Table-Driven | ✅ Pure function detection, zero/typical cases |
| M5: Confidence Scoring | ✅ Factors, thresholds, t.Skip injection |
| M6: CLI Tool | ✅ Flags, orchestration, output modes, integration test |
| M7: MCP Adapter | ✅ JSON-RPC, generate_tests, analyze_function |

## Test Status

- 22 tests across 8 packages
- All passing: `make test` ✅
- Lint clean: `make lint` ✅
- Both binaries build: `make build`, `make build-mcp` ✅

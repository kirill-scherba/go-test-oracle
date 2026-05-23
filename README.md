# go-test-oracle

Go CLI tool that reads source code via AST and generates test scaffolding:
edge cases, fuzz templates, table-driven boilerplate.

## Concept

```bash
go-test-oracle --func MyFunc ./pkg/service.go
```

→ outputs a `*_test.go` fragment with edge cases, fuzz seed, and table-driven test boilerplate.

## Generators

| Generator | Flag | Description |
|---|---|---|
| **Edge-case** | `--generator edge` | nil inputs, empty strings, zero values, boundary values |
| **Fuzz** | `--generator fuzz` | `FuzzFuncName` with type-aware seed corpus |
| **Table-driven** | `--generator table` | Input → output pairs for pure functions |

Each generated test includes a **confidence score** (0.0–1.0). Low confidence → `t.Skip("TODO: define expected behavior")`.

## Architecture

```
cmd/
├── go-test-oracle/       # CLI binary — stateless, runs like `go vet`
└── go-test-oracle-mcp/   # Optional MCP adapter — JSON-RPC for IDE integration

internal/
├── parser/     # AST parsing, function discovery, type analysis
├── generator/  # edge/, fuzz/, table/ — test code generators
├── score/      # Confidence scoring engine
└── output/     # Formatters: fragment, diff, full file
```

Stack: Go 1.22+, `go/ast`, `go/parser`, `go/token`, `text/template`. No external dependencies for core.

## Status

🚧 **Planning phase** — see [docs/plan.md](docs/plan.md) for milestone breakdown.

## License

MIT

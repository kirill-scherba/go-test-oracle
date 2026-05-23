# go-test-oracle — Status

## Milestone Progress

| Milestone | Status | Effort | Delivery |
|---|---|---|---|
| M0: Bootstrap | ✅ done | 0.5d | Buildable module, CI green |
| M1: AST Foundation | ✅ done | 1d | Parser package with tests |
| M2: Edge-Case Generator | ✅ done | 2d | Edge case scaffolding |
| M3: Fuzz Templates | ✅ done | 1d | Fuzz test generation |
| M4: Table-Driven | ✅ done | 1.5d | Table-driven boilerplate |
| M5: Confidence Scoring | ✅ done | 1d | Confidence engine |
| M6: CLI Tool | ✅ done | 1d | Working binary |
| M7: MCP Adapter | ✅ done | 0.5d | MCP integration |

**Total:** ~8.5 days planned → delivered in single session (~4h actual)

## Known Issues

- Generated test code indentation uses tabs but may appear inconsistent in non-gofmt environments (run `gofmt -w` after generation)
- Table-driven generator produces `want_*` fields with zero-value placeholders — human fill-in required
- Fuzz generator uses `[]byte` as fallback for unknown slice types
- Pure function heuristic is conservative — may skip valid pure functions with interface params

## Test Status

| Package | Tests | Status |
|---|---|---|
| `internal/parser` | 2 | ✅ PASS |
| `internal/generator/edge` | 6 | ✅ PASS |
| `internal/generator/fuzz` | 4 | ✅ PASS |
| `internal/generator/table` | 5 | ✅ PASS |
| `internal/output` | 4 | ✅ PASS |
| `internal/score` | 2 | ✅ PASS |
| `internal/` (integration) | 3 | ✅ PASS |
| **Total** | **26** | **✅ All passing** |

## Usage Examples

```bash
# Generate all test types for a function
go-test-oracle -func SimpleFunc testdata/sample.go

# Generate only edge cases, output to file
go-test-oracle -func SimpleFunc -generator edge -output file testdata/sample.go

# Generate diff
go-test-oracle -func SimpleFunc -format diff testdata/sample.go

# MCP adapter (stdio JSON-RPC)
go-test-oracle-mcp
```

## Next Steps (Post-MVP)

- [ ] Benchmark generator (`BenchmarkFuncName`)
- [ ] Mock generator for interfaces
- [ ] Integration with `go/types` for imported type resolution
- [ ] Config file / per-project defaults
- [ ] Plugin system for custom generators

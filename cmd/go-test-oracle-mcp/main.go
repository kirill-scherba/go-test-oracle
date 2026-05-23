// go-test-oracle-mcp — MCP adapter (JSON-RPC wrapper for go-test-oracle).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/edge"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/fuzz"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/table"
	"github.com/kirill-scherba/go-test-oracle/internal/parser"
	"github.com/kirill-scherba/go-test-oracle/internal/score"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	s := newServer()
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.send(rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "Parse error"}})
			continue
		}

		resp := s.handle(req)
		s.send(resp)
	}
}

type server struct{}

func newServer() *server { return &server{} }

func (s *server) send(resp rpcResponse) {
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

func (s *server) handle(req rpcRequest) rpcResponse {
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]string{
				"name":    "go-test-oracle-mcp",
				"version": "0.1.0",
			},
		}

	case "tools/list":
		resp.Result = map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "generate_tests",
					"description": "Generate Go test scaffolding for a function",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"file":      map[string]string{"type": "string", "description": "Path to .go file"},
							"function":  map[string]string{"type": "string", "description": "Function name"},
							"generator": map[string]string{"type": "string", "description": "Generator: edge, fuzz, table, all (default)"},
						},
						"required": []string{"file", "function"},
					},
				},
				{
					"name":        "analyze_function",
					"description": "Analyze a Go function signature and suggest test strategy",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"file":     map[string]string{"type": "string", "description": "Path to .go file"},
							"function": map[string]string{"type": "string", "description": "Function name"},
						},
						"required": []string{"file", "function"},
					},
				},
			},
		}

	case "tools/call":
		var p struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			resp.Error = &rpcError{Code: -32602, Message: "Invalid params"}
			return resp
		}
		resp.Result = s.callTool(p.Name, p.Arguments)

	default:
		resp.Error = &rpcError{Code: -32601, Message: "Method not found"}
	}

	return resp
}

func (s *server) callTool(name string, args map[string]interface{}) map[string]interface{} {
	file, _ := args["file"].(string)
	function, _ := args["function"].(string)

	fn, err := parser.FindFunc(file, function)
	if err != nil {
		return map[string]interface{}{
			"content": []map[string]string{{"type": "text", "value": fmt.Sprintf("Error: %v", err)}},
			"isError": true,
		}
	}
	if fn == nil {
		return map[string]interface{}{
			"content": []map[string]string{{"type": "text", "value": fmt.Sprintf("Function %q not found", function)}},
			"isError": true,
		}
	}

	switch name {
	case "generate_tests":
		genName, _ := args["generator"].(string)
		if genName == "" {
			genName = "all"
		}

		gens := selectGenerators(genName)
		var parts []string
		for _, gen := range gens {
			res, err := gen.Generate(fn)
			if err != nil {
				parts = append(parts, fmt.Sprintf("// Error (%s): %v\n", gen.Name(), err))
				continue
			}
			parts = append(parts, fmt.Sprintf("// --- %s (confidence: %.2f)\n%s\n", gen.Name(), res.Confidence, res.Code))
		}
		return map[string]interface{}{
			"content": []map[string]string{{"type": "text", "value": strings.Join(parts, "\n")}},
		}

	case "analyze_function":
		s := score.Calculate(fn)
		pure := score.IsPureHeuristic(fn)

		var advice []string
		advice = append(advice, fmt.Sprintf("Function: %s", fn.Name))
		advice = append(advice, fmt.Sprintf("Exported: %v, Method: %v, Generic: %v", fn.IsExported, fn.IsMethod, fn.IsGeneric))
		advice = append(advice, fmt.Sprintf("Params: %d, Returns: %d", len(fn.Params), len(fn.Returns)))
		advice = append(advice, fmt.Sprintf("Pure function: %v", pure))
		advice = append(advice, fmt.Sprintf("Confidence: %.2f — %s", s.Confidence, s.Reason))
		if s.Skip {
			advice = append(advice, "⚠ Low confidence — tests will be skipped with TODO")
		}
		for _, p := range fn.Params {
			advice = append(advice, fmt.Sprintf("  - %s %s (kind: %v)", p.Name, p.Type.Name, p.Type.Kind))
		}
		for _, r := range fn.Returns {
			advice = append(advice, fmt.Sprintf("  → %s", r.Name))
		}

		return map[string]interface{}{
			"content": []map[string]string{{"type": "text", "value": strings.Join(advice, "\n")}},
		}

	default:
		return map[string]interface{}{
			"content": []map[string]string{{"type": "text", "value": fmt.Sprintf("Unknown tool: %s", name)}},
			"isError": true,
		}
	}
}

func selectGenerators(name string) []generator.Generator {
	all := []generator.Generator{edge.New(), fuzz.New(), table.New()}
	if name == "all" {
		return all
	}
	switch name {
	case "edge":
		return []generator.Generator{edge.New()}
	case "fuzz":
		return []generator.Generator{fuzz.New()}
	case "table":
		return []generator.Generator{table.New()}
	default:
		return all
	}
}

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/edge"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/fuzz"
	"github.com/kirill-scherba/go-test-oracle/internal/generator/table"
	"github.com/kirill-scherba/go-test-oracle/internal/parser"
)

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	return filepath.Join(dir, "../testdata", name)
}

func TestEndToEnd_GeneratedCodeCompiles(t *testing.T) {
	// Parse sample.go
	path := testdataPath(t, "sample.go")
	funcs, err := parser.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%q): %v", path, err)
	}

	// Find a suitable function — SimpleFunc (exported, known types, returns)
	var target *parser.FuncInfo
	for i := range funcs {
		if funcs[i].Name == "SimpleFunc" {
			target = &funcs[i]
			break
		}
	}
	if target == nil {
		t.Fatal("SimpleFunc not found")
	}

	// Run all three generators — use fragment format, add single package header
	gens := []generator.Generator{edge.New(), fuzz.New(), table.New()}
	var codeParts []string
	codeParts = append(codeParts, "package fixtures\n")
	codeParts = append(codeParts, `import (
	"math"
	"strings"
	"testing"
)
`)
	for _, gen := range gens {
		res, err := gen.Generate(target)
		if err != nil {
			t.Fatalf("%s.Generate: %v", gen.Name(), err)
		}
		codeParts = append(codeParts, res.Code)
	}

	// Write combined test file to temp dir
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sample_test.go")
	fullCode := strings.Join(codeParts, "\n\n")
	if err := os.WriteFile(testFile, []byte(fullCode), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Copy original sample.go and go.mod into the same temp dir
	origBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	origCopy := filepath.Join(tmpDir, "sample.go")
	if err := os.WriteFile(origCopy, origBytes, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Copy go.mod into tmpDir so go test -c can resolve the module
	_, currentFile, _, _ := runtime.Caller(0)
	projRoot := filepath.Dir(filepath.Dir(currentFile)) // go up from internal/ to project root
	goModBytes, err := os.ReadFile(filepath.Join(projRoot, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(go.mod): %v", err)
	}
	goModCopy := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModCopy, goModBytes, 0644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	// Run go test -c to verify the generated code compiles (type resolution)
	cmd := exec.Command("go", "test", "-c", "-o", "/dev/null")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test -c failed (compilation error):\n%s", out)
	}
}

func TestEndToEnd_FindFunc(t *testing.T) {
	path := testdataPath(t, "sample.go")
	fn, err := parser.FindFunc(path, "SimpleFunc")
	if err != nil {
		t.Fatalf("FindFunc: %v", err)
	}
	if fn == nil {
		t.Fatal("FindFunc returned nil")
	}
	if fn.Name != "SimpleFunc" {
		t.Errorf("Name = %q, want SimpleFunc", fn.Name)
	}

	// Generate edge tests
	gen := edge.New()
	res, err := gen.Generate(fn)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if res.Confidence < 0.5 {
		t.Errorf("Confidence = %.2f, want >= 0.5", res.Confidence)
	}

	t.Logf("Generated:\n%s", res.Code)
}

func TestEndToEnd_CliSmoke(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "go-test-oracle")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/go-test-oracle")
	buildCmd.Dir = filepath.Join(tmpDir, "..") // project root
	// Can't use .. as Dir. Find project root from this test file.
	_, testFile, _, _ := runtime.Caller(0)
	projRoot := filepath.Dir(testFile)
	buildCmd.Dir = filepath.Dir(projRoot) // go up from internal/ to project root
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// Run CLI
	path := testdataPath(t, "sample.go")
	cmd := exec.Command(binPath, "-func", "SimpleFunc", "-format", "fragment", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI failed: %v\n%s", err, out)
	}

	outputStr := string(out)
	for _, want := range []string{"TestSimpleFunc_EdgeCases", "FuzzSimpleFunc", "TestSimpleFunc"} {
		if !strings.Contains(outputStr, want) {
			t.Errorf("CLI output missing %q", want)
		}
	}

	t.Logf("CLI output:\n%s", outputStr)
}

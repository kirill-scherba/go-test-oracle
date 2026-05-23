package output

import (
	"strings"
	"testing"

	"github.com/kirill-scherba/go-test-oracle/internal/generator"
)

func TestFormatFragment(t *testing.T) {
	res := &generator.Result{Code: "func TestFoo(t *testing.T) {}"}
	out := Format(res, "fragment", "foo", nil)
	if out != res.Code {
		t.Errorf("fragment = %q, want %q", out, res.Code)
	}
}

func TestFormatFile(t *testing.T) {
	res := &generator.Result{
		Code:    "func TestFoo(t *testing.T) {}",
		Imports: []string{"strings"},
	}
	out := Format(res, "file", "mypkg", nil)
	if !strings.Contains(out, "package mypkg") {
		t.Error("file output missing package declaration")
	}
	if !strings.Contains(out, `import (`) {
		t.Error("file output missing import block")
	}
	if !strings.Contains(out, `"strings"`) {
		t.Error("file output missing strings import")
	}
	if !strings.Contains(out, "func TestFoo") {
		t.Error("file output missing test function")
	}
}

func TestFormatDiff(t *testing.T) {
	res := &generator.Result{Code: "func TestFoo(t *testing.T) {}"}
	out := Format(res, "diff", "mypkg", nil)
	if !strings.Contains(out, "--- /dev/null") {
		t.Error("diff output missing /dev/null header")
	}
	if !strings.Contains(out, "+++ mypkg_test.go") {
		t.Error("diff output missing +++ header")
	}
	if !strings.Contains(out, "+package mypkg") {
		t.Error("diff output missing +package line")
	}
}

func TestTestFileName(t *testing.T) {
	if got := TestFileName("/path/to/foo.go"); got != "/path/to/foo_test.go" {
		t.Errorf("TestFileName = %q, want /path/to/foo_test.go", got)
	}
}

package parser

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	return filepath.Join(dir, "../../testdata", name)
}

func TestParseFile(t *testing.T) {
	path := testdataPath(t, "sample.go")
	funcs, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%q) error: %v", path, err)
	}

	// We expect 19 function declarations in testdata/sample.go
	wantCount := 19
	if len(funcs) != wantCount {
		t.Fatalf("ParseFile(%q) returned %d funcs, want %d", path, len(funcs), wantCount)
	}

	// Build name → FuncInfo map for assertions
	byName := make(map[string]FuncInfo)
	for _, f := range funcs {
		byName[f.Name] = f
	}

	// Test SimpleFunc
	simple, ok := byName["SimpleFunc"]
	if !ok {
		t.Fatal("SimpleFunc not found")
	}
	if !simple.IsExported {
		t.Errorf("SimpleFunc.IsExported = false, want true")
	}
	if simple.IsMethod {
		t.Errorf("SimpleFunc.IsMethod = true, want false")
	}
	if len(simple.Params) != 2 {
		t.Errorf("SimpleFunc.Params = %d, want 2", len(simple.Params))
	}
	if simple.Params[0].Name != "name" || simple.Params[0].Type.Name != "string" {
		t.Errorf("SimpleFunc.Params[0] = %v, want name string", simple.Params[0])
	}
	if simple.Params[1].Name != "count" || simple.Params[1].Type.Name != "int" {
		t.Errorf("SimpleFunc.Params[1] = %v, want count int", simple.Params[1])
	}
	if len(simple.Returns) != 1 || simple.Returns[0].Name != "string" {
		t.Errorf("SimpleFunc.Returns = %v, want [string]", simple.Returns)
	}

	// Test NoParams
	noParams, ok := byName["NoParams"]
	if !ok {
		t.Fatal("NoParams not found")
	}
	if len(noParams.Params) != 0 {
		t.Errorf("NoParams.Params = %d, want 0", len(noParams.Params))
	}
	if len(noParams.Returns) != 1 || noParams.Returns[0].Name != "int" {
		t.Errorf("NoParams.Returns = %v, want [int]", noParams.Returns)
	}

	// Test NoReturns
	noReturns, ok := byName["NoReturns"]
	if !ok {
		t.Fatal("NoReturns not found")
	}
	if len(noReturns.Returns) != 0 {
		t.Errorf("NoReturns.Returns = %d, want 0", len(noReturns.Returns))
	}

	// Test MultipleReturns
	multi, ok := byName["MultipleReturns"]
	if !ok {
		t.Fatal("MultipleReturns not found")
	}
	if len(multi.Returns) != 2 {
		t.Errorf("MultipleReturns.Returns = %d, want 2", len(multi.Returns))
	}
	if multi.Returns[1].Name != "error" {
		t.Errorf("MultipleReturns.Returns[1].Name = %q, want error", multi.Returns[1].Name)
	}

	// Test VariadicFunc
	variadic, ok := byName["VariadicFunc"]
	if !ok {
		t.Fatal("VariadicFunc not found")
	}
	if !variadic.IsVariadic {
		t.Errorf("VariadicFunc.IsVariadic = false, want true")
	}
	if len(variadic.Params) != 2 {
		t.Errorf("VariadicFunc.Params = %d, want 2", len(variadic.Params))
	}

	// Test PointerParam
	ptr, ok := byName["PointerParam"]
	if !ok {
		t.Fatal("PointerParam not found")
	}
	if !ptr.Params[0].Type.IsPointer {
		t.Errorf("PointerParam.Params[0].Type.IsPointer = false, want true")
	}
	if ptr.Params[0].Type.Name != "*string" {
		t.Errorf("PointerParam.Params[0].Type.Name = %q, want *string", ptr.Params[0].Type.Name)
	}

	// Test SliceParam
	slice, ok := byName["SliceParam"]
	if !ok {
		t.Fatal("SliceParam not found")
	}
	if !slice.Params[0].Type.IsSlice {
		t.Errorf("SliceParam.Params[0].Type.IsSlice = false, want true")
	}
	if slice.Params[0].Type.Kind != KindSlice {
		t.Errorf("SliceParam.Params[0].Type.Kind = %v, want KindSlice", slice.Params[0].Type.Kind)
	}

	// Test MapParam
	m, ok := byName["MapParam"]
	if !ok {
		t.Fatal("MapParam not found")
	}
	if !m.Params[0].Type.IsMap {
		t.Errorf("MapParam.Params[0].Type.IsMap = false, want true")
	}
	if m.Params[0].Type.Kind != KindMap {
		t.Errorf("MapParam.Params[0].Type.Kind = %v, want KindMap", m.Params[0].Type.Kind)
	}

	// Test ChannelParam
	ch, ok := byName["ChannelParam"]
	if !ok {
		t.Fatal("ChannelParam not found")
	}
	if !ch.Params[0].Type.IsChan {
		t.Errorf("ChannelParam.Params[0].Type.IsChan = false, want true")
	}

	// Test InterfaceParam
	iface, ok := byName["InterfaceParam"]
	if !ok {
		t.Fatal("InterfaceParam not found")
	}
	if iface.Params[0].Type.Kind != KindInterface {
		t.Errorf("InterfaceParam.Params[0].Type.Kind = %v, want KindInterface", iface.Params[0].Type.Kind)
	}

	// Test FuncParam
	fn, ok := byName["FuncParam"]
	if !ok {
		t.Fatal("FuncParam not found")
	}
	if fn.Params[0].Type.Kind != KindFunc {
		t.Errorf("FuncParam.Params[0].Type.Kind = %v, want KindFunc", fn.Params[0].Type.Kind)
	}

	// Test NamedTypeParam
	named, ok := byName["NamedTypeParam"]
	if !ok {
		t.Fatal("NamedTypeParam not found")
	}
	if named.Params[0].Type.Kind != KindNamed {
		t.Errorf("NamedTypeParam.Params[0].Type.Kind = %v, want KindNamed", named.Params[0].Type.Kind)
	}
	if named.Returns[0].Name != "MyString" {
		t.Errorf("NamedTypeParam.Returns[0].Name = %q, want MyString", named.Returns[0].Name)
	}

	// Test Method (value receiver)
	method, ok := byName["Method"]
	if !ok {
		t.Fatal("Method not found")
	}
	if !method.IsMethod {
		t.Errorf("Method.IsMethod = false, want true")
	}
	if method.Receiver == nil {
		t.Fatal("Method.Receiver = nil")
	}
	if method.Receiver.Type.Name != "Counter" {
		t.Errorf("Method.Receiver.Type.Name = %q, want Counter", method.Receiver.Type.Name)
	}

	// Test PtrMethod (pointer receiver)
	ptrMethod, ok := byName["PtrMethod"]
	if !ok {
		t.Fatal("PtrMethod not found")
	}
	if !ptrMethod.IsMethod {
		t.Errorf("PtrMethod.IsMethod = false, want true")
	}
	if !ptrMethod.Receiver.Type.IsPointer {
		t.Errorf("PtrMethod.Receiver.Type.IsPointer = false, want true")
	}

	// Test GenericFunc
	gen, ok := byName["GenericFunc"]
	if !ok {
		t.Fatal("GenericFunc not found")
	}
	if !gen.IsGeneric {
		t.Errorf("GenericFunc.IsGeneric = false, want true")
	}
	if len(gen.TypeParams) != 1 {
		t.Errorf("GenericFunc.TypeParams = %d, want 1", len(gen.TypeParams))
	}
	if gen.TypeParams[0].Name != "T" {
		t.Errorf("GenericFunc.TypeParams[0].Name = %q, want T", gen.TypeParams[0].Name)
	}

	// Test GenericFuncMultiple
	genMulti, ok := byName["GenericFuncMultiple"]
	if !ok {
		t.Fatal("GenericFuncMultiple not found")
	}
	if !genMulti.IsGeneric {
		t.Errorf("GenericFuncMultiple.IsGeneric = false, want true")
	}
	if len(genMulti.TypeParams) != 2 {
		t.Errorf("GenericFuncMultiple.TypeParams = %d, want 2", len(genMulti.TypeParams))
	}

	// Test NamedReturn
	namedRet, ok := byName["NamedReturn"]
	if !ok {
		t.Fatal("NamedReturn not found")
	}
	if len(namedRet.Returns) != 2 {
		t.Errorf("NamedReturn.Returns = %d, want 2", len(namedRet.Returns))
	}

	// Test BlankParam — blank identifier keeps name "_" in AST
	blank, ok := byName["BlankParam"]
	if !ok {
		t.Fatal("BlankParam not found")
	}
	if blank.Params[0].Name != "_" {
		t.Errorf("BlankParam.Params[0].Name = %q, want _", blank.Params[0].Name)
	}

	// Test unexportedFunc
	unexported, ok := byName["unexportedFunc"]
	if !ok {
		t.Fatal("unexportedFunc not found")
	}
	if unexported.IsExported {
		t.Errorf("unexportedFunc.IsExported = true, want false")
	}
}

func TestFindFunc(t *testing.T) {
	path := testdataPath(t, "sample.go")
	fn, err := FindFunc(path, "SimpleFunc")
	if err != nil {
		t.Fatalf("FindFunc error: %v", err)
	}
	if fn == nil {
		t.Fatal("FindFunc returned nil for SimpleFunc")
	}
	if fn.Name != "SimpleFunc" {
		t.Errorf("FindFunc.Name = %q, want SimpleFunc", fn.Name)
	}

	// Not found
	notFound, err := FindFunc(path, "NonExistent")
	if err != nil {
		t.Fatalf("FindFunc error: %v", err)
	}
	if notFound != nil {
		t.Errorf("FindFunc non-existent = %v, want nil", notFound)
	}
}

// Package fixtures provides sample functions for parser testing.
package fixtures

// SimpleFunc is a plain function with basic parameters.
func SimpleFunc(name string, count int) string {
	return name + string(rune(count))
}

// NoParams returns a value with no parameters.
func NoParams() int {
	return 42
}

// NoReturns performs a side effect with no return value.
func NoReturns(msg string) {
	println(msg)
}

// MultipleReturns returns multiple values.
func MultipleReturns(a, b int) (int, error) {
	return a + b, nil
}

// VariadicFunc accepts variadic string parameters.
func VariadicFunc(prefix string, args ...string) string {
	return prefix
}

// PointerParam takes a pointer argument.
func PointerParam(data *string) bool {
	return data != nil
}

// SliceParam takes a slice argument.
func SliceParam(items []int) int {
	return len(items)
}

// MapParam takes a map argument.
func MapParam(m map[string]int) int {
	return len(m)
}

// ChannelParam takes a channel argument.
func ChannelParam(ch chan int) int {
	return cap(ch)
}

// InterfaceParam takes an interface argument.
func InterfaceParam(v interface{}) string {
	return "ok"
}

// FuncParam takes a function argument.
func FuncParam(f func(int) int) int {
	return f(0)
}

// NamedTypeParam uses a named type.
type MyString string

func NamedTypeParam(s MyString) MyString {
	return s
}

// Method has a value receiver.
type Counter struct{ Value int }

func (c Counter) Method() int {
	return c.Value
}

// PtrMethod has a pointer receiver.
func (c *Counter) PtrMethod() int {
	return c.Value
}

// GenericFunc is a generic function.
func GenericFunc[T any](v T) T {
	return v
}

// GenericFuncMultiple has multiple type parameters.
func GenericFuncMultiple[T any, U comparable](a T, b U) (T, U) {
	return a, b
}

// NamedReturn has named return values.
func NamedReturn(x int) (result int, err error) {
	result = x
	return
}

// BlankParam uses a blank identifier.
func BlankParam(_ int, name string) string {
	return name
}

// UnexportedFunc is not exported.
func unexportedFunc() string {
	return "secret"
}

package wasm

import "fmt"

// CompositeForm discriminates the form of a composite type entry in the
// module's type section. Wasm 1.0 / 2.0 supported only function types; the
// WebAssembly GC proposal adds struct and array forms.
//
// CompositeFormFunc is the zero value so a default-constructed FunctionType
// remains a function type — preserving backward compatibility with all
// code written before the GC additions.
type CompositeForm uint8

const (
	CompositeFormFunc CompositeForm = iota
	CompositeFormStruct
	CompositeFormArray
)

func (f CompositeForm) String() string {
	switch f {
	case CompositeFormFunc:
		return "func"
	case CompositeFormStruct:
		return "struct"
	case CompositeFormArray:
		return "array"
	}
	return fmt.Sprintf("<unknown composite form %d>", f)
}

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

// PackedType is the storage type of a struct field or array element when the
// field is not a full value type but a sub-byte / 2-byte packed integer.
// Introduced by the GC proposal.
type PackedType uint8

const (
	// PackedTypeNone indicates the field is NOT a packed type; the
	// companion ValueType on FieldType describes the storage.
	PackedTypeNone PackedType = iota
	// PackedTypeI8 is an 8-bit packed integer (spec byte 0x78).
	PackedTypeI8
	// PackedTypeI16 is a 16-bit packed integer (spec byte 0x77).
	PackedTypeI16
)

// PackedTypeI8Byte and PackedTypeI16Byte are the wire-format encodings of
// the packed storage types.
const (
	PackedTypeI8Byte  byte = 0x78
	PackedTypeI16Byte byte = 0x77
)

func (p PackedType) String() string {
	switch p {
	case PackedTypeI8:
		return "i8"
	case PackedTypeI16:
		return "i16"
	}
	return "<none>"
}

// FieldType describes the type of a struct field or array element.
// Either Packed names a packed-integer storage type, or ValueType
// describes a value-type storage. Exactly one path is meaningful per
// instance.
type FieldType struct {
	// Packed, when not PackedTypeNone, takes precedence over ValueType.
	// Reads of a packed field require struct.get_s / struct.get_u
	// (array.get_s / array.get_u) to choose sign-extension.
	Packed PackedType
	// ValueType is the field's value type when Packed == PackedTypeNone.
	// The full uint64 encoding carries nullability and concrete-ref
	// information.
	ValueType ValueType
	// Mutable indicates the field can be written via struct.set / array.set.
	Mutable bool
}

// String renders the field as spec text format, e.g. "mut i8" or "(ref null any)".
func (f FieldType) String() string {
	var prefix string
	if f.Mutable {
		prefix = "mut "
	}
	if f.Packed != PackedTypeNone {
		return prefix + f.Packed.String()
	}
	return prefix + ValueTypeName(f.ValueType)
}

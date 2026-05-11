package wasm

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestCompositeForm_String(t *testing.T) {
	require.Equal(t, "func", CompositeFormFunc.String())
	require.Equal(t, "struct", CompositeFormStruct.String())
	require.Equal(t, "array", CompositeFormArray.String())
}

func TestPackedType_String(t *testing.T) {
	require.Equal(t, "i8", PackedTypeI8.String())
	require.Equal(t, "i16", PackedTypeI16.String())
	require.Equal(t, "<none>", PackedTypeNone.String())
}

func TestFieldType_String(t *testing.T) {
	require.Equal(t, "i32", FieldType{ValueType: ValueTypeI32}.String())
	require.Equal(t, "mut i32", FieldType{ValueType: ValueTypeI32, Mutable: true}.String())
	require.Equal(t, "i8", FieldType{Packed: PackedTypeI8}.String())
	require.Equal(t, "mut i16", FieldType{Packed: PackedTypeI16, Mutable: true}.String())
}

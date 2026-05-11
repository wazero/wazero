package wasm

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestFunctionTypeKey_FuncForm(t *testing.T) {
	f := FunctionType{Params: []ValueType{ValueTypeI32}, Results: []ValueType{ValueTypeI32}}
	require.Equal(t, "i32_i32", f.key())
}

func TestFunctionTypeKey_StructForm(t *testing.T) {
	f := FunctionType{
		Form: CompositeFormStruct,
		Fields: []FieldType{
			{ValueType: ValueTypeI32},
			{ValueType: ValueTypeI64, Mutable: true},
		},
	}
	require.Equal(t, "struct{i32,mut i64}", f.key())
}

func TestFunctionTypeKey_StructEmpty(t *testing.T) {
	f := FunctionType{Form: CompositeFormStruct}
	require.Equal(t, "struct{}", f.key())
}

func TestFunctionTypeKey_ArrayForm(t *testing.T) {
	f := FunctionType{
		Form:       CompositeFormArray,
		ArrayField: FieldType{ValueType: ValueTypeI32, Mutable: true},
	}
	require.Equal(t, "array(mut i32)", f.key())
}

func TestFunctionTypeKey_StructWithSuper(t *testing.T) {
	idx := uint32(3)
	f := FunctionType{
		Form:           CompositeFormStruct,
		Fields:         []FieldType{{ValueType: ValueTypeI32}},
		SuperTypeIndex: &idx,
		Final:          true,
	}
	require.Equal(t, "struct{i32}|sup=3|final", f.key())
}

func TestFunctionTypeKey_RecGroupAppended(t *testing.T) {
	f := FunctionType{
		Form:             CompositeFormStruct,
		Fields:           []FieldType{{ValueType: ValueTypeI32}},
		RecGroupSize:     2,
		RecGroupPosition: 0,
	}
	require.Equal(t, "struct{i32}|rec0/2", f.key())
}

// TestFunctionTypeKey_FuncWithRecGroupOnly preserves the legacy key shape
// for non-GC modules — no Form prefix, just the param/result encoding.
func TestFunctionTypeKey_FuncWithRecGroupOnly(t *testing.T) {
	f := FunctionType{Params: []ValueType{ValueTypeI32}}
	require.Equal(t, "i32_v", f.key())
}

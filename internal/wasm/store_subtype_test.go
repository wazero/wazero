package wasm

import (
	"testing"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

func mkStore() *Store {
	return NewStore(api.CoreFeaturesV2, nil)
}

func TestStore_IsSubtype_NoSupertype(t *testing.T) {
	s := mkStore()
	ts := []FunctionType{
		{Form: CompositeFormStruct, Fields: []FieldType{{ValueType: ValueTypeI32}}},
		{Form: CompositeFormStruct, Fields: []FieldType{{ValueType: ValueTypeI64}}},
	}
	ids, err := s.GetFunctionTypeIDs(ts)
	require.NoError(t, err)
	require.True(t, s.IsSubtype(ids[0], ids[0]))
	require.False(t, s.IsSubtype(ids[0], ids[1]))
	require.True(t, s.IsResolvedType(ids[0]))
	require.Equal(t, CompositeFormStruct, s.TypeForm(ids[0]))
}

func TestStore_IsSubtype_DeclaredSupertype(t *testing.T) {
	s := mkStore()
	zero := uint32(0)
	ts := []FunctionType{
		// type 0: base struct (no supertype).
		{Form: CompositeFormStruct, Fields: []FieldType{{ValueType: ValueTypeI32}}},
		// type 1: struct extending type 0.
		{
			Form:           CompositeFormStruct,
			Fields:         []FieldType{{ValueType: ValueTypeI32}, {ValueType: ValueTypeI64}},
			SuperTypeIndex: &zero,
			Final:          true,
		},
	}
	ids, err := s.GetFunctionTypeIDs(ts)
	require.NoError(t, err)
	// Subtype chain: type 1 <: type 0.
	require.True(t, s.IsSubtype(ids[1], ids[0]))
	require.False(t, s.IsSubtype(ids[0], ids[1]))
	require.True(t, s.IsSubtype(ids[1], ids[1]))
}

func TestStore_TypeForm_ArrayForm(t *testing.T) {
	s := mkStore()
	ts := []FunctionType{
		{Form: CompositeFormArray, ArrayField: FieldType{ValueType: ValueTypeI32, Mutable: true}},
	}
	ids, err := s.GetFunctionTypeIDs(ts)
	require.NoError(t, err)
	require.Equal(t, CompositeFormArray, s.TypeForm(ids[0]))
}

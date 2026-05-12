package binary

import (
	"bytes"
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/binaryencoding"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// TestEncodeDecodeRoundTrip_Composite verifies that the test-only encoder
// in internal/testing/binaryencoding and the production decoder in
// internal/wasm/binary agree on the wire format for all composite forms
// (func, struct, array), the sub / sub-final wrappers, and rec groups.
func TestEncodeDecodeRoundTrip_Composite(t *testing.T) {
	supIdx := uint32(0)

	tests := []struct {
		name string
		in   []wasm.FunctionType
	}{
		{
			name: "func shorthand",
			in: []wasm.FunctionType{
				{Form: wasm.CompositeFormFunc, Params: []wasm.ValueType{wasm.ValueTypeI32}, Results: []wasm.ValueType{wasm.ValueTypeI64}, Final: true},
			},
		},
		{
			name: "struct shorthand const+mut",
			in: []wasm.FunctionType{
				{
					Form: wasm.CompositeFormStruct,
					Fields: []wasm.FieldType{
						{ValueType: wasm.ValueTypeI32},
						{ValueType: wasm.ValueTypeI64, Mutable: true},
					},
					Final: true,
				},
			},
		},
		{
			name: "struct shorthand packed",
			in: []wasm.FunctionType{
				{
					Form: wasm.CompositeFormStruct,
					Fields: []wasm.FieldType{
						{Packed: wasm.PackedTypeI8, Mutable: true},
						{Packed: wasm.PackedTypeI16},
					},
					Final: true,
				},
			},
		},
		{
			name: "struct empty",
			in: []wasm.FunctionType{
				{Form: wasm.CompositeFormStruct, Final: true},
			},
		},
		{
			name: "array shorthand",
			in: []wasm.FunctionType{
				{Form: wasm.CompositeFormArray, ArrayField: wasm.FieldType{ValueType: wasm.ValueTypeI32, Mutable: true}, Final: true},
			},
		},
		{
			name: "array packed",
			in: []wasm.FunctionType{
				{Form: wasm.CompositeFormArray, ArrayField: wasm.FieldType{Packed: wasm.PackedTypeI8}, Final: true},
			},
		},
		{
			name: "sub final with supertype",
			in: []wasm.FunctionType{
				{Form: wasm.CompositeFormFunc, Params: []wasm.ValueType{wasm.ValueTypeI32}, Final: true},
				{
					Form:           wasm.CompositeFormFunc,
					Params:         []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32},
					SuperTypeIndex: &supIdx,
					Final:          true,
				},
			},
		},
		{
			name: "sub (non-final) struct with supertype",
			in: []wasm.FunctionType{
				{Form: wasm.CompositeFormStruct, Fields: []wasm.FieldType{{ValueType: wasm.ValueTypeI32}}, Final: true},
				{
					Form:           wasm.CompositeFormStruct,
					Fields:         []wasm.FieldType{{ValueType: wasm.ValueTypeI32}, {ValueType: wasm.ValueTypeI64}},
					SuperTypeIndex: &supIdx,
					Final:          false,
				},
			},
		},
		{
			name: "rec group of two structs",
			in: []wasm.FunctionType{
				{
					Form:             wasm.CompositeFormStruct,
					Fields:           []wasm.FieldType{{ValueType: wasm.ValueTypeI32}},
					RecGroupSize:     2,
					RecGroupPosition: 0,
					Final:            true,
				},
				{
					Form:             wasm.CompositeFormStruct,
					Fields:           []wasm.FieldType{{ValueType: wasm.ValueTypeI64}},
					RecGroupSize:     2,
					RecGroupPosition: 1,
					Final:            true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := binaryencoding.EncodeTypeSection(tt.in)
			r := bytes.NewReader(payload)
			got, err := decodeTypeSection(gcFeatures, r)
			require.NoError(t, err)
			require.Equal(t, len(tt.in), len(got))
			for i, want := range tt.in {
				require.Equal(t, want.Form, got[i].Form, "type[%d].Form", i)
				require.Equal(t, want.Params, got[i].Params, "type[%d].Params", i)
				require.Equal(t, want.Results, got[i].Results, "type[%d].Results", i)
				require.Equal(t, want.Fields, got[i].Fields, "type[%d].Fields", i)
				require.Equal(t, want.ArrayField, got[i].ArrayField, "type[%d].ArrayField", i)
				require.Equal(t, want.Final, got[i].Final, "type[%d].Final", i)
				require.Equal(t, want.RecGroupSize, got[i].RecGroupSize, "type[%d].RecGroupSize", i)
				require.Equal(t, want.RecGroupPosition, got[i].RecGroupPosition, "type[%d].RecGroupPosition", i)
				if want.SuperTypeIndex == nil {
					require.Nil(t, got[i].SuperTypeIndex, "type[%d].SuperTypeIndex", i)
				} else {
					require.NotNil(t, got[i].SuperTypeIndex)
					require.Equal(t, *want.SuperTypeIndex, *got[i].SuperTypeIndex)
				}
			}
		})
	}
}

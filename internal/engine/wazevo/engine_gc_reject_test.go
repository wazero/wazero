package wazevo

import (
	"context"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// TestEngine_CompileModule_RejectsGCTypes asserts that the optimizing
// compiler refuses to compile any module containing struct or array
// composite types in its type section. The runtime is expected to surface
// this so callers fall back to the interpreter rather than silently
// miscompile.
func TestEngine_CompileModule_RejectsGCTypes(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		mod  *wasm.Module
	}{
		{
			name: "struct type",
			mod: &wasm.Module{
				TypeSection: []wasm.FunctionType{
					{
						Form: wasm.CompositeFormStruct,
						Fields: []wasm.FieldType{
							{ValueType: wasm.ValueTypeI32, Mutable: true},
						},
					},
				},
				ID: wasm.ModuleID{},
			},
		},
		{
			name: "array type",
			mod: &wasm.Module{
				TypeSection: []wasm.FunctionType{
					{
						Form:       wasm.CompositeFormArray,
						ArrayField: wasm.FieldType{ValueType: wasm.ValueTypeI32, Mutable: true},
					},
				},
				ID: wasm.ModuleID{0xa},
			},
		},
		{
			name: "func then struct",
			mod: &wasm.Module{
				TypeSection: []wasm.FunctionType{
					{Form: wasm.CompositeFormFunc},
					{
						Form: wasm.CompositeFormStruct,
						Fields: []wasm.FieldType{
							{ValueType: wasm.ValueTypeI64},
						},
					},
				},
				ID: wasm.ModuleID{0xb},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEngine(ctx, 0, nil).(*engine)
			err := e.CompileModule(ctx, tt.mod, nil, false)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), "wasm-gc"), "error should mention wasm-gc: %v", err)
		})
	}
}

// TestEngine_CompileModule_AcceptsFuncOnly asserts that a module whose
// type section contains only ordinary function types still compiles
// (regression guard around the GC rejection check).
func TestEngine_CompileModule_AcceptsFuncOnly(t *testing.T) {
	ctx := context.Background()
	e := NewEngine(ctx, 0, nil).(*engine)
	mod := &wasm.Module{
		TypeSection: []wasm.FunctionType{
			{Form: wasm.CompositeFormFunc, Params: []wasm.ValueType{wasm.ValueTypeI32}, Results: []wasm.ValueType{wasm.ValueTypeI32}},
		},
		FunctionSection: []wasm.Index{0},
		CodeSection: []wasm.Code{
			{Body: []byte{wasm.OpcodeLocalGet, 0x00, wasm.OpcodeEnd}},
		},
		ID: wasm.ModuleID{0xc},
	}
	err := e.CompileModule(ctx, mod, nil, false)
	require.NoError(t, err)
}

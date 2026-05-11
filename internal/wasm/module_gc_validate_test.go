package wasm

import (
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

// gcFeatureBit mirrors experimental.CoreFeaturesGC (= api.CoreFeatureSIMD << 5),
// declared inline here because internal/wasm can't import experimental
// (circular).
const gcFeatureBit = api.CoreFeatureSIMD << 5

func TestValidateTypeSection_FuncAlwaysAllowed(t *testing.T) {
	m := &Module{
		TypeSection: []FunctionType{
			{Form: CompositeFormFunc, Params: []ValueType{ValueTypeI32}, Results: []ValueType{ValueTypeI32}},
		},
	}
	// Even with no GC feature flag, func types are accepted.
	require.NoError(t, m.validateTypeSection(api.CoreFeaturesV2))
}

func TestValidateTypeSection_StructRequiresGC(t *testing.T) {
	m := &Module{
		TypeSection: []FunctionType{
			{Form: CompositeFormStruct, Fields: []FieldType{{ValueType: ValueTypeI32}}},
		},
	}
	err := m.validateTypeSection(api.CoreFeaturesV2)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "struct"), "error should mention struct: %v", err)
	require.True(t, strings.Contains(err.Error(), "disabled"), "error should mention disabled: %v", err)

	// With GC enabled, accepted.
	require.NoError(t, m.validateTypeSection(api.CoreFeaturesV2|gcFeatureBit))
}

func TestValidateTypeSection_ArrayRequiresGC(t *testing.T) {
	m := &Module{
		TypeSection: []FunctionType{
			{Form: CompositeFormArray, ArrayField: FieldType{ValueType: ValueTypeI32}},
		},
	}
	err := m.validateTypeSection(api.CoreFeaturesV2)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "array"), "error should mention array: %v", err)

	require.NoError(t, m.validateTypeSection(api.CoreFeaturesV2|gcFeatureBit))
}

func TestValidateTypeSection_SuperTypeRequiresGC(t *testing.T) {
	idx := uint32(0)
	m := &Module{
		TypeSection: []FunctionType{
			{Form: CompositeFormFunc},
			{Form: CompositeFormFunc, SuperTypeIndex: &idx},
		},
	}
	err := m.validateTypeSection(api.CoreFeaturesV2)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "supertype"), "error should mention supertype: %v", err)

	require.NoError(t, m.validateTypeSection(api.CoreFeaturesV2|gcFeatureBit))
}

func TestValidateTypeSection_SuperTypeOutOfRange(t *testing.T) {
	idx := uint32(7) // out of range — only 1 type in TypeSection
	m := &Module{
		TypeSection: []FunctionType{
			{Form: CompositeFormStruct, SuperTypeIndex: &idx},
		},
	}
	err := m.validateTypeSection(api.CoreFeaturesV2 | gcFeatureBit)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "out of range"), "error should mention out of range: %v", err)
}

func TestValidateTypeSection_UnknownForm(t *testing.T) {
	m := &Module{
		TypeSection: []FunctionType{
			{Form: CompositeForm(99)},
		},
	}
	err := m.validateTypeSection(api.CoreFeaturesV2 | gcFeatureBit)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unknown composite form"), "got: %v", err)
}

package binary

import (
	"bytes"
	"testing"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

const gcFeatures = api.CoreFeaturesV2 | experimental.CoreFeaturesGC

func TestDecodeTypeSection_RecGroupShorthand(t *testing.T) {
	// Two standalone shorthand func types — pre-GC behavior unchanged.
	in := []byte{
		0x02,
		0x60, 0x00, 0x01, 0x7F,
		0x60, 0x00, 0x01, 0x7F,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 2, len(got))
	require.Equal(t, wasm.CompositeFormFunc, got[0].Form)
	require.Equal(t, 0, got[0].RecGroupSize)
	require.Equal(t, []wasm.ValueType{wasm.ValueTypeI32}, got[0].Results)
}

func TestDecodeTypeSection_RecGroup(t *testing.T) {
	in := []byte{
		0x01,
		0x4E, 0x02,
		0x60, 0x00, 0x01, 0x7F,
		0x60, 0x01, 0x7E, 0x00,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 2, len(got))
	require.Equal(t, 2, got[0].RecGroupSize)
	require.Equal(t, 0, got[0].RecGroupPosition)
	require.Equal(t, 2, got[1].RecGroupSize)
	require.Equal(t, 1, got[1].RecGroupPosition)
	require.Equal(t, []wasm.ValueType{wasm.ValueTypeI32}, got[0].Results)
	require.Equal(t, []wasm.ValueType{wasm.ValueTypeI64}, got[1].Params)
}

func TestDecodeTypeSection_SubFinalNoSupers(t *testing.T) {
	in := []byte{
		0x01,
		0x4F, 0x00,
		0x60, 0x00, 0x01, 0x7F,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 1, len(got))
	require.Equal(t, wasm.CompositeFormFunc, got[0].Form)
	require.True(t, got[0].Final)
	require.Nil(t, got[0].SuperTypeIndex)
}

func TestDecodeTypeSection_SubWithSuper(t *testing.T) {
	in := []byte{
		0x02,
		0x60, 0x00, 0x00,
		0x50, 0x01, 0x00,
		0x5F, 0x01, 0x7F, 0x01,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 2, len(got))
	require.Equal(t, wasm.CompositeFormStruct, got[1].Form)
	require.False(t, got[1].Final)
	require.NotNil(t, got[1].SuperTypeIndex)
	require.Equal(t, uint32(0), *got[1].SuperTypeIndex)
	require.Equal(t, 1, len(got[1].Fields))
	require.Equal(t, wasm.ValueTypeI32, got[1].Fields[0].ValueType)
	require.True(t, got[1].Fields[0].Mutable)
}

func TestDecodeTypeSection_StructShorthand(t *testing.T) {
	in := []byte{
		0x01,
		0x5F, 0x02, 0x7F, 0x00, 0x7E, 0x01,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 1, len(got))
	require.Equal(t, wasm.CompositeFormStruct, got[0].Form)
	require.Equal(t, 2, len(got[0].Fields))
	require.Equal(t, wasm.ValueTypeI32, got[0].Fields[0].ValueType)
	require.False(t, got[0].Fields[0].Mutable)
	require.Equal(t, wasm.ValueTypeI64, got[0].Fields[1].ValueType)
	require.True(t, got[0].Fields[1].Mutable)
}

func TestDecodeTypeSection_StructWithPackedFields(t *testing.T) {
	in := []byte{
		0x01,
		0x5F, 0x02, 0x78, 0x01, 0x77, 0x00,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 1, len(got))
	require.Equal(t, wasm.PackedTypeI8, got[0].Fields[0].Packed)
	require.True(t, got[0].Fields[0].Mutable)
	require.Equal(t, wasm.PackedTypeI16, got[0].Fields[1].Packed)
	require.False(t, got[0].Fields[1].Mutable)
}

func TestDecodeTypeSection_EmptyStruct(t *testing.T) {
	in := []byte{
		0x01,
		0x5F, 0x00,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 1, len(got))
	require.Equal(t, wasm.CompositeFormStruct, got[0].Form)
	require.Nil(t, got[0].Fields)
}

func TestDecodeTypeSection_ArrayShorthand(t *testing.T) {
	in := []byte{
		0x01,
		0x5E, 0x7F, 0x01,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 1, len(got))
	require.Equal(t, wasm.CompositeFormArray, got[0].Form)
	require.Equal(t, wasm.ValueTypeI32, got[0].ArrayField.ValueType)
	require.True(t, got[0].ArrayField.Mutable)
}

func TestDecodeTypeSection_ArrayPacked(t *testing.T) {
	in := []byte{
		0x01,
		0x5E, 0x78, 0x00,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 1, len(got))
	require.Equal(t, wasm.CompositeFormArray, got[0].Form)
	require.Equal(t, wasm.PackedTypeI8, got[0].ArrayField.Packed)
	require.False(t, got[0].ArrayField.Mutable)
}

func TestDecodeTypeSection_MixedRecGroup(t *testing.T) {
	in := []byte{
		0x01,
		0x4E, 0x02,
		0x5F, 0x01, 0x7F, 0x01,
		0x5E, 0x7E, 0x00,
	}
	r := bytes.NewReader(in)
	got, err := decodeTypeSection(gcFeatures, r)
	require.NoError(t, err)
	require.Equal(t, 2, len(got))
	require.Equal(t, wasm.CompositeFormStruct, got[0].Form)
	require.Equal(t, wasm.CompositeFormArray, got[1].Form)
	require.Equal(t, 2, got[0].RecGroupSize)
	require.Equal(t, 0, got[0].RecGroupPosition)
	require.Equal(t, 2, got[1].RecGroupSize)
	require.Equal(t, 1, got[1].RecGroupPosition)
}

func TestDecodeTypeSection_TooManySupers(t *testing.T) {
	in := []byte{
		0x01,
		0x50, 0x02, 0x00, 0x01,
		0x60, 0x00, 0x00,
	}
	r := bytes.NewReader(in)
	_, err := decodeTypeSection(gcFeatures, r)
	require.Error(t, err)
}

func TestDecodeTypeSection_InvalidLeadingByte(t *testing.T) {
	in := []byte{
		0x01,
		0x00,
	}
	r := bytes.NewReader(in)
	_, err := decodeTypeSection(gcFeatures, r)
	require.Error(t, err)
}

func TestDecodeTypeSection_InvalidMutability(t *testing.T) {
	in := []byte{
		0x01,
		0x5F, 0x01, 0x7F, 0x02,
	}
	r := bytes.NewReader(in)
	_, err := decodeTypeSection(gcFeatures, r)
	require.Error(t, err)
}

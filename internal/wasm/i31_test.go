package wasm

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestNewI31Ref_MasksHighBit(t *testing.T) {
	tests := []struct {
		in   uint32
		want uint32
	}{
		{0, 0},
		{1, 1},
		{0x7FFFFFFF, 0x7FFFFFFF},
		{0x80000000, 0},          // bit 31 stripped
		{0xFFFFFFFF, 0x7FFFFFFF}, // all bits set => bit 31 stripped
		{0xCAFEBABE, 0x4AFEBABE}, // bit 31 stripped, rest preserved
	}
	for _, tt := range tests {
		got := NewI31Ref(tt.in)
		require.Equal(t, tt.want, got.bits)
		require.Equal(t, tt.want, got.UnsignedI32())
	}
}

func TestI31Ref_SignedI32_SignExtend(t *testing.T) {
	tests := []struct {
		name string
		in   uint32 // arg to NewI31Ref
		want int32  // expected SignedI32 result
	}{
		{"zero", 0, 0},
		{"one", 1, 1},
		{"max positive 31-bit (bit 30 unset)", 0x3FFFFFFF, 0x3FFFFFFF},
		{"min negative 31-bit (bit 30 set)", 0x40000000, -0x40000000},
		{"-1 (all 31 bits set)", 0x7FFFFFFF, -1},
		{"input has bit 31 set, masked off then sign-extended", 0xC0000000, -0x40000000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewI31Ref(tt.in)
			require.Equal(t, tt.want, r.SignedI32())
		})
	}
}

func TestI31Ref_UnsignedI32_NoSignExtend(t *testing.T) {
	// Unsigned read never propagates bit 30 into bit 31; the result is
	// always in [0, 2^31 - 1].
	r := NewI31Ref(0x7FFFFFFF)
	require.Equal(t, uint32(0x7FFFFFFF), r.UnsignedI32())

	r = NewI31Ref(0xFFFFFFFF) // top bit will be masked
	require.Equal(t, uint32(0x7FFFFFFF), r.UnsignedI32())

	r = NewI31Ref(0x40000000)
	require.Equal(t, uint32(0x40000000), r.UnsignedI32())
}

func TestI31RefFromInt32(t *testing.T) {
	// Negative int32 values should round-trip via i31 sign-extension.
	r := I31RefFromInt32(-1)
	require.Equal(t, int32(-1), r.SignedI32())

	r = I31RefFromInt32(-0x40000000)
	require.Equal(t, int32(-0x40000000), r.SignedI32())

	// int32 with bit 31 set but not representable as 31-bit signed: bit 31
	// is stripped (NewI31Ref behavior), so the result differs from the
	// input. We construct the value through a uint32 variable so the Go
	// compiler accepts the cast — bare int32(0x80000000) is a constant
	// expression that overflows int32.
	var bitPattern uint32 = 0x80000000 // = -2^31 reinterpreted, not representable as 31-bit signed
	r = I31RefFromInt32(int32(bitPattern))
	// Strip bit 31 (becomes 0), so SignedI32 is 0.
	require.Equal(t, int32(0), r.SignedI32())

	bitPattern = 0x80000001
	r = I31RefFromInt32(int32(bitPattern)) // ditto, low bits 0x00000001
	require.Equal(t, int32(1), r.SignedI32())
}

func TestI31Ref_Equals(t *testing.T) {
	a := NewI31Ref(42)
	b := NewI31Ref(42)
	c := NewI31Ref(43)
	require.True(t, a.Equals(b)) // different pointers, same value
	require.False(t, a.Equals(c))

	// Nil semantics: only two nils are equal; nil vs non-nil is unequal.
	var nilRef *I31Ref
	require.True(t, nilRef.Equals(nil))
	require.False(t, a.Equals(nilRef))
	require.False(t, nilRef.Equals(a))
}

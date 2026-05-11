package experimental_test

import (
	"context"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/internal/testing/binaryencoding"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// TestGC_I31 builds and runs a minimal wasm-gc module that exercises
// ref.i31 followed by i31.get_s and i31.get_u. Confirms that the
// interpreter's Phase 5 i31 support produces spec-correct results
// end-to-end.
func TestGC_I31(t *testing.T) {
	ctx := context.Background()

	// Build a module with three exported functions:
	//   getS(i32) -> i32   = i31.get_s(ref.i31(local.get 0))
	//   getU(i32) -> i32   = i31.get_u(ref.i31(local.get 0))
	//   eq(i32, i32) -> i32 = ref.eq(ref.i31(local.get 0), ref.i31(local.get 1))
	//
	// (Note: ref.eq on two freshly-allocated i31 refs uses POINTER
	// equality in the current interpreter, so two distinct allocations
	// compare unequal even when their numeric values match. The eq test
	// here verifies the pointer-equality semantics, not value-equality.
	// Value-equality on i31 is a Phase 5b refinement.)

	getsBody := []byte{
		wasm.OpcodeLocalGet, 0x00, // local.get 0
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCRefI31),
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCI31GetS),
		wasm.OpcodeEnd,
	}
	getuBody := []byte{
		wasm.OpcodeLocalGet, 0x00,
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCRefI31),
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCI31GetU),
		wasm.OpcodeEnd,
	}

	mod := &wasm.Module{
		TypeSection: []wasm.FunctionType{
			{Form: wasm.CompositeFormFunc, Params: []wasm.ValueType{wasm.ValueTypeI32}, Results: []wasm.ValueType{wasm.ValueTypeI32}},
		},
		FunctionSection: []wasm.Index{0, 0},
		CodeSection: []wasm.Code{
			{Body: getsBody},
			{Body: getuBody},
		},
		ExportSection: []wasm.Export{
			{Name: "getS", Type: wasm.ExternTypeFunc, Index: 0},
			{Name: "getU", Type: wasm.ExternTypeFunc, Index: 1},
		},
	}

	bin := binaryencoding.EncodeModule(mod)

	cfg := wazero.NewRuntimeConfigInterpreter().
		WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesGC)
	r := wazero.NewRuntimeWithConfig(ctx, cfg)
	defer r.Close(ctx)

	instance, err := r.Instantiate(ctx, bin)
	require.NoError(t, err)

	getS := instance.ExportedFunction("getS")
	getU := instance.ExportedFunction("getU")

	// i31.get_s sign-extends bit 30 to bit 31.
	tests := []struct {
		name    string
		fn      api.Function
		in      uint32
		want    uint32
		wantI32 int32
	}{
		// Small positive values round-trip.
		{"getS(0)", getS, 0, 0, 0},
		{"getS(42)", getS, 42, 42, 42},
		{"getS(0x3FFFFFFF max positive 31-bit)", getS, 0x3FFFFFFF, 0x3FFFFFFF, 0x3FFFFFFF},
		// Bit 30 set => sign-extended negative.
		{"getS(0x40000000)", getS, 0x40000000, 0xC0000000, -0x40000000},
		// Bit 31 of input is stripped before storage.
		{"getS(0x80000000)", getS, 0x80000000, 0, 0},
		{"getS(0xFFFFFFFF)", getS, 0xFFFFFFFF, 0xFFFFFFFF, -1},

		// Unsigned: no sign extension.
		{"getU(0x40000000)", getU, 0x40000000, 0x40000000, 0},
		{"getU(0xFFFFFFFF)", getU, 0xFFFFFFFF, 0x7FFFFFFF, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.fn.Call(ctx, uint64(tt.in))
			require.NoError(t, err)
			gotU := uint32(res[0])
			require.Equal(t, tt.want, gotU, "got %#x, want %#x", gotU, tt.want)
		})
	}
}

// TestGC_I31RefEq verifies that ref.eq on two i31 refs compares by VALUE
// (per spec), not by pointer identity. With the tagged-uintptr encoding,
// two independent ref.i31 invocations with the same input produce
// identical bit patterns, so uint64 equality gives the right answer.
func TestGC_I31RefEq(t *testing.T) {
	ctx := context.Background()

	// sameValue(i32) -> i32: ref.eq(ref.i31(x), ref.i31(x))  -> 1
	// diffValue(i32, i32) -> i32: ref.eq(ref.i31(x), ref.i31(y))
	sameValue := []byte{
		wasm.OpcodeLocalGet, 0x00,
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCRefI31),
		wasm.OpcodeLocalGet, 0x00,
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCRefI31),
		wasm.OpcodeRefEq,
		wasm.OpcodeEnd,
	}
	diffValue := []byte{
		wasm.OpcodeLocalGet, 0x00,
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCRefI31),
		wasm.OpcodeLocalGet, 0x01,
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCRefI31),
		wasm.OpcodeRefEq,
		wasm.OpcodeEnd,
	}

	mod := &wasm.Module{
		TypeSection: []wasm.FunctionType{
			{Form: wasm.CompositeFormFunc, Params: []wasm.ValueType{wasm.ValueTypeI32}, Results: []wasm.ValueType{wasm.ValueTypeI32}},
			{Form: wasm.CompositeFormFunc, Params: []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32}, Results: []wasm.ValueType{wasm.ValueTypeI32}},
		},
		FunctionSection: []wasm.Index{0, 1},
		CodeSection: []wasm.Code{
			{Body: sameValue},
			{Body: diffValue},
		},
		ExportSection: []wasm.Export{
			{Name: "sameValue", Type: wasm.ExternTypeFunc, Index: 0},
			{Name: "diffValue", Type: wasm.ExternTypeFunc, Index: 1},
		},
	}
	bin := binaryencoding.EncodeModule(mod)

	cfg := wazero.NewRuntimeConfigInterpreter().
		WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesGC)
	r := wazero.NewRuntimeWithConfig(ctx, cfg)
	defer r.Close(ctx)

	instance, err := r.Instantiate(ctx, bin)
	require.NoError(t, err)

	t.Run("ref.eq(i31(42), i31(42)) = 1 (value-equal)", func(t *testing.T) {
		res, err := instance.ExportedFunction("sameValue").Call(ctx, 42)
		require.NoError(t, err)
		require.Equal(t, int32(1), api.DecodeI32(res[0]))
	})

	t.Run("ref.eq(i31(42), i31(43)) = 0", func(t *testing.T) {
		res, err := instance.ExportedFunction("diffValue").Call(ctx, 42, 43)
		require.NoError(t, err)
		require.Equal(t, int32(0), api.DecodeI32(res[0]))
	})

	t.Run("ref.eq(i31(0), i31(0)) = 1 (zero is a valid non-null i31)", func(t *testing.T) {
		res, err := instance.ExportedFunction("sameValue").Call(ctx, 0)
		require.NoError(t, err)
		require.Equal(t, int32(1), api.DecodeI32(res[0]))
	})

	t.Run("ref.eq(i31(-1), i31(-1)) = 1 (sign preserved across packing)", func(t *testing.T) {
		// Both refs encode the same 31-bit pattern (0x7FFFFFFF), so eq is true.
		res, err := instance.ExportedFunction("sameValue").Call(ctx, uint64(uint32(0xFFFFFFFF)))
		require.NoError(t, err)
		require.Equal(t, int32(1), api.DecodeI32(res[0]))
	})
}

func TestGC_RefAsNonNull(t *testing.T) {
	ctx := context.Background()

	// Build a module with two exported functions:
	//   passthrough(i32) -> i32  : ref.i31(local.get 0); ref.as_non_null; i31.get_s
	//   trapOnNull() -> i32      : ref.null any; ref.as_non_null; i31.get_s
	passBody := []byte{
		wasm.OpcodeLocalGet, 0x00,
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCRefI31),
		wasm.OpcodeRefAsNonNull,
		wasm.OpcodeGCPrefix, byte(wasm.OpcodeGCI31GetS),
		wasm.OpcodeEnd,
	}
	// ref.null with heap type "any" (0x6E); ref.as_non_null traps on the
	// null reference; the i32.const + drop branch never executes but is
	// required to satisfy validation that the function returns i32.
	trapBody := []byte{
		wasm.OpcodeRefNull, 0x6E,
		wasm.OpcodeRefAsNonNull,
		wasm.OpcodeDrop,
		wasm.OpcodeI32Const, 0x00,
		wasm.OpcodeEnd,
	}

	mod := &wasm.Module{
		TypeSection: []wasm.FunctionType{
			{Form: wasm.CompositeFormFunc, Params: []wasm.ValueType{wasm.ValueTypeI32}, Results: []wasm.ValueType{wasm.ValueTypeI32}},
			{Form: wasm.CompositeFormFunc, Results: []wasm.ValueType{wasm.ValueTypeI32}},
		},
		FunctionSection: []wasm.Index{0, 1},
		CodeSection: []wasm.Code{
			{Body: passBody},
			{Body: trapBody},
		},
		ExportSection: []wasm.Export{
			{Name: "passthrough", Type: wasm.ExternTypeFunc, Index: 0},
			{Name: "trapOnNull", Type: wasm.ExternTypeFunc, Index: 1},
		},
	}
	bin := binaryencoding.EncodeModule(mod)

	cfg := wazero.NewRuntimeConfigInterpreter().
		WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesGC)
	r := wazero.NewRuntimeWithConfig(ctx, cfg)
	defer r.Close(ctx)

	instance, err := r.Instantiate(ctx, bin)
	require.NoError(t, err)

	t.Run("passthrough(7)", func(t *testing.T) {
		res, err := instance.ExportedFunction("passthrough").Call(ctx, 7)
		require.NoError(t, err)
		require.Equal(t, uint32(7), uint32(res[0]))
	})

	t.Run("trapOnNull traps", func(t *testing.T) {
		_, err := instance.ExportedFunction("trapOnNull").Call(ctx)
		require.Error(t, err)
	})
}

// TestGC_Struct exercises struct.new, struct.get, struct.set, and
// struct.new_default end-to-end. Field schema: two-field struct with one
// const i32 and one mut i64.

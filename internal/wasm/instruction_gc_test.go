package wasm

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestGCInstructionName(t *testing.T) {
	// Spot-check a few names; the full mapping is enumerated to catch any
	// drift between the constants table and the name map.
	require.Equal(t, "struct.new", GCInstructionName(OpcodeGCStructNew))
	require.Equal(t, "array.len", GCInstructionName(OpcodeGCArrayLen))
	require.Equal(t, "ref.i31", GCInstructionName(OpcodeGCRefI31))
	require.Equal(t, "ref.cast null", GCInstructionName(OpcodeGCRefCastNull))
	require.Equal(t, "br_on_cast", GCInstructionName(OpcodeGCBrOnCast))
	require.Equal(t, "any.convert_extern", GCInstructionName(OpcodeGCAnyConvertExtern))

	// Unknown sub-opcode returns empty string.
	require.Equal(t, "", GCInstructionName(OpcodeGC(0xFFFF)))
}

func TestGCInstructionNameTableComplete(t *testing.T) {
	// Every defined GC sub-opcode 0x00..0x1E must have a non-empty name.
	for op := OpcodeGCStructNew; op <= OpcodeGCI31GetU; op++ {
		require.NotEqual(t, "", GCInstructionName(op),
			"missing name entry for OpcodeGC sub-opcode 0x%x", op)
	}
}

func TestGCPrefixAndOpcodes(t *testing.T) {
	// Confirm the 0xfb prefix and that typed-funcref opcodes occupy their
	// spec-mandated single-byte top-level slots.
	require.Equal(t, Opcode(0xfb), OpcodeGCPrefix)
	require.Equal(t, Opcode(0xd3), OpcodeRefEq)
	require.Equal(t, Opcode(0xd4), OpcodeRefAsNonNull)
	require.Equal(t, Opcode(0xd5), OpcodeBrOnNull)
	require.Equal(t, Opcode(0xd6), OpcodeBrOnNonNull)
	require.Equal(t, Opcode(0x14), OpcodeCallRef)
	require.Equal(t, Opcode(0x15), OpcodeReturnCallRef)
}

func TestBrOnCastFlags(t *testing.T) {
	require.Equal(t, byte(0x01), BrOnCastFlagSrcNullable)
	require.Equal(t, byte(0x02), BrOnCastFlagDstNullable)
}

func TestInstructionName_TypedFuncRef(t *testing.T) {
	// The top-level instruction name map should include the typed-funcref names.
	require.Equal(t, "ref.eq", InstructionName(OpcodeRefEq))
	require.Equal(t, "ref.as_non_null", InstructionName(OpcodeRefAsNonNull))
	require.Equal(t, "br_on_null", InstructionName(OpcodeBrOnNull))
	require.Equal(t, "br_on_non_null", InstructionName(OpcodeBrOnNonNull))
	require.Equal(t, "call_ref", InstructionName(OpcodeCallRef))
	require.Equal(t, "return_call_ref", InstructionName(OpcodeReturnCallRef))
}

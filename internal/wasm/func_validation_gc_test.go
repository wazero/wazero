package wasm

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

// validateGCBody is a small helper that runs validateFunction over a body
// against a no-args, no-results func type. Returns the error (if any).
func validateGCBody(t *testing.T, body []byte) error {
	t.Helper()
	m := &Module{
		TypeSection:     []FunctionType{{}},
		FunctionSection: []Index{0},
		CodeSection:     []Code{{Body: body}},
	}
	return m.validateFunction(&stacks{}, api.CoreFeaturesV2,
		0, []Index{0}, nil, nil, nil, nil, nil, bytes.NewReader(nil))
}

func TestValidateFunction_GCOpcodeMessage(t *testing.T) {
	// All GC sub-opcodes are now implemented. The placeholder test
	// remains as documentation; it exercises no live cases.
	tests := []struct {
		name      string
		body      []byte
		expectSub string
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGCBody(t, tt.body)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), tt.expectSub),
				"error should mention %q: %v", tt.expectSub, err)
			require.True(t, strings.Contains(err.Error(), "not yet supported"),
				"error should be the actionable Phase 5 message: %v", err)
		})
	}
}

func TestValidateFunction_UnknownGCSubOpcode(t *testing.T) {
	// 0xfb 0x7f — sub-opcode 0x7f is not assigned in the GC proposal.
	err := validateGCBody(t, []byte{OpcodeGCPrefix, 0x7F, OpcodeEnd})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unknown GC sub-opcode"),
		"error should mention unknown GC sub-opcode: %v", err)
}

func TestValidateFunction_TruncatedGCPrefix(t *testing.T) {
	// 0xfb at end-of-body with no sub-opcode byte.
	err := validateGCBody(t, []byte{OpcodeGCPrefix})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "truncated GC instruction") ||
		strings.Contains(err.Error(), "cannot read GC sub-opcode"),
		"error should describe the truncation: %v", err)
}

func TestValidateFunction_TypedFuncRefOpcodeMessage(t *testing.T) {
	// Placeholder for remaining typed-funcref opcodes that have not yet
	// been implemented. The currently-implemented ones (ref.eq,
	// ref.as_non_null, br_on_null, br_on_non_null, call_ref,
	// return_call_ref) have been removed from this list.
	tests := []struct {
		name      string
		op        Opcode
		expectSub string
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGCBody(t, []byte{tt.op, OpcodeEnd})
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), tt.expectSub),
				"error should mention %q: %v", tt.expectSub, err)
			require.True(t, strings.Contains(err.Error(), "not yet supported"),
				"error should be the actionable Phase 5 message: %v", err)
		})
	}
}

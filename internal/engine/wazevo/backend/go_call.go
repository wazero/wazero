package backend

import "github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"

// GoFunctionCallRequiredStackSize returns the size of the stack required for the Go function call.
// argBegin is the index of the first argument in the signature which is not either execution context or module context.
func GoFunctionCallRequiredStackSize(sig *ssa.Signature, argBegin int) (ret, retUnaligned int64) {
	var paramNeededInBytes, resultNeededInBytes int64
	// We use uint64 for all basic types, except SIMD v128.
	for _, p := range sig.Params[argBegin:] {
		s := max(int64(p.Size()), 8)
		paramNeededInBytes += s
	}
	for _, r := range sig.Results {
		s := max(int64(r.Size()), 8)
		resultNeededInBytes += s
	}

	ret = max(paramNeededInBytes, resultNeededInBytes)
	retUnaligned = ret
	// Align to 16 bytes.
	ret = (ret + 15) &^ 15
	return
}

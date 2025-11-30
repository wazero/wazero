package binaryencoding

import (
	"github.com/tetratelabs/wazero/internal/wasm"
)

// EncodeMemory returns the wasm.Memory encoded in WebAssembly 1.0 (20191205) Binary Format.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-memory
func EncodeMemory(i *wasm.Memory) []byte {
	var maxPtr *uint64
	if i.IsMaxEncoded {
		maxVal := uint64(i.Max)
		maxPtr = &maxVal
	}
	return EncodeLimitsType(uint64(i.Min), maxPtr, i.IsShared, i.Is64)
}

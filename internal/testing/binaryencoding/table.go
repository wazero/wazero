package binaryencoding

import (
	"github.com/tetratelabs/wazero/internal/wasm"
)

// EncodeTable returns the wasm.Table encoded in WebAssembly 1.0 (20191205) Binary Format.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-table
func EncodeTable(i *wasm.Table) []byte {
	var maxPtr *uint64
	if i.Max != nil {
		mv := uint64(*i.Max)
		maxPtr = &mv
	}
	return append([]byte{i.Type}, EncodeLimitsType(uint64(i.Min), maxPtr, false, false)...)
}

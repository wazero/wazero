package binaryencoding

import (
	"fmt"

	"github.com/tetratelabs/wazero/internal/leb128"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// EncodeImport returns the wasm.Import encoded in WebAssembly 1.0 (20191205) Binary Format.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-import
func EncodeImport(i *wasm.Import) []byte {
	data := encodeSizePrefixed([]byte(i.Module))
	data = append(data, encodeSizePrefixed([]byte(i.Name))...)
	data = append(data, i.Type)
	switch i.Type {
	case wasm.ExternTypeFunc:
		data = append(data, leb128.EncodeUint32(i.DescFunc)...)
	case wasm.ExternTypeTable:
		data = append(data, wasm.RefTypeFuncref)
		var maxPtr *uint64
		if i.DescTable.Max != nil {
			mv := uint64(*i.DescTable.Max)
			maxPtr = &mv
		}
		data = append(data, EncodeLimitsType(uint64(i.DescTable.Min), maxPtr, false, false)...)
	case wasm.ExternTypeMemory:
		var maxPtr *uint64
		if i.DescMem.IsMaxEncoded {
			mv := uint64(i.DescMem.Max)
			maxPtr = &mv
		}
		data = append(data, EncodeLimitsType(uint64(i.DescMem.Min), maxPtr, i.DescMem.IsShared, i.DescMem.Is64)...)
	case wasm.ExternTypeGlobal:
		g := i.DescGlobal
		var mutable byte
		if g.Mutable {
			mutable = 1
		}
		data = append(data, g.ValType, mutable)
	default:
		panic(fmt.Errorf("invalid externtype: %s", wasm.ExternTypeName(i.Type)))
	}
	return data
}

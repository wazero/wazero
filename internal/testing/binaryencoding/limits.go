package binaryencoding

import (
	"github.com/tetratelabs/wazero/internal/leb128"
)

// EncodeLimitsType returns the `limitsType` (min, max) encoded in WebAssembly 1.0 (20191205) Binary Format.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#limits%E2%91%A6
//
// Extended in threads proposal: https://webassembly.github.io/threads/core/binary/types.html#limits
func EncodeLimitsType(min uint64, max *uint64, shared, is64 bool) []byte {
	var flag uint32
	if max != nil {
		flag = 0x01
	}
	if shared {
		flag |= 0x02
	}
	if is64 {
		flag |= 0x04
	}
	ret := append(leb128.EncodeUint32(flag), encodeLimitValue(min, is64)...)
	if max != nil {
		ret = append(ret, encodeLimitValue(*max, is64)...)
	}
	return ret
}

func encodeLimitValue(v uint64, is64 bool) []byte {
	if is64 {
		return leb128.EncodeUint64(v)
	}
	return leb128.EncodeUint32(uint32(v))
}

package binary

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
	"unsafe"

	"github.com/tetratelabs/wazero/internal/leb128"
	"github.com/tetratelabs/wazero/internal/wasm"
)

func decodeValueTypes(r *bytes.Reader, num uint32) ([]wasm.ValueType, error) {
	if num == 0 {
		return nil, nil
	}

	ret := make([]wasm.ValueType, 0, num)
	for i := uint32(0); i < num; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		switch b {
		case wasm.ValueTypeI32, wasm.ValueTypeF32, wasm.ValueTypeI64, wasm.ValueTypeF64,
			wasm.ValueTypeExternref, wasm.ValueTypeFuncref, wasm.ValueTypeV128,
			wasm.ValueTypeExnref:
			ret = append(ret, b)
		case wasm.RefPrefixNullable, wasm.RefPrefixNonNullable:
			ht, _, err := leb128.DecodeInt33AsInt64(r)
			if err != nil {
				return nil, fmt.Errorf("read ref heap type: %w", err)
			}
			// The following nullable refs are an alternative representation of the corresponding ref types:
			// - (ref null exn) is equivalent to exnref
			// - (ref null func) is equivalent to funcref
			// - (ref null extern) is equivalent to externref
			// See https://webassembly.github.io/gc/core/syntax/types.html#reference-types
			// Current limitation: we desugar NON-NULLABLE types to NULLABLE types internally.
			// This technically breaks type-checking in some cases, but we will fix this
			// when we introduce proper ref types.
			switch ht {
			case wasm.HeapTypeExn:
				ret = append(ret, wasm.ValueTypeExnref)
			case wasm.HeapTypeFunc:
				ret = append(ret, wasm.ValueTypeFuncref)
			case wasm.HeapTypeExtern:
				ret = append(ret, wasm.ValueTypeExternref)
			default: // concrete type index — treat as nullable funcref
				ret = append(ret, wasm.ValueTypeFuncref)
			}
		default:
			return nil, fmt.Errorf("invalid value type: %d", b)
		}
	}
	return ret, nil
}

// decodeUTF8 decodes a size prefixed string from the reader, returning it and the count of bytes read.
// contextFormat and contextArgs apply an error format when present
func decodeUTF8(r *bytes.Reader, contextFormat string, contextArgs ...interface{}) (string, uint32, error) {
	size, sizeOfSize, err := leb128.DecodeUint32(r)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read %s size: %w", fmt.Sprintf(contextFormat, contextArgs...), err)
	}

	if size == 0 {
		return "", uint32(sizeOfSize), nil
	}

	buf := make([]byte, size)
	if _, err = io.ReadFull(r, buf); err != nil {
		return "", 0, fmt.Errorf("failed to read %s: %w", fmt.Sprintf(contextFormat, contextArgs...), err)
	}

	if !utf8.Valid(buf) {
		return "", 0, fmt.Errorf("%s is not valid UTF-8", fmt.Sprintf(contextFormat, contextArgs...))
	}

	ret := unsafe.String(&buf[0], int(size))
	return ret, size + uint32(sizeOfSize), nil
}

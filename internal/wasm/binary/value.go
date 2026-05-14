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
		case wasm.ValueTypeI32.Kind(), wasm.ValueTypeF32.Kind(), wasm.ValueTypeI64.Kind(), wasm.ValueTypeF64.Kind(),
			wasm.ValueTypeExternref.Kind(), wasm.ValueTypeFuncref.Kind(), wasm.ValueTypeV128.Kind(),
			wasm.ValueTypeExnref.Kind(),
			// wasm-gc nullable abstract heap-type shorthand bytes.
			wasm.ValueTypeAnyref.Kind(), wasm.ValueTypeEqref.Kind(), wasm.ValueTypeI31ref.Kind(),
			wasm.ValueTypeStructref.Kind(), wasm.ValueTypeArrayref.Kind(), wasm.ValueTypeNullref.Kind(),
			wasm.ValueTypeNoFuncref.Kind(), wasm.ValueTypeNoExternref.Kind(), wasm.ValueTypeNoExnref.Kind():
			ret = append(ret, wasm.ValueType(b))
		case wasm.RefPrefixNullable, wasm.RefPrefixNonNullable:
			nullable := b == wasm.RefPrefixNullable
			ht, _, err := leb128.DecodeInt33AsInt64(r)
			if err != nil {
				return nil, fmt.Errorf("read ref heap type: %w", err)
			}
			vt, ok := decodeHeapType(ht, nullable)
			if !ok {
				return nil, fmt.Errorf("invalid heap type: %d", ht)
			}
			ret = append(ret, vt)
		default:
			return nil, fmt.Errorf("invalid value type: %d", b)
		}
	}
	return ret, nil
}

// decodeHeapType maps an s33-encoded heap type to its ValueType encoding,
// applying the supplied nullability. Non-negative values are concrete
// type indices; negative values are abstract heap-type bytes.
func decodeHeapType(ht int64, nullable bool) (wasm.ValueType, bool) {
	if ht >= 0 {
		return wasm.ConcreteRef(uint32(ht), nullable), true
	}
	// Abstract heap-type byte values per the wasm-3.0 binary format.
	var kindByte byte
	switch ht {
	case wasm.HeapTypeFunc:
		kindByte = wasm.ValueTypeFuncref.Kind()
	case wasm.HeapTypeExtern:
		kindByte = wasm.ValueTypeExternref.Kind()
	case wasm.HeapTypeExn:
		kindByte = wasm.ValueTypeExnref.Kind()
	case -13:
		kindByte = wasm.ValueTypeNoFuncref.Kind()
	case -14:
		kindByte = wasm.ValueTypeNoExternref.Kind()
	case -12:
		kindByte = wasm.ValueTypeNoExnref.Kind()
	case -15:
		kindByte = wasm.ValueTypeNullref.Kind()
	case -18:
		kindByte = wasm.ValueTypeAnyref.Kind()
	case -19:
		kindByte = wasm.ValueTypeEqref.Kind()
	case -20:
		kindByte = wasm.ValueTypeI31ref.Kind()
	case -21:
		kindByte = wasm.ValueTypeStructref.Kind()
	case -22:
		kindByte = wasm.ValueTypeArrayref.Kind()
	default:
		return 0, false
	}
	return wasm.AbstractRef(kindByte, nullable), true
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

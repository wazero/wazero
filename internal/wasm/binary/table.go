package binary

import (
	"bytes"
	"fmt"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// decodeTable returns the wasm.Table decoded with the WebAssembly Binary Format.
//
// The legacy form is `<reftype> <limits>`. The GC proposal adds the
// `0x40 0x00 <reftype> <limits> <initexpr>` form that pre-fills the
// table from a constant expression. The reftype may itself be the
// extended `0x63 <heaptype>` / `0x64 <heaptype>` ref-prefix form.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-table
// See https://webassembly.github.io/gc/core/binary/modules.html#binary-tabletype
func decodeTable(r *bytes.Reader, enabledFeatures api.CoreFeatures, ret *wasm.Table) (err error) {
	b, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("read leading byte: %v", err)
	}

	hasInitExpr := false
	if b == 0x40 {
		next, e := r.ReadByte()
		if e != nil {
			return fmt.Errorf("read table init marker: %v", e)
		}
		if next != 0x00 {
			return fmt.Errorf("expected 0x00 after 0x40 table prefix, got 0x%x", next)
		}
		tb, e := r.ReadByte()
		if e != nil {
			return fmt.Errorf("read ref type for table: %v", e)
		}
		b = tb
		hasInitExpr = true
	}

	switch b {
	case wasm.RefPrefixNullable, wasm.RefPrefixNonNullable:
		// 0x63 / 0x64 followed by s33 heap-type byte. Map the heap
		// type to its abstract-shorthand byte (funcref/externref/
		// i31ref/etc.); concrete refs reduce to funcref. The rich
		// non-nullable / concrete-type info is enforced at the
		// operand-stack level by the validator.
		ht, _, e := readSignedLeb33(r)
		if e != nil {
			return fmt.Errorf("read ref heap type for table: %v", e)
		}
		ret.Type = heapTypeToAbstractByte(ht)
	default:
		ret.Type = wasm.ValueType(b)
	}

	if ret.Type != wasm.RefTypeFuncref {
		if err = enabledFeatures.RequireEnabled(api.CoreFeatureReferenceTypes); err != nil {
			return fmt.Errorf("table type funcref is invalid: %w", err)
		}
	}

	var shared bool
	ret.Min, ret.Max, shared, err = decodeLimitsType(r)
	if err != nil {
		return fmt.Errorf("read limits: %v", err)
	}
	if ret.Min > wasm.MaximumFunctionIndex {
		return fmt.Errorf("table min must be at most %d", wasm.MaximumFunctionIndex)
	}
	if ret.Max != nil {
		if *ret.Max < ret.Min {
			return fmt.Errorf("table size minimum must not be greater than maximum")
		}
	}
	if shared {
		return fmt.Errorf("tables cannot be marked as shared")
	}
	if hasInitExpr {
		var expr wasm.ConstantExpression
		if err := decodeConstantExpression(r, enabledFeatures, &expr); err != nil {
			return fmt.Errorf("read table init expression: %w", err)
		}
		ret.InitExpr = &expr
	}
	return
}

// readSignedLeb33 reads a 33-bit signed LEB128 used by the wasm-gc heap
// type encoding. Returns the value and the number of bytes consumed.
func readSignedLeb33(r *bytes.Reader) (int64, uint64, error) {
	// The leb128 package's DecodeInt33AsInt64 helper is the canonical
	// reader; replicate its byte-stream variant here.
	var result int64
	var shift uint
	var bytesRead uint64
	for {
		b, e := r.ReadByte()
		if e != nil {
			return 0, bytesRead, e
		}
		bytesRead++
		result |= int64(b&0x7F) << shift
		shift += 7
		if b&0x80 == 0 {
			if shift < 64 && b&0x40 != 0 {
				result |= ^int64(0) << shift
			}
			return result, bytesRead, nil
		}
		if shift >= 33 {
			return 0, bytesRead, fmt.Errorf("s33 too long")
		}
	}
}

// heapTypeToAbstractByte maps a signed s33 heap-type encoding to the
// corresponding spec shorthand byte (funcref / externref / i31ref / etc.).
// Concrete type indices (non-negative) reduce to funcref.
func heapTypeToAbstractByte(ht int64) wasm.RefType {
	if ht >= 0 {
		return wasm.RefTypeFuncref
	}
	switch ht {
	case -13:
		return wasm.ValueTypeNoFuncref
	case -12:
		return wasm.ValueTypeNoExnref
	case -14:
		return wasm.ValueTypeNoExternref
	case -15:
		return wasm.ValueTypeNullref
	case -16:
		return wasm.RefTypeFuncref
	case -17:
		return wasm.RefTypeExternref
	case -18:
		return wasm.ValueTypeAnyref
	case -19:
		return wasm.ValueTypeEqref
	case -20:
		return wasm.ValueTypeI31ref
	case -21:
		return wasm.ValueTypeStructref
	case -22:
		return wasm.ValueTypeArrayref
	case -23:
		return wasm.ValueTypeExnref
	}
	return wasm.RefTypeFuncref
}

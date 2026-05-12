package binary

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/tetratelabs/wazero/internal/leb128"
	"github.com/tetratelabs/wazero/internal/wasm"
)

func decodeCode(r *bytes.Reader, codeSectionStart uint64, ret *wasm.Code) (err error) {
	ss, _, err := leb128.DecodeUint32(r)
	if err != nil {
		return fmt.Errorf("get the size of code: %w", err)
	}
	remaining := int64(ss)

	// Parse #locals.
	ls, bytesRead, err := leb128.DecodeUint32(r)
	remaining -= int64(bytesRead)
	if err != nil {
		return fmt.Errorf("get the size locals: %v", err)
	} else if remaining < 0 {
		return io.EOF
	}

	type localGroup struct {
		num uint32
		vt  wasm.ValueType
	}
	groups := make([]localGroup, 0, ls)
	var sum uint64
	for i := uint32(0); i < ls; i++ {
		num, n, err := leb128.DecodeUint32(r)
		if err != nil {
			return fmt.Errorf("read n of locals: %v", err)
		}
		remaining -= int64(n)
		sum += uint64(num)

		b, err := r.ReadByte()
		if err != nil {
			return fmt.Errorf("read type of local: %v", err)
		}
		remaining--

		var vt wasm.ValueType
		switch b {
		case byte(wasm.ValueTypeI32), byte(wasm.ValueTypeF32), byte(wasm.ValueTypeI64), byte(wasm.ValueTypeF64),
			byte(wasm.ValueTypeFuncref), byte(wasm.ValueTypeExternref), byte(wasm.ValueTypeV128),
			byte(wasm.ValueTypeExnref),
			// wasm-gc nullable abstract heap-type shorthand bytes.
			byte(wasm.ValueTypeAnyref), byte(wasm.ValueTypeEqref), byte(wasm.ValueTypeI31ref),
			byte(wasm.ValueTypeStructref), byte(wasm.ValueTypeArrayref), byte(wasm.ValueTypeNullref),
			byte(wasm.ValueTypeNoFuncref), byte(wasm.ValueTypeNoExternref), byte(wasm.ValueTypeNoExnref):
			vt = wasm.ValueType(b)
		case wasm.RefPrefixNullable, wasm.RefPrefixNonNullable:
			// 0x63 / 0x64 — typed reference local: read s33 heap type.
			ht, hn, hErr := leb128.DecodeInt33AsInt64(r)
			if hErr != nil {
				return fmt.Errorf("read ref heap type for local: %v", hErr)
			}
			remaining -= int64(hn)
			nullable := b == wasm.RefPrefixNullable
			if ht >= 0 {
				vt = wasm.ConcreteRef(uint32(ht), nullable)
			} else {
				vt = wasm.AbstractRef(byte(ht&0x7F), nullable)
			}
		default:
			return fmt.Errorf("invalid local type: 0x%x", b)
		}
		if remaining < 0 {
			return io.EOF
		}
		groups = append(groups, localGroup{num: num, vt: vt})
	}

	if sum > math.MaxUint32 {
		return fmt.Errorf("too many locals: %d", sum)
	}

	localTypes := make([]wasm.ValueType, 0, sum)
	for _, g := range groups {
		for j := uint32(0); j < g.num; j++ {
			localTypes = append(localTypes, g.vt)
		}
	}

	bodyOffsetInCodeSection := codeSectionStart - uint64(r.Len())
	body := make([]byte, remaining)
	if _, err = io.ReadFull(r, body); err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if endIndex := len(body) - 1; endIndex < 0 || body[endIndex] != wasm.OpcodeEnd {
		return fmt.Errorf("expr not end with OpcodeEnd")
	}

	ret.BodyOffsetInCodeSection = bodyOffsetInCodeSection
	ret.LocalTypes = localTypes
	ret.Body = body
	return nil
}

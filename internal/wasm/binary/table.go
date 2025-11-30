package binary

import (
	"bytes"
	"fmt"
	"math"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// decodeTable returns the wasm.Table decoded with the WebAssembly 1.0 (20191205) Binary Format.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-table
func decodeTable(r *bytes.Reader, enabledFeatures api.CoreFeatures, ret *wasm.Table) (err error) {
	ret.Type, err = r.ReadByte()
	if err != nil {
		return fmt.Errorf("read leading byte: %v", err)
	}

	if ret.Type != wasm.RefTypeFuncref {
		if err = enabledFeatures.RequireEnabled(api.CoreFeatureReferenceTypes); err != nil {
			return fmt.Errorf("table type funcref is invalid: %w", err)
		}
	}

	var shared bool
	var min uint64
	var max *uint64
	var is64 bool
	min, max, shared, is64, err = decodeLimitsType(r)
	if err != nil {
		return fmt.Errorf("read limits: %v", err)
	}
	if is64 {
		if err = enabledFeatures.RequireEnabled(api.CoreFeatureMemory64); err != nil {
			return fmt.Errorf("table64 invalid: %w", err)
		}
	}
	if min > math.MaxUint32 {
		return fmt.Errorf("table min %d exceeds 32-bit limit", min)
	}
	ret.Min = uint32(min)
	if max != nil {
		if *max > math.MaxUint32 {
			return fmt.Errorf("table max %d exceeds 32-bit limit", *max)
		}
		mv := uint32(*max)
		ret.Max = &mv
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
	ret.Is64 = is64
	return
}

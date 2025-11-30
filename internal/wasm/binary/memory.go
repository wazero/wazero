package binary

import (
	"bytes"
	"fmt"
	"math"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// decodeMemory returns the api.Memory decoded with the WebAssembly 1.0 (20191205) Binary Format.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-memory
func decodeMemory(
	r *bytes.Reader,
	enabledFeatures api.CoreFeatures,
	memorySizer func(minPages uint32, maxPages *uint32) (min, capacity, max uint32),
	memoryLimitPages uint32,
) (*wasm.Memory, error) {
	min64, maxP64, shared, is64, err := decodeLimitsType(r)
	if err != nil {
		return nil, err
	}

	if is64 {
		if err = enabledFeatures.RequireEnabled(api.CoreFeatureMemory64); err != nil {
			return nil, fmt.Errorf("memory64: %w", err)
		}
	}

	toUint32 := func(v uint64, what string) (uint32, error) {
		if v > math.MaxUint32 {
			return 0, fmt.Errorf("%s %d pages exceeds 32-bit limit", what, v)
		}
		return uint32(v), nil
	}

	min, err := toUint32(min64, "min")
	if err != nil {
		return nil, err
	}
	var maxP *uint32
	if maxP64 != nil {
		mv, err := toUint32(*maxP64, "max")
		if err != nil {
			return nil, err
		}
		maxP = &mv
	}

	if shared {
		if !enabledFeatures.IsEnabled(experimental.CoreFeaturesThreads) {
			return nil, fmt.Errorf("shared memory requested but threads feature not enabled")
		}

		// This restriction may be lifted in the future.
		// https://webassembly.github.io/threads/core/binary/types.html#memory-types
		if maxP == nil {
			return nil, fmt.Errorf("shared memory requires a maximum size to be specified")
		}
	}

	min, capacity, max := memorySizer(min, maxP)
	mem := &wasm.Memory{Min: min, Cap: capacity, Max: max, IsMaxEncoded: maxP != nil, IsShared: shared, Is64: is64}

	return mem, mem.Validate(memoryLimitPages)
}

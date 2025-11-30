package binary

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/binaryencoding"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestLimitsType(t *testing.T) {
	zero := uint64(0)
	largest := uint64(math.MaxUint32)

	tests := []struct {
		name     string
		min      uint64
		max      *uint64
		shared   bool
		memory64 bool
		expected []byte
	}{
		{
			name:     "min 0",
			expected: []byte{0x0, 0},
		},
		{
			name:     "min 0, max 0",
			max:      &zero,
			expected: []byte{0x1, 0, 0},
		},
		{
			name:     "min largest",
			min:      largest,
			expected: []byte{0x0, 0xff, 0xff, 0xff, 0xff, 0xf},
		},
		{
			name:     "min 0, max largest",
			max:      &largest,
			expected: []byte{0x1, 0, 0xff, 0xff, 0xff, 0xff, 0xf},
		},
		{
			name:     "min largest max largest",
			min:      largest,
			max:      &largest,
			expected: []byte{0x1, 0xff, 0xff, 0xff, 0xff, 0xf, 0xff, 0xff, 0xff, 0xff, 0xf},
		},
		{
			name:     "min 0, shared",
			shared:   true,
			expected: []byte{0x2, 0},
		},
		{
			name:     "min 0, max 0, shared",
			max:      &zero,
			shared:   true,
			expected: []byte{0x3, 0, 0},
		},
		{
			name:     "min largest, shared",
			min:      largest,
			shared:   true,
			expected: []byte{0x2, 0xff, 0xff, 0xff, 0xff, 0xf},
		},
		{
			name:     "min 0, max largest, shared",
			max:      &largest,
			shared:   true,
			expected: []byte{0x3, 0, 0xff, 0xff, 0xff, 0xff, 0xf},
		},
		{
			name:     "min largest max largest, shared",
			min:      largest,
			max:      &largest,
			shared:   true,
			expected: []byte{0x3, 0xff, 0xff, 0xff, 0xff, 0xf, 0xff, 0xff, 0xff, 0xff, 0xf},
		},
		{
			name:     "memory64 min 0",
			memory64: true,
			expected: []byte{0x4, 0},
		},
		{
			name:     "memory64 min 0, max 0",
			max:      &zero,
			memory64: true,
			expected: []byte{0x5, 0, 0},
		},
	}

	for _, tt := range tests {
		tc := tt

		b := binaryencoding.EncodeLimitsType(tc.min, tc.max, tc.shared, tc.memory64)
		t.Run(fmt.Sprintf("encode - %s", tc.name), func(t *testing.T) {
			require.Equal(t, tc.expected, b)
		})

		t.Run(fmt.Sprintf("decode - %s", tc.name), func(t *testing.T) {
			min, max, shared, is64, err := decodeLimitsType(bytes.NewReader(b))
			require.NoError(t, err)
			require.Equal(t, tc.min, min)
			require.Equal(t, tc.max, max)
			require.Equal(t, tc.shared, shared)
			require.Equal(t, tc.memory64, is64)
		})
	}
}

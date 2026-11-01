package wazevoapi

import (
	"sync"
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestPerfmap_ClearFunction(t *testing.T) {
	f := &Perfmap{}
	f.AddModuleEntry(0, 0, 4, "f0:L0")
	f.AddModuleEntry(1, 0, 4, "f1:L0")
	f.AddModuleEntry(1, 4, 8, "f1:L1")
	f.AddModuleEntry(2, 0, 4, "f2:L0")

	// Function 1 is laid out again: only its entries go, not those of the functions compiled before or
	// after it.
	f.ClearFunction(1)
	require.Equal(t, []entry{
		{index: 0, offset: 0, size: 4, name: "f0:L0"},
		{index: 2, offset: 0, size: 4, name: "f2:L0"},
	}, f.entries)
}

func TestPerfmap_AddModuleEntry_concurrent(t *testing.T) {
	f := &Perfmap{}
	const workers, perWorker = 8, 1000
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				f.AddModuleEntry(w, int64(i), 4, "")
			}
		}()
	}
	wg.Wait()
	require.Equal(t, workers*perWorker, len(f.entries))
}

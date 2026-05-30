package wasm

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

// TestGCHandleTable_lifetime verifies that wasm-gc struct/array handles
// resolve via the owning module, that a module's table is reclaimed (and its
// gcID recycled) when the module is removed from the store, and that a stale
// handle from a closed module resolves to nil rather than crashing.
func TestGCHandleTable_lifetime(t *testing.T) {
	s := newStore()

	m1 := &ModuleInstance{s: s}
	st := &WasmStruct{TypeID: 1}
	ar := &WasmArray{TypeID: 2}
	h1 := m1.GCRegister(st)
	h2 := m1.GCRegister(ar)

	// The handles are GC handles (tag 0b10), not function pointers / i31.
	require.True(t, IsGCHandle(h1))
	require.True(t, IsGCHandle(h2))
	// A gcID was assigned lazily and the module registered.
	require.NotEqual(t, uint32(0), m1.gcID)
	require.Equal(t, m1, s.gcModules[m1.gcID])
	require.Equal(t, m1.gcID, gcHandleModuleID(h1))

	// Lookups route through the store to the owning module's table and
	// return the exact same objects.
	require.Equal(t, st, s.GCLookup(h1).(*WasmStruct))
	require.Equal(t, ar, s.GCLookup(h2).(*WasmArray))

	// Cross-module resolution: a different module instance resolving m1's
	// handle (every module shares the same store) still finds it.
	m2 := &ModuleInstance{s: s}
	require.Equal(t, st, m2.GetStore().GCLookup(h1).(*WasmStruct))

	id1 := m1.gcID

	// Closing m1 frees its table and recycles its gcID.
	s.gcReleaseModule(m1)
	require.Equal(t, uint32(0), m1.gcID)
	require.Nil(t, m1.gcObjects)
	_, ok := s.gcModules[id1]
	require.False(t, ok)

	// A stale handle into the closed module resolves to nil (no panic), so
	// the interpreter treats it as a failed match / null dereference.
	require.Nil(t, s.GCLookup(h1))

	// The next GC-using module reuses the recycled id.
	m3 := &ModuleInstance{s: s}
	_ = m3.GCRegister(&WasmStruct{TypeID: 3})
	require.Equal(t, id1, m3.gcID)
}

// TestGCHandleTable_deleteModuleReleases confirms the table is freed through
// the real module-close chokepoint (deleteModule), not just the helper.
func TestGCHandleTable_deleteModuleReleases(t *testing.T) {
	s := newStore()
	m := &ModuleInstance{s: s}
	_ = m.GCRegister(&WasmStruct{TypeID: 1})
	id := m.gcID
	require.Equal(t, m, s.gcModules[id])

	require.NoError(t, s.deleteModule(m))

	_, ok := s.gcModules[id]
	require.False(t, ok)
	require.Nil(t, m.gcObjects)
}

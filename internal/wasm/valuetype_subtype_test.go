package wasm

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestIsRefSubtypeOf_Equality(t *testing.T) {
	require.True(t, isRefSubtypeOf(ValueTypeFuncref, ValueTypeFuncref))
	require.True(t, isRefSubtypeOf(ValueTypeAnyref, ValueTypeAnyref))
	require.True(t, isRefSubtypeOf(ConcreteRef(3, true), ConcreteRef(3, true)))
}

func TestIsRefSubtypeOf_Nullability(t *testing.T) {
	// Non-nullable abstract can flow into nullable abstract.
	require.True(t, isRefSubtypeOf(ValueTypeFuncref.AsNonNullable(), ValueTypeFuncref))
	// But nullable cannot flow into non-nullable.
	require.False(t, isRefSubtypeOf(ValueTypeFuncref, ValueTypeFuncref.AsNonNullable()))

	// Non-nullable concrete can flow into nullable concrete (same typeIdx).
	require.True(t, isRefSubtypeOf(ConcreteRef(0, false), ConcreteRef(0, true)))
	require.False(t, isRefSubtypeOf(ConcreteRef(0, true), ConcreteRef(0, false)))
}

func TestIsRefSubtypeOf_ConcreteIntoAbstract(t *testing.T) {
	// (ref $t) and (ref null $t) flow into funcref / anyref slots.
	require.True(t, isRefSubtypeOf(ConcreteRef(0, true), ValueTypeFuncref))
	require.True(t, isRefSubtypeOf(ConcreteRef(0, false), ValueTypeFuncref))
	require.True(t, isRefSubtypeOf(ConcreteRef(0, true), ValueTypeAnyref))
	// But not into structref/arrayref/i31ref/eqref without module context.
	require.False(t, isRefSubtypeOf(ConcreteRef(0, true), ValueTypeStructref))
}

func TestIsRefSubtypeOf_ConcreteByTypeIdx(t *testing.T) {
	require.True(t, isRefSubtypeOf(ConcreteRef(3, true), ConcreteRef(3, true)))
	require.False(t, isRefSubtypeOf(ConcreteRef(3, true), ConcreteRef(4, true)))
}

func TestIsRefSubtypeOf_AbstractHierarchy(t *testing.T) {
	// i31/struct/array <: eq <: any
	require.True(t, isRefSubtypeOf(ValueTypeI31ref, ValueTypeEqref))
	require.True(t, isRefSubtypeOf(ValueTypeStructref, ValueTypeEqref))
	require.True(t, isRefSubtypeOf(ValueTypeArrayref, ValueTypeEqref))
	require.True(t, isRefSubtypeOf(ValueTypeEqref, ValueTypeAnyref))
	require.True(t, isRefSubtypeOf(ValueTypeI31ref, ValueTypeAnyref))
	// Reverse direction is not allowed.
	require.False(t, isRefSubtypeOf(ValueTypeAnyref, ValueTypeEqref))
	require.False(t, isRefSubtypeOf(ValueTypeEqref, ValueTypeI31ref))
	// Cross-hierarchy isn't allowed.
	require.False(t, isRefSubtypeOf(ValueTypeFuncref, ValueTypeAnyref))
}

func TestIsRefSubtypeOf_BottomTypes(t *testing.T) {
	require.True(t, isRefSubtypeOf(ValueTypeNoFuncref, ValueTypeFuncref))
	require.True(t, isRefSubtypeOf(ValueTypeNoExternref, ValueTypeExternref))
	require.True(t, isRefSubtypeOf(ValueTypeNoExnref, ValueTypeExnref))
	require.True(t, isRefSubtypeOf(ValueTypeNullref, ValueTypeAnyref))
	require.True(t, isRefSubtypeOf(ValueTypeNullref, ValueTypeI31ref))
}

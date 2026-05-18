package experimental

import (
	"context"
	"errors"

	"github.com/tetratelabs/wazero/internal/expctxkeys"
	"github.com/tetratelabs/wazero/internal/fuel"
	"github.com/tetratelabs/wazero/internal/wasmruntime"
)

// ErrOutOfFuel is returned from a wazero Call when fuel installed via
// SetFuel is exhausted. Use errors.Is to detect it.
var ErrOutOfFuel = wasmruntime.ErrRuntimeOutOfFuel

// ErrFuelNotSupported is returned when a wasm function is invoked with a
// fueled context under an engine that does not implement fuel metering.
var ErrFuelNotSupported = errors.New("fuel metering requires interpreter runtime")

// SetFuel installs or replaces a fuel budget on ctx. Wasm functions invoked
// under the returned context — including a module's start function during
// instantiation — are metered: each dispatched operation consumes one unit,
// and execution traps with ErrOutOfFuel when the budget is exhausted.
//
// The first call on a given ctx installs a meter and returns a new ctx.
// Subsequent calls on a descendant ctx replace the budget on the same meter
// in place; the returned ctx is the input unchanged. This makes refueling
// mid-execution from a host function simply another SetFuel call.
//
// Only the interpreter engine honors fuel; the compiler engine returns
// ErrFuelNotSupported.
func SetFuel(ctx context.Context, units uint64) context.Context {
	if m, ok := ctx.Value(expctxkeys.FuelKey{}).(*fuel.Meter); ok {
		m.Set(units)
		return ctx
	}
	return context.WithValue(ctx, expctxkeys.FuelKey{}, fuel.New(units))
}

// GetFuel returns the remaining fuel budget on ctx, or 0 if no fuel is
// installed or the budget is exhausted.
func GetFuel(ctx context.Context) uint64 {
	if m, ok := ctx.Value(expctxkeys.FuelKey{}).(*fuel.Meter); ok {
		return m.Remaining()
	}
	return 0
}

// AddFuel adds units to the fuel budget on ctx. No-op if no fuel is
// installed. Unlike SetFuel, this is a single atomic addition: safe to call
// concurrently with the engine and with other AddFuel callers without a
// read-modify-write race.
func AddFuel(ctx context.Context, units uint64) {
	if m, ok := ctx.Value(expctxkeys.FuelKey{}).(*fuel.Meter); ok {
		m.Add(units)
	}
}

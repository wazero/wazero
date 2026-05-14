package experimental

import "github.com/tetratelabs/wazero/api"

// CoreFeaturesThreads enables threads instructions ("threads").
//
// # Notes
//
//   - The instruction list is too long to enumerate in godoc.
//     See https://github.com/WebAssembly/threads/blob/main/proposals/threads/Overview.md
//   - Atomic operations are guest-only until api.Memory or otherwise expose them to host functions.
//   - On systems without mmap available, the memory will pre-allocate to the maximum size. Many
//     binaries will use a theroetical maximum like 4GB, so if using such a binary on a system
//     without mmap, consider editing the binary to reduce the max size setting of memory.
const CoreFeaturesThreads = api.CoreFeatureSIMD << 1

// CoreFeaturesTailCall enables tail call instructions ("tail-call").
const CoreFeaturesTailCall = api.CoreFeatureSIMD << 2

// CoreFeaturesExtendedConst enables extended constant expressions.
//
// # Notes
//
//   - Enables i32.add/sub/mul and i64.add/sub/mul in constant expressions.
//   - Enables references to any previous global index in constant expressions,
//     instead of just imported globals.
//
// See https://github.com/WebAssembly/extended-const for further details.
const CoreFeaturesExtendedConst = api.CoreFeatureSIMD << 3

// CoreFeaturesExceptionHandling enables exception handling instructions.
//
// See https://github.com/WebAssembly/exception-handling for further details.
const CoreFeaturesExceptionHandling = api.CoreFeatureSIMD << 4

// CoreFeaturesGC enables the WebAssembly GC proposal (part of Wasm 3.0).
//
// # Notes
//
//   - Adds typed function references: call_ref, return_call_ref, ref.as_non_null,
//     br_on_null, br_on_non_null, ref.eq, and non-nullable (ref $t) types.
//   - Adds struct and array composite types with subtyping, rec groups, and finality.
//   - Adds reference type hierarchy: any, eq, i31, struct, array, none, nofunc,
//     noextern, plus the existing func/extern.
//   - Adds GC instructions: struct.{new,new_default,get,get_s,get_u,set},
//     array.{new,new_default,new_fixed,new_data,new_elem,get,get_s,get_u,set,
//     len,fill,copy,init_data,init_elem}, ref.{test,cast,i31}, i31.{get_s,get_u},
//     br_on_cast, br_on_cast_fail, any.convert_extern, extern.convert_any.
//
// See https://github.com/WebAssembly/gc for further details.
const CoreFeaturesGC = api.CoreFeatureSIMD << 5

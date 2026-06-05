package wasm

import "unsafe"

// I31Ref is an `i31` reference value introduced by the WebAssembly GC
// proposal. An i31 holds a 31-bit integer, conceptually "tagged" in a
// reference slot. The spec defines exactly three operations:
//
//   - ref.i31    : i32 -> i31ref  (keeps the low 31 bits of the source)
//   - i31.get_s  : i31ref -> i32  (sign-extends bit 30 into bit 31)
//   - i31.get_u  : i31ref -> i32  (zero-extends; bit 31 is 0)
//
// In this implementation, I31Ref is a heap-allocated struct holding the
// 31-bit payload. Go's GC reclaims unreachable instances. Future
// optimisations (e.g., tagged uintptr or pointer-low-bit tagging) can be
// applied without changing callers, since the only public surface is the
// constructor and the two get_s / get_u helpers.
//
// See https://webassembly.github.io/spec/core/exec/instructions.html#xref-syntax-instructions-syntax-instr-i31-mathsf-i31-get-sx-mathit-sx
type I31Ref struct {
	// bits stores the 31-bit i31 value in the low 31 bits. Bit 31 is
	// always 0 and ignored by consumers.
	bits uint32
}

// I31RefMask is the bit-mask that ref.i31 applies to its input.
const I31RefMask uint32 = 0x7FFFFFFF

// NewI31Ref constructs an i31 reference holding the low 31 bits of v.
func NewI31Ref(v uint32) *I31Ref {
	return &I31Ref{bits: v & I31RefMask}
}

// I31RefFromInt32 is the typed sibling of NewI31Ref for callers that hold
// a Go int32 (e.g., the operand-stack pop helper in the interpreter).
func I31RefFromInt32(v int32) *I31Ref {
	return NewI31Ref(uint32(v))
}

// SignedI32 returns the i31 value as a 32-bit signed integer, with bit 30
// of the i31 sign-extended into the upper bit. Implements i31.get_s.
func (r *I31Ref) SignedI32() int32 {
	b := r.bits
	if b&0x40000000 != 0 {
		// Bit 30 set => negative; set bit 31 to sign-extend.
		return int32(b | 0x80000000)
	}
	return int32(b)
}

// UnsignedI32 returns the i31 value zero-extended to 32 bits. The result
// is always in [0, 2^31 - 1]. Implements i31.get_u.
func (r *I31Ref) UnsignedI32() uint32 {
	return r.bits
}

// Equals reports whether two i31 references hold the same payload. This is
// the semantic equality used by ref.eq when both operands are i31 refs.
//
// Per the spec, ref.eq on two i31 refs returns 1 iff their numeric values
// are equal — independent of whether they are the same heap pointer. So
// we compare payload bits, not pointer identity.
func (r *I31Ref) Equals(other *I31Ref) bool {
	if r == nil || other == nil {
		return r == other
	}
	return r.bits == other.bits
}

// -----------------------------------------------------------------------
// Tagged-uintptr i31 encoding used by the interpreter's operand stack.
//
// Layout:
//   bit 0:      tag — 1 if this is an i31 ref, 0 if it is a pointer (or null)
//   bits 1..31: the 31-bit i31 value
//   bits 32-63: zero
//
// The null i31 ref is encoded as uintptr 0 (no tag bit, no value). Real
// heap pointers always have bit 0 clear because Go's allocator aligns
// objects to at least 8 bytes on the supported 64-bit platforms — so the
// tag bit is unambiguous.

// Tagged representation of reference values on the interpreter operand stack.
// High bits 61–63 of the uint64 slot encode the tag; Go pointers use at
// most 48–52 bits of address space, so bits 52–63 are always zero for
// real pointers.
//
//	(none)  — function pointer (typed funcref) or null (the zero slot)
//	bit 63  — i31: 31-bit value in bits 0–30, no shift required
//	bit 62  — extern-wrapped-in-anyref: value in bits 0–61
//	bit 61  — GC ref (struct / array): pointer in bits 0–60, clear
//	          bit 61 before dereferencing. The object is kept alive
//	          by ModuleInstance.GCRoots.

const (
	tagI31       uint64 = 1 << 63
	tagExternAny uint64 = 1 << 62
	tagGCRef     uint64 = 1 << 61
	tagMask      uint64 = tagI31 | tagExternAny | tagGCRef
)

// TagGCPointer encodes a Go pointer to a WasmStruct or WasmArray as a
// tagged uint64 for the operand stack by setting bit 61.
func TagGCPointer(ptr unsafe.Pointer) uint64 {
	return uint64(uintptr(ptr)) | tagGCRef
}

// UntagGCPointer clears bit 61 and returns the raw pointer.
// Callers must check IsGCRef first.
func UntagGCPointer(v uint64) unsafe.Pointer {
	return unsafe.Pointer(uintptr(v &^ tagGCRef))
}

// IsGCRef reports whether a slot is a tagged GC-ref pointer (struct or
// array). Checks all tag bits to avoid false positives from extern-as-any
// payloads that happen to have bit 61 set.
func IsGCRef(slot uint64) bool {
	return slot&tagMask == tagGCRef
}

// PackI31 returns the tagged uint64 representation of an i31 value. The
// 32-bit input is narrowed to its low 31 bits per the spec for ref.i31.
// Bit 63 is set as the i31 tag; the value occupies bits 0–30.
func PackI31(v uint32) uint64 {
	return uint64(v&I31RefMask) | tagI31
}

// IsTaggedI31 reports whether a uint64 slot is an i31 ref.
func IsTaggedI31(v uint64) bool {
	return v&tagI31 != 0
}

// UnpackI31Signed extracts an i31 value as a sign-extended i32. Callers
// must verify v is a tagged i31 (via IsTaggedI31) first.
func UnpackI31Signed(v uint64) int32 {
	b := uint32(v) & I31RefMask
	if b&0x40000000 != 0 {
		return int32(b | 0x80000000)
	}
	return int32(b)
}

// UnpackI31Unsigned extracts an i31 value as a zero-extended u32.
func UnpackI31Unsigned(v uint64) uint32 {
	return uint32(v) & I31RefMask
}

// PackExternAsAny tags an externref value so it can be stored in an
// anyref slot without colliding with the i31 or GC-ref bit patterns.
// Sets bit 62; the original value is fully preserved in bits 0–61.
// Null passes through unchanged.
func PackExternAsAny(v uint64) uint64 {
	if v == 0 {
		return 0
	}
	return v | tagExternAny
}

// IsTaggedExternAsAny reports whether v was produced by PackExternAsAny.
func IsTaggedExternAsAny(v uint64) bool {
	return v != 0 && v&tagExternAny != 0
}

// UnpackExternAsAny extracts the original externref value by clearing
// bit 62.
func UnpackExternAsAny(v uint64) uint64 {
	if v == 0 {
		return 0
	}
	return v &^ tagExternAny
}

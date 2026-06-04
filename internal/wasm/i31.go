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
// The low 2 bits encode the tag:
//
//	0b00 — function pointer (typed funcref) or null (the zero slot)
//	0b01 — i31: payload in bits 2..32 (31 bits of value)
//	0b10 — GC ref (struct / array): tagged raw pointer. Clear the low 2
//	       bits to recover the *WasmStruct or *WasmArray. The object is
//	       kept alive by ModuleInstance.GCRoots.
//	0b11 — extern-wrapped-in-anyref: payload in bits 2..63 (62 bits)
//
// Go struct pointers are at least 8-byte aligned on 64-bit platforms, so
// the low 3 bits are always zero and the 0b10 tag is unambiguous against
// function-pointer slots (0b00).
//
// Externref values in wazero are opaque uintptrs supplied by the host.
// Storing them directly in an anyref slot is ambiguous because some
// host externref values overlap with the i31 bit pattern (any odd
// integer looks like a tagged i31). The 0b11 tag distinguishes
// externref-converted-to-anyref values so refMatches can return the
// correct answer for ref.test eqref / ref.test i31ref / etc.

const (
	PrimTagMask uintptr = 0b11
	primTagI31  uintptr = 0b01
	primTagHeap uintptr = 0b10
	primTagExtAn uintptr = 0b11
)

// TagGCPointer encodes a Go pointer to a WasmStruct or WasmArray as a
// tagged uint64 for the operand stack. The pointer must be 4-byte aligned
// (guaranteed for Go heap-allocated structs) so the low 2 bits are free
// for the primTagHeap tag.
func TagGCPointer(ptr unsafe.Pointer) uint64 {
	return uint64(uintptr(ptr)) | uint64(primTagHeap)
}

// UntagGCPointer clears the primTagHeap tag and returns the raw pointer.
// Callers must check IsGCRef first.
func UntagGCPointer(v uint64) unsafe.Pointer {
	return unsafe.Pointer(uintptr(v) &^ uintptr(PrimTagMask))
}

// IsGCRef reports whether a slot is a tagged GC-ref pointer (a struct or
// array), as opposed to null/function-pointer (0b00), i31 (0b01), or
// externref-as-any (0b11).
func IsGCRef(slot uint64) bool {
	return uintptr(slot)&PrimTagMask == primTagHeap
}

// PackI31 returns the tagged-uintptr representation of an i31 value. The
// 32-bit input is narrowed to its low 31 bits per the spec for ref.i31.
func PackI31(v uint32) uintptr {
	return (uintptr(v&I31RefMask) << 2) | primTagI31
}

// IsTaggedI31 reports whether a tagged uintptr is an i31 ref (and not the
// null reference or a real pointer).
func IsTaggedI31(t uintptr) bool {
	return t&PrimTagMask == primTagI31
}

// UnpackI31Signed extracts an i31 value as a sign-extended i32. Callers
// must verify t is a tagged i31 (via IsTaggedI31) first; on a null or
// non-i31 input the result is undefined.
func UnpackI31Signed(t uintptr) int32 {
	b := uint32(t>>2) & I31RefMask
	if b&0x40000000 != 0 {
		return int32(b | 0x80000000)
	}
	return int32(b)
}

// UnpackI31Unsigned extracts an i31 value as a zero-extended u32. As with
// UnpackI31Signed, callers verify the tag first.
func UnpackI31Unsigned(t uintptr) uint32 {
	return uint32(t>>2) & I31RefMask
}

// PackExternAsAny tags an externref value (raw uintptr from the host)
// so it can be stored in an anyref slot without colliding with the
// i31 bit pattern. The top 62 bits carry the original value. A zero
// externref (null) passes through unchanged.
func PackExternAsAny(v uintptr) uintptr {
	if v == 0 {
		return 0
	}
	return (v << 2) | primTagExtAn
}

// IsTaggedExternAsAny reports whether t was produced by PackExternAsAny.
func IsTaggedExternAsAny(t uintptr) bool {
	return t != 0 && t&PrimTagMask == primTagExtAn
}

// UnpackExternAsAny extracts the original externref uintptr from a
// value previously produced by PackExternAsAny.
func UnpackExternAsAny(t uintptr) uintptr {
	if t == 0 {
		return 0
	}
	return t >> 2
}

package ssa

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestFoldBinaryIntConst(t *testing.T) {
	for _, tc := range []struct {
		op   Opcode
		x, y uint64
		exp  uint64
	}{
		{OpcodeIadd, 7, 5, 12},
		{OpcodeIsub, 5, 7, ^uint64(1)}, // 5-7 wraps.
		{OpcodeImul, 6, 7, 42},
		{OpcodeBand, 0b1100, 0b1010, 0b1000},
		{OpcodeBor, 0b1100, 0b1010, 0b1110},
		{OpcodeBxor, 0b1100, 0b1010, 0b0110},
	} {
		require.Equal(t, tc.exp, foldBinaryIntConst(tc.op, tc.x, tc.y))
	}
}

func TestPass_constFoldingArithmetic(t *testing.T) {
	for _, tc := range []struct {
		name          string
		setup         func(b *builder)
		before, after string
	}{
		{
			name: "fold const+const",
			setup: func(b *builder) {
				entry := b.AllocateBasicBlock()
				b.SetCurrentBlock(entry)
				x := insertIconst32(b, 7)
				y := insertIconst32(b, 5)
				add := b.AllocateInstruction()
				add.AsIadd(x, y)
				b.InsertInstruction(add)
				insertReturn(b, add.Return())
			},
			before: `
blk0: ()
	v0:i32 = Iconst_32 0x7
	v1:i32 = Iconst_32 0x5
	v2:i32 = Iadd v0, v1
	Return v2
`,
			after: `
blk0: ()
	v2:i32 = Iconst_32 0xc
	Return v2
`,
		},
		{
			name: "identity x+0 -> x",
			setup: func(b *builder) {
				entry := b.AllocateBasicBlock()
				x := entry.AddParam(b, TypeI32)
				b.SetCurrentBlock(entry)
				zero := insertIconst32(b, 0)
				add := b.AllocateInstruction()
				add.AsIadd(x, zero)
				b.InsertInstruction(add)
				insertReturn(b, add.Return())
			},
			before: `
blk0: (v0:i32)
	v1:i32 = Iconst_32 0x0
	v2:i32 = Iadd v0, v1
	Return v2
`,
			after: `
blk0: (v0:i32)
	Return v0
`,
		},
		{
			name: "identity x*0 -> 0",
			setup: func(b *builder) {
				entry := b.AllocateBasicBlock()
				x := entry.AddParam(b, TypeI32)
				b.SetCurrentBlock(entry)
				zero := insertIconst32(b, 0)
				mul := b.AllocateInstruction()
				mul.AsImul(x, zero)
				b.InsertInstruction(mul)
				insertReturn(b, mul.Return())
			},
			before: `
blk0: (v0:i32)
	v1:i32 = Iconst_32 0x0
	v2:i32 = Imul v0, v1
	Return v2
`,
			after: `
blk0: (v0:i32)
	v1:i32 = Iconst_32 0x0
	Return v1
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder().(*builder)
			tc.setup(b)
			require.Equal(t, tc.before, b.Format())
			passConstFoldingArithmeticOpt(b)
			passDeadCodeEliminationOpt(b)
			require.Equal(t, tc.after, b.Format())
		})
	}
}

func insertIconst32(b *builder, v uint32) Value {
	inst := b.AllocateInstruction()
	inst.AsIconst32(v)
	b.InsertInstruction(inst)
	return inst.Return()
}

func insertReturn(b *builder, v Value) {
	ret := b.AllocateInstruction()
	args := b.varLengthPool.Allocate(1)
	args = args.Append(&b.varLengthPool, v)
	ret.AsReturn(args)
	b.InsertInstruction(ret)
}

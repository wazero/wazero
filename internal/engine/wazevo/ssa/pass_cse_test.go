package ssa

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestPass_commonSubexpressionElimination(t *testing.T) {
	for _, tc := range []struct {
		name          string
		setup         func(b *builder)
		before, after string
	}{
		{
			name: "within block: identical iadd reused",
			setup: func(b *builder) {
				entry := b.AllocateBasicBlock()
				x := entry.AddParam(b, TypeI32)
				y := entry.AddParam(b, TypeI32)
				b.SetCurrentBlock(entry)
				add1 := insertIadd(b, x, y)
				add2 := insertIadd(b, x, y)
				insertReturnVals(b, add1, add2)
			},
			before: `
blk0: (v0:i32, v1:i32)
	v2:i32 = Iadd v0, v1
	v3:i32 = Iadd v0, v1
	Return v2, v3
`,
			after: `
blk0: (v0:i32, v1:i32)
	v2:i32 = Iadd v0, v1
	Return v2, v2
`,
		},
		{
			name: "across blocks: dominating iadd reused",
			setup: func(b *builder) {
				entry := b.AllocateBasicBlock()
				x := entry.AddParam(b, TypeI32)
				y := entry.AddParam(b, TypeI32)
				end := b.AllocateBasicBlock()
				b.SetCurrentBlock(entry)
				e := insertIadd(b, x, y)
				jmp := b.AllocateInstruction()
				jmp.AsJump(ValuesNil, end)
				b.InsertInstruction(jmp)
				b.SetCurrentBlock(end)
				e2 := insertIadd(b, x, y)
				insertReturnVals(b, e, e2)
			},
			before: `
blk0: (v0:i32, v1:i32)
	v2:i32 = Iadd v0, v1
	Jump blk1

blk1: () <-- (blk0)
	v3:i32 = Iadd v0, v1
	Return v2, v3
`,
			after: `
blk0: (v0:i32, v1:i32)
	v2:i32 = Iadd v0, v1
	Jump blk1

blk1: () <-- (blk0)
	Return v2, v2
`,
		},
		{
			name: "different args are not merged",
			setup: func(b *builder) {
				entry := b.AllocateBasicBlock()
				x := entry.AddParam(b, TypeI32)
				y := entry.AddParam(b, TypeI32)
				b.SetCurrentBlock(entry)
				add1 := insertIadd(b, x, y)
				add2 := insertIadd(b, y, x) // operands swapped: distinct key.
				insertReturnVals(b, add1, add2)
			},
			before: `
blk0: (v0:i32, v1:i32)
	v2:i32 = Iadd v0, v1
	v3:i32 = Iadd v1, v0
	Return v2, v3
`,
			after: `
blk0: (v0:i32, v1:i32)
	v2:i32 = Iadd v0, v1
	v3:i32 = Iadd v1, v0
	Return v2, v3
`,
		},
		{
			name: "identical loads are not merged",
			setup: func(b *builder) {
				entry := b.AllocateBasicBlock()
				ptr := entry.AddParam(b, TypeI64)
				b.SetCurrentBlock(entry)
				ld1 := insertLoad(b, ptr)
				ld2 := insertLoad(b, ptr)
				insertReturnVals(b, ld1, ld2)
			},
			before: `
blk0: (v0:i64)
	v1:i32 = Load v0, 0x0
	v2:i32 = Load v0, 0x0
	Return v1, v2
`,
			after: `
blk0: (v0:i64)
	v1:i32 = Load v0, 0x0
	v2:i32 = Load v0, 0x0
	Return v1, v2
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder().(*builder)
			tc.setup(b)
			passCalculateImmediateDominators(b) // CSE needs dominators + reverse post-order.
			require.Equal(t, tc.before, b.Format())
			passCommonSubexpressionEliminationOpt(b)
			passDeadCodeEliminationOpt(b)
			require.Equal(t, tc.after, b.Format())
		})
	}
}

func insertIadd(b *builder, x, y Value) Value {
	add := b.AllocateInstruction()
	add.AsIadd(x, y)
	b.InsertInstruction(add)
	return add.Return()
}

func insertLoad(b *builder, ptr Value) Value {
	ld := b.AllocateInstruction()
	ld.AsLoad(ptr, 0, TypeI32)
	b.InsertInstruction(ld)
	return ld.Return()
}

func insertReturnVals(b *builder, vs ...Value) {
	ret := b.AllocateInstruction()
	args := b.varLengthPool.Allocate(len(vs))
	for _, v := range vs {
		args = args.Append(&b.varLengthPool, v)
	}
	ret.AsReturn(args)
	b.InsertInstruction(ret)
}

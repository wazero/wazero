package bench

import (
	"context"
	_ "embed"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/internal/platform"
)

// constFoldWasm is compiled from testdata/const_fold.wat. Its exported functions run
// constant integer arithmetic in a hot loop — the kind of pattern an unoptimized wasm
// producer leaves behind and that passConstFoldingArithmeticOpt collapses. wazevo has no
// LICM, so without folding those per-iteration ALU ops re-run every iteration.
//
//go:embed testdata/const_fold.wasm
var constFoldWasm []byte

// BenchmarkConstFolding measures execution of the const-folding fixtures. There is no
// in-process on/off switch for the pass, so the A/B is done across revisions: run this on
// main and on the branch that adds the pass, then compare with benchstat, e.g.
//
//	git stash; go test -run x -bench ConstFolding -count 10 ./... > /tmp/off.txt   # main
//	git stash pop; go test -run x -bench ConstFolding -count 10 ./... > /tmp/on.txt # branch
//	benchstat /tmp/off.txt /tmp/on.txt
func BenchmarkConstFolding(b *testing.B) {
	if !platform.CompilerSupported() {
		b.Skip()
	}
	ctx := context.Background()
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	defer r.Close(ctx)

	m, err := r.Instantiate(ctx, constFoldWasm)
	if err != nil {
		b.Fatal(err)
	}

	const iterations = uint64(2_000_000)
	for _, name := range []string{"const_chain", "identities"} {
		fn := m.ExportedFunction(name)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := fn.Call(ctx, iterations); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

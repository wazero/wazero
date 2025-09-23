package frontend

import (
	"github.com/wazero/wazero/internal/engine/wazevo/ssa"
	"github.com/wazero/wazero/internal/wasm"
)

func FunctionIndexToFuncRef(idx wasm.Index) ssa.FuncRef {
	return ssa.FuncRef(idx)
}

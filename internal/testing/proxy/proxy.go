package proxy

import (
	"github.com/wazero/wazero"
	"github.com/wazero/wazero/api"
	"github.com/wazero/wazero/experimental"
	"github.com/wazero/wazero/experimental/logging"
	"github.com/wazero/wazero/internal/leb128"
	"github.com/wazero/wazero/internal/testing/binaryencoding"
	"github.com/wazero/wazero/internal/wasm"
)

const proxyModuleName = "internal/testing/proxy/proxy.go"

// NewLoggingListenerFactory is like logging.NewHostLoggingListenerFactory,
// except it skips logging proxying functions from NewModuleBinary.
func NewLoggingListenerFactory(writer logging.Writer, scopes logging.LogScopes) experimental.FunctionListenerFactory {
	return &loggingListenerFactory{logging.NewHostLoggingListenerFactory(writer, scopes)}
}

type loggingListenerFactory struct {
	delegate experimental.FunctionListenerFactory
}

// NewFunctionListener implements the same method as documented on
// experimental.FunctionListener.
func (f *loggingListenerFactory) NewFunctionListener(fnd api.FunctionDefinition) experimental.FunctionListener {
	if fnd.ModuleName() == proxyModuleName {
		return nil // don't log proxy stuff
	}
	return f.delegate.NewFunctionListener(fnd)
}

// NewModuleBinary creates the proxy module to proxy a function call against
// all the exported functions in `proxyTarget`, and returns its encoded binary.
// The resulting module exports the proxy functions whose names are exactly the same
// as the proxy destination.
//
// This is used to test host call implementations. If logging, use
// NewLoggingListenerFactory to avoid messages from the proxying module.
func NewModuleBinary(moduleName string, proxyTarget wazero.CompiledModule) []byte {
	funcDefs := proxyTarget.ExportedFunctions()
	funcNum := uint32(len(funcDefs))
	proxyModule := &wasm.Module{
		MemorySection: &wasm.Memory{Min: 1},
		ExportSection: []wasm.Export{{Name: "memory", Type: api.ExternTypeMemory}},
		NameSection:   &wasm.NameSection{ModuleName: proxyModuleName},
	}
	var cnt wasm.Index
	for _, def := range funcDefs {
		proxyModule.TypeSection = append(proxyModule.TypeSection, wasm.FunctionType{
			Params: def.ParamTypes(), Results: def.ResultTypes(),
		})

		// Imports the function.
		name := def.ExportNames()[0]
		proxyModule.ImportSection = append(proxyModule.ImportSection, wasm.Import{
			Module:   moduleName,
			Name:     name,
			DescFunc: cnt,
		})

		// Ensures that type of the proxy function matches the imported function.
		proxyModule.FunctionSection = append(proxyModule.FunctionSection, cnt)

		// Build the function body of the proxy function.
		var body []byte
		for i := range def.ParamTypes() {
			body = append(body, wasm.OpcodeLocalGet)
			body = append(body, leb128.EncodeUint32(uint32(i))...)
		}

		body = append(body, wasm.OpcodeCall)
		body = append(body, leb128.EncodeUint32(cnt)...)
		body = append(body, wasm.OpcodeEnd)
		proxyModule.CodeSection = append(proxyModule.CodeSection, wasm.Code{Body: body})

		proxyFuncIndex := cnt + funcNum
		// Assigns the same params name as the imported one.
		paramNames := wasm.NameMapAssoc{Index: proxyFuncIndex}
		for i, n := range def.ParamNames() {
			paramNames.NameMap = append(paramNames.NameMap, wasm.NameAssoc{Index: wasm.Index(i), Name: n})
		}
		proxyModule.NameSection.LocalNames = append(proxyModule.NameSection.LocalNames, paramNames)

		// Plus, assigns the same function name.
		proxyModule.NameSection.FunctionNames = append(proxyModule.NameSection.FunctionNames,
			wasm.NameAssoc{Index: proxyFuncIndex, Name: name})

		// Finally, exports the proxy function with the same name as the imported one.
		proxyModule.ExportSection = append(proxyModule.ExportSection, wasm.Export{
			Type:  wasm.ExternTypeFunc,
			Name:  name,
			Index: proxyFuncIndex,
		})
		cnt++
	}
	return binaryencoding.EncodeModule(proxyModule)
}

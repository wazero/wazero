package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// This example demonstrates the pause/resume host function pattern used when
// embedding interpreters (e.g. Python, Lua) in WebAssembly via wazero.
//
// The pattern works as follows:
//
//  1. The host compiles and starts execution of guest code inside WASM.
//  2. When the guest needs to call a host-defined function, WASM execution
//     pauses and returns a status code indicating a pending function call.
//  3. The host reads a JSON-encoded request from WASM memory, dispatches the
//     call, and writes a JSON-encoded result back into WASM memory.
//  4. The host resumes WASM execution, repeating until the guest completes.
//
// This approach is useful because WASM is single-threaded and cannot perform
// I/O on its own. The pause/resume loop lets an embedded interpreter call
// arbitrary host functions (HTTP, database, file I/O) without blocking the
// WASM runtime.
//
// See README.md for a full description.
func main() {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Register host functions in the "env" module. These are the functions
	// the WASM guest can invoke through the pause/resume mechanism.
	hostFunctions := map[string]func(args map[string]any) (any, error){
		"sqrt": func(args map[string]any) (any, error) {
			x, ok := args["x"].(float64)
			if !ok {
				return nil, fmt.Errorf("sqrt: expected float64 argument 'x'")
			}
			return math.Sqrt(x), nil
		},
		"greet": func(args map[string]any) (any, error) {
			name, _ := args["name"].(string)
			return fmt.Sprintf("Hello, %s!", name), nil
		},
	}

	// Build the host module with memory management helpers.
	// In a real interpreter embedding, the WASM module exports alloc/dealloc
	// functions. Here we demonstrate the host-side orchestration pattern.
	builder := r.NewHostModuleBuilder("env")

	// Export a host function that the guest calls to signal a pause.
	// The guest writes a JSON request to memory, then calls this function
	// with (requestPtr, requestLen). The host reads the request, dispatches
	// it, and writes the response back.
	//
	// In practice, the guest side is compiled from an interpreter (e.g.
	// RustPython compiled to WASM). This example simulates that interaction.
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, requestPtr, requestLen uint32) uint32 {
			// Read the function call request from WASM memory.
			requestBytes, ok := m.Memory().Read(requestPtr, requestLen)
			if !ok {
				log.Printf("failed to read request at %d len %d", requestPtr, requestLen)
				return 0
			}

			var request struct {
				Function string         `json:"function"`
				Args     map[string]any `json:"args"`
			}
			if err := json.Unmarshal(requestBytes, &request); err != nil {
				log.Printf("failed to parse request: %v", err)
				return 0
			}

			fmt.Printf("host: received call to %q with args %v\n", request.Function, request.Args)

			// Dispatch to the registered host function.
			handler, exists := hostFunctions[request.Function]
			if !exists {
				log.Printf("unknown function: %s", request.Function)
				return 0
			}

			result, err := handler(request.Args)
			if err != nil {
				log.Printf("function %s failed: %v", request.Function, err)
				return 0
			}

			// Marshal the response and write it back to WASM memory.
			response := map[string]any{
				"status": "ok",
				"value":  result,
			}
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Printf("failed to marshal response: %v", err)
				return 0
			}

			// Write response to a known offset (simplified for this example).
			// In production, you would use the guest's allocator.
			responseOffset := uint32(4096) // Use a dedicated response buffer region.
			if !m.Memory().Write(responseOffset, responseBytes) {
				log.Printf("failed to write response at %d", responseOffset)
				return 0
			}

			fmt.Printf("host: returning result: %v\n", result)

			// Return the length of the response. The guest knows where to
			// read it from (the agreed-upon offset).
			return uint32(len(responseBytes))
		}).
		Export("host_call")

	// Export a function the guest uses to retrieve the current response
	// buffer offset. This separates the "where" from the "how much".
	builder.NewFunctionBuilder().
		WithFunc(func() uint32 {
			return 4096 // Fixed response buffer offset for this example.
		}).
		Export("host_response_offset")

	_, err := builder.Instantiate(ctx)
	if err != nil {
		log.Panicln(err)
	}

	// In a real scenario, you would load a WASM binary compiled from an
	// interpreter (like RustPython). The binary would export functions like:
	//
	//   wasm_alloc(size u32) -> ptr u32
	//   wasm_dealloc(ptr u32, size u32)
	//   compile(code_ptr u32, code_len u32, ...) -> handle u32
	//   start(handle u32, inputs_ptr u32, inputs_len u32, ...) -> status u32
	//   resume(snapshot u32, value_ptr u32, value_len u32) -> status u32
	//   result_len() -> u32
	//   result_read(buf_ptr u32, buf_len u32)
	//
	// The execution loop would look like:
	//
	//   handle := compile(code, inputNames, extFuncNames)
	//   status := start(handle, inputValues, limits)
	//   for {
	//       result := readResult()
	//       switch status {
	//       case StatusComplete:
	//           return result.Value
	//       case StatusFunctionCall:
	//           returnVal := callHostFunction(result.FunctionName, result.Args)
	//           status = resume(result.SnapshotHandle, returnVal)
	//       case StatusError:
	//           return result.Error
	//       }
	//   }

	fmt.Println("=== Pause/Resume Host Function Pattern ===")
	fmt.Println()
	fmt.Println("This example demonstrates the pattern used to embed interpreters in WASM.")
	fmt.Println("The key insight: WASM execution pauses when the guest needs host services,")
	fmt.Println("the host fulfills the request, then resumes WASM execution.")
	fmt.Println()

	// Demonstrate the pattern with a simulated execution loop.
	simulatePauseResumeLoop(hostFunctions)
}

// simulatePauseResumeLoop demonstrates the host-side orchestration of the
// pause/resume pattern without requiring an actual WASM interpreter binary.
//
// In production, each "pause" is a return from a WASM exported function with
// a status code, and each "resume" is a call back into WASM with the result.
func simulatePauseResumeLoop(hostFunctions map[string]func(args map[string]any) (any, error)) {
	// Simulate a sequence of function calls that an embedded interpreter
	// might make while executing a script like:
	//
	//   message = greet("wazero")
	//   result = sqrt(144)
	//   print(message, result)
	//
	calls := []struct {
		Function string         `json:"function"`
		Args     map[string]any `json:"args"`
	}{
		{Function: "greet", Args: map[string]any{"name": "wazero"}},
		{Function: "sqrt", Args: map[string]any{"x": float64(144)}},
	}

	fmt.Println("--- Simulated execution ---")
	for i, call := range calls {
		fmt.Printf("\n[step %d] guest pauses: calling %q\n", i+1, call.Function)

		// In real code, this JSON would be read from WASM memory after the
		// guest returned a StatusFunctionCall status code.
		requestBytes, _ := json.Marshal(call)
		fmt.Printf("[step %d] request JSON: %s\n", i+1, string(requestBytes))

		handler := hostFunctions[call.Function]
		result, err := handler(call.Args)
		if err != nil {
			log.Panicf("function call failed: %v", err)
		}

		// In real code, this result would be written to WASM memory and
		// the guest resumed via the resume() exported function.
		responseBytes, _ := json.Marshal(map[string]any{"status": "ok", "value": result})
		fmt.Printf("[step %d] response JSON: %s\n", i+1, string(responseBytes))
		fmt.Printf("[step %d] guest resumes with result: %v\n", i+1, result)
	}

	fmt.Println("\n[done] guest execution complete")
}

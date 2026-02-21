## Pause/Resume Host Function Pattern

This example demonstrates the **pause/resume host function pattern** used when
embedding interpreters (such as Python via RustPython, Lua, or other scripting
languages) inside WebAssembly using wazero.

```bash
$ go run main.go
=== Pause/Resume Host Function Pattern ===

This example demonstrates the pattern used to embed interpreters in WASM.
The key insight: WASM execution pauses when the guest needs host services,
the host fulfills the request, then resumes WASM execution.

--- Simulated execution ---

[step 1] guest pauses: calling "greet"
[step 1] request JSON: {"function":"greet","args":{"name":"wazero"}}
[step 1] response JSON: {"status":"ok","value":"Hello, wazero!"}
[step 1] guest resumes with result: Hello, wazero!

[step 2] guest pauses: calling "sqrt"
[step 2] request JSON: {"function":"sqrt","args":{"x":144}}
[step 2] response JSON: {"status":"ok","value":12}
[step 2] guest resumes with result: 12

[done] guest execution complete
```

### Background

WebAssembly is single-threaded and has no built-in I/O capabilities. When
embedding an interpreter (like Python compiled to WASM), the interpreter
needs a way to call back into the host for operations like network requests,
file I/O, database queries, or calling application-specific functions.

The pause/resume pattern solves this with a cooperative execution loop:

```
       Host (Go)                    Guest (WASM)
       ─────────                    ────────────
    1. compile(code) ──────────►  Parse & compile
    2. start(inputs) ──────────►  Begin execution
                                       │
                                       ▼
    3. ◄──── StatusFunctionCall ── Need host function
       Read request from memory
       Execute host function
       Write result to memory
    4. resume(result) ─────────►  Continue execution
                                       │
                                       ▼
    5. ◄──── StatusFunctionCall ── Need another function
       (repeat steps 3-4)
                                       │
                                       ▼
    6. ◄──── StatusComplete ────── Execution finished
       Read final result
```

### Key Concepts

#### Memory Management

The host and guest share linear memory. The pattern uses `alloc` and `dealloc`
functions exported by the WASM module to manage memory safely:

```go
// Write data to WASM memory using the guest's allocator.
results, _ := alloc.Call(ctx, uint64(len(data)))
ptr := uint32(results[0])
mod.Memory().Write(ptr, data)

// Free when done.
dealloc.Call(ctx, uint64(ptr), uint64(len(data)))
```

#### JSON Data Exchange

The host and guest exchange structured data through JSON serialized into WASM
linear memory. This allows passing complex arguments and return values without
defining a rigid binary protocol:

```go
// Host reads a function call request from WASM memory.
requestBytes, _ := mod.Memory().Read(ptr, length)
var request struct {
    Function string         `json:"function"`
    Args     map[string]any `json:"args"`
}
json.Unmarshal(requestBytes, &request)

// Host writes the result back.
responseBytes, _ := json.Marshal(map[string]any{"value": result})
mod.Memory().Write(responsePtr, responseBytes)
```

#### Status Codes

The guest returns numeric status codes to indicate why execution paused:

| Status | Meaning | Host Action |
|--------|---------|-------------|
| 0 | Error | Read error message, abort |
| 1 | Complete | Read final result |
| 2 | Function Call | Dispatch function, resume with result |
| 3 | OS/System Call | Handle system call, resume with result |

### Real-World Usage

This pattern is used by projects like
[monty-go](https://github.com/fugue-labs/monty-go), which embeds a Python
interpreter (RustPython compiled to WASM) in Go applications. The embedded
Python can call Go-defined functions as if they were native Python functions,
with the pause/resume loop handling the cross-boundary communication
transparently.

### Where Next?

The following examples cover related wazero concepts:

* [allocation](../allocation) - How to pass strings in and out of WASM functions
  using memory allocation.
* [import-go](../import-go) - How to define, import and call a Go-defined
  function from a WASM-defined function.

;; fuel.wat - module used by experimental.Example_fuel. Builds with
;;     wat2wasm fuel.wat
(module
  (func (export "add") (param i32 i32) (result i32)
    local.get 0
    local.get 1
    i32.add
    )
)

;; Bulk instructions

;; invalid uses

(assert_invalid
  (module
    (type $a (array funcref))

    (elem $e1 funcref)

    (func (export "array.init_elem-immutable") (param $1 (ref $a))
      (array.init_elem $a $e1 (local.get $1) (i32.const 0) (i32.const 0) (i32.const 0))
    )
  )
  "immutable array"
)

(assert_invalid
  (module
    (type $a (array (mut i8)))

    (elem $e1 funcref)

    (func (export "array.init_elem-invalid-1") (param $1 (ref $a))
      (array.init_elem $a $e1 (local.get $1) (i32.const 0) (i32.const 0) (i32.const 0))
    )
  )
  "type mismatch"
)

(assert_invalid
  (module
    (type $a (array (mut funcref)))

    (elem $e1 externref)

    (func (export "array.init_elem-invalid-2") (param $1 (ref $a))
      (array.init_elem $a $e1 (local.get $1) (i32.const 0) (i32.const 0) (i32.const 0))
    )
  )
  "type mismatch"
)

(module
  (type $arrref_mut (array (mut funcref)))

  (global $g_arrref_mut (ref $arrref_mut) (array.new_default $arrref_mut (i32.const 12)))

  (table $t 1 funcref)

  (elem $e1 func $zero $one $two $three $four $five $six $seven $eight $nine $ten $eleven)

  (func $zero (result i32) (i32.const 0))
  (func $one (result i32) (i32.const 1))
  (func $two (result i32) (i32.const 2))
  (func $three (result i32) (i32.const 3))
  (func $four (result i32) (i32.const 4))
  (func $five (result i32) (i32.const 5))
  (func $six (result i32) (i32.const 6))
  (func $seven (result i32) (i32.const 7))
  (func $eight (result i32) (i32.const 8))
  (func $nine (result i32) (i32.const 9))
  (func $ten (result i32) (i32.const 10))
  (func $eleven (result i32) (i32.const 11))

  (func (export "array_call_nth") (param $n i32) (result i32)
    (table.set $t (i32.const 0) (array.get $arrref_mut (global.get $g_arrref_mut) (local.get $n)))
    (call_indirect $t (result i32) (i32.const 0))
  )

  (func (export "array_init_elem-null")
    (array.init_elem $arrref_mut $e1 (ref.null $arrref_mut) (i32.const 0) (i32.const 0) (i32.const 0))
  )

  (func (export "array_init_elem") (param $1 i32) (param $2 i32) (param $3 i32)
    (array.init_elem $arrref_mut $e1 (global.get $g_arrref_mut) (local.get $1) (local.get $2) (local.get $3))
  )

  (func (export "drop_segs")
    (elem.drop $e1)
  )
)

;; null array argument traps
(assert_trap (invoke "array_init_elem-null") "null array reference")

;; OOB initial index traps
(assert_trap (invoke "array_init_elem" (i32.const 13) (i32.const 0) (i32.const 0)) "out of bounds array access")
(assert_trap (invoke "array_init_elem" (i32.const 0) (i32.const 13) (i32.const 0)) "out of bounds table access")

;; OOB length traps
(assert_trap (invoke "array_init_elem" (i32.const 0) (i32.const 0) (i32.const 13)) "out of bounds array access")
(assert_trap (invoke "array_init_elem" (i32.const 0) (i32.const 0) (i32.const 13)) "out of bounds array access")

;; start index = array size, len = 0 doesn't trap
(assert_return (invoke "array_init_elem" (i32.const 12) (i32.const 0) (i32.const 0)))
(assert_return (invoke "array_init_elem" (i32.const 0) (i32.const 12) (i32.const 0)))

;; check arrays were not modified
(assert_trap (invoke "array_call_nth" (i32.const 0)) "uninitialized element")
(assert_trap (invoke "array_call_nth" (i32.const 5)) "uninitialized element")
(assert_trap (invoke "array_call_nth" (i32.const 11)) "uninitialized element")
(assert_trap (invoke "array_call_nth" (i32.const 12)) "out of bounds array access")

;; normal cases
(assert_return (invoke "array_init_elem" (i32.const 2) (i32.const 3) (i32.const 2)))
(assert_trap (invoke "array_call_nth" (i32.const 1)) "uninitialized element")
(assert_return (invoke "array_call_nth" (i32.const 2)) (i32.const 3))
(assert_return (invoke "array_call_nth" (i32.const 3)) (i32.const 4))
(assert_trap (invoke "array_call_nth" (i32.const 4)) "uninitialized element")

;; init_data/elem with dropped segments traps for non-zero length
(assert_return (invoke "drop_segs"))
(assert_return (invoke "array_init_elem" (i32.const 0) (i32.const 0) (i32.const 0)))
(assert_trap (invoke "array_init_elem" (i32.const 0) (i32.const 0) (i32.const 1)) "out of bounds table access")

(module
  (type $arrref_mut (array (mut arrayref)))

  (global $g_arrref_mut (ref $arrref_mut) (array.new_default $arrref_mut (i32.const 2)))

  (elem $e1 arrayref
    (item (array.new_default $arrref_mut (i32.const 1)))
    (item (array.new_default $arrref_mut (i32.const 2)))
  )

  (func (export "array_init_elem") (param $1 i32) (param $2 i32) (param $3 i32)
    (array.init_elem $arrref_mut $e1 (global.get $g_arrref_mut) (local.get $1) (local.get $2) (local.get $3))
  )

  (func (export "array_len_nth") (param $n i32) (result i32)
    (array.len (array.get $arrref_mut (global.get $g_arrref_mut) (local.get $n)))
  )

  (func (export "array_eq_elems") (param $i i32) (param $j i32) (result i32)
    (ref.eq
      (array.get $arrref_mut (global.get $g_arrref_mut) (local.get $i))
      (array.get $arrref_mut (global.get $g_arrref_mut) (local.get $j))
    )
  )
)

;; Array starts uninitialized
(assert_trap (invoke "array_len_nth" (i32.const 0)) "null array reference")
(assert_trap (invoke "array_len_nth" (i32.const 1)) "null array reference")

;; Initialize the array
(assert_return (invoke "array_init_elem" (i32.const 0) (i32.const 0) (i32.const 2)))
(assert_return (invoke "array_len_nth" (i32.const 0)) (i32.const 1))
(assert_return (invoke "array_len_nth" (i32.const 1)) (i32.const 2))
(assert_return (invoke "array_eq_elems" (i32.const 0) (i32.const 1)) (i32.const 0))

;; Copy the first element at the second index and check that they are equal.
(assert_return (invoke "array_init_elem" (i32.const 1) (i32.const 0) (i32.const 1)))
(assert_return (invoke "array_len_nth" (i32.const 0)) (i32.const 1))
(assert_return (invoke "array_len_nth" (i32.const 1)) (i32.const 1))
(assert_return (invoke "array_eq_elems" (i32.const 0) (i32.const 1)) (i32.const 1))

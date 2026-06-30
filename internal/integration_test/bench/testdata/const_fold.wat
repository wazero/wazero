(module
  ;; const_chain: a chain of const-op-const that the fold pass collapses to ONE
  ;; constant. wazevo has no LICM, so without folding these 6 ALU ops run every
  ;; iteration; with folding they become a single constant load.
  (func (export "const_chain") (param $n i32) (result i32)
    (local $i i32) (local $acc i32)
    (block $B (loop $L
      (br_if $B (i32.ge_u (local.get $i) (local.get $n)))
      (local.set $acc
        (i32.add (local.get $acc)
          (i32.xor
            (i32.or
              (i32.and
                (i32.add (i32.mul (i32.const 6) (i32.const 7)) (i32.const 1337))
                (i32.const 255))
              (i32.const 256))
            (i32.const 85))))
      (local.set $i (i32.add (local.get $i) (i32.const 1)))
      (br $L)))
    (local.get $acc))

  ;; identities: x*1, x+0, x&x, x|0 on the loop variable. The fold pass rewrites
  ;; each to just x, deleting the multiply/and/or per iteration.
  (func (export "identities") (param $n i32) (result i32)
    (local $i i32) (local $acc i32)
    (block $B (loop $L
      (br_if $B (i32.ge_u (local.get $i) (local.get $n)))
      (local.set $acc
        (i32.add (local.get $acc)
          (i32.add
            (i32.add (i32.mul (local.get $i) (i32.const 1))      ;; i*1 -> i
                     (i32.add (local.get $i) (i32.const 0)))     ;; i+0 -> i
            (i32.add (i32.and (local.get $i) (local.get $i))     ;; i&i -> i
                     (i32.or  (local.get $i) (i32.const 0))))))  ;; i|0 -> i
      (local.set $i (i32.add (local.get $i) (i32.const 1)))
      (br $L)))
    (local.get $acc))
)

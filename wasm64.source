#include <stdint.h>

#if __wasm32__
#error "wasm64.c requires wasm64 pointers"
#endif

__attribute__((export_name("add64")))
uint64_t add64(uint64_t a, uint64_t b) {
    return a + b;
}

__attribute__((export_name("pointer_size_bits")))
uint32_t pointer_size_bits(void) {
    return sizeof(void *) * 8;
}

__attribute__((export_name("high_address_marker")))
uint64_t high_address_marker(void) {
    uintptr_t ptr = (uintptr_t)0x100000000ULL;
    return ptr >> 32;
}

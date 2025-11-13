//go:build linux || darwin

package platform

import "syscall"

const noopMprotectRX = false

// MprotectRX is like syscall.Mprotect with RX permission.
func MprotectRX(b []byte) (err error) {
	return syscall.Mprotect(b, syscall.PROT_READ|syscall.PROT_EXEC)
}

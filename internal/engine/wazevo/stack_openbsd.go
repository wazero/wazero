package wazevo

import "golang.org/x/sys/unix"

func allocateStack(size int) []byte {
	stack, err := unix.Mmap(
		-1,
		0,
		size,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_ANON|unix.MAP_PRIVATE|unix.MAP_STACK,
	)
	if err != nil {
		panic(err)
	}
	return stack
}

func releaseStack(stack []byte) {
	if err := unix.Munmap(stack); err != nil {
		panic(err)
	}
}

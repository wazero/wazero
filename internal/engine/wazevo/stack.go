//go:build !openbsd

package wazevo

func allocateStack(size int) []byte {
	return make([]byte, size)
}

func releaseStack([]byte) {}

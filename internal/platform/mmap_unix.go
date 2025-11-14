//go:build linux || darwin || freebsd || netbsd || dragonfly || solaris

package platform

import "syscall"

func munmapCodeSegment(code []byte) error {
	return syscall.Munmap(code)
}

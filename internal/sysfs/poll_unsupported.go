//go:build !(linux || darwin || windows) || tinygo

package sysfs

import (
	"github.com/wazero/wazero/experimental/sys"
	"github.com/wazero/wazero/internal/fsapi"
)

// poll implements `Poll` as documented on fsapi.File via a file descriptor.
func poll(uintptr, fsapi.Pflag, int32) (bool, sys.Errno) {
	return false, sys.ENOSYS
}

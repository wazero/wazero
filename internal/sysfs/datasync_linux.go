//go:build linux && !tinygo

package sysfs

import (
	"os"
	"syscall"

	"github.com/wazero/wazero/experimental/sys"
)

func datasync(f *os.File) sys.Errno {
	return sys.UnwrapOSError(syscall.Fdatasync(int(f.Fd())))
}

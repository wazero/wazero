//go:build tinygo

package sysfs

import (
	"io/fs"

	experimentalsys "github.com/wazero/wazero/experimental/sys"
	"github.com/wazero/wazero/sys"
)

func inoFromFileInfo(_ string, info fs.FileInfo) (sys.Inode, experimentalsys.Errno) {
	return 0, experimentalsys.ENOTSUP
}

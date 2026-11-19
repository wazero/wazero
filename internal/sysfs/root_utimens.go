package sysfs

import (
	"os"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/sys"
)

// rootUtimens implements Utimens for dirFS using os.Root methods.
//
// os.Root.Chtimes does not support UTIME_OMIT directly. When one timestamp
// is omitted, we stat the file first to preserve the existing value using
// sys.NewStat_t which extracts platform-specific atime/mtime.
func rootUtimens(root *os.Root, path string, atim, mtim int64) experimentalsys.Errno {
	if atim == experimentalsys.UTIME_OMIT && mtim == experimentalsys.UTIME_OMIT {
		return 0
	}

	if atim == experimentalsys.UTIME_OMIT || mtim == experimentalsys.UTIME_OMIT {
		info, err := root.Stat(path)
		if err != nil {
			return experimentalsys.UnwrapOSError(err)
		}
		st := sys.NewStat_t(info)
		if atim == experimentalsys.UTIME_OMIT {
			atim = st.Atim
		}
		if mtim == experimentalsys.UTIME_OMIT {
			mtim = st.Mtim
		}
	}
	return experimentalsys.UnwrapOSError(root.Chtimes(path, epochNanosToTime(atim), epochNanosToTime(mtim)))
}

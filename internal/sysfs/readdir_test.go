package sysfs

import (
	"io"
	"io/fs"
	"testing"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/sys"
)

// fakeDir follows the contract of os.File.Readdirnames. A positive n returns
// at most n names, and io.EOF when no names are left.
type fakeDir struct{ names []string }

func (d *fakeDir) Readdirnames(n int) ([]string, error) {
	if n <= 0 || n >= len(d.names) {
		names := d.names
		d.names = nil
		if n > 0 && len(names) == 0 {
			return nil, io.EOF
		}
		return names, nil
	}
	names := d.names[:n]
	d.names = d.names[n:]
	return names, nil
}

func TestReaddirnames(t *testing.T) {
	// On darwin, lstat of the kqueue descriptor in /dev/fd returns EBADF.
	// See https://go.dev/issue/80143
	lstat := func(path string) (sys.Stat_t, experimentalsys.Errno) {
		switch path {
		case "dir/kqueue":
			return sys.Stat_t{}, experimentalsys.EBADF
		case "dir/removed", "renamed", "renamed/file":
			return sys.Stat_t{}, experimentalsys.ENOENT
		case "dir/broken":
			return sys.Stat_t{}, experimentalsys.EIO
		case "dir/sub":
			return sys.Stat_t{Ino: 2, Mode: fs.ModeDir | 0o755}, 0
		default:
			return sys.Stat_t{Ino: 1, Mode: 0o644}, 0
		}
	}

	t.Run("skips entries the host cannot stat", func(t *testing.T) {
		dir := &fakeDir{names: []string{"file", "kqueue", "removed", "sub"}}

		dirents, errno := readdirnames(dir, "dir", -1, lstat)
		require.EqualErrno(t, 0, errno)
		require.Equal(t, []experimentalsys.Dirent{
			{Name: "file", Ino: 1},
			{Name: "sub", Ino: 2, Type: fs.ModeDir},
		}, dirents)
	})

	t.Run("keeps reading after a batch of skipped entries", func(t *testing.T) {
		dir := &fakeDir{names: []string{"kqueue", "file"}}

		dirents, errno := readdirnames(dir, "dir", 1, lstat)
		require.EqualErrno(t, 0, errno)
		require.Equal(t, []experimentalsys.Dirent{{Name: "file", Ino: 1}}, dirents)

		dirents, errno = readdirnames(dir, "dir", 1, lstat)
		require.EqualErrno(t, 0, errno)
		require.Zero(t, len(dirents))
	})

	t.Run("returns no entries when it skips every entry", func(t *testing.T) {
		dir := &fakeDir{names: []string{"kqueue"}}

		dirents, errno := readdirnames(dir, "dir", 1, lstat)
		require.EqualErrno(t, 0, errno)
		require.Zero(t, len(dirents))
	})

	t.Run("returns ENOENT after the directory is renamed or removed", func(t *testing.T) {
		dir := &fakeDir{names: []string{"file"}}

		_, errno := readdirnames(dir, "renamed", -1, lstat)
		require.EqualErrno(t, experimentalsys.ENOENT, errno)
	})

	t.Run("returns other stat errors", func(t *testing.T) {
		dir := &fakeDir{names: []string{"file", "broken"}}

		_, errno := readdirnames(dir, "dir", -1, lstat)
		require.EqualErrno(t, experimentalsys.EIO, errno)
	})
}

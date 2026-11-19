package sysfs

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/sys"
)

func ensureTrailingPathSeparator(dir string) string {
	if !os.IsPathSeparator(dir[len(dir)-1]) {
		return dir + string(os.PathSeparator)
	}
	return dir
}

func DirFS(dir string) experimentalsys.FS {
	root, err := os.OpenRoot(dir)
	if err != nil {
		// Defer the error to the first file operation, matching previous behavior
		// where DirFS("nonexistent") succeeded but OpenFile returned ENOENT.
		// For the case where dir is not a directory (ENOTDIR), fall back to the
		// old path-based behavior so that OpenFile can still open the path and
		// Readdir correctly returns EBADF.
		return &legacyDirFS{
			dir:        dir,
			cleanedDir: ensureTrailingPathSeparator(dir),
		}
	}
	return &dirFS{
		dir:  dir,
		root: root,
	}
}

// dirFS is not exported because the input fields must be maintained together.
type dirFS struct {
	experimentalsys.UnimplementedFS

	dir     string
	root    *os.Root
	openErr error // non-nil if os.OpenRoot failed; deferred to first operation
}

// String implements fmt.Stringer
func (d *dirFS) String() string {
	return d.dir
}

// checkRoot returns the deferred error if os.OpenRoot failed during construction.
func (d *dirFS) checkRoot() experimentalsys.Errno {
	if d.root == nil {
		return experimentalsys.UnwrapOSError(d.openErr)
	}
	return 0
}

// OpenFile implements the same method as documented on sys.FS
func (d *dirFS) OpenFile(p string, flag experimentalsys.Oflag, perm fs.FileMode) (experimentalsys.File, experimentalsys.Errno) {
	if errno := d.checkRoot(); errno != 0 {
		return nil, errno
	}
	p = d.clean(p)
	f, err := d.root.OpenFile(p, toOsOpenFlag(flag), perm)
	if err != nil {
		return nil, experimentalsys.UnwrapOSError(err)
	}
	// newRootOsFile stores the os.Root so that reopen() stays within the
	// sandbox. The absolute path is kept for Windows readdir inode lookups.
	absPath := filepath.Join(d.root.Name(), p)
	return newRootOsFile(absPath, p, flag, perm, f, d.root), 0
}

// Lstat implements the same method as documented on sys.FS
func (d *dirFS) Lstat(p string) (sys.Stat_t, experimentalsys.Errno) {
	if errno := d.checkRoot(); errno != 0 {
		return sys.Stat_t{}, errno
	}
	info, err := d.root.Lstat(d.clean(p))
	if err != nil {
		return sys.Stat_t{}, experimentalsys.UnwrapOSError(err)
	}
	return sys.NewStat_t(info), 0
}

// Stat implements the same method as documented on sys.FS
func (d *dirFS) Stat(p string) (sys.Stat_t, experimentalsys.Errno) {
	if errno := d.checkRoot(); errno != 0 {
		return sys.Stat_t{}, errno
	}
	info, err := d.root.Stat(d.clean(p))
	if err != nil {
		return sys.Stat_t{}, experimentalsys.UnwrapOSError(err)
	}
	return sys.NewStat_t(info), 0
}

// Mkdir implements the same method as documented on sys.FS
func (d *dirFS) Mkdir(p string, perm fs.FileMode) (errno experimentalsys.Errno) {
	if errno = d.checkRoot(); errno != 0 {
		return
	}
	// Strip type bits -- os.Root.Mkdir rejects non-permission bits like fs.ModeDir.
	err := d.root.Mkdir(d.clean(p), perm.Perm())
	if errno = experimentalsys.UnwrapOSError(err); errno == experimentalsys.ENOTDIR {
		errno = experimentalsys.ENOENT
	}
	return
}

// Chmod implements the same method as documented on sys.FS
func (d *dirFS) Chmod(p string, perm fs.FileMode) experimentalsys.Errno {
	if errno := d.checkRoot(); errno != 0 {
		return errno
	}
	err := d.root.Chmod(d.clean(p), perm)
	return experimentalsys.UnwrapOSError(err)
}

// Rename implements the same method as documented on sys.FS
func (d *dirFS) Rename(from, to string) experimentalsys.Errno {
	if errno := d.checkRoot(); errno != 0 {
		return errno
	}
	from, to = d.clean(from), d.clean(to)
	// os.Root.Rename uses RENAME_NOREPLACE which prevents overwriting existing
	// targets. POSIX rename (2) allows overwriting, which WASI expects. To
	// preserve the POSIX behavior, verify the source exists within the root
	// (preventing escape), then use the platform-specific rename with paths
	// rooted at root.Name().
	if _, err := d.root.Lstat(from); err != nil {
		return experimentalsys.UnwrapOSError(err)
	}
	// Verify the target's parent directory is within the root.
	toDir := filepath.Dir(to)
	if toDir != "." {
		if _, err := d.root.Lstat(toDir); err != nil {
			return experimentalsys.UnwrapOSError(err)
		}
	}
	return rename(filepath.Join(d.root.Name(), from), filepath.Join(d.root.Name(), to))
}

// Rmdir implements the same method as documented on sys.FS
func (d *dirFS) Rmdir(p string) experimentalsys.Errno {
	if errno := d.checkRoot(); errno != 0 {
		return errno
	}
	p = d.clean(p)
	// os.Root.Remove works for both files and directories. We need to verify
	// it's a directory first to return the correct errno.
	info, err := d.root.Lstat(p)
	if err != nil {
		return experimentalsys.UnwrapOSError(err)
	}
	if !info.IsDir() {
		return experimentalsys.ENOTDIR
	}
	return experimentalsys.UnwrapOSError(d.root.Remove(p))
}

// Unlink implements the same method as documented on sys.FS
func (d *dirFS) Unlink(p string) (err experimentalsys.Errno) {
	if errno := d.checkRoot(); errno != 0 {
		return errno
	}
	p = d.clean(p)
	// os.Root.Remove works for both files and directories. We need to verify
	// it's not a directory to return the correct errno.
	info, e := d.root.Lstat(p)
	if e != nil {
		return experimentalsys.UnwrapOSError(e)
	}
	if info.IsDir() {
		return experimentalsys.EISDIR
	}
	return experimentalsys.UnwrapOSError(d.root.Remove(p))
}

// Link implements the same method as documented on sys.FS
func (d *dirFS) Link(oldName, newName string) experimentalsys.Errno {
	if errno := d.checkRoot(); errno != 0 {
		return errno
	}
	err := d.root.Link(d.clean(oldName), d.clean(newName))
	return experimentalsys.UnwrapOSError(err)
}

// Symlink implements the same method as documented on sys.FS
func (d *dirFS) Symlink(oldName, link string) experimentalsys.Errno {
	if errno := d.checkRoot(); errno != 0 {
		return errno
	}
	// WASI requires that symlink targets must not be absolute paths.
	// https://github.com/WebAssembly/wasi-filesystem/blob/v0.2.0/path-resolution.md#symlinks
	if path.IsAbs(oldName) {
		return experimentalsys.EPERM
	}
	// Note: os.Root.Symlink does not validate the symlink target, which means
	// a guest can still create a symlink pointing outside the root. However,
	// this is safe because all other os.Root methods (Open, Stat, etc.) refuse
	// to follow symlinks that escape the root. The escape is blocked at
	// resolution time, not creation time.
	err := d.root.Symlink(oldName, d.clean(link))
	return experimentalsys.UnwrapOSError(err)
}

// Readlink implements the same method as documented on sys.FS
func (d *dirFS) Readlink(p string) (string, experimentalsys.Errno) {
	if errno := d.checkRoot(); errno != 0 {
		return "", errno
	}
	dst, err := d.root.Readlink(d.clean(p))
	if err != nil {
		return "", experimentalsys.UnwrapOSError(err)
	}
	return platform.ToPosixPath(dst), 0
}

// Utimens implements the same method as documented on sys.FS
func (d *dirFS) Utimens(p string, atim, mtim int64) experimentalsys.Errno {
	if errno := d.checkRoot(); errno != 0 {
		return errno
	}
	return rootUtimens(d.root, d.clean(p), atim, mtim)
}

// clean normalizes the path for use with os.Root methods.
func (d *dirFS) clean(p string) string {
	switch p {
	case "", ".", "/":
		return "."
	}
	return p
}

// legacyDirFS is the old path-concatenation based filesystem, used as a
// fallback when os.OpenRoot fails (e.g. directory doesn't exist, or the
// path points to a file instead of a directory).
type legacyDirFS struct {
	experimentalsys.UnimplementedFS

	dir        string
	cleanedDir string
}

func (d *legacyDirFS) String() string { return d.dir }

func (d *legacyDirFS) OpenFile(p string, flag experimentalsys.Oflag, perm fs.FileMode) (experimentalsys.File, experimentalsys.Errno) {
	return OpenOSFile(d.join(p), flag, perm)
}

func (d *legacyDirFS) Lstat(p string) (sys.Stat_t, experimentalsys.Errno) {
	return lstat(d.join(p))
}

func (d *legacyDirFS) Stat(p string) (sys.Stat_t, experimentalsys.Errno) {
	return stat(d.join(p))
}

func (d *legacyDirFS) Mkdir(p string, perm fs.FileMode) (errno experimentalsys.Errno) {
	err := os.Mkdir(d.join(p), perm)
	if errno = experimentalsys.UnwrapOSError(err); errno == experimentalsys.ENOTDIR {
		errno = experimentalsys.ENOENT
	}
	return
}

func (d *legacyDirFS) Chmod(p string, perm fs.FileMode) experimentalsys.Errno {
	return experimentalsys.UnwrapOSError(os.Chmod(d.join(p), perm))
}

func (d *legacyDirFS) Rename(from, to string) experimentalsys.Errno {
	return rename(d.join(from), d.join(to))
}

func (d *legacyDirFS) Rmdir(p string) experimentalsys.Errno {
	return rmdir(d.join(p))
}

func (d *legacyDirFS) Unlink(p string) experimentalsys.Errno {
	return unlink(d.join(p))
}

func (d *legacyDirFS) Link(oldName, newName string) experimentalsys.Errno {
	return experimentalsys.UnwrapOSError(os.Link(d.join(oldName), d.join(newName)))
}

func (d *legacyDirFS) Symlink(oldName, link string) experimentalsys.Errno {
	if path.IsAbs(oldName) {
		return experimentalsys.EPERM
	}
	return experimentalsys.UnwrapOSError(os.Symlink(oldName, d.join(link)))
}

func (d *legacyDirFS) Readlink(p string) (string, experimentalsys.Errno) {
	dst, err := os.Readlink(d.join(p))
	if err != nil {
		return "", experimentalsys.UnwrapOSError(err)
	}
	return platform.ToPosixPath(dst), 0
}

func (d *legacyDirFS) Utimens(p string, atim, mtim int64) experimentalsys.Errno {
	return utimens(d.join(p), atim, mtim)
}

func (d *legacyDirFS) join(p string) string {
	switch p {
	case "", ".", "/":
		if d.cleanedDir == "/" {
			return "/"
		}
		return d.cleanedDir[:len(d.cleanedDir)-1]
	}
	return d.cleanedDir + p
}

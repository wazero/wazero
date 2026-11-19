package sysfs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

// TestDirFS_EscapesFilename verifies that a file literally named "escapes"
// does not trigger the escape-detection logic in unwrapRootError. The inner
// error message ("permission denied") must be preserved, not mapped to ENOENT.
func TestDirFS_EscapesFilename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits differ on Windows")
	}

	tmpDir := t.TempDir()

	// Create a file named "escapes" with no read permission.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "escapes"), []byte("data"), 0o000))

	testFS := DirFS(tmpDir)

	// Opening it should return EACCES (permission denied), NOT ENOENT.
	_, errno := testFS.OpenFile("escapes", experimentalsys.O_RDONLY, 0)
	require.Equal(t, experimentalsys.EACCES, errno,
		"file named 'escapes' with 000 perms should return EACCES, got 0x%x", uint16(errno))

	// Stat should also return EACCES, not ENOENT.
	// (Lstat doesn't require read perms so it succeeds.)
	_, lstatErrno := testFS.Lstat("escapes")
	require.Equal(t, experimentalsys.Errno(0), lstatErrno,
		"Lstat on 'escapes' should succeed (no read perm needed), got 0x%x", uint16(lstatErrno))

	// A directory named "escapes" with 000 perms: listing should return EACCES.
	require.NoError(t, os.Remove(filepath.Join(tmpDir, "escapes")))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "escapes"), 0o000))

	_, errno = testFS.OpenFile("escapes/file.txt", experimentalsys.O_RDONLY, 0)
	require.True(t, errno == experimentalsys.EACCES || errno == experimentalsys.ENOENT,
		"directory named 'escapes' with 000 perms: got 0x%x", uint16(errno))

	// But a REAL escape must still return ENOENT.
	_, escapeErrno := testFS.OpenFile("../secret", experimentalsys.O_RDONLY, 0)
	require.Equal(t, experimentalsys.ENOENT, escapeErrno,
		"real escape must still return ENOENT, got 0x%x", uint16(escapeErrno))

	// Restore perms so t.TempDir() cleanup works.
	os.Chmod(filepath.Join(tmpDir, "escapes"), 0o755)
}

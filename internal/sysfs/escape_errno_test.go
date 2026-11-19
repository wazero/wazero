package sysfs

import (
	"os"
	"runtime"
	"testing"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

// TestDirFS_EscapeErrnoLeak verifies that a guest cannot distinguish a sandbox
// escape attempt from a missing file by inspecting the returned errno. Both
// must return ENOENT so the sandbox boundary is not observable.
func TestDirFS_EscapeErrnoLeak(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink escape behavior differs on Windows")
	}

	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tmpDir+"/sub", 0o755))
	require.NoError(t, os.WriteFile(tmpDir+"/exists.txt", []byte("hi"), 0o644))

	testFS := DirFS(tmpDir)

	// Baseline: opening a file that genuinely doesn't exist.
	_, baselineErrno := testFS.OpenFile("nonexistent", experimentalsys.O_RDONLY, 0)
	require.True(t, baselineErrno == experimentalsys.ENOENT,
		"baseline missing file should return ENOENT, got 0x%x", uint16(baselineErrno))

	// Escape via "../" must return the SAME errno as a missing file.
	_, escapeErrno := testFS.OpenFile("../secret", experimentalsys.O_RDONLY, 0)
	require.Equal(t, baselineErrno, escapeErrno,
		"escape via ../ returned different errno (0x%x) than missing file (0x%x) -- information leak",
		uint16(escapeErrno), uint16(baselineErrno))

	// Escape via path traversal through a real subdirectory.
	_, traversalErrno := testFS.OpenFile("sub/../../etc/passwd", experimentalsys.O_RDONLY, 0)
	require.True(t, traversalErrno != 0, "path traversal should fail")
	require.Equal(t, baselineErrno, traversalErrno,
		"traversal escape returned different errno (0x%x) than missing file (0x%x) -- information leak",
		uint16(traversalErrno), uint16(baselineErrno))

	// Absolute path escape.
	_, absErrno := testFS.OpenFile("/etc/passwd", experimentalsys.O_RDONLY, 0)
	require.True(t, absErrno != 0, "absolute path should fail")

	// Double-dot filename (NOT an escape -- "..foo" is a valid filename).
	_, dotdotErrno := testFS.OpenFile("..foo", experimentalsys.O_RDONLY, 0)
	require.Equal(t, experimentalsys.ENOENT, dotdotErrno,
		"dotdot filename should return ENOENT (it's a valid name that doesn't exist), got 0x%x",
		uint16(dotdotErrno))

	// Stat variants should also not leak.
	_, statErrno := testFS.Stat("../secret")
	require.Equal(t, baselineErrno, statErrno,
		"Stat escape returned different errno than missing file")

	_, lstatErrno := testFS.Lstat("../secret")
	require.Equal(t, baselineErrno, lstatErrno,
		"Lstat escape returned different errno than missing file")
}

// TestDirFS_EscapesFilenameNoFalsePositive verifies that files whose names
// contain "escapes" do not trigger the escape-to-ENOENT mapping. The inner
// error from os.Root contains "path escapes from parent", but the filename
// itself may also contain "escapes" -- we must check only the inner error.
func TestDirFS_EscapesFilenameNoFalsePositive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("linux-specific")
	}

	tmpDir := t.TempDir()
	testFS := DirFS(tmpDir)

	// Create a file literally named "escapes".
	require.NoError(t, os.WriteFile(tmpDir+"/escapes", []byte("data"), 0o644))
	f, errno := testFS.OpenFile("escapes", experimentalsys.O_RDONLY, 0)
	require.Zero(t, errno, "should open file named 'escapes' without false positive")
	f.Close()

	// Create a file named "escapes" with 000 permissions -- the error
	// message will contain "escapes" in the path but should return EACCES,
	// not ENOENT.
	require.NoError(t, os.Chmod(tmpDir+"/escapes", 0o000))
	_, errno = testFS.OpenFile("escapes", experimentalsys.O_RDONLY, 0)
	require.Equal(t, experimentalsys.EACCES, errno,
		"file named 'escapes' with 000 perms should return EACCES, got 0x%x", uint16(errno))
	require.NoError(t, os.Chmod(tmpDir+"/escapes", 0o644))

	// Directory named "path escapes from parent" -- must work normally.
	weirdDir := "path escapes from parent"
	require.NoError(t, os.MkdirAll(tmpDir+"/"+weirdDir, 0o755))
	require.NoError(t, os.WriteFile(tmpDir+"/"+weirdDir+"/file.txt", []byte("ok"), 0o644))
	f, errno = testFS.OpenFile(weirdDir+"/file.txt", experimentalsys.O_RDONLY, 0)
	require.Zero(t, errno, "should open file inside directory named 'path escapes from parent'")
	f.Close()
}

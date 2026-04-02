package security

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileReadableUnderRoots_regularFileUnderDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	f := filepath.Join(root, "a.log")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !FileReadableUnderRoots([]string{root}, f) {
		t.Fatal("expected allowed")
	}
}

func TestFileReadableUnderRoots_rejectsOutsidePath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := t.TempDir()
	f := filepath.Join(outside, "secret.log")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if FileReadableUnderRoots([]string{root}, f) {
		t.Fatal("expected denied")
	}
}

func TestFileReadableUnderRoots_symlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.log")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "via.link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if FileReadableUnderRoots([]string{root}, link) {
		t.Fatal("symlink outside root must be denied")
	}
}

func TestFileReadableUnderRoots_singleFileRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	f := filepath.Join(root, "only.log")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !FileReadableUnderRoots([]string{f}, f) {
		t.Fatal("expected same file allowed")
	}
	other := filepath.Join(root, "other.log")
	if err := os.WriteFile(other, []byte("y"), 0o600); err != nil {
		t.Fatal(err)
	}
	if FileReadableUnderRoots([]string{f}, other) {
		t.Fatal("expected sibling denied when root is single file")
	}
}

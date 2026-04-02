package source

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileLogSource_ReadRejectsSymlinkOutsideRoot(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink layout differs on windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.log")
	if err := os.WriteFile(target, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape.log")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	src, err := NewFileLogSource("sandbox", root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = src.Read(context.Background(), Query{StreamID: link})
	if err == nil {
		t.Fatal("expected error for symlink target outside root")
	}
}

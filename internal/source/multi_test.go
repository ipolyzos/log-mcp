package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMultiLogSource_routesStreams(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "f.log")
	if err := os.WriteFile(logPath, []byte("file-line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileSrc, err := NewFileLogSource("files", dir)
	if err != nil {
		t.Fatal(err)
	}
	runner := &mockKubectlRunner{
		PodsByNS: map[string]string{"default": "p1\n"},
		Logs:     map[string]string{"default/p1": "pod-line\n"},
	}
	k8sSrc, err := NewK8sLogSource("k8s", []string{"default"}, runner)
	if err != nil {
		t.Fatal(err)
	}
	multi, err := NewMultiLogSource("both", fileSrc, k8sSrc)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	streams, err := multi.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 2 {
		t.Fatalf("List: %d streams", len(streams))
	}

	ch1, err := multi.Read(ctx, Query{StreamID: logPath})
	if err != nil {
		t.Fatal(err)
	}
	var fileLines int
	for range ch1 {
		fileLines++
	}
	if fileLines != 1 {
		t.Fatalf("file read: %d", fileLines)
	}

	ch2, err := multi.Read(ctx, Query{StreamID: k8sStreamPrefix + "default/p1"})
	if err != nil {
		t.Fatal(err)
	}
	var podLines int
	for range ch2 {
		podLines++
	}
	if podLines != 1 {
		t.Fatalf("k8s read: %d", podLines)
	}
}

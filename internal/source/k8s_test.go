package source

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
)

// mockKubectlRunner is a test double with canned get pods / logs behavior.
type mockKubectlRunner struct {
	PodsByNS map[string]string
	Logs     map[string]string
}

func (m *mockKubectlRunner) CombinedOutput(ctx context.Context, args ...string) ([]byte, error) {
	_ = ctx
	if len(args) >= 4 && args[0] == "get" && args[1] == "pods" && args[2] == "-n" {
		ns := args[3]
		if s, ok := m.PodsByNS[ns]; ok {
			return []byte(s), nil
		}
	}
	return nil, fmt.Errorf("mock: unexpected CombinedOutput %v", args)
}

func (m *mockKubectlRunner) StdoutPipe(ctx context.Context, args ...string) (io.ReadCloser, error) {
	_ = ctx
	pod, ns, container, _ := parseMockLogsArgs(args)
	key := mockLogKey(ns, pod, container)
	body, ok := m.Logs[key]
	if !ok {
		return nil, fmt.Errorf("mock: no logs for %q", key)
	}
	return io.NopCloser(strings.NewReader(body)), nil
}

func parseMockLogsArgs(args []string) (pod, ns, container string, tail int64) {
	if len(args) < 1 || args[0] != "logs" {
		return "", "", "", 0
	}
	i := 1
	for i < len(args) {
		switch {
		case args[i] == "-n" && i+1 < len(args):
			ns = args[i+1]
			i += 2
		case args[i] == "-c" && i+1 < len(args):
			container = args[i+1]
			i += 2
		case args[i] == "--tail" && i+1 < len(args):
			tail, _ = strconv.ParseInt(args[i+1], 10, 64)
			i += 2
		case strings.HasPrefix(args[i], "-"):
			i++
		default:
			if pod == "" {
				pod = args[i]
			}
			i++
		}
	}
	return pod, ns, container, tail
}

func mockLogKey(ns, pod, container string) string {
	if container == "" {
		return ns + "/" + pod
	}
	return ns + "/" + pod + "/" + container
}

func TestNewK8sLogSource_namespaceValidation(t *testing.T) {
	t.Parallel()
	_, err := NewK8sLogSource("k", []string{"bad_namespace"}, &mockKubectlRunner{})
	if err == nil {
		t.Fatal("expected error for invalid namespace")
	}
}

func TestK8sLogSource_ListAndRead(t *testing.T) {
	t.Parallel()
	runner := &mockKubectlRunner{
		PodsByNS: map[string]string{
			"default": "api-1\napi-2\n",
		},
		Logs: map[string]string{
			"default/api-1": "one\ntwo\n",
		},
	}
	src, err := NewK8sLogSource("k8s", []string{"default"}, runner)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	streams, err := src.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 2 {
		t.Fatalf("streams: %+v", streams)
	}
	wantID := k8sStreamPrefix + "default/api-1"
	if streams[0].ID != wantID && streams[1].ID != wantID {
		t.Fatalf("missing stream id %q in %+v", wantID, streams)
	}

	ch, err := src.Read(ctx, Query{StreamID: k8sStreamPrefix + "default/api-1"})
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for e := range ch {
		lines = append(lines, e.Raw)
	}
	if len(lines) != 2 || lines[0] != "one" || lines[1] != "two" {
		t.Fatalf("lines: %v", lines)
	}
}

func TestK8sLogSource_ReadMultiContainer(t *testing.T) {
	t.Parallel()
	runner := &mockKubectlRunner{
		PodsByNS: map[string]string{"ns1": "pod-a\n"},
		Logs: map[string]string{
			"ns1/pod-a/sidecar": "side\n",
		},
	}
	src, err := NewK8sLogSource("k8s", []string{"ns1"}, runner)
	if err != nil {
		t.Fatal(err)
	}
	ch, err := src.Read(context.Background(), Query{StreamID: k8sStreamPrefix + "ns1/pod-a/sidecar"})
	if err != nil {
		t.Fatal(err)
	}
	var got string
	for e := range ch {
		got += e.Raw
	}
	if got != "side" {
		t.Fatalf("got %q", got)
	}
}

func TestK8sLogSource_ReadRejectNamespace(t *testing.T) {
	t.Parallel()
	runner := &mockKubectlRunner{PodsByNS: map[string]string{"default": "a\n"}}
	src, err := NewK8sLogSource("k8s", []string{"default"}, runner)
	if err != nil {
		t.Fatal(err)
	}
	_, err = src.Read(context.Background(), Query{StreamID: k8sStreamPrefix + "kube-system/api"})
	if err == nil {
		t.Fatal("expected error for non-allowlisted namespace")
	}
}

func TestParseK8sStreamID(t *testing.T) {
	t.Parallel()
	ns, pod, c, err := parseK8sStreamID(k8sStreamPrefix + "prod/my-pod")
	if err != nil || ns != "prod" || pod != "my-pod" || c != "" {
		t.Fatalf("got %q %q %q %v", ns, pod, c, err)
	}
	ns, pod, c, err = parseK8sStreamID(k8sStreamPrefix + "prod/my-pod/c1")
	if err != nil || c != "c1" {
		t.Fatalf("got %q %q %q %v", ns, pod, c, err)
	}
	_, _, _, err = parseK8sStreamID("/tmp/file.log")
	if err == nil {
		t.Fatal("expected error")
	}
}

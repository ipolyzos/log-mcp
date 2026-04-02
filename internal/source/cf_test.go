package source

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestCfAppsPresentInOutput(t *testing.T) {
	t.Parallel()
	sample := `Getting apps in org x / space y as z...

name       requested state   instances   memory   disk   urls
my-app     started           1/1         256M     1G     my.example.com
other      stopped           0/1         256M     1G
`
	p := cfAppsPresentInOutput(sample)
	if !p["my-app"] || !p["other"] {
		t.Fatalf("present: %#v", p)
	}
	if p["name"] || p["requested"] {
		t.Fatalf("header mistaken for app: %#v", p)
	}
}

func TestNewCFLogSource_validation(t *testing.T) {
	t.Parallel()
	r := &mockCFRunner{}
	_, err := NewCFLogSource("cf", []string{"Bad"}, r)
	if err == nil {
		t.Fatal("expected error for invalid app name")
	}
}

type mockCFRunner struct {
	AppsOutput string
	LogsByApp  map[string]string
}

func (m *mockCFRunner) CombinedOutput(ctx context.Context, args ...string) ([]byte, error) {
	_ = ctx
	if len(args) == 1 && args[0] == "apps" {
		return []byte(m.AppsOutput), nil
	}
	if len(args) >= 3 && args[0] == "logs" && args[2] == "--recent" {
		app := args[1]
		if s, ok := m.LogsByApp[app]; ok {
			return []byte(s), nil
		}
		return nil, fmt.Errorf("mock: no logs for %q", app)
	}
	return nil, fmt.Errorf("mock: unexpected cf %v", args)
}

func TestCFLogSource_ListAndRead(t *testing.T) {
	t.Parallel()
	runner := &mockCFRunner{
		AppsOutput: "name   requested state\nmy-app started\n",
		LogsByApp: map[string]string{
			"my-app": "log line one\nlog line two\n",
		},
	}
	src, err := NewCFLogSource("cf", []string{"my-app"}, runner)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	streams, err := src.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].ID != cfStreamPrefix+"my-app" {
		t.Fatalf("List: %#v", streams)
	}
	ch, err := src.Read(ctx, Query{StreamID: cfStreamPrefix + "my-app"})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	for range ch {
		n++
	}
	if n != 2 {
		t.Fatalf("read lines: %d", n)
	}
}

type mockDockerRunner struct {
	NamesList string
	Logs      map[string]string
}

func (m *mockDockerRunner) CombinedOutput(ctx context.Context, args ...string) ([]byte, error) {
	_ = ctx
	if len(args) >= 4 && args[0] == "ps" && args[1] == "-a" {
		return []byte(m.NamesList), nil
	}
	return nil, fmt.Errorf("mock docker CombinedOutput: %v", args)
}

func (m *mockDockerRunner) StdoutPipe(ctx context.Context, args ...string) (io.ReadCloser, error) {
	_ = ctx
	if len(args) < 2 || args[0] != "logs" {
		return nil, fmt.Errorf("mock docker StdoutPipe: %v", args)
	}
	name := args[1]
	body, ok := m.Logs[name]
	if !ok {
		return nil, fmt.Errorf("mock: no docker logs for %q", name)
	}
	return io.NopCloser(strings.NewReader(body)), nil
}

func TestDockerLogSource_ListAndRead(t *testing.T) {
	t.Parallel()
	runner := &mockDockerRunner{
		NamesList: "c1\nc2\n",
		Logs: map[string]string{
			"c1": "docker-a\n",
		},
	}
	src, err := NewDockerLogSource("d", []string{"c1"}, runner)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	streams, err := src.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].ID != dockerStreamPrefix+"c1" {
		t.Fatalf("List: %#v", streams)
	}
	ch, err := src.Read(ctx, Query{StreamID: dockerStreamPrefix + "c1"})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	for range ch {
		n++
	}
	if n != 1 {
		t.Fatalf("read lines: %d", n)
	}
}

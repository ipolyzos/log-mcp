package tools

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegistry_RegisterNilIgnored(t *testing.T) {
	r := NewRegistry()
	r.Register(nil)
	if r.InstallerCount() != 0 {
		t.Fatalf("expected 0 installers, got %d", r.InstallerCount())
	}
}

func TestRegistry_ApplyOrder(t *testing.T) {
	var order []int
	r := NewRegistry()
	r.Register(func(*mcp.Server) { order = append(order, 1) })
	r.Register(func(*mcp.Server) { order = append(order, 2) })
	s := mcp.NewServer(&mcp.Implementation{Name: "t", Version: "0"}, nil)
	r.Apply(s)
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("apply order: %v", order)
	}
}

// Package main is the entrypoint for the log-mcp MCP server (stdio).
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/ipolyzos/log-mcp/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

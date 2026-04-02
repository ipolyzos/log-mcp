package logcorr

import (
	"context"

	"github.com/ipolyzos/log-mcp/internal/normalize"
	"github.com/ipolyzos/log-mcp/internal/pipeline"
	"github.com/ipolyzos/log-mcp/internal/source"
)

// streamResults runs the redaction pipeline and returns the result channel.
func streamResults(ctx context.Context, src source.LogSource, q source.Query) (<-chan pipeline.Result, error) {
	pl := &pipeline.Pipeline{
		Source: src,
		Query:  q,
		Config: normalize.DefaultConfig(),
	}
	return pl.Run(ctx)
}

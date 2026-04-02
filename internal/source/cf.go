package source

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// CF stream IDs use the form cf://app_name.
const cfStreamPrefix = "cf://"

// cfAppName matches Cloud Foundry app names (lowercase DNS-like segment).
var cfAppName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

// CFLogSource exposes recent app logs for allowlisted apps via cf logs --recent.
type CFLogSource struct {
	name      string
	allow     map[string]struct{}
	allowList []string
	runner    CFRunner
}

// NewCFLogSource returns a source for the given allowlisted app names.
func NewCFLogSource(name string, apps []string, runner CFRunner) (*CFLogSource, error) {
	if name == "" {
		return nil, fmt.Errorf("source: cf source name is empty")
	}
	if runner == nil {
		return nil, fmt.Errorf("source: cf runner is nil")
	}
	seen := make(map[string]struct{})
	var list []string
	for _, a := range apps {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if len(a) > 63 || !cfAppName.MatchString(a) || strings.HasSuffix(a, "-") {
			return nil, fmt.Errorf("source: invalid cf app name %q", a)
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		list = append(list, a)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("source: cf needs at least one allowlisted app")
	}
	sort.Strings(list)
	return &CFLogSource{
		name:      name,
		allow:     seen,
		allowList: list,
		runner:    runner,
	}, nil
}

// Name implements [LogSource].
func (c *CFLogSource) Name() string {
	return c.name
}

// List implements [LogSource]: allowlisted apps that appear in cf apps.
func (c *CFLogSource) List(ctx context.Context) ([]LogStream, error) {
	b, err := c.runner.CombinedOutput(ctx, "apps")
	if err != nil {
		return nil, fmt.Errorf("source: cf apps: %w", err)
	}
	present := cfAppsPresentInOutput(string(b))
	var out []LogStream
	for _, name := range c.allowList {
		if !present[name] {
			continue
		}
		out = append(out, LogStream{
			ID:   cfStreamPrefix + name,
			Name: name,
		})
	}
	return out, nil
}

// Read implements [LogSource] for stream IDs cf://app_name.
func (c *CFLogSource) Read(ctx context.Context, q Query) (<-chan LogEntry, error) {
	app, err := parseCFStreamID(q.StreamID)
	if err != nil {
		return nil, err
	}
	if _, ok := c.allow[app]; !ok {
		return nil, fmt.Errorf("source: app %q is not allowlisted for cf source", app)
	}

	b, err := c.runner.CombinedOutput(ctx, "logs", app, "--recent")
	if err != nil {
		return nil, fmt.Errorf("source: cf logs: %w", err)
	}

	rc := io.NopCloser(strings.NewReader(string(b)))
	ch := make(chan LogEntry, 64)
	go streamReaderLines(ctx, rc, q, ch)
	return ch, nil
}

func parseCFStreamID(id string) (app string, err error) {
	if !strings.HasPrefix(id, cfStreamPrefix) {
		return "", fmt.Errorf("source: not a cf stream id")
	}
	rest := strings.TrimPrefix(id, cfStreamPrefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", fmt.Errorf("source: empty cf stream path")
	}
	if strings.Contains(rest, "/") {
		return "", fmt.Errorf("source: cf stream id must be cf://app_name")
	}
	if len(rest) > 63 || !cfAppName.MatchString(rest) || strings.HasSuffix(rest, "-") {
		return "", fmt.Errorf("source: invalid cf app name in stream id")
	}
	return rest, nil
}

// cfAppsPresentInOutput returns app names that appear in `cf apps` table rows (validated names only).
func cfAppsPresentInOutput(out string) map[string]bool {
	lines := strings.Split(out, "\n")
	inTable := false
	present := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Getting apps") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if !inTable {
			if strings.EqualFold(fields[0], "name") {
				inTable = true
			}
			continue
		}
		name := fields[0]
		if strings.EqualFold(name, "name") {
			continue
		}
		if len(name) > 63 || !cfAppName.MatchString(name) || strings.HasSuffix(name, "-") {
			continue
		}
		present[name] = true
	}
	return present
}

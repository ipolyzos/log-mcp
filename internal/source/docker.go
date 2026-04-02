package source

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Docker stream IDs use the form docker://container_name.
const dockerStreamPrefix = "docker://"

// dockerContainerName matches a conservative subset of Docker container names (no slashes).
var dockerContainerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,200}$`)

// DockerLogSource exposes container logs for allowlisted names via docker logs.
type DockerLogSource struct {
	name      string
	allow     map[string]struct{}
	allowList []string
	runner    DockerRunner
}

// NewDockerLogSource returns a source for the given allowlisted container names.
func NewDockerLogSource(name string, containers []string, runner DockerRunner) (*DockerLogSource, error) {
	if name == "" {
		return nil, fmt.Errorf("source: docker source name is empty")
	}
	if runner == nil {
		return nil, fmt.Errorf("source: docker runner is nil")
	}
	seen := make(map[string]struct{})
	var list []string
	for _, c := range containers {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !dockerContainerName.MatchString(c) {
			return nil, fmt.Errorf("source: invalid docker container name %q", c)
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		list = append(list, c)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("source: docker needs at least one allowlisted container")
	}
	sort.Strings(list)
	return &DockerLogSource{
		name:      name,
		allow:     seen,
		allowList: list,
		runner:    runner,
	}, nil
}

// Name implements [LogSource].
func (d *DockerLogSource) Name() string {
	return d.name
}

// List implements [LogSource]: streams for allowlisted containers that appear in docker ps -a.
func (d *DockerLogSource) List(ctx context.Context) ([]LogStream, error) {
	b, err := d.runner.CombinedOutput(ctx, "ps", "-a", "--format", "{{.Names}}")
	if err != nil {
		return nil, fmt.Errorf("source: docker ps: %w", err)
	}
	present := make(map[string]struct{})
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !dockerContainerName.MatchString(line) {
			continue
		}
		present[line] = struct{}{}
	}
	var out []LogStream
	for _, name := range d.allowList {
		if _, ok := present[name]; !ok {
			continue
		}
		out = append(out, LogStream{
			ID:   dockerStreamPrefix + name,
			Name: name,
		})
	}
	return out, nil
}

// Read implements [LogSource] for stream IDs docker://container_name.
func (d *DockerLogSource) Read(ctx context.Context, q Query) (<-chan LogEntry, error) {
	name, err := parseDockerStreamID(q.StreamID)
	if err != nil {
		return nil, err
	}
	if _, ok := d.allow[name]; !ok {
		return nil, fmt.Errorf("source: container %q is not allowlisted for docker source", name)
	}

	args := []string{"logs", name}
	if q.MaxLines > 0 {
		args = append(args, "--tail", strconv.FormatInt(q.MaxLines, 10))
	}

	rc, err := d.runner.StdoutPipe(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("source: docker logs: %w", err)
	}

	ch := make(chan LogEntry, 64)
	go streamReaderLines(ctx, rc, q, ch)
	return ch, nil
}

func parseDockerStreamID(id string) (container string, err error) {
	if !strings.HasPrefix(id, dockerStreamPrefix) {
		return "", fmt.Errorf("source: not a docker stream id")
	}
	rest := strings.TrimPrefix(id, dockerStreamPrefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", fmt.Errorf("source: empty docker stream path")
	}
	if strings.Contains(rest, "/") {
		return "", fmt.Errorf("source: docker stream id must be docker://container_name")
	}
	if !dockerContainerName.MatchString(rest) {
		return "", fmt.Errorf("source: invalid docker container name in stream id")
	}
	return rest, nil
}

package source

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// K8s stream IDs use the form k8s://namespace/pod or k8s://namespace/pod/container.
const k8sStreamPrefix = "k8s://"

// podListJSONPath lists pod names one per line (kubectl jsonpath).
const podListJSONPath = `{range .items[*]}{.metadata.name}{"\n"}{end}`

// dns1123Label matches a single Kubernetes DNS label (namespace, pod, container names).
var dns1123Label = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// K8sLogSource exposes pod logs in allowlisted namespaces via kubectl logs.
// Only "kubectl get pods" and "kubectl logs" are invoked, with fixed argument shapes.
type K8sLogSource struct {
	name    string
	nsList  []string
	nsAllow map[string]struct{}
	runner  KubectlRunner
}

// NewK8sLogSource returns a source for the given allowlisted namespaces.
// Each namespace must be a valid DNS label (RFC 1123).
func NewK8sLogSource(name string, namespaces []string, runner KubectlRunner) (*K8sLogSource, error) {
	if name == "" {
		return nil, fmt.Errorf("source: k8s source name is empty")
	}
	if runner == nil {
		return nil, fmt.Errorf("source: k8s kubectl runner is nil")
	}
	if len(namespaces) == 0 {
		return nil, fmt.Errorf("source: k8s needs at least one allowlisted namespace")
	}
	seen := make(map[string]struct{})
	var list []string
	for _, ns := range namespaces {
		ns = strings.TrimSpace(ns)
		if ns == "" {
			continue
		}
		if !dns1123Label.MatchString(ns) || len(ns) > 63 {
			return nil, fmt.Errorf("source: invalid namespace %q", ns)
		}
		if _, ok := seen[ns]; ok {
			continue
		}
		seen[ns] = struct{}{}
		list = append(list, ns)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("source: k8s no valid namespaces after filtering")
	}
	sort.Strings(list)
	return &K8sLogSource{
		name:    name,
		nsList:  list,
		nsAllow: seen,
		runner:  runner,
	}, nil
}

// Name implements [LogSource].
func (k *K8sLogSource) Name() string {
	return k.name
}

// List implements [LogSource]: one stream per pod in allowlisted namespaces (default container).
func (k *K8sLogSource) List(ctx context.Context) ([]LogStream, error) {
	var out []LogStream
	for _, ns := range k.nsList {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		args := []string{
			"get", "pods", "-n", ns,
			"-o", "jsonpath=" + podListJSONPath,
		}
		b, err := k.runner.CombinedOutput(ctx, args...)
		if err != nil {
			return nil, fmt.Errorf("source: kubectl get pods -n %s: %w", ns, err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if !dns1123Label.MatchString(line) || len(line) > 63 {
				continue
			}
			id := k8sStreamPrefix + ns + "/" + line
			out = append(out, LogStream{
				ID:   id,
				Name: ns + "/" + line,
			})
		}
	}
	return out, nil
}

// Read implements [LogSource] for stream IDs k8s://namespace/pod[/container].
func (k *K8sLogSource) Read(ctx context.Context, q Query) (<-chan LogEntry, error) {
	if q.StreamID == "" {
		return nil, fmt.Errorf("source: empty stream id")
	}
	ns, pod, container, err := parseK8sStreamID(q.StreamID)
	if err != nil {
		return nil, err
	}
	if _, ok := k.nsAllow[ns]; !ok {
		return nil, fmt.Errorf("source: namespace %q is not allowlisted for k8s source", ns)
	}
	if !dns1123Label.MatchString(pod) || len(pod) > 63 {
		return nil, fmt.Errorf("source: invalid pod name")
	}
	if container != "" && (!dns1123Label.MatchString(container) || len(container) > 63) {
		return nil, fmt.Errorf("source: invalid container name")
	}

	args := []string{"logs", pod, "-n", ns}
	if container != "" {
		args = append(args, "-c", container)
	}
	if q.MaxLines > 0 {
		args = append(args, "--tail", strconv.FormatInt(q.MaxLines, 10))
	}

	rc, err := k.runner.StdoutPipe(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("source: kubectl logs: %w", err)
	}

	ch := make(chan LogEntry, 64)
	go streamReaderLines(ctx, rc, q, ch)
	return ch, nil
}

func parseK8sStreamID(id string) (namespace, pod, container string, err error) {
	if !strings.HasPrefix(id, k8sStreamPrefix) {
		return "", "", "", fmt.Errorf("source: not a k8s stream id")
	}
	rest := strings.TrimPrefix(id, k8sStreamPrefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", "", "", fmt.Errorf("source: empty k8s stream path")
	}
	parts := strings.Split(rest, "/")
	switch len(parts) {
	case 2:
		return parts[0], parts[1], "", nil
	case 3:
		return parts[0], parts[1], parts[2], nil
	default:
		return "", "", "", fmt.Errorf("source: k8s stream id must be k8s://namespace/pod or k8s://namespace/pod/container")
	}
}

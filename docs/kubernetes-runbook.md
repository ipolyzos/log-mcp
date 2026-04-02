# Kubernetes logs with log-mcp

Operational notes for reading **pod logs** through **log-mcp** when
**`LOG_MCP_K8S_NAMESPACES`** is configured. For general architecture and
security limits, see [architecture.md](architecture.md).

## Prerequisites

- **`kubectl`** available on the host running **log-mcp** (override with
  **`LOG_MCP_K8S_KUBECTL`**).
- **`LOG_MCP_K8S_NAMESPACES`** set to a comma-separated **allowlist** of
  namespace names (DNS label form). Only those namespaces are queried; there is
  no cluster-wide listing beyond **`kubectl get pods -n <ns>`** per allowlist
  entry.

## Stream IDs

Log streams are addressed as:

- **`k8s://namespace/pod`**
- **`k8s://namespace/pod/container`**

Use these values as **`stream_id`** for **`read_logs`**, **`filter_logs`**,
**`summarize_errors`**, **`detect_anomalies`**, and other log tools.

## CRI-shaped lines

Plain-text logs from the **container runtime interface (CRI)** often look like:

```text
2024-06-15T10:30:02Z stderr F connection refused to redis:6379
2024-06-15T10:30:03Z stdout F GET /bad HTTP/1.1 503
```

**log-mcp**’s text parser recognizes **`stdout`** / **`stderr`**, infers **ERROR**
on stderr for common failure needles, and on stdout for **HTTP 5xx** patterns
(including structured **`status=`** fields). That improves **`summarize_errors`**
and **`detect_anomalies`** without JSON log drivers. See
[text_parser.go](../internal/parse/text_parser.go) and [tools.md](tools.md).

## Suggested checks

1. Confirm **`kubectl get pods -n <namespace>`** works with the same kubeconfig
   as the **log-mcp** process.
2. Call **`read_logs`** with a small **`max_lines`** on a known pod stream ID.
3. Use **`filter_logs`** with **`level`** and optional **`keywords_any`** for
   OR substrings on redacted lines.
4. Run **`detect_anomalies`** with default **`segment_count`**; if
   **`low_sample`** is **true**, treat **`level_spike_segments`** as intentionally
   empty for short streams.

## Troubleshooting

- **Tool error “log access not configured”** — no file or K8s (or other) source
  is active; set **`LOG_MCP_K8S_NAMESPACES`** or **`LOG_MCP_ROOT`**.
- **`stream_id` rejected** — IDs must match the **`k8s://`** pattern and an
  existing pod (and container if specified) in an allowlisted namespace.
- See [troubleshooting.md](troubleshooting.md) for kubectl/docker/stdio issues.

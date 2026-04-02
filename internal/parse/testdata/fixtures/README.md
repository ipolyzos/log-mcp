# System-style log line fixtures

Short **synthetic** samples shaped like real platforms. They exercise
[`AutoParser`](../../text_parser.go) (JSON when the line starts with `{`, else
[`TextParser`](../../text_parser.go)). Lines are **not** copied from production
systems; formats follow public documentation.

| File | Format reference |
|------|------------------|
| `01_kubernetes_cri.log` | [Kubernetes: logging architecture](https://kubernetes.io/docs/concepts/cluster-administration/logging/) (CRI stdout/stderr line prefix) |
| `02_docker_json_file.log` | [Docker: JSON file logging driver](https://docs.docker.com/engine/logging/drivers/json-file/) |
| `03_nginx_access.log` | [nginx: ngx_http_log_module](https://nginx.org/en/docs/http/ngx_http_log_module.html) `combined` |
| `04_apache_access.log` | [Apache: Log files](https://httpd.apache.org/docs/2.4/logs.html) `combined` |

More files may be added in later commits; extend [`fixtures_test.go`](../../fixtures_test.go).

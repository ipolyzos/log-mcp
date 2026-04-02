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
| `05_postgresql.log` | [PostgreSQL logging](https://www.postgresql.org/docs/current/runtime-config-logging.html) |
| `06_mysql.log` | [MySQL error log](https://dev.mysql.com/doc/refman/8.0/en/error-log.html) |
| `07_mongodb.log` | [MongoDB log messages](https://www.mongodb.com/docs/manual/reference/log-messages/) |
| `08_redis.log` | [Redis logging](https://redis.io/docs/latest/operate/rs-7-2-4/logging/) |
| `09_nomad.log` | [Nomad `log_file` / agent logs](https://developer.hashicorp.com/nomad/docs/configuration#log_file) |
| `10_cloudfoundry.log` | [Cloud Foundry gorouter](https://docs.cloudfoundry.org/gorouter/) (access-style line) |

More files may be added in later commits; extend [`fixtures_test.go`](../../fixtures_test.go).

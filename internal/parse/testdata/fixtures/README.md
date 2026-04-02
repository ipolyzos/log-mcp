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
| `11_aws_lambda.log` | [AWS Lambda logging](https://docs.aws.amazon.com/lambda/latest/dg/nodejs-logging.html) |
| `12_aws_cloudfront.log` | [CloudFront access logs](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/AccessLogs.html) |
| `13_gcp_load_balancer.log` | [GCP HTTPS LB logging](https://cloud.google.com/load-balancing/docs/https/https-logging-monitoring) |
| `14_azure_function.log` | [Azure Functions monitoring](https://learn.microsoft.com/en-us/azure/azure-functions/functions-monitoring) |
| `15_envoy_proxy.log` | [Envoy application logging](https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/application_logging) |
| `16_linkerd_proxy.log` | [Linkerd proxy log level](https://linkerd.io/2.14/reference/proxy-log-level/) |
| `17_kong_proxy.log` | [Kong Gateway logging](https://docs.konghq.com/gateway/latest/production/logging/) |
| `18_haproxy_http.log` | [HAProxy logging](https://docs.haproxy.org/3.0/configuration.html#8.2.4) |
| `19_kafka.log` | [Kafka broker logging](https://kafka.apache.org/documentation/#brokerconfigs_log.dirs) |
| `20_rabbitmq.log` | [RabbitMQ logging](https://www.rabbitmq.com/docs/logging) |
| `21_elasticsearch.log` | [Elastic ECS logging](https://www.elastic.co/guide/en/ecs/current/ecs-log.html) |
| `22_fluentd.log` | [Fluentd tail input](https://docs.fluentd.org/input/tail) (wrapped line) |
| `23_logstash.log` | [Logstash logging](https://www.elastic.co/guide/en/logstash/current/logging.html) |
| `24_prometheus.log` | [Prometheus configuration](https://prometheus.io/docs/prometheus/latest/configuration/configuration/) |
| `25_grafana.log` | [Grafana configuration / logs](https://grafana.com/docs/grafana/latest/setup-grafana/configure-grafana/#log) |
| `26_jaeger_span.log` | [Jaeger APIs](https://www.jaegertracing.io/docs/1.57/apis/#json-over-http) (trace JSON) |
| `27_cassandra.log` | [Cassandra logging](https://cassandra.apache.org/doc/latest/cassandra/managing/operating/logging.html) |
| `28_dynamodb_app.log` | [DynamoDB errors (app-level)](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Programming.Errors.html) |
| `29_s3_server_access.log` | [S3 server access log format](https://docs.aws.amazon.com/AmazonS3/latest/userguide/LogFormat.html) |
| `30_rabbitmq_json.log` | [RabbitMQ JSON logging](https://www.rabbitmq.com/docs/logging#json) |
| `31_gcs_audit.log` | [GCS audit logging](https://cloud.google.com/storage/docs/audit-logging) |
| `32_azure_blob.log` | [Monitor Azure Blob Storage](https://learn.microsoft.com/en-us/azure/storage/blobs/monitor-blob-storage) |
| `33_cortex_ingester.log` | [Grafana Mimir production checklist](https://grafana.com/docs/mimir/latest/operators-guide/run-production-environment-with-helm/production-readiness-checklist/) (Cortex/Mimir family) |
| `34_gcp_cloud_functions.log` | [Cloud Functions logging](https://cloud.google.com/functions/docs/configuring/logging) |

Extend [`fixtures_test.go`](../../fixtures_test.go) when adding files.

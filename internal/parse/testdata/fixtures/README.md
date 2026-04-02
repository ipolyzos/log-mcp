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
| `06_mysql.log` | [MariaDB KB: error log](https://mariadb.com/kb/en/error-log/) (MySQL-compatible mysqld style) |
| `07_mongodb.log` | [MongoDB log messages](https://www.mongodb.com/docs/manual/reference/log-messages/) |
| `08_redis.log` | [Redis documentation](https://redis.io/docs/) (server stdout style) |
| `09_nomad.log` | [Nomad `log_file` / agent logs](https://developer.hashicorp.com/nomad/docs/configuration#log_file) |
| `10_cloudfoundry.log` | [Cloud Foundry HTTP routing](https://docs.cloudfoundry.org/concepts/http-routing.html) (gorouter-style access line) |
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
| `33_cortex_ingester.log` | [Grafana Mimir: run in production](https://grafana.com/docs/mimir/latest/manage/run-production-environment/) (Cortex/Mimir family) |
| `34_gcp_cloud_functions.log` | [Cloud Run logging](https://cloud.google.com/run/docs/logging) (GCP managed runtime JSON style) |
| `35_java_log4j2_json.log` | [Log4j 2 JSON layout](https://logging.apache.org/log4j/2.x/manual/layouts.html#JSONLayout) |
| `36_java_logback_logstash_encoder.log` | [logstash-logback-encoder](https://github.com/logfellow/logstash-logback-encoder) |
| `37_java_slf4j_jackson_app.log` | [SLF4J user manual](https://www.slf4j.org/manual.html) (app JSON via logging facade) |
| `38_java_micrometer_observation.log` | [Micrometer observations](https://micrometer.io/docs/observation) |
| `39_java_opentelemetry_sdk.log` | [OpenTelemetry Java](https://opentelemetry.io/docs/languages/java/) |
| `40_python_json_logger.log` | [python-json-logger](https://pypi.org/project/python-json-logger/) |
| `41_python_structlog.log` | [structlog](https://www.structlog.org/en/stable/) |
| `42_javascript_pino.log` | [Pino](https://getpino.io/#/docs/api) |
| `43_javascript_winston.log` | [Winston](https://github.com/winstonjs/winston#logging) |
| `44_javascript_opentelemetry_sdk.log` | [OpenTelemetry JS](https://opentelemetry.io/docs/languages/js/) |
| `45_rust_tracing_subscriber.log` | [tracing-subscriber fmt](https://docs.rs/tracing-subscriber/latest/tracing_subscriber/fmt/index.html) |
| `46_rust_log_env_logger.log` | [env_logger](https://docs.rs/env_logger/latest/env_logger/) |
| `47_cpp_spdlog.log` | [spdlog](https://github.com/gabime/spdlog) |
| `48_c_log4c.log` | [log4c](https://log4c.sourceforge.net/) |
| `49_cpp_qlog_event.log` | [qlog (QUIC)](https://datatracker.ietf.org/doc/html/draft-ietf-quic-qlog-h3-events) (single-line event) |
| `50_cpp_opentelemetry_sdk.log` | [OpenTelemetry C++](https://opentelemetry.io/docs/languages/cpp/) |
| `51_csharp_microsoft_extensions_logging.log` | [.NET console log formatter](https://learn.microsoft.com/en-us/dotnet/core/extensions/console-log-formatter) |
| `52_csharp_serilog_compact_json.log` | [Serilog formatting](https://github.com/serilog/serilog/wiki/Formatting-Output) |
| `53_csharp_serilog_json_formatter.log` | [Serilog.Formatting.Compact](https://github.com/serilog/serilog-formatting-compact) |
| `54_golden_stack_polyglot.log` | [OpenTelemetry logs](https://opentelemetry.io/docs/concepts/signals/logs/) (shared JSON schema for Fluent Bit → ES/Loki + Jaeger) |

Extend [`fixtures_test.go`](../../fixtures_test.go) when adding files.

package parse

import (
	"bufio"
	"bytes"
	"embed"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"testing"
)

//go:embed testdata/fixtures/*.log
var fixtureLogs embed.FS

func TestFixtureLogLines_SmokeAutoParser(t *testing.T) {
	t.Parallel()
	a := AutoParser{}
	entries, err := fs.ReadDir(fixtureLogs, "testdata/fixtures")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := fixtureLogs.ReadFile(path.Join("testdata/fixtures", name))
			if err != nil {
				t.Fatal(err)
			}
			sc := bufio.NewScanner(bytes.NewReader(data))
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				got, err := a.Parse(line)
				if err != nil {
					t.Fatalf("parse %q: %v", line, err)
				}
				if !parsedLineNonTrivial(got) {
					t.Fatalf("trivial parse for %q: %+v", line, got)
				}
			}
			if err := sc.Err(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func parsedLineNonTrivial(p ParsedLine) bool {
	if strings.TrimSpace(p.Message) != "" {
		return true
	}
	if p.Level != "" {
		return true
	}
	if p.Timestamp != nil {
		return true
	}
	return len(p.Fields) > 0
}

func TestFixtureLogLines_ExpectedSubstrings(t *testing.T) {
	t.Parallel()
	a := AutoParser{}
	cases := []struct {
		file    string
		lineIdx int
		substr  string
	}{
		{"01_kubernetes_cri.log", 0, "healthz"},
		{"01_kubernetes_cri.log", 1, "refused"},
		{"01_kubernetes_cri.log", 2, "503"},
		{"02_docker_json_file.log", 0, "8080"},
		{"02_docker_json_file.log", 1, "ERROR"},
		{"03_nginx_access.log", 0, "/api/v1/ready"},
		{"03_nginx_access.log", 1, "499"},
		{"04_apache_access.log", 0, "index.html"},
		{"04_apache_access.log", 1, "404"},
		{"05_postgresql.log", 0, "duration"},
		{"05_postgresql.log", 1, "duplicate"},
		{"06_mysql.log", 0, "Aborted"},
		{"06_mysql.log", 1, "InnoDB"},
		{"07_mongodb.log", 0, "Slow query"},
		{"07_mongodb.log", 1, "WiredTiger"},
		{"08_redis.log", 0, "Ready"},
		{"08_redis.log", 1, "overcommit"},
		{"09_nomad.log", 0, "allocations"},
		{"09_nomad.log", 1, "vault"},
		{"10_cloudfoundry.log", 0, "shop.example.com"},
		{"11_aws_lambda.log", 0, "START"},
		{"11_aws_lambda.log", 1, "Duration"},
		{"12_aws_cloudfront.log", 0, "cloudfront"},
		{"13_gcp_load_balancer.log", 0, "api.example"},
		{"14_azure_function.log", 0, "HttpTrigger"},
		{"15_envoy_proxy.log", 0, "initializing"},
		{"15_envoy_proxy.log", 1, "listener"},
		{"16_linkerd_proxy.log", 0, "linkerd2_proxy"},
		{"17_kong_proxy.log", 0, "/services"},
		{"18_haproxy_http.log", 0, "haproxy"},
		{"19_kafka.log", 0, "AddPartitions"},
		{"19_kafka.log", 1, "ReplicaFetcher"},
		{"20_rabbitmq.log", 0, "AMQP"},
		{"20_rabbitmq.log", 1, "precondition"},
		{"21_elasticsearch.log", 0, "watermark"},
		{"22_fluentd.log", 0, "payment"},
		{"23_logstash.log", 0, "index"},
		{"24_prometheus.log", 0, "Discovery"},
		{"24_prometheus.log", 1, "deadline"},
		{"25_grafana.log", 0, "Listen"},
		{"26_jaeger_span.log", 0, "dispatch"},
		{"27_cassandra.log", 0, "Compacting"},
		{"27_cassandra.log", 1, "tombstone"},
		{"28_dynamodb_app.log", 0, "Sessions"},
		{"29_s3_server_access.log", 0, "REST.GET.OBJECT"},
		{"30_rabbitmq_json.log", 0, "Deprecated"},
		{"31_gcs_audit.log", 0, "storage.objects.get"},
		{"32_azure_blob.log", 0, "blob.core"},
		{"33_cortex_ingester.log", 0, "ingester"},
		{"34_gcp_cloud_functions.log", 0, "helloWorld"},
		{"35_java_log4j2_json.log", 0, "submitting"},
		{"36_java_logback_logstash_encoder.log", 0, "retrying"},
		{"37_java_slf4j_jackson_app.log", 0, "/api/v1/cart"},
		{"38_java_micrometer_observation.log", 0, "http.server.requests"},
		{"39_java_opentelemetry_sdk.log", 0, "payment"},
		{"40_python_json_logger.log", 0, "heartbeat"},
		{"41_python_structlog.log", 0, "merged"},
		{"42_javascript_pino.log", 0, "checkout"},
		{"43_javascript_winston.log", 0, "enqueued"},
		{"44_javascript_opentelemetry_sdk.log", 0, "opentelemetry"},
		{"45_rust_tracing_subscriber.log", 0, "readiness"},
		{"46_rust_log_env_logger.log", 0, "pool"},
		{"47_cpp_spdlog.log", 0, "upstream"},
		{"48_c_log4c.log", 0, "accepted"},
		{"49_cpp_qlog_event.log", 0, "recovery"},
		{"50_cpp_opentelemetry_sdk.log", 0, "OTLP"},
		{"51_csharp_microsoft_extensions_logging.log", 0, "stopping"},
		{"52_csharp_serilog_compact_json.log", 0, "SQL"},
		{"53_csharp_serilog_json_formatter.log", 0, "cache"},
		{"54_golden_stack_polyglot.log", 0, "inventory"},
		{"54_golden_stack_polyglot.log", 4, "validated"},
	}
	for _, tc := range cases {
		t.Run(tc.file+"_"+strconv.Itoa(tc.lineIdx), func(t *testing.T) {
			t.Parallel()
			line := nthFixtureLine(t, tc.file, tc.lineIdx)
			got, err := a.Parse(line)
			if err != nil {
				t.Fatal(err)
			}
			msg := got.Message
			var b strings.Builder
			b.WriteString(msg)
			for _, v := range got.Fields {
				b.WriteString(v)
			}
			combined := b.String()
			if !strings.Contains(combined, tc.substr) {
				t.Fatalf("want substring %q in message+fields, got message=%q fields=%v",
					tc.substr, got.Message, got.Fields)
			}
		})
	}
}

func nthFixtureLine(t *testing.T, file string, idx int) string {
	t.Helper()
	data, err := fixtureLogs.ReadFile(path.Join("testdata/fixtures", file))
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if n == idx {
			return line
		}
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	t.Fatalf("file %q: no line at index %d", file, idx)
	return ""
}

func TestGoldenStackPolyglot_ParseSchema(t *testing.T) {
	t.Parallel()
	a := AutoParser{}
	const goldenFile = "54_golden_stack_polyglot.log"
	data, err := fixtureLogs.ReadFile(path.Join("testdata/fixtures", goldenFile))
	if err != nil {
		t.Fatal(err)
	}
	wantServices := []string{
		"checkout-java",
		"payments-python",
		"api-node",
		"gateway-rust",
		"auth-dotnet",
	}
	const sharedTrace = "4bf92f3577b34da6a3ce929d0e0e4736"
	sc := bufio.NewScanner(bytes.NewReader(data))
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		got, err := a.Parse(line)
		if err != nil {
			t.Fatalf("line %d: %v", n, err)
		}
		if got.Timestamp == nil {
			t.Errorf("line %d: missing timestamp", n)
		}
		if got.Level == "" {
			t.Errorf("line %d: missing level", n)
		}
		if strings.TrimSpace(got.Message) == "" {
			t.Errorf("line %d: missing message", n)
		}
		if n >= len(wantServices) {
			t.Fatalf("extra data line %d", n)
		}
		if got.Fields["service"] != wantServices[n] {
			t.Errorf("line %d: service got %q want %q", n, got.Fields["service"], wantServices[n])
		}
		if got.Fields["trace_id"] != sharedTrace {
			t.Errorf("line %d: trace_id got %q want %q", n, got.Fields["trace_id"], sharedTrace)
		}
		if got.Fields["span_id"] == "" {
			t.Errorf("line %d: missing span_id", n)
		}
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if n != len(wantServices) {
		t.Fatalf("want %d golden lines, got %d", len(wantServices), n)
	}
}

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
			if msg == "" {
				// JSON lines may stash payload only in Fields; stringify for search.
				var b strings.Builder
				b.WriteString(msg)
				for _, v := range got.Fields {
					b.WriteString(v)
				}
				msg = b.String()
			}
			if !strings.Contains(msg, tc.substr) {
				t.Fatalf("want substring %q in synthesized message/fields, got message=%q fields=%v",
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

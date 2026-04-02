package llmtools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateStructuredSummary_rejectsRawLines(t *testing.T) {
	t.Parallel()
	_, err := ValidateStructuredSummary([]byte(`{"stream_id":"/x","lines":[{"line_number":1,"text":"a"}]}`))
	if err == nil {
		t.Fatal("expected error for read_logs-style payload")
	}
}

func TestValidateStructuredSummary_acceptsSummarizeErrors(t *testing.T) {
	t.Parallel()
	b := []byte(`{"stream_id":"/l","groups":[{"rank":1,"count":3,"pattern":"boom"}],"error_line_count":3}`)
	obj, err := ValidateStructuredSummary(b)
	if err != nil {
		t.Fatal(err)
	}
	br, err := BuildIncidentBrief(obj)
	if err != nil {
		t.Fatal(err)
	}
	if len(br.Bullets) < 1 {
		t.Fatalf("bullets: %+v", br)
	}
}

func TestValidateStructuredSummary_acceptsAnomaliesOnly(t *testing.T) {
	t.Parallel()
	b := []byte(`{"level_spike_segments":[{"segment_index":1,"line_start":10,"line_end":20,"error_count":9,"z_score":3}],"repeated_errors":[]}`)
	_, err := ValidateStructuredSummary(b)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateStructuredSummary_rejectsNonObject(t *testing.T) {
	t.Parallel()
	_, err := ValidateStructuredSummary([]byte(`["a"]`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildIncidentBrief_lowSampleBullet(t *testing.T) {
	t.Parallel()
	raw := `{
	  "level_spike_segments": [],
	  "repeated_errors": [],
	  "low_sample": true,
	  "scanned_lines": 4,
	  "truncated": false
	}`
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		t.Fatal(err)
	}
	br, err := BuildIncidentBrief(obj)
	if err != nil {
		t.Fatal(err)
	}
	if len(br.Bullets) < 1 {
		t.Fatalf("bullets: %+v", br.Bullets)
	}
	if !strings.Contains(br.Bullets[0], "low_sample") {
		t.Fatalf("expected low_sample in first bullet: %q", br.Bullets[0])
	}
	if !strings.Contains(br.ForLLMContext, "low_sample") {
		t.Fatalf("context: %q", br.ForLLMContext)
	}
}

func TestBuildIncidentBrief_mergedJSON(t *testing.T) {
	t.Parallel()
	var obj map[string]json.RawMessage
	raw := `{
	  "groups":[{"rank":1,"count":2,"pattern":"p"}],
	  "error_line_count":2,
	  "level_spike_segments":[{"segment_index":0,"line_start":1,"line_end":5,"error_count":4,"z_score":2.5}],
	  "repeated_errors":[{"fingerprint":"ab","sample_message":"e","count":5,"first_line":1,"last_line":5}]
	}`
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		t.Fatal(err)
	}
	br, err := BuildIncidentBrief(obj)
	if err != nil {
		t.Fatal(err)
	}
	if br.SummaryKindHint == "" {
		t.Fatal("expected kind hint")
	}
}

package server

import (
	"encoding/json"
	"testing"
)

func TestRequestRoundTrip(t *testing.T) {
	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"health_check"}}`)
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	if req.JSONRPC != JSONRPCVersion {
		t.Fatalf("jsonrpc: got %q want %q", req.JSONRPC, JSONRPCVersion)
	}
	if string(req.ID) != "1" {
		t.Fatalf("id: got %s", req.ID)
	}
	if req.Method != "tools/call" {
		t.Fatalf("method: got %q", req.Method)
	}
	if len(req.Params) == 0 {
		t.Fatal("expected params")
	}
}

func TestResponseResultRoundTrip(t *testing.T) {
	resp := Response{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage("42"),
		Result:  json.RawMessage(`{"tools":[]}`),
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var got Response
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Error != nil {
		t.Fatalf("unexpected error: %+v", got.Error)
	}
	if string(got.Result) != `{"tools":[]}` {
		t.Fatalf("result: %s", got.Result)
	}
}

func TestResponseErrorRoundTrip(t *testing.T) {
	resp := Response{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`"req-1"`),
		Error:   &ErrorObject{Code: -32601, Message: "Method not found"},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var got Response
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Result != nil {
		t.Fatalf("expected no result, got %s", got.Result)
	}
	if got.Error == nil || got.Error.Code != -32601 {
		t.Fatalf("error: %+v", got.Error)
	}
}

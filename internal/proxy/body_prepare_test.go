package proxy

import (
	"encoding/json"
	"testing"
)

func TestPrepareUpstreamBody(t *testing.T) {
	raw := []byte(`{"model":"logical-group","messages":[{"role":"user","content":"hi"}]}`)
	defaults := map[string]interface{}{
		"temperature": 0.7,
		"max_tokens":  4096,
	}

	out, err := prepareUpstreamBody(raw, "real-upstream-model", defaults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body["model"] != "real-upstream-model" {
		t.Fatalf("expected model replacement, got %v", body["model"])
	}
	if _, ok := body["temperature"]; !ok {
		t.Fatalf("expected default temperature")
	}
	if _, ok := body["max_tokens"]; !ok {
		t.Fatalf("expected default max_tokens")
	}
}

func TestPrepareUpstreamBodyKeepsCallerValue(t *testing.T) {
	raw := []byte(`{"model":"logical-group","temperature":0.1}`)
	defaults := map[string]interface{}{
		"temperature": 0.7,
	}

	out, err := prepareUpstreamBody(raw, "real-upstream-model", defaults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(out, &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body["temperature"].(float64) != 0.1 {
		t.Fatalf("expected caller value to be kept, got %v", body["temperature"])
	}
}

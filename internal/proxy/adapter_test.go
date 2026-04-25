package proxy

import (
	"encoding/json"
	"testing"

	"github.com/Uuq114/JanusLLM/internal/spend"
)

func TestBuildUpstreamURLKeepsSingleV1(t *testing.T) {
	got := buildUpstreamURL("https://example.com/v1", "/v1/chat/completions")
	want := "https://example.com/v1/chat/completions"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildUpstreamURLAppendsV1WhenBaseHasNoV1(t *testing.T) {
	got := buildUpstreamURL("https://example.com/api", "/v1/chat/completions")
	want := "https://example.com/api/v1/chat/completions"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestAnthropicBuildSpendPayloadNormalizesUsageFields(t *testing.T) {
	adapter := &AnthropicAdapter{}

	payload, err := adapter.BuildSpendPayload([]byte(`{
		"id":"msg_123",
		"usage":{"input_tokens":12,"output_tokens":6}
	}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp spend.UpstreamResp
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if resp.Id != "msg_123" {
		t.Fatalf("expected request id to be preserved, got %q", resp.Id)
	}
	if resp.Usage.PromptTokens != 12 || resp.Usage.CompletionTokens != 6 || resp.Usage.TotalTokens != 18 {
		t.Fatalf("unexpected normalized usage: %+v", resp.Usage)
	}
}

func TestAnthropicParseSpendStreamLineMergesMessageUsage(t *testing.T) {
	adapter := &AnthropicAdapter{}

	var requestID string
	var usage *spend.TokenUsage

	adapter.ParseSpendStreamLine([]byte("data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"usage\":{\"input_tokens\":10}}}\n"), &requestID, &usage)
	adapter.ParseSpendStreamLine([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":4}}\n"), &requestID, &usage)

	if requestID != "msg_123" {
		t.Fatalf("expected request id from message_start, got %q", requestID)
	}
	if usage == nil {
		t.Fatal("expected usage to be collected")
	}
	if usage.PromptTokens != 10 || usage.CompletionTokens != 4 || usage.TotalTokens != 14 {
		t.Fatalf("unexpected merged usage: %+v", *usage)
	}
}

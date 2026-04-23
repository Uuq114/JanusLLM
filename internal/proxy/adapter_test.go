package proxy

import "testing"

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

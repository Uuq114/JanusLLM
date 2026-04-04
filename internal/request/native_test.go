package request

import "testing"

func TestExtractModelAndStream(t *testing.T) {
	body := []byte(`{"model":"deepseek-v3","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	model, stream, err := ExtractModelAndStream(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "deepseek-v3" {
		t.Fatalf("expected model deepseek-v3, got %s", model)
	}
	if !stream {
		t.Fatalf("expected stream=true")
	}
}

func TestExtractModelAndStreamNoBody(t *testing.T) {
	model, stream, err := ExtractModelAndStream(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "" || stream {
		t.Fatalf("expected empty values, got model=%q stream=%v", model, stream)
	}
}

func TestExtractModelAndStreamInvalidJSON(t *testing.T) {
	_, _, err := ExtractModelAndStream([]byte("{"))
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

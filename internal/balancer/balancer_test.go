package balancer

import (
	"testing"
	"time"

	"github.com/Uuq114/JanusLLM/internal/models"
)

func TestClientStickyBalancerKeepsSameClientOnSameBackend(t *testing.T) {
	blcr := NewClientStickyBalancer()
	blcr.AddModel(&models.ModelConfig{Name: "upstream-a", BaseURL: "http://a"})
	blcr.AddModel(&models.ModelConfig{Name: "upstream-b", BaseURL: "http://b"})
	blcr.AddModel(&models.ModelConfig{Name: "upstream-c", BaseURL: "http://c"})

	ctx := SelectionContext{
		ClientKey:  "key:42",
		ModelGroup: "chat",
		Path:       "/v1/chat/completions",
	}
	first := blcr.Next(ctx)
	if first == nil {
		t.Fatal("expected backend for sticky client")
	}

	for i := 0; i < 20; i++ {
		got := blcr.Next(ctx)
		if got != first {
			t.Fatalf("expected sticky client to stay on %q, got %q", first.Name, got.Name)
		}
	}
}

func TestClientStickyBalancerUsesStableRequestInfo(t *testing.T) {
	blcr := NewClientStickyBalancer()
	blcr.AddModel(&models.ModelConfig{Name: "upstream-a", BaseURL: "http://a"})
	blcr.AddModel(&models.ModelConfig{Name: "upstream-b", BaseURL: "http://b"})

	ctx := SelectionContext{
		KeyID:      7,
		TeamID:     3,
		ModelGroup: "chat",
		Path:       "/v1/messages",
		ClientIP:   "192.0.2.10",
		Headers: map[string]string{
			"X-User-ID": "user-1",
		},
	}
	first := blcr.Next(ctx)
	if first == nil {
		t.Fatal("expected backend for sticky request info")
	}
	if got := blcr.Next(ctx); got != first {
		t.Fatalf("expected key/team/header/ip context to hash stably, got %q then %q", first.Name, got.Name)
	}
}

func TestLatencyBalancerPrefersLowestSuccessfulLatency(t *testing.T) {
	slow := &models.ModelConfig{Name: "slow", BaseURL: "http://slow"}
	fast := &models.ModelConfig{Name: "fast", BaseURL: "http://fast"}
	blcr := NewLatencyBalancer()
	blcr.AddModel(slow)
	blcr.AddModel(fast)

	blcr.Observe(slow, 200*time.Millisecond, true)
	blcr.Observe(fast, 20*time.Millisecond, true)

	got := blcr.Next(SelectionContext{})
	if got != fast {
		t.Fatalf("expected fastest backend, got %#v", got)
	}
}

func TestLatencyBalancerFallsBackWithoutHistory(t *testing.T) {
	blcr := NewLatencyBalancer()
	blcr.AddModel(&models.ModelConfig{Name: "upstream-a", BaseURL: "http://a"})
	blcr.AddModel(&models.ModelConfig{Name: "upstream-b", BaseURL: "http://b"})

	if got := blcr.Next(SelectionContext{}); got == nil {
		t.Fatal("expected fallback backend without latency history")
	}
}

func TestNewBalancerSupportsConfiguredStrategies(t *testing.T) {
	tests := map[string]string{
		"":              "*balancer.RoundRobinBalancer",
		"round_robin":   "*balancer.RoundRobinBalancer",
		"weighted":      "*balancer.WeightedBalancer",
		"latency-based": "*balancer.LatencyBalancer",
		"client-sticky": "*balancer.ClientStickyBalancer",
	}

	for strategy, want := range tests {
		got := New(strategy)
		if gotType := typeName(got); gotType != want {
			t.Fatalf("strategy %q: expected %s, got %s", strategy, want, gotType)
		}
	}
}

func typeName(value interface{}) string {
	switch value.(type) {
	case *RoundRobinBalancer:
		return "*balancer.RoundRobinBalancer"
	case *WeightedBalancer:
		return "*balancer.WeightedBalancer"
	case *LatencyBalancer:
		return "*balancer.LatencyBalancer"
	case *ClientStickyBalancer:
		return "*balancer.ClientStickyBalancer"
	default:
		return "unknown"
	}
}

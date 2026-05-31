package spend

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Uuq114/JanusLLM/internal/auth"
	"github.com/gin-gonic/gin"
)

func TestCreateKeySpendRecordSkipsWhenSpendMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ch := make(chan float64, 1)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic when spend context is missing, got %v", r)
		}
	}()

	CreateKeySpendRecord(ctx, ch)

	select {
	case got := <-ch:
		t.Fatalf("expected no spend to be enqueued, got %v", got)
	default:
	}
}

func TestCreateKeySpendRecordEnqueuesSpend(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set(ContextSpend, 1.25)
	ch := make(chan float64, 1)

	CreateKeySpendRecord(ctx, ch)

	select {
	case got := <-ch:
		if got != 1.25 {
			t.Fatalf("expected spend 1.25, got %v", got)
		}
	default:
		t.Fatalf("expected spend to be enqueued")
	}
}

func TestCreateSpendRecordEnqueuesBillingMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalPrice := ModelPrice
	ModelPrice = map[string][]float64{"chat-group": {0.01, 0.02}}
	t.Cleanup(func() { ModelPrice = originalPrice })

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set("key", auth.Key{
		KeyId:          42,
		KeyContent:     "sk-abcdef123456",
		TeamId:         7,
		OrganizationId: 3,
	})
	ctx.Set("modelGroup", "chat-group")
	ctx.Set(ContextProvider, "anthropic")
	ctx.Set(ContextLatencyMS, int64(123))
	ctx.Set(ContextCacheHit, true)

	payload, err := json.Marshal(UpstreamResp{
		Id: "req-123",
		Usage: TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	})
	if err != nil {
		t.Fatalf("failed to build upstream payload: %v", err)
	}
	ctx.Set(ContextUpstreamResp, payload)
	ch := make(chan SpendRecord, 1)

	CreateSpendRecord(ctx, ch)

	select {
	case got := <-ch:
		if got.RequestId != "req-123" {
			t.Fatalf("expected request id req-123, got %q", got.RequestId)
		}
		if got.Provider != "anthropic" || got.LatencyMS != 123 || !got.CacheHit {
			t.Fatalf("unexpected metadata: provider=%q latency=%d cache_hit=%v", got.Provider, got.LatencyMS, got.CacheHit)
		}
		if got.Tenant != "org:3/team:7" {
			t.Fatalf("expected tenant org:3/team:7, got %q", got.Tenant)
		}
		if got.Spend != 0.2 {
			t.Fatalf("expected spend 0.2, got %v", got.Spend)
		}
	default:
		t.Fatalf("expected spend record to be enqueued")
	}
}

func TestCreateSpendRecordSkipsZeroTokenUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalPrice := ModelPrice
	ModelPrice = map[string][]float64{"chat-group": {0.01, 0.02}}
	t.Cleanup(func() { ModelPrice = originalPrice })

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set("key", auth.Key{KeyId: 42, KeyContent: "sk-abcdef123456", TeamId: 7, OrganizationId: 3})
	ctx.Set("modelGroup", "chat-group")
	ctx.Set(ContextUpstreamResp, []byte(`{"id":"req-123","usage":{}}`))
	ch := make(chan SpendRecord, 1)

	CreateSpendRecord(ctx, ch)

	select {
	case got := <-ch:
		t.Fatalf("expected no spend record to be enqueued, got %+v", got)
	default:
	}
	if _, exists := ctx.Get(ContextSpend); exists {
		t.Fatalf("expected key spend context not to be set")
	}
}

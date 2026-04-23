package spend

import (
	"net/http/httptest"
	"testing"

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
	ctx.Set("spend", 1.25)
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

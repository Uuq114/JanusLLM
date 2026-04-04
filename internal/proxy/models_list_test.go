package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Uuq114/JanusLLM/internal/auth"
	"github.com/Uuq114/JanusLLM/internal/models"
)

type listModelsResp struct {
	Object string `json:"object"`
	Data   []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func TestHandleListModelsWildcard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	p := NewProxy()
	p.RegisterModelGroup(&models.ModelGroup{Name: "deepseek-v3", Strategy: "round-robin"})
	p.RegisterModelGroup(&models.ModelGroup{Name: "claude-3-sonnet", Strategy: "round-robin"})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set("key", auth.Key{ModelList: auth.StringSlice{"*"}})

	p.HandleListModels(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var resp listModelsResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 models, got %d", len(resp.Data))
	}
}

func TestHandleListModelsRestricted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	p := NewProxy()
	p.RegisterModelGroup(&models.ModelGroup{Name: "deepseek-v3", Strategy: "round-robin"})
	p.RegisterModelGroup(&models.ModelGroup{Name: "claude-3-sonnet", Strategy: "round-robin"})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set("key", auth.Key{ModelList: auth.StringSlice{"claude-3-sonnet"}})

	p.HandleListModels(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var resp listModelsResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 model, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "claude-3-sonnet" {
		t.Fatalf("unexpected model id: %s", resp.Data[0].ID)
	}
}

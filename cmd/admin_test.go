package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Uuq114/JanusLLM/internal/auth"
)

func TestValidateRequestPerMinuteRejectsNegativeValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	if validateRequestPerMinute(ctx, -1) {
		t.Fatal("expected negative request_per_minute to be rejected")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "request_per_minute must be non-negative") {
		t.Fatalf("unexpected response body: %q", rec.Body.String())
	}
}

func TestInvalidKeyResultRejectsStoredNegativeRPM(t *testing.T) {
	key := auth.Key{
		Balance:          100,
		RequestPerMinute: -1,
	}

	result, invalid := invalidKeyResult(key, time.Now())
	if !invalid {
		t.Fatal("expected key with negative request_per_minute to be invalid")
	}
	if result.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", result.StatusCode)
	}
	if result.ErrorCode != "invalid_rate_limit_config" {
		t.Fatalf("unexpected error code: %q", result.ErrorCode)
	}
}

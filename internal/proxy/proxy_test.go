package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Uuq114/JanusLLM/internal/balancer"
	"github.com/Uuq114/JanusLLM/internal/models"
	"github.com/Uuq114/JanusLLM/internal/spend"
)

type sequenceBalancer struct {
	models []*models.ModelConfig
	next   int
}

func (b *sequenceBalancer) Next() *models.ModelConfig {
	if len(b.models) == 0 {
		return nil
	}
	if b.next >= len(b.models) {
		return b.models[len(b.models)-1]
	}
	model := b.models[b.next]
	b.next++
	return model
}

func (b *sequenceBalancer) AddModel(model *models.ModelConfig) {
	b.models = append(b.models, model)
}

func (b *sequenceBalancer) Models() []*models.ModelConfig {
	out := make([]*models.ModelConfig, len(b.models))
	copy(out, b.models)
	return out
}

func (b *sequenceBalancer) RemoveModel(modelName string) {}

func (b *sequenceBalancer) Size() int {
	return len(b.models)
}

func TestHandleRequestDoesNotRetryAfterStreamingWriteStarts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var primaryHits int32
	primaryURL, closePrimary := startBrokenStreamServer(t, &primaryHits)
	defer closePrimary()

	var backupHits int32
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&backupHits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer backup.Close()

	groupName := "stream-group"
	p := &Proxy{
		balancers: map[string]balancer.Balancer{
			groupName: &sequenceBalancer{
				models: []*models.ModelConfig{
					{Name: "primary", BaseURL: primaryURL},
					{Name: "backup", BaseURL: backup.URL},
				},
			},
		},
		groups: map[string]models.ModelGroup{
			groupName: {Name: groupName},
		},
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"stream-group","stream":true}`)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	ctx.Request.Header.Set("Accept", "text/event-stream")
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("modelGroup", groupName)
	ctx.Set("rawBody", body)
	ctx.Set("logger", zap.NewNop())
	ctx.Set("isStreamRequest", true)

	p.HandleRequest(ctx)

	if got := atomic.LoadInt32(&primaryHits); got != 1 {
		t.Fatalf("expected primary upstream to be called once, got %d", got)
	}
	if got := atomic.LoadInt32(&backupHits); got != 0 {
		t.Fatalf("expected backup upstream not to be retried after stream started, got %d calls", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, "data: first\n") {
		t.Fatalf("expected partial streamed response to be preserved, got %q", body)
	}
}

func TestDistinctRetryCandidatesStayUniqueForWeightedBalancer(t *testing.T) {
	blcr := balancer.NewWeightedBalancer()
	blcr.AddModel(&models.ModelConfig{Name: "primary", BaseURL: "http://primary", Weight: 100})
	blcr.AddModel(&models.ModelConfig{Name: "backup-a", BaseURL: "http://backup-a", Weight: 1})
	blcr.AddModel(&models.ModelConfig{Name: "backup-b", BaseURL: "http://backup-b", Weight: 1})

	candidates := distinctRetryCandidates(blcr)
	if len(candidates) != 3 {
		t.Fatalf("expected 3 distinct candidates, got %d", len(candidates))
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		key := upstreamModelKey(candidate)
		if _, exists := seen[key]; exists {
			t.Fatalf("candidate %q appeared more than once", key)
		}
		seen[key] = struct{}{}
	}
}

func TestBuildHTTPClientDisablesTotalTimeoutForStreamRequests(t *testing.T) {
	client := buildHTTPClient(&models.ModelConfig{}, true, 5*time.Second)

	if client.Timeout != 0 {
		t.Fatalf("expected streaming client timeout to be disabled, got %v", client.Timeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected http.Transport, got %T", client.Transport)
	}
	if transport.ResponseHeaderTimeout != 5*time.Second {
		t.Fatalf("expected response header timeout to be preserved, got %v", transport.ResponseHeaderTimeout)
	}
}

func TestHandleRequestRetriesDistinctWeightedBackup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var primaryHits int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&primaryHits, 1)
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer primary.Close()

	var backupHits int32
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&backupHits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	defer backup.Close()

	group := &models.ModelGroup{
		Name:     "weighted-group",
		Strategy: "weighted",
		Models: []models.ModelConfig{
			{Name: "primary", BaseURL: primary.URL, Weight: 100},
			{Name: "backup", BaseURL: backup.URL, Weight: 1},
		},
	}

	p := NewProxy()
	p.RegisterModelGroup(group)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"weighted-group","messages":[{"role":"user","content":"hi"}]}`)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("modelGroup", group.Name)
	ctx.Set("rawBody", body)
	ctx.Set("logger", zap.NewNop())
	ctx.Set("isStreamRequest", false)

	p.HandleRequest(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected backup retry to succeed, got status %d body %q", rec.Code, rec.Body.String())
	}
	if got := atomic.LoadInt32(&primaryHits); got != 1 {
		t.Fatalf("expected primary upstream to be tried once, got %d", got)
	}
	if got := atomic.LoadInt32(&backupHits); got != 1 {
		t.Fatalf("expected backup upstream to be tried once, got %d", got)
	}
}

func TestHandleRequestRecordsSpendForJSONWithoutContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamHits int32
	respBody := []byte(`{"id":"chatcmpl-123","usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}}`)
	upstreamURL, closeUpstream := startRawJSONServer(t, &upstreamHits, http.StatusOK, respBody)
	defer closeUpstream()

	groupName := "json-group"
	p := &Proxy{
		balancers: map[string]balancer.Balancer{
			groupName: &sequenceBalancer{
				models: []*models.ModelConfig{
					{Name: "json-model", BaseURL: upstreamURL},
				},
			},
		},
		groups: map[string]models.ModelGroup{
			groupName: {Name: groupName},
		},
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"json-group","messages":[{"role":"user","content":"hi"}]}`)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("modelGroup", groupName)
	ctx.Set("rawBody", body)
	ctx.Set("logger", zap.NewNop())
	ctx.Set("isStreamRequest", false)

	p.HandleRequest(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected upstream response to succeed, got status %d body %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected default response content type application/json, got %q", got)
	}
	if got := atomic.LoadInt32(&upstreamHits); got != 1 {
		t.Fatalf("expected upstream to be called once, got %d", got)
	}

	spendValue, exists := ctx.Get("upstreamResp")
	if !exists {
		t.Fatal("expected upstreamResp to be recorded for JSON body without content type")
	}
	spendPayload, ok := spendValue.([]byte)
	if !ok {
		t.Fatalf("expected upstreamResp to be []byte, got %T", spendValue)
	}

	var upstreamResp spend.UpstreamResp
	if err := json.Unmarshal(spendPayload, &upstreamResp); err != nil {
		t.Fatalf("failed to decode upstreamResp: %v", err)
	}
	if upstreamResp.Id != "chatcmpl-123" {
		t.Fatalf("expected request id to be preserved, got %q", upstreamResp.Id)
	}
	if upstreamResp.Usage.PromptTokens != 3 || upstreamResp.Usage.CompletionTokens != 4 || upstreamResp.Usage.TotalTokens != 7 {
		t.Fatalf("unexpected usage recorded: %+v", upstreamResp.Usage)
	}
}

func TestHandleRequestHonorsRetryTimesForSameUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var hits int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		if hit == 1 {
			http.Error(w, "temporary failure", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	defer upstream.Close()

	groupName := "retry-group"
	p := &Proxy{
		balancers: map[string]balancer.Balancer{
			groupName: &sequenceBalancer{
				models: []*models.ModelConfig{
					{Name: "retry-model", BaseURL: upstream.URL, RetryTimes: 1},
				},
			},
		},
		groups: map[string]models.ModelGroup{
			groupName: {Name: groupName},
		},
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"retry-group","messages":[{"role":"user","content":"hi"}]}`)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("modelGroup", groupName)
	ctx.Set("rawBody", body)
	ctx.Set("logger", zap.NewNop())
	ctx.Set("isStreamRequest", false)

	p.HandleRequest(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected retry to succeed, got status %d body %q", rec.Code, rec.Body.String())
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected one original request and one retry, got %d calls", got)
	}
}

func startBrokenStreamServer(t *testing.T, hits *int32) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				atomic.AddInt32(hits, 1)

				reader := bufio.NewReader(conn)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if line == "\r\n" {
						break
					}
				}

				_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\n")
				_, _ = io.WriteString(conn, "Content-Type: text/event-stream\r\n")
				_, _ = io.WriteString(conn, "Content-Length: 100\r\n")
				_, _ = io.WriteString(conn, "\r\n")
				_, _ = io.WriteString(conn, "data: first\n")
			}(conn)
		}
	}()

	closeFn := func() {
		_ = listener.Close()
		<-done
	}

	return "http://" + listener.Addr().String(), closeFn
}

func startRawJSONServer(t *testing.T, hits *int32, status int, body []byte) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				atomic.AddInt32(hits, 1)

				reader := bufio.NewReader(conn)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if line == "\r\n" {
						break
					}
				}

				_, _ = io.WriteString(conn, "HTTP/1.1 "+strconv.Itoa(status)+" "+http.StatusText(status)+"\r\n")
				_, _ = io.WriteString(conn, "Content-Length: "+strconv.Itoa(len(body))+"\r\n")
				_, _ = io.WriteString(conn, "\r\n")
				_, _ = conn.Write(body)
			}(conn)
		}
	}()

	closeFn := func() {
		_ = listener.Close()
		<-done
	}

	return "http://" + listener.Addr().String(), closeFn
}

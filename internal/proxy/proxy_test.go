package proxy

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Uuq114/JanusLLM/internal/balancer"
	"github.com/Uuq114/JanusLLM/internal/models"
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

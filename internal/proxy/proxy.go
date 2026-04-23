package proxy

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"

	"github.com/Uuq114/JanusLLM/internal/auth"
	"github.com/Uuq114/JanusLLM/internal/balancer"
	"github.com/Uuq114/JanusLLM/internal/models"
	"github.com/Uuq114/JanusLLM/internal/spend"
)

var (
	// SpendLogQueue stores request-level spend records before batch insert.
	SpendLogQueue = make(chan spend.SpendRecord, 100)

	keyRequestRingMu sync.RWMutex
	keyRequestRing   = make(map[string]*RequestRing)

	keySpendQueueMu sync.RWMutex
	keySpendQueue   = make(map[string]chan float64)
)

type Proxy struct {
	balancers map[string]balancer.Balancer
	groups    map[string]models.ModelGroup
}

func NewProxy() *Proxy {
	return &Proxy{
		balancers: make(map[string]balancer.Balancer),
		groups:    make(map[string]models.ModelGroup),
	}
}

func (p *Proxy) RegisterModelGroup(group *models.ModelGroup) {
	var b balancer.Balancer
	switch strings.ToLower(group.Strategy) {
	case "weighted":
		b = balancer.NewWeightedBalancer()
	default:
		b = balancer.NewRoundRobinBalancer()
	}

	for _, model := range group.Models {
		b.AddModel(&model)
	}

	p.balancers[group.Name] = b
	p.groups[group.Name] = *group
}

func (p *Proxy) HandleListModels(c *gin.Context) {
	keyValue, ok := c.Get("key")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "key context not found"})
		return
	}
	keyInfo := keyValue.(auth.Key)

	allowed := p.accessibleModelGroups(keyInfo.ModelList)
	data := make([]gin.H, 0, len(allowed))
	for _, model := range allowed {
		data = append(data, gin.H{
			"id":       model,
			"object":   "model",
			"owned_by": "janusllm",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func (p *Proxy) accessibleModelGroups(modelList auth.StringSlice) []string {
	all := make([]string, 0, len(p.groups))
	for name := range p.groups {
		all = append(all, name)
	}
	sort.Strings(all)

	if len(modelList) == 0 {
		return []string{}
	}
	if modelList[0] == "*" {
		return all
	}

	allowedSet := make(map[string]struct{}, len(modelList))
	for _, model := range modelList {
		allowedSet[model] = struct{}{}
	}

	out := make([]string, 0, len(all))
	for _, model := range all {
		if _, ok := allowedSet[model]; ok {
			out = append(out, model)
		}
	}
	return out
}

func GetOrCreateRequestRing(key string, rpm int) *RequestRing {
	if rpm <= 0 {
		return nil
	}

	keyRequestRingMu.RLock()
	ring, ok := keyRequestRing[key]
	keyRequestRingMu.RUnlock()
	if ok && ring != nil && ring.maxRequests == rpm {
		return ring
	}

	keyRequestRingMu.Lock()
	defer keyRequestRingMu.Unlock()
	if existing, exists := keyRequestRing[key]; exists {
		if existing != nil && existing.maxRequests == rpm {
			return existing
		}
	}
	created := NewRequestRing(1*time.Minute, rpm)
	keyRequestRing[key] = created
	return created
}

func RemoveRequestRing(key string) {
	keyRequestRingMu.Lock()
	defer keyRequestRingMu.Unlock()
	delete(keyRequestRing, key)
}

func GetOrCreateKeySpendQueue(key string) chan float64 {
	keySpendQueueMu.RLock()
	ch, ok := keySpendQueue[key]
	keySpendQueueMu.RUnlock()
	if ok {
		return ch
	}

	keySpendQueueMu.Lock()
	defer keySpendQueueMu.Unlock()
	if existing, exists := keySpendQueue[key]; exists {
		return existing
	}
	created := make(chan float64, 100)
	keySpendQueue[key] = created
	return created
}

func SnapshotKeySpendQueue() map[string]chan float64 {
	keySpendQueueMu.RLock()
	defer keySpendQueueMu.RUnlock()

	out := make(map[string]chan float64, len(keySpendQueue))
	for k, v := range keySpendQueue {
		out[k] = v
	}
	return out
}

func (p *Proxy) HandleRequest(c *gin.Context) {
	modelGroup := c.MustGet("modelGroup").(string)
	rawBody := c.MustGet("rawBody").([]byte)
	logger := c.MustGet("logger").(*zap.Logger)
	endpointPath := c.Request.URL.Path

	blcr, exists := p.balancers[modelGroup]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "model group not found"})
		return
	}
	if blcr.Size() == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available models"})
		return
	}

	attempts := max(1, blcr.Size())
	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		upstreamModel := blcr.Next()
		if upstreamModel == nil {
			continue
		}

		status, shouldRetry, err := p.forwardOnce(c, endpointPath, modelGroup, upstreamModel, rawBody, logger)
		if err == nil {
			return
		}

		lastErr = err
		if !shouldRetry {
			logger.Warn("request failed without retry", zap.Error(err), zap.Int("status", status), zap.Int("attempt", attempt+1))
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		logger.Warn("request failed and retrying another upstream",
			zap.Error(err),
			zap.Int("status", status),
			zap.Int("attempt", attempt+1),
			zap.String("upstream", upstreamModel.Name),
		)
	}

	if lastErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": lastErr.Error()})
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": "all upstream models failed"})
}

func (p *Proxy) forwardOnce(c *gin.Context, endpointPath string, modelGroup string, upstreamModel *models.ModelConfig, rawBody []byte, logger *zap.Logger) (int, bool, error) {
	groupCfg, ok := p.groups[modelGroup]
	if !ok {
		return http.StatusNotFound, false, fmt.Errorf("model group not found: %s", modelGroup)
	}
	preparedBody, err := prepareUpstreamBody(rawBody, upstreamModel.Name, groupCfg.RequestDefaults)
	if err != nil {
		return http.StatusBadRequest, false, err
	}

	adapter := SelectAdapter(upstreamModel)
	req, err := adapter.BuildRequest(c, endpointPath, upstreamModel, preparedBody)
	if err != nil {
		return http.StatusInternalServerError, false, err
	}

	timeoutSeconds := upstreamModel.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60
	}
	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
	if upstreamModel.SkipTLSVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusBadGateway, true, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusInternalServerError {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return http.StatusBadGateway, true, readErr
		}
		return resp.StatusCode, true, fmt.Errorf("upstream status %d: %s", resp.StatusCode, truncateBody(respBody))
	}

	if shouldStream(c, resp) {
		copyResponseHeaders(c, resp.Header)
		c.Status(resp.StatusCode)
		streamUsage, streamErr := streamToClient(c, resp.Body)
		if streamErr != nil {
			return http.StatusBadGateway, true, streamErr
		}
		if len(streamUsage) > 0 {
			c.Set("upstreamResp", streamUsage)
		}
		return resp.StatusCode, false, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusBadGateway, true, err
	}

	copyResponseHeaders(c, resp.Header)
	c.Data(resp.StatusCode, contentType(resp.Header), respBody)

	if isJSONResponse(resp.Header) {
		c.Set("upstreamResp", respBody)
	}

	logger.Info("upstream request succeeded",
		zap.Int("status", resp.StatusCode),
		zap.String("model", modelGroup),
		zap.String("upstream", upstreamModel.Name),
	)

	return resp.StatusCode, false, nil
}

func shouldStream(c *gin.Context, resp *http.Response) bool {
	if !isSSEStreamResponse(resp.Header) {
		return false
	}
	if streamFlag, ok := c.Get("isStreamRequest"); ok {
		if isStream, ok := streamFlag.(bool); ok {
			return isStream
		}
	}
	return false
}

func streamToClient(c *gin.Context, body io.Reader) ([]byte, error) {
	flusher, _ := c.Writer.(http.Flusher)
	reader := bufio.NewReader(body)
	var requestID string
	var usage *spend.TokenUsage

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if _, writeErr := c.Writer.Write(line); writeErr != nil {
				return nil, writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
			parseSSEUsageLine(line, &requestID, &usage)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	if usage == nil {
		return nil, nil
	}

	payload, err := json.Marshal(spend.UpstreamResp{
		Id:    requestID,
		Usage: *usage,
	})
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func parseSSEUsageLine(line []byte, requestID *string, usage **spend.TokenUsage) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" || !strings.HasPrefix(trimmed, "data:") {
		return
	}

	data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	if data == "" || data == "[DONE]" {
		return
	}

	var chunk struct {
		ID    string            `json:"id"`
		Usage *spend.TokenUsage `json:"usage"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return
	}

	if chunk.ID != "" {
		*requestID = chunk.ID
	}
	if chunk.Usage != nil {
		copied := *chunk.Usage
		*usage = &copied
	}
}

func copyResponseHeaders(c *gin.Context, headers http.Header) {
	for key, values := range headers {
		for _, value := range values {
			c.Header(key, value)
		}
	}
}

func contentType(headers http.Header) string {
	ct := headers.Get("Content-Type")
	if ct == "" {
		return "application/json"
	}
	return ct
}

func isJSONResponse(headers http.Header) bool {
	return strings.Contains(strings.ToLower(headers.Get("Content-Type")), "application/json")
}

func isSSEStreamResponse(headers http.Header) bool {
	return strings.Contains(strings.ToLower(headers.Get("Content-Type")), "text/event-stream")
}

func truncateBody(body []byte) string {
	const limit = 256
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "..."
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func prepareUpstreamBody(rawBody []byte, upstreamModel string, defaults map[string]interface{}) ([]byte, error) {
	if len(rawBody) == 0 {
		return rawBody, nil
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	if _, ok := body["model"]; ok && upstreamModel != "" {
		body["model"] = upstreamModel
	}
	for k, v := range defaults {
		if _, exists := body[k]; !exists {
			body[k] = v
		}
	}

	out, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return out, nil
}

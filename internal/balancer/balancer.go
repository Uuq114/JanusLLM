package balancer

import (
	"hash/fnv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Uuq114/JanusLLM/internal/models"
)

type SelectionContext struct {
	ClientKey      string
	KeyID          int
	KeyName        string
	TeamID         int
	OrganizationID int
	ModelGroup     string
	Path           string
	ClientIP       string
	Headers        map[string]string
}

// Balancer chooses one upstream model endpoint for each request.
type Balancer interface {
	Next(ctx SelectionContext) *models.ModelConfig
	Models() []*models.ModelConfig
	AddModel(model *models.ModelConfig)
	RemoveModel(modelName string)
	Size() int
}

type Observer interface {
	Observe(model *models.ModelConfig, latency time.Duration, success bool)
}

func New(strategy string) Balancer {
	switch NormalizeStrategy(strategy) {
	case "weighted":
		return NewWeightedBalancer()
	case "latency":
		return NewLatencyBalancer()
	case "client-sticky":
		return NewClientStickyBalancer()
	default:
		return NewRoundRobinBalancer()
	}
}

func NormalizeStrategy(strategy string) string {
	normalized := strings.ToLower(strings.TrimSpace(strategy))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	switch normalized {
	case "", "round-robin", "roundrobin", "rr":
		return "round-robin"
	case "weighted", "weight":
		return "weighted"
	case "latency", "latency-based", "least-latency", "fastest":
		return "latency"
	case "client-sticky", "sticky", "client-sticky-hash", "sticky-hash":
		return "client-sticky"
	default:
		return "round-robin"
	}
}

// RoundRobinBalancer picks endpoints in order.
type RoundRobinBalancer struct {
	models []*models.ModelConfig
	index  uint64
	mu     sync.RWMutex
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{
		models: make([]*models.ModelConfig, 0),
	}
}

func (rb *RoundRobinBalancer) Next(ctx SelectionContext) *models.ModelConfig {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if len(rb.models) == 0 {
		return nil
	}

	index := atomic.AddUint64(&rb.index, 1) % uint64(len(rb.models))
	return rb.models[index]
}

func (rb *RoundRobinBalancer) AddModel(model *models.ModelConfig) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.models = append(rb.models, model)
}

func (rb *RoundRobinBalancer) Models() []*models.ModelConfig {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	out := make([]*models.ModelConfig, len(rb.models))
	copy(out, rb.models)
	return out
}

func (rb *RoundRobinBalancer) RemoveModel(modelName string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for i, model := range rb.models {
		if model.Name == modelName {
			rb.models = append(rb.models[:i], rb.models[i+1:]...)
			break
		}
	}
}

func (rb *RoundRobinBalancer) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return len(rb.models)
}

// WeightedBalancer picks endpoints by configured weight.
type WeightedBalancer struct {
	models []*models.ModelConfig
	index  uint64
	mu     sync.RWMutex
}

func NewWeightedBalancer() *WeightedBalancer {
	return &WeightedBalancer{
		models: make([]*models.ModelConfig, 0),
	}
}

func (wb *WeightedBalancer) Next(ctx SelectionContext) *models.ModelConfig {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	if len(wb.models) == 0 {
		return nil
	}

	totalWeight := 0
	for _, model := range wb.models {
		totalWeight += model.Weight
	}
	if totalWeight <= 0 {
		index := atomic.AddUint64(&wb.index, 1) % uint64(len(wb.models))
		return wb.models[index]
	}

	index := atomic.AddUint64(&wb.index, 1)
	currentWeight := 0
	for _, model := range wb.models {
		currentWeight += model.Weight
		if uint64(currentWeight) > index%uint64(totalWeight) {
			return model
		}
	}

	return wb.models[0]
}

func (wb *WeightedBalancer) AddModel(model *models.ModelConfig) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	wb.models = append(wb.models, model)
}

func (wb *WeightedBalancer) Models() []*models.ModelConfig {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	out := make([]*models.ModelConfig, len(wb.models))
	copy(out, wb.models)
	return out
}

func (wb *WeightedBalancer) RemoveModel(modelName string) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	for i, model := range wb.models {
		if model.Name == modelName {
			wb.models = append(wb.models[:i], wb.models[i+1:]...)
			break
		}
	}
}

func (wb *WeightedBalancer) Size() int {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	return len(wb.models)
}

type LatencyBalancer struct {
	models []*models.ModelConfig
	stats  map[string]latencyStat
	index  uint64
	mu     sync.RWMutex
}

type latencyStat struct {
	Latency time.Duration
	Seen    bool
	Healthy bool
}

func NewLatencyBalancer() *LatencyBalancer {
	return &LatencyBalancer{
		models: make([]*models.ModelConfig, 0),
		stats:  make(map[string]latencyStat),
	}
}

func (lb *LatencyBalancer) Next(ctx SelectionContext) *models.ModelConfig {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.models) == 0 {
		return nil
	}

	var selected *models.ModelConfig
	var selectedLatency time.Duration
	for _, model := range lb.models {
		stat := lb.stats[modelKey(model)]
		if stat.Seen && !stat.Healthy {
			continue
		}
		if !stat.Seen {
			continue
		}
		if selected == nil || stat.Latency < selectedLatency {
			selected = model
			selectedLatency = stat.Latency
		}
	}
	if selected != nil {
		return selected
	}

	index := atomic.AddUint64(&lb.index, 1) % uint64(len(lb.models))
	return lb.models[index]
}

func (lb *LatencyBalancer) AddModel(model *models.ModelConfig) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.models = append(lb.models, model)
}

func (lb *LatencyBalancer) Models() []*models.ModelConfig {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	out := make([]*models.ModelConfig, len(lb.models))
	copy(out, lb.models)
	return out
}

func (lb *LatencyBalancer) RemoveModel(modelName string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, model := range lb.models {
		if model.Name == modelName {
			delete(lb.stats, modelKey(model))
			lb.models = append(lb.models[:i], lb.models[i+1:]...)
			break
		}
	}
}

func (lb *LatencyBalancer) Size() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return len(lb.models)
}

func (lb *LatencyBalancer) Observe(model *models.ModelConfig, latency time.Duration, success bool) {
	if model == nil || latency < 0 {
		return
	}
	if latency == 0 {
		latency = time.Nanosecond
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	key := modelKey(model)
	stat := lb.stats[key]
	if !stat.Seen {
		stat.Latency = latency
	} else {
		stat.Latency = (stat.Latency*7 + latency) / 8
	}
	stat.Seen = true
	stat.Healthy = success
	lb.stats[key] = stat
}

type ClientStickyBalancer struct {
	models []*models.ModelConfig
	mu     sync.RWMutex
}

func NewClientStickyBalancer() *ClientStickyBalancer {
	return &ClientStickyBalancer{
		models: make([]*models.ModelConfig, 0),
	}
}

func (sb *ClientStickyBalancer) Next(ctx SelectionContext) *models.ModelConfig {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if len(sb.models) == 0 {
		return nil
	}
	key := stickyKey(ctx)

	hash := fnv.New32a()
	_, _ = hash.Write([]byte(key))
	return sb.models[int(hash.Sum32())%len(sb.models)]
}

func (sb *ClientStickyBalancer) AddModel(model *models.ModelConfig) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.models = append(sb.models, model)
}

func (sb *ClientStickyBalancer) Models() []*models.ModelConfig {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	out := make([]*models.ModelConfig, len(sb.models))
	copy(out, sb.models)
	return out
}

func (sb *ClientStickyBalancer) RemoveModel(modelName string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	for i, model := range sb.models {
		if model.Name == modelName {
			sb.models = append(sb.models[:i], sb.models[i+1:]...)
			break
		}
	}
}

func (sb *ClientStickyBalancer) Size() int {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return len(sb.models)
}

func modelKey(model *models.ModelConfig) string {
	if model == nil {
		return ""
	}
	return model.Name + "\x00" + model.BaseURL
}

func stickyKey(ctx SelectionContext) string {
	if ctx.ClientKey != "" {
		return ctx.ClientKey
	}
	if ctx.KeyID > 0 {
		return "key:" + intString(ctx.KeyID)
	}
	if ctx.TeamID > 0 {
		return "team:" + intString(ctx.TeamID)
	}
	if ctx.OrganizationID > 0 {
		return "org:" + intString(ctx.OrganizationID)
	}
	for _, header := range []string{
		"x-janus-client",
		"x-client-id",
		"x-user-id",
		"x-team-id",
		"x-forwarded-user",
	} {
		if value := headerValue(ctx.Headers, header); value != "" {
			return header + ":" + value
		}
	}
	if ctx.ClientIP != "" {
		return "ip:" + ctx.ClientIP
	}
	return ctx.ModelGroup + ":" + ctx.Path
}

func headerValue(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	v := value
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

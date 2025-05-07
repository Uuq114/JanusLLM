package balancer

import (
	"sync"
	"sync/atomic"

	"github.com/Uuq114/JanusLLM/internal/models"
)

// balancer

type Balancer interface {
	Next() *models.ModelConfig
	AddModel(model *models.ModelConfig)
	RemoveModel(modelName string)
}

// round-robin balancer

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

func (rb *RoundRobinBalancer) Next() *models.ModelConfig {
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

// weighted balancer

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

func (wb *WeightedBalancer) Next() *models.ModelConfig {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	if len(wb.models) == 0 {
		return nil
	}

	// 计算总权重
	totalWeight := 0
	for _, model := range wb.models {
		totalWeight += model.Weight
	}

	// 使用原子操作获取下一个索引
	index := atomic.AddUint64(&wb.index, 1)

	// 根据权重选择模型
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

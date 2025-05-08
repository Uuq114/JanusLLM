package proxy

import (
	"bytes"
	"io"
	"net/http"

	"github.com/Uuq114/JanusLLM/internal/balancer"
	"github.com/Uuq114/JanusLLM/internal/models"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

type Proxy struct {
	balancers map[string]balancer.Balancer
}

func NewProxy() *Proxy {
	return &Proxy{
		balancers: make(map[string]balancer.Balancer),
	}
}

func (p *Proxy) RegisterModelGroup(group *models.ModelGroup) {
	var b balancer.Balancer
	switch group.Strategy {
	case "round-robin":
		b = balancer.NewRoundRobinBalancer()
	case "weighted":
		b = balancer.NewWeightedBalancer()
	}
	for _, model := range group.Models {
		b.AddModel(&model)
	}
	p.balancers[group.Name] = b
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatReqBody struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	DoSample    bool      `json:"do_sample" default:true`
	Temperature float64   `json:"temperature" default:0.7`
	TopP        float64   `json:"top_p" default:1.0`
	MaxTokens   int       `json:"max_tokens" default:4096`
	Stream      bool      `json:"stream" default:false`
}

func (p *Proxy) HandleRequest(c *gin.Context) {
	modelGroup := c.MustGet("reqBody").(ChatReqBody).Model
	balancer, exists := p.balancers[modelGroup]
	reqBody := c.MustGet("reqBody").(ChatReqBody)
	logger := c.MustGet("logger").(*zap.Logger)
	logger.Debug("request body",
		zap.String("model", modelGroup),
		zap.Bool("do_sample", reqBody.DoSample),
		zap.Float64("temperature", reqBody.Temperature),
		zap.Float64("top_p", reqBody.TopP),
		zap.Int("max_tokens", reqBody.MaxTokens),
		zap.Bool("stream", reqBody.Stream),
		zap.Any("messages", reqBody.Messages),
	)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model group not found"})
		return
	}
	model := balancer.Next()
	if model == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No available models"})
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	// forward request
	req, err := http.NewRequest(http.MethodPost, model.BaseURL+c.Request.URL.Path, bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if model.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+model.APIKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward request"})
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

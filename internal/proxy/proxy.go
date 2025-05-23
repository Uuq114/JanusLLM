package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"

	"github.com/Uuq114/JanusLLM/internal/balancer"
	"github.com/Uuq114/JanusLLM/internal/models"
	"github.com/Uuq114/JanusLLM/internal/request"
	"github.com/Uuq114/JanusLLM/internal/spend"
)

var (
	SpendLogQueue = make(chan spend.SpendRecord, 100)
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

func (p *Proxy) HandleRequest(c *gin.Context) {
	// get model endpoint
	reqModel := c.MustGet("reqBody").(request.ChatReqBody).Model
	balancer, exists := p.balancers[reqModel]
	reqBody := c.MustGet("reqBody").(request.ChatReqBody)
	logger := c.MustGet("logger").(*zap.Logger)
	logger.Debug("request body",
		zap.String("model", reqModel),
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
	upstreamModel := balancer.Next()
	if upstreamModel == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No available models"})
		return
	}
	// replace model name with upstream model
	body := c.MustGet("reqBody").(request.ChatReqBody)
	body.Model = upstreamModel.Name
	c.Set("reqBody", body)
	byteBody, err := json.Marshal(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// forward request
	req, err := http.NewRequest(http.MethodPost, upstreamModel.BaseURL+c.Request.URL.Path, bytes.NewBuffer(byteBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Info("req body raw", zap.ByteString("body", byteBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if upstreamModel.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+upstreamModel.APIKey)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// prepare spend log info
	c.Set("upstreamResp", respBody)
	//go spend.CreateSpendRecord(c, SpendLogQueue)
	// copy response
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Data(resp.StatusCode, "application/json", respBody)
}

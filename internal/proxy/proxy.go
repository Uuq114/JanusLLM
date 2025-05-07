package proxy

import (
	"bytes"
	"io"
	"net/http"

	"github.com/Uuq114/JanusLLM/internal/balancer"
	"github.com/Uuq114/JanusLLM/internal/models"
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
	if group.Strategy == "weighted" {
		b = balancer.NewWeightedBalancer()
	} else {
		b = balancer.NewRoundRobinBalancer()
	}

	for _, model := range group.Models {
		b.AddModel(&model)
	}
	p.balancers[group.Name] = b
}

func (p *Proxy) HandleRequest(c *gin.Context) {
	modelGroup := c.Param("model_group")
	balancer, exists := p.balancers[modelGroup]
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
	req, err := http.NewRequest(c.Request.Method, model.BaseURL+c.Request.URL.Path, bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Authorization", "Bearer "+model.APIKey)
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

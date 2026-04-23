package proxy

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Uuq114/JanusLLM/internal/models"
)

type ProviderAdapter interface {
	BuildRequest(c *gin.Context, endpointPath string, upstreamModel *models.ModelConfig, rawBody []byte) (*http.Request, error)
}

type OpenAIAdapter struct{}

type AnthropicAdapter struct{}

func (a *OpenAIAdapter) BuildRequest(c *gin.Context, endpointPath string, upstreamModel *models.ModelConfig, rawBody []byte) (*http.Request, error) {
	method := c.Request.Method
	var bodyReader *bytes.Buffer
	if method == http.MethodGet {
		bodyReader = bytes.NewBuffer(nil)
	} else {
		bodyReader = bytes.NewBuffer(rawBody)
	}

	req, err := http.NewRequest(method, buildUpstreamURL(upstreamModel.BaseURL, endpointPath), bodyReader)
	if err != nil {
		return nil, err
	}

	copyHeaders(req, c)
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	if hasUpstreamAPIKey(upstreamModel.APIKey) {
		req.Header.Set("Authorization", "Bearer "+upstreamModel.APIKey)
	} else {
		req.Header.Del("Authorization")
	}
	return req, nil
}

func (a *AnthropicAdapter) BuildRequest(c *gin.Context, endpointPath string, upstreamModel *models.ModelConfig, rawBody []byte) (*http.Request, error) {
	method := c.Request.Method
	var bodyReader *bytes.Buffer
	if method == http.MethodGet {
		bodyReader = bytes.NewBuffer(nil)
	} else {
		bodyReader = bytes.NewBuffer(rawBody)
	}

	req, err := http.NewRequest(method, buildUpstreamURL(upstreamModel.BaseURL, endpointPath), bodyReader)
	if err != nil {
		return nil, err
	}

	copyHeaders(req, c)
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	if hasUpstreamAPIKey(upstreamModel.APIKey) {
		req.Header.Set("x-api-key", upstreamModel.APIKey)
		req.Header.Del("Authorization")
	} else {
		req.Header.Del("Authorization")
	}
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	return req, nil
}

func SelectAdapter(upstreamModel *models.ModelConfig) ProviderAdapter {
	modelType := strings.ToLower(strings.TrimSpace(upstreamModel.Type))
	switch modelType {
	case "anthropic", "claude":
		return &AnthropicAdapter{}
	default:
		return &OpenAIAdapter{}
	}
}

func copyHeaders(req *http.Request, c *gin.Context) {
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Accept", c.Request.Header.Get("Accept"))
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
}

func buildUpstreamURL(baseURL string, endpointPath string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path := "/" + strings.TrimLeft(endpointPath, "/")
	if strings.HasSuffix(base, "/v1") && strings.HasPrefix(path, "/v1/") {
		path = strings.TrimPrefix(path, "/v1")
	}
	return base + path
}

func hasUpstreamAPIKey(apiKey string) bool {
	apiKey = strings.TrimSpace(apiKey)
	return apiKey != "" && !strings.EqualFold(apiKey, "none")
}

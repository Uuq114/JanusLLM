package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Uuq114/JanusLLM/internal/models"
	"github.com/Uuq114/JanusLLM/internal/spend"
)

type ProviderAdapter interface {
	BuildRequest(c *gin.Context, endpointPath string, upstreamModel *models.ModelConfig, rawBody []byte) (*http.Request, error)
	BuildSpendPayload(respBody []byte) ([]byte, error)
	ParseSpendStreamLine(line []byte, requestID *string, usage **spend.TokenUsage)
}

type OpenAIAdapter struct{}

type AnthropicAdapter struct{}

type usagePayload struct {
	PromptTokens     *int `json:"prompt_tokens"`
	CompletionTokens *int `json:"completion_tokens"`
	TotalTokens      *int `json:"total_tokens"`
	InputTokens      *int `json:"input_tokens"`
	OutputTokens     *int `json:"output_tokens"`
}

type spendEnvelope struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Usage   *usagePayload `json:"usage"`
	Message *struct {
		ID    string        `json:"id"`
		Usage *usagePayload `json:"usage"`
	} `json:"message"`
}

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

func (a *OpenAIAdapter) BuildSpendPayload(respBody []byte) ([]byte, error) {
	return buildSpendPayloadFromEnvelope(respBody)
}

func (a *OpenAIAdapter) ParseSpendStreamLine(line []byte, requestID *string, usage **spend.TokenUsage) {
	parseSpendStreamLine(line, requestID, usage, false)
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

func (a *AnthropicAdapter) BuildSpendPayload(respBody []byte) ([]byte, error) {
	return buildSpendPayloadFromEnvelope(respBody)
}

func (a *AnthropicAdapter) ParseSpendStreamLine(line []byte, requestID *string, usage **spend.TokenUsage) {
	parseSpendStreamLine(line, requestID, usage, true)
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

func buildSpendPayloadFromEnvelope(respBody []byte) ([]byte, error) {
	var envelope spendEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, err
	}

	id := envelope.ID
	sourceUsage := envelope.Usage
	if envelope.Message != nil {
		if envelope.Message.ID != "" {
			id = envelope.Message.ID
		}
		if envelope.Message.Usage != nil {
			sourceUsage = envelope.Message.Usage
		}
	}
	if sourceUsage == nil {
		return nil, nil
	}

	payload, err := json.Marshal(spend.UpstreamResp{
		Id:    id,
		Usage: normalizedTokenUsage(sourceUsage, nil),
	})
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func parseSpendStreamLine(line []byte, requestID *string, usage **spend.TokenUsage, anthropic bool) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" || !strings.HasPrefix(trimmed, "data:") {
		return
	}

	data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	if data == "" || data == "[DONE]" {
		return
	}

	var envelope spendEnvelope
	if err := json.Unmarshal([]byte(data), &envelope); err != nil {
		return
	}

	if envelope.Message != nil && envelope.Message.ID != "" {
		*requestID = envelope.Message.ID
	}
	if envelope.ID != "" {
		*requestID = envelope.ID
	}

	if anthropic {
		switch envelope.Type {
		case "message_start":
			if envelope.Message != nil {
				mergeUsage(usage, envelope.Message.Usage)
			}
			return
		case "message_delta":
			mergeUsage(usage, envelope.Usage)
			return
		}
	}

	mergeUsage(usage, envelope.Usage)
}

func mergeUsage(target **spend.TokenUsage, source *usagePayload) {
	if source == nil {
		return
	}
	normalized := normalizedTokenUsage(source, currentUsage(target))
	*target = &normalized
}

func currentUsage(usage **spend.TokenUsage) *spend.TokenUsage {
	if usage == nil {
		return nil
	}
	return *usage
}

func normalizedTokenUsage(source *usagePayload, base *spend.TokenUsage) spend.TokenUsage {
	var out spend.TokenUsage
	if base != nil {
		out = *base
	}
	if source == nil {
		return out
	}

	if source.PromptTokens != nil {
		out.PromptTokens = *source.PromptTokens
	}
	if source.InputTokens != nil {
		out.PromptTokens = *source.InputTokens
	}
	if source.CompletionTokens != nil {
		out.CompletionTokens = *source.CompletionTokens
	}
	if source.OutputTokens != nil {
		out.CompletionTokens = *source.OutputTokens
	}
	if source.TotalTokens != nil {
		out.TotalTokens = *source.TotalTokens
	}
	if source.TotalTokens == nil && (source.PromptTokens != nil || source.InputTokens != nil || source.CompletionTokens != nil || source.OutputTokens != nil) {
		out.TotalTokens = out.PromptTokens + out.CompletionTokens
	}
	return out
}

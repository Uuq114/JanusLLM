package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerSwaggerRoutes(r *gin.Engine) {
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/")
	})
	r.GET("/swagger/", handleSwaggerUI)
	r.GET("/swagger/openapi.json", handleOpenAPISpec)
}

func handleSwaggerUI(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, swaggerHTML)
}

func handleOpenAPISpec(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"openapi": "3.0.3",
		"info": gin.H{
			"title":       "JanusLLM API",
			"description": "OpenAI/Anthropic-compatible LLM gateway API.",
			"version":     "0.1.0",
		},
		"servers": []gin.H{
			{"url": "/"},
		},
		"components": gin.H{
			"securitySchemes": gin.H{
				"bearerAuth": gin.H{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "API Key",
				},
			},
			"schemas": gin.H{
				"ErrorResponse": gin.H{
					"type": "object",
					"properties": gin.H{
						"error": gin.H{"type": "string"},
					},
				},
				"Message": gin.H{
					"type":     "object",
					"required": []string{"role", "content"},
					"properties": gin.H{
						"role":    gin.H{"type": "string", "example": "user"},
						"content": gin.H{"type": "string", "example": "Hello"},
					},
				},
				"ChatCompletionRequest": gin.H{
					"type":                 "object",
					"required":             []string{"model", "messages"},
					"additionalProperties": true,
					"properties": gin.H{
						"model": gin.H{
							"type":        "string",
							"description": "Gateway model group name.",
							"example":     "qwen3.5-27B",
						},
						"messages": gin.H{
							"type":  "array",
							"items": gin.H{"$ref": "#/components/schemas/Message"},
						},
						"temperature": gin.H{"type": "number", "example": 0.7},
						"max_tokens":  gin.H{"type": "integer", "example": 4096},
						"stream":      gin.H{"type": "boolean", "example": false},
					},
				},
				"NativeModelRequest": gin.H{
					"type":                 "object",
					"required":             []string{"model"},
					"additionalProperties": true,
					"properties": gin.H{
						"model": gin.H{
							"type":        "string",
							"description": "Gateway model group name.",
							"example":     "qwen3.5-27B",
						},
					},
				},
				"ModelsResponse": gin.H{
					"type": "object",
					"properties": gin.H{
						"object": gin.H{"type": "string", "example": "list"},
						"data": gin.H{
							"type": "array",
							"items": gin.H{
								"type": "object",
								"properties": gin.H{
									"id":       gin.H{"type": "string", "example": "qwen3.5-27B"},
									"object":   gin.H{"type": "string", "example": "model"},
									"owned_by": gin.H{"type": "string", "example": "janusllm"},
								},
							},
						},
					},
				},
			},
		},
		"paths": gin.H{
			"/ping": gin.H{
				"get": gin.H{
					"summary": "Health check",
					"responses": gin.H{
						"200": jsonResponse("Service is alive", gin.H{
							"type":       "object",
							"properties": gin.H{"message": gin.H{"type": "string", "example": "pong"}},
						}),
					},
				},
			},
			"/v1/models": gin.H{
				"get": gin.H{
					"summary":  "List models accessible by the current API key",
					"security": []gin.H{{"bearerAuth": []string{}}},
					"responses": gin.H{
						"200": jsonResponse("Accessible model list", gin.H{"$ref": "#/components/schemas/ModelsResponse"}),
						"401": errorResponse("Unauthorized"),
					},
				},
			},
			"/v1/chat/completions": nativeProxyPath("Chat completions", "#/components/schemas/ChatCompletionRequest"),
			"/v1/completions":      nativeProxyPath("Text completions", "#/components/schemas/NativeModelRequest"),
			"/v1/embeddings":       nativeProxyPath("Embeddings", "#/components/schemas/NativeModelRequest"),
			"/v1/messages":         nativeProxyPath("Anthropic messages", "#/components/schemas/NativeModelRequest"),
		},
	})
}

func nativeProxyPath(summary string, schemaRef string) gin.H {
	return gin.H{
		"post": gin.H{
			"summary":  summary,
			"security": []gin.H{{"bearerAuth": []string{}}},
			"requestBody": gin.H{
				"required": true,
				"content": gin.H{
					"application/json": gin.H{
						"schema": gin.H{"$ref": schemaRef},
					},
				},
			},
			"responses": gin.H{
				"200": jsonResponse("Upstream provider response", gin.H{
					"type":                 "object",
					"additionalProperties": true,
				}),
				"400": errorResponse("Bad request"),
				"401": errorResponse("Unauthorized"),
				"429": errorResponse("Rate limited"),
				"502": errorResponse("Upstream failed"),
			},
		},
	}
}

func jsonResponse(description string, schema gin.H) gin.H {
	return gin.H{
		"description": description,
		"content": gin.H{
			"application/json": gin.H{
				"schema": schema,
			},
		},
	}
}

func errorResponse(description string) gin.H {
	return jsonResponse(description, gin.H{"$ref": "#/components/schemas/ErrorResponse"})
}

const swaggerHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>JanusLLM API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #f7f7f7; }
    .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: "/swagger/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        persistAuthorization: true
      });
    };
  </script>
</body>
</html>`

package main

import (
	"net/http"
	"strings"

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
		"tags": []gin.H{
			{"name": "LLM API", "description": "OpenAI/Anthropic compatible gateway endpoints."},
			{"name": "Admin Organizations", "description": "Management API for organizations."},
			{"name": "Admin Users", "description": "Management API for gateway API users."},
			{"name": "Admin Keys", "description": "Management API for API keys."},
		},
		"components": gin.H{
			"securitySchemes": gin.H{
				"bearerAuth": gin.H{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "API Key",
				},
				"basicAuth": gin.H{
					"type":   "http",
					"scheme": "basic",
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
				"Organization": gin.H{
					"type": "object",
					"properties": gin.H{
						"organization_id":   gin.H{"type": "integer", "example": 1},
						"organization_name": gin.H{"type": "string", "example": "default-org"},
					},
				},
				"OrganizationRequest": gin.H{
					"type":     "object",
					"required": []string{"organization_name"},
					"properties": gin.H{
						"organization_name": gin.H{"type": "string", "example": "default-org"},
					},
				},
				"User": gin.H{
					"type": "object",
					"properties": gin.H{
						"user_id":         gin.H{"type": "integer", "example": 1},
						"user_name":       gin.H{"type": "string", "example": "platform-user"},
						"organization_id": gin.H{"type": "integer", "example": 1},
					},
				},
				"UserRequest": gin.H{
					"type":     "object",
					"required": []string{"user_name", "organization_id"},
					"properties": gin.H{
						"user_name":       gin.H{"type": "string", "example": "platform-user"},
						"organization_id": gin.H{"type": "integer", "example": 1},
					},
				},
				"Key": gin.H{
					"type": "object",
					"properties": gin.H{
						"key_id":               gin.H{"type": "integer", "example": 1},
						"key_content":          gin.H{"type": "string", "example": "sk-..."},
						"key_name":             gin.H{"type": "string", "example": "demo-key"},
						"model_list":           gin.H{"type": "array", "items": gin.H{"type": "string"}, "example": []string{"*"}},
						"user_id":              gin.H{"type": "integer", "example": 1},
						"organization_id":      gin.H{"type": "integer", "example": 1},
						"balance":              gin.H{"type": "number", "example": 100},
						"total_spend":          gin.H{"type": "number", "example": 0},
						"request_per_minute":   gin.H{"type": "integer", "example": 60},
						"spend_limit_per_week": gin.H{"type": "number", "example": 0},
						"expire_time":          gin.H{"type": "string", "format": "date-time", "nullable": true},
					},
				},
				"KeyRequest": gin.H{
					"type":     "object",
					"required": []string{"key_name", "user_id", "organization_id"},
					"properties": gin.H{
						"key_content":          gin.H{"type": "string", "description": "Optional. Server generates one when omitted."},
						"key_name":             gin.H{"type": "string", "example": "demo-key"},
						"all_models":           gin.H{"type": "boolean", "description": "When true, grants all models and stores model_list as [\"*\"].", "example": true},
						"model_list":           gin.H{"type": "array", "items": gin.H{"type": "string"}, "description": "Use [\"*\"] or all_models=true to grant all models.", "example": []string{"*"}},
						"user_id":              gin.H{"type": "integer", "example": 1},
						"organization_id":      gin.H{"type": "integer", "example": 1},
						"balance":              gin.H{"type": "number", "example": 100},
						"request_per_minute":   gin.H{"type": "integer", "example": 60},
						"spend_limit_per_week": gin.H{"type": "number", "example": 0},
						"expire_time":          gin.H{"type": "string", "format": "date-time"},
					},
				},
			},
		},
		"paths": gin.H{
			"/ping": gin.H{
				"get": gin.H{
					"summary": "Health check",
					"tags":    []string{"LLM API"},
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
					"tags":     []string{"LLM API"},
					"security": []gin.H{{"bearerAuth": []string{}}},
					"responses": gin.H{
						"200": jsonResponse("Accessible model list", gin.H{"$ref": "#/components/schemas/ModelsResponse"}),
						"401": errorResponse("Unauthorized"),
					},
				},
			},
			"/v1/chat/completions":                      nativeProxyPath("Chat completions", "#/components/schemas/ChatCompletionRequest"),
			"/v1/completions":                           nativeProxyPath("Text completions", "#/components/schemas/NativeModelRequest"),
			"/v1/embeddings":                            nativeProxyPath("Embeddings", "#/components/schemas/NativeModelRequest"),
			"/v1/messages":                              nativeProxyPath("Anthropic messages", "#/components/schemas/NativeModelRequest"),
			"/v1/admin/organizations":                   adminCollectionPath("Admin Organizations", "Organizations", "#/components/schemas/Organization", "#/components/schemas/OrganizationRequest"),
			"/v1/admin/organizations/{organization_id}": adminItemPath("Admin Organizations", "Organization", "organization_id", "#/components/schemas/Organization", "#/components/schemas/OrganizationRequest"),
			"/v1/admin/users":                           adminCollectionPath("Admin Users", "Users", "#/components/schemas/User", "#/components/schemas/UserRequest"),
			"/v1/admin/users/{user_id}":                 adminItemPath("Admin Users", "User", "user_id", "#/components/schemas/User", "#/components/schemas/UserRequest"),
			"/v1/admin/keys":                            adminCollectionPath("Admin Keys", "Keys", "#/components/schemas/Key", "#/components/schemas/KeyRequest"),
			"/v1/admin/keys/{key_id}":                   adminItemPath("Admin Keys", "Key", "key_id", "#/components/schemas/Key", "#/components/schemas/KeyRequest"),
		},
	})
}

func nativeProxyPath(summary string, schemaRef string) gin.H {
	return gin.H{
		"post": gin.H{
			"summary":  summary,
			"tags":     []string{"LLM API"},
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

func adminCollectionPath(tag string, name string, responseSchemaRef string, requestSchemaRef string) gin.H {
	return gin.H{
		"get": gin.H{
			"summary":  "List " + strings.ToLower(name),
			"tags":     []string{tag},
			"security": []gin.H{{"basicAuth": []string{}}},
			"responses": gin.H{
				"200": jsonResponse(name+" list", gin.H{
					"type": "object",
					"properties": gin.H{
						"data": gin.H{
							"type":  "array",
							"items": gin.H{"$ref": responseSchemaRef},
						},
					},
				}),
				"401": errorResponse("Unauthorized"),
			},
		},
		"post": gin.H{
			"summary":  "Create " + strings.ToLower(strings.TrimSuffix(name, "s")),
			"tags":     []string{tag},
			"security": []gin.H{{"basicAuth": []string{}}},
			"requestBody": gin.H{
				"required": true,
				"content": gin.H{
					"application/json": gin.H{
						"schema": gin.H{"$ref": requestSchemaRef},
					},
				},
			},
			"responses": gin.H{
				"201": jsonResponse("Created", gin.H{"$ref": responseSchemaRef}),
				"400": errorResponse("Bad request"),
				"401": errorResponse("Unauthorized"),
				"409": errorResponse("Conflict"),
			},
		},
	}
}

func adminItemPath(tag string, name string, paramName string, responseSchemaRef string, requestSchemaRef string) gin.H {
	param := gin.H{
		"name":     paramName,
		"in":       "path",
		"required": true,
		"schema":   gin.H{"type": "integer"},
	}
	return gin.H{
		"get": gin.H{
			"summary":    "Get " + strings.ToLower(name),
			"tags":       []string{tag},
			"security":   []gin.H{{"basicAuth": []string{}}},
			"parameters": []gin.H{param},
			"responses": gin.H{
				"200": jsonResponse(name, gin.H{"$ref": responseSchemaRef}),
				"401": errorResponse("Unauthorized"),
				"404": errorResponse("Not found"),
			},
		},
		"patch": gin.H{
			"summary":    "Update " + strings.ToLower(name),
			"tags":       []string{tag},
			"security":   []gin.H{{"basicAuth": []string{}}},
			"parameters": []gin.H{param},
			"requestBody": gin.H{
				"required": true,
				"content": gin.H{
					"application/json": gin.H{
						"schema": gin.H{"$ref": requestSchemaRef},
					},
				},
			},
			"responses": gin.H{
				"200": jsonResponse(name, gin.H{"$ref": responseSchemaRef}),
				"400": errorResponse("Bad request"),
				"401": errorResponse("Unauthorized"),
				"404": errorResponse("Not found"),
				"409": errorResponse("Conflict"),
			},
		},
		"delete": gin.H{
			"summary":    "Delete " + strings.ToLower(name),
			"tags":       []string{tag},
			"security":   []gin.H{{"basicAuth": []string{}}},
			"parameters": []gin.H{param},
			"responses": gin.H{
				"204": gin.H{"description": "Deleted"},
				"401": errorResponse("Unauthorized"),
				"404": errorResponse("Not found"),
				"409": errorResponse("Conflict"),
			},
		},
	}
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

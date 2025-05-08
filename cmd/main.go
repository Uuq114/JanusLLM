package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Uuq114/JanusLLM/internal/models"
	"github.com/Uuq114/JanusLLM/internal/proxy"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	config, err := loadJanusConfig("../config/config.yaml")
	if err != nil {
		logger.Error("Failed to load config", zap.Error(err))
	}

	p := proxy.NewProxy()
	for _, group := range config.ModelGroups {
		p.RegisterModelGroup(&group)
		logger.Info("Registered model group", zap.String("name", group.Name))
	}

	r := gin.Default()
	r.Use(logReqHeadersMiddleware(logger))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	api := r.Group("/v1")
	{
		api.POST("/chat/completions", p.HandleRequest)
		//api.POST("/completions", p.HandleRequest)
		//api.POST("/embeddings", p.HandleRequest)
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

type JanusConfig struct {
	ModelGroups []models.ModelGroup `yaml:"model_groups"`
}

func loadJanusConfig(path string) (*JanusConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config JanusConfig
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func logReqHeadersMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("logger", logger)
		// req headers
		headers := map[string]string{
			"User-Agent":   c.Request.UserAgent(),
			"X-Request-ID": c.Request.Header.Get("X-Request-ID"),
		}
		// req body
		var reqBody proxy.ChatReqBody
		if err := c.ShouldBindJSON(&reqBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Set("reqBody", reqBody)

		logger.Info("Request Headers",
			zap.String("method", c.Request.Method),
			zap.String("url", c.Request.URL.String()),
			zap.Any("headers", headers),
			zap.String("model", reqBody.Model),
		)
		c.Next()
	}
}

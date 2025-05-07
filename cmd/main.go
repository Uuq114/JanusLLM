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
	config, err := loadJanusConfig("config/config.yaml")
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	if err != nil {
		logger.Error("Failed to load janus config", zap.Error(err))
	}

	p := proxy.NewProxy()

	for _, group := range config.ModelGroups {
		p.RegisterModelGroup(&group)
	}

	// 设置路由
	r := gin.Default()

	// 健康检查
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// API路由
	api := r.Group("/v1")
	{
		// OpenAI兼容API
		api.POST("/chat/completions", p.HandleRequest)
		api.POST("/completions", p.HandleRequest)
		api.POST("/embeddings", p.HandleRequest)
	}

	// 启动服务器
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

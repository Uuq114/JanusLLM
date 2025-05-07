package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Uuq114/JanusLLM/internal/models"
	"github.com/Uuq114/JanusLLM/internal/proxy"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func main() {
	// 加载配置
	config, err := loadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建代理
	p := proxy.NewProxy()

	// 注册模型组
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

// Config 配置结构
type Config struct {
	ModelGroups []models.ModelGroup `yaml:"model_groups"`
}

// loadConfig 加载配置文件
func loadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

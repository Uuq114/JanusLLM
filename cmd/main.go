package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Uuq114/JanusLLM/internal/request"
	"github.com/creasty/defaults"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/Uuq114/JanusLLM/internal/auth"
	"github.com/Uuq114/JanusLLM/internal/db"
	"github.com/Uuq114/JanusLLM/internal/models"
	"github.com/Uuq114/JanusLLM/internal/proxy"
	"github.com/Uuq114/JanusLLM/internal/spend"
)

var (
	validKeys = make(map[string]auth.Key)
	mutex     sync.RWMutex
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	config, err := loadJanusConfig("../config/config.yaml")
	if err != nil {
		logger.Error("Failed to load config", zap.Error(err))
	}

	// setup proxy
	p := proxy.NewProxy()
	for _, group := range config.ModelGroups {
		p.RegisterModelGroup(&group)
		logger.Info("Registered model group", zap.String("name", group.Name))
	}

	// init
	r := gin.Default()

	go startBackgroundTasks(logger)

	r.Use(logReqHeadersMiddleware(logger))
	r.Use(checkKeyMiddleware(logger))
	r.Use(logSpendMiddleware(logger))

	// routers
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

	// run server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// structs

type JanusConfig struct {
	ModelGroups []models.ModelGroup `yaml:"model_groups"`
	DatabaseUrl string              `yaml:"database_url"`
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
	// update db global var
	janusDb.MysqlDsn = config.DatabaseUrl
	// update model price
	for _, group := range config.ModelGroups {
		spend.ModelPrice[group.Name] = []float64{group.CostPerInputToken, group.CostPerOutputToken}
	}
	return &config, nil
}

// middlewares

func logReqHeadersMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("logger", logger)
		// req headers
		headers := map[string]string{
			"User-Agent":   c.Request.UserAgent(),
			"X-Request-ID": c.Request.Header.Get("X-Request-ID"),
		}
		// req body
		var reqBody request.ChatReqBody
		if err := c.ShouldBindJSON(&reqBody); err != nil {
			logger.Error("Failed to bind request body", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := defaults.Set(&reqBody); err != nil {
			logger.Error("Failed to set default values", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Set("reqHeader", headers)
		c.Set("reqBody", reqBody)
		c.Set("modelGroup", reqBody.Model)
		logger.Info("Request Headers",
			zap.String("method", c.Request.Method),
			zap.String("url", c.Request.URL.String()),
			zap.Any("headers", headers),
			zap.String("model", reqBody.Model),
		)
		c.Next()
	}
}

func checkKeyMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Request.Header.Get("Authorization")
		if key == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header"})
			c.Abort()
			return
		}
		key = strings.TrimPrefix(key, "Bearer ")
		mutex.RLock()
		defer mutex.RUnlock()
		if _, ok := validKeys[key]; !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization key"})
			c.Abort()
			return
		}
		model := c.MustGet("reqBody").(request.ChatReqBody).Model
		if !isValidModel(model, validKeys[key].ModelList) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid request model"})
			c.Abort()
			return
		}
		c.Set("key", validKeys[key])
		logger.Info("key info",
			zap.String("key name", validKeys[key].KeyName),
			zap.Int("user id", validKeys[key].UserId),
			zap.Int("organization id", validKeys[key].OrganizationId),
		)
		c.Next()
	}
}

func logSpendMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info("call log spend middleware")
		c.Next()
		spend.CreateSpendRecord(c, proxy.SpendLogQueue)
		logger.Info("add request spend log to queue")
	}
}

func isValidModel(reqModel string, modelList auth.StringSlice) bool {
	if modelList[0] == "*" {
		return true
	}
	for _, model := range modelList {
		if model == reqModel {
			return true
		}
	}
	return false
}

// background tasks

func startBackgroundTasks(logger *zap.Logger) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	logger.Info("update key info")
	updateKeyInfo()

	for {
		select {
		case <-ticker.C:
			logger.Info("Performing background task")
			go updateKeyInfo()
			// todo: add runtime config update
			go FlushSpendLog(logger, proxy.SpendLogQueue)
		}
	}
}

func updateKeyInfo() {
	keys := auth.GetAllValidKey()
	mutex.Lock()
	defer mutex.Unlock()
	validKeys = make(map[string]auth.Key)
	for _, key := range keys {
		validKeys[key.KeyContent] = key
	}
	log.Println("Updated valid keys:", validKeys)
}

func FlushSpendLog(logger *zap.Logger, ch <-chan spend.SpendRecord) {
	var batch []spend.SpendRecord
	for {
		select {
		case logRecord, ok := <-ch:
			if !ok {
				if len(batch) > 0 {
					spend.InsertBatchSpendRecord(batch)
					batch = nil
					logger.Info("Flushed spend log records to database", zap.Int("batch size", len(batch)))
				}
				return
			}
			batch = append(batch, logRecord)
		default:
			if len(batch) > 0 {
				spend.InsertBatchSpendRecord(batch)
				batch = nil
				logger.Info("Flushed spend log records to database", zap.Int("batch size", len(batch)))
			}
			return
		}
	}
}

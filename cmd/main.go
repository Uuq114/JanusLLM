package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/Uuq114/JanusLLM/internal/auth"
	janusDb "github.com/Uuq114/JanusLLM/internal/db"
	"github.com/Uuq114/JanusLLM/internal/models"
	"github.com/Uuq114/JanusLLM/internal/proxy"
	"github.com/Uuq114/JanusLLM/internal/request"
	"github.com/Uuq114/JanusLLM/internal/spend"
)

var (
	validKeys = make(map[string]cachedKey)
	mutex     sync.RWMutex

	modelGroupSet = make(map[string]struct{})
)

const (
	keyCacheSyncTTL = 3 * time.Minute
	keyCacheIdleTTL = 30 * time.Minute
)

type cachedKey struct {
	Key          auth.Key
	LastSyncAt   time.Time
	LastAccessAt time.Time
}

func main() {
	config, err := loadJanusConfig("../config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	logger := buildLogger(config.Service.LogLevel)
	defer logger.Sync()

	p := proxy.NewProxy()
	for _, group := range config.Models.ModelGroups {
		p.RegisterModelGroup(&group)
		modelGroupSet[group.Name] = struct{}{}
		logger.Info("Registered model group", zap.String("name", group.Name))
	}

	r := gin.Default()
	go startBackgroundTasks(logger)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	registerSwaggerRoutes(r)
	registerAdminRoutes(r, config.Admin, logger)

	api := r.Group("/v1")
	api.Use(logReqHeadersMiddleware(logger))
	api.Use(checkKeyMiddleware(logger))
	api.Use(logSpendMiddleware(logger))
	{
		api.POST("/chat/completions", p.HandleRequest)
		api.POST("/completions", p.HandleRequest)
		api.POST("/embeddings", p.HandleRequest)
		api.POST("/messages", p.HandleRequest)
		api.GET("/models", p.HandleListModels)
	}

	if err := r.Run(":" + strconv.Itoa(config.Service.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

type ServiceConfig struct {
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`
}

type ModelsConfig struct {
	ModelGroups []models.ModelGroup `yaml:"model_groups"`
}

type SecretsConfig struct {
	DatabaseURL string `yaml:"database_url"`
}

type AdminConfig struct {
	MasterKey string `yaml:"master_key"`
}

type JanusConfig struct {
	Service ServiceConfig `yaml:"service"`
	Models  ModelsConfig  `yaml:"models"`
	Secrets SecretsConfig `yaml:"secrets"`
	Admin   AdminConfig   `yaml:"admin"`

	LegacyModelGroups []models.ModelGroup `yaml:"model_groups"`
	LegacyDatabaseURL string              `yaml:"database_url"`
}

func loadJanusConfig(path string) (*JanusConfig, error) {
	paths := []string{path, "config/config.yaml"}
	var file []byte
	var err error

	for _, configPath := range paths {
		file, err = os.ReadFile(configPath)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	var config JanusConfig
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	if len(config.Models.ModelGroups) == 0 && len(config.LegacyModelGroups) > 0 {
		config.Models.ModelGroups = config.LegacyModelGroups
	}
	if config.Secrets.DatabaseURL == "" && config.LegacyDatabaseURL != "" {
		config.Secrets.DatabaseURL = config.LegacyDatabaseURL
	}

	if config.Service.Port == 0 {
		config.Service.Port = 8080
	}
	if config.Service.LogLevel == "" {
		config.Service.LogLevel = "info"
	}

	if dbURL := strings.TrimSpace(os.Getenv("JANUS_DATABASE_URL")); dbURL != "" {
		config.Secrets.DatabaseURL = dbURL
	}
	if strings.TrimSpace(config.Secrets.DatabaseURL) == "" {
		return nil, errors.New("database_url is empty; set secrets.database_url or JANUS_DATABASE_URL")
	}
	if masterKey := strings.TrimSpace(os.Getenv("JANUS_ADMIN_MASTER_KEY")); masterKey != "" {
		config.Admin.MasterKey = masterKey
	}
	if strings.TrimSpace(config.Admin.MasterKey) == "" {
		return nil, errors.New("admin.master_key is empty; set admin.master_key or JANUS_ADMIN_MASTER_KEY")
	}

	janusDb.DatabaseDsn = config.Secrets.DatabaseURL
	for _, group := range config.Models.ModelGroups {
		spend.ModelPrice[group.Name] = []float64{group.CostPerInputToken, group.CostPerOutputToken}
	}
	return &config, nil
}

func buildLogger(level string) *zap.Logger {
	cfg := zap.NewProductionConfig()
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "warn", "warning":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	logger, err := cfg.Build()
	if err != nil {
		fallback, _ := zap.NewProduction()
		return fallback
	}
	return logger
}

func logReqHeadersMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("logger", logger)
		c.Set("requestStart", time.Now())

		headers := map[string]string{
			"User-Agent":   c.Request.UserAgent(),
			"X-Request-ID": c.Request.Header.Get("X-Request-ID"),
		}
		c.Set("reqHeader", headers)

		rawBody := []byte{}
		if c.Request.Method != http.MethodGet {
			body, err := ioReadAll(c)
			if err != nil {
				logger.Error("Failed to read request body", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				c.Abort()
				return
			}
			rawBody = body
		}
		c.Set("rawBody", rawBody)

		modelName, stream, err := request.ExtractModelAndStream(rawBody)
		if err != nil {
			logger.Error("Failed to parse request body", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			c.Abort()
			return
		}
		if modelName != "" {
			c.Set("modelGroup", modelName)
		}
		c.Set("isStreamRequest", stream)

		if requiresModel(c.Request.URL.Path) && modelName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
			c.Abort()
			return
		}

		logger.Info("Request accepted",
			zap.String("method", c.Request.Method),
			zap.String("url", c.Request.URL.String()),
			zap.Any("headers", headers),
			zap.String("model", modelName),
		)
		c.Next()
	}
}

func ioReadAll(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

func checkKeyMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyContent := c.Request.Header.Get("Authorization")
		if keyContent == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}
		keyContent = strings.TrimPrefix(keyContent, "Bearer ")

		keyInfo, ok, err := getValidKeyForRequest(keyContent, time.Now())
		if err != nil {
			logger.Error("Failed to validate key from database", zap.String("key", keyContent), zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "authorization check unavailable"})
			c.Abort()
			return
		}
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization key"})
			c.Abort()
			return
		}

		if keyInfo.RequestPerMinute > 0 {
			ring := proxy.GetOrCreateRequestRing(keyContent, keyInfo.RequestPerMinute)
			if ring != nil && !ring.Allow() {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "reach rate limit"})
				c.Abort()
				return
			}
		}

		c.Set("key", keyInfo)
		if c.Request.URL.Path == "/v1/models" {
			logger.Info("Key authorized",
				zap.String("key name", keyInfo.KeyName),
				zap.Int("user id", keyInfo.UserId),
				zap.Int("organization id", keyInfo.OrganizationId),
			)
			c.Next()
			return
		}

		modelGroup, err := resolveModelGroup(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		c.Set("modelGroup", modelGroup)

		if !isValidModel(modelGroup, keyInfo.ModelList) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid request model"})
			c.Abort()
			return
		}

		logger.Info("Key authorized",
			zap.String("key name", keyInfo.KeyName),
			zap.Int("user id", keyInfo.UserId),
			zap.Int("organization id", keyInfo.OrganizationId),
		)
		c.Next()
	}
}

func getValidKeyForRequest(keyContent string, now time.Time) (auth.Key, bool, error) {
	cached, ok := getCachedKey(keyContent)
	if ok {
		if !isKeyLocallyValid(cached.Key, now) {
			deleteCachedKey(keyContent)
			proxy.RemoveRequestRing(keyContent)
			return auth.Key{}, false, nil
		}
		if now.Sub(cached.LastSyncAt) <= keyCacheSyncTTL {
			touchCachedKey(keyContent, now)
			return cached.Key, true, nil
		}
	}

	loaded, err := auth.GetValidKeyByContent(keyContent)
	if err != nil {
		return auth.Key{}, false, err
	}
	if loaded == nil {
		deleteCachedKey(keyContent)
		proxy.RemoveRequestRing(keyContent)
		return auth.Key{}, false, nil
	}

	upsertCachedKey(*loaded, now, now)
	return *loaded, true, nil
}

func getCachedKey(keyContent string) (cachedKey, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	key, ok := validKeys[keyContent]
	return key, ok
}

func touchCachedKey(keyContent string, accessAt time.Time) {
	mutex.Lock()
	defer mutex.Unlock()
	if key, ok := validKeys[keyContent]; ok {
		key.LastAccessAt = accessAt
		validKeys[keyContent] = key
	}
}

func upsertCachedKey(key auth.Key, syncAt time.Time, accessAt time.Time) {
	if accessAt.IsZero() {
		accessAt = syncAt
	}
	mutex.Lock()
	defer mutex.Unlock()
	validKeys[key.KeyContent] = cachedKey{
		Key:          key,
		LastSyncAt:   syncAt,
		LastAccessAt: accessAt,
	}
}

func deleteCachedKey(keyContent string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(validKeys, keyContent)
}

func snapshotCachedKeys() map[string]cachedKey {
	mutex.RLock()
	defer mutex.RUnlock()
	out := make(map[string]cachedKey, len(validKeys))
	for key, value := range validKeys {
		out[key] = value
	}
	return out
}

func isKeyLocallyValid(key auth.Key, now time.Time) bool {
	if key.Balance <= 0 {
		return false
	}
	if !key.ExpireTime.IsZero() && !key.ExpireTime.After(now) {
		return false
	}
	return true
}

func resolveModelGroup(c *gin.Context) (string, error) {
	if modelValue, ok := c.Get("modelGroup"); ok {
		if modelName, ok := modelValue.(string); ok && modelName != "" {
			if _, exists := modelGroupSet[modelName]; !exists {
				return "", errors.New("model group not configured")
			}
			return modelName, nil
		}
	}
	return "", errors.New("model is required")
}

func logSpendMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Request.URL.Path == "/v1/models" {
			return
		}

		if _, ok := c.Get("upstreamResp"); !ok {
			return
		}
		key, ok := c.Get("key")
		if !ok {
			return
		}

		spend.CreateSpendRecord(c, proxy.SpendLogQueue)
		keyInfo := key.(auth.Key)
		keySpendQueue := proxy.GetOrCreateKeySpendQueue(keyInfo.KeyContent)
		spend.CreateKeySpendRecord(c, keySpendQueue)

		if start, ok := c.Get("requestStart"); ok {
			if startTime, ok := start.(time.Time); ok {
				logger.Info("Request spend queued", zap.Duration("latency", time.Since(startTime)))
			}
		}
	}
}

func requiresModel(path string) bool {
	switch path {
	case "/v1/chat/completions", "/v1/completions", "/v1/embeddings", "/v1/messages":
		return true
	default:
		return false
	}
}

func isValidModel(reqModel string, modelList auth.StringSlice) bool {
	if len(modelList) == 0 {
		return false
	}
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

func startBackgroundTasks(logger *zap.Logger) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	logger.Info("Starting background tasks")

	for range ticker.C {
		logger.Info("Performing background task")
		refreshCachedKeys(logger)
		go FlushSpendLog(logger, proxy.SpendLogQueue)
		go FlushKeySpend(logger, proxy.SnapshotKeySpendQueue())
	}
}

func refreshCachedKeys(logger *zap.Logger) {
	now := time.Now()
	cache := snapshotCachedKeys()
	for keyContent, keyInfo := range cache {
		if !keyInfo.LastAccessAt.IsZero() && now.Sub(keyInfo.LastAccessAt) > keyCacheIdleTTL {
			deleteCachedKey(keyContent)
			proxy.RemoveRequestRing(keyContent)
			logger.Info("Evicted idle key cache entry", zap.String("key", keyContent))
			continue
		}

		if now.Sub(keyInfo.LastSyncAt) <= keyCacheSyncTTL {
			continue
		}

		latest, err := auth.GetValidKeyByContent(keyContent)
		if err != nil {
			logger.Warn("Failed to refresh key cache entry", zap.String("key", keyContent), zap.Error(err))
			continue
		}
		if latest == nil {
			deleteCachedKey(keyContent)
			proxy.RemoveRequestRing(keyContent)
			logger.Info("Removed invalid key cache entry", zap.String("key", keyContent))
			continue
		}

		upsertCachedKey(*latest, now, keyInfo.LastAccessAt)
	}
}

func FlushSpendLog(logger *zap.Logger, ch <-chan spend.SpendRecord) {
	var batch []spend.SpendRecord
	for {
		select {
		case logRecord, ok := <-ch:
			if !ok {
				if len(batch) > 0 {
					spend.InsertBatchSpendRecord(batch)
					logger.Info("Flushed spend log records to database", zap.Int("batch size", len(batch)))
				}
				return
			}
			batch = append(batch, logRecord)
		default:
			if len(batch) > 0 {
				spend.InsertBatchSpendRecord(batch)
				logger.Info("Flushed spend log records to database", zap.Int("batch size", len(batch)))
			}
			return
		}
	}
}

func FlushKeySpend(logger *zap.Logger, queue map[string]chan float64) {
	for key, ch := range queue {
		totalSpend := 0.0
		for {
			select {
			case spd, ok := <-ch:
				if !ok {
					if totalSpend > 0 {
						spend.UpdateKeySpendRecord(totalSpend, key)
						logger.Info("Flushed key spend records to database",
							zap.String("key", key),
							zap.Float64("total spend", totalSpend),
						)
					}
					goto nextKey
				}
				totalSpend += spd
			default:
				if totalSpend > 0 {
					spend.UpdateKeySpendRecord(totalSpend, key)
					logger.Info("Flushed key spend records to database",
						zap.String("key", key),
						zap.Float64("total spend", totalSpend),
					)
				}
				goto nextKey
			}
		}
	nextKey:
	}
}

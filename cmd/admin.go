package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Uuq114/JanusLLM/internal/auth"
	janusDb "github.com/Uuq114/JanusLLM/internal/db"
	"github.com/Uuq114/JanusLLM/internal/proxy"
)

const adminMasterUser = "admin"

func syncMasterAdminUser(config AdminConfig, logger *zap.Logger) error {
	masterKey := strings.TrimSpace(config.MasterKey)
	if masterKey == "" {
		return errors.New("admin.master_key is empty")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(masterKey), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	db, err := janusDb.ConnectDatabase()
	if err != nil {
		return err
	}
	defer janusDb.CloseDatabaseConnection(db)

	user := adminUserDTO{
		Username:     adminMasterUser,
		PasswordHash: string(passwordHash),
		Enabled:      true,
	}

	var existing adminUserDTO
	err = db.Table("janus_admin_user").Where("username = ?", adminMasterUser).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Table("janus_admin_user").
			Omit("admin_user_id", "create_time", "update_time").
			Create(&user).Error; err != nil {
			return err
		}
		logger.Info("Created master admin user", zap.String("username", adminMasterUser))
		return nil
	}
	if err != nil {
		return err
	}

	if err := db.Table("janus_admin_user").
		Where("admin_user_id = ?", existing.AdminUserID).
		Updates(map[string]interface{}{
			"password_hash": user.PasswordHash,
			"enabled":       true,
		}).Error; err != nil {
		return err
	}
	logger.Info("Synced master admin user", zap.String("username", adminMasterUser))
	return nil
}

func registerAdminRoutes(r *gin.Engine, logger *zap.Logger) {
	admin := r.Group("/v1/admin")
	admin.Use(adminAuthMiddleware(logger))
	{
		admin.GET("/organizations", listOrganizations)
		admin.POST("/organizations", createOrganization)
		admin.GET("/organizations/:organization_id", getOrganization)
		admin.PATCH("/organizations/:organization_id", updateOrganization)
		admin.DELETE("/organizations/:organization_id", deleteOrganization)

		admin.GET("/users", listUsers)
		admin.POST("/users", createUser)
		admin.GET("/users/:user_id", getUser)
		admin.PATCH("/users/:user_id", updateUser)
		admin.DELETE("/users/:user_id", deleteUser)

		// Backward-compatible aliases. The persisted resource is janus_auth_user.
		admin.GET("/teams", listUsers)
		admin.POST("/teams", createUser)
		admin.GET("/teams/:team_id", getUser)
		admin.PATCH("/teams/:team_id", updateUser)
		admin.DELETE("/teams/:team_id", deleteUser)

		admin.GET("/keys", listKeys)
		admin.POST("/keys", createKey)
		admin.GET("/keys/:key_id", getKey)
		admin.PATCH("/keys/:key_id", updateKey)
		admin.DELETE("/keys/:key_id", deleteKey)
	}
}

func adminAuthMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			logger.Warn("admin auth failed", zap.String("username", username))
			c.Header("WWW-Authenticate", `Basic realm="JanusLLM Admin"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin credentials"})
			c.Abort()
			return
		}

		valid, err := validateAdminCredential(username, password)
		if err != nil {
			logger.Error("admin auth database check failed", zap.String("username", username), zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "admin auth unavailable"})
			c.Abort()
			return
		}
		if !valid {
			logger.Warn("admin auth failed", zap.String("username", username))
			c.Header("WWW-Authenticate", `Basic realm="JanusLLM Admin"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin credentials"})
			c.Abort()
			return
		}

		c.Set("adminUser", username)
		c.Next()
	}
}

func validateAdminCredential(username string, password string) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return false, nil
	}

	db, err := janusDb.ConnectDatabase()
	if err != nil {
		return false, err
	}
	defer janusDb.CloseDatabaseConnection(db)

	var user adminUserDTO
	err = db.Table("janus_admin_user").
		Where("username = ? AND enabled = TRUE", username).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return false, nil
	}
	return true, nil
}

type adminUserDTO struct {
	AdminUserID  int64     `gorm:"column:admin_user_id" json:"admin_user_id"`
	Username     string    `gorm:"column:username" json:"username"`
	PasswordHash string    `gorm:"column:password_hash" json:"-"`
	Enabled      bool      `gorm:"column:enabled" json:"enabled"`
	CreateTime   time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime   time.Time `gorm:"column:update_time" json:"update_time"`
}

type organizationDTO struct {
	OrganizationID   int64     `gorm:"primaryKey;autoIncrement;column:organization_id" json:"organization_id"`
	OrganizationName string    `gorm:"column:organization_name" json:"organization_name"`
	CreateTime       time.Time `gorm:"column:create_time" json:"-"`
	UpdateTime       time.Time `gorm:"column:update_time" json:"-"`
}

type organizationRequest struct {
	OrganizationName string `json:"organization_name" binding:"required"`
}

type organizationPatchRequest struct {
	OrganizationName *string `json:"organization_name"`
}

func listOrganizations(c *gin.Context) {
	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var organizations []organizationDTO
	if err := db.Table("janus_auth_organization").Order("organization_id").Find(&organizations).Error; err != nil {
		respondDBError(c, "list organizations failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": organizations})
}

func createOrganization(c *gin.Context) {
	var req organizationRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	organization := organizationDTO{OrganizationName: strings.TrimSpace(req.OrganizationName)}
	if organization.OrganizationName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization_name is required"})
		return
	}
	if err := db.Table("janus_auth_organization").
		Clauses(clause.Returning{}).
		Omit("organization_id", "create_time", "update_time").
		Create(&organization).Error; err != nil {
		respondDBError(c, "create organization failed", err)
		return
	}
	c.JSON(http.StatusCreated, organization)
}

func getOrganization(c *gin.Context) {
	id, ok := parseIDParam(c, "organization_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var organization organizationDTO
	if !firstByID(c, db.Table("janus_auth_organization").Where("organization_id = ?", id), &organization) {
		return
	}
	c.JSON(http.StatusOK, organization)
}

func updateOrganization(c *gin.Context) {
	id, ok := parseIDParam(c, "organization_id")
	if !ok {
		return
	}
	var req organizationPatchRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	updates := map[string]interface{}{}
	if req.OrganizationName != nil {
		name := strings.TrimSpace(*req.OrganizationName)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "organization_name cannot be empty"})
			return
		}
		updates["organization_name"] = name
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	result := db.Table("janus_auth_organization").Where("organization_id = ?", id).Updates(updates)
	if result.Error != nil {
		respondDBError(c, "update organization failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}
	getOrganization(c)
}

func deleteOrganization(c *gin.Context) {
	id, ok := parseIDParam(c, "organization_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	result := db.Table("janus_auth_organization").Where("organization_id = ?", id).Delete(&organizationDTO{})
	if result.Error != nil {
		respondDBError(c, "delete organization failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

type userDTO struct {
	UserID         int64     `gorm:"primaryKey;autoIncrement;column:user_id" json:"user_id"`
	UserName       string    `gorm:"column:user_name" json:"user_name"`
	OrganizationID int64     `gorm:"column:organization_id" json:"organization_id"`
	CreateTime     time.Time `gorm:"column:create_time" json:"-"`
	UpdateTime     time.Time `gorm:"column:update_time" json:"-"`
}

type userRequest struct {
	UserName       string `json:"user_name"`
	TeamName       string `json:"team_name"`
	OrganizationID int64  `json:"organization_id" binding:"required"`
}

func (r userRequest) resolvedName() string {
	if strings.TrimSpace(r.UserName) != "" {
		return r.UserName
	}
	return r.TeamName
}

type userPatchRequest struct {
	UserName       *string `json:"user_name"`
	TeamName       *string `json:"team_name"`
	OrganizationID *int64  `json:"organization_id"`
}

func (r userPatchRequest) resolvedName() *string {
	if r.UserName != nil {
		return r.UserName
	}
	return r.TeamName
}

func listUsers(c *gin.Context) {
	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var users []userDTO
	if err := db.Table("janus_auth_user").Order("user_id").Find(&users).Error; err != nil {
		respondDBError(c, "list users failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func createUser(c *gin.Context) {
	var req userRequest
	if !bindAdminJSON(c, &req) {
		return
	}
	user := userDTO{
		UserName:       strings.TrimSpace(req.resolvedName()),
		OrganizationID: req.OrganizationID,
	}
	if user.UserName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_name is required"})
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	if err := db.Table("janus_auth_user").
		Clauses(clause.Returning{}).
		Omit("user_id", "create_time", "update_time").
		Create(&user).Error; err != nil {
		respondDBError(c, "create user failed", err)
		return
	}
	c.JSON(http.StatusCreated, user)
}

func getUser(c *gin.Context) {
	id, ok := parseUserIDParam(c)
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var user userDTO
	if !firstByID(c, db.Table("janus_auth_user").Where("user_id = ?", id), &user) {
		return
	}
	c.JSON(http.StatusOK, user)
}

func updateUser(c *gin.Context) {
	id, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	var req userPatchRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	updates := map[string]interface{}{}
	if nameValue := req.resolvedName(); nameValue != nil {
		name := strings.TrimSpace(*nameValue)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_name cannot be empty"})
			return
		}
		updates["user_name"] = name
	}
	if req.OrganizationID != nil {
		updates["organization_id"] = *req.OrganizationID
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	result := db.Table("janus_auth_user").Where("user_id = ?", id).Updates(updates)
	if result.Error != nil {
		respondDBError(c, "update user failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	getUser(c)
}

func deleteUser(c *gin.Context) {
	id, ok := parseUserIDParam(c)
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	result := db.Table("janus_auth_user").Where("user_id = ?", id).Delete(&userDTO{})
	if result.Error != nil {
		respondDBError(c, "delete user failed", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

type keyDTO struct {
	KeyID             int64            `gorm:"primaryKey;autoIncrement;column:key_id" json:"key_id"`
	KeyContent        string           `gorm:"column:key_content" json:"key_content"`
	KeyName           string           `gorm:"column:key_name" json:"key_name"`
	ModelList         auth.StringSlice `gorm:"column:model_list" json:"model_list"`
	UserID            int64            `gorm:"column:user_id" json:"user_id"`
	OrganizationID    int64            `gorm:"column:organization_id" json:"organization_id"`
	Balance           float64          `gorm:"column:balance" json:"balance"`
	TotalSpend        float64          `gorm:"column:total_spend" json:"total_spend"`
	RequestPerMinute  int              `gorm:"column:request_per_minute" json:"request_per_minute"`
	SpendLimitPerWeek float64          `gorm:"column:spend_limit_per_week" json:"spend_limit_per_week"`
	CreateTime        time.Time        `gorm:"column:create_time" json:"-"`
	UpdateTime        time.Time        `gorm:"column:update_time" json:"-"`
	ExpireTime        *time.Time       `gorm:"column:expire_time" json:"expire_time"`
}

type keyRequest struct {
	KeyContent        string     `json:"key_content"`
	KeyName           string     `json:"key_name" binding:"required"`
	AllModels         bool       `json:"all_models"`
	ModelList         []string   `json:"model_list"`
	UserID            int64      `json:"user_id"`
	TeamID            int64      `json:"team_id"`
	OrganizationID    int64      `json:"organization_id" binding:"required"`
	Balance           float64    `json:"balance"`
	RequestPerMinute  int        `json:"request_per_minute"`
	SpendLimitPerWeek float64    `json:"spend_limit_per_week"`
	ExpireTime        *time.Time `json:"expire_time"`
}

func (r keyRequest) resolvedUserID() int64 {
	if r.UserID > 0 {
		return r.UserID
	}
	return r.TeamID
}

type keyPatchRequest struct {
	KeyContent        *string    `json:"key_content"`
	KeyName           *string    `json:"key_name"`
	AllModels         *bool      `json:"all_models"`
	ModelList         *[]string  `json:"model_list"`
	UserID            *int64     `json:"user_id"`
	TeamID            *int64     `json:"team_id"`
	OrganizationID    *int64     `json:"organization_id"`
	Balance           *float64   `json:"balance"`
	RequestPerMinute  *int       `json:"request_per_minute"`
	SpendLimitPerWeek *float64   `json:"spend_limit_per_week"`
	ExpireTime        *time.Time `json:"expire_time"`
}

func (r keyPatchRequest) resolvedUserID() int64 {
	if r.UserID != nil && *r.UserID > 0 {
		return *r.UserID
	}
	if r.TeamID != nil && *r.TeamID > 0 {
		return *r.TeamID
	}
	return 0
}

func normalizeModelList(modelList []string, allModels bool) auth.StringSlice {
	if allModels {
		return auth.StringSlice{"*"}
	}

	normalized := make(auth.StringSlice, 0, len(modelList))
	for _, model := range modelList {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if model == "*" {
			return auth.StringSlice{"*"}
		}
		normalized = append(normalized, model)
	}
	if len(normalized) == 0 {
		return auth.StringSlice{"*"}
	}
	return normalized
}

func listKeys(c *gin.Context) {
	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var keys []keyDTO
	if err := db.Table("janus_auth_key").Order("key_id").Find(&keys).Error; err != nil {
		respondDBError(c, "list keys failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": keys})
}

func createKey(c *gin.Context) {
	var req keyRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	keyContent := strings.TrimSpace(req.KeyContent)
	if keyContent == "" {
		generated, err := generateKeyContent()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "generate key failed"})
			return
		}
		keyContent = generated
	}

	userID := req.resolvedUserID()
	if userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	key := keyDTO{
		KeyContent:        keyContent,
		KeyName:           strings.TrimSpace(req.KeyName),
		ModelList:         normalizeModelList(req.ModelList, req.AllModels),
		UserID:            userID,
		OrganizationID:    req.OrganizationID,
		Balance:           req.Balance,
		TotalSpend:        0,
		RequestPerMinute:  req.RequestPerMinute,
		SpendLimitPerWeek: req.SpendLimitPerWeek,
		ExpireTime:        req.ExpireTime,
	}
	if key.KeyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_name is required"})
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	if err := db.Table("janus_auth_key").
		Clauses(clause.Returning{}).
		Omit("key_id", "create_time", "update_time").
		Create(&key).Error; err != nil {
		respondDBError(c, "create key failed", err)
		return
	}
	c.JSON(http.StatusCreated, key)
}

func getKey(c *gin.Context) {
	id, ok := parseIDParam(c, "key_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var key keyDTO
	if !firstByID(c, db.Table("janus_auth_key").Where("key_id = ?", id), &key) {
		return
	}
	c.JSON(http.StatusOK, key)
}

func updateKey(c *gin.Context) {
	id, ok := parseIDParam(c, "key_id")
	if !ok {
		return
	}
	var req keyPatchRequest
	if !bindAdminJSON(c, &req) {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var existing keyDTO
	if !firstByID(c, db.Table("janus_auth_key").Where("key_id = ?", id), &existing) {
		return
	}

	updates := map[string]interface{}{}
	if req.KeyContent != nil {
		keyContent := strings.TrimSpace(*req.KeyContent)
		if keyContent == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key_content cannot be empty"})
			return
		}
		updates["key_content"] = keyContent
	}
	if req.KeyName != nil {
		keyName := strings.TrimSpace(*req.KeyName)
		if keyName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key_name cannot be empty"})
			return
		}
		updates["key_name"] = keyName
	}
	if req.AllModels != nil && *req.AllModels {
		updates["model_list"] = auth.StringSlice{"*"}
	} else if req.ModelList != nil {
		updates["model_list"] = normalizeModelList(*req.ModelList, false)
	}
	if req.UserID != nil || req.TeamID != nil {
		userID := req.resolvedUserID()
		if userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a positive integer"})
			return
		}
		updates["user_id"] = userID
	}
	if req.OrganizationID != nil {
		updates["organization_id"] = *req.OrganizationID
	}
	if req.Balance != nil {
		updates["balance"] = *req.Balance
	}
	if req.RequestPerMinute != nil {
		updates["request_per_minute"] = *req.RequestPerMinute
	}
	if req.SpendLimitPerWeek != nil {
		updates["spend_limit_per_week"] = *req.SpendLimitPerWeek
	}
	if req.ExpireTime != nil {
		updates["expire_time"] = *req.ExpireTime
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	result := db.Table("janus_auth_key").Where("key_id = ?", id).Updates(updates)
	if result.Error != nil {
		respondDBError(c, "update key failed", result.Error)
		return
	}
	invalidateKeyCache(existing.KeyContent)
	if updatedContent, ok := updates["key_content"].(string); ok {
		invalidateKeyCache(updatedContent)
	}
	getKey(c)
}

func deleteKey(c *gin.Context) {
	id, ok := parseIDParam(c, "key_id")
	if !ok {
		return
	}

	db, ok := connectAdminDB(c)
	if !ok {
		return
	}
	defer janusDb.CloseDatabaseConnection(db)

	var existing keyDTO
	if !firstByID(c, db.Table("janus_auth_key").Where("key_id = ?", id), &existing) {
		return
	}
	result := db.Table("janus_auth_key").Where("key_id = ?", id).Delete(&keyDTO{})
	if result.Error != nil {
		respondDBError(c, "delete key failed", result.Error)
		return
	}
	invalidateKeyCache(existing.KeyContent)
	c.Status(http.StatusNoContent)
}

func invalidateKeyCache(keyContent string) {
	deleteCachedKey(keyContent)
	proxy.RemoveRequestRing(keyContent)
}

func generateKeyContent() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "sk-" + base64.RawURLEncoding.EncodeToString(buf), nil
}

func connectAdminDB(c *gin.Context) (*gorm.DB, bool) {
	db, err := janusDb.ConnectDatabase()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return nil, false
	}
	return db, true
}

func bindAdminJSON(c *gin.Context, out interface{}) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " must be a positive integer"})
		return 0, false
	}
	return id, true
}

func parseUserIDParam(c *gin.Context) (int64, bool) {
	if c.Param("user_id") != "" {
		return parseIDParam(c, "user_id")
	}
	return parseIDParam(c, "team_id")
}

func firstByID(c *gin.Context, query *gorm.DB, out interface{}) bool {
	if err := query.First(out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
			return false
		}
		respondDBError(c, "query resource failed", err)
		return false
	}
	return true
}

func respondDBError(c *gin.Context, message string, err error) {
	status := http.StatusInternalServerError
	if isConstraintError(err) {
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": message})
}

func isConstraintError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "violates foreign key") ||
		strings.Contains(msg, "violates unique constraint") ||
		strings.Contains(msg, "constraint")
}

package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/backup-saas/server/internal/middleware"
	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuthHandler struct {
	db            *gorm.DB
	jwtSecret     string
	refreshSecret string
}

func NewAuthHandler(db *gorm.DB, jwtSecret, refreshSecret string) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret, refreshSecret: refreshSecret}
}

// hashToken returns the hex-encoded SHA-256 of a token string.
// SHA-256 is appropriate here because refresh tokens are high-entropy JWTs;
// bcrypt is unnecessary and would make lookups slow.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (h *AuthHandler) writeAudit(userID, orgID uuid.UUID, action, ip string) {
	h.db.Create(&models.AuditLog{
		OrganizationID: orgID,
		UserID:         userID,
		Action:         action,
		IPAddress:      ip,
	})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Don't reveal whether the email exists
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.writeAudit(user.ID, uuid.Nil, "LOGIN_FAILED", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	accessToken, err := middleware.GenerateAccessToken(user.ID.String(), user.Email, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	rawRefresh, err := middleware.GenerateRefreshToken(user.ID.String(), h.refreshSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	// Each login creates a new token family. All rotations within this session share the family ID.
	familyID := uuid.New()
	h.db.Create(&models.RefreshToken{
		UserID:    user.ID,
		FamilyID:  familyID,
		Token:     hashToken(rawRefresh),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})

	h.writeAudit(user.ID, uuid.Nil, "LOGIN_SUCCESS", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": rawRefresh,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := middleware.ParseRefreshToken(req.RefreshToken, h.refreshSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	tokenHash := hashToken(req.RefreshToken)

	var accessToken, newRawRefresh string

	// Wrap rotation in a transaction with SELECT FOR UPDATE to prevent concurrent
	// requests from both obtaining valid replacement tokens.
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		var rt models.RefreshToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token = ? AND user_id = ? AND expires_at > ?", tokenHash, userID, time.Now()).
			First(&rt).Error; err != nil {
			return err
		}

		// Reuse detection: token found but already revoked → session compromise.
		// Revoke the entire family (only this session, not all user sessions).
		if rt.Revoked {
			tx.Model(&models.RefreshToken{}).
				Where("family_id = ? AND revoked = false", rt.FamilyID).
				Update("revoked", true)
			h.writeAudit(userID, uuid.Nil, "REFRESH_REUSE_DETECTED", c.ClientIP())
			return gorm.ErrRecordNotFound // signal 401 to caller
		}

		// Revoke the current token
		if err := tx.Model(&rt).Update("revoked", true).Error; err != nil {
			return err
		}

		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}

		var genErr error
		accessToken, genErr = middleware.GenerateAccessToken(user.ID.String(), user.Email, h.jwtSecret)
		if genErr != nil {
			return genErr
		}
		newRawRefresh, genErr = middleware.GenerateRefreshToken(user.ID.String(), h.refreshSecret)
		if genErr != nil {
			return genErr
		}

		// New token inherits the same family
		return tx.Create(&models.RefreshToken{
			UserID:    user.ID,
			FamilyID:  rt.FamilyID,
			Token:     hashToken(newRawRefresh),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}).Error
	})

	if txErr != nil {
		if txErr == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session invalidated — please log in again"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found or expired"})
		}
		return
	}

	h.writeAudit(userID, uuid.Nil, "REFRESH", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRawRefresh,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var rt models.RefreshToken
	if err := h.db.Where("token = ?", hashToken(req.RefreshToken)).First(&rt).Error; err == nil {
		h.db.Model(&rt).Update("revoked", true)
		userID, _ := uuid.Parse(c.GetString("user_id"))
		h.writeAudit(userID, uuid.Nil, "LOGOUT", c.ClientIP())
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := models.User{
		Base:         models.Base{ID: uuid.New()},
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	h.db.Model(&user).Update("password_hash", string(hash))
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

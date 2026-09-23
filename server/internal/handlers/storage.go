package handlers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StorageHandler struct {
	db            *gorm.DB
	encryptionKey []byte
}

func NewStorageHandler(db *gorm.DB, encryptionKey []byte) *StorageHandler {
	return &StorageHandler{db: db, encryptionKey: encryptionKey}
}

type createStorageRequest struct {
	Name      string            `json:"name" binding:"required"`
	Type      models.StorageType `json:"type" binding:"required"`
	Bucket    string            `json:"bucket"`
	Region    string            `json:"region"`
	Endpoint  string            `json:"endpoint"`
	AccessKey string            `json:"access_key"`
	SecretKey string            `json:"secret_key"`
	Path      string            `json:"path"`
}

func (h *StorageHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var targets []models.StorageTarget
	h.db.Where("organization_id = ?", orgID).Find(&targets)
	c.JSON(http.StatusOK, asArray(targets))
}

func (h *StorageHandler) Create(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var req createStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	encAccess, err := encrypt(h.encryptionKey, req.AccessKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}
	encSecret, err := encrypt(h.encryptionKey, req.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	target := models.StorageTarget{
		Base:               models.Base{ID: uuid.New()},
		OrganizationID:     orgID,
		Name:               req.Name,
		Type:               req.Type,
		Bucket:             req.Bucket,
		Region:             req.Region,
		Endpoint:           req.Endpoint,
		EncryptedAccessKey: encAccess,
		EncryptedSecretKey: encSecret,
		Path:               req.Path,
	}
	if err := h.db.Create(&target).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, target)
}

func (h *StorageHandler) Delete(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	h.db.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models.StorageTarget{})
	c.JSON(http.StatusNoContent, nil)
}

// encrypt uses AES-256-GCM. Returns base64-encoded ciphertext.
func encrypt(key []byte, plaintext string) (string, error) {
	if len(key) == 0 {
		return plaintext, nil // no-op if key not configured
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func decrypt(key []byte, ciphertext string) (string, error) {
	if len(key) == 0 {
		return ciphertext, nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// DecryptCredential is the exported wrapper used by the gRPC server.
func DecryptCredential(key []byte, ciphertext string) (string, error) {
	return decrypt(key, ciphertext)
}

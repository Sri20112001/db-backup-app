package handlers

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	Name         string             `json:"name" binding:"required"`
	Type         models.StorageType `json:"type" binding:"required"`
	Bucket       string             `json:"bucket"`
	Region       string             `json:"region"`
	Endpoint     string             `json:"endpoint"`
	UsePathStyle bool               `json:"use_path_style"`
	AccessKey    string             `json:"access_key"`
	SecretKey    string             `json:"secret_key"`
	Path         string             `json:"path"`
}

func (h *StorageHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var targets []models.StorageTarget
	h.db.Where("organization_id = ?", orgID).Find(&targets)
	// Transient UI flag — the encrypted values themselves never leave the
	// server (json:"-"); claims unwrap them per-job instead.
	for i := range targets {
		targets[i].HasCredentials = targets[i].EncryptedAccessKey != "" || targets[i].EncryptedSecretKey != ""
	}
	c.JSON(http.StatusOK, asArray(targets))
}

func (h *StorageHandler) Create(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var req createStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fail fast on incomplete targets: S3 needs a bucket, LOCAL needs a
	// path, and anything else isn't implemented yet (SMB is future work).
	switch req.Type {
	case models.StorageS3:
		if strings.TrimSpace(req.Bucket) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bucket is required for S3 targets"})
			return
		}
	case models.StorageLocal:
		if strings.TrimSpace(req.Path) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path is required for LOCAL targets"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported storage type %q (supported: LOCAL, S3)", req.Type)})
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
		UsePathStyle:       req.UsePathStyle,
		EncryptedAccessKey: encAccess,
		EncryptedSecretKey: encSecret,
		Path:               req.Path,
		HasCredentials:     req.AccessKey != "" || req.SecretKey != "",
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

// TestConnection exercises a target end-to-end: PUT a temp object, HEAD
// it, GET it back with checksum comparison, then DELETE it. A mere
// ListBuckets is deliberately NOT enough — scoped credentials often can't
// list, and listing proves nothing about write/read/delete.
//
// LOCAL targets can't be tested from here (their paths live on agent
// machines); S3 round-trips run from the server's network vantage point,
// which may differ from an agent's.
func (h *StorageHandler) TestConnection(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var target models.StorageTarget
	if err := h.db.Where("id = ? AND organization_id = ?", id, orgID).First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storage target not found"})
		return
	}
	if target.Type != models.StorageS3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "connection testing is only available for S3 targets"})
		return
	}
	access, err := decrypt(h.encryptionKey, target.EncryptedAccessKey)
	if err != nil || access == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target has no usable access key"})
		return
	}
	secret, err := decrypt(h.encryptionKey, target.EncryptedSecretKey)
	if err != nil || secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target has no usable secret key"})
		return
	}

	region := strings.TrimSpace(target.Region)
	if region == "" {
		region = "us-east-1"
	}
	awsCfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(access, secret, ""),
	}
	var optFns []func(*s3.Options)
	if endpoint := strings.TrimSpace(target.Endpoint); endpoint != "" {
		optFns = append(optFns, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}
	if target.UsePathStyle {
		optFns = append(optFns, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}
	client := s3.NewFromConfig(awsCfg, optFns...)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	start := time.Now()

	key := ".vaultguard/connection-tests/" + uuid.New().String() + ".txt"
	body := []byte("VaultGuard storage connectivity test")
	sum := sha256.Sum256(body)

	fail := func(stage string, err error) {
		c.JSON(http.StatusBadGateway, gin.H{"status": "error", "stage": stage, "error": err.Error()})
	}
	if _, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(target.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("text/plain"),
	}); err != nil {
		fail("put", err)
		return
	}
	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(target.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		fail("head", err)
		return
	}
	if head.ContentLength != nil && *head.ContentLength != int64(len(body)) {
		fail("head", fmt.Errorf("size mismatch: wrote %d, stored %d", len(body), *head.ContentLength))
		return
	}
	got, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(target.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		fail("get", err)
		return
	}
	data, err := io.ReadAll(got.Body)
	got.Body.Close()
	if err != nil {
		fail("get", err)
		return
	}
	if actual := sha256.Sum256(data); actual != sum {
		fail("checksum", fmt.Errorf("content mismatch: expected %s, got %s",
			hex.EncodeToString(sum[:]), hex.EncodeToString(actual[:])))
		return
	}
	if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(target.Bucket),
		Key:    aws.String(key),
	}); err != nil {
		fail("delete", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"latency_ms": time.Since(start).Milliseconds(),
		"bucket":     target.Bucket,
	})
}

// --- S3 region reference data ---

// ListRegions returns active regions (seeded AWS codes + admin customs) for
// the storage-target region picker. Reference data is global, not per-org.
func (h *StorageHandler) ListRegions(c *gin.Context) {
	var regions []models.S3Region
	h.db.Where("active = ?", true).Order("provider ASC, name ASC").Find(&regions)
	c.JSON(http.StatusOK, asArray(regions))
}

type createRegionRequest struct {
	Code     string `json:"code" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Provider string `json:"provider"`
	Endpoint string `json:"endpoint"`
}

// CreateRegion adds a custom region (e.g. a private S3 clone). Seeded
// system rows are identified by code uniqueness, not by caller.
func (h *StorageHandler) CreateRegion(c *gin.Context) {
	var req createRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	code := strings.ToLower(strings.TrimSpace(req.Code))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	provider := strings.ToUpper(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = "CUSTOM"
	}
	region := models.S3Region{
		Base:      models.Base{ID: uuid.New()},
		Code:      code,
		Name:      strings.TrimSpace(req.Name),
		Provider:  provider,
		Endpoint:  strings.TrimSpace(req.Endpoint),
		IsSystem:  false,
		Active:    true,
	}
	if err := h.db.Create(&region).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "uni_s3_regions_code") {
			c.JSON(http.StatusConflict, gin.H{"error": "region code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, region)
}

// DeleteRegion removes a custom region. System (seeded) rows are protected.
func (h *StorageHandler) DeleteRegion(c *gin.Context) {
	code := strings.ToLower(strings.TrimSpace(c.Param("code")))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	var region models.S3Region
	if err := h.db.Where("code = ?", code).First(&region).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "region not found"})
		return
	}
	if region.IsSystem {
		c.JSON(http.StatusBadRequest, gin.H{"error": "system regions cannot be deleted (deactivate instead)"})
		return
	}
	h.db.Delete(&region)
	c.JSON(http.StatusNoContent, nil)
}

// encrypt uses AES-256-GCM. Returns base64-encoded ciphertext.
func encrypt(key []byte, plaintext string) (string, error) {
	if len(key) == 0 {
		return "", errors.New("encryption key not configured")
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
		return "", errors.New("encryption key not configured")
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

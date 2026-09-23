package handlers

import (
	"net/http"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MachineHandler struct {
	db *gorm.DB
}

func NewMachineHandler(db *gorm.DB) *MachineHandler {
	return &MachineHandler{db: db}
}

func (h *MachineHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var machines []models.Machine
	h.db.Preload("Agent").Where("organization_id = ?", orgID).Find(&machines)
	c.JSON(http.StatusOK, asArray(machines))
}

func (h *MachineHandler) Get(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	machineID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var machine models.Machine
	if err := h.db.Preload("Agent").Where("id = ? AND organization_id = ?", machineID, orgID).First(&machine).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, machine)
}

package devices

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rmm-platform/backend/internal/alerts"
	"github.com/rmm-platform/backend/internal/auth"
	"github.com/rmm-platform/backend/internal/realtime"
	"github.com/rmm-platform/backend/internal/shared/config"
)

type Handler struct {
	repo   *Repository
	alerts *alerts.Repository
	hub    *realtime.Hub
}

func NewHandler(db *sqlx.DB, alertRepo *alerts.Repository, hub *realtime.Hub) *Handler {
	return &Handler{repo: NewRepository(db), alerts: alertRepo, hub: hub}
}

func checkAndCreateAlerts(r *alerts.Repository, deviceID string, req *HeartbeatRequest, hub *realtime.Hub) {
	online := true
	created, err := r.CheckAndCreateAlerts(deviceID, req.CPUPercent, req.RAMPercent, req.DiskPercent, online, req.POSRunning, req.MSSQLRunning)
	if err != nil || len(created) == 0 {
		return
	}
	for _, a := range created {
		hub.Broadcast(realtime.Message{Type: "alert.created", Payload: a})
	}
}

// List godoc
// @Summary List devices
// @Description Get paginated list of devices
// @Tags devices
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 50, max 100)"
// @Param search query string false "Search by hostname or customer"
// @Success 200 {object} DeviceListResponse
// @Failure 500 {object} map[string]string
// @Router /devices [get]
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	devices, total, err := h.repo.List(page, limit, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list devices: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, DeviceListResponse{
		Devices: devices,
		Total:   total,
		Page:    page,
		Limit:   limit,
	})
}

// Get godoc
// @Summary Get device by ID
// @Description Get detailed device information
// @Tags devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID"
// @Success 200 {object} Device
// @Failure 404 {object} map[string]string
// @Router /devices/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	device, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	c.JSON(http.StatusOK, device)
}

// Create godoc
// @Summary Create device
// @Description Register a new device
// @Tags devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body DeviceCreateRequest true "Device data"
// @Success 201 {object} Device
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices [post]
func (h *Handler) Create(c *gin.Context) {
	var req DeviceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := h.repo.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.hub.Broadcast(realtime.Message{Type: "device.created", Payload: device})
	c.JSON(http.StatusCreated, device)
}

// Update godoc
// @Summary Update device
// @Description Update device information
// @Tags devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID"
// @Param body body DeviceUpdateRequest true "Device update data"
// @Success 200 {object} Device
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var req DeviceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Update(id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update device"})
		return
	}

	device, _ := h.repo.GetByID(id)
	h.hub.Broadcast(realtime.Message{Type: "device.updated", Payload: device})
	c.JSON(http.StatusOK, device)
}

// Delete godoc
// @Summary Delete device
// @Description Soft-delete a device
// @Tags devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID"
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete device"})
		return
	}

	h.hub.Broadcast(realtime.Message{Type: "device.deleted", Payload: gin.H{"id": id}})
	c.JSON(http.StatusOK, gin.H{"message": "device deleted"})
}

// Heartbeat godoc
// @Summary Receive device heartbeat
// @Description Process agent heartbeat with system metrics
// @Tags devices
// @Accept json
// @Produce json
// @Param body body HeartbeatRequest true "Heartbeat data"
// @Success 200 {object} Device
// @Failure 400 {object} map[string]string
// @Router /devices/heartbeat [post]
func (h *Handler) Heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := h.repo.UpdateHeartbeat(req.Hostname, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	checkAndCreateAlerts(h.alerts, device.ID, &req, h.hub)

	h.hub.Broadcast(realtime.Message{Type: "device.heartbeat", Payload: device})
	c.JSON(http.StatusOK, device)
}

// GetMetrics godoc
// @Summary Get device metrics history
// @Description Get historical metrics for a device
// @Tags devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID"
// @Param limit query int false "Number of metrics entries (default 60)"
// @Success 200 {array} devices.DeviceMetric
// @Failure 500 {object} map[string]string
// @Router /devices/{id}/metrics [get]
func (h *Handler) GetMetrics(c *gin.Context) {
	id := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "60"))

	metrics, err := h.repo.GetMetrics(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get metrics"})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func RegisterRoutes(rg *gin.RouterGroup, db *sqlx.DB, cfg *config.Config, hub *realtime.Hub, alertRepo *alerts.Repository) {
	h := NewHandler(db, alertRepo, hub)
	mid := auth.NewMiddleware(&cfg.JWT)

	devices := rg.Group("/devices")
	devices.Use(mid.RequireAuth())
	{
		devices.GET("", h.List)
		devices.GET("/:id", h.Get)
		devices.POST("", h.Create)
		devices.PUT("/:id", h.Update)
		devices.DELETE("/:id", h.Delete)
		devices.GET("/:id/metrics", h.GetMetrics)
	}

	rg.POST("/devices/heartbeat", h.Heartbeat)
}

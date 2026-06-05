package monitoring

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type Handler struct {
	deviceRepo *DeviceRepository
}

type DeviceRepository struct {
	db *sqlx.DB
}

func NewDeviceRepository(db *sqlx.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) GetDashboardSummary() (*DashboardSummary, error) {
	var s DashboardSummary
	r.db.Get(&s.TotalDevices, `SELECT COUNT(*) FROM devices WHERE is_active = TRUE`)
	r.db.Get(&s.OnlineDevices, `SELECT COUNT(*) FROM devices WHERE is_active = TRUE AND is_online = TRUE`)
	r.db.Get(&s.OfflineDevices, `SELECT COUNT(*) FROM devices WHERE is_active = TRUE AND is_online = FALSE`)
	r.db.Get(&s.CriticalAlerts, `SELECT COUNT(*) FROM alerts WHERE status = 'open' AND severity IN ('critical','warning')`)
	r.db.Get(&s.OpenTickets, `SELECT COUNT(*) FROM tickets WHERE status NOT IN ('resolved','closed')`)
	return &s, nil
}

type DashboardSummary struct {
	TotalDevices   int `json:"total_devices"`
	OnlineDevices  int `json:"online_devices"`
	OfflineDevices int `json:"offline_devices"`
	CriticalAlerts int `json:"critical_alerts"`
	OpenTickets    int `json:"open_tickets"`
}

func NewHandler(db *sqlx.DB) *Handler {
	return &Handler{deviceRepo: NewDeviceRepository(db)}
}

// GetDashboard godoc
// @Summary Get dashboard summary
// @Description Get overall system health summary
// @Tags monitoring
// @Produce json
// @Success 200 {object} DashboardSummary
// @Failure 500 {object} map[string]string
// @Router /dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	summary, err := h.deviceRepo.GetDashboardSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get dashboard data"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func RegisterRoutes(rg *gin.RouterGroup, db *sqlx.DB) {
	h := NewHandler(db)
	rg.GET("/dashboard", h.GetDashboard)
}

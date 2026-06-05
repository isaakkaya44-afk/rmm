package remote

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rmm-platform/backend/internal/auth"
	"github.com/rmm-platform/backend/internal/shared/config"
)

type RemoteSession struct {
	ID              string  `json:"id" db:"id"`
	DeviceID        string  `json:"device_id" db:"device_id"`
	TechnicianID    *string `json:"technician_id,omitempty" db:"technician_id"`
	SessionType     string  `json:"session_type" db:"session_type"`
	SessionID       *string `json:"session_id,omitempty" db:"session_id"`
	Status          string  `json:"status" db:"status"`
	StartedAt       string  `json:"started_at" db:"started_at"`
	EndedAt         *string `json:"ended_at,omitempty" db:"ended_at"`
	DurationSeconds *int    `json:"duration_seconds,omitempty" db:"duration_seconds"`
	Notes           *string `json:"notes,omitempty" db:"notes"`
}

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByDevice(deviceID string) ([]RemoteSession, error) {
	var sessions []RemoteSession
	err := r.db.Select(&sessions,
		`SELECT rs.*, u.full_name as technician_name 
		FROM remote_sessions rs 
		LEFT JOIN users u ON u.id = rs.technician_id 
		WHERE rs.device_id = $1 
		ORDER BY rs.started_at DESC`, deviceID)
	return sessions, err
}

func (r *Repository) CreateSession(deviceID, technicianID, sessionType, sessionID string) (*RemoteSession, error) {
	var id string
	err := r.db.Get(&id,
		`INSERT INTO remote_sessions (device_id, technician_id, session_type, session_id, status) 
		VALUES ($1, $2, $3, $4, 'active') RETURNING id`,
		deviceID, technicianID, sessionType, sessionID)
	if err != nil {
		return nil, err
	}

	var session RemoteSession
	err = r.db.Get(&session, `SELECT * FROM remote_sessions WHERE id = $1`, id)
	return &session, err
}

func RegisterRoutes(rg *gin.RouterGroup, db *sqlx.DB) {
	repo := NewRepository(db)
	mid := auth.NewMiddleware(&config.Load().JWT)

	remote := rg.Group("/remote")
	remote.Use(mid.RequireAuth())
	{
		remote.GET("/sessions/:deviceId", func(c *gin.Context) {
			sessions, err := repo.ListByDevice(c.Param("deviceId"))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sessions"})
				return
			}
			c.JSON(http.StatusOK, sessions)
		})

		remote.POST("/sessions", func(c *gin.Context) {
			var req struct {
				DeviceID    string `json:"device_id" binding:"required"`
				SessionType string `json:"session_type" binding:"required"`
				SessionID   string `json:"session_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			userID, _ := c.Get("user_id")
			session, err := repo.CreateSession(req.DeviceID, userID.(string), req.SessionType, req.SessionID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, session)
		})
	}
}

package tickets

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rmm-platform/backend/internal/auth"
	"github.com/rmm-platform/backend/internal/realtime"
	"github.com/rmm-platform/backend/internal/shared/config"
)

type Handler struct {
	repo *Repository
	hub  *realtime.Hub
}

func NewHandler(db *sqlx.DB, hub *realtime.Hub) *Handler {
	return &Handler{repo: NewRepository(db), hub: hub}
}

// List godoc
// @Summary List tickets
// @Description Get paginated list of support tickets
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 50, max 100)"
// @Param status query string false "Filter by status"
// @Param device_id query string false "Filter by device ID"
// @Success 200 {object} TicketListResponse
// @Failure 500 {object} map[string]string
// @Router /tickets [get]
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	status := c.Query("status")
	deviceID := c.Query("device_id")

	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 50 }

	tickets, total, err := h.repo.List(page, limit, status, deviceID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tickets"})
		return
	}

	c.JSON(http.StatusOK, TicketListResponse{Tickets: tickets, Total: total, Page: page, Limit: limit})
}

// Get godoc
// @Summary Get ticket by ID
// @Description Get ticket details with comments
// @Tags tickets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /tickets/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	ticket, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	comments, _ := h.repo.GetComments(id)
	c.JSON(http.StatusOK, gin.H{"ticket": ticket, "comments": comments})
}

// Create godoc
// @Summary Create ticket
// @Description Create a new support ticket
// @Tags tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateTicketRequest true "Ticket data"
// @Success 201 {object} Ticket
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tickets [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	ticket, err := h.repo.Create(&req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.hub.Broadcast(realtime.Message{Type: "ticket.created", Payload: ticket})
	c.JSON(http.StatusCreated, ticket)
}

// Update godoc
// @Summary Update ticket
// @Description Update ticket status or details
// @Tags tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Param body body UpdateTicketRequest true "Ticket update data"
// @Success 200 {object} Ticket
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tickets/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Update(id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
		return
	}

	ticket, _ := h.repo.GetByID(id)
	h.hub.Broadcast(realtime.Message{Type: "ticket.updated", Payload: ticket})
	c.JSON(http.StatusOK, ticket)
}

// AddComment godoc
// @Summary Add ticket comment
// @Description Add a comment to a support ticket
// @Tags tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Param body body AddCommentRequest true "Comment data"
// @Success 201 {object} tickets.TicketComment
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tickets/{id}/comments [post]
func (h *Handler) AddComment(c *gin.Context) {
	id := c.Param("id")
	var req AddCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	comment, err := h.repo.AddComment(id, userID.(string), req.Content, req.IsInternal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add comment"})
		return
	}

	h.hub.Broadcast(realtime.Message{Type: "ticket.comment_added", Payload: comment})
	c.JSON(http.StatusCreated, comment)
}

func RegisterRoutes(rg *gin.RouterGroup, db *sqlx.DB, cfg *config.Config, hub *realtime.Hub) {
	h := NewHandler(db, hub)
	mid := auth.NewMiddleware(&cfg.JWT)

	tickets := rg.Group("/tickets")
	tickets.Use(mid.RequireAuth())
	{
		tickets.GET("", h.List)
		tickets.GET("/:id", h.Get)
		tickets.POST("", h.Create)
		tickets.PUT("/:id", h.Update)
		tickets.POST("/:id/comments", h.AddComment)
	}
}

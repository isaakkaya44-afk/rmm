package tickets

import (
	"time"
)

type Ticket struct {
	ID          string     `json:"id" db:"id"`
	TicketNumber int       `json:"ticket_number" db:"ticket_number"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description,omitempty" db:"description"`
	Status      string     `json:"status" db:"status"`
	Priority    string     `json:"priority" db:"priority"`
	Source      string     `json:"source" db:"source"`
	DeviceID    *string    `json:"device_id,omitempty" db:"device_id"`
	CustomerID  *string    `json:"customer_id,omitempty" db:"customer_id"`
	AssignedTo  *string    `json:"assigned_to,omitempty" db:"assigned_to"`
	CreatedBy   *string    `json:"created_by,omitempty" db:"created_by"`
	AlertID     *string    `json:"alert_id,omitempty" db:"alert_id"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`

	// Joined
	DeviceName   *string `json:"device_name,omitempty" db:"device_name"`
	CustomerName *string `json:"customer_name,omitempty" db:"customer_name"`
	AssignedName *string `json:"assigned_name,omitempty" db:"assigned_name"`
	CreatedName  *string `json:"created_name,omitempty" db:"created_name"`
}

type TicketComment struct {
	ID         string    `json:"id" db:"id"`
	TicketID   string    `json:"ticket_id" db:"ticket_id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Content    string    `json:"content" db:"content"`
	IsInternal bool      `json:"is_internal" db:"is_internal"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UserName   *string   `json:"user_name,omitempty" db:"user_name"`
}

type CreateTicketRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description *string `json:"description,omitempty"`
	Priority    string  `json:"priority"`
	Source      string  `json:"source"`
	DeviceID    *string `json:"device_id,omitempty"`
	CustomerID  *string `json:"customer_id,omitempty"`
	AssignedTo  *string `json:"assigned_to,omitempty"`
	AlertID     *string `json:"alert_id,omitempty"`
}

type UpdateTicketRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
	Priority    *string `json:"priority,omitempty"`
	AssignedTo  *string `json:"assigned_to,omitempty"`
}

type AddCommentRequest struct {
	Content    string `json:"content" binding:"required"`
	IsInternal bool   `json:"is_internal"`
}

type TicketListResponse struct {
	Tickets []Ticket `json:"tickets"`
	Total   int      `json:"total"`
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
}

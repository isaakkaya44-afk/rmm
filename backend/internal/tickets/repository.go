package tickets

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(page, limit int, status, deviceID, assignedTo string) ([]Ticket, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if status != "" {
		where += fmt.Sprintf(" AND t.status = $%d", idx)
		args = append(args, status)
		idx++
	}
	if deviceID != "" {
		where += fmt.Sprintf(" AND t.device_id = $%d", idx)
		args = append(args, deviceID)
		idx++
	}
	if assignedTo != "" {
		where += fmt.Sprintf(" AND t.assigned_to = $%d", idx)
		args = append(args, assignedTo)
		idx++
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tickets t %s", where)
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query := fmt.Sprintf(`SELECT t.*, d.hostname as device_name, 
		c.name as customer_name, 
		u1.full_name as assigned_name,
		u2.full_name as created_name
		FROM tickets t 
		LEFT JOIN devices d ON d.id = t.device_id
		LEFT JOIN customers c ON c.id = t.customer_id
		LEFT JOIN users u1 ON u1.id = t.assigned_to
		LEFT JOIN users u2 ON u2.id = t.created_by
		%s ORDER BY t.created_at DESC LIMIT $%d OFFSET $%d`, where, idx, idx+1)
	args = append(args, limit, offset)

	var tickets []Ticket
	if err := r.db.Select(&tickets, query, args...); err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

func (r *Repository) GetByID(id string) (*Ticket, error) {
	var ticket Ticket
	err := r.db.Get(&ticket, `SELECT t.*, d.hostname as device_name, 
		c.name as customer_name, 
		u1.full_name as assigned_name,
		u2.full_name as created_name
		FROM tickets t 
		LEFT JOIN devices d ON d.id = t.device_id
		LEFT JOIN customers c ON c.id = t.customer_id
		LEFT JOIN users u1 ON u1.id = t.assigned_to
		LEFT JOIN users u2 ON u2.id = t.created_by
		WHERE t.id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}
	return &ticket, nil
}

func (r *Repository) Create(req *CreateTicketRequest, createdBy string) (*Ticket, error) {
	id := uuid.New().String()
	priority := req.Priority
	source := req.Source
	if priority == "" { priority = "medium" }
	if source == "" { source = "manual" }

	_, err := r.db.Exec(
		`INSERT INTO tickets (id, title, description, priority, source, device_id, customer_id, assigned_to, created_by, alert_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, req.Title, req.Description, priority, source, req.DeviceID, req.CustomerID, req.AssignedTo, createdBy, req.AlertID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}
	return r.GetByID(id)
}

func (r *Repository) Update(id string, req *UpdateTicketRequest) error {
	query := `UPDATE tickets SET updated_at = NOW()`
	args := []interface{}{}
	idx := 1

	if req.Title != nil {
		query += fmt.Sprintf(", title = $%d", idx); args = append(args, *req.Title); idx++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", idx); args = append(args, *req.Description); idx++
	}
	if req.Status != nil {
		query += fmt.Sprintf(", status = $%d", idx); args = append(args, *req.Status); idx++
	}
	if req.Priority != nil {
		query += fmt.Sprintf(", priority = $%d", idx); args = append(args, *req.Priority); idx++
	}
	if req.AssignedTo != nil {
		query += fmt.Sprintf(", assigned_to = $%d", idx); args = append(args, *req.AssignedTo); idx++
	}

	if req.Status != nil && *req.Status == "resolved" {
		query += fmt.Sprintf(", resolved_at = NOW()")
	}

	query += fmt.Sprintf(" WHERE id = $%d", idx)
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *Repository) AddComment(ticketID, userID, content string, isInternal bool) (*TicketComment, error) {
	id := uuid.New().String()
	_, err := r.db.Exec(
		`INSERT INTO ticket_comments (id, ticket_id, user_id, content, is_internal) VALUES ($1, $2, $3, $4, $5)`,
		id, ticketID, userID, content, isInternal,
	)
	if err != nil {
		return nil, err
	}

	var comment TicketComment
	err = r.db.Get(&comment,
		`SELECT tc.*, u.full_name as user_name FROM ticket_comments tc 
		LEFT JOIN users u ON u.id = tc.user_id WHERE tc.id = $1`, id)
	return &comment, err
}

func (r *Repository) GetComments(ticketID string) ([]TicketComment, error) {
	var comments []TicketComment
	err := r.db.Select(&comments,
		`SELECT tc.*, u.full_name as user_name FROM ticket_comments tc 
		LEFT JOIN users u ON u.id = tc.user_id 
		WHERE tc.ticket_id = $1 ORDER BY tc.created_at ASC`, ticketID)
	return comments, err
}

func (r *Repository) CreateFromAlert(alertID, deviceID, title, description string) (*Ticket, error) {
	req := &CreateTicketRequest{
		Title:       title,
		Description: &description,
		Priority:    "high",
		DeviceID:    &deviceID,
		AlertID:     &alertID,
		Source:      "alert",
	}
	return r.Create(req, "")
}



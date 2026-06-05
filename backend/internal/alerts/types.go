package alerts

import (
	"time"
)

type Alert struct {
	ID              string     `json:"id" db:"id"`
	DeviceID        string     `json:"device_id" db:"device_id"`
	DeviceName      *string    `json:"device_name,omitempty" db:"device_name"`
	Type            string     `json:"type" db:"type"`
	Severity        string     `json:"severity" db:"severity"`
	Title           string     `json:"title" db:"title"`
	Message         *string    `json:"message,omitempty" db:"message"`
	MetricValue     *float64   `json:"metric_value,omitempty" db:"metric_value"`
	ThresholdValue  *float64   `json:"threshold_value,omitempty" db:"threshold_value"`
	Status          string     `json:"status" db:"status"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	AcknowledgedBy  *string    `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy      *string    `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolutionNote  *string    `json:"resolution_note,omitempty" db:"resolution_note"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

const (
	TypeCPUHigh      = "cpu_high"
	TypeRAMHigh      = "ram_high"
	TypeDiskHigh     = "disk_high"
	TypeDeviceOffline = "device_offline"
	TypePOSDown      = "pos_process_down"
	TypeMSSQLStopped = "mssql_stopped"

	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"

	StatusOpen       = "open"
	StatusAcknowledged = "acknowledged"
	StatusResolved   = "resolved"
)

type AlertListResponse struct {
	Alerts []Alert `json:"alerts"`
	Total  int     `json:"total"`
}

type ResolveRequest struct {
	ResolutionNote string `json:"resolution_note,omitempty"`
}

type AcknowledgeRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

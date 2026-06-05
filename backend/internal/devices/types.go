package devices

import (
	"time"

	"github.com/lib/pq"
)

type Device struct {
	ID               string         `json:"id" db:"id"`
	CustomerID       *string        `json:"customer_id,omitempty" db:"customer_id"`
	CustomerName     *string        `json:"customer_name,omitempty" db:"customer_name"`
	Hostname         string         `json:"hostname" db:"hostname"`
	OSVersion        *string        `json:"os_version,omitempty" db:"os_version"`
	CPUModel         *string        `json:"cpu_model,omitempty" db:"cpu_model"`
	CPUCores         *int           `json:"cpu_cores,omitempty" db:"cpu_cores"`
	RAMTotalMB       *int64         `json:"ram_total_mb,omitempty" db:"ram_total_mb"`
	DiskTotalMB      *int64         `json:"disk_total_mb,omitempty" db:"disk_total_mb"`
	RustDeskID       *string        `json:"rustdesk_id,omitempty" db:"rustdesk_id"`
	RustDeskPassword *string        `json:"rustdesk_password,omitempty" db:"rustdesk_password"`
	MSSQLStatus      string         `json:"mssql_status" db:"mssql_status"`
	POSStatus        string         `json:"pos_process_status" db:"pos_process_status"`
	AgentVersion     *string        `json:"agent_version,omitempty" db:"agent_version"`
	LastHeartbeat    *time.Time     `json:"last_heartbeat" db:"last_heartbeat"`
	IsOnline         bool           `json:"is_online" db:"is_online"`
	IsActive         bool           `json:"is_active" db:"is_active"`
	Tags             pq.StringArray `json:"tags,omitempty" db:"tags" swaggertype:"array,string"`
	Notes            *string        `json:"notes,omitempty" db:"notes"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
}

type DeviceCreateRequest struct {
	Hostname   string  `json:"hostname" binding:"required"`
	CustomerID *string `json:"customer_id,omitempty"`
	OSVersion  *string `json:"os_version,omitempty"`
	RustDeskID *string `json:"rustdesk_id,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

type DeviceUpdateRequest struct {
	Hostname         *string  `json:"hostname,omitempty"`
	CustomerID       *string  `json:"customer_id,omitempty"`
	RustDeskID       *string  `json:"rustdesk_id,omitempty"`
	RustDeskPassword *string  `json:"rustdesk_password,omitempty"`
	MSSQLStatus      *string  `json:"mssql_status,omitempty"`
	POSStatus        *string  `json:"pos_process_status,omitempty"`
	Notes            *string  `json:"notes,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}

type HeartbeatRequest struct {
	Hostname       string  `json:"hostname" binding:"required"`
	OSVersion      *string `json:"os_version,omitempty"`
	CPUModel       *string `json:"cpu_model,omitempty"`
	CPUCores       *int    `json:"cpu_cores,omitempty"`
	RAMTotalMB     *int64  `json:"ram_total_mb,omitempty"`
	RAMUsedMB      *int64  `json:"ram_used_mb,omitempty"`
	RAMPercent     *float64 `json:"ram_percent,omitempty"`
	DiskTotalMB    *int64  `json:"disk_total_mb,omitempty"`
	DiskUsedMB     *int64  `json:"disk_used_mb,omitempty"`
	DiskPercent    *float64 `json:"disk_percent,omitempty"`
	CPUPercent     *float64 `json:"cpu_percent,omitempty"`
	UptimeSeconds  *int64  `json:"uptime_seconds,omitempty"`
	RustDeskID     *string `json:"rustdesk_id,omitempty"`
	POSRunning     *bool   `json:"pos_running,omitempty"`
	MSSQLRunning   *bool   `json:"mssql_running,omitempty"`
	AgentVersion   *string `json:"agent_version,omitempty"`
}

type DeviceListResponse struct {
	Devices []Device `json:"devices"`
	Total   int      `json:"total"`
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
}

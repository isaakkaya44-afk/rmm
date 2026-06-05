package devices

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(page, limit int, search string) ([]Device, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM devices WHERE is_active = TRUE`
	if search != "" {
		searchParam := "%" + search + "%"
		countQuery += ` AND (hostname ILIKE $1 OR CAST(customer_id AS TEXT) ILIKE $1)`
		if err := r.db.Get(&total, countQuery, searchParam); err != nil {
			return nil, 0, err
		}
	} else {
		if err := r.db.Get(&total, countQuery); err != nil {
			return nil, 0, err
		}
	}

	offset := (page - 1) * limit
	query := `SELECT d.*, c.name as customer_name FROM devices d 
		LEFT JOIN customers c ON c.id = d.customer_id 
		WHERE d.is_active = TRUE`

	var args []interface{}
	if search != "" {
		query += ` AND (d.hostname ILIKE $1 OR CAST(d.customer_id AS TEXT) ILIKE $1)`
		searchParam := "%" + search + "%"
		args = append(args, searchParam, limit, offset)
		query += ` ORDER BY d.last_heartbeat DESC NULLS LAST LIMIT $2 OFFSET $3`
	} else {
		args = append(args, limit, offset)
		query += ` ORDER BY d.last_heartbeat DESC NULLS LAST LIMIT $1 OFFSET $2`
	}

	var devices []Device
	err := r.db.Select(&devices, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

func (r *Repository) GetByID(id string) (*Device, error) {
	var device Device
	query := `SELECT d.*, c.name as customer_name FROM devices d 
		LEFT JOIN customers c ON c.id = d.customer_id 
		WHERE d.id = $1`
	err := r.db.Get(&device, query, id)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	return &device, nil
}

func (r *Repository) GetByHostname(hostname string) (*Device, error) {
	var device Device
	err := r.db.Get(&device, `SELECT * FROM devices WHERE hostname = $1`, hostname)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	return &device, nil
}

func (r *Repository) Create(req *DeviceCreateRequest) (*Device, error) {
	id := uuid.New().String()
	_, err := r.db.Exec(
		`INSERT INTO devices (id, hostname, customer_id, os_version, rustdesk_id, notes) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, req.Hostname, req.CustomerID, req.OSVersion, req.RustDeskID, req.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}
	return r.GetByID(id)
}

func (r *Repository) Update(id string, req *DeviceUpdateRequest) error {
	query := `UPDATE devices SET updated_at = NOW()`
	args := []interface{}{}
	argIdx := 1

	if req.Hostname != nil {
		query += fmt.Sprintf(", hostname = $%d", argIdx)
		args = append(args, *req.Hostname)
		argIdx++
	}
	if req.CustomerID != nil {
		query += fmt.Sprintf(", customer_id = $%d", argIdx)
		args = append(args, *req.CustomerID)
		argIdx++
	}
	if req.RustDeskID != nil {
		query += fmt.Sprintf(", rustdesk_id = $%d", argIdx)
		args = append(args, *req.RustDeskID)
		argIdx++
	}
	if req.RustDeskPassword != nil {
		query += fmt.Sprintf(", rustdesk_password = $%d", argIdx)
		args = append(args, *req.RustDeskPassword)
		argIdx++
	}
	if req.MSSQLStatus != nil {
		query += fmt.Sprintf(", mssql_status = $%d", argIdx)
		args = append(args, *req.MSSQLStatus)
		argIdx++
	}
	if req.POSStatus != nil {
		query += fmt.Sprintf(", pos_process_status = $%d", argIdx)
		args = append(args, *req.POSStatus)
		argIdx++
	}
	if req.Notes != nil {
		query += fmt.Sprintf(", notes = $%d", argIdx)
		args = append(args, *req.Notes)
		argIdx++
	}
	if req.Tags != nil {
		query += fmt.Sprintf(", tags = $%d", argIdx)
		args = append(args, req.Tags)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *Repository) Delete(id string) error {
	_, err := r.db.Exec(`UPDATE devices SET is_active = FALSE, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *Repository) UpdateHeartbeat(hostname string, req *HeartbeatRequest) (*Device, error) {
	device, err := r.GetByHostname(hostname)
	if err != nil {
		id := uuid.New().String()
		_, err = r.db.Exec(
			`INSERT INTO devices (id, hostname, os_version, rustdesk_id, agent_version, is_online, last_heartbeat) 
			VALUES ($1, $2, $3, $4, $5, TRUE, NOW())`,
			id, hostname, req.OSVersion, req.RustDeskID, req.AgentVersion,
		)
		if err != nil {
			return nil, err
		}
		return r.GetByID(id)
	}

	_, err = r.db.Exec(
		`UPDATE devices SET 
			os_version = COALESCE($2, os_version),
			cpu_model = COALESCE($3, cpu_model),
			cpu_cores = COALESCE($4, cpu_cores),
			ram_total_mb = COALESCE($5, ram_total_mb),
			disk_total_mb = COALESCE($6, disk_total_mb),
			rustdesk_id = COALESCE($7, rustdesk_id),
			agent_version = COALESCE($8, agent_version),
			is_online = TRUE,
			last_heartbeat = NOW(),
			updated_at = NOW()
		WHERE hostname = $1`,
		hostname, req.OSVersion, req.CPUModel, req.CPUCores,
		req.RAMTotalMB, req.DiskTotalMB, req.RustDeskID, req.AgentVersion,
	)
	if err != nil {
		return nil, err
	}

	_, err = r.db.Exec(
		`INSERT INTO device_metrics (device_id, cpu_percent, ram_percent, ram_used_mb, 
			disk_percent, disk_used_mb, uptime_seconds, pos_running, mssql_running) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		device.ID, req.CPUPercent, req.RAMPercent, req.RAMUsedMB,
		req.DiskPercent, req.DiskUsedMB, req.UptimeSeconds, req.POSRunning, req.MSSQLRunning,
	)
	if err != nil {
		return nil, err
	}

	return r.GetByID(device.ID)
}

func (r *Repository) SetOffline(deviceID string) error {
	_, err := r.db.Exec(
		`UPDATE devices SET is_online = FALSE, updated_at = NOW() WHERE id = $1`,
		deviceID,
	)
	return err
}

func (r *Repository) MarkDevicesOffline(threshold time.Duration) ([]string, error) {
	var ids []string
	err := r.db.Select(&ids,
		`SELECT id FROM devices 
		WHERE is_online = TRUE 
		AND (last_heartbeat IS NULL OR last_heartbeat < NOW() - $1::interval)`,
		threshold.String(),
	)
	if err != nil {
		return nil, err
	}

	if len(ids) > 0 {
		_, err = r.db.Exec(
			`UPDATE devices SET is_online = FALSE, updated_at = NOW() 
			WHERE id = ANY($1)`,
			ids,
		)
	}

	return ids, err
}

func (r *Repository) SetMSSQLStatus(deviceID, status string) error {
	_, err := r.db.Exec(
		`UPDATE devices SET mssql_status = $1, updated_at = NOW() WHERE id = $2`,
		status, deviceID,
	)
	return err
}

func (r *Repository) SetPOSProcessStatus(deviceID, status string) error {
	_, err := r.db.Exec(
		`UPDATE devices SET pos_process_status = $1, updated_at = NOW() WHERE id = $2`,
		status, deviceID,
	)
	return err
}

// DeviceMetrics
type DeviceMetric struct {
	ID            int64      `json:"id" db:"id"`
	DeviceID      string     `json:"device_id" db:"device_id"`
	CPUPercent    *float64   `json:"cpu_percent" db:"cpu_percent"`
	RAMPercent    *float64   `json:"ram_percent" db:"ram_percent"`
	RAMUsedMB     *int64     `json:"ram_used_mb" db:"ram_used_mb"`
	DiskPercent   *float64   `json:"disk_percent" db:"disk_percent"`
	DiskUsedMB    *int64     `json:"disk_used_mb" db:"disk_used_mb"`
	UptimeSeconds *int64     `json:"uptime_seconds" db:"uptime_seconds"`
	POSRunning    *bool      `json:"pos_running" db:"pos_running"`
	MSSQLRunning  *bool      `json:"mssql_running" db:"mssql_running"`
	RecordedAt    time.Time  `json:"recorded_at" db:"recorded_at"`
}

func (r *Repository) GetMetrics(deviceID string, limit int) ([]DeviceMetric, error) {
	var metrics []DeviceMetric
	err := r.db.Select(&metrics,
		`SELECT * FROM device_metrics WHERE device_id = $1 
		ORDER BY recorded_at DESC LIMIT $2`,
		deviceID, limit,
	)
	return metrics, err
}

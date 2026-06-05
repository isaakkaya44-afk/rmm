package alerts

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

func (r *Repository) List(page, limit int, status, severity, deviceID string) ([]Alert, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if status != "" {
		where += fmt.Sprintf(" AND a.status = $%d", idx)
		args = append(args, status)
		idx++
	}
	if severity != "" {
		where += fmt.Sprintf(" AND a.severity = $%d", idx)
		args = append(args, severity)
		idx++
	}
	if deviceID != "" {
		where += fmt.Sprintf(" AND a.device_id = $%d", idx)
		args = append(args, deviceID)
		idx++
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM alerts a %s", where)
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query := fmt.Sprintf(`SELECT a.*, d.hostname as device_name FROM alerts a 
		LEFT JOIN devices d ON d.id = a.device_id 
		%s ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d`, where, idx, idx+1)
	args = append(args, limit, offset)

	var alerts []Alert
	if err := r.db.Select(&alerts, query, args...); err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

func (r *Repository) Create(deviceID, alertType, severity, title string, message *string, metricValue, thresholdValue *float64) (*Alert, error) {
	id := uuid.New().String()
	_, err := r.db.Exec(
		`INSERT INTO alerts (id, device_id, type, severity, title, message, metric_value, threshold_value, status) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, deviceID, alertType, severity, title, message, metricValue, thresholdValue, StatusOpen,
	)
	if err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *Repository) GetByID(id string) (*Alert, error) {
	var alert Alert
	err := r.db.Get(&alert,
		`SELECT a.*, d.hostname as device_name FROM alerts a 
		LEFT JOIN devices d ON d.id = a.device_id WHERE a.id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("alert not found: %w", err)
	}
	return &alert, nil
}

func (r *Repository) Acknowledge(id, userID string) error {
	_, err := r.db.Exec(
		`UPDATE alerts SET status = $1, acknowledged_at = NOW(), acknowledged_by = $2 WHERE id = $3`,
		StatusAcknowledged, userID, id,
	)
	return err
}

func (r *Repository) Resolve(id, userID, note string) error {
	_, err := r.db.Exec(
		`UPDATE alerts SET status = $1, resolved_at = NOW(), resolved_by = $2, resolution_note = $3 WHERE id = $4`,
		StatusResolved, userID, note, id,
	)
	return err
}

func (r *Repository) GetActiveAlertsByDevice(deviceID string) ([]Alert, error) {
	var alerts []Alert
	err := r.db.Select(&alerts,
		`SELECT * FROM alerts WHERE device_id = $1 AND status IN ($2, $3) 
		ORDER BY created_at DESC`,
		deviceID, StatusOpen, StatusAcknowledged,
	)
	return alerts, err
}

func (r *Repository) CheckAndCreateAlerts(deviceID string, cpu, ram, disk *float64, online bool, posRunning, mssqlRunning *bool) ([]*Alert, error) {
	var created []*Alert

	thresholds := []struct {
		alertType string
		severity  string
		title     string
		value     *float64
		threshold float64
		check     func() bool
	}{
		{TypeCPUHigh, SeverityCritical, "CPU usage exceeds 90%% threshold", cpu, 90, func() bool {
			return cpu != nil && *cpu > 90
		}},
		{TypeRAMHigh, SeverityWarning, "RAM usage exceeds 90%% threshold", ram, 90, func() bool {
			return ram != nil && *ram > 90
		}},
		{TypeDiskHigh, SeverityWarning, "Disk usage exceeds 90%% threshold", disk, 90, func() bool {
			return disk != nil && *disk > 90
		}},
	}

	for _, t := range thresholds {
		if !t.check() {
			continue
		}

		exists, err := r.hasActiveAlert(deviceID, t.alertType)
		if err != nil || exists {
			continue
		}

		msg := fmt.Sprintf("%s: %.1f%% (threshold: %.0f%%)", t.title, *t.value, t.threshold)
		msgPtr := &msg
		alert, err := r.Create(deviceID, t.alertType, t.severity, t.title, msgPtr, t.value, &t.threshold)
		if err == nil {
			created = append(created, alert)
		}
	}

	if !online {
		exists, _ := r.hasActiveAlert(deviceID, TypeDeviceOffline)
		if !exists {
			title := "Device is offline"
			alert, err := r.Create(deviceID, TypeDeviceOffline, SeverityCritical, title, nil, nil, nil)
			if err == nil {
				created = append(created, alert)
			}
		}
	}

	return created, nil
}

func (r *Repository) hasActiveAlert(deviceID, alertType string) (bool, error) {
	var count int
	err := r.db.Get(&count,
		`SELECT COUNT(*) FROM alerts 
		WHERE device_id = $1 AND type = $2 AND status IN ($3, $4)`,
		deviceID, alertType, StatusOpen, StatusAcknowledged,
	)
	return count > 0, err
}

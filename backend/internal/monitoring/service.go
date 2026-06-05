package monitoring

import (
	"github.com/jmoiron/sqlx"
	"github.com/rmm-platform/backend/internal/devices"
)

type Service struct {
	deviceRepo *devices.Repository
}

func NewService(db *sqlx.DB) *Service {
	return &Service{deviceRepo: devices.NewRepository(db)}
}

type HealthScore struct {
	DeviceID     string  `json:"device_id"`
	HealthScore  float64 `json:"health_score"`
	CPUHealth    float64 `json:"cpu_health"`
	RAMHealth    float64 `json:"ram_health"`
	DiskHealth   float64 `json:"disk_health"`
	OnlineHealth float64 `json:"online_health"`
	POSHealth    float64 `json:"pos_health"`
	MSSQLHealth  float64 `json:"mssql_health"`
	Status       string  `json:"status"`
}

func (s *Service) CalculateHealthScore(deviceID string) (*HealthScore, error) {
	device, err := s.deviceRepo.GetByID(deviceID)
	if err != nil {
		return nil, err
	}

	metrics, err := s.deviceRepo.GetMetrics(deviceID, 1)
	if err != nil || len(metrics) == 0 {
		return &HealthScore{DeviceID: deviceID, HealthScore: 50, Status: "unknown"}, nil
	}

	m := metrics[0]
	score := &HealthScore{DeviceID: deviceID}

	if m.CPUPercent != nil {
		cpu := 100.0 - *m.CPUPercent
		if cpu < 0 { cpu = 0 }
		score.CPUHealth = cpu
	} else { score.CPUHealth = 100 }

	if m.RAMPercent != nil {
		ram := 100.0 - *m.RAMPercent
		if ram < 0 { ram = 0 }
		score.RAMHealth = ram
	} else { score.RAMHealth = 100 }

	if m.DiskPercent != nil {
		disk := 100.0 - *m.DiskPercent
		if disk < 0 { disk = 0 }
		score.DiskHealth = disk
	} else { score.DiskHealth = 100 }

	if device.IsOnline { score.OnlineHealth = 100 }

	if m.POSRunning != nil && *m.POSRunning { score.POSHealth = 100 }
	if m.MSSQLRunning != nil && *m.MSSQLRunning { score.MSSQLHealth = 100 }

	score.HealthScore = score.CPUHealth*0.2 + score.RAMHealth*0.2 +
		score.DiskHealth*0.15 + score.OnlineHealth*0.25 +
		score.POSHealth*0.1 + score.MSSQLHealth*0.1

	switch {
	case score.HealthScore >= 80:
		score.Status = "healthy"
	case score.HealthScore >= 50:
		score.Status = "warning"
	default:
		score.Status = "critical"
	}

	return score, nil
}

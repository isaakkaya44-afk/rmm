package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Agent    AgentConfig    `yaml:"agent"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	BaseURL     string `yaml:"base_url"`
	APIKey      string `yaml:"api_key"`
	HeartbeatEndpoint string `yaml:"heartbeat_endpoint"`
	Timeout     int    `yaml:"timeout"`
}

type AgentConfig struct {
	Hostname          string `yaml:"hostname"`
	Interval          int    `yaml:"interval"`
	OfflineQueueSize  int    `yaml:"offline_queue_size"`
	RustDeskPath      string `yaml:"rustdesk_path"`
	ScreenshotEnabled bool   `yaml:"screenshot_enabled"`
}

type MonitorConfig struct {
	POSProcesses     []string `yaml:"pos_processes"`
	CriticalServices []string `yaml:"critical_services"`
	MSSQLServices    []string `yaml:"mssql_services"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	File   string `yaml:"file"`
	MaxAge int    `yaml:"max_age_days"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return applyDefaults(&cfg), nil
}

func applyDefaults(cfg *Config) *Config {
	if cfg.Agent.Interval <= 0 {
		cfg.Agent.Interval = 30
	}
	if cfg.Agent.OfflineQueueSize <= 0 {
		cfg.Agent.OfflineQueueSize = 100
	}
	if cfg.Server.Timeout <= 0 {
		cfg.Server.Timeout = 10
	}
	if cfg.Monitor.POSProcesses == nil {
		cfg.Monitor.POSProcesses = []string{"pos.exe", "restaurant_pos.exe"}
	}
	if cfg.Monitor.CriticalServices == nil {
		cfg.Monitor.CriticalServices = []string{"MpsSvc", "LanmanServer"}
	}
	if cfg.Monitor.MSSQLServices == nil {
		cfg.Monitor.MSSQLServices = []string{"MSSQLSERVER", "MSSQL$SQLEXPRESS"}
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.File == "" {
		cfg.Logging.File = "C:\\ProgramData\\RMMAgent\\agent.log"
	}
	if cfg.Logging.MaxAge <= 0 {
		cfg.Logging.MaxAge = 7
	}
	return cfg
}

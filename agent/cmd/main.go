package main

import (
	"log"
	"os"
	"time"

	"github.com/rmm-platform/agent/core/collector"
	"github.com/rmm-platform/agent/core/config"
	"github.com/rmm-platform/agent/core/monitor"
	"github.com/rmm-platform/agent/core/network"
	"github.com/rmm-platform/agent/core/remote"
	"github.com/rmm-platform/agent/service"
	"github.com/rmm-platform/agent/utils"
	"golang.org/x/sys/windows/svc"
)

var (
	version = "1.0.0"
	cfgPath = "C:\\ProgramData\\RMMAgent\\config.yaml"
)

func getConfigPath() string {
	for i, arg := range os.Args {
		if arg == "--config" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return cfgPath
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "install":
			cfgPath = getConfigPath()
			exe, _ := os.Executable()
			if err := service.InstallService("RMMAgent", "RMM Monitoring Agent", exe); err != nil {
				log.Fatalf("installation failed: %v", err)
			}
			log.Println("RMMAgent installed successfully")
			return
		case "remove":
			if err := service.RemoveService("RMMAgent"); err != nil {
				log.Fatalf("removal failed: %v", err)
			}
			log.Println("RMMAgent removed successfully")
			return
		case "run":
			cfgPath = getConfigPath()
			runAgent()
			return
		}
	}

	svc.Run("RMMAgent", service.NewAgentService())
}

func runAgent() {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	sysCollector := collector.New()
	posMonitor := monitor.NewPOSMonitor(
		cfg.Monitor.POSProcesses,
		cfg.Monitor.CriticalServices,
		cfg.Monitor.MSSQLServices,
	)
	hbClient := network.NewHeartbeatClient(
		cfg.Server.BaseURL,
		cfg.Server.APIKey,
		cfg.Server.Timeout,
		cfg.Agent.OfflineQueueSize,
	)
	rdDetector := remote.NewDetector(cfg.Agent.RustDeskPath)

	ticker := time.NewTicker(time.Duration(cfg.Agent.Interval) * time.Second)
	defer ticker.Stop()

	flushTicker := time.NewTicker(5 * time.Minute)
	defer flushTicker.Stop()

	log.Printf("RMM Agent v%s started (interval: %ds)", version, cfg.Agent.Interval)

	for {
		select {
		case <-ticker.C:
			sendHeartbeat(sysCollector, posMonitor, rdDetector, hbClient, cfg)
		case <-flushTicker.C:
			sent := hbClient.FlushQueue()
			if sent > 0 {
				log.Printf("offline queue flushed: %d messages", sent)
			}
		}
	}
}

func sendHeartbeat(sysCollector *collector.Collector, posMonitor *monitor.POSMonitor,
	rdDetector *remote.Detector, hbClient *network.HeartbeatClient, cfg *config.Config) {

	sysInfo, err := sysCollector.Collect()
	if err != nil {
		log.Printf("collector error: %v", err)
		return
	}

	posStatus := posMonitor.Check()
	rdInfo := rdDetector.Detect()

	data := map[string]interface{}{
		"hostname":       sysInfo.Hostname,
		"os_version":     sysInfo.OSVersion,
		"cpu_percent":    sysInfo.CPUPercent,
		"ram_percent":    sysInfo.RAMPercent,
		"ram_used_mb":    sysInfo.RAMUsedMB,
		"ram_total_mb":   sysInfo.RAMTotalMB,
		"disk_percent":   sysInfo.DiskPercent,
		"disk_used_mb":   sysInfo.DiskUsedMB,
		"disk_total_mb":  sysInfo.DiskTotalMB,
		"uptime_seconds": sysInfo.UptimeSeconds,
		"cpu_model":      sysInfo.CPUModel,
		"cpu_cores":      sysInfo.CPUCores,
		"pos_running":    posStatus.POSRunning,
		"mssql_running":  posStatus.MSSQLRunning,
		"rustdesk_id":    rdInfo.ID,
		"agent_version":  version,
	}

	if cfg.Agent.ScreenshotEnabled {
		if ss, err := utils.CaptureCompressed(); err == nil {
			data["screenshot"] = ss.Base64
		}
	}

	if err := hbClient.SendHeartbeat(data); err != nil {
		log.Printf("heartbeat error: %v", err)
	} else {
		log.Printf("heartbeat sent: %s", sysInfo.Hostname)
	}
}

package monitor

import (
	"fmt"
	"os/exec"
	"strings"
)

type POSMonitor struct {
	posProcesses     []string
	criticalServices []string
	mssqlServices    []string
}

type POSStatus struct {
	POSRunning      bool            `json:"pos_running"`
	MSSQLRunning    bool            `json:"mssql_running"`
	ServicesRunning map[string]bool `json:"services_running"`
	POSProcesses    map[string]bool `json:"pos_processes"`
}

func NewPOSMonitor(posProcesses, criticalServices, mssqlServices []string) *POSMonitor {
	return &POSMonitor{
		posProcesses:     posProcesses,
		criticalServices: criticalServices,
		mssqlServices:    mssqlServices,
	}
}

func (m *POSMonitor) Check() *POSStatus {
	status := &POSStatus{
		ServicesRunning: make(map[string]bool),
		POSProcesses:    make(map[string]bool),
	}

	for _, proc := range m.posProcesses {
		status.POSProcesses[proc] = isProcessRunning(proc)
		if status.POSProcesses[proc] {
			status.POSRunning = true
		}
	}

	for _, s := range m.mssqlServices {
		running := isServiceRunning(s)
		status.ServicesRunning[s] = running
		if running {
			status.MSSQLRunning = true
		}
	}

	for _, s := range m.criticalServices {
		status.ServicesRunning[s] = isServiceRunning(s)
	}

	return status
}

func isProcessRunning(name string) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", name))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), name)
}

func isServiceRunning(name string) bool {
	cmd := exec.Command("sc", "query", name)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "RUNNING")
}

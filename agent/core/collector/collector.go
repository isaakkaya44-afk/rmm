package collector

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/load"
)

type SystemInfo struct {
	Hostname      string  `json:"hostname"`
	OSVersion     string  `json:"os_version"`
	CPUModel      string  `json:"cpu_model"`
	CPUCores      int     `json:"cpu_cores"`
	CPUPercent    float64 `json:"cpu_percent"`
	RAMTotalMB    int64   `json:"ram_total_mb"`
	RAMUsedMB     int64   `json:"ram_used_mb"`
	RAMPercent    float64 `json:"ram_percent"`
	DiskTotalMB   int64   `json:"disk_total_mb"`
	DiskUsedMB    int64   `json:"disk_used_mb"`
	DiskPercent   float64 `json:"disk_percent"`
	UptimeSeconds int64   `json:"uptime_seconds"`
}

type Collector struct {
	lastCPUStats []cpu.InfoStat
}

func New() *Collector {
	return &Collector{}
}

func (c *Collector) Collect() (*SystemInfo, error) {
	info := &SystemInfo{}

	hostInfo, err := host.Info()
	if err == nil {
		info.Hostname = hostInfo.Hostname
		info.OSVersion = hostInfo.OS + " " + hostInfo.PlatformVersion
		info.UptimeSeconds = int64(hostInfo.Uptime)
	}

	if runtime.GOOS == "windows" {
		cpuPercent, err := cpu.Percent(500*time.Millisecond, false)
		if err == nil && len(cpuPercent) > 0 {
			info.CPUPercent = cpuPercent[0]
		}

		cpuInfo, err := cpu.Info()
		if err == nil && len(cpuInfo) > 0 {
			info.CPUModel = cpuInfo[0].ModelName
		}
		cpuCount, _ := cpu.Counts(true)
		info.CPUCores = cpuCount
	} else {
		avg, err := load.Avg()
		if err == nil {
			cpuCount, _ := cpu.Counts(true)
			if cpuCount > 0 {
				info.CPUPercent = (avg.Load1 / float64(cpuCount)) * 100
			}
		}
	}

	memInfo, err := mem.VirtualMemory()
	if err == nil {
		info.RAMTotalMB = int64(memInfo.Total / 1024 / 1024)
		info.RAMUsedMB = int64(memInfo.Used / 1024 / 1024)
		info.RAMPercent = memInfo.UsedPercent
	}

	diskInfo, err := disk.Usage("C:\\")
	if err == nil {
		info.DiskTotalMB = int64(diskInfo.Total / 1024 / 1024)
		info.DiskUsedMB = int64(diskInfo.Used / 1024 / 1024)
		info.DiskPercent = diskInfo.UsedPercent
	}

	return info, nil
}

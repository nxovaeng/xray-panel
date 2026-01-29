package system

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemInfo contains system information
type SystemInfo struct {
	// CPU
	CPUModel   string  `json:"cpu_model"`
	CPUCores   int     `json:"cpu_cores"`
	CPUUsage   float64 `json:"cpu_usage"`
	
	// Memory
	MemTotal   uint64  `json:"mem_total"`
	MemUsed    uint64  `json:"mem_used"`
	MemFree    uint64  `json:"mem_free"`
	MemPercent float64 `json:"mem_percent"`
	
	// Disk
	DiskTotal   uint64  `json:"disk_total"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskFree    uint64  `json:"disk_free"`
	DiskPercent float64 `json:"disk_percent"`
	
	// Network
	NetBytesSent uint64 `json:"net_bytes_sent"`
	NetBytesRecv uint64 `json:"net_bytes_recv"`
	
	// System
	Hostname   string        `json:"hostname"`
	OS         string        `json:"os"`
	Platform   string        `json:"platform"`
	Uptime     time.Duration `json:"uptime"`
	GoVersion  string        `json:"go_version"`
}

// GetSystemInfo collects system information
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		GoVersion: runtime.Version(),
	}

	// CPU Info
	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		info.CPUModel = cpuInfo[0].ModelName
	}
	info.CPUCores = runtime.NumCPU()
	
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		info.CPUUsage = cpuPercent[0]
	}

	// Memory Info
	if memInfo, err := mem.VirtualMemory(); err == nil {
		info.MemTotal = memInfo.Total
		info.MemUsed = memInfo.Used
		info.MemFree = memInfo.Free
		info.MemPercent = memInfo.UsedPercent
	}

	// Disk Info
	if diskInfo, err := disk.Usage("/"); err == nil {
		info.DiskTotal = diskInfo.Total
		info.DiskUsed = diskInfo.Used
		info.DiskFree = diskInfo.Free
		info.DiskPercent = diskInfo.UsedPercent
	}

	// Network Info
	if netIO, err := net.IOCounters(false); err == nil && len(netIO) > 0 {
		info.NetBytesSent = netIO[0].BytesSent
		info.NetBytesRecv = netIO[0].BytesRecv
	}

	// Host Info
	if hostInfo, err := host.Info(); err == nil {
		info.Hostname = hostInfo.Hostname
		info.OS = hostInfo.OS
		info.Platform = hostInfo.Platform
		info.Uptime = time.Duration(hostInfo.Uptime) * time.Second
	}

	// Fallback for hostname
	if info.Hostname == "" {
		if hostname, err := os.Hostname(); err == nil {
			info.Hostname = hostname
		}
	}

	return info, nil
}

// FormatBytes formats bytes to human readable string
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats duration to human readable string
func FormatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d天 %d小时", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%d小时 %d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

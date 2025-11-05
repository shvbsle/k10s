package tui

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemStats holds CPU and memory usage statistics
type SystemStats struct {
	CPUPercent float64
	MemPercent float64
	MemUsedMB  uint64
	MemTotalMB uint64
}

// GetSystemStats retrieves current CPU and memory usage
func GetSystemStats() (*SystemStats, error) {
	// Get CPU usage (average over 100ms)
	cpuPercent, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		return nil, err
	}

	// Get memory usage
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	stats := &SystemStats{
		MemPercent: memInfo.UsedPercent,
		MemUsedMB:  memInfo.Used / (1024 * 1024),
		MemTotalMB: memInfo.Total / (1024 * 1024),
	}

	// cpuPercent returns slice, take first value (overall CPU)
	if len(cpuPercent) > 0 {
		stats.CPUPercent = cpuPercent[0]
	}

	return stats, nil
}

// FormatCPU formats CPU percentage for display
func (s *SystemStats) FormatCPU() string {
	return fmt.Sprintf("%.1f%%", s.CPUPercent)
}

// FormatMemory formats memory usage for display
func (s *SystemStats) FormatMemory() string {
	return fmt.Sprintf("%.1f%% (%dMB/%dMB)", s.MemPercent, s.MemUsedMB, s.MemTotalMB)
}

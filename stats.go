// +build linux

package cgroupManager

type ThrottlingData struct {
	Periods uint64 `json:"periods,omitempty"`
	ThrottledPeriods uint64 `json:"throttled_periods,omitempty"`
	ThrottledTime uint64 `json:"throttled_time,omitempty"`
}

type CpuUsage struct {
	TotalUsage uint64 `json:"total_usage,omitempty"`
	PercpuUsage []uint64 `json:"percpu_usage,omitempty"`
	PercpuUsageInKernelmode []uint64 `json:"percpu_usage_in_kernelmode"`
	PercpuUsageInUsermode []uint64 `json:"percpu_usage_in_usermode"`
	UsageInKernelmode uint64 `json:"usage_in_kernelmode"`
	UsageInUsermode uint64 `json:"usage_in_usermode"`
}

type CpuStats struct {
	CpuUsage       CpuUsage       `json:"cpu_usage,omitempty"`
	ThrottlingData ThrottlingData `json:"throttling_data,omitempty"`
}

type MemoryData struct {
	Usage    uint64 `json:"usage,omitempty"`
	MaxUsage uint64 `json:"max_usage,omitempty"`
	Failcnt  uint64 `json:"failcnt"`
	Limit    uint64 `json:"limit"`
}

type MemoryStats struct {
	Cache uint64 `json:"cache,omitempty"`
	Usage MemoryData `json:"usage,omitempty"`
	SwapUsage MemoryData `json:"swap_usage,omitempty"`
	KernelUsage MemoryData `json:"kernel_usage,omitempty"`
	KernelTCPUsage MemoryData `json:"kernel_tcp_usage,omitempty"`
	UseHierarchy bool `json:"use_hierarchy"`
	Stats map[string]uint64 `json:"stats,omitempty"`
}

type Stats struct {
	CpuStats    CpuStats    `json:"cpu_stats,omitempty"`
	MemoryStats MemoryStats `json:"memory_stats,omitempty"`
}

func NewStats() *Stats {
	memoryStats := MemoryStats{Stats: make(map[string]uint64)}
	return &Stats{MemoryStats: memoryStats}
}

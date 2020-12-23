package cgroupManager

type FreezerState string

const (
        Undefined FreezerState = ""
        Frozen    FreezerState = "FROZEN"
        Thawed    FreezerState = "THAWED"
)

type CgroupConfig struct {
        Name string `json:"name,omitempty"`
        Parent string `json:"parent,omitempty"`
        Path string `json:"path,omitempty"`
        ScopePrefix string `json:"scope_prefix,omitempty"`
        Paths map[string]string `json:"paths,omitempty"`
        *Resources
}

type Resources struct {
        CpuShares uint64 `json:"cpu_shares"`
        CpuQuota int64 `json:"cpu_quota"`
        CpuPeriod uint64 `json:"cpu_period"`
        CpuRtRuntime int64 `json:"cpu_rt_quota"`
        CpuRtPeriod uint64 `json:"cpu_rt_period"`
        CpusetCpus string `json:"cpuset_cpus"`
        CpusetMems string `json:"cpuset_mems"`
        Freezer FreezerState `json:"freezer"`
        CpuWeight uint64 `json:"cpu_weight"`
}

type Config struct {
	Cgroups *CgroupConfig `json:"cgroups"`
	Version string `json:"version"`
}

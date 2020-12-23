// +build linux

package cgroupManager

type Manager interface {
	Apply(pid int) error
	GetPids() ([]int, error)
	GetAllPids() ([]int, error)
	GetStats() (*Stats, error)
	Freeze(state FreezerState) error
	Destroy() error
	Path(string) string
	Set(container *Config) error
	GetPaths() map[string]string
	GetCgroups() (*CgroupConfig, error)
	GetFreezerState() (FreezerState, error)
	Exists() bool
}

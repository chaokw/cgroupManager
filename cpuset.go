// +build linux

package cgroupManager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moby/sys/mountinfo"
	"github.com/pkg/errors"
)

type CpusetGroup struct {
	Config     *CgroupConfig
	CgroupPath string
}

func NewCpuSetCgroup(path string) *CpusetGroup {
	c := &CgroupConfig{
		Resources: &Resources{},
	}
	root, err := getCgroupRoot()
	if err != nil {
		fmt.Printf("couldn't get cgroup root: %v", err)
	}
	subsystemPath := filepath.Join(root, "cpuset")
	if err != nil {
		fmt.Println(err)
	}
	actualPath := filepath.Join(subsystemPath, path)
	if err != nil {
		fmt.Println(err)
	}
	err = os.MkdirAll(actualPath, 0755)
	if err != nil {
		fmt.Println(err)
	}
	return &CpusetGroup{Config: c, CgroupPath: actualPath}
}

func (s *CpusetGroup) Name() string {
	return "cpuset"
}

func (s *CpusetGroup) Apply(path string, d *cgroupData) error {
	return s.ApplyDir(path, d.config, d.pid)
}

func (s *CpusetGroup) AddPid(path string, pid int) error {
	return WriteCgroupProc(path, pid)
}

func (s *CpusetGroup) Set(path string, cgroup *CgroupConfig) error {
	if cgroup.Resources.CpusetCpus != "" {
		if err := WriteFile(path, "cpuset.cpus", cgroup.Resources.CpusetCpus); err != nil {
			return err
		}
	}
	if cgroup.Resources.CpusetMems != "" {
		if err := WriteFile(path, "cpuset.mems", cgroup.Resources.CpusetMems); err != nil {
			return err
		}
	}
	return nil
}

func (s *CpusetGroup) GetStats(path string, stats *Stats) error {
	return nil
}

// Get the source mount point of directory passed in as argument.
func getMount(dir string) (string, error) {
	mi, err := mountinfo.GetMounts(mountinfo.ParentsFilter(dir))
	if err != nil {
		return "", err
	}
	if len(mi) < 1 {
		return "", errors.Errorf("Can't find mount point of %s", dir)
	}

	var idx, maxlen int
	for i := range mi {
		if len(mi[i].Mountpoint) > maxlen {
			maxlen = len(mi[i].Mountpoint)
			idx = i
		}
	}

	return mi[idx].Mountpoint, nil
}

func (s *CpusetGroup) ApplyDir(dir string, cgroup *CgroupConfig, pid int) error {
	if dir == "" {
		return nil
	}
	root, err := getMount(dir)
	if err != nil {
		return err
	}
	root = filepath.Dir(root)
	if err := cpusetEnsureParent(filepath.Dir(dir), root); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := s.ensureCpusAndMems(dir, cgroup); err != nil {
		return err
	}

	return WriteCgroupProc(dir, pid)
}

func getCpusetSubsystemSettings(parent string) (cpus, mems string, err error) {
	if cpus, err = ReadFile(parent, "cpuset.cpus"); err != nil {
		return
	}
	if mems, err = ReadFile(parent, "cpuset.mems"); err != nil {
		return
	}
	return cpus, mems, nil
}

// cpusetEnsureParent makes sure that the parent directory of current is created
// and populated with the proper cpus and mems files copied from
// its parent.
func cpusetEnsureParent(current, root string) error {
	parent := filepath.Dir(current)
	if CleanPath(parent) == root {
		return nil
	}
	if parent == current {
		return errors.New("cpuset: cgroup parent path outside cgroup root")
	}
	if err := cpusetEnsureParent(parent, root); err != nil {
		return err
	}
	if err := os.MkdirAll(current, 0755); err != nil {
		return err
	}
	return cpusetCopyIfNeeded(current, parent)
}

// cpusetCopyIfNeeded copies the cpuset.cpus and cpuset.mems from the parent
// directory to the current directory if the file's contents are 0
func cpusetCopyIfNeeded(current, parent string) error {
	currentCpus, currentMems, err := getCpusetSubsystemSettings(current)
	if err != nil {
		return err
	}
	parentCpus, parentMems, err := getCpusetSubsystemSettings(parent)
	if err != nil {
		return err
	}

	if isEmptyCpuset(currentCpus) {
		if err := WriteFile(current, "cpuset.cpus", string(parentCpus)); err != nil {
			return err
		}
	}
	if isEmptyCpuset(currentMems) {
		if err := WriteFile(current, "cpuset.mems", string(parentMems)); err != nil {
			return err
		}
	}
	return nil
}

func isEmptyCpuset(str string) bool {
	return str == "" || str == "\n"
}

func (s *CpusetGroup) ensureCpusAndMems(path string, cgroup *CgroupConfig) error {
	if err := s.Set(path, cgroup); err != nil {
		return err
	}
	return cpusetCopyIfNeeded(path, filepath.Dir(path))
}

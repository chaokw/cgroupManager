// +build linux

package subsystem

import (
	"os"
	"path/filepath"

	"github.com/moby/sys/mountinfo"
	cgroups "cgroupManager"
	"github.com/pkg/errors"
)

type CpusetGroup struct {
}

func (s *CpusetGroup) Name() string {
	return "cpuset"
}

func (s *CpusetGroup) Apply(path string, d *cgroupData) error {
	return s.ApplyDir(path, d.config, d.pid)
}

func (s *CpusetGroup) AddPid(path string, pid int) error {
        return cgroups.WriteCgroupProc(path, pid)
}

func (s *CpusetGroup) Set(path string, cgroup *cgroups.CgroupConfig) error {
	if cgroup.Resources.CpusetCpus != "" {
		if err := cgroups.WriteFile(path, "cpuset.cpus", cgroup.Resources.CpusetCpus); err != nil {
			return err
		}
	}
	if cgroup.Resources.CpusetMems != "" {
		if err := cgroups.WriteFile(path, "cpuset.mems", cgroup.Resources.CpusetMems); err != nil {
			return err
		}
	}
	return nil
}

func (s *CpusetGroup) GetStats(path string, stats *cgroups.Stats) error {
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

func (s *CpusetGroup) ApplyDir(dir string, cgroup *cgroups.CgroupConfig, pid int) error {
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

	return cgroups.WriteCgroupProc(dir, pid)
}

func getCpusetSubsystemSettings(parent string) (cpus, mems string, err error) {
	if cpus, err = cgroups.ReadFile(parent, "cpuset.cpus"); err != nil {
		return
	}
	if mems, err = cgroups.ReadFile(parent, "cpuset.mems"); err != nil {
		return
	}
	return cpus, mems, nil
}

// cpusetEnsureParent makes sure that the parent directory of current is created
// and populated with the proper cpus and mems files copied from
// its parent.
func cpusetEnsureParent(current, root string) error {
	parent := filepath.Dir(current)
	if cgroups.CleanPath(parent) == root {
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
		if err := cgroups.WriteFile(current, "cpuset.cpus", string(parentCpus)); err != nil {
			return err
		}
	}
	if isEmptyCpuset(currentMems) {
		if err := cgroups.WriteFile(current, "cpuset.mems", string(parentMems)); err != nil {
			return err
		}
	}
	return nil
}

func isEmptyCpuset(str string) bool {
	return str == "" || str == "\n"
}

func (s *CpusetGroup) ensureCpusAndMems(path string, cgroup *cgroups.CgroupConfig) error {
	if err := s.Set(path, cgroup); err != nil {
		return err
	}
	return cpusetCopyIfNeeded(path, filepath.Dir(path))
}

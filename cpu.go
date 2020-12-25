// +build linux

package cgroupManager

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"path/filepath"
)

type CpuGroup struct {
	Config    *CgroupConfig
        CgroupPath string
}

func NewCpuCgroup(path string) *CpuGroup {
	c := &CgroupConfig{
                Resources: &Resources{},
        }
        root, err := getCgroupRoot()
        if err != nil {
                fmt.Printf("couldn't get cgroup root: %v", err)
        }
	subsystemPath := filepath.Join(root, "cpu")
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
	return &CpuGroup{Config: c, CgroupPath: actualPath}
}

func (s *CpuGroup) Name() string {
	return "cpu"
}

func (s *CpuGroup) Apply(path string, d *cgroupData) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	if err := s.SetRtSched(path, d.config); err != nil {
		return err
	}
	return WriteCgroupProc(path, d.pid)
}

func (s *CpuGroup) AddPid(path string, pid int) error {
	return WriteCgroupProc(path, pid)
}

func (s *CpuGroup) SetRtSched(path string, cgroup *CgroupConfig) error {
	if cgroup.Resources.CpuRtPeriod != 0 {
		if err := WriteFile(path, "cpu.rt_period_us", strconv.FormatUint(cgroup.Resources.CpuRtPeriod, 10)); err != nil {
			return err
		}
	}
	if cgroup.Resources.CpuRtRuntime != 0 {
		if err := WriteFile(path, "cpu.rt_runtime_us", strconv.FormatInt(cgroup.Resources.CpuRtRuntime, 10)); err != nil {
			return err
		}
	}
	return nil
}

func (s *CpuGroup) Set(path string, cgroup *CgroupConfig) error {
	if cgroup.Resources.CpuShares != 0 {
		shares := cgroup.Resources.CpuShares
		if err := WriteFile(path, "cpu.shares", strconv.FormatUint(shares, 10)); err != nil {
			return err
		}
		sharesRead, err := GetCgroupParamUint(path, "cpu.shares")
		if err != nil {
			return err
		}
		if shares > sharesRead {
			return fmt.Errorf("the maximum allowed cpu-shares is %d", sharesRead)
		} else if shares < sharesRead {
			return fmt.Errorf("the minimum allowed cpu-shares is %d", sharesRead)
		}
	}
	if cgroup.Resources.CpuPeriod != 0 {
		if err := WriteFile(path, "cpu.cfs_period_us", strconv.FormatUint(cgroup.Resources.CpuPeriod, 10)); err != nil {
			return err
		}
	}
	if cgroup.Resources.CpuQuota != 0 {
		if err := WriteFile(path, "cpu.cfs_quota_us", strconv.FormatInt(cgroup.Resources.CpuQuota, 10)); err != nil {
			return err
		}
	}
	return s.SetRtSched(path, cgroup)
}

func (s *CpuGroup) GetStats(path string, stats *Stats) error {
	f, err := OpenFile(path, "cpu.stat", os.O_RDONLY)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t, v, err := GetCgroupParamKeyValue(sc.Text())
		if err != nil {
			return err
		}
		switch t {
		case "nr_periods":
			stats.CpuStats.ThrottlingData.Periods = v

		case "nr_throttled":
			stats.CpuStats.ThrottlingData.ThrottledPeriods = v

		case "throttled_time":
			stats.CpuStats.ThrottlingData.ThrottledTime = v
		}
	}
	return nil
}

func (s *CpuGroup) Cleanup() {
        os.RemoveAll(s.CgroupPath)
}

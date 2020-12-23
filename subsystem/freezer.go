// +build linux

package subsystem

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	cgroups "cgroupManager"
	"golang.org/x/sys/unix"
)

type FreezerGroup struct {
}

func (s *FreezerGroup) Name() string {
	return "freezer"
}

func (s *FreezerGroup) Apply(path string, d *cgroupData) error {
	return join(path, d.pid)
}

func (s *FreezerGroup) AddPid(path string, pid int) error {
        return cgroups.WriteCgroupProc(path, pid)
}

func (s *FreezerGroup) Set(path string, cgroup *cgroups.CgroupConfig) error {
	switch cgroup.Resources.Freezer {
	case cgroups.Frozen, cgroups.Thawed:
		for {
			if err := cgroups.WriteFile(path, "freezer.state", string(cgroup.Resources.Freezer)); err != nil {
				return err
			}

			state, err := s.GetState(path)
			if err != nil {
				return err
			}
			if state == cgroup.Resources.Freezer {
				break
			}

			time.Sleep(1 * time.Millisecond)
		}
	case cgroups.Undefined:
		return nil
	default:
		return fmt.Errorf("Invalid argument '%s' to freezer.state", string(cgroup.Resources.Freezer))
	}

	return nil
}

func (s *FreezerGroup) GetStats(path string, stats *cgroups.Stats) error {
	return nil
}

func (s *FreezerGroup) GetState(path string) (cgroups.FreezerState, error) {
	for {
		state, err := cgroups.ReadFile(path, "freezer.state")
		if err != nil {
			if os.IsNotExist(err) || errors.Is(err, unix.ENODEV) {
				err = nil
			}
			return cgroups.Undefined, err
		}
		switch strings.TrimSpace(state) {
		case "THAWED":
			return cgroups.Thawed, nil
		case "FROZEN":
			return cgroups.Frozen, nil
		case "FREEZING":
			time.Sleep(1 * time.Millisecond)
			continue
		default:
			return cgroups.Undefined, fmt.Errorf("unknown freezer.state %q", state)
		}
	}
}

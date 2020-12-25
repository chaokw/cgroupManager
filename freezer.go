// +build linux

package cgroupManager

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FreezerGroup struct {
	Config     *CgroupConfig
	CgroupPath string
}

func NewFreezerCgroup(path string) *FreezerGroup {
	c := &CgroupConfig{
		Resources: &Resources{},
	}
	root, err := getCgroupRoot()
	if err != nil {
		fmt.Printf("couldn't get cgroup root: %v", err)
	}
	subsystemPath := filepath.Join(root, "freezer")
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
	return &FreezerGroup{Config: c, CgroupPath: actualPath}
}

func (s *FreezerGroup) Name() string {
	return "freezer"
}

func (s *FreezerGroup) Apply(path string, d *cgroupData) error {
	return join(path, d.pid)
}

func (s *FreezerGroup) AddPid(path string, pid int) error {
	return WriteCgroupProc(path, pid)
}

func (s *FreezerGroup) Set(path string, cgroup *CgroupConfig) error {
	switch cgroup.Resources.Freezer {
	case Frozen, Thawed:
		for {
			if err := WriteFile(path, "freezer.state", string(cgroup.Resources.Freezer)); err != nil {
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
	case Undefined:
		return nil
	default:
		return fmt.Errorf("Invalid argument '%s' to freezer.state", string(cgroup.Resources.Freezer))
	}

	return nil
}

func (s *FreezerGroup) GetStats(path string, stats *Stats) error {
	return nil
}

func (s *FreezerGroup) Cleanup() {
	os.RemoveAll(s.CgroupPath)
}

func (s *FreezerGroup) GetState(path string) (FreezerState, error) {
	for {
		state, err := ReadFile(path, "freezer.state")
		if err != nil {
			if os.IsNotExist(err) || errors.Is(err, unix.ENODEV) {
				err = nil
			}
			return Undefined, err
		}
		switch strings.TrimSpace(state) {
		case "THAWED":
			return Thawed, nil
		case "FROZEN":
			return Frozen, nil
		case "FREEZING":
			time.Sleep(1 * time.Millisecond)
			continue
		default:
			return Undefined, fmt.Errorf("unknown freezer.state %q", state)
		}
	}
}

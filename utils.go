// +build linux

package cgroupManager

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	CgroupProcesses   = "cgroup.procs"
	unifiedMountpoint = "/sys/fs/cgroup"
)

var (
	isUnifiedOnce sync.Once
	isUnified     bool
)

func CleanPath(path string) string {
	if path == "" {
		return ""
	}

	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		path = filepath.Clean(string(os.PathSeparator) + path)
		path, _ = filepath.Rel(string(os.PathSeparator), path)
	}
	return filepath.Clean(path)
}

func IsCgroup2UnifiedMode() bool {
	isUnifiedOnce.Do(func() {
		var st unix.Statfs_t
		err := unix.Statfs(unifiedMountpoint, &st)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.WithError(err).Debugf("%s missing, assuming cgroup v1", unifiedMountpoint)
				isUnified = false
				return
			}
			panic(fmt.Sprintf("cannot statfs cgroup root: %s", err))
		}
		isUnified = st.Type == unix.CGROUP2_SUPER_MAGIC
	})
	return isUnified
}

type Mount struct {
	Mountpoint string
	Root       string
	Subsystems []string
}

func GetCgroupMounts(all bool) ([]Mount, error) {
	if IsCgroup2UnifiedMode() {
		availableControllers, err := GetAllSubsystems()
		if err != nil {
			return nil, err
		}
		m := Mount{
			Mountpoint: unifiedMountpoint,
			Root:       unifiedMountpoint,
			Subsystems: availableControllers,
		}
		return []Mount{m}, nil
	}

	return getCgroupMountsV1(all)
}

func GetAllSubsystems() ([]string, error) {
	if IsCgroup2UnifiedMode() {
		pseudo := []string{"devices", "freezer"}
		data, err := ReadFile("/sys/fs/cgroup", "cgroup.controllers")
		if err != nil {
			return nil, err
		}
		subsystems := append(pseudo, strings.Fields(data)...)
		return subsystems, nil
	}
	f, err := os.Open("/proc/cgroups")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	subsystems := []string{}

	s := bufio.NewScanner(f)
	for s.Scan() {
		text := s.Text()
		if text[0] != '#' {
			parts := strings.Fields(text)
			if len(parts) >= 4 && parts[3] != "0" {
				subsystems = append(subsystems, parts[0])
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return subsystems, nil
}

func readProcsFile(file string) ([]int, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		s   = bufio.NewScanner(f)
		out = []int{}
	)

	for s.Scan() {
		if t := s.Text(); t != "" {
			pid, err := strconv.Atoi(t)
			if err != nil {
				return nil, err
			}
			out = append(out, pid)
		}
	}
	return out, s.Err()
}

func ParseCgroupFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseCgroupFromReader(f)
}

func parseCgroupFromReader(r io.Reader) (map[string]string, error) {
	s := bufio.NewScanner(r)
	cgroups := make(map[string]string)

	for s.Scan() {
		text := s.Text()
		parts := strings.SplitN(text, ":", 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid cgroup entry: must contain at least two colons: %v", text)
		}

		for _, subs := range strings.Split(parts[1], ",") {
			cgroups[subs] = parts[2]
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return cgroups, nil
}

func PathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func EnterPid(cgroupPaths map[string]string, pid int) error {
	for _, path := range cgroupPaths {
		if PathExists(path) {
			if err := WriteCgroupProc(path, pid); err != nil {
				return err
			}
		}
	}
	return nil
}

func rmdir(path string) error {
	err := unix.Rmdir(path)
	if err == nil || err == unix.ENOENT {
		return nil
	}
	return &os.PathError{Op: "rmdir", Path: path, Err: err}
}

func RemovePath(path string) error {
	if err := rmdir(path); err == nil {
		return nil
	}

	infos, err := ioutil.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return err
	}
	for _, info := range infos {
		if info.IsDir() {
			if err = RemovePath(filepath.Join(path, info.Name())); err != nil {
				break
			}
		}
	}
	if err == nil {
		err = rmdir(path)
	}
	return err
}

func RemovePaths(paths map[string]string) (err error) {
	const retries = 5
	delay := 10 * time.Millisecond
	for i := 0; i < retries; i++ {
		if i != 0 {
			time.Sleep(delay)
			delay *= 2
		}
		for s, p := range paths {
			if err := RemovePath(p); err != nil {
				switch i {
				case 0:
					logrus.WithError(err).Warnf("Failed to remove cgroup (will retry)")
				case retries - 1:
					logrus.WithError(err).Error("Failed to remove cgroup")
				}

			}
			_, err := os.Stat(p)
			if os.IsNotExist(err) {
				delete(paths, s)
			}
		}
		if len(paths) == 0 {
			paths = make(map[string]string)
			return nil
		}
	}
	return fmt.Errorf("Failed to remove paths: %v", paths)
}

func GetPids(dir string) ([]int, error) {
	return readProcsFile(filepath.Join(dir, CgroupProcesses))
}

func GetAllPids(path string) ([]int, error) {
	var pids []int
	err := filepath.Walk(path, func(p string, info os.FileInfo, iErr error) error {
		if iErr != nil {
			return iErr
		}
		if info.IsDir() || info.Name() != CgroupProcesses {
			return nil
		}
		cPids, err := readProcsFile(p)
		if err != nil {
			return err
		}
		pids = append(pids, cPids...)
		return nil
	})
	return pids, err
}

func WriteCgroupProc(dir string, pid int) error {
	if dir == "" {
		return fmt.Errorf("no such directory for %s", CgroupProcesses)
	}

	if pid == -1 {
		return nil
	}

	file, err := OpenFile(dir, CgroupProcesses, os.O_WRONLY)
	if err != nil {
		return fmt.Errorf("failed to write %v to %v: %v", pid, CgroupProcesses, err)
	}
	defer file.Close()

	for i := 0; i < 5; i++ {
		_, err = file.WriteString(strconv.Itoa(pid))
		if err == nil {
			return nil
		}

		if errors.Is(err, unix.EINVAL) {
			time.Sleep(30 * time.Millisecond)
			continue
		}

		return fmt.Errorf("failed to write %v to %v: %v", pid, CgroupProcesses, err)
	}
	return err
}

func ConvertCPUSharesToCgroupV2Value(cpuShares uint64) uint64 {
	if cpuShares == 0 {
		return 0
	}
	return (1 + ((cpuShares-2)*9999)/262142)
}

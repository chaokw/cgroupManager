// +build linux

package cgroupManager

import (
	"bytes"
	"fmt"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	cgroupfsDir    = "/sys/fs/cgroup"
	cgroupfsPrefix = cgroupfsDir + "/"
)

var (
	TestMode          bool
	cgroupFd          int = -1
	prepOnce          sync.Once
	prepErr           error
	resolveFlags      uint64
	ErrNotValidFormat = errors.New("line is not a valid key value format")
)

func prepareOpenat2() error {
	prepOnce.Do(func() {
		fd, err := unix.Openat2(-1, cgroupfsDir, &unix.OpenHow{
			Flags: unix.O_DIRECTORY | unix.O_PATH})
		if err != nil {
			prepErr = &os.PathError{Op: "openat2", Path: cgroupfsDir, Err: err}
			if err != unix.ENOSYS {
				logrus.Warnf("falling back to securejoin: %s", prepErr)
			} else {
				logrus.Debug("openat2 not available, falling back to securejoin")
			}
			return
		}
		var st unix.Statfs_t
		if err = unix.Fstatfs(fd, &st); err != nil {
			prepErr = &os.PathError{Op: "statfs", Path: cgroupfsDir, Err: err}
			logrus.Warnf("falling back to securejoin: %s", prepErr)
			return
		}

		cgroupFd = fd

		resolveFlags = unix.RESOLVE_BENEATH | unix.RESOLVE_NO_MAGICLINKS
		if st.Type == unix.CGROUP2_SUPER_MAGIC {
			// cgroupv2 has a single mountpoint and no "cpu,cpuacct" symlinks
			resolveFlags |= unix.RESOLVE_NO_XDEV | unix.RESOLVE_NO_SYMLINKS
		}

	})

	return prepErr
}

func OpenFile(dir, file string, flags int) (*os.File, error) {
	if dir == "" {
		return nil, errors.Errorf("no directory specified for %s", file)
	}
	mode := os.FileMode(0)
	if TestMode && flags&os.O_WRONLY != 0 {
		flags |= os.O_TRUNC | os.O_CREATE
	}
	reldir := strings.TrimPrefix(dir, cgroupfsPrefix)
	if len(reldir) == len(dir) {
		return openWithSecureJoin(dir, file, flags, mode)
	}
	if prepareOpenat2() != nil {
		return openWithSecureJoin(dir, file, flags, mode)
	}

	relname := reldir + "/" + file
	fd, err := unix.Openat2(cgroupFd, relname,
		&unix.OpenHow{
			Resolve: resolveFlags,
			Flags:   uint64(flags) | unix.O_CLOEXEC,
			Mode:    uint64(mode),
		})
	if err != nil {
		return nil, &os.PathError{Op: "openat2", Path: dir + "/" + file, Err: err}
	}

	return os.NewFile(uintptr(fd), cgroupfsPrefix+relname), nil
}

func openWithSecureJoin(dir, file string, flags int, mode os.FileMode) (*os.File, error) {
	path, err := securejoin.SecureJoin(dir, file)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(path, flags, mode)
}

// WriteFile writes data to a cgroup file in dir.
// It is supposed to be used for cgroup files only.
func WriteFile(dir, file, data string) error {
	fd, err := OpenFile(dir, file, unix.O_WRONLY)
	if err != nil {
		return err
	}
	defer fd.Close()
	if err := retryingWriteFile(fd, data); err != nil {
		return errors.Wrapf(err, "failed to write %q", data)
	}
	return nil
}

// ReadFile reads data from a cgroup file in dir.
// It is supposed to be used for cgroup files only.
func ReadFile(dir, file string) (string, error) {
	fd, err := OpenFile(dir, file, unix.O_RDONLY)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer

	_, err = buf.ReadFrom(fd)
	return buf.String(), err
}

func retryingWriteFile(fd *os.File, data string) error {
	for {
		_, err := fd.Write([]byte(data))
		if errors.Is(err, unix.EINTR) {
			logrus.Infof("interrupted while writing %s to %s", data, fd.Name())
			continue
		}
		return err
	}
}

// ParseUint converts a string to an uint64 integer.
// Negative values are returned at zero as, due to kernel bugs,
// some of the memory cgroup stats can be negative.
func ParseUint(s string, base, bitSize int) (uint64, error) {
	value, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil && intErr.(*strconv.NumError).Err == strconv.ErrRange && intValue < 0 {
			return 0, nil
		}

		return value, err
	}

	return value, nil
}

// GetCgroupParamKeyValue parses a space-separated "name value" kind of cgroup
// parameter and returns its components. For example, "io_service_bytes 1234"
// will return as "io_service_bytes", 1234.
func GetCgroupParamKeyValue(t string) (string, uint64, error) {
	parts := strings.Fields(t)
	switch len(parts) {
	case 2:
		value, err := ParseUint(parts[1], 10, 64)
		if err != nil {
			return "", 0, fmt.Errorf("unable to convert to uint64: %v", err)
		}

		return parts[0], value, nil
	default:
		return "", 0, ErrNotValidFormat
	}
}

// GetCgroupParamUint reads a single uint64 value from the specified cgroup file.
// If the value read is "max", the math.MaxUint64 is returned.
func GetCgroupParamUint(path, file string) (uint64, error) {
	contents, err := GetCgroupParamString(path, file)
	if err != nil {
		return 0, err
	}
	contents = strings.TrimSpace(contents)
	if contents == "max" {
		return math.MaxUint64, nil
	}

	res, err := ParseUint(contents, 10, 64)
	if err != nil {
		return res, fmt.Errorf("unable to parse file %q", path+"/"+file)
	}
	return res, nil
}

// GetCgroupParamString reads a string from the specified cgroup file.
func GetCgroupParamString(path, file string) (string, error) {
	contents, err := ReadFile(path, file)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(contents), nil
}

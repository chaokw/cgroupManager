// +build linux

package cgroupManager

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidCgroupPath(t *testing.T) {
	if IsCgroup2UnifiedMode() {
		t.Skip("cgroup v2 is not supported")
	}

	root, err := getCgroupRoot()
	if err != nil {
		t.Fatalf("couldn't get cgroup root: %v", err)
	}

	testCases := []struct {
		test               string
		path, name, parent string
	}{
		{
			test: "invalid cgroup path",
			path: "../../../../../../../../../../some/path",
		},
		{
			test: "invalid absolute cgroup path",
			path: "/../../../../../../../../../../some/path",
		},
		{
			test:   "invalid cgroup parent",
			parent: "../../../../../../../../../../some/path",
			name:   "name",
		},
		{
			test:   "invalid absolute cgroup parent",
			parent: "/../../../../../../../../../../some/path",
			name:   "name",
		},
		{
			test:   "invalid cgroup name",
			parent: "parent",
			name:   "../../../../../../../../../../some/path",
		},
		{
			test:   "invalid absolute cgroup name",
			parent: "parent",
			name:   "/../../../../../../../../../../some/path",
		},
		{
			test:   "invalid cgroup name and parent",
			parent: "../../../../../../../../../../some/path",
			name:   "../../../../../../../../../../some/path",
		},
		{
			test:   "invalid absolute cgroup name and parent",
			parent: "/../../../../../../../../../../some/path",
			name:   "/../../../../../../../../../../some/path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.test, func(t *testing.T) {
			config := &CgroupConfig{Path: tc.path, Name: tc.name, Parent: tc.parent}

			data, err := getCgroupData(config, 0)
			if err != nil {
				t.Fatalf("couldn't get cgroup data: %v", err)
			}

			// Make sure the final innerPath doesn't go outside the cgroup mountpoint.
			if strings.HasPrefix(data.innerPath, "..") {
				t.Errorf("SECURITY: cgroup innerPath is outside cgroup mountpoint!")
			}

			// Double-check, using an actual cgroup.
			deviceRoot := filepath.Join(root, "devices")
			devicePath, err := data.path("devices")
			if err != nil {
				t.Fatalf("couldn't get cgroup path: %v", err)
			}
			if !strings.HasPrefix(devicePath, deviceRoot) {
				t.Errorf("SECURITY: cgroup path() is outside cgroup mountpoint!")
			}
		})
	}
}

func TestTryDefaultCgroupRoot(t *testing.T) {
	res := tryDefaultCgroupRoot()
	exp := defaultCgroupRoot
	if IsCgroup2UnifiedMode() {
		// checking that tryDefaultCgroupRoot does return ""
		// in case /sys/fs/cgroup is not cgroup v1 root dir.
		exp = ""
	}
	if res != exp {
		t.Errorf("tryDefaultCgroupRoot: want %q, got %q", exp, res)
	}
}

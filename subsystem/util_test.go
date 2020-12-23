// +build linux

package subsystem

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cgroups "cgroupManager"
)

func init() {
	cgroups.TestMode = true
}

type cgroupTestUtil struct {
	CgroupData *cgroupData
	CgroupPath string
	tempDir string
	t       *testing.T
}

// Creates a new test util for the specified subsystem
func NewCgroupTestUtil(subsystem string, t *testing.T) *cgroupTestUtil {
	d := &cgroupData{
		config: &cgroups.CgroupConfig{},
	}
	d.config.Resources = &cgroups.Resources{}
	tempDir, err := ioutil.TempDir("", "cgroup_test")
	if err != nil {
		t.Fatal(err)
	}
	d.root = tempDir
	testCgroupPath := filepath.Join(d.root, subsystem)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure the full mock cgroup path exists.
	err = os.MkdirAll(testCgroupPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	return &cgroupTestUtil{CgroupData: d, CgroupPath: testCgroupPath, tempDir: tempDir, t: t}
}

func (c *cgroupTestUtil) cleanup() {
	os.RemoveAll(c.tempDir)
}

// Write the specified contents on the mock of the specified cgroup files.
func (c *cgroupTestUtil) writeFileContents(fileContents map[string]string) {
	for file, contents := range fileContents {
		err := cgroups.WriteFile(c.CgroupPath, file, contents)
		if err != nil {
			c.t.Fatal(err)
		}
	}
}

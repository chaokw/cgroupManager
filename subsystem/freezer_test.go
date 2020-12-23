// +build linux

package subsystem

import (
	"testing"

	cgroups "cgroupManager"
)

func TestFreezerSetState(t *testing.T) {
	helper := NewCgroupTestUtil("freezer", t)
	defer helper.cleanup()

	helper.writeFileContents(map[string]string{
		"freezer.state": string(cgroups.Frozen),
	})

	helper.CgroupData.config.Resources.Freezer = cgroups.Thawed
	freezer := &FreezerGroup{}
	if err := freezer.Set(helper.CgroupPath, helper.CgroupData.config); err != nil {
		t.Fatal(err)
	}

	value, err := cgroups.GetCgroupParamString(helper.CgroupPath, "freezer.state")
	if err != nil {
		t.Fatalf("Failed to parse freezer.state - %s", err)
	}
	if value != string(cgroups.Thawed) {
		t.Fatal("Got the wrong value, set freezer.state failed.")
	}
}

func TestFreezerSetInvalidState(t *testing.T) {
	helper := NewCgroupTestUtil("freezer", t)
	defer helper.cleanup()

	const (
		invalidArg cgroups.FreezerState = "Invalid"
	)

	helper.CgroupData.config.Resources.Freezer = invalidArg
	freezer := &FreezerGroup{}
	if err := freezer.Set(helper.CgroupPath, helper.CgroupData.config); err == nil {
		t.Fatal("Failed to return invalid argument error")
	}
}

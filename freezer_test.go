// +build linux

package cgroupManager

import (
	"testing"
)

func TestFreezerSetState(t *testing.T) {
	helper := NewCgroupTestUtil("freezer", t)
	defer helper.cleanup()

	helper.writeFileContents(map[string]string{
		"freezer.state": string(Frozen),
	})

	helper.CgroupData.config.Resources.Freezer = Thawed
	freezer := &FreezerGroup{}
	if err := freezer.Set(helper.CgroupPath, helper.CgroupData.config); err != nil {
		t.Fatal(err)
	}

	value, err := GetCgroupParamString(helper.CgroupPath, "freezer.state")
	if err != nil {
		t.Fatalf("Failed to parse freezer.state - %s", err)
	}
	if value != string(Thawed) {
		t.Fatal("Got the wrong value, set freezer.state failed.")
	}
}

func TestFreezerSetInvalidState(t *testing.T) {
	helper := NewCgroupTestUtil("freezer", t)
	defer helper.cleanup()

	const (
		invalidArg FreezerState = "Invalid"
	)

	helper.CgroupData.config.Resources.Freezer = invalidArg
	freezer := &FreezerGroup{}
	if err := freezer.Set(helper.CgroupPath, helper.CgroupData.config); err == nil {
		t.Fatal("Failed to return invalid argument error")
	}
}

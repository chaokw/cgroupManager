// +build linux

package cgroupManager

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
	"io/ioutil"
	"math"
)

const (
	cgroupFile  = "cgroup.file"
	floatValue  = 2048.0
	floatString = "2048"
)

func TestWriteCgroupFileHandlesInterrupt(t *testing.T) {
	const (
		memoryCgroupMount = "/sys/fs/cgroup/memory"
		memoryLimit       = "memory.limit_in_bytes"
	)
	if _, err := os.Stat(memoryCgroupMount); err != nil {
		// most probably cgroupv2
		t.Skip(err)
	}

	cgroupName := fmt.Sprintf("test-eint-%d", time.Now().Nanosecond())
	cgroupPath := filepath.Join(memoryCgroupMount, cgroupName)
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cgroupPath)

	if _, err := os.Stat(filepath.Join(cgroupPath, memoryLimit)); err != nil {
		// either cgroupv2, or memory controller is not available
		t.Skip(err)
	}

	for i := 0; i < 100000; i++ {
		limit := 1024*1024 + i
		if err := WriteFile(cgroupPath, memoryLimit, strconv.Itoa(limit)); err != nil {
			t.Fatalf("Failed to write %d on attempt %d: %+v", limit, i, err)
		}
	}
}

func TestGetCgroupParamsInt(t *testing.T) {
	// Setup tempdir.
	tempDir, err := ioutil.TempDir("", "cgroup_utils_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	tempFile := filepath.Join(tempDir, cgroupFile)

	// Success.
	err = ioutil.WriteFile(tempFile, []byte(floatString), 0755)
	if err != nil {
		t.Fatal(err)
	}
	value, err := GetCgroupParamUint(tempDir, cgroupFile)
	if err != nil {
		t.Fatal(err)
	} else if value != floatValue {
		t.Fatalf("Expected %d to equal %f", value, floatValue)
	}

	// Success with new line.
	err = ioutil.WriteFile(tempFile, []byte(floatString+"\n"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	value, err = GetCgroupParamUint(tempDir, cgroupFile)
	if err != nil {
		t.Fatal(err)
	} else if value != floatValue {
		t.Fatalf("Expected %d to equal %f", value, floatValue)
	}

	// Success with negative values
	err = ioutil.WriteFile(tempFile, []byte("-12345"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	value, err = GetCgroupParamUint(tempDir, cgroupFile)
	if err != nil {
		t.Fatal(err)
	} else if value != 0 {
		t.Fatalf("Expected %d to equal %d", value, 0)
	}

	// Success with negative values lesser than min int64
	s := strconv.FormatFloat(math.MinInt64, 'f', -1, 64)
	err = ioutil.WriteFile(tempFile, []byte(s), 0755)
	if err != nil {
		t.Fatal(err)
	}
	value, err = GetCgroupParamUint(tempDir, cgroupFile)
	if err != nil {
		t.Fatal(err)
	} else if value != 0 {
		t.Fatalf("Expected %d to equal %d", value, 0)
	}

	// Not a float.
	err = ioutil.WriteFile(tempFile, []byte("not-a-float"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	_, err = GetCgroupParamUint(tempDir, cgroupFile)
	if err == nil {
		t.Fatal("Expecting error, got none")
	}

	// Unknown file.
	err = os.Remove(tempFile)
	if err != nil {
		t.Fatal(err)
	}
	_, err = GetCgroupParamUint(tempDir, cgroupFile)
	if err == nil {
		t.Fatal("Expecting error, got none")
	}
}

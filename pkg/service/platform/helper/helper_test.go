package helper

import (
	"os"
	"testing"
)

func TestFileExists(t *testing.T) {
	if fileExists("is not") {
		t.Fatal("file is not exists")
	}

	fp, err := os.Create("/tmp/test")
	if err != nil {
		t.Fatal(err)
	}
	if err := fp.Close(); err != nil {
		t.Fatal(err)
	}

	if !fileExists("/tmp/test") {
		t.Fatal("file exists")
	}

	if err := os.Remove("/tmp/test"); err != nil {
		t.Fatal(err)
	}
}

func TestGetExecutableFilePath(t *testing.T) {
	if _, err := GetExecutableFilePath(); err != nil {
		t.Fatal(err)
	}
}

func TestRunningInCluster(t *testing.T) {
	if !RunningInCluster() {
		t.Fatal("must running in cluster")
	}
}

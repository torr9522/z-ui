package xray

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigFileWrittenWithRestrictedPermissions(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "bin"), 0755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	process := newProcess(&Config{})
	if err := process.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}

	info, err := os.Stat(GetConfigPath())
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("unexpected config file permission: %#o", got)
	}
}

package runtimeconfig

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestToolVersionProbeIsBounded(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "blocked.go")
	if err := os.WriteFile(source, []byte("package main\nimport \"time\"\nfunc main(){ time.Sleep(time.Hour) }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(root, "blocked")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	if output, err := exec.CommandContext(t.Context(), "go", "build", "-o", executable, source).CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v: %s", err, output)
	}
	previous := toolVersionTimeout
	toolVersionTimeout = 25 * time.Millisecond
	defer func() { toolVersionTimeout = previous }()
	started := time.Now()
	if got := toolVersion(executable, "--version"); got != "indisponível" {
		t.Fatalf("blocked probe returned %q", got)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("blocked probe took %s", elapsed)
	}
}

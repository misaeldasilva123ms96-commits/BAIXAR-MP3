package runtimeconfig

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

func TestRuntimeVersionsRejectOutdatedTools(t *testing.T) {
	status := map[string]string{"yt-dlp": "2026.06.09", "deno": "deno 2.2.0 (release)", "ffmpeg": "ffmpeg version 9.0", "ffprobe": "ffprobe version 9.0"}
	invalid := validateToolVersions(status)
	if len(invalid) != 2 || !strings.Contains(status["yt-dlp"], "incompatível") || !strings.Contains(status["deno"], "incompatível") {
		t.Fatalf("invalid=%#v status=%#v", invalid, status)
	}
}

func TestRuntimeVersionsRejectInvalidFFmpegProbe(t *testing.T) {
	status := map[string]string{"yt-dlp": "2026.07.04", "deno": "deno 2.9.5 (release)", "ffmpeg": "unexpected", "ffprobe": "ffprobe version 9.0"}
	invalid := validateToolVersions(status)
	if len(invalid) != 1 || invalid[0] != "ffmpeg" || !strings.Contains(status["ffmpeg"], "inválida") {
		t.Fatalf("invalid=%#v status=%#v", invalid, status)
	}
}

func TestParseYTDLPRuntimeDiagnosticRequiresEJSAndDeno(t *testing.T) {
	valid := "[debug] Optional libraries: yt_dlp_ejs-0.9.2\n[debug] JS runtimes: deno-2.9.5"
	if got := parseYTDLPRuntimeDiagnostic(valid); got != "yt_dlp_ejs-0.9.2 (Deno detectado)" {
		t.Fatalf("diagnostic=%q", got)
	}
	if got := parseYTDLPRuntimeDiagnostic("[debug] JS runtimes: deno-2.9.5"); got != "indisponível" {
		t.Fatalf("missing EJS accepted: %q", got)
	}
	if got := parseYTDLPRuntimeDiagnostic("yt_dlp_ejs-0.9.2"); got != "indisponível" {
		t.Fatalf("missing Deno accepted: %q", got)
	}
}

func TestFindToolsReportsEveryMissingTool(t *testing.T) {
	root := t.TempDir()
	t.Setenv("YTDLP_PATH", filepath.Join(root, "missing-ytdlp"))
	t.Setenv("FFMPEG_PATH", filepath.Join(root, "missing-ffmpeg"))
	t.Setenv("FFPROBE_PATH", filepath.Join(root, "missing-ffprobe"))
	t.Setenv("DENO_PATH", filepath.Join(root, "missing-deno"))
	_, status, err := FindTools(root)
	if err == nil {
		t.Fatal("missing tools were accepted")
	}
	for _, name := range []string{"yt-dlp", "ffmpeg", "ffprobe", "deno"} {
		if status[name] != "indisponível" {
			t.Errorf("%s=%q", name, status[name])
		}
	}
}

func TestVersionAtLeast(t *testing.T) {
	for _, test := range []struct {
		value, minimum string
		want           bool
	}{
		{"2026.07.04", "2026.07.04", true}, {"2026.08.01", "2026.07.04", true}, {"2026.06.09", "2026.07.04", false},
		{"2.9.5", "2.3.0", true}, {"2.2.9", "2.3.0", false},
	} {
		if got := versionAtLeast(test.value, test.minimum); got != test.want {
			t.Errorf("versionAtLeast(%q,%q)=%v", test.value, test.minimum, got)
		}
	}
}

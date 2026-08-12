package core

import (
	"context"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateMediaURL(t *testing.T) {
	valid := []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://music.youtube.com/playlist?list=PL123",
	}
	for _, raw := range valid {
		if _, err := ValidateMediaURL(raw); err != nil {
			t.Errorf("ValidateMediaURL(%q) returned %v", raw, err)
		}
	}

	invalid := []string{
		"", "not a url", "file:///etc/passwd", "ftp://youtube.com/a",
		"http://localhost/video", "http://127.0.0.1/video",
		"http://10.0.0.1/video", "http://172.20.0.1/video",
		"http://192.168.1.1/video", "https://youtube.com.evil.example/video",
		"https://user:password@youtube.com/video",
		"https://www.youtube.com/redirect?q=http%3A%2F%2F127.0.0.1",
		"https://youtube.com/attribution_link?u=%2Fredirect",
	}
	for _, raw := range invalid {
		if _, err := ValidateMediaURL(raw); err == nil {
			t.Errorf("ValidateMediaURL(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestDirectorySize(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one"), make([]byte, 7), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "two"), make([]byte, 5), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := directorySize(dir); got != 12 {
		t.Fatalf("directorySize = %d", got)
	}
}

func TestOutputLimitStopsRunningProcess(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	if err := os.MkdirAll(output, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "writer.go")
	program := `package main
import ("os"; "path/filepath"; "time")
func main() { f, err := os.Create(filepath.Join(os.Getenv("MP3_TEST_OUTPUT"), "oversized.mp3")); if err != nil { panic(err) }; defer f.Close(); chunk := make([]byte, 32768); for i := 0; i < 64; i++ { if _, err = f.Write(chunk); err != nil { return }; _ = f.Sync(); time.Sleep(25*time.Millisecond) } }
`
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(root, "writer.exe")
	if output, err := exec.Command("go", "build", "-o", wrapper, source).CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v: %s", err, output)
	}
	t.Setenv("MP3_TEST_OUTPUT", output)
	p := &ExecProcessor{Tools: ToolPaths{YTDLP: wrapper}, OutputRoot: root, TempRoot: filepath.Join(root, "temp"), Cloud: true, MaxItems: 1, MaxOutputBytes: 64 * 1024}
	_, err := p.Start(context.Background(), "output", DownloadRequest{URL: "https://youtu.be/limit", Quality: QualityVBR0}, func(ProgressEvent) {})
	if err == nil || !strings.Contains(err.Error(), "excedeu") {
		t.Fatalf("expected enforced output limit, got %v", err)
	}
	if got := directorySize(output); got != 0 {
		t.Fatalf("partial output retained: %d bytes", got)
	}
}

func TestAddressIsPublic(t *testing.T) {
	private := []string{"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.0.1", "169.254.1.1", "::1", "fc00::1"}
	for _, raw := range private {
		if AddressIsPublic(netip.MustParseAddr(raw)) {
			t.Errorf("%s must not be public", raw)
		}
	}
	if !AddressIsPublic(netip.MustParseAddr("8.8.8.8")) {
		t.Error("8.8.8.8 should be public")
	}
}

func TestBuildYTDLPArguments(t *testing.T) {
	request := DownloadRequest{URL: "https://youtu.be/abc", Quality: QualityVBR0, PlaylistStart: 2, PlaylistEnd: 5, OrganizePlaylist: true}
	args, err := BuildYTDLPArguments(request, ToolPaths{FFmpegDir: `C:\\tools`, Deno: `C:\\tools\\deno.exe`}, `C:\\out`, `C:\\tmp`, `C:\\history.txt`)
	if err != nil {
		t.Fatal(err)
	}
	wantPairs := map[string]string{"--audio-quality": "0", "--playlist-start": "2", "--playlist-end": "5", "--download-archive": `C:\\history.txt`}
	for flag, want := range wantPairs {
		found := false
		for i := 0; i+1 < len(args); i++ {
			if args[i] == flag && args[i+1] == want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %s %s in %#v", flag, want, args)
		}
	}
	if got := args[len(args)-1]; got != request.URL {
		t.Errorf("last arg = %q", got)
	}
}

func TestRejectInvalidQualityAndPlaylistRange(t *testing.T) {
	bad := []DownloadRequest{
		{URL: "https://youtu.be/a", Quality: "lossless"},
		{URL: "https://youtu.be/a", Quality: Quality192, PlaylistStart: 9, PlaylistEnd: 2},
		{URL: "https://youtu.be/a", Quality: Quality192, PlaylistEnd: 501},
	}
	for _, request := range bad {
		if _, err := BuildYTDLPArguments(request, ToolPaths{}, t.TempDir(), t.TempDir(), ""); err == nil {
			t.Errorf("request unexpectedly accepted: %#v", request)
		}
	}
}

func TestSafeOutputName(t *testing.T) {
	got := SafeOutputName(`  música: teste / faixa * 01?  `)
	if got != "música_ teste _ faixa _ 01_" {
		t.Fatalf("unexpected name %q", got)
	}
	if filepath.IsAbs(got) {
		t.Fatal("name must remain relative")
	}
}

func TestQualityValuesRemainCompatibleWithV2(t *testing.T) {
	want := map[Quality]string{QualityVBR0: "0", Quality320: "320K", Quality256: "256K", Quality192: "192K", Quality128: "128K"}
	if !reflect.DeepEqual(QualityArguments(), want) {
		t.Fatalf("quality mapping changed: %#v", QualityArguments())
	}
}

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
	"time"
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
	got, err := directorySize(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != 12 {
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
import ("os"; "os/exec"; "path/filepath"; "time")
func main() {
 if os.Getenv("MP3_HELPER_CHILD") != "1" { child := exec.Command(os.Args[0]); child.Env = append(os.Environ(), "MP3_HELPER_CHILD=1"); if err := child.Start(); err != nil { panic(err) }; _ = child.Wait(); return }
 output, err := os.Create(filepath.Join(os.Getenv("MP3_TEST_OUTPUT"), "oversized.mp3")); if err != nil { panic(err) }; defer output.Close()
 heartbeat, err := os.OpenFile(os.Getenv("MP3_TEST_HEARTBEAT"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); if err != nil { panic(err) }; defer heartbeat.Close()
 chunk := make([]byte, 32768); for { if _, err = output.Write(chunk); err != nil { return }; _, _ = heartbeat.Write([]byte("alive\n")); _ = output.Sync(); _ = heartbeat.Sync(); time.Sleep(25*time.Millisecond) }
}
`
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(root, "writer.exe")
	if output, err := exec.Command("go", "build", "-o", wrapper, source).CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v: %s", err, output)
	}
	t.Setenv("MP3_TEST_OUTPUT", output)
	heartbeat := filepath.Join(root, "heartbeat.log")
	t.Setenv("MP3_TEST_HEARTBEAT", heartbeat)
	p := &ExecProcessor{Tools: ToolPaths{YTDLP: wrapper}, OutputRoot: root, TempRoot: filepath.Join(root, "temp"), Cloud: true, MaxItems: 1, MaxOutputBytes: 64 * 1024}
	organize := false
	_, startErr := p.Start(context.Background(), "output", DownloadRequest{URL: "https://youtu.be/limit", Quality: QualityVBR0, OrganizePlaylist: &organize}, func(ProgressEvent) {})
	if startErr == nil || !strings.Contains(startErr.Error(), "excedeu") {
		t.Fatalf("expected enforced output limit, got %v", startErr)
	}
	got, sizeErr := directorySize(output)
	if sizeErr != nil && !os.IsNotExist(sizeErr) {
		t.Fatal(sizeErr)
	}
	if got != 0 {
		t.Fatalf("partial output retained: %d bytes", got)
	}
	before, statErr := os.Stat(heartbeat)
	if statErr != nil {
		t.Fatal(statErr)
	}
	time.Sleep(250 * time.Millisecond)
	after, statErr := os.Stat(heartbeat)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if after.Size() != before.Size() {
		t.Fatalf("child process continued writing after cancellation: %d -> %d", before.Size(), after.Size())
	}
}

func TestOutputLimitIncludesGeneratedPlaylistArchive(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "playlist.go")
	program := `package main
import ("crypto/rand"; "fmt"; "os"; "path/filepath")
func main() { for _, name := range []string{"one.mp3", "two.mp3"} { path := filepath.Join(os.Getenv("MP3_TEST_OUTPUT"), name); data := make([]byte, 40*1024); if _, err := rand.Read(data); err != nil { panic(err) }; if err := os.WriteFile(path, data, 0600); err != nil { panic(err) }; fmt.Println("MP3_RESULT|"+path) } }
`
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(root, "playlist.exe")
	if output, err := exec.CommandContext(t.Context(), "go", "build", "-o", wrapper, source).CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v: %s", err, output)
	}
	outputDir := filepath.Join(root, "job")
	t.Setenv("MP3_TEST_OUTPUT", outputDir)
	organize := false
	p := &ExecProcessor{Tools: ToolPaths{YTDLP: wrapper}, OutputRoot: root, TempRoot: filepath.Join(root, "temp"), Cloud: true, MaxItems: 2, MaxOutputBytes: 100 * 1024}
	_, err := p.Start(context.Background(), "job", DownloadRequest{URL: "https://youtu.be/playlist", Quality: QualityVBR0, OrganizePlaylist: &organize}, func(ProgressEvent) {})
	if err == nil || !strings.Contains(err.Error(), "excedeu") {
		t.Fatalf("final archive was not included in quota: %v", err)
	}
	if _, statErr := os.Stat(outputDir); !os.IsNotExist(statErr) {
		t.Fatalf("oversized output retained: %v", statErr)
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
	organize := true
	request := DownloadRequest{URL: "https://youtu.be/abc", Quality: QualityVBR0, PlaylistStart: 2, PlaylistEnd: 5, OrganizePlaylist: &organize}
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
	organize := false
	bad := []DownloadRequest{
		{URL: "https://youtu.be/a", Quality: "lossless", OrganizePlaylist: &organize},
		{URL: "https://youtu.be/a", Quality: Quality192, PlaylistStart: 9, PlaylistEnd: 2, OrganizePlaylist: &organize},
		{URL: "https://youtu.be/a", Quality: Quality192, PlaylistEnd: 501, OrganizePlaylist: &organize},
		{URL: "https://youtu.be/a", Quality: Quality192},
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

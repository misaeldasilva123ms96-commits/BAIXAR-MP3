package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/core"
)

type fakeProcessor struct{}

type failingProcessor struct {
	code core.ErrorCode
}

func (p failingProcessor) Analyze(context.Context, string) (core.Analysis, error) {
	return core.Analysis{}, &core.RuntimeError{Code: p.code, Message: "safe upstream message", Retryable: p.code == core.CodeYouTubeRateLimited}
}
func (p failingProcessor) Start(context.Context, string, core.DownloadRequest, func(core.ProgressEvent)) (core.DownloadResult, error) {
	return core.DownloadResult{}, &core.RuntimeError{Code: p.code, Message: "safe upstream message"}
}

func (fakeProcessor) Analyze(_ context.Context, rawURL string) (core.Analysis, error) {
	return core.Analysis{Type: "video", Title: "Real title from processor", WebpageURL: rawURL}, nil
}

type blockingProcessor struct{}

func (blockingProcessor) Analyze(_ context.Context, rawURL string) (core.Analysis, error) {
	return core.Analysis{Type: "video", WebpageURL: rawURL}, nil
}
func (blockingProcessor) Start(ctx context.Context, _ string, _ core.DownloadRequest, _ func(core.ProgressEvent)) (core.DownloadResult, error) {
	<-ctx.Done()
	return core.DownloadResult{}, ctx.Err()
}

type fileProcessor struct{ path string }

type gatedProcessor struct {
	started chan struct{}
	release chan struct{}
}

func (p gatedProcessor) Analyze(_ context.Context, rawURL string) (core.Analysis, error) {
	return core.Analysis{Type: "video", WebpageURL: rawURL}, nil
}
func (p gatedProcessor) Start(_ context.Context, _ string, _ core.DownloadRequest, _ func(core.ProgressEvent)) (core.DownloadResult, error) {
	close(p.started)
	<-p.release
	return core.DownloadResult{Title: "Track", Format: "mp3", FileName: "track.mp3"}, nil
}

type memorySettings struct{ value core.Settings }

func (s *memorySettings) Get() core.Settings             { return s.value }
func (s *memorySettings) Save(value core.Settings) error { s.value = value; return nil }

func (p fileProcessor) Analyze(_ context.Context, rawURL string) (core.Analysis, error) {
	return core.Analysis{Type: "video", WebpageURL: rawURL}, nil
}
func (p fileProcessor) Start(_ context.Context, _ string, _ core.DownloadRequest, _ func(core.ProgressEvent)) (core.DownloadResult, error) {
	info, err := os.Stat(p.path)
	if err != nil {
		return core.DownloadResult{}, err
	}
	return core.DownloadResult{Title: "Track", Format: "mp3", FileName: filepath.Base(p.path), FilePath: p.path, Size: info.Size()}, nil
}
func (fakeProcessor) Start(_ context.Context, _ string, _ core.DownloadRequest, _ func(core.ProgressEvent)) (core.DownloadResult, error) {
	return core.DownloadResult{Title: "Track", Format: "mp3", FileName: "track.mp3"}, nil
}

func TestHealthAndVersion(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud, Version: "3.0.0", AllowedOrigins: []string{"https://misaeldasilva123ms96-commits.github.io"}}, fakeProcessor{})
	for _, path := range []string{"/health", "/version"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, w.Code)
		}
	}
}

func TestAnalyzeRejectsPrivateAndInvalidURLs(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud}, fakeProcessor{})
	for _, rawURL := range []string{"http://127.0.0.1/admin", "file:///tmp/a", "https://evil.example/a"} {
		body, _ := json.Marshal(map[string]string{"url": rawURL})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body)))
		if w.Code != http.StatusBadRequest {
			t.Errorf("%q returned %d", rawURL, w.Code)
		}
	}
}

func TestCORSIsRestricted(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeLocalEngine, AllowedOrigins: []string{"https://misaeldasilva123ms96-commits.github.io"}}, fakeProcessor{})
	for _, tc := range []struct {
		origin  string
		allowed bool
	}{{"https://misaeldasilva123ms96-commits.github.io", true}, {"https://evil.example", false}} {
		r := httptest.NewRequest(http.MethodOptions, "/health", nil)
		r.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		got := w.Header().Get("Access-Control-Allow-Origin")
		if tc.allowed && got != tc.origin {
			t.Errorf("official origin not allowed: %q", got)
		}
		if !tc.allowed && got != "" {
			t.Errorf("unexpected origin allowed: %q", got)
		}
	}
}

func TestPrivateNetworkPreflightIsGrantedOnlyToAllowedOrigin(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeLocalEngine, AllowedOrigins: []string{"https://misaeldasilva123ms96-commits.github.io"}}, fakeProcessor{})
	for _, origin := range []string{"https://misaeldasilva123ms96-commits.github.io", "https://evil.example"} {
		r := httptest.NewRequest(http.MethodOptions, "/health", nil)
		r.Header.Set("Origin", origin)
		r.Header.Set("Access-Control-Request-Private-Network", "true")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		got := w.Header().Get("Access-Control-Allow-Private-Network")
		if origin == "https://misaeldasilva123ms96-commits.github.io" && got != "true" {
			t.Fatalf("allowed PNA preflight missing grant: %q", got)
		}
		if origin == "https://evil.example" && got != "" {
			t.Fatalf("unknown origin received PNA grant: %q", got)
		}
	}
}

func TestAnalyzeReturnsStructuredUpstreamError(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud}, failingProcessor{code: core.CodeYouTubeRateLimited})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewBufferString(`{"url":"https://youtu.be/abc"}`)))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != string(core.CodeYouTubeRateLimited) || body.Error.Message != "safe upstream message" {
		t.Fatalf("unexpected error: %#v", body.Error)
	}
}

func TestRateLimit(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud, RateLimit: 2, RateWindow: time.Minute}, fakeProcessor{})
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
		if i < 2 && w.Code != http.StatusOK {
			t.Fatalf("request %d returned %d", i, w.Code)
		}
		if i == 2 && w.Code != http.StatusTooManyRequests {
			t.Fatalf("rate limit returned %d", w.Code)
		}
	}
}

func TestLocalMutationsRequireEngineToken(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeLocalEngine, EngineToken: "expected"}, fakeProcessor{})
	body := bytes.NewBufferString(`{"url":"https://youtu.be/abc","quality":"vbr0","organizePlaylist":true}`)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/downloads", body))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("without token returned %d", w.Code)
	}
	body = bytes.NewBufferString(`{"url":"https://youtu.be/abc","quality":"vbr0","organizePlaylist":true}`)
	r := httptest.NewRequest(http.MethodPost, "/downloads", body)
	r.Header.Set("X-MP3-Engine-Token", "expected")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("with token returned %d: %s", w.Code, w.Body.String())
	}
}

func TestDesktopMutationsAlsoRequireEngineToken(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeDesktopLocal, EngineToken: "expected"}, fakeProcessor{})
	body := bytes.NewBufferString(`{"url":"https://youtu.be/abc","quality":"vbr0","organizePlaylist":true}`)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/downloads", body))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("desktop without token returned %d", w.Code)
	}
}

func TestLocalSettingsAreTypedAndProtected(t *testing.T) {
	store := &memorySettings{value: core.Settings{DefaultQuality: core.QualityVBR0, DownloadDirectory: t.TempDir()}}
	h := NewHandler(Config{Mode: core.ModeDesktopLocal, EngineToken: "expected", Settings: store}, fakeProcessor{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/settings", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unprotected settings read returned %d", w.Code)
	}
	requestSettings := core.Settings{DefaultQuality: core.Quality192, DownloadDirectory: filepath.Join(t.TempDir(), "Music"), AvoidDuplicates: true, EmbedThumbnail: true, EmbedMetadata: true}
	encoded, _ := json.Marshal(requestSettings)
	payload := string(encoded)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/settings", bytes.NewBufferString(payload)))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unprotected settings returned %d", w.Code)
	}
	r := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewBufferString(payload))
	r.Header.Set("X-MP3-Engine-Token", "expected")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("settings update returned %d: %s", w.Code, w.Body.String())
	}
	if store.value.DefaultQuality != core.Quality192 {
		t.Fatalf("quality not stored: %s", store.value.DefaultQuality)
	}
}

func TestCancellationReflectsReality(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud, JobTimeout: time.Minute}, blockingProcessor{})
	id := createTestJob(t, h)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/downloads/"+id, nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("cancel returned %d", w.Code)
	}
	job := waitForState(t, h, id, core.StateCancelled)
	if job.Progress.State != core.StateCancelled {
		t.Fatalf("progress state %s", job.Progress.State)
	}
}

func TestTimeoutIsNotReportedAsCompleted(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud, JobTimeout: 20 * time.Millisecond}, blockingProcessor{})
	id := createTestJob(t, h)
	job := waitForState(t, h, id, core.StateCancelled)
	if job.Result != nil {
		t.Fatal("timed out job must not have a result")
	}
}

func TestCloudFileExpiresAndIsRemoved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.mp3")
	if err := os.WriteFile(path, []byte("mp3"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Config{Mode: core.ModeWebCloud, FileTTL: 20 * time.Millisecond}, fileProcessor{path: path})
	id := createTestJob(t, h)
	_ = waitForState(t, h, id, core.StateCompleted)
	time.Sleep(30 * time.Millisecond)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/downloads/"+id+"/file", nil))
	if w.Code != http.StatusNotFound && w.Code != http.StatusGone {
		t.Fatalf("expired file returned %d", w.Code)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expired file still exists: %v", err)
	}
}

func TestFailedCloudJobExpires(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud, FileTTL: 20 * time.Millisecond}, blockingProcessor{})
	id := createTestJob(t, h)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/downloads/"+id, nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("cancel returned %d", w.Code)
	}
	_ = waitForState(t, h, id, core.StateCancelled)
	time.Sleep(30 * time.Millisecond)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/downloads/"+id, nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expired terminal job returned %d", w.Code)
	}
}

func TestCloudJobCapacityIsBounded(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud, MaxJobs: 1}, blockingProcessor{})
	_ = createTestJob(t, h)
	body := bytes.NewBufferString(`{"url":"https://youtu.be/second","quality":"vbr0","organizePlaylist":true}`)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/downloads", body))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("full job store returned %d", w.Code)
	}
}

func TestDownloadRequestIsValidatedBeforeQueueing(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud}, fakeProcessor{})
	for _, payload := range []string{
		`{"url":"https://youtu.be/abc","quality":"vbr0"}`,
		`{"url":"https://youtu.be/abc","quality":"vbr0","organizePlaylist":false,"playlistStart":9,"playlistEnd":2}`,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/downloads", bytes.NewBufferString(payload)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid request returned %d: %s", w.Code, w.Body.String())
		}
	}
}

func TestProgressStreamClosesAtTerminalState(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud}, fakeProcessor{})
	id := createTestJob(t, h)
	_ = waitForState(t, h, id, core.StateCompleted)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/downloads/"+id+"/events", nil))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("event stream did not close after terminal state")
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"state":"COMPLETED"`)) {
		t.Fatalf("terminal event missing: %s", w.Body.String())
	}
}

func TestRateLimitKeyStoreIsBounded(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeWebCloud, RateLimit: 10, GlobalRateLimit: 100, MaxRateKeys: 3, RateWindow: time.Minute}, fakeProcessor{})
	for i, remote := range []string{"192.0.2.1:1000", "192.0.2.2:1000", "192.0.2.3:1000"} {
		r := httptest.NewRequest(http.MethodGet, "/health", nil)
		r.RemoteAddr = remote
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if i < 2 && w.Code != http.StatusOK {
			t.Fatalf("request %d returned %d", i, w.Code)
		}
		if i == 2 && w.Code != http.StatusTooManyRequests {
			t.Fatalf("new key beyond capacity returned %d", w.Code)
		}
	}
	if len(h.visits) > h.config.MaxRateKeys {
		t.Fatalf("tracked keys grew to %d", len(h.visits))
	}
}

func TestLocalTerminalJobsReleaseCapacity(t *testing.T) {
	h := NewHandler(Config{Mode: core.ModeDesktopLocal, EngineToken: "expected", MaxJobs: 1}, fakeProcessor{})
	request := func() *httptest.ResponseRecorder {
		body := bytes.NewBufferString(`{"url":"https://youtu.be/abc","quality":"vbr0","organizePlaylist":false}`)
		r := httptest.NewRequest(http.MethodPost, "/downloads", body)
		r.Header.Set("X-MP3-Engine-Token", "expected")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	first := request()
	if first.Code != http.StatusAccepted {
		t.Fatalf("first request returned %d", first.Code)
	}
	var job Job
	if err := json.Unmarshal(first.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	_ = waitForState(t, h, job.ID, core.StateCompleted)
	if second := request(); second.Code != http.StatusAccepted {
		t.Fatalf("terminal job did not release capacity: %d %s", second.Code, second.Body.String())
	}
}

func TestCloudEvictionRemovesResultDirectory(t *testing.T) {
	dir := t.TempDir()
	resultDir := filepath.Join(dir, "job-output")
	if err := os.MkdirAll(resultDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(resultDir, "track.mp3")
	if err := os.WriteFile(path, []byte("mp3"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Config{Mode: core.ModeWebCloud, MaxJobs: 1}, fileProcessor{path: path})
	firstID := createTestJob(t, h)
	_ = waitForState(t, h, firstID, core.StateCompleted)
	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/downloads", bytes.NewBufferString(`{"url":"https://youtu.be/second","quality":"vbr0","organizePlaylist":false}`)))
	if second.Code != http.StatusAccepted {
		t.Fatalf("second request returned %d: %s", second.Code, second.Body.String())
	}
	if _, err := os.Stat(resultDir); !os.IsNotExist(err) {
		t.Fatalf("evicted cloud result directory remains: %v", err)
	}
}

func TestShutdownDrainsBackgroundJobs(t *testing.T) {
	processor := gatedProcessor{started: make(chan struct{}), release: make(chan struct{})}
	h := NewHandler(Config{Mode: core.ModeWebCloud, JobTimeout: time.Minute}, processor)
	_ = createTestJob(t, h)
	select {
	case <-processor.started:
	case <-time.After(time.Second):
		t.Fatal("job did not start")
	}
	done := make(chan error, 1)
	go func() { done <- h.Shutdown(t.Context()) }()
	select {
	case err := <-done:
		t.Fatalf("shutdown returned before job completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(processor.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finish after job completed")
	}
}

func createTestJob(t *testing.T, h http.Handler) string {
	t.Helper()
	body := bytes.NewBufferString(`{"url":"https://youtu.be/abc","quality":"vbr0","organizePlaylist":true}`)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/downloads", body))
	if w.Code != http.StatusAccepted {
		t.Fatalf("create returned %d: %s", w.Code, w.Body.String())
	}
	var job Job
	if err := json.Unmarshal(w.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if job.State != core.StateQueued {
		t.Fatalf("accepted response returned %s instead of %s", job.State, core.StateQueued)
	}
	return job.ID
}
func waitForState(t *testing.T, h http.Handler, id string, state core.JobState) *Job {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/downloads/"+id, nil))
		if w.Code == http.StatusOK {
			var job Job
			if err := json.Unmarshal(w.Body.Bytes(), &job); err != nil {
				t.Fatal(err)
			}
			if job.State == state {
				return &job
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach %s", id, state)
	return nil
}

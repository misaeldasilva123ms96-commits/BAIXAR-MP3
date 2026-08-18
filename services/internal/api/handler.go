package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/core"
)

type Processor interface {
	Analyze(context.Context, string) (core.Analysis, error)
	Start(context.Context, string, core.DownloadRequest, func(core.ProgressEvent)) (core.DownloadResult, error)
}

type SettingsStore interface {
	Get() core.Settings
	Save(core.Settings) error
}

type Config struct {
	Mode            core.Mode
	Version         string
	AllowedOrigins  []string
	EngineToken     string
	RateLimit       int
	GlobalRateLimit int
	RateWindow      time.Duration
	MaxConcurrent   int
	MaxJobs         int
	MaxRateKeys     int
	JobTimeout      time.Duration
	FileTTL         time.Duration
	Ready           bool
	Tools           map[string]string
	Settings        SettingsStore
}

type Job struct {
	ID        string               `json:"id"`
	Mode      core.Mode            `json:"mode"`
	State     core.JobState        `json:"state"`
	Request   core.DownloadRequest `json:"request"`
	Progress  core.ProgressEvent   `json:"progress"`
	Result    *core.DownloadResult `json:"result,omitempty"`
	Error     string               `json:"error,omitempty"`
	ErrorCode string               `json:"errorCode,omitempty"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
	ExpiresAt time.Time            `json:"expiresAt,omitempty"`
	cancel    context.CancelFunc
}

type Handler struct {
	config    Config
	processor Processor
	mux       *http.ServeMux
	mu        sync.RWMutex
	jobs      map[string]*Job
	limitMu   sync.Mutex
	visits    map[string][]time.Time
	semaphore chan struct{}
	jobsWG    sync.WaitGroup
}

func NewHandler(config Config, processor Processor) *Handler {
	if config.Version == "" {
		config.Version = "3.0.1"
	}
	if config.RateLimit <= 0 {
		config.RateLimit = 60
	}
	if config.GlobalRateLimit <= 0 {
		config.GlobalRateLimit = config.RateLimit * 20
	}
	if config.RateWindow <= 0 {
		config.RateWindow = time.Minute
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 2
	}
	if config.MaxJobs <= 0 {
		config.MaxJobs = 1000
	}
	if config.MaxRateKeys <= 0 {
		config.MaxRateKeys = 10000
	}
	if config.JobTimeout <= 0 {
		config.JobTimeout = 30 * time.Minute
	}
	if config.FileTTL <= 0 {
		config.FileTTL = 30 * time.Minute
	}
	h := &Handler{config: config, processor: processor, mux: http.NewServeMux(), jobs: map[string]*Job{}, visits: map[string][]time.Time{}, semaphore: make(chan struct{}, config.MaxConcurrent)}
	h.mux.HandleFunc("GET /health", h.health)
	h.mux.HandleFunc("GET /version", h.version)
	h.mux.HandleFunc("POST /analyze", h.analyze)
	h.mux.HandleFunc("POST /downloads", h.createDownload)
	h.mux.HandleFunc("GET /downloads/{id}", h.getDownload)
	h.mux.HandleFunc("DELETE /downloads/{id}", h.cancelDownload)
	h.mux.HandleFunc("GET /downloads/{id}/events", h.events)
	h.mux.HandleFunc("GET /downloads/{id}/file", h.file)
	h.mux.HandleFunc("GET /settings", h.getSettings)
	h.mux.HandleFunc("PUT /settings", h.saveSettings)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.cleanupExpired()
	origin := r.Header.Get("Origin")
	if origin != "" && h.originAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers, Access-Control-Request-Private-Network")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-MP3-Engine-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		localMode := h.config.Mode == core.ModeLocalEngine || h.config.Mode == core.ModeDesktopLocal
		if localMode && strings.EqualFold(r.Header.Get("Access-Control-Request-Private-Network"), "true") {
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
		}
	}
	securityHeaders(w)
	if r.Method == http.MethodOptions {
		if origin != "" && !h.originAllowed(origin) {
			http.Error(w, "origem não permitida", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !h.allowRate(remoteIP(r)) {
		writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Limite de requisições excedido.")
		return
	}
	if origin != "" && !h.originAllowed(origin) {
		writeError(w, http.StatusForbidden, "ORIGIN_DENIED", "Origem não permitida.")
		return
	}
	unsafeMethod := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
	localSensitiveRead := r.Method == http.MethodGet && (r.URL.Path == "/settings" || strings.HasSuffix(r.URL.Path, "/file"))
	if (h.config.Mode == core.ModeLocalEngine || h.config.Mode == core.ModeDesktopLocal) && (unsafeMethod || localSensitiveRead) {
		supplied := r.Header.Get("X-MP3-Engine-Token")
		validToken := h.config.EngineToken != "" && len(supplied) == len(h.config.EngineToken) && subtle.ConstantTimeCompare([]byte(supplied), []byte(h.config.EngineToken)) == 1
		if !validToken {
			writeError(w, http.StatusUnauthorized, "ENGINE_AUTH_REQUIRED", "Código de conexão local inválido.")
			return
		}
	}
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) getSettings(w http.ResponseWriter, _ *http.Request) {
	if h.config.Settings == nil {
		writeError(w, http.StatusNotFound, "NOT_AVAILABLE", "Configurações locais indisponíveis.")
		return
	}
	writeJSON(w, http.StatusOK, h.config.Settings.Get())
}
func (h *Handler) saveSettings(w http.ResponseWriter, r *http.Request) {
	if h.config.Settings == nil {
		writeError(w, http.StatusNotFound, "NOT_AVAILABLE", "Configurações locais indisponíveis.")
		return
	}
	var settings core.Settings
	if err := decodeJSON(w, r, &settings); err != nil {
		return
	}
	if err := settings.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
		return
	}
	if err := h.config.Settings.Save(settings); err != nil {
		writeError(w, http.StatusInternalServerError, "SETTINGS_FAILED", "Não foi possível salvar as configurações.")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "mode": h.config.Mode, "version": h.config.Version, "ready": h.config.Ready, "tools": h.config.Tools})
}
func (h *Handler) version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"version": h.config.Version, "apiVersion": "v1", "mode": h.config.Mode})
}

func (h *Handler) analyze(w http.ResponseWriter, r *http.Request) {
	var request struct {
		URL string `json:"url"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		return
	}
	if _, err := core.ValidateMediaURL(request.URL); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_URL", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	analysis, err := h.processor.Analyze(ctx, request.URL)
	if err != nil {
		code, message, _ := core.RuntimeErrorDetails(err)
		writeError(w, upstreamStatus(code), string(code), message)
		return
	}
	writeJSON(w, http.StatusOK, analysis)
}

func (h *Handler) createDownload(w http.ResponseWriter, r *http.Request) {
	var request core.DownloadRequest
	if err := decodeJSON(w, r, &request); err != nil {
		return
	}
	if err := core.ValidateDownloadRequest(request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	id, err := newID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ID_GENERATION_FAILED", "Não foi possível criar o download com segurança.")
		return
	}
	now := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), h.config.JobTimeout)
	job := &Job{ID: id, Mode: h.config.Mode, State: core.StateQueued, Request: request, Progress: core.ProgressEvent{JobID: id, State: core.StateQueued, UpdatedAt: now}, CreatedAt: now, UpdatedAt: now, cancel: cancel}
	h.mu.Lock()
	if len(h.jobs) >= h.config.MaxJobs {
		if !h.evictOldestTerminalLocked() {
			h.mu.Unlock()
			cancel()
			writeError(w, http.StatusServiceUnavailable, "JOB_CAPACITY_REACHED", "A fila está temporariamente cheia.")
			return
		}
	}
	h.jobs[id] = job
	h.mu.Unlock()
	writeJSON(w, http.StatusAccepted, job)
	h.jobsWG.Add(1)
	go func() {
		defer h.jobsWG.Done()
		h.run(ctx, job)
	}()
}

func (h *Handler) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		h.jobsWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		h.mu.RLock()
		for _, job := range h.jobs {
			if !terminal(job.State) && job.cancel != nil {
				job.cancel()
			}
		}
		h.mu.RUnlock()
		return ctx.Err()
	}
}

func (h *Handler) run(ctx context.Context, job *Job) {
	select {
	case h.semaphore <- struct{}{}:
	case <-ctx.Done():
		h.finishError(job.ID, core.StateCancelled, ctx.Err())
		return
	}
	defer func() { <-h.semaphore }()
	h.update(job.ID, core.ProgressEvent{State: core.StateDownloading, UpdatedAt: time.Now().UTC()})
	result, err := h.processor.Start(ctx, job.ID, job.Request, func(event core.ProgressEvent) {
		event.JobID = job.ID
		if event.UpdatedAt.IsZero() {
			event.UpdatedAt = time.Now().UTC()
		}
		h.update(job.ID, event)
	})
	if err != nil {
		state := core.StateFailed
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			state = core.StateCancelled
		}
		h.finishError(job.ID, state, err)
		return
	}
	now := time.Now().UTC()
	h.mu.Lock()
	if current := h.jobs[job.ID]; current != nil {
		current.State = core.StateCompleted
		current.Progress = core.ProgressEvent{JobID: job.ID, State: core.StateCompleted, UpdatedAt: now}
		current.Result = &result
		current.UpdatedAt = now
		if h.config.Mode == core.ModeWebCloud {
			current.ExpiresAt = now.Add(h.config.FileTTL)
		}
	}
	h.mu.Unlock()
}

func (h *Handler) update(id string, event core.ProgressEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if job := h.jobs[id]; job != nil {
		job.State = event.State
		job.Progress = event
		job.UpdatedAt = event.UpdatedAt
	}
}
func (h *Handler) finishError(id string, state core.JobState, err error) {
	now := time.Now().UTC()
	h.mu.Lock()
	defer h.mu.Unlock()
	if job := h.jobs[id]; job != nil {
		job.State = state
		if errors.Is(err, context.Canceled) {
			job.ErrorCode, job.Error = "CANCELLED", "O processamento foi cancelado."
		} else if errors.Is(err, context.DeadlineExceeded) {
			job.ErrorCode, job.Error = string(core.CodeUpstreamTimeout), "O processamento excedeu o tempo limite."
		} else {
			code, message, _ := core.RuntimeErrorDetails(err)
			job.ErrorCode, job.Error = string(code), message
		}
		job.Progress = core.ProgressEvent{JobID: id, State: state, Message: job.Error, UpdatedAt: now}
		job.UpdatedAt = now
		if h.config.Mode == core.ModeWebCloud {
			job.ExpiresAt = now.Add(h.config.FileTTL)
		}
	}
}

func (h *Handler) getDownload(w http.ResponseWriter, r *http.Request) {
	job := h.job(r.PathValue("id"))
	if job == nil {
		writeError(w, 404, "NOT_FOUND", "Download não encontrado.")
		return
	}
	writeJSON(w, 200, job)
}
func (h *Handler) cancelDownload(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	job := h.jobs[r.PathValue("id")]
	if job == nil {
		h.mu.Unlock()
		writeError(w, 404, "NOT_FOUND", "Download não encontrado.")
		return
	}
	if job.cancel != nil {
		job.cancel()
	}
	h.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "SSE_UNAVAILABLE", "Streaming indisponível.")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	last := time.Time{}
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()
	for {
		job := h.job(r.PathValue("id"))
		if job == nil {
			fmt.Fprint(w, "event: error\ndata: {\"code\":\"NOT_FOUND\"}\n\n")
			flusher.Flush()
			return
		}
		if job.UpdatedAt.After(last) {
			payload, _ := json.Marshal(job)
			fmt.Fprintf(w, "event: progress\ndata: %s\n\n", payload)
			flusher.Flush()
			last = job.UpdatedAt
		}
		if terminal(job.State) {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (h *Handler) file(w http.ResponseWriter, r *http.Request) {
	job := h.job(r.PathValue("id"))
	if job == nil || job.Result == nil || job.State != core.StateCompleted {
		writeError(w, 404, "FILE_NOT_READY", "Arquivo não está disponível.")
		return
	}
	if !job.ExpiresAt.IsZero() && time.Now().After(job.ExpiresAt) {
		writeError(w, 410, "FILE_EXPIRED", "O arquivo temporário expirou.")
		return
	}
	path := job.Result.FilePath
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		writeError(w, 410, "FILE_EXPIRED", "O arquivo temporário não existe mais.")
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(path)))
	http.ServeFile(w, r, path)
}

func (h *Handler) job(id string) *Job {
	h.mu.RLock()
	defer h.mu.RUnlock()
	source := h.jobs[id]
	if source == nil {
		return nil
	}
	clone := *source
	clone.cancel = nil
	return &clone
}
func (h *Handler) originAllowed(origin string) bool {
	for _, allowed := range h.config.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
func (h *Handler) allowRate(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-h.config.RateWindow)
	h.limitMu.Lock()
	defer h.limitMu.Unlock()
	for key, entries := range h.visits {
		kept := entries[:0]
		for _, entry := range entries {
			if entry.After(cutoff) {
				kept = append(kept, entry)
			}
		}
		if len(kept) == 0 {
			delete(h.visits, key)
		} else {
			h.visits[key] = kept
		}
	}
	allow := func(key string, limit int) bool {
		entries := h.visits[key]
		if len(entries) >= limit {
			return false
		}
		if _, exists := h.visits[key]; !exists && len(h.visits) >= h.config.MaxRateKeys {
			return false
		}
		h.visits[key] = append(entries, now)
		return true
	}
	if !allow("global", h.config.GlobalRateLimit) {
		return false
	}
	return allow("ip:"+ip, h.config.RateLimit)
}
func (h *Handler) evictOldestTerminalLocked() bool {
	var oldest *Job
	for _, job := range h.jobs {
		if terminal(job.State) && (oldest == nil || job.UpdatedAt.Before(oldest.UpdatedAt)) {
			oldest = job
		}
	}
	if oldest == nil {
		return false
	}
	if oldest.Result != nil && oldest.Result.FilePath != "" {
		_ = os.RemoveAll(filepath.Dir(oldest.Result.FilePath))
	}
	delete(h.jobs, oldest.ID)
	return true
}
func (h *Handler) cleanupExpired() {
	now := time.Now()
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, job := range h.jobs {
		if !job.ExpiresAt.IsZero() && now.After(job.ExpiresAt) {
			if job.Result != nil && job.Result.FilePath != "" {
				_ = os.RemoveAll(filepath.Dir(job.Result.FilePath))
			}
			delete(h.jobs, id)
		}
	}
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, 400, "INVALID_JSON", "Corpo JSON inválido.")
		return err
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func securityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
}
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
func terminal(state core.JobState) bool {
	return state == core.StateCompleted || state == core.StateFailed || state == core.StateCancelled || state == core.StateSkipped
}
func newID() (string, error) {
	value := make([]byte, 10)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "job_" + hex.EncodeToString(value), nil
}
func upstreamStatus(code core.ErrorCode) int {
	switch code {
	case core.CodeYouTubeRateLimited, core.CodeYouTubeBotChallenge, core.CodeYouTubePOToken:
		return http.StatusServiceUnavailable
	case core.CodeUpstreamTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusUnprocessableEntity
	}
}

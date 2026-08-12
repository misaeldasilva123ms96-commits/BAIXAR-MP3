package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/api"
	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/core"
	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/runtimeconfig"
)

const version = "3.0.0"

func main() {
	root := runtimeconfig.Env("MP3_DATA_DIR", filepath.Join(os.TempDir(), "mp3-downloader"))
	if err := os.MkdirAll(root, 0o750); err != nil {
		log.Fatal(err)
	}
	tools, status, toolErr := runtimeconfig.FindTools(root)
	if toolErr != nil {
		log.Printf("runtime não pronto: %v", toolErr)
	}
	origins := strings.Split(runtimeconfig.Env("MP3_ALLOWED_ORIGINS", "https://misaeldasilva123ms96-commits.github.io"), ",")
	processor := &core.ExecProcessor{Tools: tools, OutputRoot: filepath.Join(root, "jobs"), TempRoot: filepath.Join(root, "temp"), Cloud: true, MaxItems: runtimeconfig.EnvInt("MP3_MAX_PLAYLIST_ITEMS", 100), MaxOutputBytes: int64(runtimeconfig.EnvInt("MP3_MAX_OUTPUT_MB", 500)) * 1024 * 1024}
	handler := api.NewHandler(api.Config{Mode: core.ModeWebCloud, Version: version, AllowedOrigins: origins, RateLimit: runtimeconfig.EnvInt("MP3_RATE_LIMIT", 30), GlobalRateLimit: runtimeconfig.EnvInt("MP3_GLOBAL_RATE_LIMIT", 300), RateWindow: time.Minute, MaxConcurrent: runtimeconfig.EnvInt("MP3_MAX_CONCURRENT", 2), MaxJobs: runtimeconfig.EnvInt("MP3_MAX_JOBS", 1000), JobTimeout: runtimeconfig.EnvDuration("MP3_JOB_TIMEOUT", 30*time.Minute), FileTTL: runtimeconfig.EnvDuration("MP3_FILE_TTL", 30*time.Minute), Ready: toolErr == nil, Tools: status}, processor)
	address := runtimeconfig.Env("MP3_API_ADDR", ":8080")
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 35 * time.Minute, IdleTimeout: 2 * time.Minute, MaxHeaderBytes: 32 * 1024}
	log.Printf("MP3 Web API v%s ouvindo em %s", version, address)
	log.Fatal(server.ListenAndServe())
}

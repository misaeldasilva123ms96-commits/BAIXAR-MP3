package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/api"
	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/core"
	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/runtimeconfig"
)

const version = "3.0.0"

func main() {
	executable, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	root := filepath.Dir(executable)
	token, err := runtimeconfig.EngineToken()
	if err != nil {
		log.Fatal(err)
	}
	tools, status, toolErr := runtimeconfig.FindTools(root)
	if toolErr != nil {
		log.Printf("preparação necessária: %v", toolErr)
	}
	dataDir := filepath.Join(root, "dados")
	downloads := runtimeconfig.Env("MP3_DOWNLOAD_DIR", defaultDownloads())
	_ = os.MkdirAll(dataDir, 0o750)
	_ = os.MkdirAll(downloads, 0o750)
	settings, err := runtimeconfig.NewFileSettingsStore(filepath.Join(dataDir, "configuracao-v3.json"), downloads)
	if err != nil {
		log.Fatal(err)
	}
	processor := &core.ExecProcessor{Tools: tools, OutputRoot: downloads, OutputRootFunc: func() string { return settings.Get().DownloadDirectory }, TempRoot: filepath.Join(dataDir, "temporarios"), ArchivePath: filepath.Join(dataDir, "historico_downloads.txt"), ArchivePathFunc: func() string {
		if settings.Get().AvoidDuplicates {
			return filepath.Join(dataDir, "historico_downloads.txt")
		}
		return ""
	}, Cloud: false, MaxItems: 500}
	origins := []string{"https://misaeldasilva123ms96-commits.github.io", "http://127.0.0.1:38765", "http://localhost:38765"}
	apiHandler := api.NewHandler(api.Config{Mode: core.ModeDesktopLocal, Version: version, AllowedOrigins: origins, EngineToken: token, RateLimit: 120, RateWindow: time.Minute, MaxConcurrent: 1, JobTimeout: 6 * time.Hour, Ready: toolErr == nil, Tools: status, Settings: settings}, processor)
	webRoot := filepath.Join(root, "web")
	files := http.FileServer(http.Dir(webRoot))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			apiHandler.ServeHTTP(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; img-src 'self' https: data:; connect-src 'self' http://127.0.0.1:38765; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		files.ServeHTTP(w, r)
	})
	address := "127.0.0.1:38765"
	url := "http://" + address + "/#token=" + token
	fmt.Printf("MP3 Downloader v%s\nEngine: %s\nCódigo de conexão: %s\n", version, address, token)
	if os.Getenv("MP3_NO_BROWSER") != "1" {
		go func() {
			time.Sleep(400 * time.Millisecond)
			if err := openBrowser(url); err != nil {
				log.Printf("Abra manualmente %s", url)
			}
		}()
	}
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 2 * time.Minute, IdleTimeout: 2 * time.Minute, MaxHeaderBytes: 32 * 1024}
	log.Fatal(server.ListenAndServe())
}

func isAPIPath(path string) bool {
	return path == "/health" || path == "/version" || path == "/analyze" || path == "/settings" || path == "/downloads" || strings.HasPrefix(path, "/downloads/")
}
func defaultDownloads() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("dados", "downloads")
	}
	return filepath.Join(home, "Downloads", "Musicas_MP3")
}
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

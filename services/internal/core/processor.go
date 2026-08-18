package core

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ExecProcessor struct {
	Tools           ToolPaths
	OutputRoot      string
	TempRoot        string
	ArchivePath     string
	Cloud           bool
	MaxItems        int
	MaxOutputBytes  int64
	OutputRootFunc  func() string
	ArchivePathFunc func() string
	PlayerClients   string
	AnalysisRetries int
	RetryBaseDelay  time.Duration
	Sleep           func(context.Context, time.Duration) error
}

func (p *ExecProcessor) Analyze(ctx context.Context, rawURL string) (Analysis, error) {
	if _, err := ValidateMediaURL(rawURL); err != nil {
		return Analysis{}, err
	}
	if p.Tools.YTDLP == "" {
		return Analysis{}, errors.New("yt-dlp não está disponível")
	}
	maxItems := p.MaxItems
	if maxItems <= 0 {
		maxItems = 100
	}
	args := []string{"--ignore-config", "--dump-single-json", "--skip-download", "--flat-playlist", "--playlist-end", strconv.Itoa(maxItems)}
	args = append(args, runtimeArguments(p.Tools)...)
	args = append(args, youtubeExtractorArguments(p.PlayerClients)...)
	args = append(args, rawURL)
	output, err := p.analyzeWithRetry(ctx, args)
	if err != nil {
		return Analysis{}, err
	}
	var payload struct {
		Type       string  `json:"_type"`
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Uploader   string  `json:"uploader"`
		Channel    string  `json:"channel"`
		Thumbnail  string  `json:"thumbnail"`
		WebpageURL string  `json:"webpage_url"`
		Duration   float64 `json:"duration"`
		Entries    []struct {
			ID        string  `json:"id"`
			Title     string  `json:"title"`
			Uploader  string  `json:"uploader"`
			Channel   string  `json:"channel"`
			Thumbnail string  `json:"thumbnail"`
			Duration  float64 `json:"duration"`
		}
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return Analysis{}, errors.New("resposta inválida do analisador")
	}
	result := Analysis{Type: "video", ID: payload.ID, Title: payload.Title, Artist: first(payload.Uploader, payload.Channel), Duration: payload.Duration, Thumbnail: payload.Thumbnail, WebpageURL: first(payload.WebpageURL, rawURL)}
	if len(payload.Entries) > 0 || payload.Type == "playlist" {
		result.Type, result.PlaylistTitle, result.ItemCount = "playlist", payload.Title, len(payload.Entries)
		for _, entry := range payload.Entries {
			result.Items = append(result.Items, AnalysisItem{ID: entry.ID, Title: entry.Title, Artist: first(entry.Uploader, entry.Channel), Duration: entry.Duration, Thumbnail: entry.Thumbnail})
		}
	}
	return result, nil
}

func (p *ExecProcessor) Start(ctx context.Context, jobID string, request DownloadRequest, emit func(ProgressEvent)) (DownloadResult, error) {
	if err := ValidateDownloadRequest(request); err != nil {
		return DownloadResult{}, err
	}
	if p.Tools.YTDLP == "" {
		return DownloadResult{}, errors.New("yt-dlp não está disponível")
	}
	outputDir := p.OutputRoot
	if p.OutputRootFunc != nil {
		outputDir = p.OutputRootFunc()
	}
	archive := p.ArchivePath
	if p.ArchivePathFunc != nil {
		archive = p.ArchivePathFunc()
	}
	if p.Cloud {
		outputDir = filepath.Join(p.OutputRoot, jobID)
		archive = ""
	}
	tempDir := filepath.Join(p.TempRoot, jobID)
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return DownloadResult{}, err
	}
	if err := os.MkdirAll(tempDir, 0o750); err != nil {
		return DownloadResult{}, err
	}
	args, err := BuildYTDLPArguments(request, p.Tools, outputDir, tempDir, archive)
	if err != nil {
		return DownloadResult{}, err
	}
	if p.Cloud && p.MaxItems > 0 && request.PlaylistEnd == 0 {
		args = append(args[:len(args)-1], "--playlist-end", strconv.Itoa(p.MaxItems), args[len(args)-1])
	}
	target := args[len(args)-1]
	args = append(args[:len(args)-1], youtubeExtractorArguments(p.PlayerClients)...)
	args = append(args, target)
	cmd := exec.Command(p.Tools.YTDLP, args...) //nolint:noctx // cancellation must terminate the complete process tree below
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	processTree, err := prepareProcessTree(cmd)
	if err != nil {
		return DownloadResult{}, err
	}
	defer processTree.close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return DownloadResult{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return DownloadResult{}, err
	}
	if err := cmd.Start(); err != nil {
		return DownloadResult{}, err
	}
	if err := processTree.attach(cmd.Process); err != nil {
		_ = processTree.terminate(cmd.Process)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return DownloadResult{}, fmt.Errorf("não foi possível proteger a árvore do processo: %w", err)
	}
	var outputLimitExceeded atomic.Bool
	var outputScanFailed atomic.Bool
	var youtubeBotChallenge atomic.Bool
	var stderrMu sync.Mutex
	var stderrSample strings.Builder

	var mu sync.Mutex
	files := []string{}
	parse := func(reader io.Reader, captureErrors bool) {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if captureErrors {
				stderrMu.Lock()
				if stderrSample.Len() < 16*1024 {
					remaining := 16*1024 - stderrSample.Len()
					if len(line) > remaining {
						line = line[:remaining]
					}
					stderrSample.WriteString(line)
					stderrSample.WriteByte('\n')
				}
				stderrMu.Unlock()
			}
			if isYouTubeBotChallenge(line) {
				youtubeBotChallenge.Store(true)
			}
			event := ProgressEvent{State: StateDownloading, UpdatedAt: time.Now().UTC()}
			switch {
			case strings.HasPrefix(line, "MP3_PROGRESS|"):
				parts := strings.SplitN(line, "|", 5)
				if len(parts) > 1 {
					if value, parseErr := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(parts[1], "%")), 64); parseErr == nil && value < 100 {
						event.Percent = &value
					}
				}
				if len(parts) > 2 {
					event.Speed = cleanUnknown(parts[2])
				}
				if len(parts) > 3 {
					event.ETA = cleanUnknown(parts[3])
				}
				if len(parts) > 4 {
					event.Size = cleanUnknown(parts[4])
				}
				emit(event)
			case strings.HasPrefix(line, "MP3_ITEM|"):
				parts := strings.SplitN(line, "|", 4)
				if len(parts) > 1 {
					event.Item, _ = strconv.Atoi(parts[1])
				}
				if len(parts) > 2 {
					event.Total, _ = strconv.Atoi(parts[2])
				}
				if len(parts) > 3 {
					event.Title = parts[3]
				}
				emit(event)
			case strings.HasPrefix(line, "MP3_RESULT|"):
				path := strings.TrimPrefix(line, "MP3_RESULT|")
				if isWithin(path, outputDir) {
					mu.Lock()
					files = append(files, path)
					mu.Unlock()
				}
			case strings.Contains(strings.ToLower(line), "post-process") || strings.Contains(strings.ToLower(line), "extractaudio"):
				event.State = StateConverting
				emit(event)
			case strings.Contains(strings.ToLower(line), "metadata") || strings.Contains(strings.ToLower(line), "thumbnail"):
				event.State = StateAddingMetadata
				emit(event)
			}
		}
	}
	var readers sync.WaitGroup
	readers.Add(2)
	go func() { defer readers.Done(); parse(stdout, false) }()
	go func() { defer readers.Done(); parse(stderr, true) }()
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	var ticker *time.Ticker
	var quotaTick <-chan time.Time
	if p.Cloud && p.MaxOutputBytes > 0 {
		ticker = time.NewTicker(250 * time.Millisecond)
		quotaTick = ticker.C
		defer ticker.Stop()
	}
	waiting := true
	for waiting {
		select {
		case err = <-waitDone:
			waiting = false
		case <-ctx.Done():
			_ = processTree.terminate(cmd.Process)
			err = <-waitDone
			waiting = false
		case <-quotaTick:
			size, sizeErr := directorySize(outputDir)
			if sizeErr != nil || size > p.MaxOutputBytes {
				outputScanFailed.Store(sizeErr != nil)
				outputLimitExceeded.Store(sizeErr == nil)
				_ = processTree.terminate(cmd.Process)
				err = <-waitDone
				waiting = false
			}
		}
	}
	readers.Wait()
	_ = os.RemoveAll(tempDir)
	if outputScanFailed.Load() {
		_ = os.RemoveAll(outputDir)
		return DownloadResult{}, errors.New("não foi possível contabilizar o tamanho do resultado")
	}
	if outputLimitExceeded.Load() {
		_ = os.RemoveAll(outputDir)
		return DownloadResult{}, errors.New("o resultado excedeu o limite de tamanho")
	}
	if ctx.Err() != nil {
		return DownloadResult{}, ctx.Err()
	}
	if err != nil {
		stderrMu.Lock()
		failureOutput := stderrSample.String()
		stderrMu.Unlock()
		if youtubeBotChallenge.Load() && failureOutput == "" {
			failureOutput = "sign in to confirm you're not a bot"
		}
		return DownloadResult{}, classifyYTDLPError(err, failureOutput)
	}
	mu.Lock()
	completedFiles := append([]string(nil), files...)
	mu.Unlock()
	if len(completedFiles) == 0 {
		return DownloadResult{}, errors.New("nenhum arquivo foi gerado")
	}
	resultPath := completedFiles[0]
	if p.Cloud && len(completedFiles) > 1 {
		resultPath = filepath.Join(outputDir, "playlist.zip")
		if err := zipFiles(resultPath, completedFiles); err != nil {
			_ = os.RemoveAll(outputDir)
			return DownloadResult{}, err
		}
	}
	if p.MaxOutputBytes > 0 {
		total, sizeErr := directorySize(outputDir)
		if sizeErr != nil {
			_ = os.RemoveAll(outputDir)
			return DownloadResult{}, fmt.Errorf("não foi possível contabilizar o tamanho do resultado: %w", sizeErr)
		}
		if total > p.MaxOutputBytes {
			_ = os.RemoveAll(outputDir)
			return DownloadResult{}, errors.New("o resultado excedeu o limite de tamanho")
		}
	}
	info, err := os.Stat(resultPath)
	if err != nil {
		return DownloadResult{}, err
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(resultPath)), ".")
	return DownloadResult{Title: strings.TrimSuffix(filepath.Base(resultPath), filepath.Ext(resultPath)), Format: format, Quality: request.Quality, FileName: filepath.Base(resultPath), FilePath: resultPath, Size: info.Size(), Count: len(completedFiles)}, nil
}

func runtimeArguments(tools ToolPaths) []string {
	if tools.Deno == "" {
		return nil
	}
	return []string{"--js-runtimes", "deno:" + tools.Deno}
}

func youtubeExtractorArguments(clients string) []string {
	clients = strings.TrimSpace(clients)
	if clients == "" {
		return nil
	}
	for _, client := range strings.Split(clients, ",") {
		switch strings.TrimSpace(client) {
		case "default", "web", "web_embedded", "mweb", "android_vr", "web_safari":
		default:
			return nil
		}
	}
	return []string{"--extractor-args", "youtube:player_client=" + clients}
}

func (p *ExecProcessor) analyzeWithRetry(ctx context.Context, args []string) ([]byte, error) {
	attempts := p.AnalysisRetries
	if attempts <= 0 {
		attempts = 2
	}
	base := p.RetryBaseDelay
	if base <= 0 {
		base = 400 * time.Millisecond
	}
	sleep := p.Sleep
	if sleep == nil {
		sleep = func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		}
	}
	for attempt := 0; attempt < attempts; attempt++ {
		cmd := exec.CommandContext(ctx, p.Tools.YTDLP, args...)
		cmd.Env = append(os.Environ(), "NO_COLOR=1")
		output, err := cmd.Output()
		if err == nil {
			return output, nil
		}
		failure := classifyYTDLPError(err, "")
		if !failure.Retryable || attempt == attempts-1 {
			return nil, failure
		}
		jitter := time.Duration(rand.Int64N(int64(base/2) + 1))
		if err := sleep(ctx, base*time.Duration(1<<attempt)+jitter); err != nil {
			return nil, classifyYTDLPError(err, "")
		}
	}
	return nil, &RuntimeError{Code: CodeExtractorFailed, Message: "O extrator não conseguiu processar este conteúdo."}
}

func isYouTubeBotChallenge(value string) bool {
	normalized := strings.ToLower(value)
	return strings.Contains(normalized, "sign in to confirm you’re not a bot") ||
		strings.Contains(normalized, "sign in to confirm you're not a bot")
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
func cleanUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "NA" || value == "Unknown" {
		return ""
	}
	return value
}
func isWithin(path, root string) bool {
	p, e1 := filepath.Abs(path)
	r, e2 := filepath.Abs(root)
	if e1 != nil || e2 != nil {
		return false
	}
	rel, err := filepath.Rel(r, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func directorySize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		total += info.Size()
		return nil
	})
	return total, err
}
func zipFiles(destination string, files []string) error {
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o640)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(out)
	for _, path := range files {
		in, openErr := os.Open(path)
		if openErr != nil {
			_ = zw.Close()
			_ = out.Close()
			return openErr
		}
		entry, createErr := zw.Create(SafeOutputName(filepath.Base(path)))
		if createErr == nil {
			_, createErr = io.Copy(entry, in)
		}
		_ = in.Close()
		if createErr != nil {
			_ = zw.Close()
			_ = out.Close()
			return createErr
		}
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

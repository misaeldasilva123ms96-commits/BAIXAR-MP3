package runtimeconfig

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/core"
)

func Env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
func EnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(Env(key, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
func EnvDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(Env(key, ""))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func FindTools(root string) (core.ToolPaths, map[string]string, error) {
	dir := Env("MP3_TOOLS_DIR", filepath.Join(root, "ferramentas"))
	ytdlp := Env("YTDLP_PATH", findExecutable(dir, exeName("yt-dlp")))
	ffmpeg := Env("FFMPEG_PATH", findExecutable(dir, exeName("ffmpeg")))
	ffprobe := Env("FFPROBE_PATH", findExecutable(dir, exeName("ffprobe")))
	deno := Env("DENO_PATH", findExecutable(dir, exeName("deno")))
	status := map[string]string{"yt-dlp": toolVersion(ytdlp, "--version"), "ffmpeg": toolVersion(ffmpeg, "-version"), "ffprobe": toolVersion(ffprobe, "-version"), "deno": toolVersion(deno, "--version")}
	missing := []string{}
	for name, value := range status {
		if value == "indisponível" {
			missing = append(missing, name)
		}
	}
	paths := core.ToolPaths{YTDLP: ytdlp, FFmpegDir: filepath.Dir(ffmpeg), Deno: deno}
	if len(missing) > 0 {
		return paths, status, errors.New("ferramentas ausentes: " + strings.Join(missing, ", "))
	}
	return paths, status, nil
}

func findExecutable(dir, name string) string {
	candidate := filepath.Join(dir, name)
	if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
		return candidate
	}
	if path, err := exec.LookPath(strings.TrimSuffix(name, ".exe")); err == nil {
		return path
	}
	return candidate
}
func exeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
func toolVersion(path, arg string) string {
	if path == "" {
		return "indisponível"
	}
	output, err := exec.Command(path, arg).CombinedOutput()
	if err != nil {
		return "indisponível"
	}
	line := strings.Split(strings.TrimSpace(string(output)), "\n")[0]
	if len(line) > 120 {
		line = line[:120]
	}
	return line
}

func EngineToken() (string, error) {
	if value := strings.TrimSpace(os.Getenv("MP3_ENGINE_TOKEN")); len(value) >= 32 {
		return value, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "MP3Downloader")
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "engine-token.txt")
	if value, readErr := os.ReadFile(path); readErr == nil && len(strings.TrimSpace(string(value))) >= 32 {
		return strings.TrimSpace(string(value)), nil
	}
	value := make([]byte, 24)
	if _, err = rand.Read(value); err != nil {
		return "", err
	}
	token := hex.EncodeToString(value)
	if err = os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", err
	}
	return token, nil
}

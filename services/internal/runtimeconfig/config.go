package runtimeconfig

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/core"
)

var toolVersionTimeout = 10 * time.Second

const minimumYTDLPVersion = "2026.07.04"
const minimumDenoVersion = "2.3.0"

var ejsVersionPattern = regexp.MustCompile(`yt_dlp_ejs-([0-9]+(?:\.[0-9]+)+)`)

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
	invalid := validateToolVersions(status)
	ejs := ytdlpRuntimeDiagnostic(ytdlp, deno)
	status["EJS"] = ejs
	if ejs == "indisponível" {
		invalid = append(invalid, "EJS/Deno")
	}
	if len(invalid) > 0 {
		return paths, status, errors.New("runtime incompatível: " + strings.Join(invalid, ", "))
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
	ctx, cancel := context.WithTimeout(context.Background(), toolVersionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, arg)
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "indisponível"
	}
	line := strings.Split(strings.TrimSpace(string(output)), "\n")[0]
	if len(line) > 120 {
		line = line[:120]
	}
	return line
}

func validateToolVersions(status map[string]string) []string {
	var invalid []string
	ytdlpFields := strings.Fields(status["yt-dlp"])
	ytdlpVersion := ""
	if len(ytdlpFields) > 0 {
		ytdlpVersion = ytdlpFields[0]
	}
	if !versionAtLeast(ytdlpVersion, minimumYTDLPVersion) {
		status["yt-dlp"] += " (incompatível; mínimo " + minimumYTDLPVersion + ")"
		invalid = append(invalid, "yt-dlp")
	}
	denoFields := strings.Fields(status["deno"])
	denoVersion := ""
	if len(denoFields) >= 2 {
		denoVersion = denoFields[1]
	}
	if !versionAtLeast(denoVersion, minimumDenoVersion) {
		status["deno"] += " (incompatível; mínimo " + minimumDenoVersion + ")"
		invalid = append(invalid, "deno")
	}
	for _, name := range []string{"ffmpeg", "ffprobe"} {
		if !strings.HasPrefix(strings.ToLower(status[name]), name+" version ") {
			status[name] += " (versão inválida)"
			invalid = append(invalid, name)
		}
	}
	return invalid
}

func versionAtLeast(value, minimum string) bool {
	valueParts := strings.Split(strings.TrimPrefix(value, "v"), ".")
	minimumParts := strings.Split(strings.TrimPrefix(minimum, "v"), ".")
	if len(valueParts) != len(minimumParts) {
		return false
	}
	for i := range minimumParts {
		valuePart, valueErr := strconv.Atoi(valueParts[i])
		minimumPart, minimumErr := strconv.Atoi(minimumParts[i])
		if valueErr != nil || minimumErr != nil {
			return false
		}
		if valuePart > minimumPart {
			return true
		}
		if valuePart < minimumPart {
			return false
		}
	}
	return true
}

func ytdlpRuntimeDiagnostic(ytdlp, deno string) string {
	ctx, cancel := context.WithTimeout(context.Background(), toolVersionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ytdlp, "--verbose", "--ignore-config", "--js-runtimes", "deno:"+deno, "--simulate", "file:///__mp3_runtime_probe__")
	cmd.Stdin = nil
	output, _ := cmd.CombinedOutput()
	return parseYTDLPRuntimeDiagnostic(string(output))
}

func parseYTDLPRuntimeDiagnostic(value string) string {
	match := ejsVersionPattern.FindStringSubmatch(value)
	if len(match) != 2 || !strings.Contains(value, "JS runtimes: deno-") {
		return "indisponível"
	}
	return "yt_dlp_ejs-" + match[1] + " (Deno detectado)"
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

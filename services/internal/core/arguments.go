package core

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

func QualityArguments() map[Quality]string {
	return map[Quality]string{QualityVBR0: "0", Quality320: "320K", Quality256: "256K", Quality192: "192K", Quality128: "128K"}
}

func BuildYTDLPArguments(request DownloadRequest, tools ToolPaths, outputDir, tempDir, archivePath string) ([]string, error) {
	if err := ValidateDownloadRequest(request); err != nil {
		return nil, err
	}
	if outputDir == "" || tempDir == "" {
		return nil, errors.New("diretórios de saída inválidos")
	}
	quality := QualityArguments()[request.Quality]
	template := "%(playlist_index&{} - |)s%(title)s.%(ext)s"
	if *request.OrganizePlaylist {
		template = "%(playlist_title&{}/|)s%(playlist_index&{} - |)s%(title)s.%(ext)s"
	}
	args := []string{
		"--ignore-config", "--yes-playlist", "--ignore-errors", "--continue", "--no-overwrites",
		"--windows-filenames", "--trim-filenames", "180", "--format", "bestaudio/best",
		"--extract-audio", "--audio-format", "mp3", "--audio-quality", quality,
		"--retries", "3", "--fragment-retries", "3", "--extractor-retries", "2",
		"--retry-sleep", "http:exp=1:8", "--retry-sleep", "fragment:exp=1:8", "--retry-sleep", "extractor:exp=1:4", "--concurrent-fragments", "3", "--newline",
		"--progress-template", "download:MP3_PROGRESS|%(progress._percent_str)s|%(progress._speed_str)s|%(progress._eta_str)s|%(progress._total_bytes_str)s",
		"--print", "before_dl:MP3_ITEM|%(playlist_index)s|%(playlist_count)s|%(title)s",
		"--print", "after_move:MP3_RESULT|%(filepath)s", "-P", outputDir, "-P", "temp:" + tempDir, "-o", template,
	}
	if tools.FFmpegDir != "" {
		args = append(args, "--ffmpeg-location", tools.FFmpegDir)
	}
	args = append(args, runtimeArguments(tools)...)
	if archivePath != "" {
		args = append(args, "--download-archive", archivePath)
	}
	embedThumbnail := request.EmbedThumbnail == nil || *request.EmbedThumbnail
	embedMetadata := request.EmbedMetadata == nil || *request.EmbedMetadata
	if embedThumbnail {
		args = append(args, "--embed-thumbnail", "--convert-thumbnails", "jpg")
	}
	if embedMetadata {
		args = append(args, "--embed-metadata")
	}
	if request.PlaylistStart > 0 {
		args = append(args, "--playlist-start", strconv.Itoa(request.PlaylistStart))
	}
	if request.PlaylistEnd > 0 {
		args = append(args, "--playlist-end", strconv.Itoa(request.PlaylistEnd))
	}
	return append(args, request.URL), nil
}

func SafeOutputName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\\|?*`, r) {
			return '_'
		}
		if unicode.IsControl(r) {
			return '_'
		}
		return r
	}, value)
	value = strings.Trim(value, " .")
	if len([]rune(value)) > 120 {
		value = string([]rune(value)[:120])
	}
	if value == "" {
		return "audio"
	}
	return value
}

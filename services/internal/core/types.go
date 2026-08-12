package core

import "time"

type Mode string

const (
	ModeWebCloud     Mode = "WEB_CLOUD"
	ModeLocalEngine  Mode = "LOCAL_ENGINE"
	ModeDesktopLocal Mode = "DESKTOP_LOCAL"
)

type Quality string

const (
	QualityVBR0 Quality = "vbr0"
	Quality320  Quality = "320"
	Quality256  Quality = "256"
	Quality192  Quality = "192"
	Quality128  Quality = "128"
)

type JobState string

const (
	StateAnalyzing      JobState = "ANALYZING"
	StateQueued         JobState = "QUEUED"
	StateDownloading    JobState = "DOWNLOADING"
	StateConverting     JobState = "CONVERTING"
	StateAddingMetadata JobState = "ADDING_METADATA"
	StateFinalizing     JobState = "FINALIZING"
	StateCompleted      JobState = "COMPLETED"
	StateFailed         JobState = "FAILED"
	StateCancelled      JobState = "CANCELLED"
	StateSkipped        JobState = "SKIPPED"
)

type AnalysisItem struct {
	ID        string  `json:"id,omitempty"`
	Title     string  `json:"title,omitempty"`
	Artist    string  `json:"artist,omitempty"`
	Duration  float64 `json:"duration,omitempty"`
	Thumbnail string  `json:"thumbnail,omitempty"`
}

type Analysis struct {
	Type          string         `json:"type"`
	ID            string         `json:"id,omitempty"`
	Title         string         `json:"title,omitempty"`
	Artist        string         `json:"artist,omitempty"`
	Duration      float64        `json:"duration,omitempty"`
	Thumbnail     string         `json:"thumbnail,omitempty"`
	WebpageURL    string         `json:"webpageUrl,omitempty"`
	PlaylistTitle string         `json:"playlistTitle,omitempty"`
	ItemCount     int            `json:"itemCount,omitempty"`
	Items         []AnalysisItem `json:"items,omitempty"`
}

type DownloadRequest struct {
	URL              string  `json:"url"`
	Quality          Quality `json:"quality"`
	PlaylistStart    int     `json:"playlistStart,omitempty"`
	PlaylistEnd      int     `json:"playlistEnd,omitempty"`
	OrganizePlaylist bool    `json:"organizePlaylist"`
	EmbedThumbnail   *bool   `json:"embedThumbnail,omitempty"`
	EmbedMetadata    *bool   `json:"embedMetadata,omitempty"`
}

type ProgressEvent struct {
	JobID     string    `json:"jobId,omitempty"`
	State     JobState  `json:"state"`
	Item      int       `json:"item,omitempty"`
	Total     int       `json:"total,omitempty"`
	Title     string    `json:"title,omitempty"`
	Thumbnail string    `json:"thumbnail,omitempty"`
	Percent   *float64  `json:"percent,omitempty"`
	Speed     string    `json:"speed,omitempty"`
	ETA       string    `json:"eta,omitempty"`
	Size      string    `json:"size,omitempty"`
	Message   string    `json:"message,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DownloadResult struct {
	Title    string  `json:"title,omitempty"`
	Format   string  `json:"format"`
	Quality  Quality `json:"quality,omitempty"`
	FileName string  `json:"fileName,omitempty"`
	FilePath string  `json:"-"`
	Size     int64   `json:"size,omitempty"`
	Count    int     `json:"count,omitempty"`
}

type ToolPaths struct {
	YTDLP     string
	FFmpegDir string
	Deno      string
}

type Settings struct {
	DefaultQuality     Quality `json:"defaultQuality"`
	DownloadDirectory  string  `json:"downloadDirectory"`
	OrganizePlaylist   bool    `json:"organizePlaylist"`
	AvoidDuplicates    bool    `json:"avoidDuplicates"`
	EmbedThumbnail     bool    `json:"embedThumbnail"`
	EmbedMetadata      bool    `json:"embedMetadata"`
	OpenFolderWhenDone bool    `json:"openFolderWhenDone"`
}

export type ProcessingMode = 'WEB_CLOUD' | 'LOCAL_ENGINE' | 'DESKTOP_LOCAL';
export type Quality = 'vbr0' | '320' | '256' | '192' | '128';
export type JobState = 'ANALYZING' | 'QUEUED' | 'DOWNLOADING' | 'CONVERTING' | 'ADDING_METADATA' | 'FINALIZING' | 'COMPLETED' | 'FAILED' | 'CANCELLED' | 'SKIPPED';

export interface AnalysisItem { id?: string; title?: string; artist?: string; duration?: number; thumbnail?: string }
export interface Analysis {
  type: 'video' | 'playlist'; id?: string; title?: string; artist?: string; duration?: number;
  thumbnail?: string; webpageUrl?: string; playlistTitle?: string; itemCount?: number; items?: AnalysisItem[];
}
export interface DownloadRequest {
  url: string; quality: Quality; playlistStart?: number; playlistEnd?: number;
  organizePlaylist: boolean; embedThumbnail?: boolean; embedMetadata?: boolean;
}
export interface ProgressEvent {
  jobId?: string; state: JobState; item?: number; total?: number; title?: string; thumbnail?: string;
  percent?: number; speed?: string; eta?: string; size?: string; message?: string; updatedAt?: string;
}
export interface DownloadResult { title?: string; format: string; quality?: Quality; fileName?: string; size?: number; count?: number }
export interface DownloadJob {
  id: string; mode: ProcessingMode; state: JobState; request: DownloadRequest; progress: ProgressEvent;
  result?: DownloadResult; error?: string; errorCode?: string; createdAt: string; updatedAt: string; expiresAt?: string;
}
export interface ToolStatus { [name: string]: string }
export interface Settings { defaultQuality: Quality; downloadDirectory: string; organizePlaylist: boolean; avoidDuplicates: boolean; embedThumbnail: boolean; embedMetadata: boolean; openFolderWhenDone: boolean }
export type EngineDetectionState = 'NOT_DETECTED' | 'PERMISSION_REQUIRED' | 'REACHABLE' | 'TOOLS_NOT_READY' | 'READY' | 'AUTH_REQUIRED';
export interface EngineDetection { state: EngineDetectionState; available: boolean; reachable: boolean; baseUrl?: string; version?: string; tools?: ToolStatus; reason?: 'timeout' | 'network' | 'invalid_response' | 'permission' }

export interface DownloadProvider {
  readonly mode: ProcessingMode;
  readonly baseUrl: string;
  health(): Promise<{ status: string; mode: ProcessingMode; version: string; ready: boolean; tools?: ToolStatus }>;
  analyze(url: string): Promise<Analysis>;
  download(request: DownloadRequest): Promise<DownloadJob>;
  getProgress(id: string): Promise<DownloadJob>;
  cancel(id: string): Promise<void>;
  eventsUrl(id: string): string;
  downloadFile(id: string): Promise<void>;
  getSettings(): Promise<Settings>;
  saveSettings(settings: Settings): Promise<Settings>;
}

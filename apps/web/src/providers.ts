import type { Analysis, DownloadJob, DownloadProvider, DownloadRequest, EngineDetection, ProcessingMode, Settings, ToolStatus } from './contracts';

const LOCAL_ENGINE_URL = 'http://127.0.0.1:38765';

export class ProviderError extends Error {
  constructor(message: string, readonly code = 'REQUEST_FAILED', readonly status = 0) { super(message); }
}

export class HttpDownloadProvider implements DownloadProvider {
  constructor(readonly mode: ProcessingMode, readonly baseUrl: string, private readonly token = '') {}

  private async fetchResponse(path: string, init: RequestInit = {}): Promise<Response> {
    const headers: Record<string, string> = { Accept: 'application/json', ...(init.body ? { 'Content-Type': 'application/json' } : {}) };
    if (this.mode === 'LOCAL_ENGINE' || this.mode === 'DESKTOP_LOCAL') {
      if (this.token) headers['X-MP3-Engine-Token'] = this.token;
    }
    const response = await fetch(`${this.baseUrl}${path}`, { ...init, headers: { ...headers, ...(init.headers as Record<string, string> | undefined) } });
    if (!response.ok) {
      let message = `A solicitação falhou (${response.status}).`;
      let code = 'REQUEST_FAILED';
      try { const body = await response.json() as { error?: { message?: string; code?: string } }; message = body.error?.message || message; code = body.error?.code || code; } catch { /* response was not JSON */ }
      throw new ProviderError(message, code, response.status);
    }
    return response;
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await this.fetchResponse(path, init);
    if (response.status === 204) return undefined as T;
    return response.json() as Promise<T>;
  }

  health() { return this.request<{ status: string; mode: ProcessingMode; version: string; ready: boolean; tools?: ToolStatus }>('/health'); }
  analyze(url: string) { return this.request<Analysis>('/analyze', { method: 'POST', body: JSON.stringify({ url }) }); }
  download(request: DownloadRequest) { return this.request<DownloadJob>('/downloads', { method: 'POST', body: JSON.stringify(request) }); }
  getProgress(id: string) { return this.request<DownloadJob>(`/downloads/${encodeURIComponent(id)}`); }
  cancel(id: string) { return this.request<void>(`/downloads/${encodeURIComponent(id)}`, { method: 'DELETE' }); }
  eventsUrl(id: string) { return `${this.baseUrl}/downloads/${encodeURIComponent(id)}/events`; }
  async downloadFile(id: string) {
    const response = await this.fetchResponse(`/downloads/${encodeURIComponent(id)}/file`);
    const blobUrl = URL.createObjectURL(await response.blob());
    const disposition = response.headers.get('Content-Disposition') || '';
    const fileName = disposition.match(/filename="?([^";]+)"?/i)?.[1] || `download-${id}.mp3`;
    const link = document.createElement('a');
    link.href = blobUrl;
    link.download = fileName;
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(blobUrl);
  }
  getSettings() { return this.request<Settings>('/settings'); }
  saveSettings(settings: Settings) { return this.request<Settings>('/settings', { method: 'PUT', body: JSON.stringify(settings) }); }
}

export async function detectLocalEngine(timeoutMs = 1200): Promise<EngineDetection> {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(`${LOCAL_ENGINE_URL}/health`, { signal: controller.signal, headers: { Accept: 'application/json' } });
    if (!response.ok) return { available: false };
    const body = await response.json() as { status?: string; mode?: string; version?: string; ready?: boolean };
    if (body.status !== 'ok' || (body.mode !== 'LOCAL_ENGINE' && body.mode !== 'DESKTOP_LOCAL')) return { available: false };
    return { available: body.ready === true, reachable: true, version: body.version, tools: (body as { tools?: ToolStatus }).tools };
  } catch { return { available: false }; }
  finally { window.clearTimeout(timeout); }
}

export function cloudProvider(): HttpDownloadProvider {
  const configured = import.meta.env.VITE_MP3_API_BASE_URL?.trim();
  return new HttpDownloadProvider('WEB_CLOUD', configured || '');
}

export function localProvider(token: string, desktop = false): HttpDownloadProvider {
  const base = desktop ? '' : LOCAL_ENGINE_URL;
  return new HttpDownloadProvider(desktop ? 'DESKTOP_LOCAL' : 'LOCAL_ENGINE', base, token);
}

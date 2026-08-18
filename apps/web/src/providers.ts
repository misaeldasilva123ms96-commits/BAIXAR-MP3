import type { Analysis, DownloadJob, DownloadProvider, DownloadRequest, EngineDetection, ProcessingMode, Settings, ToolStatus } from './contracts';

const LOCAL_ENGINE_URL = 'http://127.0.0.1:38765';
const LOCAL_ENGINE_FALLBACK_URL = 'http://localhost:38765';

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

type DetectOptions = { timeoutMs?: number; attempts?: number; interactive?: boolean };

async function fetchEngineHealth(baseUrl: string, timeoutMs: number): Promise<EngineDetection> {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(`${baseUrl}/health`, { signal: controller.signal, headers: { Accept: 'application/json' }, cache: 'no-store', credentials: 'omit' });
    if (response.status === 401 || response.status === 403) return { state: 'AUTH_REQUIRED', available: false, reachable: true, baseUrl };
    if (!response.ok) return { state: 'REACHABLE', available: false, reachable: true, baseUrl, reason: 'invalid_response' };
    const body = await response.json() as { status?: string; mode?: string; version?: string; ready?: boolean; tools?: ToolStatus };
    if (body.status !== 'ok' || (body.mode !== 'LOCAL_ENGINE' && body.mode !== 'DESKTOP_LOCAL')) return { state: 'REACHABLE', available: false, reachable: true, baseUrl, reason: 'invalid_response' };
    return { state: body.ready === true ? 'READY' : 'TOOLS_NOT_READY', available: body.ready === true, reachable: true, baseUrl, version: body.version, tools: body.tools || {} };
  } catch (error) {
    return { state: 'NOT_DETECTED', available: false, reachable: false, baseUrl, reason: error instanceof DOMException && error.name === 'AbortError' ? 'timeout' : 'network' };
  } finally { window.clearTimeout(timeout); }
}

async function loopbackPermissionState(): Promise<PermissionState | undefined> {
  if (!navigator.permissions?.query) return undefined;
  for (const name of ['loopback-network', 'local-network-access']) {
    try { return (await navigator.permissions.query({ name } as PermissionDescriptor)).state; } catch { /* try the compatibility alias */ }
  }
  return undefined;
}

export async function detectLocalEngine(options: number | DetectOptions = {}): Promise<EngineDetection> {
  const settings = typeof options === 'number' ? { timeoutMs: options } : options;
  const timeoutMs = settings.timeoutMs ?? 1200;
  const attempts = Math.max(1, Math.min(settings.attempts ?? 1, 2));
  let last: EngineDetection = { state: 'NOT_DETECTED', available: false, reachable: false, reason: 'network' };
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    for (const baseUrl of [LOCAL_ENGINE_URL, LOCAL_ENGINE_FALLBACK_URL]) {
      const result = await fetchEngineHealth(baseUrl, timeoutMs);
      if (result.reachable) return result;
      last = result;
    }
    if (attempt + 1 < attempts) await new Promise((resolve) => window.setTimeout(resolve, 150));
  }
  const permission = await loopbackPermissionState();
  if (settings.interactive && permission && permission !== 'granted') return { ...last, state: 'PERMISSION_REQUIRED', reason: 'permission' };
  return last;
}

export function cloudProvider(): HttpDownloadProvider {
  const configured = import.meta.env.VITE_MP3_API_BASE_URL?.trim();
  return new HttpDownloadProvider('WEB_CLOUD', configured || '');
}

export function localProvider(token: string, desktop = false, detectedBaseUrl = LOCAL_ENGINE_URL): HttpDownloadProvider {
  const base = desktop ? '' : detectedBaseUrl;
  return new HttpDownloadProvider(desktop ? 'DESKTOP_LOCAL' : 'LOCAL_ENGINE', base, token);
}

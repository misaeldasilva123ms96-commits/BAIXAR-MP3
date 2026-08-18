import { beforeEach, describe, expect, it, vi } from 'vitest';
import { HttpDownloadProvider, detectLocalEngine } from './providers';

describe('HttpDownloadProvider', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('uses the real analysis returned by the selected backend', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ type: 'video', title: 'Título real' }), { status: 200 }));
    const provider = new HttpDownloadProvider('WEB_CLOUD', 'https://api.example.test');
    await expect(provider.analyze('https://youtu.be/abc')).resolves.toMatchObject({ title: 'Título real' });
    expect(fetch).toHaveBeenCalledWith('https://api.example.test/analyze', expect.objectContaining({ method: 'POST' }));
  });

  it('sends the engine token only to the configured local engine', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ id: 'job_1' }), { status: 202 }));
    const provider = new HttpDownloadProvider('LOCAL_ENGINE', 'http://127.0.0.1:38765', 'secret');
    await provider.download({ url: 'https://youtu.be/abc', quality: 'vbr0', organizePlaylist: true });
    expect(fetch).toHaveBeenCalledWith('http://127.0.0.1:38765/downloads', expect.objectContaining({ headers: expect.objectContaining({ 'X-MP3-Engine-Token': 'secret' }) }));
  });

  it('downloads local files through the authenticated fetch flow', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(new Blob(['mp3']), { status: 200, headers: { 'Content-Disposition': 'attachment; filename="track.mp3"' } }));
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: vi.fn(() => 'blob:test') });
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() });
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined);
    const provider = new HttpDownloadProvider('LOCAL_ENGINE', 'http://127.0.0.1:38765', 'secret');
    await provider.downloadFile('job_1');
    expect(fetch).toHaveBeenCalledWith('http://127.0.0.1:38765/downloads/job_1/file', expect.objectContaining({ headers: expect.objectContaining({ 'X-MP3-Engine-Token': 'secret' }) }));
    expect(click).toHaveBeenCalledOnce();
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:test');
  });
});

describe('detectLocalEngine', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('reports local mode only after a valid health response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ status: 'ok', mode: 'LOCAL_ENGINE', version: '3.0.0', ready: true }), { status: 200 }));
    await expect(detectLocalEngine()).resolves.toMatchObject({ state: 'READY', available: true, reachable: true, version: '3.0.0', baseUrl: 'http://127.0.0.1:38765' });
  });

  it('keeps cloud available when the engine cannot be reached', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('network error'));
    await expect(detectLocalEngine()).resolves.toMatchObject({ state: 'NOT_DETECTED', available: false, reachable: false, reason: 'network' });
  });

  it('falls back to localhost when the numeric loopback address is unavailable', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      if (String(input).startsWith('http://127.0.0.1')) throw new TypeError('blocked');
      return new Response(JSON.stringify({ status: 'ok', mode: 'LOCAL_ENGINE', version: '3.0.1', ready: true }), { status: 200 });
    });
    await expect(detectLocalEngine()).resolves.toMatchObject({ state: 'READY', baseUrl: 'http://localhost:38765' });
  });

  it('distinguishes a reachable engine with incomplete tools', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ status: 'ok', mode: 'LOCAL_ENGINE', ready: false, tools: { deno: 'indisponível' } }), { status: 200 }));
    await expect(detectLocalEngine()).resolves.toMatchObject({ state: 'TOOLS_NOT_READY', available: false, reachable: true, tools: { deno: 'indisponível' } });
  });

  it('bounds detection attempts with a timeout', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((_input, init) => new Promise((_resolve, reject) => {
      init?.signal?.addEventListener('abort', () => reject(new DOMException('timed out', 'AbortError')));
    }));
    await expect(detectLocalEngine({ timeoutMs: 5 })).resolves.toMatchObject({ state: 'NOT_DETECTED', reason: 'timeout' });
  });

  it('reports an explicit local network permission state after an interactive attempt', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('blocked by local network access'));
	Object.defineProperty(navigator, 'permissions', { configurable: true, value: { query: vi.fn().mockResolvedValue({ state: 'denied' }) } });
    await expect(detectLocalEngine({ interactive: true })).resolves.toMatchObject({ state: 'PERMISSION_REQUIRED', reason: 'permission' });
  });
});

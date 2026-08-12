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
});

describe('detectLocalEngine', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('reports local mode only after a valid health response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ status: 'ok', mode: 'LOCAL_ENGINE', version: '3.0.0', ready: true }), { status: 200 }));
    await expect(detectLocalEngine()).resolves.toMatchObject({ available: true, reachable: true, version: '3.0.0' });
  });

  it('keeps cloud available when the engine cannot be reached', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('network error'));
    await expect(detectLocalEngine()).resolves.toEqual({ available: false });
  });
});

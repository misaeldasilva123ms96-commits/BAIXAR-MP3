import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';

describe('MP3 Downloader interface', () => {
  beforeEach(() => { localStorage.clear(); vi.restoreAllMocks(); vi.unstubAllEnvs(); });
  afterEach(() => cleanup());

  it('shows the functional cloud entry point and an honest configuration error', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('engine offline'));
    render(<App />);
    expect(screen.getByRole('heading', { name: /Baixe e organize seus áudios/i })).toBeVisible();
    await userEvent.type(screen.getByLabelText('Cole o link do vídeo ou playlist'), 'https://youtu.be/abc');
    await userEvent.click(screen.getByRole('button', { name: /Analisar/i }));
    expect(await screen.findByRole('alert')).toHaveTextContent('backend online ainda não foi configurado');
    expect(screen.getByRole('button', { name: 'Conectar Engine' })).toBeEnabled();
    expect(screen.getByRole('link', { name: 'Baixar Engine para Windows' })).toHaveAttribute('href', expect.stringContaining('/releases/latest'));
  });

  it('renders only metadata returned by a healthy backend', async () => {
    vi.stubEnv('VITE_MP3_API_BASE_URL', 'https://api.example.test');
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url.startsWith('http://127.0.0.1')) throw new TypeError('engine offline');
      if (url.endsWith('/health')) return new Response(JSON.stringify({ status: 'ok', mode: 'WEB_CLOUD', version: '3.0.0', ready: true }), { status: 200 });
      if (url.endsWith('/analyze')) return new Response(JSON.stringify({ type: 'video', title: 'Título confirmado', artist: 'Canal real', duration: 83 }), { status: 200 });
      return new Response('{}', { status: 404 });
    });
    render(<App />);
    await userEvent.type(screen.getByLabelText('Cole o link do vídeo ou playlist'), 'https://youtu.be/abc');
    await userEvent.click(screen.getByRole('button', { name: /Analisar/i }));
    expect(await screen.findByRole('heading', { name: 'Título confirmado' })).toBeVisible();
    expect(screen.getByText(/Canal real/)).toBeVisible();
    expect(screen.queryByText(/artista inventado/i)).not.toBeInTheDocument();
  });

  it('exposes keyboard-addressable settings and tool verification', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('engine offline'));
    render(<App />);
    await userEvent.click(screen.getByRole('button', { name: 'Configurações' }));
    expect(screen.getByLabelText('Qualidade padrão')).toBeVisible();
    expect(screen.getByLabelText('Incorporar thumbnail')).toBeChecked();
    await userEvent.click(screen.getByRole('button', { name: 'Ferramentas' }));
    await waitFor(() => expect(screen.getByRole('button', { name: 'Verificar ferramentas' })).toBeVisible());
    expect(screen.getByRole('link', { name: 'Atualizar ferramentas' })).toHaveAttribute('href', expect.stringContaining('/releases/latest'));
  });

  it('routes a saved local job to the local provider after switching to cloud', async () => {
    localStorage.setItem('mp3-engine-token', 'secret');
    localStorage.setItem('mp3-download-history', JSON.stringify([{ id: 'job_local', mode: 'LOCAL_ENGINE', state: 'COMPLETED', request: { url: 'https://youtu.be/abc', quality: 'vbr0', organizePlaylist: true }, progress: { state: 'COMPLETED' }, result: { format: 'mp3' }, createdAt: '', updatedAt: '' }]));
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: vi.fn(() => 'blob:local') });
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() });
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined);
    const requests: string[] = [];
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const target = String(input); requests.push(target);
      if (target.endsWith('/health')) throw new TypeError('engine offline');
      if (target.endsWith('/downloads/job_local/file')) {
        expect(init?.headers).toEqual(expect.objectContaining({ 'X-MP3-Engine-Token': 'secret' }));
        return new Response(new Blob(['mp3']), { status: 200 });
      }
      return new Response('{}', { status: 404 });
    });
    render(<App />);
    await userEvent.click(screen.getByRole('button', { name: 'Histórico' }));
    await userEvent.click(screen.getByRole('button', { name: 'Baixar arquivo' }));
    await waitFor(() => expect(requests).toContain('http://127.0.0.1:38765/downloads/job_local/file'));
  });

  it('pairs a reachable engine with a valid code and switches Online to Local', async () => {
    localStorage.setItem('mp3-engine-token', 'secret');
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const target = String(input);
      if (target.endsWith('/health')) return new Response(JSON.stringify({ status: 'ok', mode: 'LOCAL_ENGINE', version: '3.0.1', ready: true, tools: { 'yt-dlp': '2026.07.04' } }), { status: 200 });
      if (target.endsWith('/settings')) {
        expect(init?.headers).toEqual(expect.objectContaining({ 'X-MP3-Engine-Token': 'secret' }));
        return new Response(JSON.stringify({ defaultQuality: 'vbr0', downloadDirectory: 'C:\\Music', organizePlaylist: true, avoidDuplicates: true, embedThumbnail: true, embedMetadata: true, openFolderWhenDone: false }), { status: 200 });
      }
      return new Response('{}', { status: 404 });
    });
    render(<App />);
    expect(await screen.findByText('Engine 3.0.1')).toBeVisible();
    await userEvent.click(screen.getByRole('button', { name: 'Conectar Engine' }));
    expect(await screen.findByText('Processamento Local')).toBeVisible();
    expect(screen.getByText('Processamento local disponível')).toBeVisible();
  });

  it('rejects an invalid engine code without enabling local processing', async () => {
    localStorage.setItem('mp3-engine-token', 'wrong');
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const target = String(input);
      if (target.endsWith('/health')) return new Response(JSON.stringify({ status: 'ok', mode: 'LOCAL_ENGINE', ready: true }), { status: 200 });
      if (target.endsWith('/settings')) return new Response(JSON.stringify({ error: { code: 'ENGINE_AUTH_REQUIRED', message: 'Código inválido.' } }), { status: 401 });
      return new Response('{}', { status: 404 });
    });
    render(<App />);
    await userEvent.click(await screen.findByRole('button', { name: 'Conectar Engine' }));
    expect(await screen.findByRole('heading', { name: 'Configurações' })).toBeVisible();
    expect(screen.getByRole('alert')).toHaveTextContent('Código de conexão inválido');
    expect(screen.queryByText('✓ Engine conectado')).not.toBeInTheDocument();
  });

  it('maps structured YouTube errors to actionable online recovery', async () => {
    vi.stubEnv('VITE_MP3_API_BASE_URL', 'https://api.example.test');
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const target = String(input);
      if (target.startsWith('http://')) throw new TypeError('engine offline');
      if (target.endsWith('/health')) return new Response(JSON.stringify({ status: 'ok', mode: 'WEB_CLOUD', ready: true }), { status: 200 });
      if (target.endsWith('/analyze')) return new Response(JSON.stringify({ error: { code: 'YOUTUBE_RATE_LIMITED', message: 'internal text' } }), { status: 503 });
      return new Response('{}', { status: 404 });
    });
    render(<App />);
    await userEvent.type(screen.getByLabelText('Cole o link do vídeo ou playlist'), 'https://youtu.be/abc');
    await userEvent.click(screen.getByRole('button', { name: /Analisar/i }));
    expect(await screen.findByRole('alert')).toHaveTextContent('modo online está temporariamente limitado');
    expect(screen.getByRole('button', { name: 'Tentar novamente' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Usar processamento local' })).toBeEnabled();
  });
});

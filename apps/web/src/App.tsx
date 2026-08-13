import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { ArrowRight, Check, Cloud, Download, ExternalLink, History, Home, Info, MonitorDown, Music2, Settings as SettingsIcon, Wrench } from 'lucide-react';
import type { Analysis, DownloadJob, DownloadProvider, DownloadRequest, ProcessingMode, Quality, Settings, ToolStatus } from './contracts';
import { cloudProvider, detectLocalEngine, localProvider, ProviderError } from './providers';

const qualityOptions: Array<{ value: Quality; label: string }> = [
  { value: 'vbr0', label: 'Melhor qualidade — VBR 0' }, { value: '320', label: '320 kbps' },
  { value: '256', label: '256 kbps' }, { value: '192', label: '192 kbps' }, { value: '128', label: '128 kbps' }
];
const terminalStates = new Set(['COMPLETED', 'FAILED', 'CANCELLED', 'SKIPPED']);
const navigation = [{ label: 'Início', Icon: Home }, { label: 'Downloads', Icon: Download }, { label: 'Histórico', Icon: History }, { label: 'Configurações', Icon: SettingsIcon }, { label: 'Ferramentas', Icon: Wrench }, { label: 'Sobre', Icon: Info }];

function isDesktop() { return (location.hostname === '127.0.0.1' || location.hostname === 'localhost') && location.port === '38765'; }
function formatDuration(seconds?: number) { if (!seconds) return ''; const hours = Math.floor(seconds / 3600); const minutes = Math.floor((seconds % 3600) / 60); const rest = Math.floor(seconds % 60); return [hours || null, minutes, rest].filter((v) => v !== null).map((v) => String(v).padStart(2, '0')).join(':'); }
function formatBytes(bytes?: number) { if (!bytes) return ''; const units = ['B', 'KB', 'MB', 'GB']; let value = bytes; let unit = 0; while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit += 1; } return `${value.toFixed(unit ? 1 : 0)} ${units[unit]}`; }
function modeLabel(mode: ProcessingMode) { return mode === 'WEB_CLOUD' ? 'Modo Online' : mode === 'LOCAL_ENGINE' ? 'Processamento Local' : 'Aplicativo Windows'; }
function initialEngineToken() {
  const saved = localStorage.getItem('mp3-engine-token');
  if (saved) return saved;
  const token = new URLSearchParams(location.hash.slice(1)).get('token') || '';
  if (token) history.replaceState(null, '', location.pathname + location.search);
  return token;
}

export default function App() {
  const desktop = isDesktop();
  const [page, setPage] = useState('Início');
  const [engineAvailable, setEngineAvailable] = useState(false);
  const [engineReachable, setEngineReachable] = useState(false);
  const [engineVersion, setEngineVersion] = useState<string>();
  const [tools, setTools] = useState<ToolStatus>({});
  const [engineToken, setEngineToken] = useState(initialEngineToken);
  const [mode, setMode] = useState<ProcessingMode>(desktop ? 'DESKTOP_LOCAL' : 'WEB_CLOUD');
  const [url, setUrl] = useState('');
  const [analysis, setAnalysis] = useState<Analysis>();
  const [quality, setQuality] = useState<Quality>(() => (localStorage.getItem('mp3-default-quality') as Quality) || 'vbr0');
  const [settings, setSettings] = useState<Settings>({ defaultQuality: quality, downloadDirectory: '', organizePlaylist: true, avoidDuplicates: true, embedThumbnail: true, embedMetadata: true, openFolderWhenDone: false });
  const [playlistStart, setPlaylistStart] = useState('');
  const [playlistEnd, setPlaylistEnd] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [jobs, setJobs] = useState<DownloadJob[]>(() => { try { return JSON.parse(localStorage.getItem('mp3-download-history') || '[]') as DownloadJob[]; } catch { return []; } });

  const provider = useMemo<DownloadProvider>(() => mode === 'WEB_CLOUD' ? cloudProvider() : localProvider(engineToken, desktop), [mode, engineToken, desktop]);
  const providerForMode = useCallback((jobMode: ProcessingMode): DownloadProvider => jobMode === 'WEB_CLOUD' ? cloudProvider() : localProvider(engineToken, jobMode === 'DESKTOP_LOCAL' && desktop), [engineToken, desktop]);

  useEffect(() => {
    detectLocalEngine().then((result) => { setEngineAvailable(result.available); setEngineReachable(Boolean(result.reachable)); setEngineVersion(result.version); setTools(result.tools || {}); });
  }, [desktop]);
  useEffect(() => { if (engineToken) localStorage.setItem('mp3-engine-token', engineToken); }, [engineToken]);
  useEffect(() => { localStorage.setItem('mp3-default-quality', quality); }, [quality]);
  useEffect(() => { if (mode !== 'WEB_CLOUD' && engineReachable) provider.getSettings().then((value) => { setSettings(value); setQuality(value.defaultQuality); }).catch(() => undefined); }, [mode, engineReachable, provider]);
  useEffect(() => { localStorage.setItem('mp3-download-history', JSON.stringify(jobs.slice(0, 50).map(({ result, ...job }) => ({ ...job, result: result ? { ...result } : undefined })))); }, [jobs]);
  useEffect(() => {
    const active = jobs.filter((job) => !terminalStates.has(job.state));
    if (!active.length) return;
    const timer = window.setInterval(async () => {
      for (const job of active) {
        try { const next = await providerForMode(job.mode).getProgress(job.id); setJobs((current) => current.map((item) => item.id === next.id ? next : item)); } catch { /* transient polling failure is shown by the existing state */ }
      }
    }, 1000);
    return () => window.clearInterval(timer);
  }, [jobs, providerForMode]);

  async function analyze(event: FormEvent) {
    event.preventDefault(); setError(''); setAnalysis(undefined); setBusy(true);
    if (mode === 'WEB_CLOUD' && !provider.baseUrl) { setError('O backend online ainda não foi configurado para esta publicação.'); setBusy(false); return; }
    try { const health = await provider.health(); if (!health.ready) throw new ProviderError('O runtime respondeu, mas uma ou mais ferramentas estão indisponíveis.', 'TOOLS_NOT_READY'); setAnalysis(await provider.analyze(url.trim())); }
    catch (reason) { setError(reason instanceof ProviderError ? reason.message : 'Não foi possível analisar o link agora.'); }
    finally { setBusy(false); }
  }

  async function startDownload() {
    setError(''); setBusy(true);
    const request: DownloadRequest = { url: url.trim(), quality, organizePlaylist: settings.organizePlaylist, embedThumbnail: settings.embedThumbnail, embedMetadata: settings.embedMetadata, playlistStart: playlistStart ? Number(playlistStart) : undefined, playlistEnd: playlistEnd ? Number(playlistEnd) : undefined };
    try { const job = await provider.download(request); setJobs((current) => [job, ...current.filter((item) => item.id !== job.id)]); setPage('Downloads'); }
    catch (reason) { setError(reason instanceof ProviderError ? reason.message : 'Não foi possível iniciar o download.'); }
    finally { setBusy(false); }
  }

  function chooseLocal() {
    if (!engineToken && !desktop) { setError('Abra o aplicativo Windows, copie o código de conexão e informe-o em Configurações.'); setPage('Configurações'); return; }
    setMode(desktop ? 'DESKTOP_LOCAL' : 'LOCAL_ENGINE'); setError('');
  }

  const activeJobs = jobs.filter((job) => !terminalStates.has(job.state));
  return <div className="app-shell">
    <header className="topbar">
      <button className="brand" onClick={() => setPage('Início')} aria-label="Ir para o início"><span className="brand-mark"><Music2 size={20} aria-hidden="true" /></span><span>MP3 <strong>Downloader</strong></span></button>
      <div className={`mode-pill ${mode === 'WEB_CLOUD' ? 'cloud' : 'local'}`}>{mode === 'WEB_CLOUD' ? <Cloud size={15} aria-hidden="true" /> : <MonitorDown size={15} aria-hidden="true" />}{modeLabel(mode)}</div>
    </header>
    <aside className="sidebar" aria-label="Navegação principal">
      {navigation.map(({ label, Icon }) => <button key={label} className={page === label ? 'active' : ''} onClick={() => setPage(label)}><span aria-hidden="true"><Icon size={19} /></span>{label}{label === 'Downloads' && activeJobs.length ? <b>{activeJobs.length}</b> : null}</button>)}
    </aside>
    <main id="main-content">
      {page === 'Início' && <section className="home">
        <div className="hero-copy"><p className="eyebrow">ÁUDIO, DO SEU JEITO</p><h1>Baixe e organize seus áudios <em>sem complicação.</em></h1><p>Uma experiência única para usar online ou processar diretamente no seu computador.</p></div>
        <form className="download-card" onSubmit={analyze}>
          <label htmlFor="media-url">Cole o link do vídeo ou playlist</label>
          <div className="url-row"><ExternalLink size={18} aria-hidden="true" /><input id="media-url" type="url" required value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://www.youtube.com/watch?v=..." autoComplete="url" /><button disabled={busy}>{busy ? 'Analisando…' : 'Analisar'} <ArrowRight size={17} aria-hidden="true" /></button></div>
          <div className="provider-row">
            <button type="button" className={mode === 'WEB_CLOUD' ? 'provider active' : 'provider'} onClick={() => { setMode('WEB_CLOUD'); setError(''); }}><span><Cloud size={22} aria-hidden="true" /></span><span><strong>Usar online</strong><small>Sem instalar nada</small></span><i>{mode === 'WEB_CLOUD' ? 'Selecionado' : 'Selecionar'}</i></button>
            <button type="button" className={mode !== 'WEB_CLOUD' ? 'provider active' : 'provider'} disabled={!engineReachable && !desktop} onClick={chooseLocal}><span><MonitorDown size={22} aria-hidden="true" /></span><span><strong>{engineAvailable ? 'Processamento local disponível' : engineReachable || desktop ? 'Engine aberto — preparação necessária' : 'Engine local não detectado'}</strong><small>{engineVersion ? `Engine ${engineVersion}` : 'Mais privado e direto'}</small></span><i>{mode !== 'WEB_CLOUD' ? 'Selecionado' : engineAvailable ? 'Selecionar' : engineReachable || desktop ? 'Verificar' : 'Indisponível'}</i></button>
          </div>
          {error && <div className="notice error" role="alert">{error}</div>}
        </form>
        {analysis && <section className="analysis-card" aria-live="polite">
          {analysis.thumbnail && <img src={analysis.thumbnail} alt="Miniatura do conteúdo analisado" />}
          <div className="analysis-copy"><span className="tag">{analysis.type === 'playlist' ? 'Playlist' : 'Vídeo'}</span><h2>{analysis.title || analysis.playlistTitle || 'Título não disponível'}</h2><p>{[analysis.artist, formatDuration(analysis.duration), analysis.itemCount ? `${analysis.itemCount} itens` : ''].filter(Boolean).join(' • ') || 'Informações adicionais não disponíveis'}</p></div>
          <div className="download-options"><label>Qualidade<select value={quality} onChange={(e) => setQuality(e.target.value as Quality)}>{qualityOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>{analysis.type === 'playlist' && <div className="range"><label>De<input type="number" min="1" max="500" value={playlistStart} onChange={(e) => setPlaylistStart(e.target.value)} /></label><label>Até<input type="number" min="1" max="500" value={playlistEnd} onChange={(e) => setPlaylistEnd(e.target.value)} /></label></div>}<small>Converter para 320 kbps não aumenta a qualidade de uma fonte de menor qualidade.</small><div className="summary"><span>Modo <strong>{modeLabel(mode)}</strong></span><span>Destino <strong>{mode === 'WEB_CLOUD' ? 'Download pelo navegador' : 'Pasta configurada'}</strong></span></div><button className="primary" disabled={busy} onClick={startDownload}>Baixar MP3</button></div>
        </section>}
        <div className="trust-strip"><span><Check size={15} /> Conversão para MP3</span><span><Check size={15} /> Playlists e intervalos</span><span><Check size={15} /> Metadados e capas</span><span><Check size={15} /> Progresso real</span></div>
        <p className="privacy">{mode === 'WEB_CLOUD' ? 'O modo online processa temporariamente o conteúdo em nosso servidor. Arquivos temporários são removidos automaticamente.' : 'No modo local, o áudio é processado pelo Engine no seu computador e não passa pelo backend web.'}</p>
      </section>}
      {page === 'Downloads' && <Downloads jobs={jobs} providerForMode={providerForMode} onError={setError} onJobs={setJobs} onNew={() => { setAnalysis(undefined); setUrl(''); setPage('Início'); }} />}
      {page === 'Histórico' && <SimplePage title="Histórico" subtitle="Mantido somente neste navegador.">{jobs.length ? jobs.map((job) => <JobRow key={job.id} job={job} provider={providerForMode(job.mode)} onError={setError} />) : <Empty text="Nenhum download registrado neste navegador." />}</SimplePage>}
      {page === 'Configurações' && <SimplePage title="Configurações" subtitle="Preferências deste dispositivo."><div className="settings-card"><label>Qualidade padrão<select value={settings.defaultQuality} onChange={(e) => setSettings((value) => ({ ...value, defaultQuality: e.target.value as Quality }))}>{qualityOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>{mode !== 'WEB_CLOUD' && <label>Pasta de downloads<input value={settings.downloadDirectory} onChange={(e) => setSettings((value) => ({ ...value, downloadDirectory: e.target.value }))} placeholder="C:\Users\...\Downloads\Musicas_MP3" /></label>}<label className="check"><input type="checkbox" checked={settings.organizePlaylist} onChange={(e) => setSettings((value) => ({ ...value, organizePlaylist: e.target.checked }))} /> Criar pasta para playlists</label><label className="check"><input type="checkbox" checked={settings.avoidDuplicates} onChange={(e) => setSettings((value) => ({ ...value, avoidDuplicates: e.target.checked }))} /> Evitar downloads repetidos no Windows</label><label className="check"><input type="checkbox" checked={settings.embedThumbnail} onChange={(e) => setSettings((value) => ({ ...value, embedThumbnail: e.target.checked }))} /> Incorporar thumbnail</label><label className="check"><input type="checkbox" checked={settings.embedMetadata} onChange={(e) => setSettings((value) => ({ ...value, embedMetadata: e.target.checked }))} /> Incorporar metadados</label><label>Código de conexão do Engine<input type="password" value={engineToken} onChange={(e) => setEngineToken(e.target.value.trim())} placeholder="Cole o código exibido pelo Engine" /></label><button className="primary" onClick={async () => { localStorage.setItem('mp3-engine-token', engineToken); localStorage.setItem('mp3-default-quality', settings.defaultQuality); setQuality(settings.defaultQuality); if (mode !== 'WEB_CLOUD') { try { await provider.saveSettings(settings); } catch (reason) { setError(reason instanceof Error ? reason.message : 'Não foi possível salvar.'); return; } } if (engineAvailable || desktop) chooseLocal(); }}>Salvar configurações</button><p>O código impede que outros sites controlem o Engine apenas por ele estar em localhost.</p></div>{error && <div className="notice error" role="alert">{error}</div>}</SimplePage>}
      {page === 'Ferramentas' && <SimplePage title="Ferramentas" subtitle="Estado informado pelo runtime, sem estimativas."><div className="settings-card"><p><strong>Engine local:</strong> {engineAvailable ? `Pronto${engineVersion ? ` — ${engineVersion}` : ''}` : engineReachable || desktop ? 'Aberto, mas incompleto' : 'Não detectado'}</p>{Object.entries(tools).map(([name, value]) => <p key={name}><strong>{name}:</strong> {value}</p>)}<button className="primary" onClick={() => detectLocalEngine().then((result) => { setEngineAvailable(result.available); setEngineReachable(Boolean(result.reachable)); setEngineVersion(result.version); setTools(result.tools || {}); })}>Verificar ferramentas</button> <a className="primary tool-link" href="https://github.com/misaeldasilva123ms96-commits/BAIXAR-MP3/releases/latest">Atualizar ferramentas</a><p>O Engine não afirma que uma ferramenta está atualizada sem consultar o runtime. Atualizações são distribuídas em pacote versionado e verificado.</p></div></SimplePage>}
      {page === 'Sobre' && <SimplePage title="Sobre" subtitle="Um produto, uma interface, múltiplos modos."><div className="settings-card"><p>MP3 Downloader v3.0.0</p><p>Use apenas conteúdo próprio, licenciado, em domínio público ou para o qual você tenha autorização. Respeite direitos autorais e os termos da plataforma.</p></div></SimplePage>}
    </main>
    <footer><span>MP3 Downloader v3</span><span>Privacidade • Segurança • Código aberto</span></footer>
  </div>;
}

function SimplePage({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) { return <section className="simple-page"><p className="eyebrow">MP3 DOWNLOADER</p><h1>{title}</h1><p>{subtitle}</p><div className="stack">{children}</div></section>; }
function Empty({ text }: { text: string }) { return <div className="empty"><Music2 size={32} aria-hidden="true" /><p>{text}</p></div>; }
function JobRow({ job, provider, onError }: { job: DownloadJob; provider: DownloadProvider; onError: (message: string) => void }) { const percent = job.progress.percent; return <article className="job-row"><div><span className={`state state-${job.state.toLowerCase()}`}>{job.state}</span><h3>{job.progress.title || job.result?.title || job.id}</h3><p>{job.error || [job.progress.speed, job.progress.eta && `ETA ${job.progress.eta}`, job.progress.size || formatBytes(job.result?.size)].filter(Boolean).join(' • ') || modeLabel(job.mode)}</p></div><div className="job-progress"><progress value={typeof percent === 'number' ? percent : undefined} max="100" aria-label={`Progresso de ${job.progress.title || job.id}`} /><span>{typeof percent === 'number' ? `${percent.toFixed(1)}%` : job.state}</span></div>{job.state === 'COMPLETED' && <button className="primary small" onClick={() => provider.downloadFile(job.id).catch((reason) => onError(reason instanceof Error ? reason.message : 'Não foi possível baixar o arquivo.'))}>Baixar arquivo</button>}</article>; }
function Downloads({ jobs, providerForMode, onError, onJobs, onNew }: { jobs: DownloadJob[]; providerForMode: (mode: ProcessingMode) => DownloadProvider; onError: (message: string) => void; onJobs: React.Dispatch<React.SetStateAction<DownloadJob[]>>; onNew: () => void }) { return <SimplePage title="Downloads" subtitle="Progresso informado pelo yt-dlp e pelo conversor."><div className="page-actions"><button className="primary" onClick={onNew}>Novo download</button></div>{jobs.length ? jobs.map((job) => { const jobProvider = providerForMode(job.mode); return <div key={job.id} className="job-wrap"><JobRow job={job} provider={jobProvider} onError={onError} />{!terminalStates.has(job.state) && <button className="text-button" onClick={async () => { try { await jobProvider.cancel(job.id); onJobs((current) => current.map((item) => item.id === job.id ? { ...item, state: 'CANCELLED', progress: { ...item.progress, state: 'CANCELLED' } } : item)); } catch (reason) { onError(reason instanceof Error ? reason.message : 'Não foi possível cancelar o download.'); } }}>Cancelar</button>}</div>; }) : <Empty text="Seus downloads aparecerão aqui." />}</SimplePage>; }

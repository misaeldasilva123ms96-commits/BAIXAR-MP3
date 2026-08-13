# MP3 Downloader

[![CI](https://github.com/misaeldasilva123ms96-commits/BAIXAR-MP3/actions/workflows/ci.yml/badge.svg)](https://github.com/misaeldasilva123ms96-commits/BAIXAR-MP3/actions/workflows/ci.yml)
[![Pages](https://github.com/misaeldasilva123ms96-commits/BAIXAR-MP3/actions/workflows/pages.yml/badge.svg)](https://misaeldasilva123ms96-commits.github.io/BAIXAR-MP3/)
[![Release](https://img.shields.io/github/v/release/misaeldasilva123ms96-commits/BAIXAR-MP3)](https://github.com/misaeldasilva123ms96-commits/BAIXAR-MP3/releases/latest)

**[Usar Online](https://misaeldasilva123ms96-commits.github.io/BAIXAR-MP3/)** · **[Baixar para Windows](https://github.com/misaeldasilva123ms96-commits/BAIXAR-MP3/releases/latest)**

Baixe vídeos ou playlists autorizadas como MP3 usando a mesma interface no navegador e no Windows. O V3 escolhe explicitamente entre processamento temporário no backend e processamento local; o terminal V2 permanece disponível durante a transição.

> Use apenas conteúdo próprio, licenciado, em domínio público ou para o qual você tenha autorização. Respeite direitos autorais e os termos da plataforma.

## Escolha como usar

| Modo | Melhor para | Onde o áudio é processado |
| --- | --- | --- |
| **Online** | Uso rápido sem instalar | Backend temporário; o resultado é baixado pelo navegador |
| **Windows** | Uso frequente e playlists maiores | Engine local; o MP3 é salvo diretamente no computador |
| **Local via Web** | Quem já possui o Engine | Engine em `127.0.0.1`, após autorização por código local |

O site não exige o Engine. Quando ele não está disponível, o modo online continua selecionável. A aplicação nunca troca silenciosamente do modo local para o cloud.

## Recursos

- vídeo individual, playlist completa e intervalo de itens;
- VBR 0 recomendado, 320, 256, 192 e 128 kbps;
- análise real de título, miniatura, artista, duração e itens quando o provedor os informa;
- progresso derivado da saída real do yt-dlp, sem barra artificial;
- miniatura e metadados incorporados;
- histórico persistente local no Windows e histórico somente no navegador no modo web;
- cancelamento, fila controlada e expiração automática no cloud;
- mesma interface responsiva e acessível nos três modos.

## Windows: primeira execução

1. Baixe `MP3_Downloader_v3.0.0_Windows.zip` em **Releases**.
2. Compare o SHA-256 com `checksums.sha256` e extraia todo o ZIP.
3. Abra `ABRIR_MP3_DOWNLOADER.bat`.
4. O Engine abre a interface em `http://127.0.0.1:38765` e mostra o código de conexão local.
5. Cole uma URL, escolha a qualidade e confirme o resumo.

O pacote inclui `MP3_Downloader.exe`, yt-dlp, FFmpeg, ffprobe, Deno e a interface. Python não é necessário. O V2 continua acessível por `Abrir_Baixador_MP3_V2.bat`.

## Privacidade

- **Online:** a URL e o conteúdo necessário à conversão passam pelo backend. Arquivos são temporários, expiram pelo TTL configurado e não formam uma biblioteca permanente.
- **Windows/local:** o processamento ocorre no computador. O Engine escuta somente em `127.0.0.1` e exige um token para analisar, iniciar ou cancelar downloads.
- O backend não associa histórico permanente a usuários. O navegador guarda apenas a lista local exibida na interface.
- Logs usam IDs de job e não devem registrar cookies, tokens, cabeçalhos privados ou URLs completas.

## Segurança e limites do serviço público

O backend aceita somente URLs HTTP(S) de hosts oficiais do YouTube, rejeita credenciais embutidas, protocolos locais, localhost e redes privadas/reservadas. A API usa parâmetros tipados; não existe endpoint de execução nem passagem livre de flags do yt-dlp.

Limites configuráveis por ambiente incluem concorrência, requisições por minuto, quantidade de itens, timeout e TTL. Veja [SECURITY.md](SECURITY.md) e [docs/DEPLOY.md](docs/DEPLOY.md).

## Configuração

| Variável | Padrão | Uso |
| --- | --- | --- |
| `VITE_MP3_API_BASE_URL` | vazio | URL HTTPS do backend inserida no build do Pages |
| `MP3_ALLOWED_ORIGINS` | origem oficial do Pages | CORS permitido no cloud |
| `MP3_MAX_CONCURRENT` | `2` | jobs cloud simultâneos |
| `MP3_MAX_PLAYLIST_ITEMS` | `100` | limite de análise/playlist cloud |
| `MP3_FILE_TTL` | `30m` | retenção do resultado temporário |
| `MP3_JOB_TIMEOUT` | `30m` | duração máxima de um job cloud |
| `MP3_RATE_LIMIT` | `30` | requisições por IP e minuto |
| `MP3_GLOBAL_RATE_LIMIT` | `300` | requisições globais por minuto e instância |
| `MP3_MAX_OUTPUT_MB` | `500` | teto do resultado temporário por job |
| `MP3_DOWNLOAD_DIR` | `%USERPROFILE%\Downloads\Musicas_MP3` | pasta do modo Windows |

## Arquitetura

```text
apps/web (React + TypeScript)
        │ DownloadProvider
        ├────────────── Cloud API (Go + Docker)
        │                    └─ yt-dlp + FFmpeg + Deno → arquivo temporário
        └────────────── Engine local (Go, 127.0.0.1 + token)
                             └─ yt-dlp + FFmpeg + Deno → pasta do usuário
```

Contratos tipados e estados são compartilhados entre providers; o frontend não conhece flags do yt-dlp. Detalhes e decisões estão em [docs/ARQUITETURA.md](docs/ARQUITETURA.md) e a auditoria do legado em [docs/V3_AUDIT.md](docs/V3_AUDIT.md).

## Desenvolvimento

Requisitos: Node 24.15+, Go 1.26.5+ e PowerShell 7.

```powershell
npm ci --ignore-scripts
npm test
npm run lint
npm run typecheck
npm run build
go test ./services/...
go vet ./services/...
pwsh -NoProfile -File .\tests\legacy-contract.ps1
```

Backend local de desenvolvimento:

```powershell
$env:MP3_ALLOWED_ORIGINS='http://localhost:5173'
go run ./services/cmd/web-api
npm run dev --workspace apps/web
```

## Publicação

- Pull requests executam lint, testes, build, segurança, smoke test Windows e Docker, mas não publicam.
- `main` publica o Pages somente após todos os passos do workflow Pages e exige a variável `MP3_API_BASE_URL`.
- tags `vX.Y.Z` geram o ZIP Windows, hashes SHA-256 e atestação GitHub; nenhuma release existente é substituída.

Consulte [Deploy](docs/DEPLOY.md), [Release](docs/RELEASE.md), [Guia do usuário](docs/GUIA_DO_USUARIO.md) e [CHANGELOG.md](CHANGELOG.md).

## Limitações

- O GitHub Pages hospeda apenas o frontend; uma instância separada do backend é obrigatória para processamento online.
- Conteúdo privado, autenticado, protegido por DRM ou indisponível não é suportado.
- Seleção individual de faixas fica para uma fase posterior; o V3 inicial suporta playlist completa e intervalo.
- Disponibilidade e compatibilidade dependem das plataformas de origem, do yt-dlp e dos limites do provedor de backend.

# Arquitetura V3

## Componentes

- `apps/web`: única UI React/TypeScript usada no Pages e servida pelo executável Windows.
- `services/internal/core`: tipos, validação de URL, argumentos permitidos, análise, execução e progresso.
- `services/internal/api`: contrato HTTP, jobs, rate limit, CORS, cancelamento, SSE, download e expiração.
- `services/cmd/web-api`: composição cloud e configuração por ambiente.
- `services/cmd/local-engine`: composição local, servidor estático, token e bind fixo em loopback.
- `scripts`: preparação verificada das ferramentas e empacotamento por allowlist.

## Contrato do provider

`DownloadProvider` expõe `health`, `analyze`, `download`, `getProgress`, `cancel`, `eventsUrl`, `downloadFile`, `getSettings` e `saveSettings`. `HttpDownloadProvider` recebe modo, URL base e opcionalmente token. A UI não recebe nem produz argumentos de linha de comando.

Os modos são `WEB_CLOUD`, `LOCAL_ENGINE` e `DESKTOP_LOCAL`. A escolha fica visível. A detecção tenta `127.0.0.1` e depois `localhost`, com timeout e no máximo duas tentativas explícitas. Os estados são `NOT_DETECTED`, `PERMISSION_REQUIRED`, `REACHABLE`, `TOOLS_NOT_READY`, `READY` e `AUTH_REQUIRED`; seleção local só ocorre após `/health` pronto e validação do token em `/settings`.

## API

| Método | Rota | Comportamento |
| --- | --- | --- |
| GET | `/health` | status, modo, versão, prontidão e versões reais das ferramentas |
| GET | `/version` | versão do produto e do contrato |
| POST | `/analyze` | metadados reais via yt-dlp |
| POST | `/downloads` | cria job tipado |
| GET | `/downloads/:id` | snapshot verdadeiro do job |
| DELETE | `/downloads/:id` | cancela o contexto/processo |
| GET | `/downloads/:id/events` | SSE até estado terminal |
| GET | `/downloads/:id/file` | entrega resultado cloud dentro do TTL |
| GET | `/settings` | lê configurações locais |
| PUT | `/settings` | salva configurações locais |

Playlists cloud com mais de um resultado são empacotadas em ZIP. Um vídeo resulta em MP3. O servidor nunca mantém uma biblioteca permanente.

## Estados

`ANALYZING`, `QUEUED`, `DOWNLOADING`, `CONVERTING`, `ADDING_METADATA`, `FINALIZING`, `COMPLETED`, `FAILED`, `CANCELLED` e `SKIPPED` formam o vocabulário compartilhado. A implementação só publica percentuais extraídos do yt-dlp e nunca publica 100% antes de `COMPLETED`.

Falhas do upstream usam `YOUTUBE_BOT_CHALLENGE`, `YOUTUBE_PO_TOKEN_REQUIRED`, `YOUTUBE_RATE_LIMITED`, `YOUTUBE_UNAVAILABLE`, `UPSTREAM_TIMEOUT` e `EXTRACTOR_FAILED`. Somente 429 e timeout recebem uma repetição controlada; desafio antibot, PO Token, conteúdo indisponível e falha permanente não são repetidos. Mensagens e jobs não expõem stderr, stack trace, URL completa ou token.

## Limites de confiança

- Cloud: Internet → CORS permitido → validação de URL → fila/concurrency → processo com argumentos construídos internamente → diretório temporário por job.
- Local: site oficial ou UI local → CORS → token secreto → API tipada → processo local → pasta configurada pelo Engine.
- A validação aceita somente hosts oficiais conhecidos. Isso reduz a superfície SSRF além de uma simples rejeição de IP privado.
- Não existem cookies, upload, importação de sessão, flags arbitrárias, shell ou endpoint `/execute`.

## Decisões pendentes

- seleção individual de faixas;
- shell desktop nativo em vez da UI aberta no navegador;
- assinatura Authenticode do executável.

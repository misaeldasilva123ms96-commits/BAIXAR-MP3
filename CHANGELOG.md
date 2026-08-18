# Changelog

## 3.0.1 — não lançado

### Corrigido

- modo online usa a estratégia padrão atual do yt-dlp e classifica erros do YouTube sem expor logs internos;
- retries são limitados a falhas transitórias, com backoff, jitter, timeout e cancelamento;
- `/health` só declara prontidão após validar yt-dlp, FFmpeg, ffprobe, Deno e EJS;
- Engine local pode ser reconectado por ação do usuário em `127.0.0.1` ou `localhost`, com estados reais de permissão, ferramentas e autenticação;
- CORS local reconhece preflight de Private Network Access somente para origens permitidas;
- pacote Windows prepara ferramentas automaticamente e verifica os hashes do manifest.

### Ferramentas

- yt-dlp `2026.07.04`;
- Deno `2.9.5`;
- FFmpeg e ffprobe `9.0.1-essentials`.

## 3.0.0 — 2026-08-13

### Adicionado

- uma interface React responsiva para web e Windows;
- providers cloud, Engine local e desktop;
- análise e progresso reais via yt-dlp;
- API Go com jobs, SSE, cancelamento, rate limit, TTL e cleanup;
- Engine Go em loopback com token de conexão;
- Docker, Render blueprint, GitHub Pages e release Windows reproduzível;
- testes de core, segurança, API, frontend e contrato legado;
- documentação de auditoria, arquitetura, deploy, release e uso.

### Compatibilidade

- o PowerShell V2 e seu BAT foram preservados;
- qualidades, playlist, intervalo, miniaturas, metadados e histórico continuam disponíveis.

### Limitações conhecidas

- seleção individual de faixas e shell desktop nativo ficam para versões posteriores;
- Pages depende de um backend HTTPS configurado;
- o binário Windows ainda não possui assinatura Authenticode.

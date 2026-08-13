# Deploy do backend e do Pages

## Backend Docker

```bash
docker build -t baixar-mp3-api .
docker run --rm -p 8080:8080 \
  -e MP3_ALLOWED_ORIGINS=https://misaeldasilva123ms96-commits.github.io \
  baixar-mp3-api
```

Verifique `GET http://127.0.0.1:8080/health`. O campo `ready` precisa ser `true`; isso confirma yt-dlp, FFmpeg, ffprobe e Deno no runtime.

O `render.yaml` fornece uma implantação inicial no Render. Railway, Fly.io, VPS e outros hosts Docker podem usar a mesma imagem. Escolha um plano com disco temporário suficiente e processos longos; planos gratuitos podem suspender ou interromper conversões.

## Variáveis

- `MP3_API_ADDR=:8080`
- `MP3_ALLOWED_ORIGINS=https://misaeldasilva123ms96-commits.github.io`
- `MP3_DATA_DIR=/var/lib/mp3`
- `MP3_MAX_CONCURRENT=2`
- `MP3_MAX_PLAYLIST_ITEMS=100`
- `MP3_RATE_LIMIT=30`
- `MP3_GLOBAL_RATE_LIMIT=300`
- `MP3_MAX_OUTPUT_MB=500`
- `MP3_FILE_TTL=30m`
- `MP3_JOB_TIMEOUT=30m`

Não grave secrets na imagem. Monte armazenamento temporário apenas se o provedor exigir; resultados expirados são removidos.

## Pages

1. Faça o deploy HTTPS do backend.
2. No repositório, crie a variável Actions `MP3_API_BASE_URL` com a origem da API, sem barra final. O workflow Pages mapeia essa variável para `VITE_MP3_API_BASE_URL` durante o build do Vite.
3. Configure GitHub Pages com **GitHub Actions** como source.
4. Após merge em `main`, o workflow testa, gera assets relativos compatíveis com `/BAIXAR-MP3/` e publica.

O workflow recusa deploy de `main` quando a variável está vazia. Assim, uma landing page sem backend não é publicada como se fosse o aplicativo online funcional.

## Operação

- monitore `/health`, uso de CPU, memória, disco temporário, duração e quantidade de jobs;
- aplique limite adicional no proxy/provedor;
- não registre URLs completas, tokens ou headers;
- atualize as ferramentas por PR, alterando versão e hash juntos;
- valide download, cancelamento e cleanup antes de promover uma nova imagem.

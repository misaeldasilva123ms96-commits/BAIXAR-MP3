FROM golang:1.26.5-bookworm AS api
WORKDIR /src
COPY go.mod ./
COPY services services
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /mp3-web-api ./services/cmd/web-api

FROM debian:12.10-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl ffmpeg unzip && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL -o /usr/local/bin/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/download/2026.06.09/yt-dlp_linux \
 && echo "bf8aac79b72287a6d2043074415132558b43743a8f9461a22b0141e90f16ce66  /usr/local/bin/yt-dlp" | sha256sum -c - \
 && chmod 0755 /usr/local/bin/yt-dlp
RUN curl -fsSL -o /tmp/deno.zip https://github.com/denoland/deno/releases/download/v2.8.1/deno-x86_64-unknown-linux-gnu.zip \
 && echo "2d7bb6195226ac832e0bf7109a115f0af65ee69ac797a4bbde5b27a06cc242d9  /tmp/deno.zip" | sha256sum -c - \
 && unzip -q /tmp/deno.zip -d /usr/local/bin && rm /tmp/deno.zip && chmod 0755 /usr/local/bin/deno
COPY --from=api /mp3-web-api /usr/local/bin/mp3-web-api
RUN useradd --system --uid 10001 --create-home mp3 && mkdir -p /var/lib/mp3 && chown mp3:mp3 /var/lib/mp3
USER 10001
ENV MP3_API_ADDR=:8080 MP3_DATA_DIR=/var/lib/mp3 MP3_FILE_TTL=30m MP3_JOB_TIMEOUT=30m MP3_MAX_CONCURRENT=2 MP3_MAX_JOBS=1000 MP3_RATE_LIMIT=30
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["curl","-fsS","http://127.0.0.1:8080/health"]
ENTRYPOINT ["/usr/local/bin/mp3-web-api"]

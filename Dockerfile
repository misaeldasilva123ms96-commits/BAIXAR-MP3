FROM golang:1.26.5-bookworm AS api
WORKDIR /src
COPY go.mod ./
COPY services services
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /mp3-web-api ./services/cmd/web-api

FROM debian:12.10-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl ffmpeg unzip && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL -o /usr/local/bin/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/download/2026.07.04/yt-dlp_linux \
 && echo "6bbb3d314cde4febe36e5fa1d55462e29c974f63444e707871834f6d8cc210ae  /usr/local/bin/yt-dlp" | sha256sum -c - \
 && chmod 0755 /usr/local/bin/yt-dlp
RUN curl -fsSL -o /tmp/deno.zip https://github.com/denoland/deno/releases/download/v2.9.5/deno-x86_64-unknown-linux-gnu.zip \
 && echo "8b010a3b1a4a0188a67cdb8a7a27348b2a501af78aec7fc74f2ace167368d530  /tmp/deno.zip" | sha256sum -c - \
 && unzip -q /tmp/deno.zip -d /usr/local/bin && rm /tmp/deno.zip && chmod 0755 /usr/local/bin/deno
COPY --from=api /mp3-web-api /usr/local/bin/mp3-web-api
RUN useradd --system --uid 10001 --create-home mp3 && mkdir -p /var/lib/mp3 && chown mp3:mp3 /var/lib/mp3
USER 10001
ENV MP3_API_ADDR=:8080 MP3_DATA_DIR=/var/lib/mp3 MP3_FILE_TTL=30m MP3_JOB_TIMEOUT=30m MP3_MAX_CONCURRENT=2 MP3_MAX_JOBS=1000 MP3_RATE_LIMIT=30
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["curl","-fsS","http://127.0.0.1:8080/health"]
ENTRYPOINT ["/usr/local/bin/mp3-web-api"]

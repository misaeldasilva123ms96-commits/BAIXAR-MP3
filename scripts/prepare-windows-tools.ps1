[CmdletBinding()]
param([string]$OutputDirectory = (Join-Path (Split-Path -Parent $PSScriptRoot) 'ferramentas'))

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$output = [IO.Path]::GetFullPath($OutputDirectory)
New-Item -ItemType Directory -Force -Path $output | Out-Null
$staging = Join-Path ([IO.Path]::GetTempPath()) ("mp3-tools-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $staging | Out-Null

function Receive-VerifiedFile {
    param([string]$Url, [string]$Destination, [string]$Sha256)
    $partial = "$Destination.download"
    Invoke-WebRequest -Uri $Url -OutFile $partial -UseBasicParsing
    $actual = (Get-FileHash -LiteralPath $partial -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $Sha256) { throw "Hash inválido para $Url. Esperado $Sha256, recebido $actual." }
    Move-Item -LiteralPath $partial -Destination $Destination -Force
}

try {
    Receive-VerifiedFile `
        -Url 'https://github.com/yt-dlp/yt-dlp/releases/download/2026.07.04/yt-dlp.exe' `
        -Destination (Join-Path $output 'yt-dlp.exe') `
        -Sha256 '52fe3c26dcf71fbdc85b528589020bb0b8e383155cfa81b64dd447bbe35e24b8'

    $denoZip = Join-Path $staging 'deno.zip'
    Receive-VerifiedFile `
        -Url 'https://github.com/denoland/deno/releases/download/v2.9.5/deno-x86_64-pc-windows-msvc.zip' `
        -Destination $denoZip `
        -Sha256 '171efab55ac6b9881fd53ee4c20f8bf3bb1340ffc618483746909014db12216a'
    Expand-Archive -LiteralPath $denoZip -DestinationPath (Join-Path $staging 'deno') -Force
    Copy-Item -LiteralPath (Join-Path $staging 'deno/deno.exe') -Destination (Join-Path $output 'deno.exe') -Force

    $ffmpegZip = Join-Path $staging 'ffmpeg.zip'
    Receive-VerifiedFile `
        -Url 'https://github.com/GyanD/codexffmpeg/releases/download/9.0.1/ffmpeg-9.0.1-essentials_build.zip' `
        -Destination $ffmpegZip `
        -Sha256 'fec81ae03971d9dd4be3ebe02e263bd2ec1d789483f931bdba5f5715e65da2e9'
    Expand-Archive -LiteralPath $ffmpegZip -DestinationPath (Join-Path $staging 'ffmpeg') -Force
    $ffmpeg = Get-ChildItem -LiteralPath (Join-Path $staging 'ffmpeg') -Recurse -File -Filter ffmpeg.exe | Select-Object -First 1
    $ffprobe = Get-ChildItem -LiteralPath (Join-Path $staging 'ffmpeg') -Recurse -File -Filter ffprobe.exe | Select-Object -First 1
    if (-not $ffmpeg -or -not $ffprobe) { throw 'ffmpeg.exe ou ffprobe.exe ausente no pacote verificado.' }
    Copy-Item -LiteralPath $ffmpeg.FullName -Destination (Join-Path $output 'ffmpeg.exe') -Force
    Copy-Item -LiteralPath $ffprobe.FullName -Destination (Join-Path $output 'ffprobe.exe') -Force
	$ffmpegLicense = Get-ChildItem -LiteralPath (Join-Path $staging 'ffmpeg') -Recurse -File | Where-Object { $_.Name -match '^LICENSE' } | Select-Object -First 1
	if ($ffmpegLicense) { Copy-Item -LiteralPath $ffmpegLicense.FullName -Destination (Join-Path $output 'FFmpeg-LICENSE.txt') -Force }

	$manifest = [ordered]@{
		schemaVersion = 2
		tools = [ordered]@{
			'yt-dlp' = [ordered]@{ version = '2026.07.04'; sha256 = (Get-FileHash -LiteralPath (Join-Path $output 'yt-dlp.exe') -Algorithm SHA256).Hash.ToLowerInvariant() }
			deno = [ordered]@{ version = '2.9.5'; sha256 = (Get-FileHash -LiteralPath (Join-Path $output 'deno.exe') -Algorithm SHA256).Hash.ToLowerInvariant() }
			ffmpeg = [ordered]@{ version = '9.0.1-essentials'; sha256 = (Get-FileHash -LiteralPath (Join-Path $output 'ffmpeg.exe') -Algorithm SHA256).Hash.ToLowerInvariant() }
			ffprobe = [ordered]@{ version = '9.0.1-essentials'; sha256 = (Get-FileHash -LiteralPath (Join-Path $output 'ffprobe.exe') -Algorithm SHA256).Hash.ToLowerInvariant() }
		}
	} | ConvertTo-Json -Depth 6
    [IO.File]::WriteAllText((Join-Path $output 'tools-manifest.json'), $manifest, [Text.UTF8Encoding]::new($false))
}
finally {
    if (Test-Path -LiteralPath $staging) { Remove-Item -LiteralPath $staging -Recurse -Force }
}

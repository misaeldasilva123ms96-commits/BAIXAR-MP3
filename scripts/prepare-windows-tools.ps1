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
        -Url 'https://github.com/yt-dlp/yt-dlp/releases/download/2026.06.09/yt-dlp.exe' `
        -Destination (Join-Path $output 'yt-dlp.exe') `
        -Sha256 '3a48cb955d55c8821b60ccbdbbc6f61bc958f2f3d3b7ad5eaf3d83a543293a27'

    $denoZip = Join-Path $staging 'deno.zip'
    Receive-VerifiedFile `
        -Url 'https://github.com/denoland/deno/releases/download/v2.8.1/deno-x86_64-pc-windows-msvc.zip' `
        -Destination $denoZip `
        -Sha256 '5fb5bac71f609fb91ec8960fb290885aadc27eeb22f07a8eca0c3db6be38b11a'
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
        tools = [ordered]@{ 'yt-dlp' = '2026.06.09'; deno = '2.8.1'; ffmpeg = '9.0.1-essentials' }
    } | ConvertTo-Json -Depth 4
    [IO.File]::WriteAllText((Join-Path $output 'tools-manifest.json'), $manifest, [Text.UTF8Encoding]::new($false))
}
finally {
    if (Test-Path -LiteralPath $staging) { Remove-Item -LiteralPath $staging -Recurse -Force }
}

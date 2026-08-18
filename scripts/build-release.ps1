[CmdletBinding()]
param(
    [Parameter(Mandatory)][ValidatePattern('^\d+\.\d+\.\d+$')][string]$Version,
    [Parameter(Mandatory)][string]$OutputDirectory
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$output = [IO.Path]::GetFullPath($OutputDirectory)
$project = [IO.Path]::GetFullPath($root)
if ($output.StartsWith($project, [StringComparison]::OrdinalIgnoreCase)) { throw 'A saída deve ficar fora da árvore do projeto.' }
if (Test-Path -LiteralPath $output) { throw "A saída já existe: $output" }

$packageName = "MP3_Downloader_v${Version}_Windows"
$package = Join-Path $output $packageName
New-Item -ItemType Directory -Path $package | Out-Null

Push-Location $root
try {
    npm ci --ignore-scripts
    npm run build
    $env:CGO_ENABLED = '0'; $env:GOOS = 'windows'; $env:GOARCH = 'amd64'
    go build -trimpath -buildvcs=false -ldflags '-s -w -buildid=' -o (Join-Path $package 'MP3_Downloader.exe') ./services/cmd/local-engine
}
finally { Pop-Location }

New-Item -ItemType Directory -Path (Join-Path $package 'web') | Out-Null
& (Join-Path $PSScriptRoot 'prepare-windows-tools.ps1') -OutputDirectory (Join-Path $root 'ferramentas')
$requiredTools = @('yt-dlp.exe','ffmpeg.exe','ffprobe.exe','deno.exe','tools-manifest.json','FFmpeg-LICENSE.txt')
$missingTools = $requiredTools | Where-Object { -not (Test-Path -LiteralPath (Join-Path $root "ferramentas/$_")) }
if ($missingTools) { throw "Ferramentas verificadas ausentes: $($missingTools -join ', ')" }
Copy-Item -Path (Join-Path $root 'apps/web/dist/*') -Destination (Join-Path $package 'web') -Recurse
Copy-Item -LiteralPath (Join-Path $root 'ferramentas') -Destination $package -Recurse
Copy-Item -LiteralPath (Join-Path $root 'docs') -Destination $package -Recurse
foreach ($file in @('ABRIR_MP3_DOWNLOADER.bat','Abrir_Baixador_MP3_V2.bat','Baixador_MP3_V2.ps1','configuracao.exemplo.txt','README.md','SECURITY.md','THIRD_PARTY_NOTICES.md','CHANGELOG.md')) {
    Copy-Item -LiteralPath (Join-Path $root $file) -Destination $package
}

$forbidden = Get-ChildItem -LiteralPath $package -Recurse -File | Where-Object {
    $_.Name -match '\.(log|mp3|download)$' -or $_.FullName -match '(historico_downloads|configuracao\.txt|temporarios|node_modules)'
}
if ($forbidden) { throw "O pacote contém dados proibidos: $($forbidden.FullName -join ', ')" }

$zip = Join-Path $output "$packageName.zip"
Compress-Archive -LiteralPath $package -DestinationPath $zip -CompressionLevel Optimal
$checksum = Join-Path $output 'checksums.sha256'
$files = Get-Item -LiteralPath $zip, (Join-Path $package 'MP3_Downloader.exe')
$lines = $files | ForEach-Object { "$(($_ | Get-FileHash -Algorithm SHA256).Hash.ToLowerInvariant())  $($_.Name)" }
[IO.File]::WriteAllLines($checksum, $lines, [Text.UTF8Encoding]::new($false))
$files + (Get-Item -LiteralPath $checksum) | Select-Object Name,Length

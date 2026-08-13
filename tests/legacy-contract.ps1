$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$script = Get-Content -LiteralPath (Join-Path $root 'Baixador_MP3_V2.ps1') -Raw

$required = @(
    '--download-archive', '--playlist-start', '--playlist-end', '--embed-thumbnail',
    '--embed-metadata', '--audio-quality', '320K', '256K', '192K', '128K',
    'ffmpeg.exe', 'ffprobe.exe', 'deno.exe', 'yt-dlp.exe'
)
foreach ($value in $required) {
    if (-not $script.Contains($value)) { throw "Contrato legado ausente: $value" }
}

$tokens = $null
$errors = $null
[void][Management.Automation.Language.Parser]::ParseFile(
    (Join-Path $root 'Baixador_MP3_V2.ps1'), [ref]$tokens, [ref]$errors
)
if ($errors.Count) { throw ($errors | Out-String) }
Write-Host 'Contrato e sintaxe do legado preservados.'

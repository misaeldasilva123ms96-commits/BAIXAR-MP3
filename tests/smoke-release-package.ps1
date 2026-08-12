[CmdletBinding()]
param([Parameter(Mandatory)][string]$PackagePath)

$ErrorActionPreference = 'Stop'
$package = [IO.Path]::GetFullPath($PackagePath)
$enginePath = Join-Path $package 'MP3_Downloader.exe'
$manifestPath = Join-Path $package 'ferramentas/tools-manifest.json'
if (-not (Test-Path -LiteralPath $enginePath) -or -not (Test-Path -LiteralPath $manifestPath)) {
    throw 'Pacote Windows incompleto.'
}

$smoke = Join-Path ([IO.Path]::GetTempPath()) ('mp3-release-smoke-' + [guid]::NewGuid().ToString('N'))
$resolvedTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$resolvedSmoke = [IO.Path]::GetFullPath($smoke)
if (-not $resolvedSmoke.StartsWith($resolvedTemp, [StringComparison]::OrdinalIgnoreCase)) { throw 'Destino temporário inválido.' }
New-Item -ItemType Directory -Path $resolvedSmoke | Out-Null

$env:MP3_ENGINE_TOKEN = '0123456789abcdef0123456789abcdef0123456789abcdef'
$env:MP3_NO_BROWSER = '1'
$env:MP3_DOWNLOAD_DIR = Join-Path $resolvedSmoke 'downloads'
$outLog = Join-Path $resolvedSmoke 'engine.out.log'
$errorLog = Join-Path $resolvedSmoke 'engine.err.log'
$engine = Start-Process -FilePath $enginePath -PassThru -WindowStyle Hidden -RedirectStandardOutput $outLog -RedirectStandardError $errorLog
Remove-Item Env:MP3_ENGINE_TOKEN, Env:MP3_NO_BROWSER, Env:MP3_DOWNLOAD_DIR

try {
    $health = $null
    for ($attempt = 0; $attempt -lt 30 -and -not $health; $attempt++) {
        Start-Sleep -Milliseconds 250
        try { $health = Invoke-RestMethod -Uri 'http://127.0.0.1:38765/health' -TimeoutSec 2 } catch { }
    }
    if (-not $health) { throw "Health do pacote indisponível: $(Get-Content -LiteralPath $errorLog -Raw)" }
    if (-not $health.ready) { throw "Ferramentas do pacote não ficaram prontas: $($health.tools | ConvertTo-Json -Compress)" }
    $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
    [pscustomobject]@{ Health = $health.status; Ready = $health.ready; Mode = $health.mode; Tools = $manifest.tools }
}
finally {
    if (-not $engine.HasExited) { Stop-Process -Id $engine.Id -Force; $engine.WaitForExit() }
    if (Test-Path -LiteralPath $resolvedSmoke) { [IO.Directory]::Delete($resolvedSmoke, $true) }
}

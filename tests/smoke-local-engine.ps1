$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$smokeRoot = Join-Path ([IO.Path]::GetTempPath()) ('mp3-engine-smoke-' + [guid]::NewGuid().ToString('N'))
$resolvedTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$resolvedSmoke = [IO.Path]::GetFullPath($smokeRoot)
if (-not $resolvedSmoke.StartsWith($resolvedTemp, [StringComparison]::OrdinalIgnoreCase)) { throw 'Destino temporário inválido.' }

New-Item -ItemType Directory -Path (Join-Path $smokeRoot 'web') | Out-Null
Copy-Item -Path (Join-Path $projectRoot 'apps/web/dist/*') -Destination (Join-Path $smokeRoot 'web') -Recurse
go build -trimpath -buildvcs=false -ldflags '-s -w -buildid=' -o (Join-Path $smokeRoot 'MP3_Downloader.exe') ./services/cmd/local-engine
if ($LASTEXITCODE) { throw 'Falha ao compilar Engine.' }

$env:MP3_ENGINE_TOKEN = '0123456789abcdef0123456789abcdef0123456789abcdef'
$env:MP3_NO_BROWSER = '1'
$env:MP3_DOWNLOAD_DIR = Join-Path $smokeRoot 'downloads'
$engineOut = Join-Path $smokeRoot 'engine.out.log'
$engineError = Join-Path $smokeRoot 'engine.err.log'
$engine = Start-Process -FilePath (Join-Path $smokeRoot 'MP3_Downloader.exe') -PassThru -WindowStyle Hidden -RedirectStandardOutput $engineOut -RedirectStandardError $engineError
Remove-Item Env:MP3_ENGINE_TOKEN, Env:MP3_NO_BROWSER, Env:MP3_DOWNLOAD_DIR

try {
    $health = $null
    for ($attempt = 0; $attempt -lt 20 -and -not $health; $attempt++) {
        Start-Sleep -Milliseconds 250
        try { $health = Invoke-RestMethod -Uri 'http://127.0.0.1:38765/health' -TimeoutSec 2 } catch { }
    }
    if (-not $health) { throw 'Health local indisponível.' }
    $index = Invoke-WebRequest -Uri 'http://127.0.0.1:38765/' -UseBasicParsing
    $newSettings = @{
        defaultQuality = '192'; downloadDirectory = (Join-Path $smokeRoot 'music')
        organizePlaylist = $true; avoidDuplicates = $true; embedThumbnail = $true
        embedMetadata = $true; openFolderWhenDone = $false
    } | ConvertTo-Json
    $saved = Invoke-RestMethod -Uri 'http://127.0.0.1:38765/settings' -Method Put `
        -Headers @{ 'X-MP3-Engine-Token' = '0123456789abcdef0123456789abcdef0123456789abcdef' } `
        -ContentType 'application/json' -Body $newSettings
    $listener = Get-NetTCPConnection -LocalPort 38765 -State Listen
    if ($listener.LocalAddress -ne '127.0.0.1') { throw "Bind inseguro: $($listener.LocalAddress)" }
    if ($health.status -ne 'ok' -or $health.mode -ne 'DESKTOP_LOCAL' -or $index.StatusCode -ne 200 -or $saved.defaultQuality -ne '192') { throw 'Smoke test local falhou.' }
    [pscustomobject]@{ Health = $health.status; Ready = $health.ready; Mode = $health.mode; Index = $index.StatusCode; SavedQuality = $saved.defaultQuality; Bind = $listener.LocalAddress }
}
finally {
    if (-not $engine.HasExited) { Stop-Process -Id $engine.Id -Force; $engine.WaitForExit() }
    if (Test-Path -LiteralPath $resolvedSmoke) { [IO.Directory]::Delete($resolvedSmoke, $true) }
}

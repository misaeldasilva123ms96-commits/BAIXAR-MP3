param()

$ErrorActionPreference = "Stop"
[Console]::InputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$ProgressPreference = "SilentlyContinue"
$Host.UI.RawUI.WindowTitle = "Baixador MP3 V2"

$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ToolsDir = Join-Path $RootDir "ferramentas"
$ConfigPath = Join-Path $RootDir "configuracao.txt"
$ArchivePath = Join-Path $RootDir "historico_downloads.txt"
$LogsDir = Join-Path $RootDir "logs"
$LastLogPath = Join-Path $RootDir "ultimo_download.log"
$TempDir = Join-Path $RootDir "temporarios"

New-Item -ItemType Directory -Force -Path $ToolsDir, $LogsDir, $TempDir | Out-Null

function Write-Header {
    param([string]$Subtitle = "")
    Clear-Host
    Write-Host "====================================================================" -ForegroundColor Cyan
    Write-Host "                     BAIXADOR MP3 V2" -ForegroundColor Cyan
    Write-Host "====================================================================" -ForegroundColor Cyan
    if (-not [string]::IsNullOrWhiteSpace($Subtitle)) {
        Write-Host $Subtitle -ForegroundColor DarkCyan
        Write-Host "--------------------------------------------------------------------" -ForegroundColor DarkCyan
    }
}

function Pause-App {
    Write-Host ""
    [void](Read-Host "Pressione ENTER para continuar")
}

function Get-Config {
    $config = @{
        "PASTA_DESTINO" = "%USERPROFILE%\Downloads\Musicas_MP3"
        "QUALIDADE_PADRAO" = "0"
        "ORGANIZAR_POR_PLAYLIST" = "SIM"
        "ABRIR_PASTA_AO_FINAL" = "SIM"
    }

    if (Test-Path -LiteralPath $ConfigPath) {
        foreach ($rawLine in Get-Content -LiteralPath $ConfigPath -Encoding UTF8) {
            $line = $rawLine.Trim()
            if ([string]::IsNullOrWhiteSpace($line) -or $line.StartsWith("#")) {
                continue
            }

            $separator = $line.IndexOf("=")
            if ($separator -lt 1) {
                continue
            }

            $key = $line.Substring(0, $separator).Trim().ToUpperInvariant()
            $value = $line.Substring($separator + 1).Trim()
            $config[$key] = $value
        }
    }

    return $config
}

function Save-Config {
    param([hashtable]$Config)

    $content = @(
        "# Configurações do Baixador MP3 V2",
        "# Use somente conteúdos próprios, licenciados, em domínio público ou autorizados.",
        "",
        "PASTA_DESTINO=$($Config['PASTA_DESTINO'])",
        "QUALIDADE_PADRAO=$($Config['QUALIDADE_PADRAO'])",
        "ORGANIZAR_POR_PLAYLIST=$($Config['ORGANIZAR_POR_PLAYLIST'])",
        "ABRIR_PASTA_AO_FINAL=$($Config['ABRIR_PASTA_AO_FINAL'])"
    )

    $content | Set-Content -LiteralPath $ConfigPath -Encoding UTF8
}

function Get-OutputDirectory {
    param([hashtable]$Config)

    $configured = $Config["PASTA_DESTINO"]
    if ([string]::IsNullOrWhiteSpace($configured)) {
        $configured = "%USERPROFILE%\Downloads\Musicas_MP3"
    }

    $expanded = [Environment]::ExpandEnvironmentVariables($configured)
    if (-not [System.IO.Path]::IsPathRooted($expanded)) {
        $expanded = Join-Path $RootDir $expanded
    }

    return [System.IO.Path]::GetFullPath($expanded)
}

function Test-YouTubeUrl {
    param([string]$Url)

    try {
        $uri = [System.Uri]$Url
        if ($uri.Scheme -notin @("http", "https")) {
            return $false
        }

        $hostName = $uri.Host.ToLowerInvariant()
        return ($hostName -eq "youtu.be" -or
                $hostName -eq "youtube.com" -or
                $hostName.EndsWith(".youtube.com") -or
                $hostName -eq "youtube-nocookie.com" -or
                $hostName.EndsWith(".youtube-nocookie.com"))
    }
    catch {
        return $false
    }
}

function Download-File {
    param(
        [Parameter(Mandatory = $true)][string]$Url,
        [Parameter(Mandatory = $true)][string]$Destination,
        [Parameter(Mandatory = $true)][string]$Label
    )

    Write-Host "Baixando $Label..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Destination) | Out-Null

    # Baixa primeiro para um arquivo temporário. Assim, uma queda de conexão não
    # deixa um executável incompleto que seria aceito na próxima inicialização.
    $partialPath = "$Destination.download"
    Remove-Item -LiteralPath $partialPath -Force -ErrorAction SilentlyContinue

    try {
        try {
            Invoke-WebRequest -Uri $Url -OutFile $partialPath -UseBasicParsing
        }
        catch {
            Write-Host "Tentando método alternativo de download..." -ForegroundColor DarkYellow
            Remove-Item -LiteralPath $partialPath -Force -ErrorAction SilentlyContinue
            Import-Module BitsTransfer -ErrorAction Stop
            Start-BitsTransfer -Source $Url -Destination $partialPath -ErrorAction Stop
        }

        if (-not (Test-Path -LiteralPath $partialPath) -or
            (Get-Item -LiteralPath $partialPath).Length -eq 0) {
            throw "O arquivo recebido está vazio."
        }

        Move-Item -LiteralPath $partialPath -Destination $Destination -Force
    }
    catch {
        Remove-Item -LiteralPath $partialPath -Force -ErrorAction SilentlyContinue
        throw "Não foi possível baixar $Label. $($_.Exception.Message)"
    }
}

function Test-UsableToolFile {
    param([Parameter(Mandatory = $true)][string]$Path)

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return $false
    }

    try {
        # Todos os executáveis usados pelo projeto são muito maiores que 64 KiB.
        # O limite também detecta respostas HTML e downloads claramente truncados.
        return (Get-Item -LiteralPath $Path).Length -ge 65536
    }
    catch {
        return $false
    }
}

function Ensure-Tools {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

    $ytDlp = Join-Path $ToolsDir "yt-dlp.exe"
    $ffmpeg = Join-Path $ToolsDir "ffmpeg.exe"
    $ffprobe = Join-Path $ToolsDir "ffprobe.exe"
    $deno = Join-Path $ToolsDir "deno.exe"

    if (-not (Test-UsableToolFile -Path $ytDlp)) {
        Download-File -Url "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe" -Destination $ytDlp -Label "yt-dlp"
    }

    if ((-not (Test-UsableToolFile -Path $ffmpeg)) -or
        (-not (Test-UsableToolFile -Path $ffprobe))) {
        $zipPath = Join-Path $ToolsDir "ffmpeg.zip"
        $extractPath = Join-Path $ToolsDir "_ffmpeg_extraido"
        Remove-Item -LiteralPath $extractPath -Recurse -Force -ErrorAction SilentlyContinue

        Download-File -Url "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip" -Destination $zipPath -Label "FFmpeg"
        Expand-Archive -LiteralPath $zipPath -DestinationPath $extractPath -Force

        $foundFfmpeg = Get-ChildItem -LiteralPath $extractPath -Recurse -File -Filter "ffmpeg.exe" | Select-Object -First 1
        $foundFfprobe = Get-ChildItem -LiteralPath $extractPath -Recurse -File -Filter "ffprobe.exe" | Select-Object -First 1
        if (-not $foundFfmpeg -or -not $foundFfprobe) {
            throw "Os executáveis do FFmpeg não foram encontrados no pacote baixado."
        }

        Copy-Item -LiteralPath $foundFfmpeg.FullName -Destination $ffmpeg -Force
        Copy-Item -LiteralPath $foundFfprobe.FullName -Destination $ffprobe -Force
        Remove-Item -LiteralPath $zipPath -Force -ErrorAction SilentlyContinue
        Remove-Item -LiteralPath $extractPath -Recurse -Force -ErrorAction SilentlyContinue
    }

    if (-not (Test-UsableToolFile -Path $deno)) {
        $zipPath = Join-Path $ToolsDir "deno.zip"
        $extractPath = Join-Path $ToolsDir "_deno_extraido"
        Remove-Item -LiteralPath $extractPath -Recurse -Force -ErrorAction SilentlyContinue

        Download-File -Url "https://github.com/denoland/deno/releases/latest/download/deno-x86_64-pc-windows-msvc.zip" -Destination $zipPath -Label "Deno"
        Expand-Archive -LiteralPath $zipPath -DestinationPath $extractPath -Force

        $foundDeno = Get-ChildItem -LiteralPath $extractPath -Recurse -File -Filter "deno.exe" | Select-Object -First 1
        if (-not $foundDeno) {
            throw "O executável do Deno não foi encontrado no pacote baixado."
        }

        Copy-Item -LiteralPath $foundDeno.FullName -Destination $deno -Force
        Remove-Item -LiteralPath $zipPath -Force -ErrorAction SilentlyContinue
        Remove-Item -LiteralPath $extractPath -Recurse -Force -ErrorAction SilentlyContinue
    }

    return @{
        "YtDlp" = $ytDlp
        "Ffmpeg" = $ffmpeg
        "Ffprobe" = $ffprobe
        "Deno" = $deno
    }
}

function Get-QualityLabel {
    param([string]$Quality)

    switch ($Quality.ToUpperInvariant()) {
        "0" { return "Alta — VBR 0" }
        "320K" { return "320 kbps" }
        "256K" { return "256 kbps" }
        "192K" { return "192 kbps" }
        "128K" { return "128 kbps" }
        default { return "Alta — VBR 0" }
    }
}

function Select-Quality {
    param([string]$CurrentQuality)

    Write-Host ""
    Write-Host "Qualidade do MP3:" -ForegroundColor Cyan
    Write-Host "  1. Alta qualidade — VBR 0 (recomendado)"
    Write-Host "  2. 320 kbps"
    Write-Host "  3. 256 kbps"
    Write-Host "  4. 192 kbps"
    Write-Host "  5. 128 kbps — arquivo menor"
    Write-Host ""
    Write-Host "Observação: aumentar o bitrate não melhora o áudio original do YouTube." -ForegroundColor DarkYellow
    $choice = Read-Host "Escolha [ENTER = $((Get-QualityLabel $CurrentQuality))]"

    if ([string]::IsNullOrWhiteSpace($choice)) {
        return $CurrentQuality
    }

    switch ($choice.Trim()) {
        "1" { return "0" }
        "2" { return "320K" }
        "3" { return "256K" }
        "4" { return "192K" }
        "5" { return "128K" }
        default {
            Write-Host "Opção inválida. Será usada a qualidade padrão." -ForegroundColor Yellow
            return $CurrentQuality
        }
    }
}

function Read-OptionalPositiveInteger {
    param([string]$Prompt)

    while ($true) {
        $raw = Read-Host $Prompt
        if ([string]::IsNullOrWhiteSpace($raw)) {
            return $null
        }

        $value = 0
        if ([int]::TryParse($raw.Trim(), [ref]$value) -and $value -gt 0) {
            return $value
        }

        Write-Host "Informe um número inteiro maior que zero ou deixe vazio." -ForegroundColor Yellow
    }
}

function Invoke-Download {
    Write-Header "Novo download"
    $config = Get-Config
    $tools = Ensure-Tools
    $outputDir = Get-OutputDirectory -Config $config
    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

    Write-Host "Use somente conteúdos próprios, licenciados, em domínio público ou autorizados." -ForegroundColor Yellow
    Write-Host ""
    $url = Read-Host "Cole o link da playlist ou do vídeo"
    $url = $url.Trim()

    if (-not (Test-YouTubeUrl -Url $url)) {
        Write-Host ""
        Write-Host "O link informado não parece ser uma URL válida do YouTube." -ForegroundColor Red
        Pause-App
        return
    }

    $quality = Select-Quality -CurrentQuality $config["QUALIDADE_PADRAO"]

    Write-Host ""
    $organizeDefault = $config["ORGANIZAR_POR_PLAYLIST"]
    $organizeAnswer = Read-Host "Criar uma pasta com o nome da playlist? [S/N, ENTER = $organizeDefault]"
    if ([string]::IsNullOrWhiteSpace($organizeAnswer)) {
        $organize = ($organizeDefault -eq "SIM")
    }
    else {
        $organize = ($organizeAnswer.Trim().ToUpperInvariant().StartsWith("S"))
    }

    Write-Host ""
    Write-Host "Faixa opcional da playlist — deixe vazio para baixar tudo." -ForegroundColor DarkCyan
    $playlistStart = Read-OptionalPositiveInteger -Prompt "Começar pelo item"
    $playlistEnd = Read-OptionalPositiveInteger -Prompt "Terminar no item"
    if ($playlistStart -and $playlistEnd -and $playlistStart -gt $playlistEnd) {
        Write-Host "O item inicial não pode ser maior que o item final." -ForegroundColor Red
        Pause-App
        return
    }

    if ($organize) {
        $outputTemplate = "%(playlist_title&{}/|)s%(playlist_index&{} - |)s%(title)s.%(ext)s"
    }
    else {
        $outputTemplate = "%(playlist_index&{} - |)s%(title)s.%(ext)s"
    }

    Write-Host ""
    Write-Host "Resumo:" -ForegroundColor Cyan
    Write-Host "  Destino:     $outputDir"
    Write-Host "  Qualidade:   $(Get-QualityLabel $quality)"
    Write-Host "  Organização: $(if ($organize) { 'pasta por playlist' } else { 'pasta principal' })"
    if ($playlistStart -or $playlistEnd) {
        $startLabel = if ($playlistStart) { $playlistStart.ToString() } else { "início" }
        $endLabel = if ($playlistEnd) { $playlistEnd.ToString() } else { "fim" }
        Write-Host "  Faixa:       $startLabel até $endLabel"
    }
    Write-Host ""
    $confirm = Read-Host "Iniciar o download? [S/N]"
    if (-not $confirm.Trim().ToUpperInvariant().StartsWith("S")) {
        Write-Host "Download cancelado." -ForegroundColor Yellow
        Pause-App
        return
    }

    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $logPath = Join-Path $LogsDir "download_$timestamp.log"
    $header = @(
        "Baixador MP3 V2",
        "Início: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')",
        "URL: $url",
        "Destino: $outputDir",
        "Qualidade: $(Get-QualityLabel $quality)",
        ""
    )
    $header | Set-Content -LiteralPath $logPath -Encoding UTF8

    $denoRuntime = "deno:$ToolsDir"
    $arguments = @(
        "--ignore-config",
        "--yes-playlist",
        "--ignore-errors",
        "--continue",
        "--no-overwrites",
        "--windows-filenames",
        "--trim-filenames", "180",
        "--download-archive", $ArchivePath,
        "--ffmpeg-location", $ToolsDir,
        "--js-runtimes", $denoRuntime,
        "--remote-components", "ejs:github",
        "--format", "bestaudio/best",
        "--extract-audio",
        "--audio-format", "mp3",
        "--audio-quality", $quality,
        "--embed-thumbnail",
        "--convert-thumbnails", "jpg",
        "--embed-metadata",
        "--retries", "10",
        "--fragment-retries", "10",
        "--extractor-retries", "5",
        "--retry-sleep", "2",
        "--concurrent-fragments", "3",
        "--console-title",
        "--progress",
        "--print", "after_move:Concluído: %(filepath)s",
        "-P", $outputDir,
        "-P", "temp:$TempDir",
        "-o", $outputTemplate
    )

    if ($playlistStart) {
        $arguments += @("--playlist-start", $playlistStart.ToString())
    }
    if ($playlistEnd) {
        $arguments += @("--playlist-end", $playlistEnd.ToString())
    }
    $arguments += $url

    Write-Host ""
    Write-Host "======================= DOWNLOAD EM ANDAMENTO ======================" -ForegroundColor Cyan
    Write-Host ""

    try {
        & $tools["YtDlp"] @arguments 2>&1 | ForEach-Object {
            $line = "$_"
            Write-Host $line
            $line | Add-Content -LiteralPath $logPath -Encoding UTF8
        }
        $exitCode = $LASTEXITCODE
    }
    catch {
        $exitCode = 1
        $errorLine = "ERRO: $($_.Exception.Message)"
        Write-Host $errorLine -ForegroundColor Red
        $errorLine | Add-Content -LiteralPath $logPath -Encoding UTF8
    }

    Copy-Item -LiteralPath $logPath -Destination $LastLogPath -Force

    Write-Host ""
    if ($exitCode -eq 0) {
        Write-Host "Download concluído com sucesso." -ForegroundColor Green
    }
    else {
        Write-Host "A execução terminou com avisos ou algum item não pôde ser baixado." -ForegroundColor Yellow
        Write-Host "Log: $logPath" -ForegroundColor Yellow
    }
    Write-Host "Pasta: $outputDir" -ForegroundColor Green

    if ($config["ABRIR_PASTA_AO_FINAL"] -eq "SIM") {
        try {
            Start-Process -FilePath "explorer.exe" -ArgumentList ('"{0}"' -f $outputDir)
        }
        catch {
            Write-Host "Não foi possível abrir a pasta automaticamente." -ForegroundColor DarkYellow
        }
    }

    Pause-App
}

function Open-DownloadsFolder {
    $config = Get-Config
    $outputDir = Get-OutputDirectory -Config $config
    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
    Start-Process -FilePath "explorer.exe" -ArgumentList ('"{0}"' -f $outputDir)
}

function Select-DestinationFolder {
    param([string]$CurrentPath)

    try {
        Add-Type -AssemblyName System.Windows.Forms
        $dialog = New-Object System.Windows.Forms.FolderBrowserDialog
        $dialog.Description = "Escolha a pasta onde os MP3 serão salvos"
        $dialog.ShowNewFolderButton = $true
        $expanded = [Environment]::ExpandEnvironmentVariables($CurrentPath)
        if (Test-Path -LiteralPath $expanded) {
            $dialog.SelectedPath = $expanded
        }

        $result = $dialog.ShowDialog()
        if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
            return $dialog.SelectedPath
        }
    }
    catch {
        Write-Host "O seletor visual não pôde ser aberto." -ForegroundColor Yellow
    }

    return $null
}

function Configure-App {
    while ($true) {
        $config = Get-Config
        $outputDir = Get-OutputDirectory -Config $config

        Write-Header "Configurações"
        Write-Host "  1. Pasta de destino: $outputDir"
        Write-Host "  2. Qualidade padrão: $(Get-QualityLabel $config['QUALIDADE_PADRAO'])"
        Write-Host "  3. Organizar por playlist: $($config['ORGANIZAR_POR_PLAYLIST'])"
        Write-Host "  4. Abrir pasta ao terminar: $($config['ABRIR_PASTA_AO_FINAL'])"
        Write-Host "  0. Voltar"
        Write-Host ""
        $choice = Read-Host "Escolha uma opção"

        switch ($choice.Trim()) {
            "1" {
                $selected = Select-DestinationFolder -CurrentPath $config["PASTA_DESTINO"]
                if ($selected) {
                    $config["PASTA_DESTINO"] = $selected
                    Save-Config -Config $config
                }
            }
            "2" {
                $config["QUALIDADE_PADRAO"] = Select-Quality -CurrentQuality $config["QUALIDADE_PADRAO"]
                Save-Config -Config $config
            }
            "3" {
                $config["ORGANIZAR_POR_PLAYLIST"] = if ($config["ORGANIZAR_POR_PLAYLIST"] -eq "SIM") { "NAO" } else { "SIM" }
                Save-Config -Config $config
            }
            "4" {
                $config["ABRIR_PASTA_AO_FINAL"] = if ($config["ABRIR_PASTA_AO_FINAL"] -eq "SIM") { "NAO" } else { "SIM" }
                Save-Config -Config $config
            }
            "0" { return }
            default {
                Write-Host "Opção inválida." -ForegroundColor Yellow
                Start-Sleep -Seconds 1
            }
        }
    }
}

function Update-And-CheckTools {
    Write-Header "Ferramentas"
    $tools = Ensure-Tools

    Write-Host "Atualizando yt-dlp no canal atual..." -ForegroundColor Yellow
    try {
        & $tools["YtDlp"] --ignore-config --update 2>&1 | ForEach-Object { Write-Host $_ }
    }
    catch {
        Write-Host "Não foi possível atualizar agora: $($_.Exception.Message)" -ForegroundColor Red
    }

    Write-Host ""
    Write-Host "Versões instaladas:" -ForegroundColor Cyan
    try { Write-Host "  yt-dlp: $(& $tools['YtDlp'] --version | Select-Object -First 1)" } catch { Write-Host "  yt-dlp: indisponível" }
    try { Write-Host "  Deno:   $(& $tools['Deno'] --version | Select-Object -First 1)" } catch { Write-Host "  Deno: indisponível" }
    try { Write-Host "  FFmpeg: $(& $tools['Ffmpeg'] -version 2>&1 | Select-Object -First 1)" } catch { Write-Host "  FFmpeg: indisponível" }

    Pause-App
}

function Open-LastLog {
    if (Test-Path -LiteralPath $LastLogPath) {
        Start-Process -FilePath "notepad.exe" -ArgumentList ('"{0}"' -f $LastLogPath)
    }
    else {
        Write-Header "Último log"
        Write-Host "Ainda não existe um log de download nesta versão." -ForegroundColor Yellow
        Pause-App
    }
}

function Clear-DownloadHistory {
    Write-Header "Limpar histórico"
    Write-Host "O histórico impede que a mesma música seja baixada novamente." -ForegroundColor Yellow
    Write-Host "Limpar o histórico NÃO apaga os MP3 já salvos." -ForegroundColor Yellow
    Write-Host ""
    $confirmation = Read-Host "Digite LIMPAR para confirmar"
    if ($confirmation.Trim().ToUpperInvariant() -eq "LIMPAR") {
        "" | Set-Content -LiteralPath $ArchivePath -Encoding UTF8
        Write-Host "Histórico limpo." -ForegroundColor Green
    }
    else {
        Write-Host "Operação cancelada." -ForegroundColor Yellow
    }
    Pause-App
}

# Cria a configuração padrão na primeira abertura.
if (-not (Test-Path -LiteralPath $ConfigPath)) {
    Save-Config -Config (Get-Config)
}

while ($true) {
    $config = Get-Config
    $outputDir = Get-OutputDirectory -Config $config

    Write-Header "Menu principal"
    Write-Host "Destino atual: $outputDir" -ForegroundColor DarkGray
    Write-Host "Qualidade:     $(Get-QualityLabel $config['QUALIDADE_PADRAO'])" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host "  1. Baixar playlist ou vídeo em MP3" -ForegroundColor Green
    Write-Host "  2. Abrir pasta de músicas"
    Write-Host "  3. Configurações"
    Write-Host "  4. Atualizar e verificar ferramentas"
    Write-Host "  5. Abrir o último log"
    Write-Host "  6. Limpar histórico de downloads"
    Write-Host "  0. Sair"
    Write-Host ""

    $choice = Read-Host "Escolha uma opção"
    switch ($choice.Trim()) {
        "1" { Invoke-Download }
        "2" { Open-DownloadsFolder }
        "3" { Configure-App }
        "4" { Update-And-CheckTools }
        "5" { Open-LastLog }
        "6" { Clear-DownloadHistory }
        "0" { break }
        default {
            Write-Host "Opção inválida." -ForegroundColor Yellow
            Start-Sleep -Seconds 1
        }
    }

    if ($choice.Trim() -eq "0") {
        break
    }
}

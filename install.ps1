# fire-commit installer for Windows
# Usage: iwr -useb https://raw.githubusercontent.com/lieyanc/fire-commit/master/install.ps1 | iex
#
# Options (pass via $args when not piping):
#   -Channel latest   Install latest channel (dev + stable builds, default)
#   -Channel stable   Install stable channel only
#
# Non-interactive examples:
#   & ([scriptblock]::Create((iwr -useb https://raw.githubusercontent.com/lieyanc/fire-commit/master/install.ps1))) -Channel latest
#   & ([scriptblock]::Create((iwr -useb https://raw.githubusercontent.com/lieyanc/fire-commit/master/install.ps1))) -Channel stable
param(
    [string]$Channel = ""
)

$ErrorActionPreference = "Stop"

$Repo       = "lieyanc/fire-commit"
$InstallDir = "$env:USERPROFILE\.fire-commit\bin"

function Write-Info($msg)  { Write-Host "==> $msg" -ForegroundColor Blue }
function Exit-Error($msg)  { Write-Host "Error: $msg" -ForegroundColor Red; exit 1 }

# --- Architecture detection -----------------------------------------------
switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $Arch = "amd64" }
    "ARM64" { $Arch = "arm64" }
    default  { Exit-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE. Only amd64 and arm64 are supported." }
}

# --- Channel selection -------------------------------------------------------
if (-not $Channel) {
    Write-Host ""
    Write-Host "Select update channel:" -ForegroundColor White
    Write-Host "  1) latest  - includes dev builds and stable releases (default)"
    Write-Host "  2) stable  - only stable releases"
    Write-Host ""
    $choice = Read-Host "Choice [1]"
    switch ($choice) {
        "2"     { $Channel = "stable" }
        default { $Channel = "latest" }
    }
}

Write-Info "Detected platform: windows/$Arch"
Write-Info "Channel: $Channel"

# --- Fetch release metadata --------------------------------------------------
function Get-ReleaseJson($url) {
    try {
        $resp = Invoke-WebRequest -Uri $url -UseBasicParsing -ErrorAction Stop
        return $resp.Content | ConvertFrom-Json
    } catch {
        Exit-Error "Failed to fetch release info from $url`n$_"
    }
}

if ($Channel -eq "stable") {
    Write-Info "Fetching latest stable release..."
    $release        = Get-ReleaseJson "https://api.github.com/repos/$Repo/releases/latest"
    $Tag            = $release.tag_name
    $VersionNum     = $Tag -replace '^v', ''
    $Archive        = "fire-commit_${VersionNum}_windows_${Arch}.zip"
    $DownloadUrl    = "https://github.com/$Repo/releases/download/$Tag/$Archive"
    $ChecksumsUrl   = "https://github.com/$Repo/releases/download/$Tag/checksums.txt"
    $DisplayVersion = $Tag
} else {
    Write-Info "Fetching latest dev release..."
    $release        = Get-ReleaseJson "https://api.github.com/repos/$Repo/releases/tags/dev"
    $DisplayVersion = $release.name
    $Archive        = "fire-commit_dev_windows_${Arch}.zip"
    $DownloadUrl    = "https://github.com/$Repo/releases/download/dev/$Archive"
    $ChecksumsUrl   = "https://github.com/$Repo/releases/download/dev/checksums.txt"
}

Write-Info "Version: $DisplayVersion"

# --- Download ----------------------------------------------------------------
$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    $ArchivePath    = Join-Path $TempDir $Archive
    $ChecksumsPath  = Join-Path $TempDir "checksums.txt"

    Write-Info "Downloading $Archive..."
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ArchivePath -UseBasicParsing

    # Checksum verification (best-effort)
    try {
        Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsPath -UseBasicParsing
    } catch {
        # checksums.txt unavailable — skip verification
    }

    if (Test-Path $ChecksumsPath) {
        $lines    = Get-Content $ChecksumsPath
        $expected = ($lines | Where-Object { $_ -match [regex]::Escape($Archive) }) -replace '\s.*$', ''
        if ($expected) {
            $actual = (Get-FileHash $ArchivePath -Algorithm SHA256).Hash.ToLower()
            if ($actual -ne $expected.ToLower()) {
                Exit-Error "Checksum verification failed!`n  Expected: $expected`n  Got:      $actual"
            }
            Write-Info "Checksum verified."
        }
    }

    # --- Extract & install ---------------------------------------------------
    Write-Info "Installing to $InstallDir..."
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force

    $Binary = Get-ChildItem -Path $TempDir -Recurse -Filter "firecommit.exe" | Select-Object -First 1
    if (-not $Binary) { Exit-Error "Could not find firecommit.exe in archive." }

    Copy-Item $Binary.FullName "$InstallDir\firecommit.exe" -Force

    $Fcmt = Get-ChildItem -Path $TempDir -Recurse -Filter "fcmt.exe" | Select-Object -First 1
    if ($Fcmt) { Copy-Item $Fcmt.FullName "$InstallDir\fcmt.exe" -Force }

    $GitFc = Get-ChildItem -Path $TempDir -Recurse -Filter "git-fire-commit.exe" | Select-Object -First 1
    if ($GitFc) { Copy-Item $GitFc.FullName "$InstallDir\git-fire-commit.exe" -Force }

    # --- Write update channel to config --------------------------------------
    # On Windows, adrg/xdg resolves XDG_CONFIG_HOME to %APPDATA%
    $ConfigDir  = if ($env:XDG_CONFIG_HOME) { "$env:XDG_CONFIG_HOME\firecommit" } else { "$env:APPDATA\firecommit" }
    $ConfigFile = Join-Path $ConfigDir "config.yaml"
    New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null

    if (Test-Path $ConfigFile) {
        $content = Get-Content $ConfigFile -Raw
        if ($content -match '(?m)^update_channel:') {
            $content = $content -replace '(?m)^update_channel:.*$', "update_channel: $Channel"
        } else {
            $content = $content.TrimEnd() + "`nupdate_channel: $Channel`n"
        }
        Set-Content $ConfigFile $content -NoNewline
    } else {
        Set-Content $ConfigFile "update_channel: $Channel`n"
    }
    Write-Info "Update channel set to '$Channel' in $ConfigFile"

    # --- Configure PATH ------------------------------------------------------
    $UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($UserPath -notlike "*\.fire-commit\bin*") {
        [Environment]::SetEnvironmentVariable("PATH", "$InstallDir;$UserPath", "User")
        Write-Info "Added $InstallDir to user PATH."
    }

    Write-Host ""
    Write-Host "fire-commit $DisplayVersion installed successfully!" -ForegroundColor Green
    Write-Host "Channel: $Channel"
    Write-Host "Restart your terminal, or load the new PATH in this session:"
    Write-Host "  `$env:PATH = `"$InstallDir;`$env:PATH`""
    Write-Host "Then run: firecommit --help"

} finally {
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}

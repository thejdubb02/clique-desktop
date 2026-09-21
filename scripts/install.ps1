# CLIque desktop client — one-line install.
#
#   irm https://raw.githubusercontent.com/thejdubb02/clique-desktop/main/scripts/install.ps1 | iex
#
# Downloads the latest CLIque-Setup.exe, verifies it against the checksum
# published alongside it (scripts/package.sh computes both from the same
# signed build), and runs it. No admin needed: the installer puts CLIque in
# your own user profile, adds a Start Menu entry and an uninstaller.
#
# The installer and the app are signed with our own certificate, not one
# from a paid CA (see the main README for why). Windows SmartScreen still
# runs its own separate reputation check on a first download regardless of
# signing, which a self-signed cert cannot clear — if you see a blue
# "Windows protected your PC" screen, that is expected on first install
# only, click "More info" then "Run anyway".

$ErrorActionPreference = "Stop"

$Repo = "thejdubb02/clique-desktop"
$Base = "https://github.com/$Repo/releases/latest/download"
$Tmp = Join-Path $env:TEMP "clique-install-$([guid]::NewGuid())"
New-Item -ItemType Directory -Path $Tmp | Out-Null

try {
    $Exe = Join-Path $Tmp "CLIque-Setup.exe"
    $Sum = Join-Path $Tmp "CLIque-Setup.exe.sha256"

    Write-Host "Downloading CLIque..."
    Invoke-WebRequest -Uri "$Base/CLIque-Setup.exe" -OutFile $Exe
    Invoke-WebRequest -Uri "$Base/CLIque-Setup.exe.sha256" -OutFile $Sum

    # The checksum file is "<hex>  CLIque-Setup.exe", the same format
    # sha256sum writes and update.go's own verifier reads.
    $Expected = (Get-Content $Sum -Raw).Trim().Split(" ")[0].ToLower()
    $Actual = (Get-FileHash -Path $Exe -Algorithm SHA256).Hash.ToLower()
    if ($Expected -ne $Actual) {
        Write-Error "checksum mismatch: expected $Expected, got $Actual — not running this file. Download may be corrupt or tampered; try again, and if it keeps happening say so."
        exit 1
    }
    Write-Host "Checksum verified."

    Write-Host "Installing..."
    Start-Process -FilePath $Exe -ArgumentList "/S" -Wait

    Write-Host "Done. CLIque should open on its own; if it doesn't, it's in your Start Menu."
}
finally {
    Remove-Item -Path $Tmp -Recurse -Force -ErrorAction SilentlyContinue
}

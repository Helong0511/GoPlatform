$ErrorActionPreference = "Stop"

param(
    [ValidateSet("debug", "release")]
    [string]$Profile = "release"
)

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptDir
$rustDir = Join-Path $repoRoot "rust-security"

if (-not (Test-Path -LiteralPath (Join-Path $rustDir "Cargo.toml"))) {
    Write-Error "rust-security/Cargo.toml not found."
    exit 1
}

if (-not (Get-Command cargo -ErrorAction SilentlyContinue)) {
    Write-Error "cargo is required but was not found in PATH."
    exit 1
}

Push-Location $rustDir
try {
    if ($Profile -eq "release") {
        cargo build --release
    } else {
        cargo build
    }
} finally {
    Pop-Location
}

Write-Output "[INFO] Rust security library build completed. profile=$Profile"

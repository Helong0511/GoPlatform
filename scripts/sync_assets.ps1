$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot

$mappings = @(
    @{ Source = Join-Path $repoRoot "ubuntu\nginx\nginx.conf"; Target = Join-Path $PSScriptRoot "nginx.conf" },
    @{ Source = Join-Path $repoRoot "ubuntu\postgresql\postgresql.conf"; Target = Join-Path $PSScriptRoot "postgresql.conf" },
    @{ Source = Join-Path $repoRoot "ubuntu\service\ad-platform-main.service"; Target = Join-Path $PSScriptRoot "ad-platform-main.service" },
    @{ Source = Join-Path $repoRoot "sql\init_db_PostgreSQL.sql"; Target = Join-Path $PSScriptRoot "init_db_PostgreSQL.sql" },
    @{ Source = Join-Path $repoRoot "sql\init_db_mysql.sql"; Target = Join-Path $PSScriptRoot "init_db_mysql.sql" }
)

foreach ($item in $mappings) {
    if (-not (Test-Path -LiteralPath $item.Source)) {
        Write-Warning ("Skip missing source: {0}" -f $item.Source)
        continue
    }

    Copy-Item -LiteralPath $item.Source -Destination $item.Target -Force
    Write-Output ("Synced: {0}" -f $item.Target)
}

Write-Output "Script assets sync completed."

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

# 只扫描常见文本配置与源码文件，减少误报与性能消耗
$includeExtensions = @(
    ".go", ".md", ".yaml", ".yml", ".json", ".ini", ".conf", ".service",
    ".env", ".txt", ".sql", ".ps1", ".sh", ".js", ".jsx", ".ts", ".tsx"
)

$excludeDirs = @(
    ".git", "node_modules", "admin\node_modules", "dist", "build", "bin"
)

function Is-ExcludedPath {
    param([string]$fullPath)
    foreach ($dir in $excludeDirs) {
        if ($fullPath -like "*$dir*") {
            return $true
        }
    }
    return $false
}

$rules = @(
    @{
        Name = "JWT secret assignment"
        Regex = '(?i)\bJWT_SECRET\s*=\s*["''][^"'']+["'']|\bJWT_SECRET\s*:\s*["''][^"'']+["'']'
    },
    @{
        Name = "AES secret assignment"
        Regex = '(?i)\bAES_SECRET_KEY\s*=\s*["''][^"'']+["'']|\baes_secret\s*:\s*["''][^"'']+["'']'
    },
    @{
        Name = "DB password assignment"
        Regex = '(?i)\bDB_PASSWORD\s*=\s*["''][^"'']+["'']|\bpassword\s*:\s*["''][^"'']+["'']'
    }
)

$ignoredValueHints = @(
    "__REPLACE_ME__", '""', "''", "password: \"\""
)

$findings = New-Object System.Collections.Generic.List[object]

$files = Get-ChildItem -Path $repoRoot -Recurse -File | Where-Object {
    $ext = $_.Extension.ToLowerInvariant()
    ($includeExtensions -contains $ext) -and -not (Is-ExcludedPath $_.FullName)
}

foreach ($file in $files) {
    $lineNumber = 0
    foreach ($line in Get-Content -LiteralPath $file.FullName) {
        $lineNumber++
        $trimmed = $line.Trim()
        if ($trimmed.StartsWith("#") -or $trimmed.StartsWith("//")) {
            continue
        }

        $ignoreLine = $false
        foreach ($hint in $ignoredValueHints) {
            if ($line -like "*$hint*") {
                $ignoreLine = $true
                break
            }
        }
        if ($ignoreLine) {
            continue
        }

        foreach ($rule in $rules) {
            if ($line -match $rule.Regex) {
                $relativePath = $file.FullName.Substring($repoRoot.Length + 1)
                $findings.Add([PSCustomObject]@{
                    Rule = $rule.Name
                    File = $relativePath
                    Line = $lineNumber
                    Content = $trimmed
                })
            }
        }
    }
}

if ($findings.Count -gt 0) {
    Write-Host "Potential hardcoded secrets found:" -ForegroundColor Red
    foreach ($item in $findings) {
        Write-Host ("[{0}] {1}:{2}" -f $item.Rule, $item.File, $item.Line) -ForegroundColor Yellow
        Write-Host ("  {0}" -f $item.Content)
    }
    Write-Host ""
    Write-Host "Please replace with environment variables or __REPLACE_ME__ placeholders." -ForegroundColor Red
    exit 1
}

Write-Host "No obvious hardcoded secret assignments detected." -ForegroundColor Green
exit 0

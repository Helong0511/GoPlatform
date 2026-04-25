$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

# 只扫描常见文本配置与源码文件，减少误报与性能消耗
$includeExtensions = @(".go", ".md", ".yaml", ".yml", ".json", ".ini", ".conf", ".service", ".env", ".txt", ".sql", ".ps1", ".sh", ".js", ".jsx", ".ts", ".tsx")

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
        Regex = '(?i)\bJWT_SECRET\s*[:=]'
    },
    @{
        Name = "AES secret assignment"
        Regex = '(?i)\b(AES_SECRET_KEY|aes_secret)\s*[:=]'
    },
    @{
        Name = "DB password assignment"
        Regex = '(?i)(\bDB_PASSWORD\b|\bpassword\b)\s*[:=]'
    }
)

$fullWidthColon = [char]65306
$dbPasswordLabel = -join ([char[]](25968, 25454, 24211, 23494, 30721))

$ignoredValueHints = @(
    "__REPLACE_ME__", '""', "''", 'password: ""'
)

$findings = New-Object System.Collections.Generic.List[object]

function Mask-SecretLine {
    param([string]$line)
    return ($line -replace "([:=$fullWidthColon]\s*).+$", '$1***')
}

$trackedPaths = @()
try {
    $trackedPaths = git -C $repoRoot -c core.quotepath=false ls-files
} catch {
    $trackedPaths = @()
}

if ($trackedPaths.Count -gt 0) {
    $files = $trackedPaths | ForEach-Object {
        Join-Path $repoRoot $_
    } | Where-Object {
        (Test-Path -LiteralPath $_) -and -not (Is-ExcludedPath $_)
    } | ForEach-Object {
        Get-Item -LiteralPath $_
    } | Where-Object {
        $includeExtensions -contains $_.Extension.ToLowerInvariant()
    }
} else {
    # 回退模式：无 git 环境时扫描工作区文件
    $files = Get-ChildItem -Path $repoRoot -Recurse -File | Where-Object {
        $ext = $_.Extension.ToLowerInvariant()
        ($includeExtensions -contains $ext) -and -not (Is-ExcludedPath $_.FullName)
    }
}

$infoFile = Join-Path $repoRoot "ubuntu\info.txt"
if (Test-Path -LiteralPath $infoFile) {
    $files = @($files) + @(Get-Item -LiteralPath $infoFile)
}
$files = $files | Sort-Object -Property FullName -Unique

foreach ($file in $files) {
    $lineNumber = 0
    foreach ($line in Get-Content -LiteralPath $file.FullName -Encoding UTF8) {
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

        $matchedRules = @()
        foreach ($rule in $rules) {
            if ($line -match $rule.Regex) {
                $matchedRules += $rule.Name
            }
        }

        if ($line -like "*$dbPasswordLabel*") {
            $matchedRules += "DB password assignment"
        }

        foreach ($ruleName in ($matchedRules | Select-Object -Unique)) {
            # 过滤明显的空值与模板占位
            if ($line -match "(?i)[:=$fullWidthColon]\s*[""']?\s*[""']?\s*$") {
                continue
            }
            $relativePath = $file.FullName.Substring($repoRoot.Length + 1)
            $findings.Add([PSCustomObject]@{
                Rule = $ruleName
                File = $relativePath
                Line = $lineNumber
                Content = (Mask-SecretLine $trimmed)
            })
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

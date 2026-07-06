param(
    [string]$RemoteHost = "47.85.29.80",
    [string]$RemoteUser = "root",
    [string]$IdentityFile = "$HOME\.ssh\id_ed25519_aliyun",
    [string]$LocalRoot = "D:\ServerBackups\recurdream",
    [int]$DailyRetentionDays = 30,
    [int]$WeeklyRetentionMonths = 12
)

$ErrorActionPreference = "Stop"

function Write-Log {
    param([string]$Message)

    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $line = "[$timestamp] $Message"
    Write-Output $line
    Add-Content -LiteralPath $script:LogFile -Value $line -Encoding UTF8
}

function Remove-OldBackupDirectories {
    param(
        [string]$BackupRoot,
        [datetime]$OlderThan,
        [string]$Description
    )

    if (-not (Test-Path -LiteralPath $BackupRoot)) {
        return
    }

    $resolvedLocalRoot = (Resolve-Path -LiteralPath $LocalRoot).Path.TrimEnd('\')
    $resolvedBackupRoot = (Resolve-Path -LiteralPath $BackupRoot).Path.TrimEnd('\')

    if (-not $resolvedBackupRoot.StartsWith($resolvedLocalRoot + '\', [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to clean backups outside local root: $resolvedBackupRoot"
    }

    Get-ChildItem -LiteralPath $BackupRoot -Directory |
        Where-Object { $_.LastWriteTime -lt $OlderThan } |
        ForEach-Object {
            $candidate = (Resolve-Path -LiteralPath $_.FullName).Path
            if (-not $candidate.StartsWith($resolvedLocalRoot + '\', [StringComparison]::OrdinalIgnoreCase)) {
                throw "Refusing to remove path outside local root: $candidate"
            }
            Remove-Item -LiteralPath $candidate -Recurse -Force
        }

    Write-Log "Applied local $Description retention"
}

New-Item -ItemType Directory -Force -Path $LocalRoot | Out-Null
$script:LogFile = Join-Path $LocalRoot "pull.log"

$remoteDaily = "${RemoteUser}@${RemoteHost}:/backup/recurdream/daily"
$remoteWeekly = "${RemoteUser}@${RemoteHost}:/backup/recurdream/weekly"
$scpArgs = @(
    "-i", $IdentityFile,
    "-o", "BatchMode=yes",
    "-o", "ConnectTimeout=20",
    "-r",
    $remoteDaily,
    $remoteWeekly,
    $LocalRoot
)

Write-Log "Starting backup pull from ${RemoteUser}@${RemoteHost}:/backup/recurdream/"

try {
    & scp @scpArgs
    if ($LASTEXITCODE -ne 0) {
        throw "scp exited with code $LASTEXITCODE"
    }

    Write-Log "Backup pull completed"

    $dailyRoot = Join-Path $LocalRoot "daily"
    Remove-OldBackupDirectories `
        -BackupRoot $dailyRoot `
        -OlderThan (Get-Date).AddDays(-$DailyRetentionDays) `
        -Description "daily: $DailyRetentionDays days"

    $weeklyRoot = Join-Path $LocalRoot "weekly"
    Remove-OldBackupDirectories `
        -BackupRoot $weeklyRoot `
        -OlderThan (Get-Date).AddMonths(-$WeeklyRetentionMonths) `
        -Description "weekly: $WeeklyRetentionMonths months"
} catch {
    Write-Log "Backup pull failed: $($_.Exception.Message)"
    throw
}

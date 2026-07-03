param(
    [string]$Python = "python3"
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Push-Location $repoRoot
try {
    & $Python "scripts\verify_closed_loop.py"
} finally {
    Pop-Location
}

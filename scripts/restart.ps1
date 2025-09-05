$ErrorActionPreference = "Stop"

$Root = (Resolve-Path "$PSScriptRoot/..")
$Bin = Join-Path $Root "bin"
$Run = Join-Path $Root ".run"
$App = Join-Path $Bin "server.exe"
$PidFile = Join-Path $Run "server.pid"

New-Item -ItemType Directory -Force -Path $Bin | Out-Null
New-Item -ItemType Directory -Force -Path $Run | Out-Null

Write-Host "Building..."
go build -o $App ./cmd/server

if (Test-Path $PidFile) {
  $pid = Get-Content $PidFile -ErrorAction SilentlyContinue
  if ($pid -and (Get-Process -Id $pid -ErrorAction SilentlyContinue)) {
    Write-Host "Stopping existing process $pid..."
    Stop-Process -Id $pid -Force
  }
}

Write-Host "Starting..."
Start-Process -FilePath $App -PassThru | ForEach-Object { $_.Id } | Set-Content $PidFile
Write-Host "Started with PID $(Get-Content $PidFile)"


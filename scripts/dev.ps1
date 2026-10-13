# Start the API with the current Windows user proxy unless explicitly overridden.
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
& (Join-Path $PSScriptRoot "initialize-environment.ps1")
& (Join-Path $PSScriptRoot "prepare-pansou.ps1")
& (Join-Path $PSScriptRoot "generate-pansou.ps1") -Check
go run ./cmd/server
exit $LASTEXITCODE
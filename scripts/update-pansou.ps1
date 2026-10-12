$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$submodule = Join-Path $repo 'third_party/pansou'
$allowed = @('plugin/plugin.go', 'util/cache/sharded_memory_cache.go')
Push-Location $repo
try {
    $dirty = @(git -C $submodule status --porcelain)
    if ($dirty.Count -gt 0) {
        $other = @($dirty | Where-Object { $line = $_.Substring(3).Replace('\', '/'); $line -notin $allowed })
        if ($other.Count -gt 0) { throw 'PanSou submodule has local changes outside the recorded lifecycle files; preserve or resolve them before updating' }
        $currentPatch = git -C $submodule diff -- plugin/plugin.go util/cache/sharded_memory_cache.go | Out-String
        $savedPatch = [IO.File]::ReadAllText((Join-Path $repo 'third_party/pansou-lifecycle.patch'))
        if ($currentPatch.Trim() -ne $savedPatch.Trim()) { throw 'PanSou files have edits beyond the recorded lifecycle patch; refusing to discard them' }
        git -C $submodule restore -- plugin/plugin.go util/cache/sharded_memory_cache.go
    }
    git submodule update --remote -- third_party/pansou
    if ($LASTEXITCODE -ne 0) { throw 'submodule update failed' }
    $main = Get-Content (Join-Path $submodule 'main.go')
    $active = @($main | ForEach-Object {
        if ($_ -match '^\s*_\s+"pansou/plugin/([^"]+)"') { " _ `"pansou/plugin/$($Matches[1])`"" }
    })
    if ($active.Count -eq 0) { throw 'no active PanSou plugin imports found' }
    $generated = @(
        '// Code generated from active blank imports in third_party/pansou/main.go; DO NOT EDIT.'
        'package pansoubridge'
        ''
        'import ('
        $active
        ')'
        ''
    ) -join "`n"
    Set-Content -Encoding utf8 (Join-Path $repo 'internal/pansoubridge/plugins_generated.go') $generated
    & (Join-Path $PSScriptRoot 'prepare-pansou.ps1')
    if ($LASTEXITCODE -ne 0) { throw 'PanSou lifecycle preparation failed' }
    $upstreamMod = Get-Content (Join-Path $submodule 'go.mod') -Raw
    if ($upstreamMod -notmatch '(?m)^module\s+pansou\s*$') { throw 'PanSou module path changed; update the bridge imports before proceeding' }
    $rootMod = Get-Content (Join-Path $repo 'go.mod') -Raw
    if ($rootMod -notmatch '(?m)^\s*pansou\s+v\S+') { throw 'go.mod is missing the local PanSou module requirement' }
    if ($rootMod -notmatch '(?m)^replace\s+pansou\s+=>\s+\./third_party/pansou\s*$') { throw 'go.mod is missing the PanSou submodule replace' }
    go mod tidy
    if ($LASTEXITCODE -ne 0) { throw 'go mod tidy failed' }
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'go test failed' }
    $bin = Join-Path $env:TEMP 'media-center-api-pansou-update-check.exe'
    go build -o $bin ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw 'single-binary build failed' }
    Remove-Item -LiteralPath $bin -Force
    Write-Host "PanSou pinned commit: $(git -C $submodule rev-parse HEAD)"
    Write-Host "Verified $($active.Count) active plugin imports, go.mod, tests, and one-binary build."
} finally { Pop-Location }


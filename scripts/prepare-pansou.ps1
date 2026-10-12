$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$submodule = Join-Path $repo 'third_party/pansou'
$cacheSource = Join-Path $submodule 'util/cache/sharded_memory_cache.go'
$pluginSource = Join-Path $submodule 'plugin/plugin.go'
$hasCacheStop = Select-String -Path $cacheSource -Pattern 'func StopGlobalCleanupTask' -Quiet
$hasPluginWait = Select-String -Path $pluginSource -Pattern 'func WaitForAsyncTasks' -Quiet
if ($hasCacheStop -and $hasPluginWait) { Write-Host 'PanSou lifecycle hooks already prepared.'; exit 0 }
$patch = Join-Path $repo 'third_party/pansou-lifecycle.patch'
git -C $submodule apply --check (Join-Path '..' 'pansou-lifecycle.patch')
if ($LASTEXITCODE -ne 0) { throw 'PanSou lifecycle patch no longer applies; inspect upstream changes before building' }
git -C $submodule apply (Join-Path '..' 'pansou-lifecycle.patch')
if ($LASTEXITCODE -ne 0) { throw 'failed to apply PanSou lifecycle patch' }
Write-Host 'Applied PanSou cache-stop and async-drain hooks.'

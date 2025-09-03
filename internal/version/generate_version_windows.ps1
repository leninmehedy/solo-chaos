$commit = git rev-parse HEAD

$versionFile = Join-Path $PSScriptRoot "VERSION"
$commitFile = Join-Path $PSScriptRoot "COMMIT"

if (!(Test-Path $versionFile))
{
    Set-Content -Path $versionFile -Value "0.0.0" -Force
    Write-Host "Generated VERSION file with the default version...."
}

Set-Location $PSScriptRoot
Set-Content -Path $commitFile -Value $commit -Force
Write-Host "Generated COMMIT file with the current commit hash...."

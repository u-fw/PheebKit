Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$packages = @(& (Join-Path $PSScriptRoot 'project-go-packages.ps1'))
if ($LASTEXITCODE -ne 0 -or $packages.Count -eq 0) {
    throw 'Unable to enumerate first-party Go packages.'
}

& go test @packages
if ($LASTEXITCODE -ne 0) {
    throw "go test failed with exit code $LASTEXITCODE"
}

& go vet @packages
if ($LASTEXITCODE -ne 0) {
    throw "go vet failed with exit code $LASTEXITCODE"
}

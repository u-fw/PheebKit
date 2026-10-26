Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$allPackages = @(& go list ./...)
if ($LASTEXITCODE -ne 0) {
    throw "go list failed with exit code $LASTEXITCODE"
}

$projectPackages = @(
    $allPackages |
        Where-Object { $_ -and $_ -notmatch '(^|/)frontend/node_modules(?:/|$)' }
)
if ($projectPackages.Count -eq 0) {
    throw 'No first-party Go packages were found.'
}

$projectPackages

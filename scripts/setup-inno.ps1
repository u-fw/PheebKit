Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'Inno Setup compiler provisioning requires Windows.' }
$workspace = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$directory = Join-Path $workspace '.tools/inno-7.1.0'
$downloadDirectory = Join-Path $workspace '.tools/inno-download'
$installer = Join-Path $downloadDirectory 'innosetup-7.1.0-x64.exe'
New-Item -ItemType Directory -Force -Path $downloadDirectory | Out-Null
if (-not (Test-Path -LiteralPath $installer)) {
    Invoke-WebRequest 'https://github.com/jrsoftware/issrc/releases/download/is-7_1_0/innosetup-7.1.0-x64.exe' -OutFile $installer
}
if ((Get-FileHash -LiteralPath $installer -Algorithm SHA256).Hash.ToLowerInvariant() -ne '0362a383ed217d4c4239b5933866dd96d3eb2102737da92f80f6057a4b40df2f') { throw 'Inno Setup download SHA-256 mismatch' }
$signature = Get-AuthenticodeSignature -LiteralPath $installer
if ($signature.Status -ne 'Valid' -or $signature.SignerCertificate.Subject -notmatch 'Pyrsys B.V.') { throw 'Inno Setup publisher verification failed' }
if (Test-Path -LiteralPath (Join-Path $directory 'ISCC.exe')) {
    foreach ($file in @('ISCC.exe', 'ISCmplr.dll')) {
        $installedSignature = Get-AuthenticodeSignature -LiteralPath (Join-Path $directory $file)
        if ($installedSignature.Status -ne 'Valid' -or $installedSignature.SignerCertificate.Subject -notmatch 'Pyrsys B.V.') { throw "Installed compiler signature invalid: $file" }
    }
    Write-Host "Reusing verified portable compiler: $directory"
    exit 0
}
$process = Start-Process -FilePath $installer -ArgumentList @('/PORTABLE=1', '/VERYSILENT', '/SUPPRESSMSGBOXES', '/SP-', '/NORESTART', '/NOICONS', ('/DIR="' + $directory + '"')) -WindowStyle Hidden -PassThru
if (-not $process.WaitForExit(60000)) { throw "Compiler provisioning timed out (PID $($process.Id)); inspect that process before retrying." }
if ($process.ExitCode -ne 0 -or -not (Test-Path -LiteralPath (Join-Path $directory 'ISCC.exe'))) { throw "Inno compiler provisioning failed: $($process.ExitCode)" }
Write-Host "Verified portable compiler installed: $directory"

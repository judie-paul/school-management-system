# Builds build\school.exe from the sources in code\. Run from any directory:
#   powershell -ExecutionPolicy Bypass -File build.ps1
#   powershell -ExecutionPolicy Bypass -File build.ps1 -Test   (run tests first)
param([switch]$Test)
$ErrorActionPreference = 'Stop'

Push-Location (Join-Path $PSScriptRoot 'code')
try {
    if ($Test) {
        go test ./...
        if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
    }
    go build -o ..\build\school.exe .\cmd\server
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
    Write-Host 'Built build\school.exe'
}
finally {
    Pop-Location
}

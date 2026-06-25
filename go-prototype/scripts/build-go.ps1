param(
    [string]$OutputDirectory = "publish"
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $projectRoot
try {
    $goPath = "C:\Program Files\Go\bin"
    $gccPath = "C:\msys64\ucrt64\bin"
    $env:Path = "$goPath;$gccPath;$env:Path"
    $env:CGO_ENABLED = "1"

    $winres = Join-Path $env:USERPROFILE "go\bin\go-winres.exe"
    if (-not (Test-Path -LiteralPath $winres)) {
        go install github.com/tc-hib/go-winres@v0.3.3
    }

    & $winres make `
        --in "winres\winres.json" `
        --arch "amd64" `
        --out "cmd\listetiime\rsrc"
    if ($LASTEXITCODE -ne 0) {
        throw "Échec de la génération des ressources Windows."
    }

    go test ./...
    if ($LASTEXITCODE -ne 0) {
        throw "Échec des tests Go."
    }

    New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
    $output = Join-Path $OutputDirectory "Liste Tiime Go.exe"
    Remove-Item -LiteralPath $output -Force -ErrorAction SilentlyContinue
    go build `
        -trimpath `
        -ldflags "-s -w -H=windowsgui" `
        -o $output `
        ".\cmd\listetiime"
    if ($LASTEXITCODE -ne 0) {
        throw "Échec de la compilation Go."
    }

    Get-Item -LiteralPath $output
}
finally {
    Pop-Location
}

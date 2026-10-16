param(
    [switch]$PrepareOnly,
    [double]$Seconds = 60,
    [long]$SamplingSeed = 20261016
)
$ErrorActionPreference = 'Stop'
$repoRoot = (Get-Item -LiteralPath $PSScriptRoot).Parent.Parent.FullName
$activeHelper = Join-Path $repoRoot 'src\dungeon_trials_experiment_test.go'
$trialExecutable = Join-Path $PSScriptRoot 'trials.exe'
$gameExecutable = Join-Path $PSScriptRoot 'ghostdog-verification.exe'
$env:TEMP = Join-Path $PSScriptRoot 'temp'
$env:TMP = $env:TEMP
New-Item -ItemType Directory -Path $env:TEMP -Force | Out-Null
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $PSScriptRoot 'go-cache'
}
if (Test-Path -LiteralPath $activeHelper) {
    throw "Refusing to overwrite existing helper: $activeHelper"
}
Push-Location $repoRoot
try {
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'helper_test.go.txt') -Destination $activeHelper
    try {
        & gofmt -w $activeHelper
        if ($LASTEXITCODE -ne 0) { throw 'gofmt failed' }
        Copy-Item -LiteralPath $activeHelper -Destination (Join-Path $PSScriptRoot 'helper_test.go.txt') -Force
        & go vet ./src
        if ($LASTEXITCODE -ne 0) { throw 'go vet failed' }
        & go test -c -o $trialExecutable ./src
        if ($LASTEXITCODE -ne 0) { throw 'trial compilation failed' }
        & go build -o $gameExecutable ./src
        if ($LASTEXITCODE -ne 0) { throw 'game compilation failed' }
    } finally {
        Remove-Item -LiteralPath $activeHelper
    }
    if ($PrepareOnly) {
        Write-Output "Prepared trial executable: $trialExecutable"
        return
    }
    $runOutput = Join-Path $PSScriptRoot ('run-' + [DateTime]::UtcNow.ToString('yyyyMMdd-HHmmssfff'))
    New-Item -ItemType Directory -Path $runOutput | Out-Null
    $env:GHOST_TRIAL_OUTPUT = $runOutput
    $env:GHOST_TRIAL_SECONDS = $Seconds.ToString([Globalization.CultureInfo]::InvariantCulture)
    $env:GHOST_TRIAL_SAMPLING_SEED = $SamplingSeed.ToString([Globalization.CultureInfo]::InvariantCulture)
    Write-Output "Output: $runOutput"
    & $trialExecutable -test.run '^TestDungeonTrials$' -test.timeout (($Seconds + 15).ToString([Globalization.CultureInfo]::InvariantCulture) + 's') -test.v
    if ($LASTEXITCODE -ne 0) { throw 'trial process failed' }
} finally {
    Pop-Location
}


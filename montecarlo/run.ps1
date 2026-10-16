param(
    [switch]$PrepareOnly,
    [ValidateRange(0.01, 86400)]
    [double]$Seconds = 60,
    [long]$SamplingSeed = 20261016
)
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$buildRoot = Join-Path $PSScriptRoot 'bin'
$tempRoot = Join-Path $PSScriptRoot 'tmp'
$dataRoot = Join-Path $PSScriptRoot 'data'
$templatePath = Join-Path $PSScriptRoot 'helper_test.go.txt'
$virtualPath = Join-Path $repoRoot 'src/montecarlo_experiment_test.go'
$overlayPath = Join-Path $tempRoot 'overlay.json'
$exeSuffix = ''
if ([Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT) { $exeSuffix = '.exe' }
$trialExecutable = Join-Path $buildRoot ('trials' + $exeSuffix)
$gameExecutable = Join-Path $buildRoot ('ghostdog-verification' + $exeSuffix)
$env:TEMP = Join-Path $tempRoot 'temp'
$env:TMP = $env:TEMP
$env:TMPDIR = $env:TEMP
$env:GOCACHE = Join-Path $tempRoot 'go-cache'
New-Item -ItemType Directory -Path $buildRoot, $dataRoot, $env:TEMP -Force | Out-Null
if (Test-Path -LiteralPath $virtualPath) {
    throw "The overlay target already exists: $virtualPath"
}
$utf8 = [System.Text.UTF8Encoding]::new($false)
$replacementMap = @{}
$replacementMap[$virtualPath] = $templatePath
$overlay = @{ Replace = $replacementMap } | ConvertTo-Json -Depth 3
[System.IO.File]::WriteAllText($overlayPath, $overlay, $utf8)
Push-Location $repoRoot
try {
    & gofmt -w $templatePath
    if ($LASTEXITCODE -ne 0) { throw 'gofmt failed' }
    & go vet "-overlay=$overlayPath" ./src
    if ($LASTEXITCODE -ne 0) { throw 'go vet failed' }
    & go test "-overlay=$overlayPath" -c -o $trialExecutable ./src
    if ($LASTEXITCODE -ne 0) { throw 'trial compilation failed' }
    & go build -o $gameExecutable ./src
    if ($LASTEXITCODE -ne 0) { throw 'game compilation failed' }
    & $trialExecutable '-test.run=^TestDungeonTrialShrineEligibility$' '-test.timeout=15s' '-test.v'
    if ($LASTEXITCODE -ne 0) { throw 'policy regression checks failed' }
    if ($PrepareOnly) {
        Write-Output "Prepared trial executable: $trialExecutable"
        return
    }
    $runOutput = Join-Path $dataRoot ('run-' + [DateTime]::UtcNow.ToString('yyyyMMdd-HHmmssfff'))
    New-Item -ItemType Directory -Path $runOutput | Out-Null
    Copy-Item -LiteralPath $templatePath -Destination (Join-Path $runOutput 'helper_test.go.txt')
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'run.ps1') -Destination (Join-Path $runOutput 'run.ps1')
    $sourceHashes = [ordered]@{}
    Get-ChildItem -LiteralPath (Join-Path $repoRoot 'src') -Filter '*.go' |
        Where-Object { $_.Name -notlike '*_test.go' } |
        Sort-Object Name |
        ForEach-Object { $sourceHashes[$_.Name] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash }
    $manifest = [ordered]@{
        go_version = (& go version | Out-String).Trim()
        helper_sha256 = (Get-FileHash -LiteralPath $templatePath -Algorithm SHA256).Hash
        source_sha256 = $sourceHashes
        seconds = $Seconds
        sampling_seed = $SamplingSeed
    } | ConvertTo-Json -Depth 4
    [System.IO.File]::WriteAllText((Join-Path $runOutput 'manifest.json'), $manifest, $utf8)
    $env:GHOST_TRIAL_OUTPUT = $runOutput
    $env:GHOST_TRIAL_SECONDS = $Seconds.ToString([Globalization.CultureInfo]::InvariantCulture)
    $env:GHOST_TRIAL_SAMPLING_SEED = $SamplingSeed.ToString([Globalization.CultureInfo]::InvariantCulture)
    Write-Output "Output: $runOutput"
    $testTimeout = ($Seconds + 15).ToString([Globalization.CultureInfo]::InvariantCulture) + 's'
    & $trialExecutable '-test.run=^TestDungeonTrials$' "-test.timeout=$testTimeout" '-test.v'
    if ($LASTEXITCODE -ne 0) { throw 'trial process failed' }
} finally {
    Pop-Location
}


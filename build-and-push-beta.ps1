<#
.SYNOPSIS
Build multi-arch Docker image of the project and push to predheadtx/uisp-noc:beta

.SYNTAX
./build-and-push-beta.ps1 [-Image <string>] [-Push] [-Load] [-NoCache] [-Pull]

.DESCRIPTION
This script ensures a Docker buildx builder is available then builds the image for linux/amd64,linux/arm64,linux/arm/v7 and pushes it to the Docker repo.
Defaults to pushing to predheadtx/uisp-noc:beta.

By default the script will push (--push). Use -Load to load the image into the local docker instead of pushing (useful for local testing on one platform). Do not specify both -Push and -Load.
#>
param(
    [string]$Image = 'predheadtx/uisp-noc:beta',
    [switch]$Load = $false,
    [switch]$NoCache = $false,
    [switch]$Pull = $false,
    [switch]$Push
)

# If -Push is not provided, default to true
if (-not $PSBoundParameters.ContainsKey('Push')) { $Push = $true }

function Write-ErrorAndExit {
    param([string]$msg, [int]$code = 1)
    Write-Host $msg -ForegroundColor Red
    exit $code
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-ErrorAndExit "Docker CLI not found in PATH. Please install Docker Desktop or Docker Engine and ensure 'docker' is available."
}

if ($Load -and $Push) {
    Write-Host "Both -Load and -Push were provided; prefer only one. Defaulting to -Load taking precedence." -ForegroundColor Yellow
    $Push = $false
}

# Ensure buildx builder exists and is in use
$builderName = 'multiarch-builder'
try {
    $existing = docker buildx ls 2>&1 | Out-String
    if ($existing -notmatch [regex]::Escape($builderName)) {
        Write-Host "Creating buildx builder '$builderName' and setting it to use..."
        docker buildx create --name $builderName --use | Out-Host
    } else {
        Write-Host "Using existing buildx builder '$builderName'. Ensuring it's active..."
        docker buildx use $builderName | Out-Host
    }
} catch {
    Write-ErrorAndExit "Failed to ensure buildx builder: $_"
}

# Platforms
$platforms = 'linux/amd64,linux/arm64,linux/arm/v7'

# Build flags
$flags = @()
if ($NoCache) { $flags += '--no-cache' }
if ($Pull) { $flags += '--pull' }

# Choose push vs load
if ($Load) {
    $flags += '--load'
} elseif ($Push) {
    $flags += '--push'
}

$flagsStr = if ($flags.Count -gt 0) { $flags -join ' ' } else { '' }

Write-Host "Building image: $Image for platforms: $platforms"
if ($flagsStr -ne '') { Write-Host "Using flags: $flagsStr" }

# Run build
try {
    $cmd = "docker buildx build --platform $platforms -t $Image $flagsStr ."
    Write-Host "Running: $cmd"
    Invoke-Expression $cmd
} catch {
    Write-ErrorAndExit "docker buildx build failed: $_"
}

Write-Host "Build finished." -ForegroundColor Green

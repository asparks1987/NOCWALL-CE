<#
.SYNOPSIS
Multi-arch Docker build script (PowerShell)

.SYNTAX
./build-multiarch.ps1 <image>

.DESCRIPTION
Equivalent to build-multiarch.sh but implemented for PowerShell. It ensures a Docker buildx builder exists and runs a multi-platform build for linux/amd64, linux/arm64 and linux/arm/v7 and pushes the image.
#>
param(
    [Parameter(Mandatory=$true, Position=0)]
    [string]$Image
)

function Write-ErrorAndExit {
    param([string]$msg, [int]$code = 1)
    Write-Host $msg -ForegroundColor Red
    exit $code
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-ErrorAndExit "Docker CLI not found in PATH. Please install Docker Desktop or Docker Engine and ensure 'docker' is available."
}

# Ensure buildx builder exists and is in use
$builderName = 'multiarch-builder'
$useBuilder = $false
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

# Run buildx build for multiple platforms and push
$platforms = 'linux/amd64,linux/arm64,linux/arm/v7'
Write-Host "Building and pushing multi-arch image: $Image for platforms: $platforms"
try {
    docker buildx build --platform $platforms -t $Image --push . | Out-Host
} catch {
    Write-ErrorAndExit "docker buildx build failed: $_"
}

Write-Host "Build finished." -ForegroundColor Green

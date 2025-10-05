<#
.SYNOPSIS
Build and push the UISP NOC Docker image for multiple architectures using Docker Buildx.

.DESCRIPTION
Creates (if necessary) a reusable Buildx builder, builds the container image for the
specified platforms, and pushes it to Docker Hub (or loads it locally when -Load is used).

.PARAMETER Image
Target Docker image name (including tag), e.g. "predheadtx/uisp-noc:latest".

.PARAMETER Context
Docker build context directory. Defaults to the current directory.

.PARAMETER Platforms
Comma-separated platform list passed to docker buildx --platform. Defaults to
"linux/amd64,linux/arm64,linux/arm/v7".

.PARAMETER Builder
Optional name for the buildx builder to use or create. Defaults to "uisp-noc-multiarch".

.PARAMETER Push
When provided, push the image to the registry (default behaviour if neither -Push nor -Load supplied).

.PARAMETER Load
Load the image into the local Docker daemon instead of pushing (mutually exclusive with -Push).

.PARAMETER Pull
Pass --pull to docker buildx build to always attempt to fetch newer base images.

.PARAMETER NoCache
Pass --no-cache to docker buildx build.

.PARAMETER BuildArg
Additional build arguments in KEY=VALUE form. May be specified multiple times.

.EXAMPLE
./build-multiarch.ps1 -Image predheadtx/uisp-noc:latest -BuildArg BUILD_SHA=git-sha-placeholder
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Image,

    [string]$Context = ".",

    [string]$Platforms = "linux/amd64,linux/arm64,linux/arm/v7",

    [string]$Builder = "uisp-noc-multiarch",

    [switch]$Push,

    [switch]$Load,

    [switch]$Pull,

    [switch]$NoCache,

    [string[]]$BuildArg
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not $PSBoundParameters.ContainsKey('Push') -and -not $Load.IsPresent) {
    $Push = $true
}

if ($Push -and $Load) {
    throw "-Push and -Load are mutually exclusive."
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker CLI not found in PATH. Please install Docker Desktop/Engine."
}

try {
    docker info | Out-Null
} catch {
    throw "Unable to communicate with the Docker daemon. Ensure Docker is running."
}

Write-Host "Using builder: $Builder"
$builderExists = $false
try {
    docker buildx inspect $Builder | Out-Null
    $builderExists = $true
} catch {
    $builderExists = $false
}

if (-not $builderExists) {
    Write-Host "Creating buildx builder '$Builder'"
    docker buildx create --name $Builder --use | Out-Null
} else {
    docker buildx use $Builder | Out-Null
}

docker buildx inspect --bootstrap | Out-Null

$cmd = @('docker','buildx','build','--platform', $Platforms,'-t', $Image)

if ($Push) {
    $cmd += '--push'
} elseif ($Load) {
    $cmd += '--load'
}

if ($Pull) {
    $cmd += '--pull'
}

if ($NoCache) {
    $cmd += '--no-cache'
}

if ($BuildArg) {
    foreach ($arg in $BuildArg) {
        $cmd += @('--build-arg', $arg)
    }
}

$cmd += $Context

Write-Host "Running: " + ($cmd -join ' ') -ForegroundColor Cyan

try {
    & $cmd[0] @($cmd[1..($cmd.Count-1)])
} catch {
    throw ("docker buildx build failed: {0}" -f $_.Exception.Message)
}

Write-Host "Multi-arch build completed successfully." -ForegroundColor Green


#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$Repo = "Ray-D-Song/goread"
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "goread\bin" }

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$filename = "goread-windows-$arch.exe"

$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$version = $release.tag_name
$url = "https://github.com/$Repo/releases/download/$version/$filename"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Write-Host "Downloading $url ..."
Invoke-WebRequest -Uri $url -OutFile (Join-Path $InstallDir "goread.exe")

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$InstallDir*") {
	$newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
	[Environment]::SetEnvironmentVariable("Path", $newPath, "User")
	$env:Path = "$env:Path;$InstallDir"
	Write-Host "Added $InstallDir to your user PATH."
}

Write-Host "Installed goread $version to $(Join-Path $InstallDir 'goread.exe')"
Write-Host "Restart your terminal to use goread."
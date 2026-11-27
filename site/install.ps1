# Weft installer: irm https://weft.base.al/install.ps1 | iex
#
# Environment:
#   WEFT_BASE     where releases live (default https://weft.base.al)
#   WEFT_VERSION  version to install (default: the latest release)
#   WEFT_BIN_DIR  where to put the binary (default $HOME\.local\bin)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$base = if ($env:WEFT_BASE) { $env:WEFT_BASE.TrimEnd('/') } else { 'https://weft.base.al' }
$binDir = if ($env:WEFT_BIN_DIR) { $env:WEFT_BIN_DIR } else { Join-Path $HOME '.local\bin' }

function Fail($msg) {
	[Console]::Error.WriteLine("weft install: $msg")
	exit 1
}

function Get-WeftArch($raw) {
	switch ($raw.Trim().ToLowerInvariant()) {
		{ $_ -in 'amd64', 'x86_64', 'x64' } { return 'amd64' }
		{ $_ -in 'arm64', 'aarch64' } { return 'arm64' }
		default { Fail "no build for windows/$_" }
	}
}

function Get-WeftChecksum($sumsText, $name) {
	foreach ($line in ($sumsText -split "`r?`n")) {
		$fields = ($line.Trim() -split '\s+')
		if ($fields.Count -eq 2 -and $fields[1].TrimStart('*') -eq $name) {
			return $fields[0].ToLowerInvariant()
		}
	}
	return ''
}

$rawArch = if ($env:WEFT_TEST_ARCH) {
	$env:WEFT_TEST_ARCH
} elseif ($env:PROCESSOR_ARCHITEW6432) {
	$env:PROCESSOR_ARCHITEW6432
} elseif ($env:PROCESSOR_ARCHITECTURE) {
	$env:PROCESSOR_ARCHITECTURE
} else {
	[System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
}
$arch = Get-WeftArch $rawArch

$version = if ($env:WEFT_VERSION) { $env:WEFT_VERSION.Trim() } else { '' }
if (-not $version) {
	try {
		$version = (Invoke-WebRequest -Uri "$base/releases/latest" -UseBasicParsing).Content
		if ($version -is [byte[]]) {
			$version = [System.Text.Encoding]::UTF8.GetString($version)
		}
		$version = ([string]$version).Trim()
	} catch {
		Fail "could not reach $base to find the latest version"
	}
	if (-not $version) {
		Fail "could not reach $base to find the latest version"
	}
}

$name = "weft_windows_$arch.zip"
$url = "$base/releases/$version"
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("weft-install-" + [System.Guid]::NewGuid().ToString('N'))
try {
	$null = New-Item -ItemType Directory -Path $tmp -Force
} catch {
	Fail 'could not create a temporary directory'
}

try {
	$zipPath = Join-Path $tmp $name
	$sumsPath = Join-Path $tmp 'SHA256SUMS'
	try {
		Invoke-WebRequest -Uri "$url/$name" -OutFile $zipPath -UseBasicParsing
	} catch {
		Fail "could not download $url/$name"
	}
	try {
		Invoke-WebRequest -Uri "$url/SHA256SUMS" -OutFile $sumsPath -UseBasicParsing
	} catch {
		Fail "could not download $url/SHA256SUMS"
	}

	$want = Get-WeftChecksum (Get-Content -Path $sumsPath -Raw) $name
	$got = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
	if (-not $want -or $want -ne $got) {
		Fail "checksum mismatch for $name"
	}

	try {
		Expand-Archive -Path $zipPath -DestinationPath $tmp -Force
	} catch {
		Fail "could not unpack $name"
	}
	$extracted = Join-Path $tmp 'weft.exe'
	if (-not (Test-Path -LiteralPath $extracted)) {
		Fail "could not unpack $name"
	}

	try {
		$null = New-Item -ItemType Directory -Path $binDir -Force
	} catch {
		Fail "could not create $binDir"
	}

	$newExe = Join-Path $binDir '.weft.new.exe'
	$target = Join-Path $binDir 'weft.exe'
	$oldExe = Join-Path $binDir 'weft.exe.old'
	try {
		Copy-Item -LiteralPath $extracted -Destination $newExe -Force
	} catch {
		Remove-Item -LiteralPath $newExe -Force -ErrorAction SilentlyContinue
		Fail "could not write to $binDir"
	}
	# Rename beside the target: an old weft.exe stays whole until the new one takes its place,
	# and a running weft.exe on Windows can be renamed aside to weft.exe.old (cleaned up on weft's next run).
	$movedOld = $false
	if (Test-Path -LiteralPath $target) {
		try {
			Move-Item -LiteralPath $target -Destination $oldExe -Force
			$movedOld = $true
		} catch {
			Remove-Item -LiteralPath $newExe -Force -ErrorAction SilentlyContinue
			Fail "could not write to $binDir"
		}
	}
	try {
		Move-Item -LiteralPath $newExe -Destination $target -Force
	} catch {
		if ($movedOld -and (Test-Path -LiteralPath $oldExe)) {
			Move-Item -LiteralPath $oldExe -Destination $target -Force -ErrorAction SilentlyContinue
		}
		Remove-Item -LiteralPath $newExe -Force -ErrorAction SilentlyContinue
		Fail "could not write to $binDir"
	}
} finally {
	Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

[Console]::Out.WriteLine("Installed weft $version to $target")

function Test-InPath($pathList, $dir) {
	$norm = $dir.TrimEnd('\', '/')
	foreach ($entry in ($pathList -split ';')) {
		if ($entry.Trim().TrimEnd('\', '/') -ieq $norm) {
			return $true
		}
	}
	return $false
}

$inSession = Test-InPath $env:PATH $binDir
if ($env:WEFT_NO_MODIFY_PATH -eq '1') {
	if (-not $inSession) {
		[Console]::Out.WriteLine("Add $binDir to your PATH to run weft.")
	}
} else {
	$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
	if (-not (Test-InPath $userPath $binDir)) {
		$next = if ($userPath) { "$userPath;$binDir" } else { $binDir }
		[Environment]::SetEnvironmentVariable('Path', $next, 'User')
		if (-not $inSession) {
			$env:PATH = "$env:PATH;$binDir"
		}
		[Console]::Out.WriteLine("Added $binDir to your PATH.")
	} elseif (-not $inSession) {
		$env:PATH = "$env:PATH;$binDir"
	}
}

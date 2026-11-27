# Weft installer: irm https://weft.base.al/install.ps1 | iex
#
# Environment:
#   WEFT_BASE     where releases live (default https://weft.base.al)
#   WEFT_VERSION  version to install (default: the latest release)
#   WEFT_BIN_DIR  where to put the binary (default $HOME\.local\bin)
try {
	& {
		$ErrorActionPreference = 'Stop'
		$ProgressPreference = 'SilentlyContinue'
		if ($PSVersionTable.PSVersion.Major -lt 6) {
			[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
		}

		function Fail($msg) {
			throw "weft install: $msg"
		}

		function Get-WeftArch($raw) {
			switch ($raw.Trim().ToLowerInvariant()) {
				{ $_ -in 'amd64', 'x86_64', 'x64' } { return 'amd64' }
				{ $_ -in 'arm64', 'aarch64' } { return 'arm64' }
				default { Fail "no build for windows/$_; Weft supports darwin, linux and windows on amd64 and arm64" }
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

		function Test-InPath($pathList, $dir, $expand) {
			$norm = $dir.TrimEnd('\', '/')
			foreach ($entry in ($pathList -split ';')) {
				$e = $entry.Trim()
				if (-not $e) { continue }
				if ($expand) {
					$e = [Environment]::ExpandEnvironmentVariables($e)
				}
				if ($e.TrimEnd('\', '/') -ieq $norm) {
					return $true
				}
			}
			return $false
		}

		$base = if ($env:WEFT_BASE) { $env:WEFT_BASE.TrimEnd('/') } else { 'https://weft.base.al' }
		$binDir = if ($env:WEFT_BIN_DIR) { $env:WEFT_BIN_DIR } else { Join-Path $HOME '.local\bin' }
		$binDir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($binDir)

		$rawArch = if ($env:WEFT_TEST_ARCH) {
			$env:WEFT_TEST_ARCH
		} elseif (-not [Environment]::Is64BitProcess -and $env:PROCESSOR_ARCHITEW6432) {
			$env:PROCESSOR_ARCHITEW6432
		} else {
			try {
				[System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
			} catch {
				if ($env:PROCESSOR_ARCHITEW6432) {
					$env:PROCESSOR_ARCHITEW6432
				} else {
					[string]$env:PROCESSOR_ARCHITECTURE
				}
			}
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
				Fail "could not reach $base to find the latest version; check your connection, or set WEFT_VERSION"
			}
			if (-not $version) {
				Fail "could not reach $base to find the latest version; check your connection, or set WEFT_VERSION"
			}
		}

		$name = "weft_windows_$arch.zip"
		$url = "$base/releases/$version"
		$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("weft-install-" + [System.Guid]::NewGuid().ToString('N'))
		try {
			$null = New-Item -ItemType Directory -Path $tmp -Force
		} catch {
			Fail 'could not create a temporary directory; check that TEMP is writable'
		}

		try {
			$zipPath = Join-Path $tmp $name
			$sumsPath = Join-Path $tmp 'SHA256SUMS'
			try {
				Invoke-WebRequest -Uri "$url/$name" -OutFile $zipPath -UseBasicParsing
			} catch {
				Fail "could not download $url/$name; check your connection and that release $version exists"
			}
			try {
				Invoke-WebRequest -Uri "$url/SHA256SUMS" -OutFile $sumsPath -UseBasicParsing
			} catch {
				Fail "could not download $url/SHA256SUMS; check your connection and that release $version exists"
			}

			$want = Get-WeftChecksum (Get-Content -Path $sumsPath -Raw) $name
			$got = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
			if (-not $want -or $want -ne $got) {
				Fail "checksum mismatch for $name; run the installer again"
			}

			try {
				Expand-Archive -Path $zipPath -DestinationPath $tmp -Force
			} catch {
				Fail "could not unpack $name; run the installer again"
			}
			$extracted = Join-Path $tmp 'weft.exe'
			if (-not (Test-Path -LiteralPath $extracted)) {
				Fail "could not unpack $name; run the installer again"
			}

			try {
				$null = New-Item -ItemType Directory -Path $binDir -Force
			} catch {
				Fail "could not create $binDir; set WEFT_BIN_DIR to a writable folder"
			}

			$newExe = Join-Path $binDir '.weft.new.exe'
			$target = Join-Path $binDir 'weft.exe'
			$oldExe = Join-Path $binDir 'weft.exe.old'
			try {
				Copy-Item -LiteralPath $extracted -Destination $newExe -Force
			} catch {
				Remove-Item -LiteralPath $newExe -Force -ErrorAction SilentlyContinue
				Fail "could not write to $binDir; set WEFT_BIN_DIR to a writable folder"
			}
			# Rename beside the target: an old weft.exe stays whole until the new one takes its place,
			# and a running weft.exe on Windows can be renamed aside to weft.exe.old.
			$movedOld = $false
			if (Test-Path -LiteralPath $target) {
				try {
					Move-Item -LiteralPath $target -Destination $oldExe -Force
					$movedOld = $true
				} catch {
					Remove-Item -LiteralPath $newExe -Force -ErrorAction SilentlyContinue
					Fail "could not write to $binDir; close any running weft and try again"
				}
			}
			try {
				Move-Item -LiteralPath $newExe -Destination $target -Force
			} catch {
				if ($movedOld -and (Test-Path -LiteralPath $oldExe)) {
					Move-Item -LiteralPath $oldExe -Destination $target -Force -ErrorAction SilentlyContinue
				}
				Remove-Item -LiteralPath $newExe -Force -ErrorAction SilentlyContinue
				Fail "could not write to $binDir; set WEFT_BIN_DIR to a writable folder"
			}
			if ($movedOld) {
				# Removes weft.exe.old when it was not running; if it is in use, weft's next run removes it.
				Remove-Item -LiteralPath $oldExe -Force -ErrorAction SilentlyContinue
			}
		} finally {
			Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
		}

		[Console]::Out.WriteLine("Installed weft $version to $target")

		$inSession = Test-InPath $env:PATH $binDir $false
		if ($env:WEFT_NO_MODIFY_PATH -eq '1') {
			if (-not $inSession) {
				[Console]::Out.WriteLine("Add $binDir to your PATH to run weft.")
			}
		} else {
			$pathName = if ($env:WEFT_TEST_PATH_NAME) { $env:WEFT_TEST_PATH_NAME } else { 'Path' }
			$key = Get-Item -LiteralPath 'HKCU:\Environment'
			$userPath = [string]$key.GetValue($pathName, '', 'DoNotExpandEnvironmentNames')
			if (-not (Test-InPath $userPath $binDir $true)) {
				$trimmed = $userPath.TrimEnd(';')
				$next = if ($trimmed) { "$trimmed;$binDir" } else { $binDir }
				Set-ItemProperty -LiteralPath 'HKCU:\Environment' -Name $pathName -Value $next -Type ExpandString
				# Tell new windows: SetEnvironmentVariable broadcasts WM_SETTINGCHANGE.
				[Environment]::SetEnvironmentVariable('WEFT_INSTALL_BROADCAST', '1', 'User')
				[Environment]::SetEnvironmentVariable('WEFT_INSTALL_BROADCAST', $null, 'User')
				if (-not $inSession) {
					$env:PATH = "$env:PATH;$binDir"
				}
				[Console]::Out.WriteLine("Added $binDir to your PATH (open a new terminal elsewhere to use it).")
			} elseif (-not $inSession) {
				$env:PATH = "$env:PATH;$binDir"
			}
		}
	}
} catch {
	[Console]::Error.WriteLine(($_.Exception.Message -replace '^(?!weft install: )', 'weft install: '))
	# Run as a file (the tests, powershell -File): exit 1. Under irm | iex, exit would close the window.
	if ($PSCommandPath) { exit 1 }
}

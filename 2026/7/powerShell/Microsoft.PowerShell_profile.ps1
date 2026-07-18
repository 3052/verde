# ────────────────────────────────────────────────────────────────────
# Section 1 — Array declarations (never expires)
# ────────────────────────────────────────────────────────────────────

$basePaths = @()

$androidPaths = @()

# ────────────────────────────────────────────────────────────────────
# Section 2 — Environment variables, array items & core behavior (can expire — dated)
# ────────────────────────────────────────────────────────────────────

# 2026-07-18
$basePaths += 'C:\MinGit\mingw64\bin'
# 2026-07-18
$basePaths += 'C:\Users\Steven\go\bin'
# 2026-07-18
$basePaths += 'C:\curl\bin'
# 2026-07-18
$basePaths += 'C:\fd'
# 2026-07-18
$basePaths += 'C:\gdu_windows_amd64'
# 2026-07-18
$basePaths += 'C:\go\bin'
# 2026-07-18
$basePaths += 'C:\less-x64'
# 2026-07-18
$basePaths += 'C:\mitmproxy'
# 2026-07-18
$basePaths += 'C:\neocities-deploy-Windows-x86_64'
# 2026-07-18
$basePaths += 'C:\rclone'
# 2026-07-18
$basePaths += 'C:\ripgrep'
# 2026-07-18
$basePaths += 'C:\staticcheck'
# 2026-07-18
$basePaths += 'C:\vim92'

# 2026-07-18
$androidPaths += 'C:\Program Files\Android\Android Studio\jbr\bin'
# 2026-07-18
$androidPaths += 'C:\Users\Steven\AppData\Local\Android\Sdk\build-tools\36.0.0'
# 2026-07-18
$androidPaths += 'C:\Users\Steven\AppData\Local\Android\Sdk\emulator'
# 2026-07-18
$androidPaths += 'C:\Users\Steven\AppData\Local\Android\Sdk\platform-tools'
# 2026-07-18
$androidPaths += 'C:\Users\Steven\go\bin'
# 2026-07-18
$androidPaths += 'C:\jadx\bin'

# 2026-07-18
$env:RIPGREP_CONFIG_PATH = 'C:\ripgrep\ripgrep.txt'

# 2026-07-18
$env:GOPROXY = 'direct'

# 2026-07-18
$env:GOSUMDB = 'off'

# 2026-07-18
$env:EDITOR = 'gvim'

# 2026-07-18
Remove-Item -Path Alias:* -Force

# 2026-07-18
Set-PSReadLineKeyHandler Ctrl+UpArrow {
   Set-Location ..
   [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
}

# 2026-07-18
function Set-PathAndroid {
   $env:PATH = $androidPaths -join ';'
   Write-Host 'PATH switched to Android/Python' -ForegroundColor Green
}

# ────────────────────────────────────────────────────────────────────
# Section 3 — Join (never expires)
# ────────────────────────────────────────────────────────────────────

$env:PATH = $basePaths -join ';'

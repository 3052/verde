# ────────────────────────────────────────────────────────────────────
# Section 1 — Array declarations (never expires)
# ────────────────────────────────────────────────────────────────────

$basePath = @()

$androidPath = @()

# ────────────────────────────────────────────────────────────────────
# Section 2 — Environment variables, array items & core behavior (can expire — dated)
# ────────────────────────────────────────────────────────────────────

# 2026-07-19
$androidPath += 'C:\Users\Steven\AppData\Local\Programs\Python\Python314'
$androidPath += 'C:\Users\Steven\AppData\Local\Programs\Python\Python314\Scripts'
$basePath += 'C:\Users\Steven\AppData\Local\Programs\Python\Python314'
$basePath += 'C:\Users\Steven\AppData\Local\Programs\Python\Python314\Scripts'
$basePath += 'C:\ffmpeg\bin'

# 2026-07-18
$basePath += 'C:\MinGit\mingw64\bin'
$basePath += 'C:\Users\Steven\go\bin'
$basePath += 'C:\curl\bin'
$basePath += 'C:\fd'
$basePath += 'C:\gdu_windows_amd64'
$basePath += 'C:\go\bin'
$basePath += 'C:\less-x64'
$basePath += 'C:\mitmproxy'
$basePath += 'C:\neocities-deploy-Windows-x86_64'
$basePath += 'C:\rclone'
$basePath += 'C:\ripgrep'
$basePath += 'C:\staticcheck'
$basePath += 'C:\vim92'

# 2026-07-18
$androidPath += 'C:\Program Files\Android\Android Studio\jbr\bin'
$androidPath += 'C:\Users\Steven\AppData\Local\Android\Sdk\build-tools\36.0.0'
$androidPath += 'C:\Users\Steven\AppData\Local\Android\Sdk\emulator'
$androidPath += 'C:\Users\Steven\AppData\Local\Android\Sdk\platform-tools'
$androidPath += 'C:\Users\Steven\go\bin'
$androidPath += 'C:\jadx\bin'

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
   $env:PATH = $androidPath -join ';'
   Write-Host 'PATH switched to Android/Python' -ForegroundColor Green
}

# ────────────────────────────────────────────────────────────────────
# Section 3 — Join (never expires)
# ────────────────────────────────────────────────────────────────────

$env:PATH = $basePath -join ';'

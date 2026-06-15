$path = @()
#################################################################################

# 2026-06-14
# go.dev/doc/go1.13
$env:GOPROXY = 'direct'
$env:GOSUMDB = 'off'

# 2026-06-09
$path += 'D:\vim'

# 2026-06-08
$path += 'C:\Users\Steven\AppData\Local\Programs\Python\Python314'
$path += 'C:\Users\Steven\AppData\Local\Programs\Python\Python314\Scripts'
$path += 'C:\Users\Steven\go\bin'
$path += 'D:\MinGit\mingw64\bin'
$path += 'D:\bin'
$path += 'D:\go\bin'

# 2025-11-29
$path += 'D:\Bento4\bin'

# 2025-11-04
$env:RIPGREP_CONFIG_PATH = "C:\Users\Steven\AppData\Local\ripgrep\ripgrep.txt"

# disable auto complete
Set-PSReadLineOption -PredictionSource None

$MaximumHistoryCount = 9999

Set-PSReadLineOption -AddToHistoryHandler $null

# git diff unicode
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()

Get-Alias | Remove-Alias -Force

Set-PSReadLineKeyHandler Ctrl+UpArrow {
   Set-Location ..
   [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
}

# git commit -v
$env:EDITOR = 'gvim'

#################################################################################
$env:path = $path -join ';'

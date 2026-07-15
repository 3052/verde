Set-PSReadLineKeyHandler Ctrl+UpArrow {
   Set-Location ..
   [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
}

$env:GOPROXY = 'direct'
$env:GOSUMDB = 'off'

$dirs = @(
   'C:\MinGit\mingw64\bin'
   'C:\Users\Steven\go\bin'
   'C:\fd'
   'C:\go\bin'
   'C:\less-x64'
   'C:\neocities-deploy-Windows-x86_64'
   'C:\ripgrep'
   'C:\staticcheck'
   'C:\vim92'
)

$env:PATH = $dirs -join ';'

# git commit -v
$env:EDITOR = 'gvim'

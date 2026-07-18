Remove-Item -Path Alias:* -Force 

Set-PSReadLineKeyHandler Ctrl+UpArrow {
   Set-Location ..
   [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
}

$env:GOPROXY = 'direct'
$env:GOSUMDB = 'off'

$dirs = @(
   'C:\MinGit\mingw64\bin'
   'C:\Users\Steven\go\bin'
   'C:\curl\bin'
   'C:\fd'
   'C:\go\bin'
   'C:\less-x64'
   'C:\mitmproxy'
   'C:\neocities-deploy-Windows-x86_64'
   'C:\rclone'
   'C:\ripgrep'
   'C:\staticcheck'
   'C:\vim92'
)

$env:PATH = $dirs -join ';'

# git commit -v
$env:EDITOR = 'gvim'

function Set-PathAndroid {
   $env:PATH = @(
      'C:\Program Files\Android\Android Studio\jbr\bin'
      'C:\Users\Steven\AppData\Local\Android\Sdk\build-tools\36.0.0'
      'C:\Users\Steven\AppData\Local\Android\Sdk\emulator'
      'C:\Users\Steven\AppData\Local\Android\Sdk\platform-tools'
      'C:\Users\Steven\go\bin'
      'C:\jadx\bin'
   ) -join ';'
   Write-Host "PATH switched to Android/Python" -ForegroundColor Green
}

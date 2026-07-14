$dirs = @(
   'C:\MinGit\mingw64\bin'
   'C:\Users\Steven\go\bin'
   'C:\fd'
   'C:\go\bin'
   'C:\vim92'
)

$env:PATH = $dirs -join ';'

# git commit -v
$env:EDITOR = 'gvim'


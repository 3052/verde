# vim

Might need to delete this:

~~~
HKEY_CLASSES_ROOT\Applications\gvim.exe
~~~

https://github.com/vim/vim-win32-installer/releases

## syntax/go.vim

~~~
diff --git a/syntax/go.vim b/syntax/go.vim
index 9af11ac..8801bbd 100644
--- a/syntax/go.vim
+++ b/syntax/go.vim
@@ -88,0 +89 @@ syn keyword     goBuiltins          make new panic print println
+syn keyword     goBuiltins          max min
~~~

https://github.com/google/vim-ft-go/blob/master/syntax/go.vim

## syntax/markdown.vim

~~~diff
diff --git a/syntax/markdown.vim b/syntax/markdown.vim
index a069746..4afea16 100644
--- a/syntax/markdown.vim
+++ b/syntax/markdown.vim
@@ -190,0 +191,2 @@ hi def link markdownCodeDelimiter         Delimiter
+hi def link markdownCode                  String
+hi def link markdownCodeBlock             String
~~~

https://github.com/tpope/vim-markdown/tree/master/syntax

## syntax/zig.vim

~~~diff
diff --git a/syntax/zig.vim b/syntax/zig.vim
index 80df1f8..a8ffc32 100644
--- a/syntax/zig.vim
+++ b/syntax/zig.vim
@@ -261 +261 @@ highlight default link zigBuiltinFn Statement
-highlight default link zigKeyword Keyword
+highlight default link zigKeyword Structure
@@ -292 +292 @@ highlight default link zigSpecial Special
-highlight default link zigVarDecl Function
+highlight default link zigVarDecl Structure
~~~

https://github.com/ziglang/zig.vim

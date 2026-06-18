# vim

~~~reg
Windows Registry Editor Version 5.00
[HKEY_CLASSES_ROOT\Unknown\shell\Gvim\command]
@="C:\\vim\\gvim \"%1\""
~~~

Might need to delete this:

~~~
HKEY_CLASSES_ROOT\Applications\gvim.exe
~~~

https://github.com/vim/vim-win32-installer/releases

## colors/PaperColor.vim

~~~diff
diff --git a/colors/PaperColor.vim b/colors/PaperColor.vim
index afef1d4..31f97a4 100755
--- a/colors/PaperColor.vim
+++ b/colors/PaperColor.vim
@@ -1230 +1230 @@ fun! s:apply_syntax_highlightings()
-  exec 'hi Exception' . s:fg_red
+  exec 'hi Exception' . s:fg_red . s:ft_bold
@@ -1238 +1238 @@ fun! s:apply_syntax_highlightings()
-  exec 'hi Type' . s:fg_pink . s:ft_bold
+  exec 'hi Type' . s:fg_positive . s:ft_bold
@@ -1253 +1253 @@ fun! s:apply_syntax_highlightings()
-  exec 'hi Title' . s:fg_comment
+  exec 'hi Title' . s:fg_foreground
~~~

https://github.com/NLKNguyen/papercolor-theme/tree/master/colors

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

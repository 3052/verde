if exists("b:current_syntax")
  finish
endif

" --- Emphasis -----------------------------------------------------------------

syn region markdownBold          matchgroup=markdownBoldDelimiter          start=/\*\*/  end=/\*\*/  keepend oneline
syn region markdownBold          matchgroup=markdownBoldDelimiter          start=/__/    end=/__/    keepend oneline
syn region markdownItalic        matchgroup=markdownItalicDelimiter        start=/\*/    end=/\*/    keepend oneline
syn region markdownItalic        matchgroup=markdownItalicDelimiter        start=/_/     end=/_/     keepend oneline
syn region markdownBoldItalic    matchgroup=markdownBoldItalicDelimiter    start=/\*\*\*/ end=/\*\*\*/ keepend oneline
syn region markdownBoldItalic    matchgroup=markdownBoldItalicDelimiter    start=/___/   end=/___/   keepend oneline
syn region markdownStrike        matchgroup=markdownStrikeDelimiter        start=/\~\~/  end=/\~\~/  keepend oneline

" --- Inline code --------------------------------------------------------------

syn region markdownCode          matchgroup=markdownCodeDelimiter start=/`/  end=/`/  keepend oneline contains=@NoSpell
syn region markdownCode          matchgroup=markdownCodeDelimiter start=/``/ skip=/[^`]`[^`]/ end=/``/ keepend oneline contains=@NoSpell

" --- Fenced code blocks -------------------------------------------------------

syn region markdownFencedCodeBlock matchgroup=markdownCodeDelimiter start=/\n\?\s*\zs```/ end=/^\s*```/ keepend contains=@NoSpell
syn region markdownFencedCodeBlock matchgroup=markdownCodeDelimiter start=/\n\?\s*\zs\~\~\~/ end=/^\s*\~\~\~/ keepend contains=@NoSpell

" --- Headings -----------------------------------------------------------------

syn match markdownH1 /^.\+\n=+\s*$/ contains=@Spell
syn match markdownH2 /^.\+\n-+\s*$/ contains=@Spell
syn match markdownH1 /^#\s.\+$/ contains=@Spell
syn match markdownH2 /^##\s.\+$/ contains=@Spell
syn match markdownH3 /^###\s.\+$/ contains=@Spell
syn match markdownH4 /^####\s.\+$/ contains=@Spell
syn match markdownH5 /^#####\s.\+$/ contains=@Spell
syn match markdownH6 /^######\s.\+$/ contains=@Spell

" --- Blockquotes --------------------------------------------------------------

syn match markdownBlockquote /^\s*>\s.*$/ contains=@Spell

" --- Lists --------------------------------------------------------------------

syn match markdownListItem /^\s*[-*+]\s\+/ contained
syn match markdownListItem /^\s*\d\+\.\s\+/ contained
syn match markdownListMarker /^\s*[-*+]\s\+/
syn match markdownListMarker /^\s*\d\+\.\s\+/
syn match markdownTaskListTodo /^\s*[-*+]\s\+\[\s\]/
syn match markdownTaskListDone /^\s*[-*+]\s\+\[[xX]\]/

" --- Horizontal rules ---------------------------------------------------------

syn match markdownRule /^\s*\*\s*\*\s*\*\s*\**\s*$/
syn match markdownRule /^\s*-\s*-\s*-\s*-*\s*$/
syn match markdownRule /^\s*_\s*_\s*_\s*_*\s*$/

" --- Links --------------------------------------------------------------------

" [text](url)
syn region markdownLink matchgroup=markdownLinkDelimiter start=/\[/ end=/\]/ nextgroup=markdownUrl skipwhite oneline contains=@Spell
syn region markdownUrl matchgroup=markdownLinkDelimiter start=/(/ end=/)/ contained oneline
syn match markdownUrlAuto /<\(https\?:\/\/[^>]\+\)>/
syn match markdownUrlAuto /\(https\?:\/\/\)\?\w\+\(\.\w\+\)\+\(\/\S*\)\?/

" Reference link definitions: [id]: url "title"
syn match markdownLinkDef /^\s*\[^]\+]:\s*\S.*/ contains=markdownLinkText,markdownLinkDefUrl,markdownLinkTitle
syn match markdownLinkText /\[[^]]\+\]/ contained
syn match markdownLinkDefUrl /\S\+/ contained
syn region markdownLinkTitle start=/"/ end=/"/ contained oneline

" --- Tables -------------------------------------------------------------------

syn match markdownTableSeparator /\(^\s*|.*\n\)\@<=|[: -]*-[[: -]*|]*[: -]*-[[: -]*|]*|[: -]*-[[: -]*|]*/

" --- Highlight links ----------------------------------------------------------

hi def link markdownBold                Bold
hi def link markdownBoldDelimiter       Type
hi def link markdownItalic              Italic
hi def link markdownItalicDelimiter     Type
hi def link markdownBoldItalic          BoldItalic
hi def link markdownBoldItalicDelimiter Type
hi def link markdownStrike              Special
hi def link markdownStrikeDelimiter     Delimiter

hi def link markdownCode                String
hi def link markdownCodeDelimiter       Delimiter
hi def link markdownFencedCodeBlock     String

hi def link markdownH1                  Title
hi def link markdownH2                  Title
hi def link markdownH3                  Title
hi def link markdownH4                  Title
hi def link markdownH5                  Title
hi def link markdownH6                  Title

hi def link markdownBlockquote          Comment

hi def link markdownListMarker          Keyword
hi def link markdownListItem            Keyword
hi def link markdownTaskListTodo        Todo
hi def link markdownTaskListDone        Comment

hi def link markdownRule                Special

hi def link markdownLink                Underlined
hi def link markdownLinkDelimiter       Delimiter
hi def link markdownUrl                 Underlined
hi def link markdownUrlAuto             Underlined
hi def link markdownLinkText            Underlined
hi def link markdownLinkDef             Comment
hi def link markdownLinkDefUrl          Underlined
hi def link markdownLinkTitle           String

hi def link markdownTableSeparator      Delimiter

let b:current_syntax = "markdown"

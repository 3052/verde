" A light Vim colorscheme built on Google's Material Design color palette.
" https://material.io/design/color/the-color-system.html

if exists('g:loaded_material')
  finish
endif
let g:loaded_material = 1

" --- Material Design palette -------------------------------------------------

let s:grey_100 = '#f5f5f5'
let s:grey_200 = '#eeeeee'
let s:grey_300 = '#e0e0e0'
let s:grey_400 = '#bdbdbd'
let s:grey_500 = '#9e9e9e'
let s:grey_600 = '#757575'
let s:grey_700 = '#616161'
let s:grey_800 = '#424242'
let s:grey_900 = '#212121'

let s:bg_100 = '#cfd8dc'
let s:bg_200 = '#b0bec5'
let s:bg_400 = '#78909c'
let s:bg_500 = '#607d8b'

let s:red_300  = '#e57373'
let s:red_500  = '#f44336'
let s:red_700  = '#d32f2f'
let s:red_900  = '#b71c1c'
let s:red_a200 = '#ff5252'

let s:pink_300 = '#f06292'
let s:pink_500 = '#e91e63'

let s:purple_300 = '#ba68c8'
let s:purple_500 = '#9c27b0'

let s:blue_300 = '#64b5f6'
let s:blue_500 = '#2196f3'
let s:blue_700 = '#1976d2'

let s:cyan_300 = '#4dd0e1'
let s:cyan_500 = '#00bcd4'
let s:cyan_700 = '#0097a7'

let s:green_300 = '#81c784'
let s:green_500 = '#4caf50'
let s:green_700 = '#388e3c'

let s:yellow_300 = '#fff176'
let s:yellow_500 = '#ffeb3b'
let s:yellow_700 = '#fbc02d'

let s:amber_300 = '#ffd54f'
let s:amber_500 = '#ffc107'
let s:amber_700 = '#ffa000'

let s:orange_300 = '#ffb74d'
let s:orange_500 = '#ff9800'
let s:orange_700 = '#f57c00'

let s:dorange_300 = '#ff8a65'
let s:dorange_500 = '#ff5722'
let s:dorange_700 = '#e64a19'

let s:white = '#ffffff'

" --- 256-color fallback ------------------------------------------------------

let s:ct = {}
let s:ct.white      = 15
let s:ct.grey100    = 255
let s:ct.grey200    = 255
let s:ct.grey300    = 252
let s:ct.grey400    = 249
let s:ct.grey500    = 245
let s:ct.grey600    = 243
let s:ct.grey700    = 240
let s:ct.grey800    = 235
let s:ct.grey900    = 233
let s:ct.bg400      = 66
let s:ct.bg500      = 66
let s:ct.red        = 203
let s:ct.red_dark   = 167
let s:ct.pink       = 204
let s:ct.purple     = 170
let s:ct.blue       = 39
let s:ct.blue_dark  = 26
let s:ct.cyan       = 44
let s:ct.cyan_dark  = 37
let s:ct.green      = 71
let s:ct.green_dark = 29
let s:ct.yellow     = 226
let s:ct.yellow_dk  = 220
let s:ct.amber      = 214
let s:ct.amber_dark = 172
let s:ct.orange     = 208
let s:ct.orange_dk  = 130
let s:ct.dorange    = 202
let s:ct.dorange_dk = 166

" --- Semantic colors ---------------------------------------------------------

let s:fg_normal     = s:grey_900
let s:fg_dim        = s:grey_700
let s:fg_comment    = s:grey_600
let s:fg_string     = s:green_700
let s:fg_number     = s:orange_700
let s:fg_keyword    = s:purple_500
let s:fg_type       = s:cyan_700
let s:fg_function   = s:blue_700
let s:fg_constant   = s:dorange_700
let s:fg_preproc    = s:pink_500
let s:fg_error      = s:red_a200
let s:bg_error      = s:red_900
let s:fg_warn       = s:amber_700
let s:fg_match      = s:yellow_700
let s:fg_search     = s:grey_900
let s:bg_search     = s:yellow_500
let s:bg_normal      = s:white
let s:bg_dim        = s:grey_100
let s:bg_hi         = s:grey_100
let s:bg_sel        = s:blue_300

let s:fg_normal_ct  = s:ct.grey900
let s:fg_dim_ct     = s:ct.grey700
let s:fg_comment_ct = s:ct.grey600
let s:fg_string_ct  = s:ct.green_dark
let s:fg_number_ct  = s:ct.orange_dk
let s:fg_keyword_ct = s:ct.purple
let s:fg_type_ct    = s:ct.cyan_dark
let s:fg_function_ct = s:ct.blue_dark
let s:fg_constant_ct = s:ct.dorange_dk
let s:fg_preproc_ct = s:ct.pink
let s:fg_error_ct   = s:ct.red
let s:bg_error_ct   = s:ct.red_dark
let s:fg_warn_ct    = s:ct.amber_dark
let s:fg_match_ct   = s:ct.yellow_dk
let s:fg_search_ct  = s:ct.grey900
let s:bg_search_ct  = s:ct.yellow_dk
let s:bg_normal_ct  = s:ct.white
let s:bg_dim_ct     = s:ct.grey100
let s:bg_hi_ct      = s:ct.grey100
let s:bg_sel_ct     = s:ct.blue

" --- Helper ------------------------------------------------------------------

let s:gui = has('gui_running') || has('termguicolors') && &termguicolors

function! s:HI(group, guifg, guibg, guisp, style, cfg, cbg)
  let l:cmd = 'highlight ' . a:group

  " GUI colors
  if s:gui
    if a:guifg !=# ''
      let l:cmd .= ' guifg=' . a:guifg
    endif
    if a:guibg !=# ''
      let l:cmd .= ' guibg=' . a:guibg
    endif
    if a:guisp !=# ''
      let l:cmd .= ' guisp=' . a:guisp
    endif
    let l:cmd .= ' gui=' . a:style
  endif

  " cterm colors
  if a:cfg !=# ''
    let l:cmd .= ' ctermfg=' . a:cfg
  endif
  if a:cbg !=# ''
    let l:cmd .= ' ctermbg=' . a:cbg
  endif
  let l:cmd .= ' cterm=' . a:style

  execute l:cmd
endfunction

highlight clear
if exists('syntax_on')
  syntax reset
endif
let g:colors_name = 'material'

" --- UI -----------------------------------------------------------------------

call s:HI('Normal',        s:fg_normal,   s:bg_normal,  '', 'NONE',     s:fg_normal_ct,   s:bg_normal_ct)
call s:HI('Cursor',        s:bg_normal,   s:fg_normal,  '', 'NONE',     s:bg_normal_ct,   s:fg_normal_ct)
call s:HI('CursorColumn',  '',            s:bg_hi,      '', 'NONE',     '',               s:bg_hi_ct)
call s:HI('CursorLine',    '',            s:bg_hi,      '', 'NONE',     '',               s:bg_hi_ct)
call s:HI('CursorLineNr',  s:fg_warn,     s:bg_hi,      '', 'NONE',     s:fg_warn_ct,     s:bg_hi_ct)
call s:HI('LineNr',        s:fg_comment,  s:bg_normal,  '', 'NONE',     s:fg_comment_ct,  s:bg_normal_ct)
call s:HI('FoldColumn',    s:fg_comment,  s:bg_dim,     '', 'NONE',     s:fg_comment_ct,  s:bg_dim_ct)
call s:HI('Folded',        s:fg_dim,      s:bg_dim,     '', 'NONE',     s:fg_dim_ct,      s:bg_dim_ct)
call s:HI('SignColumn',    '',            s:bg_normal,  '', 'NONE',     '',               s:bg_normal_ct)
call s:HI('VertSplit',     s:bg_400,      '',           '', 'NONE',     s:ct.bg400,       '')
call s:HI('StatusLine',    s:bg_normal,   s:bg_500,     '', 'NONE',     s:bg_normal_ct,   s:ct.bg500)
call s:HI('StatusLineNC',  s:fg_dim,      s:bg_dim,     '', 'NONE',     s:fg_dim_ct,      s:bg_dim_ct)
call s:HI('WildMenu',      s:fg_search,   s:bg_search,  '', 'NONE',     s:fg_search_ct,   s:bg_search_ct)
call s:HI('TabLine',       s:fg_dim,      s:bg_dim,     '', 'NONE',     s:fg_dim_ct,      s:bg_dim_ct)
call s:HI('TabLineFill',   '',            s:bg_normal,  '', 'NONE',     '',               s:bg_normal_ct)
call s:HI('TabLineSel',    s:fg_normal,   s:bg_hi,      '', 'NONE',     s:fg_normal_ct,   s:bg_hi_ct)
call s:HI('Search',        s:fg_search,   s:bg_search,  '', 'NONE',     s:fg_search_ct,   s:bg_search_ct)
call s:HI('IncSearch',     s:fg_search,   s:bg_search,  '', 'NONE',     s:fg_search_ct,   s:bg_search_ct)
call s:HI('MatchParen',    s:fg_match,    '',           '', 'bold',     s:fg_match_ct,    '')
call s:HI('Visual',        '',            s:bg_sel,    '', 'NONE',     '',               s:bg_sel_ct)
call s:HI('VisualNOS',     '',            s:bg_dim,    '', 'underline','',                s:bg_dim_ct)
call s:HI('ErrorMsg',      s:fg_error,    s:bg_normal,  '', 'NONE',     s:fg_error_ct,    s:bg_normal_ct)
call s:HI('WarningMsg',    s:fg_warn,     s:bg_normal,  '', 'NONE',     s:fg_warn_ct,     s:bg_normal_ct)
call s:HI('ModeMsg',       s:fg_dim,      '',           '', 'NONE',     s:fg_dim_ct,      '')
call s:HI('MoreMsg',       s:fg_warn,     '',           '', 'NONE',     s:fg_warn_ct,     '')
call s:HI('Question',      s:fg_warn,     '',           '', 'NONE',     s:fg_warn_ct,     '')
call s:HI('NonText',       s:fg_comment,  '',           '', 'NONE',     s:fg_comment_ct,  '')
call s:HI('SpecialKey',    s:fg_comment,  '',           '', 'NONE',     s:fg_comment_ct,  '')
call s:HI('Conceal',       s:fg_comment,  s:bg_normal,  '', 'NONE',     s:fg_comment_ct,  s:bg_normal_ct)
call s:HI('Title',         s:fg_type,     '',           '', 'bold',     s:fg_type_ct,    '')
call s:HI('Directory',     s:fg_function, '',           '', 'bold',     s:fg_function_ct, '')
call s:HI('Pmenu',         s:fg_normal,   s:bg_hi,      '', 'NONE',     s:fg_normal_ct,   s:bg_hi_ct)
call s:HI('PmenuSel',      s:bg_normal,   s:bg_sel,    '', 'NONE',     s:bg_normal_ct,   s:bg_sel_ct)
call s:HI('PmenuSbar',     '',            s:bg_hi,      '', 'NONE',     '',               s:bg_hi_ct)
call s:HI('PmenuThumb',    '',            s:fg_comment, '', 'NONE',     '',               s:fg_comment_ct)
call s:HI('SpellBad',      s:fg_error,    '',           '', 'undercurl',s:fg_error_ct,    '')
call s:HI('SpellCap',      s:blue_500,    '',           '', 'undercurl',s:ct.blue,        '')
call s:HI('SpellLocal',    s:cyan_500,    '',           '', 'undercurl',s:ct.cyan,        '')
call s:HI('SpellRare',     s:purple_500,  '',           '', 'undercurl',s:ct.purple,      '')
call s:HI('DiffAdd',        s:green_700,   s:bg_normal, '', 'NONE',     s:ct.green_dark,  s:bg_normal_ct)
call s:HI('DiffChange',     s:amber_700,   s:bg_normal, '', 'NONE',     s:ct.amber_dark,  s:bg_normal_ct)
call s:HI('DiffDelete',     s:red_700,     s:bg_normal, '', 'NONE',     s:ct.red_dark,    s:bg_normal_ct)
call s:HI('DiffText',       s:amber_700,   s:bg_normal, '', 'bold',     s:ct.amber_dark,  s:bg_normal_ct)

" --- Syntax -------------------------------------------------------------------

call s:HI('Comment',         s:fg_comment,  '', '', 'italic',    s:fg_comment_ct,  '')
call s:HI('Constant',        s:fg_constant, '', '', 'NONE',      s:fg_constant_ct, '')
call s:HI('String',          s:fg_string,   '', '', 'NONE',      s:fg_string_ct,   '')
call s:HI('Character',       s:fg_string,   '', '', 'NONE',      s:fg_string_ct,   '')
call s:HI('Number',          s:fg_number,   '', '', 'NONE',      s:fg_number_ct,   '')
call s:HI('Boolean',         s:fg_number,   '', '', 'bold',      s:fg_number_ct,   '')
call s:HI('Float',           s:fg_number,   '', '', 'NONE',      s:fg_number_ct,   '')
call s:HI('Identifier',       s:fg_normal,   '', '', 'NONE',      s:fg_normal_ct,   '')
call s:HI('Function',        s:fg_function, '', '', 'bold',      s:fg_function_ct, '')
call s:HI('Statement',       s:fg_keyword,  '', '', 'NONE',      s:fg_keyword_ct,  '')
call s:HI('Conditional',     s:fg_keyword,  '', '', 'bold',      s:fg_keyword_ct,  '')
call s:HI('Repeat',          s:fg_keyword,  '', '', 'bold',      s:fg_keyword_ct,  '')
call s:HI('Label',           s:fg_keyword,  '', '', 'NONE',      s:fg_keyword_ct,  '')
call s:HI('Operator',        s:fg_normal,   '', '', 'NONE',      s:fg_normal_ct,   '')
call s:HI('Keyword',         s:fg_keyword,  '', '', 'bold',      s:fg_keyword_ct,  '')
call s:HI('Exception',       s:fg_error,    '', '', 'bold',      s:fg_error_ct,    '')
call s:HI('PreProc',         s:fg_preproc,  '', '', 'NONE',      s:fg_preproc_ct,  '')
call s:HI('Include',         s:fg_preproc,  '', '', 'NONE',      s:fg_preproc_ct,  '')
call s:HI('Define',          s:fg_preproc,  '', '', 'NONE',      s:fg_preproc_ct,  '')
call s:HI('Macro',           s:fg_preproc,  '', '', 'NONE',      s:fg_preproc_ct,  '')
call s:HI('PreCondit',       s:fg_preproc,  '', '', 'NONE',      s:fg_preproc_ct,  '')
call s:HI('Type',            s:fg_type,     '', '', 'NONE',      s:fg_type_ct,     '')
call s:HI('StorageClass',    s:fg_type,     '', '', 'bold',      s:fg_type_ct,     '')
call s:HI('Structure',       s:fg_type,     '', '', 'bold',      s:fg_type_ct,     '')
call s:HI('Typedef',         s:fg_type,     '', '', 'bold',      s:fg_type_ct,     '')
call s:HI('Special',         s:fg_warn,     '', '', 'NONE',      s:fg_warn_ct,     '')
call s:HI('SpecialChar',     s:fg_warn,     '', '', 'NONE',      s:fg_warn_ct,     '')
call s:HI('Tag',             s:fg_preproc,  '', '', 'bold',      s:fg_preproc_ct,  '')
call s:HI('Delimiter',       s:fg_normal,   '', '', 'NONE',      s:fg_normal_ct,   '')
call s:HI('SpecialComment',  s:fg_comment,  '', '', 'bold',      s:fg_comment_ct,  '')
call s:HI('Debug',           s:fg_error,    '', '', 'NONE',      s:fg_error_ct,    '')
call s:HI('Error',           s:fg_error,    s:bg_error, '', 'bold', s:fg_error_ct,  s:bg_error_ct)
call s:HI('Todo',            s:bg_normal,   s:fg_warn,  '', 'bold', s:bg_normal_ct, s:fg_warn_ct)
call s:HI('Underlined',      s:fg_function, '', '', 'underline', s:fg_function_ct, '')
call s:HI('Ignore',          s:fg_comment,  '', '', 'NONE',      s:fg_comment_ct,  '')

" --- VimL ---------------------------------------------------------------------

call s:HI('vimOption',   s:fg_preproc,  '', '', 'NONE', s:fg_preproc_ct, '')
call s:HI('vimGroup',    s:fg_type,     '', '', 'NONE', s:fg_type_ct,    '')
call s:HI('vimHiGroup',  s:fg_type,     '', '', 'NONE', s:fg_type_ct,    '')
call s:HI('vimCommand',  s:fg_keyword,  '', '', 'NONE', s:fg_keyword_ct, '')
call s:HI('vimLet',      s:fg_keyword,  '', '', 'NONE', s:fg_keyword_ct, '')
call s:HI('vimMap',      s:fg_preproc,  '', '', 'NONE', s:fg_preproc_ct, '')
call s:HI('vimNotation', s:fg_warn,     '', '', 'NONE', s:fg_warn_ct,    '')

" --- Neovim: LSP / Treesitter ------------------------------------------------

if has('nvim')
  call s:HI('DiagnosticError', s:fg_error,    '', '', 'NONE',      s:fg_error_ct,    '')
  call s:HI('DiagnosticWarn',  s:fg_warn,     '', '', 'NONE',      s:fg_warn_ct,     '')
  call s:HI('DiagnosticInfo',  s:fg_function, '', '', 'NONE',      s:fg_function_ct, '')
  call s:HI('DiagnosticHint',  s:fg_type,     '', '', 'NONE',      s:fg_type_ct,     '')
  call s:HI('DiagnosticUnderlineError', '', '', s:fg_error,    'undercurl', '', '')
  call s:HI('DiagnosticUnderlineWarn',  '', '', s:fg_warn,     'undercurl', '', '')
  call s:HI('DiagnosticUnderlineInfo',  '', '', s:fg_function, 'undercurl', '', '')
  call s:HI('DiagnosticUnderlineHint',  '', '', s:fg_type,     'undercurl', '', '')

  call s:HI('@variable',             s:fg_normal,   '', '', 'NONE',      s:fg_normal_ct,   '')
  call s:HI('@variable.builtin',     s:fg_constant, '', '', 'NONE',      s:fg_constant_ct, '')
  call s:HI('@variable.parameter',   s:fg_normal,   '', '', 'italic',    s:fg_normal_ct,   '')
  call s:HI('@constant',             s:fg_constant, '', '', 'NONE',      s:fg_constant_ct, '')
  call s:HI('@constant.builtin',     s:fg_constant, '', '', 'bold',      s:fg_constant_ct, '')
  call s:HI('@string',               s:fg_string,   '', '', 'NONE',      s:fg_string_ct,   '')
  call s:HI('@string.escape',        s:fg_warn,     '', '', 'NONE',      s:fg_warn_ct,     '')
  call s:HI('@number',               s:fg_number,   '', '', 'NONE',      s:fg_number_ct,   '')
  call s:HI('@boolean',              s:fg_number,   '', '', 'bold',      s:fg_number_ct,   '')
  call s:HI('@function',             s:fg_function, '', '', 'bold',      s:fg_function_ct, '')
  call s:HI('@function.builtin',     s:fg_function, '', '', 'NONE',      s:fg_function_ct, '')
  call s:HI('@function.call',        s:fg_function, '', '', 'NONE',      s:fg_function_ct, '')
  call s:HI('@method',               s:fg_function, '', '', 'NONE',      s:fg_function_ct, '')
  call s:HI('@method.call',          s:fg_function, '', '', 'NONE',      s:fg_function_ct, '')
  call s:HI('@constructor',          s:fg_type,     '', '', 'NONE',      s:fg_type_ct,     '')
  call s:HI('@keyword',              s:fg_keyword,  '', '', 'bold',      s:fg_keyword_ct,  '')
  call s:HI('@keyword.function',     s:fg_keyword,  '', '', 'bold',      s:fg_keyword_ct,  '')
  call s:HI('@keyword.operator',     s:fg_keyword,  '', '', 'NONE',      s:fg_keyword_ct,  '')
  call s:HI('@operator',             s:fg_normal,   '', '', 'NONE',      s:fg_normal_ct,   '')
  call s:HI('@punctuation',          s:fg_normal,   '', '', 'NONE',      s:fg_normal_ct,   '')
  call s:HI('@punctuation.bracket',  s:fg_comment,  '', '', 'NONE',      s:fg_comment_ct,  '')
  call s:HI('@punctuation.delimiter',s:fg_comment, '', '', 'NONE',      s:fg_comment_ct,  '')
  call s:HI('@type',                 s:fg_type,     '', '', 'NONE',      s:fg_type_ct,     '')
  call s:HI('@type.builtin',         s:fg_type,     '', '', 'bold',      s:fg_type_ct,     '')
  call s:HI('@comment',              s:fg_comment,  '', '', 'italic',    s:fg_comment_ct,  '')
  call s:HI('@tag',                  s:fg_keyword,  '', '', 'NONE',      s:fg_keyword_ct,  '')
  call s:HI('@tag.attribute',        s:fg_preproc,  '', '', 'NONE',      s:fg_preproc_ct,  '')
  call s:HI('@tag.delimiter',        s:fg_comment,  '', '', 'NONE',      s:fg_comment_ct,  '')
  call s:HI('@text',                 s:fg_normal,   '', '', 'NONE',      s:fg_normal_ct,   '')
  call s:HI('@text.strong',          s:fg_normal,   '', '', 'bold',      s:fg_normal_ct,   '')
  call s:HI('@text.italic',          s:fg_normal,   '', '', 'italic',    s:fg_normal_ct,   '')
  call s:HI('@text.underline',       s:fg_normal,   '', '', 'underline', s:fg_normal_ct,   '')
  call s:HI('@text.uri',             s:fg_function, '', '', 'underline', s:fg_function_ct, '')
  call s:HI('@text.title',           s:fg_type,     '', '', 'bold',      s:fg_type_ct,     '')
  call s:HI('@text.literal',         s:fg_string,   '', '', 'NONE',      s:fg_string_ct,   '')
  call s:HI('@module',               s:fg_type,     '', '', 'NONE',      s:fg_type_ct,     '')
  call s:HI('@label',                s:fg_preproc,  '', '', 'NONE',      s:fg_preproc_ct,  '')
endif

" --- Cleanup ------------------------------------------------------------------

delfunction s:HI

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
let s:bg_normal     = s:white
let s:bg_dim        = s:grey_100
let s:bg_hi         = s:grey_100
let s:bg_sel        = s:blue_300

" --- Helper ------------------------------------------------------------------

function! s:HI(group, guifg, guibg, guisp, style)
  let l:cmd = 'highlight ' . a:group

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

  execute l:cmd
endfunction

highlight clear
if exists('syntax_on')
  syntax reset
endif
let g:colors_name = 'material'

" --- UI -----------------------------------------------------------------------

call s:HI('Normal',        s:fg_normal,   s:bg_normal,  '', 'NONE')
call s:HI('Cursor',        s:bg_normal,   s:fg_normal,  '', 'NONE')
call s:HI('CursorColumn',  '',            s:bg_hi,      '', 'NONE')
call s:HI('CursorLine',    '',            s:bg_hi,      '', 'NONE')
call s:HI('CursorLineNr',  s:fg_warn,     s:bg_hi,      '', 'NONE')
call s:HI('LineNr',        s:fg_comment,  s:bg_normal,  '', 'NONE')
call s:HI('FoldColumn',    s:fg_comment,  s:bg_dim,     '', 'NONE')
call s:HI('Folded',        s:fg_dim,      s:bg_dim,     '', 'NONE')
call s:HI('SignColumn',    '',            s:bg_normal,  '', 'NONE')
call s:HI('VertSplit',     s:bg_400,      '',           '', 'NONE')
call s:HI('StatusLine',    s:bg_normal,   s:bg_500,     '', 'NONE')
call s:HI('StatusLineNC',  s:fg_dim,      s:bg_dim,     '', 'NONE')
call s:HI('WildMenu',      s:fg_search,   s:bg_search,  '', 'NONE')
call s:HI('TabLine',       s:fg_dim,      s:bg_dim,     '', 'NONE')
call s:HI('TabLineFill',   '',            s:bg_normal,  '', 'NONE')
call s:HI('TabLineSel',    s:fg_normal,   s:bg_hi,      '', 'NONE')
call s:HI('Search',        s:fg_search,   s:bg_search,  '', 'NONE')
call s:HI('IncSearch',     s:fg_search,   s:bg_search,  '', 'NONE')
call s:HI('MatchParen',    s:fg_match,    '',           '', 'bold')
call s:HI('Visual',        '',            s:bg_sel,     '', 'NONE')
call s:HI('VisualNOS',     '',            s:bg_dim,     '', 'underline')
call s:HI('ErrorMsg',      s:fg_error,    s:bg_normal,  '', 'NONE')
call s:HI('WarningMsg',    s:fg_warn,     s:bg_normal,  '', 'NONE')
call s:HI('ModeMsg',       s:fg_dim,      '',           '', 'NONE')
call s:HI('MoreMsg',       s:fg_warn,     '',           '', 'NONE')
call s:HI('Question',      s:fg_warn,     '',           '', 'NONE')
call s:HI('NonText',       s:fg_comment,  '',           '', 'NONE')
call s:HI('SpecialKey',    s:fg_comment,  '',           '', 'NONE')
call s:HI('Conceal',       s:fg_comment,  s:bg_normal,  '', 'NONE')
call s:HI('Title',         s:fg_type,     '',           '', 'bold')
call s:HI('Directory',     s:fg_function, '',           '', 'bold')
call s:HI('Pmenu',         s:fg_normal,   s:bg_hi,      '', 'NONE')
call s:HI('PmenuSel',      s:bg_normal,   s:bg_sel,     '', 'NONE')
call s:HI('PmenuSbar',     '',            s:bg_hi,      '', 'NONE')
call s:HI('PmenuThumb',    '',            s:fg_comment, '', 'NONE')
call s:HI('SpellBad',      s:fg_error,    '',           '', 'undercurl')
call s:HI('SpellCap',      s:blue_500,    '',           '', 'undercurl')
call s:HI('SpellLocal',    s:cyan_500,    '',           '', 'undercurl')
call s:HI('SpellRare',     s:purple_500,  '',           '', 'undercurl')
call s:HI('DiffAdd',       s:green_700,   s:bg_normal,  '', 'NONE')
call s:HI('DiffChange',    s:amber_700,   s:bg_normal,  '', 'NONE')
call s:HI('DiffDelete',    s:red_700,     s:bg_normal,  '', 'NONE')
call s:HI('DiffText',      s:amber_700,   s:bg_normal,  '', 'bold')

" --- Syntax -------------------------------------------------------------------

call s:HI('Comment',         s:fg_comment,  '', '', 'NONE')
call s:HI('Constant',        s:fg_constant, '', '', 'NONE')
call s:HI('String',          s:fg_string,   '', '', 'NONE')
call s:HI('Character',       s:fg_string,   '', '', 'NONE')
call s:HI('Number',          s:fg_number,   '', '', 'NONE')
call s:HI('Boolean',         s:fg_number,   '', '', 'bold')
call s:HI('Float',           s:fg_number,   '', '', 'NONE')
call s:HI('Identifier',      s:fg_normal,   '', '', 'NONE')
call s:HI('Function',        s:fg_function, '', '', 'bold')
call s:HI('Statement',       s:fg_keyword,  '', '', 'bold')
call s:HI('Conditional',     s:fg_keyword,  '', '', 'bold')
call s:HI('Repeat',          s:fg_keyword,  '', '', 'bold')
call s:HI('Label',           s:fg_keyword,  '', '', 'NONE')
call s:HI('Operator',        s:fg_normal,   '', '', 'NONE')
call s:HI('Keyword',         s:fg_keyword,  '', '', 'bold')
call s:HI('Exception',       s:fg_error,    '', '', 'bold')
call s:HI('PreProc',         s:fg_preproc,  '', '', 'NONE')
call s:HI('Include',         s:fg_preproc,  '', '', 'NONE')
call s:HI('Define',          s:fg_preproc,  '', '', 'NONE')
call s:HI('Macro',           s:fg_preproc,  '', '', 'NONE')
call s:HI('PreCondit',       s:fg_preproc,  '', '', 'NONE')
call s:HI('Type',            s:fg_type,     '', '', 'NONE')
call s:HI('StorageClass',    s:fg_type,     '', '', 'bold')
call s:HI('Structure',       s:fg_type,     '', '', 'bold')
call s:HI('Typedef',         s:fg_type,     '', '', 'bold')
call s:HI('Special',         s:fg_warn,     '', '', 'NONE')
call s:HI('SpecialChar',     s:fg_warn,     '', '', 'NONE')
call s:HI('Tag',             s:fg_preproc,  '', '', 'bold')
call s:HI('Delimiter',       s:fg_normal,   '', '', 'NONE')
call s:HI('SpecialComment',  s:fg_comment,  '', '', 'bold')
call s:HI('Debug',           s:fg_error,    '', '', 'NONE')
call s:HI('Error',           s:fg_error,    s:bg_error, '', 'bold')
call s:HI('Todo',            s:bg_normal,   s:fg_warn,  '', 'bold')
call s:HI('Underlined',      s:fg_function, '', '', 'underline')
call s:HI('Ignore',          s:fg_comment,  '', '', 'NONE')

" --- VimL ---------------------------------------------------------------------

call s:HI('vimOption',   s:fg_preproc,  '', '', 'NONE')
call s:HI('vimGroup',    s:fg_type,     '', '', 'NONE')
call s:HI('vimHiGroup',  s:fg_type,     '', '', 'NONE')
call s:HI('vimCommand',  s:fg_keyword,  '', '', 'NONE')
call s:HI('vimLet',      s:fg_keyword,  '', '', 'NONE')
call s:HI('vimMap',      s:fg_preproc,  '', '', 'NONE')
call s:HI('vimNotation', s:fg_warn,     '', '', 'NONE')

" --- Cleanup ------------------------------------------------------------------

delfunction s:HI

package scarlet

import (
   "html"
   "strings"
)

const (
   stateDefault = iota
   stateOrderedList
   stateUnorderedList
   stateOrderedListNestedUL
   stateUnorderedListNestedOL
   stateCodeBlock
   stateOrderedListPending
   stateUnorderedListPending
)

func isBlank(line string) bool {
   return strings.TrimSpace(line) == ""
}

func isCodeFence(line string) (int, bool) {
   trimmed := strings.TrimLeft(line, " ")
   indent := len(line) - len(trimmed)
   if strings.TrimSpace(trimmed) == "```" {
      return indent, true
   }
   return 0, false
}

func isIndented(line string) bool {
   return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}

func isOrderedListLine(line string) (string, bool) {
   rest := strings.TrimLeft(line, " \t")
   n := 0
   for len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
      n = n*10 + int(rest[0]-'0')
      rest = rest[1:]
   }
   if n == 0 || len(rest) < 2 || rest[0] != '.' {
      return "", false
   }
   rest = strings.TrimLeft(rest[1:], " ")
   return rest, true
}

func isUnorderedListLine(line string) (string, bool) {
   rest := strings.TrimLeft(line, " \t")
   if len(rest) == 0 || rest[0] != '*' {
      return "", false
   }
   content := strings.TrimLeft(rest[1:], " ")
   if len(content) > 0 && content[len(content)-1] == '*' {
      return "", false
   }
   return content, true
}

func openLi(b *strings.Builder, text string) {
   b.WriteString("<li>")
   b.WriteString(renderInline(text))
}

func renderInline(s string) string {
   s = strings.ReplaceAll(s, "-&gt;", "→")

   var result strings.Builder
   inBold := false
   inItalic := false
   inCode := false
   i := 0
   for i < len(s) {
      if s[i] == '`' {
         if inCode {
            result.WriteString("</code>")
         } else {
            result.WriteString("<code>")
         }
         inCode = !inCode
         i++
      } else if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
         if inBold {
            result.WriteString("</b>")
         } else {
            result.WriteString("<b>")
         }
         inBold = !inBold
         i += 2
      } else if s[i] == '*' {
         if inItalic {
            result.WriteString("</i>")
         } else {
            result.WriteString("<i>")
         }
         inItalic = !inItalic
         i++
      } else {
         result.WriteByte(s[i])
         i++
      }
   }
   return result.String()
}

// renderMarkdown renders a complete markdown string to HTML (non-streaming).
func renderMarkdown(s string) string {
   s = html.EscapeString(s)
   lines := strings.Split(s, "\n")
   mr := newMarkdownRenderer()
   var result strings.Builder
   for _, line := range lines {
      result.WriteString(mr.renderLine(line))
   }
   result.WriteString(mr.flush())
   out := result.String()
   out = strings.TrimSuffix(out, "<br>\n")
   return out
}

// markdownRenderer holds parser state across multiple lines,
// enabling streaming (line-by-line) markdown rendering.
type markdownRenderer struct {
   state                int
   codeLines            []string
   codeBlockIndent      int
   codeBlockReturnState int
   liOpen               bool
}

func newMarkdownRenderer() *markdownRenderer {
   return &markdownRenderer{state: stateDefault}
}

func (mr *markdownRenderer) closeListIfOpen(b *strings.Builder) {
   if mr.liOpen {
      b.WriteString("</li>")
      mr.liOpen = false
   }
}

// flush closes any open elements and returns the remaining HTML.
// After calling flush, the renderer is reset to the default state.
func (mr *markdownRenderer) flush() string {
   b := &strings.Builder{}
   mr.closeListIfOpen(b)
   switch mr.state {
   case stateCodeBlock:
      b.WriteString("<pre>")
      b.WriteString(strings.Join(mr.codeLines, "\n"))
      b.WriteString("</pre>\n")
   case stateOrderedList, stateOrderedListPending:
      b.WriteString("</ol>")
   case stateUnorderedList, stateUnorderedListPending:
      b.WriteString("</ul>")
   case stateOrderedListNestedUL:
      b.WriteString("</ul></ol>")
   case stateUnorderedListNestedOL:
      b.WriteString("</ol></ul>")
   }
   mr.state = stateDefault
   mr.codeLines = nil
   mr.liOpen = false
   return b.String()
}

func (mr *markdownRenderer) processLine(b *strings.Builder, line string) {
   switch mr.state {
   case stateCodeBlock:
      if _, ok := isCodeFence(line); ok {
         b.WriteString("<pre>")
         b.WriteString(strings.Join(mr.codeLines, "\n"))
         b.WriteString("</pre>\n")
         mr.codeLines = nil
         mr.state = mr.codeBlockReturnState
      } else {
         if len(line) >= mr.codeBlockIndent {
            line = line[mr.codeBlockIndent:]
         }
         mr.codeLines = append(mr.codeLines, line)
      }

   case stateDefault:
      if indent, ok := isCodeFence(line); ok {
         mr.codeBlockIndent = indent
         mr.codeBlockReturnState = stateDefault
         mr.state = stateCodeBlock
      } else if strings.TrimSpace(line) == "---" {
         b.WriteString("<hr>\n")
      } else {
         ordText, isOrd := isOrderedListLine(line)
         unordText, isUnord := isUnorderedListLine(line)
         if isOrd {
            b.WriteString("<ol>")
            openLi(b, ordText)
            mr.liOpen = true
            mr.state = stateOrderedList
         } else if isUnord {
            b.WriteString("<ul>")
            openLi(b, unordText)
            mr.liOpen = true
            mr.state = stateUnorderedList
         } else if isBlank(line) {
            b.WriteString("<br>\n")
         } else {
            b.WriteString(renderInline(line))
            b.WriteString("<br>\n")
         }
      }

   case stateOrderedList:
      if indent, ok := isCodeFence(line); ok {
         mr.closeListIfOpen(b)
         mr.codeBlockIndent = indent
         mr.codeBlockReturnState = stateOrderedList
         mr.state = stateCodeBlock
      } else {
         ordText, isOrd := isOrderedListLine(line)
         unordText, isUnord := isUnorderedListLine(line)
         if isOrd {
            mr.closeListIfOpen(b)
            openLi(b, ordText)
            mr.liOpen = true
         } else if isUnord && isIndented(line) {
            mr.closeListIfOpen(b)
            b.WriteString("<ul>")
            openLi(b, unordText)
            mr.liOpen = true
            mr.state = stateOrderedListNestedUL
         } else if isUnord {
            mr.closeListIfOpen(b)
            b.WriteString("</ol><ul>")
            openLi(b, unordText)
            mr.liOpen = true
            mr.state = stateUnorderedList
         } else if isBlank(line) {
            mr.state = stateOrderedListPending
         } else if isIndented(line) {
            b.WriteString("<br>\n")
            b.WriteString(renderInline(strings.TrimLeft(line, " \t")))
         } else {
            mr.closeListIfOpen(b)
            b.WriteString("</ol>")
            if strings.TrimSpace(line) == "---" {
               b.WriteString("<hr>\n")
            } else {
               b.WriteString(renderInline(line))
               b.WriteByte('\n')
            }
            mr.state = stateDefault
         }
      }

   case stateUnorderedList:
      if indent, ok := isCodeFence(line); ok {
         mr.closeListIfOpen(b)
         mr.codeBlockIndent = indent
         mr.codeBlockReturnState = stateUnorderedList
         mr.state = stateCodeBlock
      } else {
         ordText, isOrd := isOrderedListLine(line)
         unordText, isUnord := isUnorderedListLine(line)
         if isUnord {
            mr.closeListIfOpen(b)
            openLi(b, unordText)
            mr.liOpen = true
         } else if isOrd && isIndented(line) {
            mr.closeListIfOpen(b)
            b.WriteString("<ol>")
            openLi(b, ordText)
            mr.liOpen = true
            mr.state = stateUnorderedListNestedOL
         } else if isOrd {
            mr.closeListIfOpen(b)
            b.WriteString("</ul><ol>")
            openLi(b, ordText)
            mr.liOpen = true
            mr.state = stateOrderedList
         } else if isBlank(line) {
            mr.state = stateUnorderedListPending
         } else if isIndented(line) {
            b.WriteString("<br>\n")
            b.WriteString(renderInline(strings.TrimLeft(line, " \t")))
         } else {
            mr.closeListIfOpen(b)
            b.WriteString("</ul>")
            if strings.TrimSpace(line) == "---" {
               b.WriteString("<hr>\n")
            } else {
               b.WriteString(renderInline(line))
               b.WriteByte('\n')
            }
            mr.state = stateDefault
         }
      }

   case stateOrderedListNestedUL:
      if indent, ok := isCodeFence(line); ok {
         mr.closeListIfOpen(b)
         mr.codeBlockIndent = indent
         mr.codeBlockReturnState = stateOrderedListNestedUL
         mr.state = stateCodeBlock
      } else {
         ordText, isOrd := isOrderedListLine(line)
         unordText, isUnord := isUnorderedListLine(line)
         if isUnord {
            mr.closeListIfOpen(b)
            openLi(b, unordText)
            mr.liOpen = true
         } else if isOrd && !isIndented(line) {
            mr.closeListIfOpen(b)
            b.WriteString("</ul>")
            openLi(b, ordText)
            mr.liOpen = true
            mr.state = stateOrderedList
         } else if isBlank(line) {
            mr.closeListIfOpen(b)
            b.WriteString("</ul>")
            mr.state = stateOrderedListPending
         } else if isIndented(line) {
            b.WriteString("<br>\n")
            b.WriteString(renderInline(strings.TrimLeft(line, " \t")))
         } else {
            mr.closeListIfOpen(b)
            b.WriteString("</ul></ol>")
            if strings.TrimSpace(line) == "---" {
               b.WriteString("<hr>\n")
            } else {
               b.WriteString(renderInline(line))
               b.WriteByte('\n')
            }
            mr.state = stateDefault
         }
      }

   case stateUnorderedListNestedOL:
      if indent, ok := isCodeFence(line); ok {
         mr.closeListIfOpen(b)
         mr.codeBlockIndent = indent
         mr.codeBlockReturnState = stateUnorderedListNestedOL
         mr.state = stateCodeBlock
      } else {
         ordText, isOrd := isOrderedListLine(line)
         unordText, isUnord := isUnorderedListLine(line)
         if isOrd {
            mr.closeListIfOpen(b)
            openLi(b, ordText)
            mr.liOpen = true
         } else if isUnord && !isIndented(line) {
            mr.closeListIfOpen(b)
            b.WriteString("</ol>")
            openLi(b, unordText)
            mr.liOpen = true
            mr.state = stateUnorderedList
         } else if isBlank(line) {
            mr.closeListIfOpen(b)
            b.WriteString("</ol>")
            mr.state = stateUnorderedListPending
         } else if isIndented(line) {
            b.WriteString("<br>\n")
            b.WriteString(renderInline(strings.TrimLeft(line, " \t")))
         } else {
            mr.closeListIfOpen(b)
            b.WriteString("</ol></ul>")
            if strings.TrimSpace(line) == "---" {
               b.WriteString("<hr>\n")
            } else {
               b.WriteString(renderInline(line))
               b.WriteByte('\n')
            }
            mr.state = stateDefault
         }
      }

   case stateOrderedListPending:
      if isBlank(line) {
         mr.closeListIfOpen(b)
         b.WriteString("</ol>\n")
         mr.state = stateDefault
      } else {
         ordText, isOrd := isOrderedListLine(line)
         unordText, isUnord := isUnorderedListLine(line)
         if isOrd {
            mr.closeListIfOpen(b)
            openLi(b, ordText)
            mr.liOpen = true
            mr.state = stateOrderedList
         } else if isUnord && isIndented(line) {
            mr.closeListIfOpen(b)
            b.WriteString("<ul>")
            openLi(b, unordText)
            mr.liOpen = true
            mr.state = stateOrderedListNestedUL
         } else if isUnord {
            mr.closeListIfOpen(b)
            b.WriteString("</ol><ul>")
            openLi(b, unordText)
            mr.liOpen = true
            mr.state = stateUnorderedList
         } else if isIndented(line) {
            b.WriteString("<br>\n")
            b.WriteString(renderInline(strings.TrimLeft(line, " \t")))
            mr.state = stateOrderedList
         } else if indent, ok := isCodeFence(line); ok {
            mr.closeListIfOpen(b)
            b.WriteString("</ol>")
            mr.codeBlockIndent = indent
            mr.codeBlockReturnState = stateDefault
            mr.state = stateCodeBlock
         } else {
            mr.closeListIfOpen(b)
            b.WriteString("</ol>")
            if strings.TrimSpace(line) == "---" {
               b.WriteString("<hr>\n")
            } else {
               b.WriteString(renderInline(line))
               b.WriteByte('\n')
            }
            mr.state = stateDefault
         }
      }

   case stateUnorderedListPending:
      if isBlank(line) {
         mr.closeListIfOpen(b)
         b.WriteString("</ul>\n")
         mr.state = stateDefault
      } else {
         ordText, isOrd := isOrderedListLine(line)
         unordText, isUnord := isUnorderedListLine(line)
         if isUnord {
            mr.closeListIfOpen(b)
            openLi(b, unordText)
            mr.liOpen = true
            mr.state = stateUnorderedList
         } else if isOrd && isIndented(line) {
            mr.closeListIfOpen(b)
            b.WriteString("<ol>")
            openLi(b, ordText)
            mr.liOpen = true
            mr.state = stateUnorderedListNestedOL
         } else if isOrd {
            mr.closeListIfOpen(b)
            b.WriteString("</ul><ol>")
            openLi(b, ordText)
            mr.liOpen = true
            mr.state = stateOrderedList
         } else if isIndented(line) {
            b.WriteString("<br>\n")
            b.WriteString(renderInline(strings.TrimLeft(line, " \t")))
            mr.state = stateUnorderedList
         } else if indent, ok := isCodeFence(line); ok {
            mr.closeListIfOpen(b)
            b.WriteString("</ul>")
            mr.codeBlockIndent = indent
            mr.codeBlockReturnState = stateDefault
            mr.state = stateCodeBlock
         } else {
            mr.closeListIfOpen(b)
            b.WriteString("</ul>")
            if strings.TrimSpace(line) == "---" {
               b.WriteString("<hr>\n")
            } else {
               b.WriteString(renderInline(line))
               b.WriteByte('\n')
            }
            mr.state = stateDefault
         }
      }
   }
}

// renderLine processes a single line and returns the HTML fragment for it.
// Parser state is preserved across calls.
func (mr *markdownRenderer) renderLine(line string) string {
   b := &strings.Builder{}
   mr.processLine(b, line)
   return b.String()
}

// streamingMarkdownRenderer wraps markdownRenderer for streaming use.
// It buffers incoming text, renders complete lines as they arrive,
// and flushes remaining state on demand.
type streamingMarkdownRenderer struct {
   renderer *markdownRenderer
   lineBuf  strings.Builder
   onToken  func(string)
}

func newStreamingMarkdownRenderer(onToken func(string)) *streamingMarkdownRenderer {
   return &streamingMarkdownRenderer{
      renderer: newMarkdownRenderer(),
      onToken:  onToken,
   }
}

// finish flushes any partial line and closes all open elements.
// Safe to call multiple times (subsequent calls are no-ops).
func (s *streamingMarkdownRenderer) finish() {
   if s.lineBuf.Len() > 0 {
      line := s.lineBuf.String()
      s.lineBuf.Reset()
      output := s.renderer.renderLine(html.EscapeString(line))
      if output != "" && s.onToken != nil {
         s.onToken(output)
      }
   }
   output := s.renderer.flush()
   if output != "" && s.onToken != nil {
      s.onToken(output)
   }
}

func (s *streamingMarkdownRenderer) write(text string) {
   s.lineBuf.WriteString(text)
   for {
      str := s.lineBuf.String()
      idx := strings.IndexByte(str, '\n')
      if idx < 0 {
         return
      }
      line := str[:idx]
      s.lineBuf.Reset()
      s.lineBuf.WriteString(str[idx+1:])
      output := s.renderer.renderLine(html.EscapeString(line))
      if output != "" && s.onToken != nil {
         s.onToken(output)
      }
   }
}

// markdown.go

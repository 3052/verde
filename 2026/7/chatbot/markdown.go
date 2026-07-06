package main

import (
   "html"
   "strings"
)

func isNumberedList(s string) bool {
   idx := strings.Index(s, ". ")
   if idx > 0 && idx <= 3 {
      for i := 0; i < idx; i++ {
         if s[i] < '0' || s[i] > '9' {
            return false
         }
      }
      return true
   }
   return false
}

// Markdown is a stateful parser used for history and live-streaming chunks
type Markdown struct {
   inCodeBlock bool
   codeIndent  int
   inList      bool
   prevBlock   bool
   wantsBreak  bool
   hasText     bool
}

// Render processes a full historical message block
func (m *Markdown) Render(raw string) string {
   var out strings.Builder
   lines := strings.Split(raw, "\n")

   for _, line := range lines {
      out.WriteString(m.RenderLine(line))
   }

   if m.inList {
      out.WriteString("</ul>")
   }
   if m.inCodeBlock {
      out.WriteString("</pre>")
   }

   return out.String()
}

// RenderLine processes a single buffered line of Markdown into clean HTML
func (m *Markdown) RenderLine(line string) string {
   trimmed := strings.TrimSpace(line)
   var out strings.Builder

   isListItem := strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") || isNumberedList(trimmed)

   // Close an active list if the current line is not a list item or is empty
   if m.inList && !isListItem && !m.inCodeBlock {
      m.inList = false
      out.WriteString("</ul>")
      m.prevBlock = true
   }

   // 1. Check for code block toggles
   if strings.HasPrefix(trimmed, "```") {
      if !m.inCodeBlock {
         m.inCodeBlock = true
         // Calculate how many spaces the AI indented the code block by
         m.codeIndent = len(line) - len(strings.TrimLeft(line, " "))
         out.WriteString("<pre>")
      } else {
         m.inCodeBlock = false
         m.codeIndent = 0
         out.WriteString("</pre>")
      }
      m.prevBlock = true
      m.wantsBreak = false
      return out.String()
   }

   // 2. Safely escape code lines natively and strip the base AI indentation
   if m.inCodeBlock {
      trimCount := 0
      for i := 0; i < len(line); i++ {
         if line[i] == ' ' && trimCount < m.codeIndent {
            trimCount++
         } else {
            break
         }
      }
      out.WriteString(html.EscapeString(line[trimCount:]) + "\n")
      return out.String()
   }

   // 3. Empty lines act as spacing markers, not literal breaks
   if trimmed == "" {
      m.wantsBreak = true
      return out.String()
   }

   // 4. Horizontal Rule
   if trimmed == "---" || trimmed == "***" {
      out.WriteString("<hr>")
      m.prevBlock = true
      m.wantsBreak = false
      return out.String()
   }

   // 5. Headers
   if strings.HasPrefix(trimmed, "### ") {
      out.WriteString("<h3>" + m.parseInline(strings.TrimPrefix(trimmed, "### ")) + "</h3>")
      m.prevBlock = true
      m.wantsBreak = false
      return out.String()
   } else if strings.HasPrefix(trimmed, "## ") {
      out.WriteString("<h2>" + m.parseInline(strings.TrimPrefix(trimmed, "## ")) + "</h2>")
      m.prevBlock = true
      m.wantsBreak = false
      return out.String()
   } else if strings.HasPrefix(trimmed, "# ") {
      out.WriteString("<h1>" + m.parseInline(strings.TrimPrefix(trimmed, "# ")) + "</h1>")
      m.prevBlock = true
      m.wantsBreak = false
      return out.String()
   }

   // 6. List Items
   if isListItem {
      if !m.inList {
         m.inList = true
         out.WriteString("<ul>")
      }

      content := trimmed
      if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
         content = trimmed[2:]
      } else {
         idx := strings.Index(trimmed, ". ")
         if idx != -1 {
            content = trimmed[idx+2:]
         }
      }
      out.WriteString("<li>" + m.parseInline(content) + "</li>")
      m.prevBlock = true
      m.wantsBreak = false
      return out.String()
   }

   // 7. Normal text (Smart Line Breaks)
   if !m.prevBlock {
      if m.wantsBreak {
         out.WriteString("<br><br>")
      } else if m.hasText {
         out.WriteString("<br>")
      }
   }

   m.prevBlock = false
   m.wantsBreak = false
   m.hasText = true
   out.WriteString(m.parseInline(line))
   return out.String()
}

func (m *Markdown) parseInline(line string) string {
   var out strings.Builder
   inInlineCode := false
   inBold := false
   inItalic := false

   runes := []rune(line)
   for j := 0; j < len(runes); j++ {
      r := runes[j]

      if r == '`' {
         inInlineCode = !inInlineCode
         if inInlineCode {
            out.WriteString("<code>")
         } else {
            out.WriteString("</code>")
         }
         continue
      }

      if !inInlineCode && r == '*' {
         if j < len(runes)-1 && runes[j+1] == '*' {
            inBold = !inBold
            if inBold {
               out.WriteString("<strong>")
            } else {
               out.WriteString("</strong>")
            }
            j++
         } else {
            inItalic = !inItalic
            if inItalic {
               out.WriteString("<em>")
            } else {
               out.WriteString("</em>")
            }
         }
         continue
      }

      // Convert "->" to a right arrow, but ONLY if we are not inside an inline code snippet
      if !inInlineCode && r == '-' && j < len(runes)-1 && runes[j+1] == '>' {
         out.WriteString("&rarr;")
         j++ // skip the '>'
         continue
      }

      switch r {
      case '<':
         out.WriteString("&lt;")
      case '>':
         out.WriteString("&gt;")
      case '&':
         out.WriteString("&amp;")
      case '"':
         out.WriteString("&#34;")
      case '\'':
         out.WriteString("&#39;")
      default:
         out.WriteRune(r)
      }
   }

   if inInlineCode {
      out.WriteString("</code>")
   }
   if inBold {
      out.WriteString("</strong>")
   }
   if inItalic {
      out.WriteString("</em>")
   }

   return out.String()
}

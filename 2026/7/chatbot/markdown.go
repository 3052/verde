package main

import (
   "html"
   "strings"
)

// Markdown is a stateful parser used for both history and live-streaming chunks
type Markdown struct {
   inCodeBlock bool
}

// Render processes a full historical message block
func (m *Markdown) Render(raw string) string {
   var out strings.Builder
   lines := strings.Split(raw, "\n")

   for i, line := range lines {
      out.WriteString(m.RenderLine(line))
      if i < len(lines)-1 {
         if m.inCodeBlock {
            out.WriteString("\n")
         } else {
            out.WriteString("<br>")
         }
      }
   }

   // Auto-close open unclosed blocks
   if m.inCodeBlock {
      out.WriteString("</pre>")
   }

   return out.String()
}

// RenderLine processes a single buffered line of Markdown
func (m *Markdown) RenderLine(line string) string {
   trimmed := strings.TrimSpace(line)

   // 1. Check for code block toggles
   if strings.HasPrefix(trimmed, "```") {
      m.inCodeBlock = !m.inCodeBlock
      if m.inCodeBlock {
         return "<pre>"
      }
      return "</pre>"
   }

   // 2. Safely escape code lines
   if m.inCodeBlock {
      return html.EscapeString(line)
   }

   // 3. Process Headers natively
   if strings.HasPrefix(trimmed, "### ") {
      return "<h3>" + m.parseInline(strings.TrimPrefix(trimmed, "### ")) + "</h3>"
   } else if strings.HasPrefix(trimmed, "## ") {
      return "<h2>" + m.parseInline(strings.TrimPrefix(trimmed, "## ")) + "</h2>"
   } else if strings.HasPrefix(trimmed, "# ") {
      return "<h1>" + m.parseInline(strings.TrimPrefix(trimmed, "# ")) + "</h1>"
   }

   // 4. Fallback to inline processing for standard text
   return m.parseInline(line)
}

func (m *Markdown) parseInline(line string) string {
   var out strings.Builder
   inInlineCode := false
   inBold := false

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

      if !inInlineCode && r == '*' && j < len(runes)-1 && runes[j+1] == '*' {
         inBold = !inBold
         if inBold {
            out.WriteString("<strong>")
         } else {
            out.WriteString("</strong>")
         }
         j++ // Skip second asterisk
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

   // Auto-close inline formatting if model stopped mid-sentence on this line
   if inInlineCode {
      out.WriteString("</code>")
   }
   if inBold {
      out.WriteString("</strong>")
   }

   return out.String()
}

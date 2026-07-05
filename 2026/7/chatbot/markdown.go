package main

import (
   "html"
   "strings"
)

// Markdown is a stateful parser used for both history and live-streaming chunks
type Markdown struct {
   inCodeBlock bool
   inList      bool
   hasContent  bool
}

// Render processes a full historical message block
func (m *Markdown) Render(raw string) string {
   var out strings.Builder
   lines := strings.Split(raw, "\n")

   for i, line := range lines {
      htmlStr, needsBreak := m.RenderLine(line)
      out.WriteString(htmlStr)

      if i < len(lines)-1 && needsBreak {
         if m.inCodeBlock {
            out.WriteString("\n")
         } else {
            out.WriteString("<br>")
         }
      }
   }

   // Auto-close open unclosed blocks
   if m.inList {
      out.WriteString("</ul>")
   }
   if m.inCodeBlock {
      out.WriteString("</pre>")
   }

   return out.String()
}

// RenderLine processes a single buffered line of Markdown.
// It returns the HTML string and a boolean indicating if a line break (<br> or \n) should follow.
func (m *Markdown) RenderLine(line string) (string, bool) {
   trimmed := strings.TrimSpace(line)

   if trimmed != "" {
      m.hasContent = true
   }

   var prefix string
   isListItem := strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ")

   // Close an active list if the current line is not a list item
   if m.inList && !isListItem && !m.inCodeBlock {
      m.inList = false
      prefix = "</ul>"
   }

   // 1. Check for code block toggles
   if strings.HasPrefix(trimmed, "```") {
      m.inCodeBlock = !m.inCodeBlock
      if m.inCodeBlock {
         return prefix + "<pre>", false
      }
      return prefix + "</pre>", false
   }

   // 2. Safely escape code lines
   if m.inCodeBlock {
      return prefix + html.EscapeString(line), true
   }

   // 3. Horizontal Rule
   if trimmed == "---" || trimmed == "***" {
      return prefix + "<hr>", false
   }

   // 4. Headers
   if strings.HasPrefix(trimmed, "### ") {
      return prefix + "<h3>" + m.parseInline(strings.TrimPrefix(trimmed, "### ")) + "</h3>", false
   } else if strings.HasPrefix(trimmed, "## ") {
      return prefix + "<h2>" + m.parseInline(strings.TrimPrefix(trimmed, "## ")) + "</h2>", false
   } else if strings.HasPrefix(trimmed, "# ") {
      return prefix + "<h1>" + m.parseInline(strings.TrimPrefix(trimmed, "# ")) + "</h1>", false
   }

   // 5. List Items
   if isListItem {
      if !m.inList {
         m.inList = true
         prefix += "<ul>"
      }
      return prefix + "<li>" + m.parseInline(trimmed[2:]) + "</li>", false
   }

   // 6. Normal text lines
   // Swallow empty newlines if the AI hasn't started generating actual content yet
   if trimmed == "" && !m.hasContent {
      return "", false
   }

   return prefix + m.parseInline(line), true
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
         // Bold (**)
         if j < len(runes)-1 && runes[j+1] == '*' {
            inBold = !inBold
            if inBold {
               out.WriteString("<strong>")
            } else {
               out.WriteString("</strong>")
            }
            j++ // Skip second asterisk
         } else {
            // Italic (*)
            inItalic = !inItalic
            if inItalic {
               out.WriteString("<em>")
            } else {
               out.WriteString("</em>")
            }
         }
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

   // Auto-close inline formatting if model stopped abruptly
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

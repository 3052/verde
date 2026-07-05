package main

import (
   "html"
   "strings"
)

// renderMarkdown is a custom, zero-dependency, zero-regex state machine
// that parses core Markdown elements securely.
func renderMarkdown(raw string) string {
   var out strings.Builder
   inBlockCode := false
   inInlineCode := false
   inBold := false

   lines := strings.Split(raw, "\n")
   for i, line := range lines {
      trimmed := strings.TrimSpace(line)

      // 1. Check for code block toggles (```)
      if strings.HasPrefix(trimmed, "```") {
         inBlockCode = !inBlockCode
         if inBlockCode {
            out.WriteString("<pre>")
         } else {
            out.WriteString("</pre>")
         }
         continue
      }

      // 2. Safely escape and preserve native newlines inside code blocks
      if inBlockCode {
         out.WriteString(html.EscapeString(line))
         if i < len(lines)-1 {
            out.WriteString("\n")
         }
         continue
      }

      // 3. Otherwise, parse inline formatting rune-by-rune
      runes := []rune(line)
      for j := 0; j < len(runes); j++ {
         r := runes[j]

         // Inline code toggle (`)
         if r == '`' {
            inInlineCode = !inInlineCode
            if inInlineCode {
               out.WriteString("<code>")
            } else {
               out.WriteString("</code>")
            }
            continue
         }

         // Bold toggle (**) - ignored if inside inline code
         if !inInlineCode && r == '*' && j < len(runes)-1 && runes[j+1] == '*' {
            inBold = !inBold
            if inBold {
               out.WriteString("<strong>")
            } else {
               out.WriteString("</strong>")
            }
            j++
            continue
         }

         // HTML escaping for text to prevent script injection
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

      // 4. Convert normal newlines to <br> outside of code blocks
      if i < len(lines)-1 {
         out.WriteString("<br>")
      }
   }

   // 5. Auto-close any unclosed tags
   if inInlineCode {
      out.WriteString("</code>")
   }
   if inBold {
      out.WriteString("</strong>")
   }
   if inBlockCode {
      out.WriteString("</pre>")
   }

   return out.String()
}

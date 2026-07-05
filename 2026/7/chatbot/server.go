package main

import (
   "encoding/json"
   "fmt"
   "html"
   "io"
   "log"
   "mime/multipart"
   "net/http"
   "os"
   "strings"
)

func handleRoot(w http.ResponseWriter, r *http.Request, apiKey, headerHTML, footerHTML string) error {
   var messages []Message
   sessionData, err := os.ReadFile(sessionFileName)
   if err != nil {
      log.Printf("error reading %s: %v", sessionFileName, err)
   } else if err := json.Unmarshal(sessionData, &messages); err != nil {
      return fmt.Errorf("critical error parsing %s: %w", sessionFileName, err)
   }

   if r.Method == http.MethodPost {
      r.ParseMultipartForm(10 << 20)

      userText := r.FormValue("text")
      combinedInput := userText

      if files := r.MultipartForm.File["files"]; len(files) > 0 {
         for _, fileHeader := range files {
            fileChunk, err := processUploadedFile(fileHeader)
            if err != nil {
               return err
            }
            combinedInput += fileChunk
         }
      }

      if combinedInput != "" {
         messages = append(messages, Message{Role: "user", Content: combinedInput})
      }
   }

   w.Header().Set("Content-Type", "text/html; charset=utf-8")
   w.Header().Set("Cache-Control", "no-cache")
   flusher, canFlush := w.(http.Flusher)

   fmt.Fprint(w, headerHTML)

   // Render historical messages
   for _, msg := range messages {
      if msg.Role == "system" {
         fmt.Fprintf(w, `<div class="msg %s">%s</div>`+"\n", msg.Role, html.EscapeString(msg.Content))
      } else {
         fmt.Fprintf(w, `<div class="msg %s">`, msg.Role)

         // Dynamically wrap historical reasoning data if it exists
         if msg.ReasoningContent != "" {
            safeRc := strings.ReplaceAll(html.EscapeString(msg.ReasoningContent), "\n", "<br>")
            fmt.Fprintf(w, `<div class="reasoning">%s</div><hr>`, safeRc)
         }

         md := &Markdown{} // Fresh state machine for each message
         fmt.Fprintf(w, "%s</div>\n", md.Render(msg.Content))
      }
   }

   if canFlush {
      flusher.Flush()
   }

   // Stream new AI response
   if r.Method == http.MethodPost {
      fmt.Fprint(w, `<div class="msg assistant">`)
      if canFlush {
         flusher.Flush()
      }

      onToken := func(text string) {
         fmt.Fprint(w, text)
         if canFlush {
            flusher.Flush()
         }
      }

      replyReasoning, replyContent, err := processChat(messages, apiKey, onToken)
      if err != nil {
         return fmt.Errorf("API error: %w", err)
      }

      fmt.Fprintln(w, "</div>")

      messages = append(messages, Message{
         Role:             "assistant",
         Content:          replyContent,
         ReasoningContent: replyReasoning,
      })

      newSessionData, err := json.MarshalIndent(messages, "", " ")
      if err != nil {
         return fmt.Errorf("error marshaling session data: %w", err)
      }

      if err := os.WriteFile(sessionFileName, newSessionData, 0644); err != nil {
         return fmt.Errorf("error writing session file: %w", err)
      }
   }

   fmt.Fprint(w, footerHTML)
   return nil
}

func processUploadedFile(fileHeader *multipart.FileHeader) (string, error) {
   file, err := fileHeader.Open()
   if err != nil {
      return "", fmt.Errorf("error opening uploaded file %s: %w", fileHeader.Filename, err)
   }
   defer file.Close()

   fileData, err := io.ReadAll(file)
   if err != nil {
      return "", fmt.Errorf("error reading uploaded file %s: %w", fileHeader.Filename, err)
   }

   return fmt.Sprintf("\n\nFile: %s\n```\n%s\n```\n", fileHeader.Filename, string(fileData)), nil
}

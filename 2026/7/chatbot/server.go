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

// handleRoot handles rendering the page (GET) and streaming new responses (POST)
func handleRoot(w http.ResponseWriter, r *http.Request, apiKey, headerHTML, footerHTML string) error {
   // 1. Load existing session
   var messages []Message
   sessionData, err := os.ReadFile(sessionFileName)
   if err != nil {
      log.Printf("error reading %s: %v", sessionFileName, err)
   } else if err := json.Unmarshal(sessionData, &messages); err != nil {
      return fmt.Errorf("critical error parsing %s: %w", sessionFileName, err)
   }

   // Initialize a new session with the formatting instructions
   if len(messages) == 0 {
      messages = append(messages, Message{
         Role:    "system",
         Content: "You are a helpful assistant. Please format your response using standard HTML tags (like <p>, <b>, <pre>, <code>, <ul>, <li>) instead of Markdown. Do not use markdown wrappers (like ```html).",
      })
   }

   // 2. Process new user input if it's a POST request
   if r.Method == http.MethodPost {
      r.ParseMultipartForm(10 << 20) // 10MB limit

      // Escape the user text and convert newlines to <br> so it renders perfectly in HTML
      userText := html.EscapeString(r.FormValue("text"))
      userText = strings.ReplaceAll(userText, "\n", "<br>")

      combinedInput := userText

      if files := r.MultipartForm.File["files"]; len(files) > 0 {
         for _, fileHeader := range files {
            fileChunk, err := processUploadedFile(fileHeader)
            if err != nil {
               return err
            }

            // Append directly. CSS margins will handle the spacing cleanly.
            combinedInput += fileChunk
         }
      }

      if combinedInput != "" {
         messages = append(messages, Message{Role: "user", Content: combinedInput})
      }
   }

   // 3. Start writing the HTML response
   w.Header().Set("Content-Type", "text/html; charset=utf-8")
   w.Header().Set("Cache-Control", "no-cache")
   flusher, canFlush := w.(http.Flusher)

   fmt.Fprint(w, headerHTML)

   // Render all historical messages
   for _, msg := range messages {
      if msg.Role == "system" {
         // Escape the system prompt so its literal HTML tags show up correctly on screen
         fmt.Fprintf(w, `<div class="msg %s">%s</div>`+"\n", msg.Role, html.EscapeString(msg.Content))
      } else {
         // User input is already escaped before appending. Assistant HTML should not be escaped.
         fmt.Fprintf(w, `<div class="msg %s">%s</div>`+"\n", msg.Role, msg.Content)
      }
   }

   if canFlush {
      flusher.Flush()
   }

   // 4. Stream the new AI response if it's a POST request
   if r.Method == http.MethodPost {
      fmt.Fprint(w, `<div class="msg assistant">`)
      if canFlush {
         flusher.Flush()
      }

      // Callback for streaming tokens natively to HTML
      onToken := func(text string) {
         // Since we requested HTML, we pass the AI's tokens through directly without escaping them!
         fmt.Fprint(w, text)
         if canFlush {
            flusher.Flush()
         }
      }

      reply, err := processChat(messages, apiKey, onToken)
      if err != nil {
         return fmt.Errorf("API error: %w", err)
      }

      fmt.Fprintln(w, "</div>")

      // Save updated session
      messages = append(messages, Message{Role: "assistant", Content: reply})

      newSessionData, err := json.MarshalIndent(messages, "", " ")
      if err != nil {
         return fmt.Errorf("error marshaling session data: %w", err)
      }

      if err := os.WriteFile(sessionFileName, newSessionData, 0644); err != nil {
         return fmt.Errorf("error writing session file: %w", err)
      }

      log.Printf("Writing %d items to %s", len(messages), sessionFileName)
   }

   // 5. Render footer and close connection
   fmt.Fprint(w, footerHTML)

   return nil
}

// processUploadedFile opens, reads, safely closes, and formats an uploaded file into an HTML block
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

   // Wrap file contents in an HTML <details> block
   safeData := html.EscapeString(string(fileData))
   safeName := html.EscapeString(fileHeader.Filename)

   return fmt.Sprintf("<details><summary>File: %s</summary><pre><code>%s</code></pre></details>", safeName, safeData), nil
}

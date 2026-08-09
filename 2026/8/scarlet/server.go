package scarlet

import (
   _ "embed"
   "encoding/json"
   "fmt"
   "html"
   "io"
   "log"
   "net/http"
   "os"
   "strings"
)

const serverAddress = "localhost:8080"
const sessionFileName = "session.json"

//go:embed favicon.svg
var faviconSVG string

//go:embed index.html
var indexHTML string

//go:embed style.css
var styleCSS string

// RunServer initializes the HTTP routes and starts the web server
func RunServer(cfg *AppConfig) error {
   headerHTML, footerHTML, found := strings.Cut(indexHTML, "<!-- CHAT_CONTENT -->")
   if !found {
      return fmt.Errorf("error: index.html is missing the <!-- CHAT_CONTENT --> marker")
   }

   http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
      w.Header().Set("Content-Type", "text/css")
      fmt.Fprint(w, styleCSS)
   })

   http.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
      w.Header().Set("Content-Type", "image/svg+xml")
      fmt.Fprint(w, faviconSVG)
   })

   http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      if err := handleRoot(w, r, cfg, headerHTML, footerHTML); err != nil {
         log.Printf("Handler error: %v", err)
      }
   })

   log.Printf("Starting local server at http://%s - Press Ctrl+C to stop", serverAddress)
   return http.ListenAndServe(serverAddress, nil)
}

func handleRoot(w http.ResponseWriter, r *http.Request, cfg *AppConfig, headerHTML, footerHTML string) error {
   var messages []Message
   sessionData, err := os.ReadFile(sessionFileName)
   if err != nil {
      log.Println(err)
   } else if err := json.Unmarshal(sessionData, &messages); err != nil {
      return fmt.Errorf("critical error parsing %s: %w", sessionFileName, err)
   }

   if r.Method == http.MethodPost {
      r.ParseMultipartForm(10 << 20) // 10MB limit

      /* strip CRLF to LF */
      userText := strings.ReplaceAll(r.FormValue("text"), "\r\n", "\n")
      var files []FileAttachment

      if fileHeaders := r.MultipartForm.File["files"]; len(fileHeaders) > 0 {
         for _, fh := range fileHeaders {
            file, err := fh.Open()
            if err != nil {
               return fmt.Errorf("error opening uploaded file %s: %w", fh.Filename, err)
            }
            fileData, err := io.ReadAll(file)
            file.Close()
            if err != nil {
               return fmt.Errorf("error reading uploaded file %s: %w", fh.Filename, err)
            }
            files = append(files, FileAttachment{
               Filename: fh.Filename,
               Content:  string(fileData),
            })
         }
      }

      if userText != "" || len(files) > 0 {
         messages = append(messages, Message{
            Role:    "user",
            Content: userText,
            Files:   files,
         })
      }
   }

   w.Header().Set("Content-Type", "text/html; charset=utf-8")
   w.Header().Set("Cache-Control", "no-cache")

   flusher, canFlush := w.(http.Flusher)

   fmt.Fprint(w, headerHTML)

   for _, msg := range messages {
      if msg.Role == "system" {
         fmt.Fprintf(w, `<div class="msg %s">%s</div>`+"\n", msg.Role, html.EscapeString(msg.Content))
      } else if msg.Role == "user" {
         if msg.ReasoningContent != "" {
            fmt.Fprintf(w, `<details class="reasoning"><summary>Reasoning</summary>%s</details>`, renderMarkdown(msg.ReasoningContent))
         }
         if msg.Content != "" {
            fmt.Fprintf(w, `<div class="user-text">%s</div>`+"\n", renderMarkdown(msg.Content))
         }
         for _, f := range msg.Files {
            fmt.Fprintf(w, "\n<details class=\"file-attachment\">\n<summary>%s</summary>\n<pre>%s</pre>\n</details>\n", html.EscapeString(f.Filename), html.EscapeString(strings.TrimRight(f.Content, "\n")))
         }
      } else {
         if msg.ReasoningContent != "" {
            fmt.Fprintf(w, `<details class="reasoning"><summary>Reasoning</summary>%s</details>`, renderMarkdown(msg.ReasoningContent))
         }
         if msg.Content != "" {
            fmt.Fprintf(w, `<div class="completion">%s</div>`+"\n", renderMarkdown(msg.Content))
         }
      }
   }

   if canFlush {
      flusher.Flush()
   }

   if r.Method == http.MethodPost {
      if canFlush {
         flusher.Flush()
      }

      onToken := func(text string) {
         fmt.Fprint(w, text)
         if canFlush {
            flusher.Flush()
         }
      }

      replyMsg, err := processChat(messages, cfg, onToken)
      if err != nil {
         return fmt.Errorf("API error: %w", err)
      }

      messages = append(messages, *replyMsg)

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

// server.go

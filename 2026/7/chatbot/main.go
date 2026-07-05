package main

import (
   _ "embed"
   "encoding/json"
   "flag"
   "fmt"
   "html"
   "io"
   "log"
   "net/http"
   "os"
   "path/filepath"
   "strings"
)

const sessionFileName = "session.json"

//go:embed index.html
var indexHTML string

// handleRoot handles rendering the page (GET) and streaming new responses (POST)
func handleRoot(w http.ResponseWriter, r *http.Request, apiKey, headerHTML, footerHTML string) {
   if r.URL.Path != "/" {
      http.NotFound(w, r)
      return
   }

   // 1. Load existing session
   var messages []Message
   sessionData, err := os.ReadFile(sessionFileName)
   if err != nil {
      log.Printf("error reading %s: %v", sessionFileName, err)
   } else if err := json.Unmarshal(sessionData, &messages); err != nil {
      log.Fatalf("critical error parsing %s: %v", sessionFileName, err)
   }

   // 2. Process new user input if it's a POST request
   if r.Method == http.MethodPost {
      r.ParseMultipartForm(10 << 20) // 10MB limit

      combinedInput := r.FormValue("text")

      if files := r.MultipartForm.File["files"]; len(files) > 0 {
         for _, fileHeader := range files {
            file, err := fileHeader.Open()
            if err != nil {
               log.Fatalf("error opening uploaded file %s: %v", fileHeader.Filename, err)
            }

            fileData, err := io.ReadAll(file)
            if err != nil {
               log.Fatalf("error reading uploaded file %s: %v", fileHeader.Filename, err)
            }
            file.Close()

            // Ensure there is a line break between text and files (or between multiple files)
            if combinedInput != "" {
               combinedInput += "\n"
            }
            combinedInput += fmt.Sprintf("### %s\n```\n%s\n```\n\n", fileHeader.Filename, fileData)
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
      fmt.Fprintf(w, "<div class=\"msg %s\">%s</div>\n", msg.Role, html.EscapeString(msg.Content))
   }

   if canFlush {
      flusher.Flush()
   }

   // 4. Stream the new AI response if it's a POST request
   if r.Method == http.MethodPost {
      fmt.Fprint(w, "<div class=\"msg assistant\">")
      if canFlush {
         flusher.Flush()
      }

      // Callback for streaming tokens natively to HTML
      onToken := func(text string, isReasoning bool) {
         safeText := html.EscapeString(text)
         if isReasoning {
            fmt.Fprintf(w, "<span class=\"reasoning\">%s</span>", safeText)
         } else {
            fmt.Fprintf(w, "%s", safeText)
         }

         if canFlush {
            flusher.Flush()
         }
      }

      reply, err := processChat(messages, apiKey, onToken)
      if err != nil {
         log.Fatalf("API Error: %v", err)
      }

      fmt.Fprint(w, "</div>\n")

      // Save updated session
      messages = append(messages, Message{Role: "assistant", Content: reply})

      newSessionData, err := json.MarshalIndent(messages, "", " ")
      if err != nil {
         log.Fatalf("Error marshaling session data: %v", err)
      }

      if err := os.WriteFile(sessionFileName, newSessionData, 0644); err != nil {
         log.Fatalf("Error writing session file: %v", err)
      }

      log.Printf("Writing %d items to %s", len(messages), sessionFileName)
   }

   // 5. Render footer and close connection
   fmt.Fprint(w, footerHTML)
}

func main() {
   log.SetFlags(log.Ltime)

   apiKeyFlag := flag.String("api-key", "", "Save the provided API key to your configuration directory")
   flag.Parse()

   if err := run(*apiKeyFlag); err != nil {
      log.Fatal(err)
   }
}

// run handles the configuration loading/saving and starts the web server
func run(apiKeyFlag string) error {
   // Split the embedded HTML explicitly using strings.Cut
   headerHTML, footerHTML, found := strings.Cut(indexHTML, "<!-- CHAT_CONTENT -->")
   if !found {
      return fmt.Errorf("error: index.html is missing the <!-- CHAT_CONTENT --> marker")
   }

   configDir, err := os.UserConfigDir()
   if err != nil {
      return fmt.Errorf("error getting user config directory: %w", err)
   }

   appConfigDir := filepath.Join(configDir, "chatbot")
   keyFilePath := filepath.Join(appConfigDir, "api-key")

   // If the user provided an API key flag, save it globally and exit
   if apiKeyFlag != "" {
      if err := os.MkdirAll(appConfigDir, 0700); err != nil {
         return fmt.Errorf("error creating config directory: %w", err)
      }
      if err := os.WriteFile(keyFilePath, []byte(apiKeyFlag), 0600); err != nil {
         return fmt.Errorf("error writing API key to file: %w", err)
      }
      log.Println("API key saved successfully.")
      return nil
   }

   // Read the API key from the global config file
   apiKeyBytes, err := os.ReadFile(keyFilePath)
   if err != nil {
      return fmt.Errorf("API key not found. Please run with '-api-key YOUR_KEY' first")
   }
   apiKey := string(apiKeyBytes)

   // Setup HTTP Server with a single route
   http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      handleRoot(w, r, apiKey, headerHTML, footerHTML)
   })

   port := ":8080"
   log.Printf("Starting local server at http://localhost%s - Press Ctrl+C to stop", port)
   return http.ListenAndServe(port, nil)
}

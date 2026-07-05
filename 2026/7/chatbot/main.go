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

//go:embed style.css
var styleCSS string

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

      var inputParts []string

      // Add the user text FIRST
      if userText != "" {
         inputParts = append(inputParts, userText)
      }

      // Then process and add attached files BELOW the user text
      if files := r.MultipartForm.File["files"]; len(files) > 0 {
         for _, fileHeader := range files {
            file, err := fileHeader.Open()
            if err != nil {
               return fmt.Errorf("error opening uploaded file %s: %w", fileHeader.Filename, err)
            }

            fileData, err := io.ReadAll(file)
            if err != nil {
               file.Close() // Clean up before returning error
               return fmt.Errorf("error reading uploaded file %s: %w", fileHeader.Filename, err)
            }

            if err := file.Close(); err != nil {
               return fmt.Errorf("error closing uploaded file %s: %w", fileHeader.Filename, err)
            }

            // Wrap file contents in an HTML <details> block
            safeData := html.EscapeString(string(fileData))
            safeName := html.EscapeString(fileHeader.Filename)
            inputParts = append(inputParts, fmt.Sprintf("<details><summary>File: %s</summary><pre><code>%s</code></pre></details>", safeName, safeData))
         }
      }

      combinedInput := strings.Join(inputParts, "<br><br>")

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

   // Setup HTTP Server
   http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
      w.Header().Set("Content-Type", "text/css")
      fmt.Fprint(w, styleCSS)
   })

   http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      if err := handleRoot(w, r, apiKey, headerHTML, footerHTML); err != nil {
         log.Fatalf("Handler error: %v", err)
      }
   })

   port := ":8080"
   log.Printf("Starting local server at http://localhost%s - Press Ctrl+C to stop", port)
   return http.ListenAndServe(port, nil)
}

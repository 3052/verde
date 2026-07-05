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

var headerHTML, footerHTML string

//go:embed index.html
var indexHTML string

// handleRoot handles rendering the page (GET) and streaming new responses (POST)
func handleRoot(w http.ResponseWriter, r *http.Request, apiKey string) {
   if r.URL.Path != "/" {
      http.NotFound(w, r)
      return
   }

   // 1. Load existing session
   var messages []Message
   sessionData, err := os.ReadFile(sessionFileName)
   if err == nil {
      json.Unmarshal(sessionData, &messages)
   }

   // 2. Process new user input if it's a POST request
   if r.Method == http.MethodPost {
      r.ParseMultipartForm(10 << 20) // 10MB limit

      userText := r.FormValue("text")
      combinedInput := ""

      if files := r.MultipartForm.File["files"]; len(files) > 0 {
         for _, fileHeader := range files {
            file, err := fileHeader.Open()
            if err != nil {
               continue
            }
            fileData, _ := io.ReadAll(file)
            file.Close()

            if combinedInput != "" {
               combinedInput += "\n"
            }
            combinedInput += fmt.Sprintf("### %s\n```\n%s\n```\n\n", fileHeader.Filename, fileData)
         }
      }

      combinedInput += userText
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
         continue
      }
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
         fmt.Fprintf(w, "<br><br><b>Error:</b> %s", html.EscapeString(err.Error()))
      }

      fmt.Fprint(w, "</div>\n")

      // Save updated session
      messages = append(messages, Message{Role: "assistant", Content: reply})
      newSessionData, _ := json.MarshalIndent(messages, "", " ")
      os.WriteFile(sessionFileName, newSessionData, 0644)
   }

   // 5. Render footer and close connection
   fmt.Fprint(w, footerHTML)
}

func init() {
   // Split the HTML file at the marker so we can stream chat content in the middle
   parts := strings.Split(indexHTML, "<!-- CHAT_CONTENT -->")
   if len(parts) == 2 {
      headerHTML = parts[0]
      footerHTML = parts[1]
   } else {
      log.Fatal("Error: index.html is missing the <!-- CHAT_CONTENT --> marker")
   }
}

func main() {
   log.SetFlags(log.Ltime)

   apiKeyFlag := flag.String("api-key", "", "Save the provided API key to your configuration directory")
   flag.Parse()

   configDir, err := os.UserConfigDir()
   if err != nil {
      log.Fatalf("error getting user config directory: %v", err)
   }

   appConfigDir := filepath.Join(configDir, "chatbot")
   keyFilePath := filepath.Join(appConfigDir, "api-key")

   // If the user provided an API key flag, save it globally and exit
   if *apiKeyFlag != "" {
      if err := os.MkdirAll(appConfigDir, 0700); err != nil {
         log.Fatalf("error creating config directory: %v", err)
      }
      if err := os.WriteFile(keyFilePath, []byte(*apiKeyFlag), 0600); err != nil {
         log.Fatalf("error writing API key to file: %v", err)
      }
      log.Println("API key saved successfully.")
      return
   }

   // Read the API key from the global config file
   apiKeyBytes, err := os.ReadFile(keyFilePath)
   if err != nil {
      log.Fatalf("API key not found. Please run with '-api-key YOUR_KEY' first")
   }
   apiKey := string(apiKeyBytes)

   // Setup HTTP Server with a single route
   http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      handleRoot(w, r, apiKey)
   })

   port := ":8080"
   log.Printf("Starting local server at http://localhost%s", port)
   if err := http.ListenAndServe(port, nil); err != nil {
      log.Fatal(err)
   }
}

package main

import (
   _ "embed"
   "encoding/json"
   "flag"
   "fmt"
   "io"
   "log"
   "net/http"
   "os"
   "path/filepath"
)

const sessionFileName = "session.json"

//go:embed index.html
var indexHTML []byte

// Handles a new message submission, file attachments, and streams the response
func handleChat(w http.ResponseWriter, r *http.Request, apiKey string) {
   if r.Method != http.MethodPost {
      http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
      return
   }

   // Parse multipart form (up to 10MB in memory)
   err := r.ParseMultipartForm(10 << 20)
   if err != nil {
      http.Error(w, "Failed to parse form", http.StatusBadRequest)
      return
   }

   userText := r.FormValue("text")
   combinedInput := ""

   // Process attached files
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

   if combinedInput == "" {
      http.Error(w, "Empty input", http.StatusBadRequest)
      return
   }

   // Load session
   var messages []Message
   sessionData, err := os.ReadFile(sessionFileName)
   if err == nil {
      json.Unmarshal(sessionData, &messages)
   }
   messages = append(messages, Message{Role: "user", Content: combinedInput})

   // Setup streaming headers
   w.Header().Set("Content-Type", "application/json")
   w.Header().Set("Cache-Control", "no-cache")
   w.Header().Set("Connection", "keep-alive")
   flusher, ok := w.(http.Flusher)
   if !ok {
      http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
      return
   }

   // Callback to push JSON chunks to the browser
   onToken := func(text string, isReasoning bool) {
      chunk := map[string]any{
         "text":         text,
         "is_reasoning": isReasoning,
      }
      b, _ := json.Marshal(chunk)
      fmt.Fprintf(w, "%s\n", b)
      flusher.Flush()
   }

   // Call API
   reply, err := processChat(messages, apiKey, onToken)
   if err != nil {
      log.Printf("API Error: %v", err)
      return
   }

   // Save session
   messages = append(messages, Message{Role: "assistant", Content: reply})
   newSessionData, _ := json.MarshalIndent(messages, "", " ")
   os.WriteFile(sessionFileName, newSessionData, 0644)
}

// Returns the current session history as JSON
func handleGetSession(w http.ResponseWriter, r *http.Request) {
   sessionData, err := os.ReadFile(sessionFileName)
   if err != nil {
      if os.IsNotExist(err) {
         w.Write([]byte("[]"))
         return
      }
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
   }
   w.Header().Set("Content-Type", "application/json")
   w.Write(sessionData)
}

// Serves the frontend HTML/JS interface
func handleIndex(w http.ResponseWriter, r *http.Request) {
   if r.URL.Path != "/" {
      http.NotFound(w, r)
      return
   }
   w.Header().Set("Content-Type", "text/html")
   w.Write(indexHTML)
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

   // Setup HTTP Server
   http.HandleFunc("/", handleIndex)
   http.HandleFunc("/api/session", handleGetSession)
   http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
      handleChat(w, r, apiKey)
   })

   port := ":8080"
   log.Printf("Starting local server at http://localhost%s", port)
   if err := http.ListenAndServe(port, nil); err != nil {
      log.Fatal(err)
   }
}

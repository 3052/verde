package main

import (
   _ "embed"
   "flag"
   "fmt"
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

   // Serve the CSS file
   http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
      w.Header().Set("Content-Type", "text/css")
      fmt.Fprint(w, styleCSS)
   })

   // Setup HTTP Server with the main chat route
   http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      if err := handleRoot(w, r, apiKey, headerHTML, footerHTML); err != nil {
         log.Fatalf("Handler error: %v", err)
      }
   })

   port := ":8080"
   log.Printf("Starting local server at http://localhost%s - Press Ctrl+C to stop", port)
   return http.ListenAndServe(port, nil)
}

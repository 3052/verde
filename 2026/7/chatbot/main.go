package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "os"
   "path/filepath"
)

const sessionFileName = "session.json"

func main() {
   // Set log output to show only the time (omits the date)
   log.SetFlags(log.Ltime)

   var filesFlag []string

   // Define and parse command-line flags
   flag.Func("f", "Include a file (can be used multiple times)", func(value string) error {
      filesFlag = append(filesFlag, value)
      return nil
   })

   inputFlag := flag.String("i", "", "Provide input text directly via flag")
   apiKeyFlag := flag.String("api-key", "", "Save the provided API key to your configuration directory")
   flag.Parse()

   // If no flags were provided, print usage and exit
   if flag.NFlag() == 0 {
      flag.Usage()
      return
   }

   if err := run(*inputFlag, *apiKeyFlag, filesFlag); err != nil {
      log.Fatal(err)
   }
}

// run handles the configuration loading/saving and orchestrates the chat process
func run(inputFlag, apiKeyFlag string, filesFlag []string) error {
   // Get the OS-specific user configuration directory for the API key
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
      log.Printf("Writing API key to %s\n", keyFilePath)
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

   // Read and combine all provided files
   var combinedInput string
   for _, file := range filesFlag {
      fileData, err := os.ReadFile(file)
      if err != nil {
         return fmt.Errorf("error reading file %s: %w", file, err)
      }
      combinedInput += fmt.Sprintf("--- File: %s ---\n%s\n\n", file, fileData)
   }

   // Append the user input flag (appending an empty string is a no-op)
   combinedInput += inputFlag

   if combinedInput == "" {
      return fmt.Errorf("no input provided. Please use the -i or -f flag")
   }

   // Load previous session history from the current directory if it exists
   var messages []Message
   sessionData, err := os.ReadFile(sessionFileName)
   if err == nil {
      if err := json.Unmarshal(sessionData, &messages); err != nil {
         return fmt.Errorf("error parsing %s: %w", sessionFileName, err)
      }
   }

   // Append the new user prompt to the history
   messages = append(messages, Message{Role: "user", Content: combinedInput})

   // Pass the full history to processChat, which returns the assistant's reply
   reply, err := processChat(messages, string(apiKeyBytes))
   if err != nil {
      return err
   }

   // Append the assistant's reply to the history
   messages = append(messages, Message{Role: "assistant", Content: reply})

   // Save the updated history back to the local session file
   newSessionData, err := json.MarshalIndent(messages, "", " ")
   if err != nil {
      return fmt.Errorf("error marshaling session data: %w", err)
   }

   log.Printf("Writing %d items to %s", len(messages), sessionFileName)
   if err := os.WriteFile(sessionFileName, newSessionData, 0644); err != nil {
      return fmt.Errorf("error writing session file: %w", err)
   }

   return nil
}

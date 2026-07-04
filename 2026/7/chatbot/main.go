package main

import (
   "flag"
   "fmt"
   "log"
   "os"
   "path/filepath"
)

func main() {
   var filesFlag stringSlice

   // Define and parse command-line flags
   flag.Var(&filesFlag, "f", "Include a file (can be used multiple times)")
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
   // Get the OS-specific user configuration directory
   configDir, err := os.UserConfigDir()
   if err != nil {
      return fmt.Errorf("error getting user config directory: %w", err)
   }

   // Use "chatbot" for the configuration folder
   appConfigDir := filepath.Join(configDir, "chatbot")
   keyFilePath := filepath.Join(appConfigDir, "apikey")

   // If the user provided an API key flag, save it and exit
   if apiKeyFlag != "" {
      // Create the directory if it doesn't exist (permissions 0700 for privacy)
      if err := os.MkdirAll(appConfigDir, 0700); err != nil {
         return fmt.Errorf("error creating config directory: %w", err)
      }

      // Write the key to the file (permissions 0600 so only the user can read it)
      log.Printf("Writing API key to %s\n", keyFilePath)
      if err := os.WriteFile(keyFilePath, []byte(apiKeyFlag), 0600); err != nil {
         return fmt.Errorf("error writing API key to file: %w", err)
      }

      fmt.Println("API key saved successfully.")
      return nil
   }

   // Read the API key from the config file since it was not provided via flag
   data, err := os.ReadFile(keyFilePath)
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
      combinedInput += fmt.Sprintf("--- File: %s ---\n%s\n\n", file, string(fileData))
   }

   // Append the user input flag if provided
   if inputFlag != "" {
      combinedInput += inputFlag
   }

   // Pass the combined input and the API key to processChat
   return processChat(combinedInput, string(data))
}

// stringSlice allows a flag to be used multiple times
type stringSlice []string

func (s *stringSlice) Set(value string) error {
   *s = append(*s, value)
   return nil
}

func (s *stringSlice) String() string {
   return fmt.Sprintf("%v", *s)
}

package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "net/http"
   "os"
   "path/filepath"
)

const apiURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"

func main() {
   var filesFlag stringSlice

   // Define and parse command-line flags
   flag.Var(&filesFlag, "f", "Include a file (can be used multiple times)")
   inputFlag := flag.String("i", "", "Provide input text directly via flag")
   apiKeyFlag := flag.String("api-key", "", "Save the provided API key to your configuration directory")
   flag.Parse()

   if err := run(*inputFlag, *apiKeyFlag, filesFlag); err != nil {
      log.Fatal(err)
   }
}

// processChat handles the API request and response printing.
// It returns an error if any step of the process fails.
func processChat(userInput, apiKey string) error {
   if userInput == "" {
      return fmt.Errorf("no input provided. Please use the -i or -f flag")
   }

   messages := []Message{
      {Role: "system", Content: "You are a helpful assistant."},
      {Role: "user", Content: userInput},
   }

   // Build the raw API payload
   payload := map[string]any{
      "model":    "glm-5.2",
      "messages": messages,
   }

   body, err := json.Marshal(payload)
   if err != nil {
      return fmt.Errorf("marshaling JSON payload: %w", err)
   }

   // Send direct HTTP request to Z.ai
   req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
   if err != nil {
      return fmt.Errorf("creating HTTP request: %w", err)
   }

   req.Header.Set("Content-Type", "application/json")
   req.Header.Set("Authorization", "Bearer "+apiKey)

   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return fmt.Errorf("executing HTTP request: %w", err)
   }
   defer resp.Body.Close()

   // Decode the JSON response directly from the stream
   var result ChatResponse
   if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
      return fmt.Errorf("decoding response JSON: %w", err)
   }

   // Validate API response content
   if len(result.Choices) == 0 {
      return fmt.Errorf("API error: unexpected or empty choices array")
   }

   reply := result.Choices[0].Message.Content
   fmt.Printf("\nZ.ai:\n%s\n", reply)

   return nil
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

type ChatResponse struct {
   Choices []Choice `json:"choices"`
}

type Choice struct {
   Message Message `json:"message"`
}

// Structs for API interaction
type Message struct {
   Role    string `json:"role"`
   Content string `json:"content"`
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

package main

import (
   "bufio"
   "bytes"
   "encoding/json"
   "fmt"
   "log"
   "net/http"
   "strings"
)

const apiURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"

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

   // Build the raw API payload with stream set to true
   payload := map[string]any{
      "model":    "glm-5.2",
      "messages": messages,
      "stream":   true,
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
   req.Header.Set("Accept", "text/event-stream")

   log.Printf("Executing HTTP request to %s...\n", apiURL)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return fmt.Errorf("executing HTTP request: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
   }

   fmt.Printf("\nZ.ai:\n")

   // Read the streaming response line by line
   scanner := bufio.NewScanner(resp.Body)
   for scanner.Scan() {
      line := scanner.Text()

      // The API sends empty lines as spacers, ignore them
      if line == "" {
         continue
      }

      // Streaming data lines start with "data: "
      if strings.HasPrefix(line, "data: ") {
         dataStr := strings.TrimPrefix(line, "data: ")

         // "[DONE]" signals the end of the stream
         if dataStr == "[DONE]" {
            break
         }

         var streamResp StreamResponse
         if err := json.Unmarshal([]byte(dataStr), &streamResp); err != nil {
            // If we can't parse a chunk, continue reading the rest
            continue
         }

         if len(streamResp.Choices) > 0 {
            content := streamResp.Choices[0].Delta.Content
            fmt.Print(content)
         }
      }
   }

   fmt.Println() // Print a final newline when the stream is finished

   if err := scanner.Err(); err != nil {
      return fmt.Errorf("error reading stream: %w", err)
   }

   return nil
}

// Structs for API interaction
type Message struct {
   Role    string `json:"role"`
   Content string `json:"content"`
}

type StreamChoice struct {
   Delta StreamDelta `json:"delta"`
}

type StreamDelta struct {
   Content string `json:"content"`
}

// Streaming response structs
type StreamResponse struct {
   Choices []StreamChoice `json:"choices"`
}

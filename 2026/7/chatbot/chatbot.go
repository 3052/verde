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

// ANSI color codes for terminal output
const (
   colorYellow = "\033[33m"
   colorReset  = "\033[0m"
)

const apiURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"

// processChat handles the API request and streams the response to the terminal.
// It now takes the full message history and returns the generated reply string.
func processChat(messages []Message, apiKey string) (string, error) {
   // Build the raw API payload with stream set to true
   payload := map[string]any{
      "model":    "glm-5.2",
      "messages": messages,
      "stream":   true,
   }

   body, err := json.Marshal(payload)
   if err != nil {
      return "", fmt.Errorf("marshaling JSON payload: %w", err)
   }

   // Send direct HTTP request to Z.ai
   req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
   if err != nil {
      return "", fmt.Errorf("creating HTTP request: %w", err)
   }

   req.Header.Set("Content-Type", "application/json")
   req.Header.Set("Authorization", "Bearer "+apiKey)
   req.Header.Set("Accept", "text/event-stream")

   log.Printf("POST %s", apiURL)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return "", fmt.Errorf("executing HTTP request: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return "", fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
   }

   var fullReply string
   var printedReasoning bool
   var transitionedToContent bool

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
         line = strings.TrimPrefix(line, "data: ")

         // "[DONE]" signals the end of the stream
         if line == "[DONE]" {
            break
         }

         var streamResp StreamResponse
         if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
            // A complete line starting with "data: " should always be valid JSON.
            // If it fails, return the error instead of silently ignoring it.
            return "", fmt.Errorf("error unmarshaling stream chunk: %w\nRaw line: %s", err, line)
         }

         // Iterate over all choices just in case the API sends multiple or none
         for _, choice := range streamResp.Choices {
            // Print reasoning tokens (Chain-of-Thought) in yellow if the model provides them
            if choice.Delta.ReasoningContent != "" {
               printedReasoning = true
               fmt.Print(colorYellow + choice.Delta.ReasoningContent + colorReset)
            }

            // Print and save the actual final response content
            if choice.Delta.Content != "" {
               // If we just finished thinking, add a visual break before the final answer
               if printedReasoning && !transitionedToContent {
                  fmt.Print("\n\n")
                  transitionedToContent = true
               }

               content := choice.Delta.Content
               fmt.Print(content)
               fullReply += content // Append to the complete string
            }
         }
      }
   }

   fmt.Println() // Print a final newline when the stream is finished

   if err := scanner.Err(); err != nil {
      return "", fmt.Errorf("error reading stream: %w", err)
   }

   return fullReply, nil
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
   Content          string `json:"content"`
   ReasoningContent string `json:"reasoning_content"`
}

// Streaming response structs
type StreamResponse struct {
   Choices []StreamChoice `json:"choices"`
}

package main

import (
   "bufio"
   "bytes"
   "encoding/json"
   "fmt"
   "io"
   "net/http"
   "os"
   "strings"
)

const (
   apiKey = "YOUR_Z_AI_API_KEY" // <--- PASTE YOUR KEY HERE
   apiURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
)

func main() {
   // History array to keep your massive JSON in memory for the cache discount
   messages := []Message{
      {Role: "system", Content: "You are a helpful assistant."},
   }

   fmt.Println("--- Z.ai Minimal Go Chat ---")
   fmt.Println("Paste your JSON/text. Type 'SEND' on a new line to submit, or 'EXIT' to quit.")

   scanner := bufio.NewScanner(os.Stdin)

   for {
      fmt.Print("\nYou:\n")
      var lines []string

      // This inner loop allows you to safely paste massive multi-line JSON files
      for scanner.Scan() {
         line := scanner.Text()
         trimLine := strings.ToUpper(strings.TrimSpace(line))
         if trimLine == "SEND" {
            break
         }
         if trimLine == "EXIT" {
            return
         }
         lines = append(lines, line)
      }

      userInput := strings.Join(lines, "\n")
      if strings.TrimSpace(userInput) == "" {
         continue
      }

      // Add user prompt to history
      messages = append(messages, Message{Role: "user", Content: userInput})

      // Build the raw API payload
      payload := map[string]interface{}{
         "model":    "glm-5.2",
         "messages": messages,
      }
      body, _ := json.Marshal(payload)

      // Send direct HTTP request to Z.ai
      req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
      req.Header.Set("Content-Type", "application/json")
      req.Header.Set("Authorization", "Bearer "+apiKey)

      resp, err := (&http.Client{}).Do(req)
      if err != nil {
         fmt.Println("Connection Error:", err)
         messages = messages[:len(messages)-1] // Remove failed message so you can try again
         continue
      }

      respBody, _ := io.ReadAll(resp.Body)
      resp.Body.Close()

      // Parse the response
      var result map[string]interface{}
      json.Unmarshal(respBody, &result)

      if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
         if msg, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{}); ok {
            reply := msg["content"].(string)
            fmt.Printf("\nZ.ai:\n%s\n", reply)

            // Save Z.ai's reply to history to maintain cache flow
            messages = append(messages, Message{Role: "assistant", Content: reply})
         }
      } else {
         fmt.Println("API Error:", string(respBody))
         messages = messages[:len(messages)-1]
      }
   }
}

type Message struct {
   Role    string `json:"role"`
   Content string `json:"content"`
}

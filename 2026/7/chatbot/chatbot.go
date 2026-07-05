package main

import (
   "bufio"
   "bytes"
   "encoding/json"
   "fmt"
   "html"
   "log"
   "net/http"
   "strings"
)

const apiURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"

// processChat calls the API and streams tokens back via the onToken callback.
func processChat(messages []Message, apiKey string, onToken func(text string)) (string, error) {
   payload := map[string]any{
      "model":    "glm-5.2",
      "messages": messages,
      "stream":   true,
   }

   body, err := json.Marshal(payload)
   if err != nil {
      return "", fmt.Errorf("marshaling JSON payload: %w", err)
   }

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

   scanner := bufio.NewScanner(resp.Body)

   for scanner.Scan() {
      line := scanner.Text()

      if line == "" {
         continue
      }

      if strings.HasPrefix(line, "data: ") {
         line = strings.TrimPrefix(line, "data: ")

         if line == "[DONE]" {
            break
         }

         var streamResp StreamResponse
         if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
            return "", fmt.Errorf("error unmarshaling stream chunk: %w\nRaw line: %s", err, line)
         }

         for _, choice := range streamResp.Choices {
            // Send reasoning tokens wrapped in a dedicated div
            if choice.Delta.ReasoningContent != "" {
               if !printedReasoning {
                  tag := "<div class=\"reasoning\">"
                  if onToken != nil {
                     onToken(tag)
                  }
                  fullReply += tag
                  printedReasoning = true
               }

               // We still escape reasoning content because the model assumes it's internal plain text
               safeRc := html.EscapeString(choice.Delta.ReasoningContent)
               if onToken != nil {
                  onToken(safeRc)
               }
               fullReply += safeRc
            }

            // Send actual content tokens (native HTML allowed)
            if choice.Delta.Content != "" {
               // Cap off the reasoning div with an <hr> separator before the final answer
               if printedReasoning && !transitionedToContent {
                  tag := "</div><hr>"
                  if onToken != nil {
                     onToken(tag)
                  }
                  fullReply += tag
                  transitionedToContent = true
               }

               content := choice.Delta.Content
               if onToken != nil {
                  onToken(content)
               }
               fullReply += content
            }
         }
      }
   }

   // Just in case it stopped generating before answering, close the reasoning div
   if printedReasoning && !transitionedToContent {
      tag := "</div>"
      if onToken != nil {
         onToken(tag)
      }
      fullReply += tag
   }

   if err := scanner.Err(); err != nil {
      return "", fmt.Errorf("error reading stream: %w", err)
   }

   return fullReply, nil
}

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

type StreamResponse struct {
   Choices []StreamChoice `json:"choices"`
}

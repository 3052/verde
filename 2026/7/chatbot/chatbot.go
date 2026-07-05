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

const (
   apiURL           = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
   reasoningStart   = `<div class="reasoning">`
   reasoningEnd     = `</div>`
   reasoningEndLine = `</div><hr>`
)

// processChat calls the API and streams tokens back via the onToken callback.
func processChat(messages []Message, apiKey string, onToken func(text string)) (string, error) {
   payload := map[string]any{
      "model":    "glm-5.2",
      "messages": messages,
      "stream":   true,
      "stream_options": map[string]any{
         "include_usage": true,
      },
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
                  if onToken != nil {
                     onToken(reasoningStart)
                  }
                  fullReply += reasoningStart
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
                  if onToken != nil {
                     onToken(reasoningEndLine)
                  }
                  fullReply += reasoningEndLine
                  transitionedToContent = true
               }

               content := choice.Delta.Content
               if onToken != nil {
                  onToken(content)
               }
               fullReply += content
            }
         }

         // Check if this chunk includes usage statistics
         if streamResp.Usage != nil && (streamResp.Usage.TotalTokens > 0 || streamResp.Usage.PromptTokens > 0) {
            // Cap off reasoning if the stream ended abruptly before generating standard text
            if printedReasoning && !transitionedToContent {
               if onToken != nil {
                  onToken(reasoningEndLine)
               }
               fullReply += reasoningEndLine
               transitionedToContent = true
            }

            stats := fmt.Sprintf(`<div class="token-stats">Tokens: %d prompt (%d cached) | %d completion | %d total</div>`,
               streamResp.Usage.PromptTokens,
               streamResp.Usage.PromptTokensDetails.CachedTokens,
               streamResp.Usage.CompletionTokens,
               streamResp.Usage.TotalTokens,
            )

            if onToken != nil {
               onToken(stats)
            }

            // Append to fullReply so it persists in the chat history across page refreshes
            fullReply += stats
         }
      }
   }

   // Just in case it stopped generating before answering, close the reasoning div
   if printedReasoning && !transitionedToContent {
      if onToken != nil {
         onToken(reasoningEnd)
      }
      fullReply += reasoningEnd
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

type PromptTokensDetails struct {
   CachedTokens int `json:"cached_tokens"`
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
   Usage   *Usage         `json:"usage,omitempty"`
}

type Usage struct {
   PromptTokens        int                 `json:"prompt_tokens"`
   CompletionTokens    int                 `json:"completion_tokens"`
   TotalTokens         int                 `json:"total_tokens"`
   PromptTokensDetails PromptTokensDetails `json:"prompt_tokens_details"`
}

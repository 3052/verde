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

func processChat(messages []Message, apiKey string, onToken func(text string)) (string, string, error) {
   payload := map[string]any{
      "model":          "glm-5.2",
      "messages":       messages,
      "stream":         true,
      "stream_options": map[string]bool{"include_usage": true},
   }

   body, err := json.Marshal(payload)
   if err != nil {
      return "", "", fmt.Errorf("marshaling JSON payload: %w", err)
   }

   req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
   if err != nil {
      return "", "", fmt.Errorf("creating HTTP request: %w", err)
   }

   req.Header.Set("Content-Type", "application/json")
   req.Header.Set("Authorization", "Bearer "+apiKey)
   req.Header.Set("Accept", "text/event-stream")

   log.Printf("POST %s", apiURL)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return "", "", fmt.Errorf("executing HTTP request: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return "", "", fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
   }

   var fullReasoning strings.Builder
   var fullContent strings.Builder
   var contentBuf string

   var printedReasoning bool
   var transitionedToContent bool
   md := &Markdown{} // Stateful stream parser

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
            return "", "", fmt.Errorf("error unmarshaling stream chunk: %w\nRaw line: %s", err, line)
         }

         for _, choice := range streamResp.Choices {
            if choice.Delta.ReasoningContent != "" {
               if !printedReasoning {
                  if onToken != nil {
                     onToken(`<div class="reasoning">`)
                  }
                  printedReasoning = true
               }

               fullReasoning.WriteString(choice.Delta.ReasoningContent)

               // Reasoning is plain text, so we stream it directly
               safeRc := html.EscapeString(choice.Delta.ReasoningContent)
               safeRc = strings.ReplaceAll(safeRc, "\n", "<br>")
               if onToken != nil {
                  onToken(safeRc)
               }
            }

            if choice.Delta.Content != "" {
               if printedReasoning && !transitionedToContent {
                  if onToken != nil {
                     onToken(`</div><hr>`)
                  }
                  transitionedToContent = true
               }

               fullContent.WriteString(choice.Delta.Content)
               contentBuf += choice.Delta.Content

               // Line-buffer the content so Markdown can parse headers/blocks securely
               for {
                  idx := strings.IndexByte(contentBuf, '\n')
                  if idx == -1 {
                     break
                  }
                  lineChunk := contentBuf[:idx]
                  contentBuf = contentBuf[idx+1:]

                  htmlStr := md.RenderLine(lineChunk)
                  if md.inCodeBlock {
                     if onToken != nil {
                        onToken(htmlStr + "\n")
                     }
                  } else {
                     if onToken != nil {
                        onToken(htmlStr + "<br>")
                     }
                  }
               }
            }
         }

         if streamResp.Usage != nil && streamResp.Usage.PromptTokens > 0 {
            if printedReasoning && !transitionedToContent {
               if onToken != nil {
                  onToken(`</div><hr>`)
               }
               transitionedToContent = true
            }

            // Flush whatever is left in the buffer before stream closes
            if contentBuf != "" {
               if onToken != nil {
                  onToken(md.RenderLine(contentBuf))
               }
               contentBuf = ""
            }
            if md.inCodeBlock {
               if onToken != nil {
                  onToken("</pre>")
               }
            }

            stats := fmt.Sprintf(`<div class="token-stats">Input Tokens: %d (%d cached)</div>`,
               streamResp.Usage.PromptTokens,
               streamResp.Usage.PromptTokensDetails.CachedTokens,
            )

            if onToken != nil {
               onToken(stats)
            }
         }
      }
   }

   // Fallback closures if stream stopped abruptly
   if printedReasoning && !transitionedToContent {
      if onToken != nil {
         onToken(`</div>`)
      }
   }
   if contentBuf != "" {
      if onToken != nil {
         onToken(md.RenderLine(contentBuf))
      }
   }
   if md.inCodeBlock {
      if onToken != nil {
         onToken("</pre>")
      }
   }

   if err := scanner.Err(); err != nil {
      return "", "", fmt.Errorf("error reading stream: %w", err)
   }

   return fullReasoning.String(), fullContent.String(), nil
}

// Note: ReasoningContent added so HTML structure isn't saved to DB
type Message struct {
   Role             string `json:"role"`
   Content          string `json:"content"`
   ReasoningContent string `json:"reasoning_content,omitempty"`
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

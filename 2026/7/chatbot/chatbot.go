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
   var reasoningBuf string
   var contentBuf string

   var printedReasoning bool
   var transitionedToContent bool

   reasoningMd := &Markdown{}
   contentMd := &Markdown{}

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
            // Stream and parse Reasoning strictly through Markdown Engine
            if choice.Delta.ReasoningContent != "" {
               if !printedReasoning {
                  if onToken != nil {
                     onToken(`<div class="reasoning">`)
                  }
                  printedReasoning = true
               }

               fullReasoning.WriteString(choice.Delta.ReasoningContent)
               reasoningBuf += choice.Delta.ReasoningContent

               for {
                  idx := strings.IndexByte(reasoningBuf, '\n')
                  if idx == -1 {
                     break
                  }
                  lineChunk := reasoningBuf[:idx]
                  reasoningBuf = reasoningBuf[idx+1:]
                  if onToken != nil {
                     onToken(reasoningMd.RenderLine(lineChunk))
                  }
               }
            }

            // Stream and parse Content strictly through Markdown Engine
            if choice.Delta.Content != "" {
               if printedReasoning && !transitionedToContent {
                  if onToken != nil {
                     onToken(`</div><hr>`)
                  }
                  transitionedToContent = true
               }

               fullContent.WriteString(choice.Delta.Content)
               contentBuf += choice.Delta.Content

               for {
                  idx := strings.IndexByte(contentBuf, '\n')
                  if idx == -1 {
                     break
                  }
                  lineChunk := contentBuf[:idx]
                  contentBuf = contentBuf[idx+1:]
                  if onToken != nil {
                     onToken(contentMd.RenderLine(lineChunk))
                  }
               }
            }
         }

         if streamResp.Usage != nil && streamResp.Usage.PromptTokens > 0 {
            // Flush remaining reasoning buffers
            if printedReasoning && !transitionedToContent {
               if reasoningBuf != "" {
                  if onToken != nil {
                     onToken(reasoningMd.RenderLine(reasoningBuf))
                  }
                  reasoningBuf = ""
               }
               if reasoningMd.inList {
                  if onToken != nil {
                     onToken("</ul>")
                  }
               }
               if reasoningMd.inCodeBlock {
                  if onToken != nil {
                     onToken("</pre>")
                  }
               }
               if onToken != nil {
                  onToken(`</div><hr>`)
               }
               transitionedToContent = true
            }

            // Flush remaining content buffers
            if contentBuf != "" {
               if onToken != nil {
                  onToken(contentMd.RenderLine(contentBuf))
               }
               contentBuf = ""
            }
            if contentMd.inList {
               if onToken != nil {
                  onToken("</ul>")
               }
            }
            if contentMd.inCodeBlock {
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

   // Safety closures if stream stops abruptly
   if printedReasoning && !transitionedToContent {
      if onToken != nil {
         onToken(`</div>`)
      }
   }
   if contentBuf != "" {
      if onToken != nil {
         onToken(contentMd.RenderLine(contentBuf))
      }
   }
   if contentMd.inList {
      if onToken != nil {
         onToken("</ul>")
      }
   }
   if contentMd.inCodeBlock {
      if onToken != nil {
         onToken("</pre>")
      }
   }

   if err := scanner.Err(); err != nil {
      return "", "", fmt.Errorf("error reading stream: %w", err)
   }

   return fullReasoning.String(), fullContent.String(), nil
}

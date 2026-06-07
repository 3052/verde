package main

import (
   "encoding/json"
   "fmt"
   "net/url"
   "os"
   "sort"
   "strings"
)

type HAR struct {
   Log struct {
      Version string            `json:"version"`
      Creator interface{}       `json:"creator"`
      Pages   interface{}       `json:"pages,omitempty"`
      Entries []json.RawMessage `json:"entries"`
   } `json:"log"`
}

type PartialEntry struct {
   StartedDateTime string `json:"startedDateTime"`
   Request         struct {
      URL      string `json:"url"`
      PostData struct {
         MimeType string          `json:"mimeType"`
         Text     string          `json:"text"`
         Params   []NameValuePair `json:"params"`
      } `json:"postData"`
      Headers     []NameValuePair `json:"headers"`
      Cookies     []NameValuePair `json:"cookies"`
      QueryString []NameValuePair `json:"queryString"`
   } `json:"request"`
   Response struct {
      Content struct {
         Text string `json:"text"`
      } `json:"content"`
      Headers []NameValuePair `json:"headers"`
      Cookies []NameValuePair `json:"cookies"`
   } `json:"response"`
}

type NameValuePair struct {
   Name  string `json:"name"`
   Value string `json:"value"`
}

func processHAR(inputFile, targetTime, outputFile string) error {
   data, err := os.ReadFile(inputFile)
   if err != nil {
      return fmt.Errorf("error reading file: %w", err)
   }

   var har HAR
   if err := json.Unmarshal(data, &har); err != nil {
      return fmt.Errorf("error parsing HAR JSON: %w", err)
   }

   entries := make([]PartialEntry, len(har.Log.Entries))
   targetIdx := -1

   for i, rawEntry := range har.Log.Entries {
      if err := json.Unmarshal(rawEntry, &entries[i]); err != nil {
         fmt.Printf("Error parsing entry %d: %v\n", i, err)
         continue
      }
      if entries[i].StartedDateTime == targetTime {
         targetIdx = i
      }
   }

   if targetIdx == -1 {
      return fmt.Errorf("target request with the specified startedDateTime not found")
   }

   queue := []int{targetIdx}
   kept := map[int]bool{targetIdx: true}

   fmt.Printf("Target found at index %d. Tracing dependencies...\n", targetIdx)

   for len(queue) > 0 {
      currIdx := queue[0]
      queue = queue[1:]

      tokens := extractPotentialTokens(&entries[currIdx])

      for _, token := range tokens {
         for i := currIdx - 1; i >= 0; i-- {
            if kept[i] {
               continue
            }

            if responseContainsToken(&entries[i], token) {
               fmt.Printf("  -> Found origin of token '%s...' in request idx %d\n", token[:min(10, len(token))], i)
               kept[i] = true
               queue = append(queue, i)
               break
            }
         }
      }
   }

   var keptIndices []int
   for idx := range kept {
      keptIndices = append(keptIndices, idx)
   }
   sort.Ints(keptIndices)

   var filteredEntries []json.RawMessage
   for _, idx := range keptIndices {
      filteredEntries = append(filteredEntries, har.Log.Entries[idx])
   }

   har.Log.Entries = filteredEntries

   outData, err := json.MarshalIndent(har, "", "  ")
   if err != nil {
      return fmt.Errorf("error marshalling output: %w", err)
   }

   if err := os.WriteFile(outputFile, outData, 0644); err != nil {
      return fmt.Errorf("error writing to output file: %w", err)
   }

   fmt.Printf("\nSuccess! Kept %d out of %d requests. Saved to %s\n", len(filteredEntries), len(entries), outputFile)
   return nil
}

func extractPotentialTokens(req *PartialEntry) []string {
   tokenMap := make(map[string]bool)

   addToken := func(s string) {
      if len(s) >= 8 {
         tokenMap[s] = true
      }
   }

   for _, h := range req.Request.Headers {
      val := strings.TrimPrefix(h.Value, "Bearer ")
      addToken(val)
   }

   for _, c := range req.Request.Cookies {
      addToken(c.Value)
   }
   for _, q := range req.Request.QueryString {
      addToken(q.Value)
   }

   for _, p := range req.Request.PostData.Params {
      addToken(p.Value)
   }

   text := req.Request.PostData.Text
   if text != "" {
      var jsonData interface{}
      if err := json.Unmarshal([]byte(text), &jsonData); err == nil {
         extractJSONValues(jsonData, addToken)
      } else if strings.Contains(text, "=") {
         if parsedForm, err := url.ParseQuery(text); err == nil {
            for _, vals := range parsedForm {
               for _, v := range vals {
                  addToken(v)
               }
            }
         }
      }
   }

   if parsedURL, err := url.Parse(req.Request.URL); err == nil {
      segments := strings.Split(parsedURL.Path, "/")
      for _, segment := range segments {
         addToken(segment)
      }
   }

   var tokens []string
   for k := range tokenMap {
      tokens = append(tokens, k)
   }
   return tokens
}

func extractJSONValues(v interface{}, addToken func(string)) {
   switch val := v.(type) {
   case map[string]interface{}:
      for _, child := range val {
         extractJSONValues(child, addToken)
      }
   case []interface{}:
      for _, child := range val {
         extractJSONValues(child, addToken)
      }
   case string:
      addToken(val)
   case float64:
      addToken(fmt.Sprintf("%v", val))
   }
}

func responseContainsToken(res *PartialEntry, token string) bool {
   if strings.Contains(res.Response.Content.Text, token) {
      return true
   }
   for _, h := range res.Response.Headers {
      if strings.Contains(h.Value, token) {
         return true
      }
   }
   for _, c := range res.Response.Cookies {
      if strings.Contains(c.Value, token) {
         return true
      }
   }
   return false
}

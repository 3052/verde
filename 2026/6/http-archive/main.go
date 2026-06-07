package main

import (
   "encoding/json"
   "flag"
   "fmt"
   "os"
   "sort"
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

func main() {
   inputFile := flag.String("in", "", "Path to the input .har file (required)")
   targetTime := flag.String("time", "", "startedDateTime value of the target request (required)")
   outputFile := flag.String("out", "", "Path to the output .har file (required)")

   flag.Parse()

   if *inputFile == "" || *targetTime == "" || *outputFile == "" {
      fmt.Println("Error: Missing required arguments.")
      fmt.Println("Usage:")
      flag.PrintDefaults()
      os.Exit(1)
   }

   if err := processHAR(*inputFile, *targetTime, *outputFile); err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
      os.Exit(1)
   }
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

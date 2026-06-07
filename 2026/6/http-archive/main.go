package main

import (
   "encoding/json"
   "flag"
   "fmt"
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
         MimeType string `json:"mimeType"`
         Text     string `json:"text"`
      } `json:"postData"`
      Headers     []NameValuePair `json:"headers"`
      QueryString []NameValuePair `json:"queryString"`
   } `json:"request"`
   Response struct {
      Content struct {
         Text string `json:"text"`
      } `json:"content"`
      Headers []NameValuePair `json:"headers"`
   } `json:"response"`
}

type NameValuePair struct {
   Name  string `json:"name"`
   Value string `json:"value"`
}

// stringSlice allows accepting multiple flags of the same name (e.g. -req-header A -req-header B)
type stringSlice []string

func (s *stringSlice) String() string {
   return strings.Join(*s, ", ")
}

func (s *stringSlice) Set(value string) error {
   *s = append(*s, value)
   return nil
}

type TraceConfig struct {
   TargetTime string
   InputFile  string
   OutputFile string
   Headers    []string
   Queries    []string
   JSONKeys   []string
}

func main() {
   var config TraceConfig
   var headers, queries, jsonKeys stringSlice

   flag.StringVar(&config.InputFile, "in", "", "Path to the input .har file (required)")
   flag.StringVar(&config.TargetTime, "time", "", "startedDateTime value of the target request (required)")

   flag.Var(&headers, "req-header", "Name of a Request Header to trace (can be specified multiple times)")
   flag.Var(&queries, "req-query", "Name of a URL Query Parameter to trace (can be specified multiple times)")
   flag.Var(&jsonKeys, "req-json", "Name of a JSON body key to trace (can be specified multiple times)")

   flag.Parse()

   if config.InputFile == "" || config.TargetTime == "" {
      fmt.Fprintln(os.Stderr, "Error: Missing required arguments.")
      fmt.Fprintln(os.Stderr, "Usage:")
      flag.PrintDefaults()
      os.Exit(1)
   }

   config.Headers = headers
   config.Queries = queries
   config.JSONKeys = jsonKeys
   config.OutputFile = "output.har"

   // Passed as a pointer
   if err := processHAR(&config); err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
      os.Exit(1)
   }
}

// Accepts *TraceConfig
func processHAR(cfg *TraceConfig) error {
   data, err := os.ReadFile(cfg.InputFile)
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
      if entries[i].StartedDateTime == cfg.TargetTime {
         targetIdx = i
      }
   }

   if targetIdx == -1 {
      return fmt.Errorf("target request with the specified startedDateTime not found")
   }

   queue := []int{targetIdx}
   kept := map[int]bool{targetIdx: true}

   fmt.Printf("Target found at index %d. Extracting configured keys and tracing...\n", targetIdx)

   for len(queue) > 0 {
      currIdx := queue[0]
      queue = queue[1:]

      // Passed as a pointer
      valuesToTrace := extractExplicitValues(&entries[currIdx], cfg)

      for _, val := range valuesToTrace {
         for i := currIdx - 1; i >= 0; i-- {
            if kept[i] {
               continue
            }

            if responseContainsValue(&entries[i], val) {
               fmt.Printf("  -> Found origin of value '%s...' in response of request idx %d\n", val[:min(15, len(val))], i)
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

   if err := os.WriteFile(cfg.OutputFile, outData, 0644); err != nil {
      return fmt.Errorf("error writing to output file: %w", err)
   }

   fmt.Printf("\nSuccess! Kept %d out of %d requests. Saved to %s\n", len(filteredEntries), len(entries), cfg.OutputFile)
   return nil
}
